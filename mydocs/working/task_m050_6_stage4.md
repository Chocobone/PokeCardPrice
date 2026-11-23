# Task #6 단계4 완료 보고서: TCGdex Cardmarket 어댑터

## 수행 내용

- TCGdex 카드 상세의 `pricing.cardmarket` 해석(`Card.Cardmarket()`)을 추가했다.
- 3단계 카드 상세 순회 작업(`tcgdex_cardmarket`)이 같은 응답으로 Cardmarket 가격을 변형에 나눠 `price_history`에 저장하게 했다. 순회 한 번으로 Cardmarket 저장과 TCGplayer 상품 ID 연결을 함께 한다.
- 변형 배분은 Q7 규칙에 Q9 결정(TCGplayer 연결이 `holofoil`에만 있으면 기본 필드도 `holofoil`)을 더해 구현했다.
- 개발 DB에서 표본 7세트 1,060장으로 실제 수집하고, 소요 시간·배분 결과·원본 값을 확인했다.

자세한 규칙과 실측값은 `mydocs/tech/price_collector.md` "Cardmarket 수집 (4단계)"에 있다.

## 변경 파일

| 파일 | 변경 내용 |
|------|----------|
| `src/backend/internal/tcgdex/types.go` | `CardmarketPricing`, `Card.Cardmarket()` |
| `src/backend/internal/tcgdex/client_test.go` | Cardmarket 해석 확인 추가 |
| `src/backend/internal/prices/cardmarket.go` | 신규. 변형 배분 규칙(Q7·Q9), 대표값, 0 값 처리 |
| `src/backend/internal/prices/tcgdex.go` | 순회 작업에 Cardmarket 저장 추가 (500행 단위 트랜잭션, 저장 실패 카드 재시도), 실행 상태를 `tcgdexRun`으로 정리 |
| `src/backend/internal/prices/cardmarket_test.go` | 신규. 배분 규칙 13가지, 테스트 DB 멱등성 |
| `src/backend/internal/prices/tcgcsv_test.go` | `priceRows`에 소스 필터, Runner 테스트에 Cardmarket 저장 확인 |
| `mydocs/tech/price_collector.md` | "Cardmarket 수집 (4단계)" 절 추가 |
| `mydocs/orders/20261123.md` | 진행 상황 |

수집기(`cmd/collector`)에는 아직 연결하지 않았다 (5단계). 개발 DB 수집은 임시 실행 파일(`cmd/_pricesonce`)로 했고, 실행 후 지웠다.

## 설계 결정

1. **`Result`의 의미를 Cardmarket 기준으로 바꿨다.** 이 작업의 `fetched`는 받은 카드 상세 수, `linked`·`saved`는 Cardmarket 가격 행, `skipped`는 저장하지 않은 이유 건수이다. TCGplayer 상품 ID 연결 건수는 별도 로그(`tcgdex tcgplayer ids`)로 남긴다.
2. **`0` 값은 모든 필드에서 NULL로 둔다.** 1단계 결론(대표값에서 `0`은 값 없음)을 보조 필드(`low`, `avg1` 등)에도 똑같이 적용했다. 가격 `0`이 저장되면 #8 변동률 계산이 틀어진다.
3. **`-holo` 값이 없으면 후보로 세지 않는다.** 홀로 카드는 대부분 `-holo`가 비어 있어서, 셌다면 건너뜀 건수가 의미 없이 커진다. 값이 있는데 리버스 변형이 없을 때만 `variant_missing`으로 센다.
4. **저장은 500행 단위 트랜잭션이다.** 저장에 실패한 묶음의 카드는 조회 실패 카드와 같은 방식으로 끝에 한 번 더 받는다. 이때 집계가 두 번 세지지 않게 뺀다.
5. **시작할 때 `price_sources.currency`가 `EUR`인지 확인한다.** 소스 통화 고정(#4 원칙)이 DB 설정과 어긋나면 저장하지 않고 바로 실패한다.

## 검증 결과

| 항목 | 결과 |
|------|------|
| `go vet ./...` | 통과 |
| `go build ./...` | 통과 |
| `go test ./...` | 전체 통과. `prices` 2개 테스트 추가 (배분 규칙 13가지, Cardmarket 멱등성), Runner 테스트에 Cardmarket 저장 확인 추가 |
| 테스트 DB 멱등성 | 두 번 실행해도 Cardmarket 4행, 값 같음 |
| 개발 DB 수집 | 1,060장 4분 27초 (초당 약 4장, 1단계 계산과 같음), 실패 카드 0, 저장 1,444행 |
| 배분 | `base1` 무제한판 102 / 102, 초판 0. `sm1` Q9 대상 27장 모두 `holofoil`. `PMCG2` 48장 가격 없음 |
| 원본 대조 | 표본 5장(`sv03.5-001`, `sm1-12`, `base1-4`, `SV2a-001`, `svp-173`)의 모든 필드가 API 응답과 같음 |
| 재실행 (`sv03.5`) | 행 수 불변 (`price_history` 2,583, `external_ids` 1,241, `card_variants` 49,083) |

- 개발 DB에는 표본 세트의 Cardmarket 가격(기준일 2026-11-23)이 남아 있다. 5단계 전체 수집에서 같은 키로 갱신된다.
- 전체 34,588장 기준 예상 소요 시간은 1단계 계산(약 2시간 25분)과 같다. 실측은 5단계 전체 수집에서 한다.

## 발견 사항

1. **일본판 리버스는 Cardmarket에만 가격이 있다.** `SV2a` 리버스 153개는 TCGCSV에서 패턴 상품 두 개로 나뉘어 연결하지 않지만(1단계), Cardmarket `-holo`로는 153개 모두 가격이 붙었다.
2. **Q9는 TCGplayer 연결이 있어야 적용된다.** 빈 DB 첫 실행에서는 Q5 변형(`holofoil`)이 같은 실행의 TCGCSV 마지막 단계에서 만들어지므로, 그런 카드의 첫날 Cardmarket 가격은 `normal`에 붙고 다음 날부터 `holofoil`에 붙는다. 첫날 하루치 이력만 다른 변형에 남는다. 운영 DB 첫 적재 뒤 하루 지나면 해소되므로 별도 처리는 하지 않았다.

## 작업지시자 확인 사항

없음.

## 다음 단계

승인되면 4단계 소스와 이 보고서를 커밋하고 5단계(수집기 연결·원화 환산·실측·문서)를 진행한다. 5단계 전체 수집(`-once`, 약 2시간 30분)은 상시 수집기가 같은 IP에서 TCGdex를 부르므로, 실행 시점을 정할 때 함께 확인을 요청한다.
