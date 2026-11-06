# Task #4 단계4 완료 보고서: sqlc 타입 설정·기본 쿼리·문서 정리

## 수행 내용

- 금액 Go 타입 후보 3개(`pgtype.Numeric`, `string`, `shopspring/decimal`)로 sqlc 코드를 생성했다. 임시 DB(`poc_task4m`, 확인 후 삭제)에서 저장·조회·NULL·반올림·JSON 출력을 비교해 **`shopspring/decimal`**로 정했다.
- `sqlc.yaml`에 타입 `overrides`를 추가했다. 생성 코드에서 `pgtype`을 모두 없애고 NULL 허용 컬럼은 포인터로 통일했다.
- 기본 쿼리 17개(`catalog.sql` 5, `prices.sql` 7, `users.sql` 5)를 작성하고 생성 코드를 커밋 대상에 넣었다.
- 쿼리 테스트 4개를 작성했다.
- `db_schema.md` 10절(타입 결정, sqlc 설정, 쿼리 목록, 후속 타스크 인계), `backend_stack.md`, `CLAUDE.md`를 갱신했다.

## 변경 파일

| 파일 | 변경 내용 |
|------|----------|
| `src/backend/sqlc.yaml` | `emit_pointers_for_null_types`, `numeric`·`uuid`·`timestamptz`·`date` overrides |
| `src/backend/internal/db/queries/catalog.sql` | 신규. 세트·카드·변형 upsert, 카드 조회, 변형 목록 |
| `src/backend/internal/db/queries/prices.sql` | 신규. 가격 소스 조회, 외부 ID upsert·조회, 가격 이력 배치 upsert, 기간 조회, 환율 upsert·직전 값 조회 |
| `src/backend/internal/db/queries/users.sql` | 신규. 사용자 생성·이메일 조회, 컬렉션 upsert·목록, 사용자 거래가 등록 |
| `src/backend/internal/db/dbq/*.go` | 신규. sqlc 생성 코드 (6개 파일) |
| `src/backend/internal/db/queries_test.go` | 신규. 쿼리 테스트 4개 |
| `src/backend/go.mod`, `go.sum` | `shopspring/decimal` v1.6.0, `google/uuid` v1.6.0 직접 의존성 |
| `mydocs/tech/db_schema.md` | 10절 추가, 상태·결론·적용 방향 갱신 |
| `mydocs/tech/backend_stack.md` | "#4에서 정한다"로 남긴 금액 타입 두 곳에 결정 결과 기록 |
| `CLAUDE.md` | DB 테스트 실행·건너뛰기 명령, 스키마 문서 위치, 생성 코드 커밋 규칙, 테스트 DB 주의 |
| `mydocs/working/task_m050_4_stage3.md` | 커밋 해시 기록 |
| `mydocs/orders/20261106.md` | Stage 4 진행 상태 |

## 기술적 결정

| 항목 | 결정 | 근거 |
|------|------|------|
| 금액 Go 타입 | `shopspring/decimal` v1.6.0 | Go에서 정확한 산술이 가능하다(나머지 두 후보는 `float64` 변환 필요). 타입이 금액을 드러낸다. pgx 어댑터 없이 동작하고 의존성은 1개이다 |
| NULL 표현 | 모두 포인터 (`*string`, `*decimal.Decimal`, `*time.Time` 등) | `pgtype.Text`, `decimal.NullDecimal` 같은 타입이 섞이지 않게 한다 |
| 시각·날짜 | `time.Time` | `pgtype.Timestamptz`보다 다루기 쉽다 |
| JSON 태그 | 생성하지 않음 | `dbq.User`를 그대로 직렬화하면 `password_hash`가 나간다. API 응답은 #8~#10에서 별도 구조체로 만든다 |
| 가격 이력 일괄 저장 | `:batchexec` (pgx 배치) | upsert가 필요해 `COPY`(`:copyfrom`)는 쓸 수 없다 |

## 발견 사항

- **decimal 버전을 v1.4.0으로 고정하면 goose가 내려간다.** `go get ...decimal@v1.4.0`이 goose를 v3.29.0에서 v3.28.0으로 내렸다. goose v3.29.0이 decimal v1.5.0 이상을 요구하기 때문이다. go.mod를 되돌리고 최신 v1.6.0(2026-10-18 릴리스)으로 받았다. goose는 v3.29.0 그대로이다.
- **sqlc override의 `db_type` 이름**: `pg_catalog.timestamptz`로 쓰면 적용되지 않고 `pgtype.Timestamptz`가 그대로 생성되었다. `timestamptz`로 써야 한다(`numeric`은 `pg_catalog.numeric`으로 동작).
- **decimal은 뒤 0을 유지하지 않는다**: DB의 `0.20`은 `decimal`에서 `0.2`로, `1500.00`은 `1500`으로 읽힌다. 값은 같지만, 화면에 소수 둘째 자리까지 보이려면 API에서 `StringFixed(2)` 등으로 형식을 맞춰야 한다(#8).

## 검증

- `go tool sqlc generate`를 다시 실행해도 생성 파일 6개의 해시가 같다: ✅
- 생성 코드에 `pgtype` 사용 없음: ✅
- `go build ./...`, `go vet ./...`: ✅
- `go test ./...`: ✅ (`internal/db` 6개 테스트, 기존 server·scheduler 테스트)
  - `TestCatalogUpsertIsIdempotent`: 다시 저장하면 같은 ID, 같은 키로 이름·희귀도 갱신, NULL은 `nil`, 변형 2개 순서
  - `TestPriceHistoryUpsertAndRange`: 외부 ID로 변형 찾기, 배치 저장 3행 후 같은 날짜 재저장 → 행 수 그대로·값 갱신(0.25 → 0.27), 기간 필터와 날짜순
  - `TestFxRateOnOrBefore`: 일요일(11/01) 조회 시 10/30 값, 소수 8자리 보존, `0.27 USD × 1418.41702250 = 383 KRW`
  - `TestUserCollectionAndTrade`: UUID 버전 7, 이메일 대소문자 무시 조회, 컬렉션 수량 갱신 시 같은 ID, 거래가 기본 검증 상태 `pending`
- gofmt: CR을 지우고 검사하면 차이 없음: ✅

## 다음 단계

최종 결과보고서(`mydocs/report/task_m050_4_report.md`)를 작성한다.

## 커밋

`bc226c9` Task #4: Stage 4 sqlc 타입 설정·기본 쿼리·문서 정리
