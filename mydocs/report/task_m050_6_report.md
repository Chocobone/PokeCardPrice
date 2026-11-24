# Task #6 최종 결과보고서: 시세 수집 스케줄러 (해외 소스)

> **작성일**: 2026-11-24
> **브랜치**: `local/task6`
> **마일스톤**: M050 (v0.5.0)

## 개요

해외 시세 소스 두 곳과 환율을 매일 수집해 #4의 `external_ids`·`price_history`·`fx_rates`에 저장하는 수집기를 만들었다.

- **소스:** TCGplayer 가격은 TCGCSV, Cardmarket 가격은 TCGdex 카드 상세, 환율은 Frankfurter v2이다.
- **실행 일정:** 매일 05:30 KST `collect-daily`에서 카탈로그 갱신 다음에 실행한다. 세 작업은 동시에 돈다.
- **저장 원칙:** 가격은 소스 통화(USD·EUR)로 저장하고, 원화는 조회 쿼리 `ListPriceHistoryKRW`가 직전 환율로 환산한다.

개발 DB 전체 수집 결과 (2026-11-24, 2시간 36분):

| 항목 | 수 |
|------|-----|
| TCGCSV 가격 저장 | 42,476행 (받은 가격 행 75,143) |
| Cardmarket 가격 저장 | 40,434행 (카드 34,588장) |
| 환율 | USD/KRW, EUR/KRW |
| TCGCSV 근거로 추가한 변형 (Q5) | 4,363 |
| 최종 실패 | 0 |

이제 #7·#8은 날짜별 변형 가격 이력과 원화 환산 조회를 바로 쓸 수 있다.

## 수행 결과

| 단계 | 내용 | 커밋 |
|------|------|------|
| 계획 | 수행 계획서, 구현 계획서 (1단계 커밋에 포함) | `01a7dc2` |
| 1단계 | TCGCSV·TCGdex Cardmarket·Frankfurter 응답 확인, 상품 ↔ 변형 연결 두 방법 표본 측정, `-holo` 필드 규칙, 호출 수·소요 시간 계산, Q5~Q7 | `01a7dc2` |
| 2단계 | 어댑터 구조(`Source`·`Runner`), 공용 HTTP(`httpx`), Frankfurter 클라이언트·환율 저장 | `c6aaf1f` |
| 3단계 | TCGCSV 클라이언트·어댑터 (저장된 연결 → TCGdex 대기 → 방법 B), 특별 상품 처리, Q5 변형 추가, Q8·Q9 | `10b9ded` |
| 4단계 | TCGdex Cardmarket 어댑터 (카드 상세 순회 한 번으로 Cardmarket 저장 + TCGplayer 상품 ID 연결), 변형 배분 | `9f2a6f2` |
| 5단계 | 수집기 연결, 원화 환산 조회, 전체 실측, TCGdex 상품 ID 중복 결함 수정, 문서·운영 스크립트 | `fdc401e` |

## 산출물

### 코드

| 패키지·파일 | 내용 |
|-------------|------|
| `internal/prices` | 작업 실행 틀(`Runner`, 작업 간 대기), 환율·TCGCSV·TCGdex 작업, 연결 규칙(`link.go`), Cardmarket 배분(`cardmarket.go`) |
| `internal/tcgcsv` | TCGCSV 클라이언트 (그룹·상품·가격·`last-updated`), 동시성 1·요청 간 120 ms |
| `internal/fx` | Frankfurter v2 클라이언트 |
| `internal/httpx` | 공용 HTTP (전용 User-Agent, 지수 백오프 재시도) |
| `internal/tcgdex` | 카드 상세 `pricing.tcgplayer`·`pricing.cardmarket` 해석 추가 (카탈로그 적재는 그대로) |
| `internal/db/queries/prices.sql` | 외부 ID 일괄 조회·삭제, 세트 카드 변형, 대상 카드, 소스 가격 정리, `ListPriceHistoryKRW` |
| `cmd/collector` | `collect-daily`에 시세 연결, `PRICES_SOURCES`·`PRICES_TCGCSV_GROUPS` |
| `scripts/collector.sh`, `Docker/*` | `PRICES_*` 전달, `status`에 소스별 기준일·행 수·환율 |

### 문서

- `mydocs/tech/price_collector.md`: 조사·측정, 연결 규칙, 실행 흐름, 실측, 중복 결함과 처리, 후속 인계.
- 갱신: `CLAUDE.md`, `mydocs/manual/collector_remote.md`, `mydocs/tech/db_schema.md`, `mydocs/tech/price_data_sources.md`.

### 작업지시자 결정 사항

| # | 결정 |
|---|------|
| Q1 | 연결은 TCGdex의 TCGplayer 상품 ID(방법 A)를 먼저 쓰고, 없으면 그룹 ↔ 세트 + 카드 번호(방법 B)로 대조 |
| Q2 | Cardmarket은 매일 `en`·`ja` 전체 카드 (약 2시간 30분) |
| Q3 | 수집 이력은 로그로만 남김 |
| Q4 | 과거 이력 백필은 별도 이슈 → #18 |
| Q5 | TCGdex에 없는 변형은 TCGCSV 근거로 추가 (영어판, 기본 상품, `normal`·`holofoil`·`reverse_holofoil`) |
| Q6 | TCG Pocket 세트는 시세 대상에서만 제외, 카탈로그는 #17 |
| Q7 | Cardmarket 기본 필드 → `normal` 우선, `-holo` → 리버스 홀로, 초판에는 붙이지 않음 |
| Q8 | 이름에 괄호가 있는 단독 상품은 변형을 추가하지 않음 (5단계 재판단 후 현행 유지) |
| Q9 | TCGplayer 가격이 `holofoil`에만 있으면 Cardmarket 기본 필드도 `holofoil` |
| 5단계 | TCGdex 상품 ID 중복은 연결을 바꾸지 않고, 저장된 연결은 상품 번호로 확인해 틀리면 정리. 검증은 줄인 범위로 |

### 사용 방법

```bash
# Docker/ 에서
docker compose logs -f collector                          # 상시 수집기 로그
docker compose exec backend go run ./cmd/collector -once  # 즉시 1회 (카탈로그 + 시세, 약 2시간 40분)
# 범위 지정 (영어판 TCGCSV 그룹은 ID로)
docker compose exec -e CATALOG_LANGUAGES=en -e CATALOG_SETS=sv03.5 -e PRICES_TCGCSV_GROUPS=23237 backend go run ./cmd/collector -once
# 시세 끄기 / 일부만
docker compose exec -e PRICES_SOURCES=none backend go run ./cmd/collector -once
docker compose exec -e PRICES_SOURCES=fx,tcgcsv backend go run ./cmd/collector -once
```

## 발견 사항

- **TCGdex가 서로 다른 카드에 같은 TCGplayer 상품 ID를 준다** (30개 상품, 예: `bw11-6`·`bw11-RC1`, `ru1-5`·`ru1-6`). 처음 구현에서는 마지막에 처리한 카드가 연결을 가져가 가격이 다른 카드에 붙었다. 5단계에서 중복 상품은 연결을 바꾸지 않고, 저장된 연결을 상품 번호로 확인하도록 고쳤다.
- **TCGdex가 특별 상품 ID를 주는 경우가 있다** (`svp-173` → Pokemon Center Exclusive). 같은 번호의 기본 상품이 있으면 저장된 연결을 쓰지 않는다 (3단계).
- **TCGdex 카탈로그에 변형이 빠진 세트가 많다** (`sm1`~`sm12` 등 리버스·홀로 없음). TCGCSV 근거로 4,363개 변형을 추가했다 (Q5).
- **일본판은 방법 A를 쓸 수 없다.** TCGdex 일본판에는 TCGplayer 상품 ID가 없어서, 그룹 이름 코드 대응 + 번호 대조만 쓴다. 일본판 리버스(패턴 상품)는 Cardmarket에만 가격이 있다.
- **TCGCSV 그룹 ↔ 세트를 대응시키지 못한 그룹이 320개**로, 건너뛴 가격 행(32,667)의 절반 이상(17,295)이 여기서 나온다. 트레이너 키트처럼 그룹 하나에 세트 둘이 섞인 경우도 있다.
- **네트워크 끊김**: 첫 전체 수집이 약 50분의 DNS 장애 끝에 중단됐다. 수집기는 카드마다 재시도한 뒤 실패로 넘기며 계속 진행하고, 실패 카드는 끝에 다시 받는다.

## 검증

| 항목 | 결과 |
|------|------|
| 빌드 (`go build ./...`, `go vet ./...`, 변경 파일 `gofmt`) | ✅ |
| 단위 테스트 (`go test ./...`, DB 테스트 포함) | PASS 111 (하위 테스트 포함), FAIL·SKIP 0 |
| 전체 수집 | 2시간 36분, 최종 실패 0 ✅ |
| 재실행 멱등성 | 표본 세트(3·4단계)와 결함 수정 후 축소 범위(15세트, 2회)에서 행 수 불변, 연결 변경 0 ✅ |
| 원본 대조 | TCGCSV 667건(3단계), Cardmarket 5장(4단계)의 모든 필드 일치, 중복 상품 표본 연결이 TCGCSV 상품 번호·이름과 일치 (5단계) ✅ |
| 원화 환산 | 주말 직전 환율·첫 환율 이전 NULL 테스트, 실행 계획 기본 키 범위 스캔 ✅ |
| 회귀 | 카탈로그·서버·스케줄러 테스트 통과 |

## 계획 대비 차이

| 계획 | 실제 | 사유 |
|------|------|------|
| 소스별 어댑터 (TCGCSV, Cardmarket) | TCGdex 작업 하나가 Cardmarket 저장과 TCGplayer 상품 ID 연결을 함께 하고, TCGCSV는 그 작업을 기다림 | 같은 카드 상세 응답을 두 번 받지 않으려고 (1단계 검토) |
| 재시도 로직은 각 클라이언트에 | 공용 `internal/httpx` (2단계) | 소스마다 같은 재시도·요청 간격 로직이라 묶음 (`tcgdex`는 #5 코드 유지) |
| TCGdex 변형에만 연결 | Q5로 변형 추가 | 1단계에서 TCGdex 변형 누락 발견 |
| 전체 실측 후 재실행으로 멱등성 확인 | 전체 실측 1회 + 결함 수정 후 축소 범위 2회 | 실측 중 결함 발견, 작업지시자 지시로 검증 범위 축소 |
| — | 상품 번호로 저장된 연결 확인, 번호 정규화 접미사 처리 | 5단계에서 발견한 TCGdex 상품 ID 중복 |

## 후속 조치

- 후속 타스크(#7·#8·#17·#18) 인계 사항은 `mydocs/tech/price_collector.md` "후속 타스크 인계"에 정리했다.
- **#18 해외 시세 과거 이력 백필**을 M050에 등록했다 (Q4).
- 상시 수집기는 2026-11-24 20:13에 이번 코드로 다시 시작했다. 다음 실행은 2026-11-25 05:30 KST이다. 이 실행 로그의 `duplicate_product`·`links_removed`로 전체 범위의 수정 효과를 확인할 수 있다.
- 번호까지 같은 중복(트레이너 키트 6세트 168장)은 풀지 못했다. 저장된 연결이 틀렸을 수 있다.

## 이슈

- https://github.com/Chocobone/PokeCardPrice/issues/6
- 후속: https://github.com/Chocobone/PokeCardPrice/issues/18
