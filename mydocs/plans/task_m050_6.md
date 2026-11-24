# Task #6: 시세 수집 스케줄러 (해외 소스) — 수행 계획서

## 수행 목표

해외 시세 소스 두 곳(TCGplayer 가격은 TCGCSV, Cardmarket 가격은 TCGdex)과 환율(Frankfurter v2)을 매일 수집해 #4에서 만든 `external_ids`, `price_history`, `fx_rates`에 저장한다.
소스마다 어댑터를 두고 고루틴으로 병렬 수집하며, 실패는 재시도하고 결과를 로그로 남긴다.
이 타스크가 끝나면 #7(일본판 참고가)과 #8(시세 조회 API)이 날짜별 가격 이력과 환율을 바로 쓸 수 있다.

## 범위

1. **사전 확인** (1단계, 결과 승인 후 구현)
   - TCGCSV: 카테고리(`Pokemon` 3, `Pokemon Japan` 85)·그룹·상품·가격 응답 필드 재확인, 갱신 시각과 `last-updated` 확인 방법.
   - **TCGCSV 상품 ↔ 카드 변형 연결 규칙** (Q1): TCGdex 카드 상세의 TCGplayer 상품 ID(`variants_detailed[].thirdParty.tcgplayer`)와, TCGCSV 그룹 ↔ TCGdex 세트 + 카드 번호(`extendedData` `Number`) 대조를 표본 세트(`en`·`ja` 신·구 세트)로 비교해 연결률과 오연결을 측정한다.
   - TCGdex Cardmarket: `pricing.cardmarket` 필드, `-holo` 필드를 어느 변형에 넣을지(#4 인계), 하루 호출 수와 소요 시간(Q2).
   - Frankfurter v2: KRW 기준 USD·EUR 응답, 주말·공휴일 동작, 호출 한도(#2 미해결 사항).
2. **소스 어댑터 구조**: 소스마다 "외부 ID 연결 → 가격 조회 → 대표값 계산 → 저장"을 같은 인터페이스로 구현한다. 새 소스는 어댑터 하나를 더해 붙인다.
3. **환율 수집**: Frankfurter v2로 USD→KRW, EUR→KRW를 조회해 `fx_rates`(provider `frankfurter`)에 저장한다. 값이 없는 날은 저장하지 않고 조회 시 직전 값을 쓴다(`GetFxRateOnOrBefore`).
4. **TCGCSV 어댑터**: `Pokemon`·`Pokemon Japan` 그룹별 상품·가격을 받아 1번에서 정한 규칙으로 `external_ids`(`productId` + `subTypeName`)를 채우고 `price_history`(소스 `tcgcsv`, 대표값 `market` → 없으면 `mid`)에 저장한다. 카드 변형과 연결되지 않는 상품(박스·팩 등)은 저장하지 않고 건수만 남긴다.
5. **TCGdex Cardmarket 어댑터**: `en`·`ja` 카드 상세의 `pricing.cardmarket`을 받아 `price_history`(소스 `tcgdex_cardmarket`, 대표값 `trend` → 없으면 `avg`)에 저장한다. `tcgdex.Client`(#5)를 그대로 써서 요청 간격·재시도를 공유한다.
6. **병렬 수집과 실패 처리**: 환율을 먼저 받고, 두 가격 소스는 고루틴으로 동시에 실행한다. 소스마다 동시성·요청 간격 상한을 지키고(TCGCSV 동시성 1·요청 간 100 ms 이상, TCGdex는 #5 클라이언트 설정), 요청 단위 지수 백오프 재시도 + 실행 끝 실패 항목 재시도를 한다. 한 소스의 실패가 다른 소스를 멈추지 않는다.
7. **원화 환산 조회**: 원화 값은 저장하지 않고, `price_history`와 `fx_rates`를 합쳐 원화로 바꾸는 조회 쿼리를 추가해 #8이 쓸 수 있게 한다(#4 원칙: 원본 통화로 저장).
8. **수집기 연결**: `collect-daily`(05:30 KST)에서 카탈로그 다음에 시세 수집을 실행한다(TCGCSV 갱신 약 05:00 KST 이후). `-once`로 수동 실행하고, 개발용으로 소스·언어·세트를 좁힐 수 있게 한다. 소스별 처리 건수·연결 실패·소요 시간을 로그로 남긴다.
9. **테스트·문서**: 어댑터는 고정 응답(`httptest`), 저장·환산 쿼리는 테스트 DB로 검증한다. 조사 결과·연결 규칙·운영 방법은 기술 문서와 `CLAUDE.md`에 반영한다.

### 범위 밖

- 일본판 참고가 산출, 사용자 거래가·관리자 입력가 → #7
- 시세 조회·순위·변동률 API → #8 (이 타스크는 원화 환산 조회 쿼리까지)
- 과거 이력 백필(TCGCSV 일별 압축 파일, 2024-02-08~) → Q4
- 한국수출입은행 환율(대체 소스, 키 발급 필요) — 어댑터 자리만 고려
- 카드 이미지 저장, 공개 배포 전 이용 조건 재확인 → #16

## 핵심 설계 원칙

- **원본 그대로 저장**: 가격은 소스 통화(USD·EUR)와 소스 필드 그대로 저장하고, `price_date`는 소스 기준일로 한다. 원화는 조회 시 환산한다.
- **멱등 저장**: `UpsertExternalID`·`UpsertPriceHistory`·`UpsertFxRate`(`ON CONFLICT ... DO UPDATE`)를 써서 같은 날 다시 실행해도 행이 늘지 않는다.
- **소스에 대한 배려**: TCGCSV 규칙(하루 1회 동기화, 요청 간 100 ms, 24시간 1만 회 미만, 전용 User-Agent, 사용자 요청마다 직접 호출 금지)과 TCGdex 요청 간격을 지킨다.
- **부분 실패 허용**: 소스·그룹·카드 단위로 실패를 가두고, 실패 항목은 로그로 남겨 실행 끝과 다음 실행에서 다시 시도한다.
- **틀린 연결보다 빈 연결**: 상품과 변형을 확실히 대응시킬 수 없으면 연결하지 않고 건수를 남긴다. 잘못된 가격이 붙는 것이 가격이 없는 것보다 나쁘다.
- **오래된 가격 구분**: 소스 갱신일(`source_updated_at`)을 함께 저장해 #8이 오래된 가격을 현재가로 보이지 않게 한다.
- **조사 결과 승인 후 구현**: 연결 규칙(Q1)과 Cardmarket 수집 빈도(Q2)는 1단계 측정 결과를 문서로 제시하고 작업지시자 승인 후 구현한다.

## 위험과 대응

| 위험 | 대응 |
|------|------|
| TCGCSV 상품과 카드 변형의 연결률이 낮거나 오연결이 많음 | 1단계에서 두 방법을 표본으로 측정하고, 확실한 연결만 쓰는 규칙을 정한다. 연결 실패 상품 목록을 로그·문서로 남긴다 |
| `variants_detailed`는 TCGdex 문서에 없는 필드 | 연결에 쓰더라도 값이 없거나 형식이 바뀌면 세트+번호 대조로 대신하거나 연결하지 않는다. 이미 저장한 `external_ids`는 유지한다 |
| TCGdex Cardmarket 가격은 카드 단건 조회에만 있어 하루 약 3만 6천 회 요청, 수 시간 소요 (#4 인계) | 1단계에서 소요 시간을 실측해 Q2로 빈도·대상을 정한다. 카탈로그 매월 1일 전체 갱신(약 3시간)과 같은 날 겹치는 일정을 고려한다 |
| TCGdex 수 시간 단위 `503` (#5 확인) | #5와 같은 재시도 방식(요청 단위 + 실행 끝 재시도)을 쓴다 |
| 상시 수집기가 실행 중 (#5) | 개발은 소스·세트 필터와 `-once`로 하고, 상시 수집기 재시작은 단계 승인 후 작업지시자와 정한다 |

## 확인 사항 (2026-11-19 작업지시자 결정: Q1~Q4 모두 제안)

| # | 사항 | 제안 | 대안 | 결정 |
|---|------|------|------|------|
| Q1 | TCGCSV 상품 ↔ 카드 변형 연결 방법 | **1단계 측정 후 결정.** 잠정안: TCGdex `thirdParty.tcgplayer` 상품 ID를 우선 쓰고, 없으면 그룹↔세트 + 카드 번호 + `subTypeName`으로 대조 | 세트+번호 대조만 사용 (문서에 없는 필드에 의존하지 않음) | 제안 (1단계 측정 후 확정) |
| Q2 | Cardmarket(TCGdex) 수집 빈도 | **1단계 실측 후 결정.** 잠정안: 매일 전체 카드 | 주 1회 전체 + 매일은 가격이 있었던 카드만 / 매일 최근 세트만 | 제안 (1단계 실측 후 확정) |
| Q3 | 수집 이력 기록 방식 | **로그만** 남긴다 (소스별 건수·실패·소요 시간). 테이블 추가 없음 | `collection_runs` 테이블을 추가해 실행 결과를 저장 (#8·#13에서 "마지막 수집 시각" 표시에 사용 가능) | 제안 |
| Q4 | 과거 이력 백필 (TCGCSV 압축 파일, 약 15 GB 추가 추정, #4 8절) | **이 타스크에서 하지 않는다.** 이용 조건 판단과 함께 별도 이슈로 남긴다 | 최근 90일 등 기간을 정해 이 타스크에서 적재 | 제안 |

## 기존 구현 현황

- `src/backend/migrations/00003_prices.sql`: `price_sources`(`tcgcsv`, `tcgdex_cardmarket` 등 5행), `external_ids`, `price_history`(넓은 형태), `fx_rates`
- `src/backend/internal/db/queries/prices.sql`: `GetPriceSourceByCode`, `UpsertExternalID`, `FindVariantByExternalID`, `UpsertPriceHistory`(`:batchexec`), `ListPriceHistory`, `UpsertFxRate`, `GetFxRateOnOrBefore`
- `src/backend/internal/tcgdex/`: TCGdex 클라이언트 (요청 간격 250 ms·동시성 2, 재시도). 카드 상세 응답 타입에 가격 필드는 아직 없음
- `src/backend/internal/catalog/`: 카탈로그 적재, 변형 매핑 7종(TCGCSV `subTypeName`과 1:1)
- `src/backend/cmd/collector/main.go`: `collect-daily` 작업에서 카탈로그만 실행. "해외 시세(#6)는 카탈로그 뒤에 이어 붙인다" 주석
- `mydocs/tech/price_data_sources.md`: 소스 확정안, #6 설계 입력
- `mydocs/tech/db_schema.md`: 6절 대표값 규칙, 8절 시세 규모, 10절 #6 인계
- `mydocs/tech/catalog_collector.md`: 5절 변형 매핑, 8절·후속 타스크 인계

## 산출물

- `mydocs/tech/price_collector.md`: 소스 응답 확인, 상품 ↔ 변형 연결 규칙과 측정 결과, Cardmarket 필드 매핑, 환율 동작, 운영 방법 (신규, 이름은 구현 계획서에서 확정)
- `src/backend/internal/tcgcsv/`, `src/backend/internal/fx/`, `src/backend/internal/prices/`: TCGCSV 클라이언트, 환율 클라이언트, 어댑터·수집 로직 (신규, 패키지 구성은 구현 계획서에서 확정)
- `src/backend/internal/tcgdex/`: 카드 상세 가격 필드 추가
- `src/backend/internal/db/queries/prices.sql`: 연결·저장·원화 환산 쿼리 추가와 생성 코드
- `src/backend/cmd/collector/`: 시세 수집 작업 연결
- 테스트, `mydocs/working/task_m050_6_stage{N}.md`, `mydocs/report/task_m050_6_report.md`

## 구현 계획서

`mydocs/plans/task_m050_6_impl.md` 참조 (본 계획서 승인 후 작성)

## 브랜치

`local/task6`

## 예상 단계: 5단계

1. 사전 확인: TCGCSV·TCGdex 가격·Frankfurter 응답, 상품 ↔ 변형 연결 측정, Cardmarket 소요 시간 → 작업지시자 승인 (Q1·Q2 확정)
2. 어댑터 구조와 환율 수집·저장
3. TCGCSV 어댑터: 외부 ID 연결과 가격 저장
4. TCGdex Cardmarket 어댑터
5. 수집기 연결(병렬 실행·재시도·로그), 원화 환산 쿼리, 실측, 문서 정리
