# Task #6 단계2 완료 보고서: 어댑터 구조와 환율 수집

## 수행 내용

- 외부 소스 공용 HTTP 클라이언트 `internal/httpx`를 만들었다. 요청 간격 공유, 일시 오류만 재시도, 4xx 즉시 실패를 TCGCSV(3단계)와 Frankfurter가 함께 쓴다.
- Frankfurter v2 클라이언트 `internal/fx`를 만들었다.
- `internal/prices`에 작업 인터페이스 `Source`, 동시 실행기 `Runner`, 환율 작업 `FX`를 만들었다.
- 개발 DB에서 환율을 실제로 1회 수집해 저장을 확인했다.

## 변경 파일

| 파일 | 변경 내용 |
|------|----------|
| `src/backend/internal/httpx/client.go` | 신규. 공용 HTTP 클라이언트 (`GetJSON`, `GetText`, `Sleep`) |
| `src/backend/internal/httpx/client_test.go` | 신규. 5xx·429 재시도, 재시도 한도, 404·422 즉시 실패, JSON 형식 오류, 요청 간격 공유, 취소, `Retry-After` |
| `src/backend/internal/fx/client.go` | 신규. Frankfurter v2 `Rates` |
| `src/backend/internal/fx/client_test.go` | 신규. 날짜 지정·최신 값·빈 응답·422·0 환율 |
| `src/backend/internal/prices/source.go` | 신규. `Source`, `Options`, `Result` |
| `src/backend/internal/prices/runner.go` | 신규. `Runner` (동시 실행, 실패·패닉 격리, 요약 로그) |
| `src/backend/internal/prices/fx.go` | 신규. 환율 작업 `FX` |
| `src/backend/internal/prices/prices_test.go` | 신규. Runner 동시 실행·실패 격리·작업 선택, 환율 저장 멱등성·부분 실패 (테스트 DB) |
| `mydocs/tech/price_collector.md` | "구현 결과" 절 추가 |
| `mydocs/orders/20261119.md` | 진행 상황 |

수집기(`cmd/collector`)에는 아직 연결하지 않았다 (5단계).

## 설계 결정

1. **`httpx` 공용 클라이언트를 만들었다.** 구현 계획서는 클라이언트마다 재시도 로직을 두고 중복이 크면 제안하기로 했으나, TCGCSV·Frankfurter 두 곳이 같은 코드를 쓰게 되어 이번 단계에서 묶었다. `tcgdex` 클라이언트는 상시 수집기가 쓰는 #5 코드라 그대로 두었다 (중복 약 100줄).
2. **환율도 `Source`로 만들었다.** 구현 계획서는 "환율 → 가격 소스" 순서였으나, 가격은 원래 통화로 저장하고 환산은 조회할 때 하므로 환율이 먼저 끝날 필요가 없다. 세 작업을 모두 동시에 돌리고, 개발용 `Options.Sources`로 하나씩 실행할 수도 있다.
3. **작업 오류는 합쳐서 돌려주되 다른 작업을 취소하지 않는다.** `errgroup`은 첫 오류에서 나머지를 취소하므로 쓰지 않았다 (`sync.WaitGroup.Go`).

## 검증 결과

| 항목 | 결과 |
|------|------|
| `go vet ./...` | 통과 |
| `go build ./...` | 통과 |
| `go test ./...` | 전체 통과 (`httpx` 8개, `fx` 3개, `prices` 4개 테스트 추가. DB 테스트는 테스트 DB `cardmarket_test_prices`에서 실행) |
| 환율 멱등성 | 같은 날 두 번 실행 → 2행, 값만 갱신. 다음 날 값 → 3행 |
| 개발 DB 실제 수집 | `USD/KRW 2026-11-19 1,332.93`, `EUR/KRW 2026-11-19 1,464.90`, 호출 2회, 1.4초 |

- 개발 DB 수집은 임시 실행 파일(`cmd/_fxonce`)로 했고, 실행 후 지웠다.

## 다음 단계

승인되면 3단계(TCGCSV 어댑터)를 진행한다. 1단계 승인안에 따라 TCGdex 카드 상세에서 TCGplayer 상품 ID를 꺼내는 부분(방법 A)도 3단계에서 함께 구현한다.
