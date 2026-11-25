# Task #8: 카드 검색·시세 조회 API — 수행 계획서

## 수행 목표

#5 카탈로그와 #6 해외 시세(`price_history`·`fx_rates`)를 읽어 카드 검색, 카드 상세, 시세 추이·변동률, 급등·급락 순위를 HTTP API로 제공한다.
#12에서 정한 API 계약(`mydocs/tech/api_contract.md`)을 따르고, 끝나면 프론트엔드 목업(`VITE_API_MOCK`)을 실제 API로 바꾼다.
#13(카드 상세 화면)·#10(내 컬렉션)이 이 API와 응답 타입을 그대로 쓸 수 있게 한다.

## 범위

1. **계약 검토·보완**: `api_contract.md`의 검색·순위 계약을 확정하고, 카드 상세·시세 추이 엔드포인트를 새로 정한다. 대표가·순위 산출 기준·검색 정렬(계약이 #8로 미룬 항목)을 Q1~Q6로 정한다.
2. **`GET /cards` 카드 검색**: 카드 이름(영·일·한), 한글 종 이름, 세트 이름·ID, 카드 번호(앞 0 무시)로 공백 낱말 AND 검색, 언어 필터, 쪽 넘김, 카드별 대표가.
3. **`GET /cards/{id}` 카드 상세** (신규): 카드·세트 정보, 한글판 ↔ 일본판 연결, 변형 × 소스별 최신가(원화 환산 포함).
4. **`GET /cards/{id}/prices` 시세 추이** (신규): `period`=`7d`·`30d`·`90d`·`1y`, 변형 × 소스별 날짜 계열(원 통화 + 원화)과 요약(현재가, 기간 평균·최저·최고, 변동률).
5. **`GET /rankings/movers` 급등·급락 순위**: 계약대로 `7d`·`30d`, `up`·`down`. 저가 카드 잡음과 오래된 가격을 거르는 기준을 둔다.
6. **공통**: 계약의 오류 형식(`{"error": {code, message}}`)으로 오류 처리기를 바꾸고, 매개변수 검증, 금액 문자열(`decimal`) 직렬화, 원화 환산(직전 환율, 원 단위 반올림)을 한 곳에 둔다.
7. **프론트엔드 전환**: `types.ts`를 바뀐 계약에 맞추고(통화 `EUR` 추가 등), `.env.development`의 `VITE_API_MOCK`을 `false`로 바꿔 홈·검색 화면이 실제 API로 동작하는지 확인한다. 화면 추가는 하지 않는다.
8. **테스트·문서**: 쿼리는 테스트 DB, 핸들러는 `httptest`로 검증하고, 개발 DB 전체 적재 상태에서 응답 시간을 잰다. `api_contract.md`·`CLAUDE.md`를 갱신한다.

### 범위 밖

- 카드 상세 화면 → #13 (이 타스크는 API까지)
- 한글판 카드 가격(일본판 참고가), 국내 거래가 → #7. 이 타스크에서 한글판 카드의 대표가는 `null`이다 (Q1)
- 응답 캐시·성능 정비(집계 테이블, HTTP 캐시 헤더) → #11. 이 타스크는 쿼리와 인덱스로 응답 시간 목표만 맞춘다 (Q6)
- 인증·내 컬렉션 → #9·#10
- 과거 이력 백필 → #18 (지금 이력은 2026-11-23부터라 `30d`·`90d`·`1y`는 데이터가 쌓일 때까지 짧다)

## 핵심 설계 원칙

- **계약 우선**: 응답 형식은 `api_contract.md`가 기준이다. 바꾸면 문서와 `types.ts`를 같은 커밋에서 고친다.
- **원본 통화 + 원화 함께**: 가격은 소스 통화(USD·EUR) 그대로 보내고, 원화 환산값을 옆에 붙인다. 환산은 `ListPriceHistoryKRW`와 같은 규칙(가격 기준일 이전 가장 가까운 환율, 원 단위 반올림)을 Go 한 곳에서 한다.
- **변형 단위가 기본**: 소스마다 붙는 변형이 다르므로(#6 Q9, 초판) 상세·추이는 변형 × 소스로 보여 주고, 카드 하나의 대표가는 Q1 규칙으로만 고른다.
- **오래된 가격은 숨기지 않고 표시**: `source_updated_at`·가격 기준일을 응답에 넣어 화면이 판단하게 하되, 순위에서는 기준일이 아닌 가격을 뺀다.
- **읽기 전용**: API 서버는 DB에 쓰지 않는다. 수집기와 같은 DB를 쓰되 쓰기 경로와 섞지 않는다.
- **sqlc로 쿼리 관리**: 새 쿼리는 `internal/db/queries/`에 두고 생성 코드를 함께 커밋한다. 동적 조건(검색 낱말 수)은 배열 매개변수로 풀어 정적 쿼리로 유지한다.

## 확인 사항 (2026-11-25 작업지시자 결정: Q1~Q6 모두 제안)

| # | 사항 | 제안 | 대안 |
|---|------|------|------|
| Q1 | 카드 대표가(`CardSummary.price`) 고르는 규칙 | 소스 우선순위 **TCGCSV(USD) → Cardmarket(EUR)**, 변형 우선순위 `normal` → `holofoil` → `unlimited_holofoil` → `unlimited` → `reverse_holofoil` → `first_edition_holofoil` → `first_edition`. 각 변형의 최신가 중 처음 맞는 것. 한글판은 `null`(#7) | 변형 중 최저가 / 일본판은 Cardmarket 우선 |
| Q2 | 원화 환산값을 응답에 둘지 | `Price`에 `krw: string \| null`(원 단위)과 `fxDate`를 추가. 통화에 `EUR` 추가 | 원 통화만 보내고 화면이 환율 API로 환산 |
| Q3 | 급등·급락 순위 기준 | **TCGCSV(USD) 변형 단위**. 기준일 = TCGCSV 최신 `price_date`, 비교일 = 기준일 − 7일/30일(그날 없으면 그 이전 가장 가까운 날). **두 시점 가격 모두 $1 이상**인 변형만, 같은 카드는 변동률이 가장 큰 변형 1개만 | Cardmarket도 따로 순위 / 기준 금액 $0.5·$5 |
| Q4 | 검색 정렬 | ① 세트 ID·카드 번호가 정확히 맞는 카드 → ② 이름이 검색어로 시작 → ③ 나머지. 같은 순위 안에서는 세트 발매일 최신순, 카드 번호순 | 대표가 높은 순 / 발매일순만 |
| Q5 | 시세 추이의 변동률·평균 정의 | 변동률 = (기간 내 마지막 가격 − 기간 내 첫 가격) / 첫 가격, 평균 = 기간 내 날짜별 대표값 산술 평균. 기간보다 이력이 짧으면 있는 만큼으로 계산하고 실제 시작일(`from`)을 응답에 넣는다 | 기간 시작일에 가격이 없으면 변동률 `null` |
| Q6 | 응답 시간 목표와 집계 방식 | 집계 테이블 없이 조회 시 계산. 개발 DB 전체 적재 상태에서 검색·상세·추이 **p95 100 ms 이하**, 순위 **300 ms 이하**를 목표로 측정하고, 넘으면 인덱스를 추가한다(마이그레이션). 캐시는 #11 | 순위를 수집기 실행 끝에 미리 계산해 테이블에 저장 |

## 위험과 대응

| 위험 | 대응 |
|------|------|
| 한글 종 이름·세트 이름·번호를 합친 AND 검색이 느림 (짧은 검색어는 trigram 불가, #4 S8) | 낱말마다 조건을 만들어 기존 trigram·GIN 인덱스를 쓰게 하고, 1단계에서 `EXPLAIN ANALYZE`로 측정한다. 느리면 검색용 생성 열 또는 인덱스를 추가한다 |
| 순위 쿼리가 행 수에 비례해 느려짐 (#4 S3: 30일치 24.5 ms) | 보조 인덱스 `price_history_source_id_price_date_idx`를 쓰는 형태로 쓰고 실측한다. 이력이 쌓인 뒤 문제가 되면 #11에서 집계로 바꾼다 |
| 이력이 3일치라 `30d`·`90d`·`1y` 추이와 `30d` 순위를 실데이터로 검증할 수 없음 | 테스트 DB에 날짜를 만든 고정 데이터로 검증하고, 개발 DB에서는 있는 기간으로만 확인한다. 규모 측정은 #4 `poc_task4` 방식처럼 임시 DB에 날짜를 복제해 잰다 |
| 계약 변경(통화·원화 필드)으로 프론트엔드 목업·화면이 깨짐 | 계약 변경은 `types.ts`·목업을 같은 단계에서 고치고 `typecheck`·`lint`·`build`로 확인한다 |
| 오류 응답 형식 변경이 `/healthz` 등 기존 응답과 다름 | 오류 처리기만 계약 형식으로 바꾸고 `/healthz` 정상 응답은 그대로 둔다. 기존 서버 테스트를 함께 고친다 |
| Docker가 꺼져 있어 오늘 상시 수집이 빠졌을 수 있음 | 구현 착수 전에 확인하고, 빠졌으면 작업지시자와 수동 실행 여부를 정한다 |

## 기존 구현 현황

- `src/backend/internal/server/server.go`: Echo v5 라우터, `/healthz`, 오류 처리기(`{"error": "..."}` 형식 — 계약과 다름)
- `src/backend/cmd/server/main.go`: API 서버 진입점 (DB 풀, 마이그레이션, 종료 처리)
- `src/backend/internal/db/queries/catalog.sql`: `GetCard`, `ListCardVariants`, `ListCardsByKoSpeciesName` 등 카탈로그 쿼리
- `src/backend/internal/db/queries/prices.sql`: `ListPriceHistory`, `ListPriceHistoryKRW`(원화 환산 추이), `GetFxRateOnOrBefore`
- `src/backend/migrations/00002_catalog.sql`·`00003_prices.sql`: `cards_name_trgm_idx`, `cards_dex_ids_idx`, `price_history_source_id_price_date_idx` 등
- `mydocs/tech/api_contract.md`: 검색·순위 계약 초안 (#12)
- `mydocs/tech/db_schema.md` 7절: 대표 쿼리 S1~S8 측정값
- `mydocs/tech/price_collector.md` "후속 타스크 인계": 소스 기준일 차이, 변형별 소스 차이
- `src/frontend/src/api/`: `types.ts`·`client.ts`·`cards.ts`·`mock/` (계약대로 동작하는 목업)

## 산출물

- `src/backend/internal/server/`: 라우트·핸들러·매개변수 검증·오류 형식 (파일 구성은 구현 계획서에서 확정)
- `src/backend/internal/db/queries/cards.sql`(가칭): 검색·상세·추이·순위 쿼리와 생성 코드
- 필요 시 인덱스 마이그레이션 `00005_*.sql`
- `src/frontend/src/api/types.ts`·목업 갱신, `.env.development` 전환
- `mydocs/tech/api_contract.md` 확정판, `CLAUDE.md` 갱신
- 테스트, `mydocs/working/task_m050_8_stage{N}.md`, `mydocs/report/task_m050_8_report.md`

## 구현 계획서

`mydocs/plans/task_m050_8_impl.md` 참조 (본 계획서 승인 후 작성)

## 브랜치

`local/task8`

## 예상 단계: 5단계

1. 계약 확정과 쿼리 설계: 상세·추이 엔드포인트 정의, 검색·순위 쿼리 초안을 개발 DB에서 `EXPLAIN ANALYZE`로 측정 → 작업지시자 승인
2. 공통 기반(오류 형식, 매개변수 검증, 금액·원화 직렬화)과 `GET /cards` 검색
3. `GET /cards/{id}` 상세와 `GET /cards/{id}/prices` 추이
4. `GET /rankings/movers` 순위와 규모 측정
5. 프론트엔드 실제 API 전환, 응답 시간 실측, 문서 정리
