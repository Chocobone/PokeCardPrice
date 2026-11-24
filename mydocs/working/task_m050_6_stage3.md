# Task #6 단계3 완료 보고서: TCGCSV 어댑터

## 수행 내용

- TCGCSV 클라이언트 `internal/tcgcsv`를 만들었다 (갱신 시각, 그룹, 상품, 가격).
- TCGdex 카드 상세에서 TCGplayer 상품 ID를 꺼내 `external_ids`를 갱신하는 순회 작업(방법 A)을 만들었다. 1단계 승인안대로 3단계에서 함께 구현했고, 4단계에서 같은 순회에 Cardmarket 저장을 더한다.
- TCGCSV 가격 작업을 만들었다: 저장된 연결로 바로 저장 → TCGdex 순회 종료 대기 → 남은 행을 새 연결과 번호 대조(방법 B)로 저장. Q5(빠진 변형 추가)도 구현했다.
- 개발 DB에서 표본 5세트로 실제 수집해 1단계 측정값과 연결률을 비교하고, 저장 가격을 TCGCSV 원본과 대조했다.
- 표본 수집 중 틀린 연결 1건(한정판 가격이 기본 카드에 저장됨)을 발견해 규칙을 보강했다.

자세한 동작과 실측값은 `mydocs/tech/price_collector.md` "TCGCSV 수집 (3단계)"에 있다.

## 변경 파일

| 파일 | 변경 내용 |
|------|----------|
| `src/backend/internal/tcgcsv/client.go` | 신규. `LastUpdated`, `Groups`, `Products`, `Prices`, `Group.Code`, `Product.Number` |
| `src/backend/internal/tcgcsv/client_test.go`, `testdata/*` | 신규. 실제 응답에서 잘라 낸 고정 응답(그룹 3, 상품 5, 가격 7) |
| `src/backend/internal/tcgdex/types.go` | `Card.Pricing`(원본 JSON), `Card.TCGPlayer()`, `TCGPlayerPricing`, `Serie` |
| `src/backend/internal/tcgdex/client.go` | `GetSerie` (TCG Pocket 세트 목록) |
| `src/backend/internal/tcgdex/client_test.go`, `testdata/card_en_sv03.5-001_pricing.json` | 가격 해석 테스트, 현재 응답 고정본 |
| `src/backend/internal/prices/link.go` | 신규. `subTypeName` ↔ 변형, 변형 대체 규칙, 번호 정규화, 특별 상품 판별 |
| `src/backend/internal/prices/tcgdex.go` | 신규. TCGdex 카드 상세 순회 작업 (`tcgdex_cardmarket`, 3단계는 방법 A만) |
| `src/backend/internal/prices/tcgcsv.go` | 신규. TCGCSV 가격 작업 |
| `src/backend/internal/prices/source.go`, `runner.go` | `Options.TCGCSVGroups`(개발용), 작업 종료 대기(`waitFor`), 같은 코드 중복 실행 방지 |
| `src/backend/internal/prices/tcgcsv_test.go` | 신규. 연결 규칙 단위 테스트, TCGCSV 멱등성·그룹 실패 격리, TCGdex → TCGCSV 순서 (테스트 DB) |
| `src/backend/internal/db/queries/prices.sql`, `dbq/prices.sql.go` | `ListExternalIDLinks`, `ListSetCardVariants`, `ListSetTcgdexIDs`, `ListPriceTargetCards` |
| `mydocs/tech/price_collector.md` | "TCGCSV 수집 (3단계)" 절 추가 |
| `mydocs/orders/20261123.md` | 진행 상황 |

수집기(`cmd/collector`)에는 아직 연결하지 않았다 (5단계). 개발 DB 수집은 임시 실행 파일(`cmd/_pricesonce`)로 했고, 실행 후 지웠다.

## 설계 결정

1. **두 작업은 동시에 돌고, TCGCSV가 중간에 TCGdex를 기다린다.** `Runner`가 작업마다 "끝남" 채널을 만들어 넘긴다. TCGCSV는 저장된 연결로 먼저 저장하고, 남은 행만 TCGdex 순회가 끝난 뒤 처리한다. TCGdex 순회가 실패해도 채널은 닫히므로 그때까지 저장된 연결로 진행한다. 평소에는 대부분 첫 단계에서 저장되고, 빈 DB에서도 같은 날 모든 연결이 끝난다.
2. **TCGdex 순회 작업의 이름은 처음부터 `tcgdex_cardmarket`이다.** 3단계에서는 상품 ID 연결만 하지만, 4단계에서 같은 응답으로 Cardmarket 가격을 저장하므로 작업을 둘로 나누지 않았다.
3. **`tcgdex.Card.Pricing`은 원본 JSON으로 받는다.** 카탈로그 적재(#5)가 같은 타입을 쓰므로, 가격 필드 형식이 바뀌어도 카탈로그 적재가 실패하지 않게 시세 수집기에서만 해석한다.
4. **특별 상품의 저장된 연결은 쓰지 않는다 (신규 규칙).** 같은 번호의 기본 상품이 그룹에 따로 있으면 이름에 괄호가 있는 상품은 방법 A 연결이 있어도 건너뛴다 (발견 사항 1).
5. **모호한 연결은 저장하지 않는다.** 여러 가격 행이 한 변형에 연결되면 모두 버린다 (예: 일본판 단일 변형 카드에 `Normal`·`Holofoil` 두 행).
6. **Q5 변형 추가는 `normal`·`holofoil`·`reverse_holofoil`로 한정했다.** 초판·무제한판을 추가하면 카드의 판 구성 자체가 바뀌어서 승인 조건보다 좁혔다.
7. **같은 날 재수집을 막는 장치는 두지 않았다** (구현 계획서 검토 항목). 그룹 일부가 실패한 날을 구분하려면 실행 기록 테이블이 필요한데 Q3에서 로그만 남기기로 했다. 정기 실행은 하루 1회이다.
8. **개발용 그룹 범위 `TCGCSVGroups`를 추가했다.** 영어판 그룹은 이름 코드(`SV: Scarlet & Violet 151`)가 TCGdex 세트 ID(`sv03.5`)와 달라 `SetIDs`만으로 고를 수 없다. 5단계에서 환경 변수로 연결한다.
9. **수동 대응표(`manualGroupSets`)는 틀만 두고 비워 두었다.** 표본에서는 필요한 그룹이 없었다. 5단계 전체 수집 결과로 대응 없는 그룹을 보고 채운다.

## 검증 결과

| 항목 | 결과 |
|------|------|
| `go vet ./...` | 통과 |
| `go build ./...` | 통과 |
| `go test ./...` | 전체 통과. `tcgcsv` 4개, `tcgdex` 1개, `prices` 6개 테스트 추가 (연결 규칙 3, TCGCSV 멱등성·그룹 실패 격리, TCGdex → TCGCSV 순서) |
| 테스트 DB 멱등성 | 두 번 실행해도 `price_history` 5, `external_ids` 7, `card_variants` 8 불변. 2회째는 첫 단계에서 모두 연결 |
| 개발 DB 연결률 | 표본 4세트 1단계 측정값과 같음 (`sv03.5` 360, `svp` 212, `SV2a` 210). `SV4K` 95 → 94는 오늘 TCGCSV에 가격 행이 없는 카드 1장 |
| 개발 DB Q5 | `sm1` 리버스 126·홀로 27 변형 추가, 가격 저장 110 → 263 |
| 개발 DB 재실행 | 행 수 불변 (`external_ids` 877, `price_history` 876, 표본 4세트 기준) |
| 원본 대조 | 저장 가격 667건의 모든 가격 필드·대표값이 TCGCSV 원본과 같음 |
| 소요 시간 | 카드 상세 738장 3분 5초 (초당 약 4회, 1단계 계산과 같음). TCGCSV 그룹 4개 2.5초 |

- 개발 DB에는 표본 5세트의 `external_ids`·`price_history`(기준일 2026-11-22)와 `sm1` 추가 변형이 남아 있다. 5단계 전체 수집에서 같은 키로 갱신된다.

## 발견 사항

1. **TCGdex가 특별 상품 ID를 주는 경우가 있다.** `svp-173` Eevee 홀로의 TCGdex 상품 ID는 포켓몬센터 한정판(`610757`, 61.22달러)이고 기본 상품은 `610758`(8.00달러)이다. 첫 실행에서는 두 행이 한 변형을 다퉈 저장하지 않았으나, 두 번째 실행에서 저장된 연결로 한정판 가격이 저장되었다. 설계 결정 4로 고친 뒤 8.00달러로 바로잡혔다. 1단계의 "A·B 결과가 다른 건수 0"은 같은 가격 행끼리 비교한 값이라 이런 경우를 세지 않았다.
2. **이름에 괄호가 있는 단독 상품은 Q5 대상이 아니다.** `sm1`의 `Lurantis GX (Full Art)` 등 26행은 같은 번호에 상품이 하나뿐인데도 Q5 조건(괄호 없는 기본 상품) 때문에 변형을 추가하지 않아 가격이 저장되지 않았다 → **Q8**.
3. **Q5로 홀로 변형을 추가한 카드에는 가격 없는 `normal`이 남는다.** `sm1` 홀로 레어 27장은 TCGdex에 `normal`만 있고 TCGplayer에는 `Holofoil`만 있다. 4단계 Cardmarket 기본 필드 규칙(`normal` 우선, Q7)을 그대로 쓰면 TCGplayer 가격은 `holofoil`, Cardmarket 가격은 `normal`에 붙는다 → **Q9**.

## 작업지시자 확인 사항 (2026-11-23 결정: 모두 제안대로)

| # | 사항 | 제안 | 대안 |
|---|------|------|------|
| Q8 | 이름에 괄호가 있는 **단독** 상품 (발견 사항 2) | **현행 유지.** 괄호 상품은 Q5로 변형을 추가하지 않는다. 기존 변형과는 그대로 연결된다(`sm1` Full Art도 TCGdex 변형이 맞으면 연결됨). 전체 수집(5단계)에서 건수를 보고 다시 판단한다 | 같은 번호의 상품이 하나뿐이면 괄호가 있어도 Q5 대상으로 한다 (`sm1` 26행 추가 연결, 오연결 위험은 낮음) |
| Q9 | Q5로 `holofoil`을 추가한 카드의 Cardmarket 기본 필드 (발견 사항 3) | **TCGplayer 가격이 있는 변형을 먼저 고른다.** 4단계에서 기본 필드를 넣을 변형은 Q7 순서(`normal` → `holofoil` → …)를 따르되, 같은 카드에서 TCGplayer 가격(`external_ids`)이 `holofoil`에만 있으면 `holofoil`을 고른다 | Q7 순서 그대로 (`normal`에 Cardmarket, `holofoil`에 TCGplayer) |

## 다음 단계

승인되면 3단계 소스와 이 보고서를 커밋하고 4단계(TCGdex Cardmarket 어댑터)를 진행한다. Q9 결정은 4단계 변형 배분 규칙에 반영한다.
