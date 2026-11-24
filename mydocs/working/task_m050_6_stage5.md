# Task #6 단계5 완료 보고서: 수집기 연결·원화 환산·실측·문서

## 수행 내용

- `collect-daily`(05:30 KST)에서 카탈로그 다음에 해외 시세(환율·TCGCSV·Cardmarket)를 실행하게 연결했다. 카탈로그가 실패해도 시세는 기존 카드로 수집한다.
- 원화 환산 조회 `ListPriceHistoryKRW`(직전 환율 결합)를 추가하고 실행 계획을 확인했다.
- 개발 DB에서 전체 `-once`를 실측했다 (2시간 36분).
- 실측 중 발견한 **TCGdex 상품 ID 중복으로 가격이 다른 카드에 붙는 결함**을 작업지시자 승인(2026-11-24)을 받아 고치고, 줄인 범위로 검증했다.
- 문서(`price_collector.md`, `CLAUDE.md`, `collector_remote.md`, `db_schema.md`, `price_data_sources.md`)와 설정(`.env.example`, `docker-compose.yml`, `scripts/collector.sh`)을 갱신했다.

자세한 규칙과 실측값은 `mydocs/tech/price_collector.md`의 다음 절에 있다: "전체 수집 실측 (5단계)", "TCGdex 상품 ID 중복과 저장된 연결 확인 (5단계)", "원화 환산 조회 (5단계)", "수집기 연결과 운영 (5단계)".

## 변경 파일

| 파일 | 변경 내용 |
|------|----------|
| `src/backend/cmd/collector/main.go` | `collect-daily`에 시세 수집 연결. 카탈로그·시세가 TCGdex 클라이언트 하나를 공유(요청 간격 공유), 두 오류를 합쳐 반환 |
| `src/backend/cmd/collector/catalog.go` | TCGdex 클라이언트를 인자로 받음 |
| `src/backend/cmd/collector/prices.go` | 신규. `PRICES_SOURCES`(`none`이면 끔)·`PRICES_TCGCSV_GROUPS`, 언어·세트는 `CATALOG_*` 공유 |
| `src/backend/cmd/collector/prices_test.go` | 신규. 환경 변수 해석·기본값·오류 |
| `src/backend/internal/db/queries/prices.sql` | `ListPriceHistoryKRW` 추가. `ListExternalIDLinks`에 카드 ID·번호 추가. `DeleteExternalIDLink`, `DeleteVariantSourcePrices` 추가 |
| `src/backend/internal/db/dbq/prices.sql.go` | sqlc 생성 코드 |
| `src/backend/internal/db/queries_test.go` | 원화 환산: 첫 환율 이전, 주말 직전 값, 다른 제공처·통화 무시 |
| `src/backend/internal/prices/tcgdex.go` | TCGplayer 연결을 실행 끝에 일괄 저장, 여러 카드가 같은 상품 ID를 주면 바꾸지 않음 (`duplicate_product`) |
| `src/backend/internal/prices/tcgcsv.go` | 저장된 연결의 카드 번호와 상품 번호가 다르면 연결·가격을 지우고 번호 대조로 다시 연결 (`links_removed`) |
| `src/backend/internal/prices/link.go` | 번호 정규화: 숫자로 시작하는 번호는 접미사가 있어도 앞자리 0 제거 (`050A` → `50A`) |
| `src/backend/internal/prices/tcgcsv_test.go` | 번호가 다른 연결 정리, 중복 상품 미연결, 정규화 사례 추가 |
| `Docker/.env.example`, `Docker/docker-compose.yml`, `scripts/collector.sh` | `PRICES_*` 전달, `status`에 소스별 최근 기준일·행 수·환율 |
| `CLAUDE.md` | 수집기 명령·소요 시간·시세 환경 변수, 시세 수집기 문서 안내 |
| `mydocs/manual/collector_remote.md` | 일정·소요 시간·시세 범위 지정 |
| `mydocs/tech/price_collector.md` | 5단계 절 4개, 연결 규칙 표 갱신 |
| `mydocs/tech/db_schema.md`, `mydocs/tech/price_data_sources.md` | `-holo` 규칙·쿼리 목록, Frankfurter 미해결 사항 결과 |
| `mydocs/orders/20261124.md` | 진행 상황 |

## 설계 결정

1. **카탈로그와 시세는 한 작업 안에서 차례로 돈다.** 시세는 카드가 있어야 저장할 수 있다. 두 작업이 TCGdex 클라이언트 하나를 나눠 써 요청 간격(250 ms)을 지킨다.
2. **시세 범위는 카탈로그 변수를 같이 쓴다.** 영어판 TCGCSV 그룹은 세트 ID로 고를 수 없어 `PRICES_TCGCSV_GROUPS`만 따로 두었다.
3. **원화는 저장하지 않고 조회할 때 환산한다.** 환율은 `price_date` 이전 가장 가까운 값이고, 곱셈은 호출 측에서 `decimal`로 한다 (sqlc가 SQL 계산 결과를 `pgtype.Numeric`으로 만들어서).
4. **중복 상품은 TCGdex 쪽에서 막고, 틀린 연결은 TCGCSV 쪽에서 상품 번호로 걸러 낸다.** 중복만 막으면 이미 틀린 연결이 남고, 번호 대조로 바로잡은 연결을 다음 날 TCGdex가 덮어쓸 수 있다. 번호 확인으로 지운 연결의 변형에서는 이 소스 가격도 지운다. 다른 카드의 가격이기 때문이다. 다만 이번 실행에서 다른 상품으로 저장한 오늘 값은 남긴다.

## 검증 결과

| 항목 | 결과 |
|------|------|
| `go vet ./...` | 통과 |
| `go build ./...` | 통과 |
| `go test ./...` | 전체 통과. 추가: 원화 환산 1, 수집기 환경 변수 3, 번호가 다른 연결 정리 1, 중복 상품 1, 정규화 사례 4 |
| 원화 환산 실행 계획 | 직전 날짜는 `fx_rates_pkey` 역순 Index Only Scan, 값은 기본 키 Index Scan (환율 행 수와 무관) |
| 전체 `-once` 실측 | 2시간 36분. TCGCSV 75,143행 중 42,476 저장, Cardmarket 34,588장에서 40,434행 저장, 환율 2쌍, 최종 실패 0 (503 실패 5장은 끝 재시도에서 성공) |
| 결함 수정 후 축소 검증 | 영어판 15세트 1,042장 + TCGCSV 그룹 13개, 2회 실행. 1회차: 중복 46개 유지, 틀린 연결 37개 정리. 2회차: 지운 연결 0, 연결 변경 0, 행 수 불변 |
| 정규화 수정 후 (`ecard2`, 2회) | 1회차 11행 새로 연결, 2회차 모두 1번 단계에서 저장, 행 수 불변 |
| 원본 대조 | 89381 → `bw11-6`, 87775 → `ru1-3` Ninetales, 84971 → `ecard2-74a`가 TCGCSV 상품 번호·이름과 같음 |

- 작업지시자 지시에 따라 결함 수정 후 전체 재수집(약 2시간 30분)은 하지 않았다. 같은 날 재실행 시 행 수 불변은 축소 범위로 확인했다. 전체 범위의 수정 효과는 다음 정기 실행(11-25 05:30) 로그의 `duplicate_product`·`links_removed`로 확인할 수 있다.
- 상시 수집기(`collector` 컨테이너)는 시작할 때 마운트된 소스를 빌드하므로, 재시작하면 이번 코드로 실행된다. 11-24 14:20 재기동이라 지금은 이전 코드로 떠 있다. 재시작 시점은 작업지시자가 정한다.

## 발견 사항

1. **TCGdex 상품 ID 중복** (위 설계 결정 4). 30개 상품, 카드 약 60장. 전체 수집에서 실행마다 연결이 두 번씩 바뀌었다(로그 92건).
2. **번호도 같은 중복은 풀지 못한다.** 트레이너 키트는 한 TCGCSV 그룹에 두 세트가 섞여(`tk-sm-l`·`tk-sm-r` 등) 그룹 ↔ 세트 대응도 되지 않는다. 저장된 연결이 틀렸을 수 있고, 해당 카드는 6세트 168장이다. 수동 대응표로도 풀 수 없어(그룹 하나에 세트 둘) 현행대로 둔다.
3. **네트워크가 끊기면 끊긴 동안의 카드가 모두 실패로 쌓인다.** 첫 시도에서는 약 50분 끊긴 동안 726장이 실패했다. 끝 재시도로 짧은 끊김은 복구되지만, 실행 도중 프로세스가 종료되면 그날 Cardmarket 일부와 TCGCSV 3번 단계가 빠진다. 다음 날 실행으로 채워진다.
4. **Q8 재판단 (괄호가 붙은 단독 상품).** 전체 수집에서 `variant_missing`은 1,300행이다. 괄호 붙은 단독 상품은 이 안에 들어 있지만 따로 세지 않았다. 전체 연결 행(42,476)의 3% 이하라 현행 유지를 제안한다.

## 작업지시자 확인 사항

1. **Q4 과거 이력 백필을 별도 이슈로 등록할지.** 수행 계획서에서 범위 밖으로 둔 항목이다. 제안: "해외 시세 과거 이력 백필(TCGCSV 아카이브)" 이슈를 M050으로 등록한다.
2. **상시 수집기 재시작 시점.** 다음 정기 실행(11-25 05:30) 전에 재시작하면 수정한 코드로 첫 전체 실행이 된다. 제안: 승인 직후 `docker compose restart collector`.
3. **Q8은 현행 유지할지** (발견 사항 4).

→ 2026-11-24 작업지시자 결정: 세 항목 모두 제안대로 (백필 이슈 등록, 승인 직후 수집기 재시작, Q8 현행 유지).

## 다음 단계

승인되면 5단계 소스와 이 보고서를 커밋하고 최종 결과보고서(`mydocs/report/task_m050_6_report.md`)를 작성한다.
