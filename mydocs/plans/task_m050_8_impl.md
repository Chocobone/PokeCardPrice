# Task #8: 카드 검색·시세 조회 API — 구현 계획서

수행 계획서: `mydocs/plans/task_m050_8.md` (2026-11-25 승인, Q1~Q6 제안대로 결정)

1단계는 계약 확정과 쿼리 측정 단계다. 문서와 측정용 SQL만 작성하고 `src/`는 바꾸지 않는다. 1단계 결과(확정 계약, 쿼리 형태, 인덱스 추가 여부)를 작업지시자가 승인한 뒤 2~5단계에서 소스를 변경한다.

## 공통 규칙

### 패키지·파일 구성 (안)

| 위치 | 역할 |
|------|------|
| `internal/db/queries/cards.sql` (신규) | 검색·카드 상세·최신가·추이·순위 쿼리 |
| `internal/api` (신규) | 응답 타입(계약과 1:1), 금액·원화 변환, 대표가 선택(Q1), 추이 요약 계산(Q5). DB·HTTP와 무관한 순수 함수 위주로 단위 테스트 |
| `internal/server` (기존) | 라우트, 핸들러, 매개변수 검증, 계약 오류 형식 |
| `migrations/00005_*.sql` | 1단계 측정에서 필요하다고 판단한 경우에만 |

- 핸들러는 sqlc `*dbq.Queries`를 받는 작은 인터페이스에 의존해 `httptest`에서는 테스트 DB를 쓴다.
- `server.New(db Pinger)`는 `server.New(pool *pgxpool.Pool)` 또는 의존성 구조체로 바꾼다 (2단계에서 확정).

### 원화 환산 (Q2)

- 환율: 가격 기준일 이전 가장 가까운 Frankfurter 환율(`ListPriceHistoryKRW`와 같은 규칙). 소스 통화가 KRW면 그대로.
- 계산: `decimal` `price × rate`, 원 단위 반올림(`Round(0)`, half-away-from-zero). 환율이 없으면 `krw: null`.
- `Price` 타입 확장: `{ amount, currency, source, date, krw: string | null, fxDate: string | null, updatedAt: string | null }`. `Currency`에 `EUR` 추가.

### 테스트

- DB 테스트는 `dbtest.Pool(t, "<패키지명>")`로 패키지별 테스트 DB에서 실행한다. 카탈로그·가격·환율 고정 데이터를 테스트 안에서 만든다(날짜는 고정값).
- 각 단계 끝에 `go vet ./...`, `go test ./...`, `go build ./...` 통과. 프론트엔드를 바꾼 단계는 `npm run typecheck`·`lint`·`build`도 통과.
- 수집기는 계속 돌고 있으므로 개발 DB에는 읽기만 한다. 측정용 임시 DB(`poc_task8`)는 측정 후 지운다.

---

## 1단계: 계약 확정과 쿼리 설계

### 목표

새 엔드포인트(상세·추이)를 포함한 계약을 확정하고, 검색·대표가·추이·순위 쿼리 초안을 개발 DB(전체 적재)와 이력을 늘린 임시 DB에서 `EXPLAIN ANALYZE`로 재서 형태와 인덱스 필요 여부를 정한다.

### 구현

**`mydocs/tech/api_contract.md`** — 확정판으로 갱신:

```ts
// 신규
GET /cards/{id}            → CardDetail
GET /cards/{id}/prices     ?period=7d|30d|90d|1y (기본 30d)  → PriceHistoryResponse

interface CardDetail extends CardSummary {
  rarity: string | null
  set: { tcgdexId: string; name: string; releaseDate: string | null; logoUrl: string | null }
  linked: CardSummary[]            // 한글판 ↔ 일본판 연결 카드
  variants: { variant: string; prices: Price[] }[]   // 변형 × 소스별 최신가
}

interface PriceSeries {
  variant: string
  source: string
  currency: Currency
  points: { date: string; amount: string; krw: string | null }[]
  summary: { current: Price; average: string; min: string; max: string;
             changeRate: number | null; from: string }   // Q5
}
interface PriceHistoryResponse { period: string; series: PriceSeries[] }
```

- 오류 코드에 `404 not_found`(카드 없음) 추가. `id`가 숫자가 아니면 `400 invalid_param`.
- 대표가(Q1)·순위(Q3)·정렬(Q4)·요약(Q5) 규칙을 문서에 명시.

**측정용 SQL** (세션 임시 폴더, 결과만 문서에 기록):

| # | 쿼리 | 확인 사항 |
|---|------|-----------|
| A1 | 검색: 낱말 배열을 `unnest`하고 낱말마다 `name ILIKE`·한글 종 이름(`pokemon_species.name_ko` → `dex_ids &&`)·세트 이름·세트 ID·번호(`ltrim(local_id,'0')`) 중 하나가 맞아야 함 (AND) | 낱말 1·2·3개, 2자 한글·영문 7자·`sv03.5 25` 시간. trigram 인덱스 사용 여부 |
| A2 | 검색 전체 건수(`total`) | `count(*) OVER()`와 별도 쿼리 비교 |
| A3 | 카드 N개(24)의 대표가: 변형 × 소스 최신가 `DISTINCT ON` 후 Q1 우선순위 | 24장 기준 시간 |
| A4 | 카드 1장 변형 × 소스 최신가 + 원화 (#4 S2 확장) | |
| A5 | 카드 1장 1년 추이 + 원화 | 임시 DB 365일치 |
| A6 | 순위 7d·30d: TCGCSV 기준일·비교일 가격 조인, $1 이상, 카드당 1개 | 개발 DB(3일치)·임시 DB(365일치) |

임시 DB는 개발 DB의 `price_history`를 날짜만 바꿔 복제(가격에 임의 변동)해 365일치로 만든다. 규모는 #4 8절 추정과 대조한다.

**`mydocs/working/task_m050_8_stage1.md`** — 확정 계약 요약, 측정 결과 표, 쿼리 형태와 인덱스 제안.

### 검증

- 모든 측정 쿼리가 Q6 목표(검색·상세·추이 p95 100 ms, 순위 300 ms) 안인지, 넘으면 인덱스 안과 그 효과까지 측정.
- 개발 DB는 읽기만 했는지 확인 (임시 DB 삭제 포함).

---

## 2단계: 공통 기반과 `GET /cards` 검색

### 목표

계약 오류 형식·매개변수 검증·금액/원화 변환을 만들고, 검색 API를 구현한다.

### 구현

**`internal/server/server.go`** — 오류 처리기를 `{"error": {"code", "message"}}`로 변경. 도메인 오류 타입:

```go
type apiError struct { Status int; Code, Message string }
func invalidParam(msg string) *apiError
func notFound(msg string) *apiError
```

Echo 기본 오류(404 경로 없음, 405)도 같은 형식으로 낸다. 500은 `internal` + 고정 한국어 메시지, 원인은 로그에만.

**`internal/api/price.go`** — `func NewPrice(amount decimal.Decimal, currency, source string, date time.Time, fx *FxRate, updatedAt *time.Time) Price`, `func PickRepresentative(prices []VariantPrice) *Price` (Q1).

**`internal/db/queries/cards.sql`** — `SearchCards`(1단계 확정 형태, `@terms::text[]`, `@language`, `@limit`, `@offset`), `CountSearchCards`(필요 시), `ListLatestPricesForCards`(`@card_ids::bigint[]`, 원화 포함).

**`internal/server/cards.go`** — `GET /cards`: `q` 공백 분리·1~100자(`invalid_query`), `lang`·`page`·`size` 검증, 검색 → 대표가 일괄 조회 → `Page<CardSummary>`.

**`cmd/server/main.go`** — 핸들러에 쿼리 객체 전달.

### 검증

- 단위: 원화 반올림, 환율 없음, Q1 우선순위(소스·변형·한글판 `null`).
- DB+HTTP: 이름(영·일·한)·한글 종 이름·세트 ID·번호(`25`=`025`)·AND 조합·언어 필터·쪽 넘김·`total`·정렬(Q4), 오류 코드 3종.
- 기존 `server_test.go`(`/healthz`)를 새 오류 형식에 맞춰 수정 후 통과.

---

## 3단계: 카드 상세와 시세 추이

### 목표

`GET /cards/{id}`와 `GET /cards/{id}/prices`를 구현한다.

### 구현

**`cards.sql`** — `GetCardDetail`(카드·세트·종 이름), `ListLinkedCards`(`card_links` 양방향), `ListCardLatestPrices`(변형 × 소스 최신가 + 원화), `ListCardPriceHistoryKRW`(카드의 모든 변형 × 소스, `@from_date`, 원화 포함 — `ListPriceHistoryKRW`를 카드 단위로 확장).

**`internal/api/history.go`** — `Summarize(points) Summary` (Q5: 첫·마지막 가격 변동률, 산술 평균은 `decimal` 소수 2자리 반올림, 최저·최고, 실제 시작일). 첫 가격이 0이면 `changeRate: null`.

**`internal/server/cards.go`** — 두 핸들러, `id`·`period` 검증, 없는 카드 404.

### 검증

- 단위: 요약 계산(점 1개, 첫 가격 0, 이력이 기간보다 짧음).
- DB+HTTP: 변형별 소스가 다른 카드(Cardmarket 없는 초판), 한글판 ↔ 일본판 연결, 환율 공백일(주말) 직전 값, 기간 경계, 시세 없는 카드(`variants[].prices` 빈 배열, `series` 빈 배열).

---

## 4단계: 급등·급락 순위

### 목표

`GET /rankings/movers`를 Q3 규칙으로 구현하고 규모 측정으로 응답 시간을 확인한다.

### 구현

**`cards.sql`** — `GetLatestPriceDate`(`@source_id`), `ListMovers`(`@as_of`, `@compare_on`, `@min_price`, `@direction`, `@limit`): 기준일 가격 ↔ 비교일 이전 가장 가까운 날 가격, 두 가격 모두 `@min_price`(USD 1) 이상, 카드당 변동률 최대(하락은 최소) 변형 1개. 1단계에서 필요하다고 정한 인덱스가 있으면 `00005` 마이그레이션.

**`internal/server/rankings.go`** — `period`·`direction`·`limit` 검증, 시세 없으면 `asOf: null`·`items: []`, `Mover.card`는 2단계 `CardSummary` 조립 재사용(대표가 포함), `current`·`previous`는 해당 변형 가격.

### 검증

- DB+HTTP: 상승·하락 정렬, $1 미만 제외, 비교일 공백 시 이전 날 사용, 카드당 1개, 데이터 없음, 오류 코드.
- 1단계 임시 DB(365일치)에서 순위 응답 시간 재측정.

---

## 5단계: 프론트엔드 전환, 실측, 문서

### 목표

프론트엔드가 실제 API로 동작하게 하고, 응답 시간을 실측하고, 문서를 정리한다.

### 구현

- **`src/frontend/src/api/types.ts`** — `Currency`에 `EUR`, `Price`에 `krw`·`fxDate`·`updatedAt`, `CardDetail`·`PriceHistoryResponse` 타입 추가(#13 대비). `cards.ts`에 `getCard`·`getCardPrices` 추가(목업은 간단히).
- **`src/frontend/src/api/client.ts`** — 오류 형식이 계약과 맞는지 확인(이미 계약 형식을 읽으면 변경 없음).
- 홈·검색 화면의 가격 표시가 `EUR`·`krw`를 처리하는지 확인하고 필요한 최소 수정(원화 병기 여부는 화면 변경이 필요하면 작업지시자 확인).
- **`src/frontend/.env.development`** — `VITE_API_MOCK=false`.
- **응답 시간 실측**: 개발 DB에서 엔드포인트별 대표 요청 100회씩, p50·p95 기록.
- **문서**: `api_contract.md` 최종 반영, `CLAUDE.md`(API 목록·실행 방법), `db_schema.md` 7절(새 쿼리 측정값, 인덱스 추가 시).

### 검증

- 백엔드 `go vet`·`go test`·`go build`, 프론트엔드 `typecheck`·`lint`·`build` 통과.
- 브라우저에서 홈(급등·급락)·검색(한글 종 이름, 세트+번호) 화면이 실제 데이터로 표시되는지 확인.
- 실측값이 Q6 목표 안인지.

---

## 검증 (전체)

- `docker compose exec backend go vet ./...`, `go test ./...`, `go build ./...` 통과
- `docker compose exec frontend npm run typecheck`, `npm run lint`, `npm run build` 통과
- 4개 엔드포인트가 계약 문서와 같은 형식으로 응답하고, 오류는 `{"error": {code, message}}`
- 응답 시간: 검색·상세·추이 p95 100 ms 이하, 순위 300 ms 이하 (개발 DB 전체 적재)

## 예상 단계: 5단계
