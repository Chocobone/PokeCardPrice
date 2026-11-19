# Task #6: 시세 수집 스케줄러 (해외 소스) — 구현 계획서

수행 계획서: `mydocs/plans/task_m050_6.md` (2026-11-19 승인, Q1~Q4 제안대로 결정)

1단계는 조사·측정 단계라 문서만 작성하고 저장소 소스는 바꾸지 않는다. 1단계 결과(상품 ↔ 변형 연결 규칙 Q1, Cardmarket 수집 빈도 Q2, `-holo` 필드 연결 규칙)를 작업지시자가 승인한 뒤 2~5단계에서 `src/backend/`를 변경한다.

## 공통 규칙

### 외부 호출

- 조사 단계의 호출도 수집기와 같은 예절을 지킨다: 전용 User-Agent(`PokeCardPrice-collector/0.1` 형식, 개인 정보 미포함), 동시성 1.
  - TCGCSV: 요청 간 100 ms 이상(조사는 120 ms), 24시간 1만 회 미만.
  - TCGdex: 요청 간 250 ms 이상(#5 설정). 상시 수집기가 같은 IP에서 돌고 있으므로 조사 호출은 표본으로 제한한다.
  - Frankfurter: 조사 호출은 수십 회 이내.
- 조사용 스크립트와 원본 응답은 세션 임시 폴더에 두고, 결과 문서에는 측정값과 필요한 발췌만 남긴다.
- 확인 상태 표기는 #2·#5와 같이 `응답 확인`, `문서 확인`, `미확인`, `확인 불가`를 쓴다.

### 테스트

- 외부 API에 의존하는 테스트는 만들지 않는다. 클라이언트·어댑터는 `httptest.Server`와 저장한 고정 응답(`testdata/*.json`, 표본 몇 건만)으로 검증한다.
- DB 테스트는 `dbtest.Pool(t, "<패키지명>")`으로 패키지별 테스트 DB에서 실행한다.
- 각 단계 끝에 `go vet ./...`, `go test ./...`, `go build ./...`을 통과시킨다.

### 결과 문서

**`mydocs/tech/price_collector.md`** (신규, `tech_investigation_template.md` 형식). 1단계에서 조사·측정 결과와 규칙을 쓰고, 2~5단계에서 실제 구현·실측값과 맞춰 갱신한다.

### 패키지 구성 (안)

| 패키지 | 역할 |
|--------|------|
| `internal/tcgcsv` | TCGCSV HTTP 클라이언트와 응답 타입 (그룹·상품·가격, `last-updated`) |
| `internal/fx` | Frankfurter v2 클라이언트 |
| `internal/tcgdex` (기존) | 카드 상세 응답에 `pricing.cardmarket`, `variants_detailed` 필드 추가 |
| `internal/prices` | 어댑터 인터페이스, 소스별 어댑터(TCGCSV·Cardmarket), 환율 수집, 병렬 실행·재시도·요약 로그 |
| `cmd/collector` | `collect-daily` 작업에 시세 수집 연결, `PRICES_*` 환경 변수 |

HTTP 재시도·요청 간격 로직은 `tcgdex.Client`와 비슷하지만, 소스마다 규칙이 다르고 코드가 작아 2단계에서는 각 클라이언트에 둔다. 중복이 커지면 2단계 보고서에서 공용 패키지로 묶을지 제안한다.

---

## 1단계: 사전 확인과 측정

### 목표

구현을 좌우하는 사실(응답 필드, 상품 ↔ 변형 연결률, Cardmarket 필드 매핑, 호출 수·소요 시간, 환율 동작)을 확인하고, Q1·Q2의 확정안을 낸다. **이 단계 결과를 작업지시자가 승인하면 2단계부터 소스를 작성한다.**

### 구현

**TCGCSV 응답 확인.**

| 엔드포인트 | 확인 항목 |
|------------|-----------|
| `/tcgplayer/categories` | `Pokemon`·`Pokemon Japan` 카테고리 ID |
| `/tcgplayer/{category}/groups` | 그룹 수, `groupId`·`name`·`abbreviation`·`publishedOn` |
| `/tcgplayer/{category}/{group}/products` | `productId`, `name`, `extendedData`(`Number`, `Rarity` 등), 카드가 아닌 상품을 구분할 방법 |
| `/tcgplayer/{category}/{group}/prices` | `productId`, `subTypeName`, 가격 필드, NULL 비율 |
| `/last-updated.txt` | 갱신 시각 형식. `price_date`(소스 기준일)를 정하는 방법 |

**상품 ↔ 카드 변형 연결 측정 (Q1).** 표본 세트 약 10개(영어 구세트·초판 세트·최근 세트, 일본어 구·신 세트, 프로모)로 두 방법을 비교한다.

| 방법 | 내용 |
|------|------|
| A. TCGdex 상품 ID | 카드 상세 `variants_detailed[].thirdParty.tcgplayer` → TCGCSV `productId`, `variants_detailed[].type`·TCGCSV `subTypeName` → `card_variants.variant` |
| B. 세트 + 번호 대조 | TCGCSV 그룹 ↔ TCGdex 세트(이름·약어·발매일), 상품 `extendedData` `Number`(예: `001/165`) ↔ 카드 `localId`, `subTypeName` → 변형 |

| 측정 | 의미 |
|------|------|
| 카드 변형 중 TCGCSV 가격 행과 연결된 비율 (방법별) | 연결률 |
| 두 방법의 결과가 다른 건수와 원인 | 오연결 후보. 표본 몇 건은 상품 이름·번호를 눈으로 대조 |
| 연결되지 않은 TCGCSV 가격 행 분류 | 박스·팩 / 도장·Shadowless 등 7종으로 표현 못하는 상품 / 기타 |
| 그룹 ↔ 세트 대응을 자동으로 정할 수 있는 비율 | 방법 B의 비용 (수동 대응표가 필요한지) |

결과로 연결 규칙(우선순위, 두 방법이 다를 때 처리, 연결하지 않는 경우)을 제안한다.

**TCGdex Cardmarket 확인.**

- 카드 상세 `pricing.cardmarket` 필드(`avg`, `low`, `trend`, `avg1`, `avg7`, `avg30`, `-holo` 계열, `updated`, `unit`)와 값이 있는 비율 (`en`·`ja` 표본 각 40장 내외).
- **`-holo` 필드의 변형 연결 규칙** (#4 인계): 카드 변형 구성(`normal`만, `normal`+`reverse_holofoil`, `holofoil`만, 초판 계열)별로 기본 필드와 `-holo` 필드를 어느 변형에 넣을지 표본으로 정한다.
- **하루 호출 수·소요 시간 (Q2)**: `en`·`ja` 카드 수 × 요청 간격(250 ms, 동시성 2) 기준 소요 시간을 계산하고, 표본 호출의 실제 응답 시간·`503` 빈도를 반영한다. 매월 1일 카탈로그 전체 갱신과 겹칠 때의 종료 시각도 계산한다. 가격 필드가 비어 있는 카드 비율로 "가격이 있었던 카드만" 방안의 절감 폭을 함께 낸다.
- 카탈로그 상세 조회와 가격 조회를 한 번의 호출로 합칠 수 있는지(같은 응답) 검토한다.

**Frankfurter v2 확인** (#2 미해결 사항).

- `base=KRW` 또는 `base=USD`·`EUR`로 KRW 환율을 받는 경로와 응답 형식, 값 기준일 필드.
- 주말·공휴일(한국·유럽) 날짜 조회 시 동작, `providers` 기본값.
- 호출 한도 수치 기재 여부, 응답 헤더.

**`mydocs/working/task_m050_6_stage1.md`** — 단계별 완료 보고서. 결정이 필요한 사항은 질문 목록(Q5~)으로 정리한다.

### 검증

- 측정 수치는 호출 시각·표본 목록과 함께 기록하고, 재현할 수 있도록 측정 방법을 적는다.
- 소스 코드 변경 없음 (`git status`로 `src/` 변경 없음 확인).

---

## 2단계: 어댑터 구조와 환율 수집

### 목표

모든 소스가 따를 어댑터 인터페이스와 실행 틀을 만들고, 첫 소스로 환율을 수집·저장한다.

### 구현

**`src/backend/internal/prices/source.go`** — 어댑터 인터페이스와 결과 요약:

```go
// Source 는 가격 소스 어댑터이다. Collect 는 외부 ID 연결부터 저장까지 한 소스를 끝까지 처리한다.
type Source interface {
    Code() string // price_sources.code
    Collect(ctx context.Context, q Store, opt Options) (Result, error)
}

// Result 는 소스 하나의 실행 요약이다 (Q3: 로그로만 남긴다).
type Result struct {
    Source               string
    Fetched, Linked      int // 받은 가격 행, 변형과 연결된 행
    Saved, Skipped       int
    Failed               []string // 실패한 그룹·카드 ID (실행 끝 재시도 후 남은 것)
    Duration             time.Duration
}
```

- `Options`: 언어·세트 필터, 오늘 날짜(테스트 주입용) 등.
- `Store`는 `dbq.Queries` 중 필요한 메서드만 담은 인터페이스로 두어 테스트에서 테스트 DB를 그대로 쓴다.

**`src/backend/internal/fx/`** — Frankfurter v2 클라이언트 (1단계에서 확인한 경로·형식). 타임아웃, 지수 백오프 재시도, 전용 User-Agent.

**`src/backend/internal/prices/fx.go`** — 환율 수집: USD→KRW, EUR→KRW를 조회해 `UpsertFxRate`(provider `frankfurter`, `rate_date`는 응답의 값 기준일)로 저장한다. 값이 없으면 저장하지 않고 경고 로그만 남긴다(조회 시 `GetFxRateOnOrBefore`가 직전 값을 씀).

**`src/backend/internal/prices/runner.go`** — 실행 틀: 환율 → 가격 소스들(고루틴, `errgroup` 없이 각 소스 결과를 모아 한 소스의 오류가 다른 소스를 취소하지 않게 함) → 소스별 요약 로그. 이 단계에서는 가격 소스 없이 환율만 연결한다.

### 검증

- `fx` 클라이언트: 고정 응답으로 정상·주말·오류(5xx 재시도, 4xx 즉시 실패) 확인.
- 환율 저장: 테스트 DB에서 같은 날 두 번 실행해도 행이 1개, 값 갱신 확인.
- 실행 틀: 가짜 소스 두 개 중 하나가 실패해도 다른 소스가 끝까지 실행되는지 확인.
- 개발 DB에서 환율 1회 실제 수집 후 `fx_rates` 확인.

---

## 3단계: TCGCSV 어댑터

### 목표

TCGCSV 가격을 1단계에서 정한 규칙으로 카드 변형에 연결하고 `price_history`에 저장한다.

### 구현

**`src/backend/internal/tcgcsv/`** — 클라이언트와 응답 타입: 그룹 목록, 그룹별 상품·가격, `last-updated`. 동시성 1, 요청 간 100 ms 이상(기본 120 ms), 지수 백오프 재시도.

**`src/backend/internal/prices/tcgcsv.go`** — 어댑터:

1. `last-updated`로 소스 기준일(`price_date`)과 `source_updated_at`을 정한다. 이미 그 기준일 행이 저장되어 있으면 다시 받지 않는 방안을 함께 검토한다(하루 1회 동기화 규칙).
2. 카테고리(`Pokemon` → `en`, `Pokemon Japan` → `ja`)별 그룹을 돌며 상품·가격을 받는다.
3. Q1 확정 규칙으로 (`productId`, `subTypeName`) → `card_variant_id`를 정하고 `UpsertExternalID`로 저장한다. 이미 있는 `external_ids`는 먼저 조회해 재사용한다.
4. 대표값 `market`, 없으면 `mid`(둘 다 없으면 저장하지 않음)로 `UpsertPriceHistory` 배치 저장. 그룹 단위로 묶어 한 그룹 실패가 다른 그룹에 번지지 않게 한다.
5. 연결 못한 가격 행은 분류별 건수만 요약 로그에 남긴다(디버그 로그로 상품 ID 목록).

**`src/backend/internal/db/queries/prices.sql`** — 필요한 쿼리 추가 (예: 소스별 외부 ID 일괄 조회, 연결 대상 변형 조회). `go tool sqlc generate` 후 생성 코드 함께 커밋.

### 검증

- 고정 응답(표본 그룹 1~2개)으로 연결·대표값·NULL 처리·카드가 아닌 상품 제외 확인.
- 테스트 DB에서 두 번 실행해도 `price_history`·`external_ids` 행 수 동일.
- 개발 DB에서 표본 세트(예: `sv03.5`, `SV2a`)로 실제 수집 후 1단계 측정값과 연결률 비교, 표본 카드 가격을 TCGCSV 원본과 대조.

---

## 4단계: TCGdex Cardmarket 어댑터

### 목표

TCGdex 카드 상세의 Cardmarket 가격을 카드 변형에 붙여 `price_history`에 저장한다.

### 구현

**`src/backend/internal/tcgdex/types.go`** — `Card`에 `Pricing.Cardmarket`(기본·`-holo` 필드, `updated`, `unit`) 추가. 카탈로그 적재 동작은 바꾸지 않는다.

**`src/backend/internal/prices/cardmarket.go`** — 어댑터:

1. Q2 확정 범위의 카드(`en`·`ja`)를 DB에서 읽는다.
2. `tcgdex.Client`(#5와 같은 설정, 요청 간격·동시성 공유)로 카드 상세를 받는다.
3. 1단계에서 정한 `-holo` 규칙으로 기본 필드·`-holo` 필드를 변형에 나눈다.
4. 대표값 `trend`, 없으면 `avg`. `price_date`는 `updated`의 날짜(UTC), `source_updated_at`은 `updated`.
5. 카드 단위 실패는 모아 실행 끝에 한 번 더 시도하고, 남은 실패는 요약 로그에 남긴다. 저장은 일정 건수 단위 배치.
6. `unit`이 `EUR`가 아니면 저장하지 않고 경고한다(소스 통화 고정).

### 검증

- 고정 응답으로 변형 구성별 `-holo` 분배, 가격 없는 카드, `unit` 이상 처리 확인.
- 테스트 DB 멱등성 확인.
- 개발 DB에서 표본 세트로 실제 수집, 소요 시간이 1단계 계산과 맞는지 확인.

---

## 5단계: 수집기 연결·원화 환산·실측·문서

### 목표

시세 수집을 매일 일정에 연결하고, 원화 환산 조회 쿼리를 추가하며, 전체 수집을 실측하고 문서를 정리한다.

### 구현

**`src/backend/cmd/collector/`** — `collect-daily`에서 카탈로그 다음에 `prices` 실행. 카탈로그가 실패해도 시세 수집은 기존 카드로 진행한다. 개발용 환경 변수(안): `PRICES_SOURCES=fx,tcgcsv,cardmarket`(비우면 전체), 언어·세트 필터는 `CATALOG_LANGUAGES`·`CATALOG_SETS`와 같은 값을 쓸지 별도 변수로 둘지 단계 보고서에서 정한다.

**`src/backend/internal/db/queries/prices.sql`** — 원화 환산 조회 (예: 변형·소스·기간의 가격 이력을 `price_date` 기준 직전 환율로 원화 환산, `LATERAL` + `fx_rates` 기본 키 역순 스캔). #8이 그대로 쓰거나 바탕으로 삼는다. `EXPLAIN`으로 기본 키 범위 스캔을 확인한다.

**실측**: 개발 DB 전체 수집 1회(`-once`)의 소스별 건수·연결률·소요 시간·요청 수를 기록한다. 상시 수집기 재시작 시점은 작업지시자와 정한다.

**문서**: `mydocs/tech/price_collector.md` 구현 결과·운영 방법·후속 인계(#7·#8), `CLAUDE.md` 빌드 및 실행(시세 수집 명령·환경 변수), `price_data_sources.md` 미해결 사항 갱신(Frankfurter 한도·공휴일), Q4 과거 이력 백필 별도 이슈 등록 제안.

### 검증

- `go vet ./...`, `go test ./...`, `go build ./...` 통과.
- 원화 환산 쿼리: 테스트 DB에서 주말(환율 없는 날) 직전 값 사용 확인.
- 전체 `-once` 실행 후 같은 날 재실행 시 행 수 불변.

---

## 검증 (전체)

- `docker compose exec backend go vet ./...` 통과
- `docker compose exec backend go test ./...` 통과
- `docker compose exec backend go build ./...` 통과
- 개발 DB 실제 수집: 환율 2쌍, TCGCSV·Cardmarket 가격 저장, 재실행 시 행 수 불변
- 표본 카드 가격을 소스 원본 응답과 대조

## 예상 단계: 5단계
