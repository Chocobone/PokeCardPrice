# Task #8: 카드 검색·시세 조회 API — 4단계 완료 보고서

## 단계 목표

`GET /rankings/movers` 급등·급락 순위를 Q3 규칙(1단계 확정 안 c, 비교일은 순위 전체에 하나)으로 구현하고, 1년치 규모(`poc_task8`)에서 실제 API 서버로 응답 시간을 잰다.

## 변경 파일

| 파일 | 내용 |
|------|------|
| `src/backend/internal/db/queries/cards.sql` | `GetLatestPriceDate`(소스의 지정일 이하 최신 가격 기준일), `ListMovers`(1단계 안 c: 두 날짜를 `MATERIALIZED` CTE로 따로 읽어 해시 조인, $1 이상, 카드당 1개, 상위 N개에만 갱신 시각·카드 정보 조인) |
| `src/backend/internal/db/dbq/cards.sql.go` (생성) | sqlc 생성 코드 |
| `src/backend/internal/api/movers.go` (신규) | `Mover`·`MoversResponse`, 순위 소스(`tcgcsv`)·최소 가격($1)·기간(7d·30d) |
| `src/backend/internal/server/rankings.go` (신규) | 핸들러, 매개변수 검증, 기준일·비교일 결정, 두 날짜 환율 조회 |
| `src/backend/internal/server/server.go` | 라우트 |
| `src/backend/internal/server/rankings_test.go` (신규), `cards_test.go` | 순위 테스트, 테스트 서버가 DB 풀도 돌려주도록 변경 |

인덱스 추가는 없다 (1단계 판단대로 기존 `price_history_source_id_price_date_idx`를 쓴다).

## 구현 내용

1. `period`(`7d`·`30d`, 필수), `direction`(`up`·`down`, 필수), `limit`(1~50, 기본 10)을 검증한다.
2. 기준일 `asOf` = TCGCSV 최신 가격 기준일. 없으면 `asOf: null`, `items: []`.
3. 비교일 = 기준일 − 기간 이하 가장 늦은 TCGCSV 가격 기준일. 없으면 `asOf`는 채우고 `items: []`.
4. `ListMovers`로 순위를 낸다. 변동률 정렬은 SQL(`numeric`), 응답의 `changeRate`는 Go `api.ChangeRate`(소수 여섯째 자리, 추이 요약과 같은 함수)로 계산한다.
5. 원화는 기준일·비교일 각각 직전 환율 한 번씩 조회해 모든 항목에 적용한다(항목마다 조회하지 않음).
6. `card`는 `CardSummary`로, 대표가(Q1)는 순위 변형과 다를 수 있다(예: 순위는 초판, 대표가는 TCGCSV에서 우선순위가 가장 높은 변형).

## 검증

| 항목 | 결과 |
|------|------|
| `go vet ./...`, `go build ./...` | 통과 |
| `go test ./...` (DB 테스트 포함) | 전체 통과 |

**테스트 내용** (`rankings_test.go`, 기본 고정 데이터에 비교일 11-16 가격 9행 추가)

- 7d 상승: 0보다 큰 것만 큰 순, 같은 카드에서는 변동률이 큰 변형(리버스 +25% > normal +23%), 비교일 $1 미만 제외(+500%인 카드가 빠짐), 변동 없음 제외.
- 7d 하락, `limit=1`, 30d(비교일이 11-16이 아니라 10-20으로 내려감), 30d 상승 없음.
- 항목 필드: 기준일 가격 원화(직전 환율 11-21, 200497.5 → 200498), 비교일 환율 없음 → `krw: null`, `updatedAt` KST, `card` 대표가·세트·종 이름.
- 데이터 없음: 비교일 없음(`asOf`만), 시세 없음(`asOf: null`). 둘 다 `items: []`.
- 검증 오류 7건.
- 테스트 작성 중 고정 데이터의 `날짜 + 시각`이 DB 세션 시간대로 해석되는 문제가 있어 `AT TIME ZONE 'UTC'`로 못 박았다.

**1년치 규모 측정** (`poc_task8`: 가격 이력 3,021만 행·4.7 GB, 같은 코드의 API 서버를 포트 8090으로 따로 띄워 측정. 시간은 서버 로그의 처리 시간)

| 요청 | 횟수 | 첫 요청 | p50 | p95 |
|------|-----:|--------:|----:|----:|
| 순위 7d 상승 | 21 | 137 ms | 68 ms | 125 ms |
| 순위 7d 하락 | 20 | 107 ms | 68 ms | 107 ms |
| 순위 30d 상승 | 20 | 136 ms | 67 ms | 136 ms |
| 순위 30d 하락 `limit=50` | 20 | 247 ms | 79 ms | 247 ms |
| 추이 `1y` (카드 10장 × 2회) | 21 | 71 ms | 45 ms | 97 ms |
| 카드 상세 (10장) | 10 | 9 ms | 3 ms | 9 ms |

- 순위는 Q6 목표(300 ms) 안이다. p95가 높게 나온 것은 각 묶음의 첫 요청(캐시 없음)이다.
- 1년 추이는 p95 97 ms로 목표(100 ms)에 거의 닿는다. 카드마다 처음 읽을 때 디스크를 읽는 시간이다(`shared_buffers` 128 MB, 1단계 발견 사항). 응답 크기는 변형 2개 카드가 약 78 KB(점 약 1,460개)다. 캐시·압축은 #11에서 다룬다.
- 측정 후 `poc_task8`을 지웠다(1단계 S3 결정).

**개발 DB 실제 요청**: `/rankings/movers?period=7d&direction=up` → `{"asOf":"2026-11-23","items":[]}`. 이력이 11-22·11-23뿐이라 7d·30d 비교일이 없어 빈 목록이다(계약대로). 11-30 이후에 7d 순위가 나온다.

## 참고

- 상시 수집이 11-25를 건너뛰어 TCGCSV에 11-24 가격이 없다. 7d 순위는 기준일과 비교일이 모두 있어야 하므로, 수집이 매일 이어지면 11-30(기준일 11-29, 비교일 11-22) 무렵부터 나온다.
- 홈 화면(#12)은 지금 목업을 쓰므로 빈 순위 화면은 5단계 전환 후 확인한다.

## 다음 단계

5단계: 프론트엔드 실제 API 전환(`types.ts`·목업·`VITE_API_MOCK=false`), 응답 시간 실측(100회 p50·p95), 문서 정리(`api_contract.md`, `CLAUDE.md`, `db_schema.md` 7절).
