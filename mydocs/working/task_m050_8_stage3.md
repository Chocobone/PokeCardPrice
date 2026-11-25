# Task #8: 카드 검색·시세 조회 API — 3단계 완료 보고서

## 단계 목표

`GET /cards/{id}` 카드 상세와 `GET /cards/{id}/prices` 시세 추이를 계약(`mydocs/tech/api_contract.md`)대로 구현한다.

## 변경 파일

| 파일 | 내용 |
|------|------|
| `src/backend/internal/db/queries/cards.sql` | `GetCardDetail`(카드·세트·종 이름), `ListLinkedCards`(`card_links` 양방향), `ListCardPriceHistory`(카드의 모든 변형 × 판매가 소스 기간 추이 + 직전 환율) 추가 |
| `src/backend/internal/db/dbq/cards.sql.go` (생성) | sqlc 생성 코드 |
| `src/backend/internal/api/detail.go` (신규) | `CardDetail`·`PriceSeries`·`PriceHistoryResponse` 타입, `PeriodStart`(오늘 KST − 일수), `Summarize`(Q5), `ChangeRate`, `SortBySource` |
| `src/backend/internal/api/price.go` | `SourceRank` 추가 |
| `src/backend/internal/server/detail.go` (신규) | 두 핸들러, `id`·`period` 검증, 없는 카드 404 |
| `src/backend/internal/server/server.go`, `cards.go` | 라우트 2개, 현재 시각 주입(`newRouter`, 테스트에서 고정) |
| `src/backend/internal/api/detail_test.go`, `server/detail_test.go` (신규), `server/cards_test.go` | 단위·DB 테스트, 고정 데이터 보강 |

## 구현 내용

### 카드 상세 (`GET /cards/{id}`)

- 변형은 `ListCardVariants`(id 순)로 모두 싣고, 가격은 2단계 `ListLatestPrices`를 카드 하나로 불러 변형마다 소스 우선순위(TCGCSV → Cardmarket) 순으로 넣는다. 시세 없는 변형은 `prices: []`.
- 대표가(`price`)는 검색과 같은 규칙(Q1)이다.
- `linked`: 한글판은 일본판, 일본판은 한글판들. 항목은 `CardSummary`(대표가 포함, 한글판은 `null`)이며 2단계 `summaries`를 다시 쓴다.
- `dexIds`는 트레이너스·에너지에서 `[]`(NULL을 빈 배열로).

### 시세 추이 (`GET /cards/{id}/prices`)

- `period` 기본 `30d`, `7d`·`30d`·`90d`·`1y`(365일). 시작일 = 오늘(KST) − 일수.
- 쿼리 결과(변형 id → 소스 id → 날짜 순)를 변형·소스마다 계열로 나누고, 같은 변형 안에서는 소스 우선순위 순으로 둔다.
- 점마다 원화(그 날짜 이전 가장 가까운 환율), 요약은 `Summarize`:
  - `current` = 마지막 가격(`Price` 전체 필드), `average` = 산술 평균 소수 둘째 자리 반올림, `min`·`max`, `from` = 첫 가격 날짜
  - `changeRate` = (마지막 − 첫) / 첫, 소수 여섯째 자리 반올림. 가격이 하나뿐이거나 첫 가격이 0이면 `null`
- 없는 카드는 404(추이 조회 전에 카드를 확인), 시세 없는 카드는 `series: []`.

## 검증

| 항목 | 결과 |
|------|------|
| `go vet ./...`, `go build ./...` | 통과 |
| `go test ./...` (DB 테스트 포함) | 전체 통과 |

**테스트 내용**

- 단위(`internal/api`): 기간 시작일(UTC 15:30 = KST 다음 날 경계), 요약(평균 반올림 0.5 올림, 하락, 가격 하나, 첫 가격 0), 변동률 자릿수, 소스 정렬.
- 상세: 카드·세트 필드, 변형 id 순 × 소스 우선순위, 대표가·원화, 연결 없음 `linked: []`, 트레이너스 `dexIds: []`·`prices: []`, 한글판 → 일본판(대표가 있음), 일본판 → 한글판(대표가 `null`), 오류 6건(404·잘못된 id·잘못된 period).
- 추이: 기본 30d 시작일과 기간 밖 가격 제외, 계열 순서, 점별 원화(직전 환율), 요약 값, EUR 환율 없음 → 원화 `null`, 90d(첫 환율보다 이른 날은 원화 `null`, 하락 변동률), 7d·1y 시작일, 시세 없는 카드 `series: []`.
- 테스트 중 고정 데이터 문제 하나를 고쳤다: `INSERT … SELECT … JOIN`이 행을 만드는 순서가 정해져 있지 않아 변형 id 순서가 실행마다 달랐다. 순번을 붙여 `ORDER BY`로 넣는다.

**개발 DB 실제 요청** (백엔드 재시작 후, 호스트 `curl`, 3회)

| 요청 | 시간 |
|------|------|
| `/cards/16970` (sv03.5-025 상세) | 105 ms(첫 요청) → 71 → 10 ms |
| `/cards/16970/prices` (30d) | 9~12 ms |
| `/cards/16970/prices?period=1y` | 8~27 ms (개발 DB 이력은 3일치) |
| `/cards/36319` (한글판, 연결 1장) | 13~25 ms |

- 응답 확인: 상세는 세트 로고·카드 수·희귀도까지 채워지고, 한글판 `SV4K-025`는 일본판 `ウミトリオ`(대표가 $0.13, ₩174)와 연결돼 나온다.
- 1년치 규모의 추이 SQL은 1단계에서 측정했다(캐시 후 17~46 ms, 첫 조회 최대 137 ms).

## 참고

- `updatedAt`은 RFC 3339라 소수 초를 버린다(`12:12:41.912` → `12:12:41`). 화면 표시에는 영향이 없다.
- `period=1y`의 점은 최대 365개 × 계열 수(보통 2~6)다. 응답 크기는 1년치가 쌓이면 카드당 수십 KB 수준으로 예상되며, 5단계 실측 때 함께 본다.

## 다음 단계

4단계: `GET /rankings/movers` 급등·급락 순위와 1년치 규모(`poc_task8`) 측정.
