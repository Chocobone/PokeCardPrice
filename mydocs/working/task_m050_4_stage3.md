# Task #4 단계3 완료 보고서: 마이그레이션 작성과 검증

## 수행 내용

- 2단계에서 승인된 DDL(`db_schema.md` 6절, Q5 `pokemon_species` 포함)을 goose 마이그레이션 3개로 옮겼다. 13개 테이블과 가격 소스 초기 5행이다.
- 테스트 전용 DB를 쓰는 헬퍼와, 마이그레이션 왕복·제약 테스트를 작성했다.
- `backend` 서비스에 `TEST_DATABASE_URL`을 추가했다.

## 변경 파일

| 파일 | 변경 내용 |
|------|----------|
| `src/backend/migrations/00002_catalog.sql` | 신규. `pg_trgm` 확장, `sets`, `cards`, `card_variants`, `card_links`, `pokemon_species`와 검색 인덱스 |
| `src/backend/migrations/00003_prices.sql` | 신규. `price_sources`(초기 5행), `external_ids`, `price_history`(넓은 형태 + 보조 인덱스), `fx_rates` |
| `src/backend/migrations/00004_users.sql` | 신규. `users`(`uuidv7()`, 이메일 대소문자 무시 유일), `collection_items`, `user_trades`, `admin_price_entries` |
| `src/backend/internal/db/testdb_test.go` | 신규. `testPool`: `TEST_DATABASE_URL`이 없으면 건너뜀, DB가 없으면 생성, `public` 스키마 초기화, 개발 DB와 같으면 실패 |
| `src/backend/internal/db/migrate_test.go` | 신규. `TestMigrationsUpDownUp`, `TestSchemaConstraints` (하위 테스트 17개) |
| `Docker/docker-compose.yml` | `backend`에 `TEST_DATABASE_URL` (`${POSTGRES_DB}_test`) 추가 |
| `mydocs/working/task_m050_4_stage2.md` | 커밋 해시 기록 |
| `mydocs/orders/20261106.md` | Stage 3 진행 상태 |

## 기술적 결정

- **테스트 DB 생성 방법: 테스트 헬퍼가 `CREATE DATABASE` 실행.** compose 초기화 스크립트(`/docker-entrypoint-initdb.d`)는 DB 볼륨이 처음 만들어질 때만 실행되어, 이미 볼륨이 있는 개발 환경에는 적용되지 않는다. 헬퍼 방식은 볼륨 상태와 상관없이 동작한다.
- **테스트 DB 이름: `pokecard_test` 대신 `${POSTGRES_DB}_test`(기본 `cardmarket_test`).** 구현 계획서와 다르다. `.env`의 DB 이름을 바꿔도 테스트 DB 이름이 따라 바뀌어 설정이 한곳에 모인다.
- **개발 DB 보호**: 테스트는 `public` 스키마를 통째로 지우므로, `TEST_DATABASE_URL`의 호스트·포트·DB 이름이 `DATABASE_URL`과 같으면 테스트를 실패시킨다.
- **`Down`에서 `pg_trgm` 확장도 삭제한다.** 버전 0까지 내렸을 때 마이그레이션 이전 상태와 같아지게 하기 위해서다. 테스트로 확인했다.

## 발견 사항

- `ON DELETE RESTRICT` 위반은 SQLSTATE `23503`(foreign_key_violation)이 아니라 **`23001`(restrict_violation)**이다. 처음에는 테스트 기대값을 `23503`으로 써서 실패했고, 기대값을 고쳤다. #10 API가 "사용자 데이터가 있어 삭제할 수 없음" 오류를 구분할 때 이 코드를 써야 한다.
- 백엔드와 수집기가 동시에 시작해도, 백엔드가 적용하는 동안 수집기는 goose 세션 잠금을 기다린 뒤 `version=4`를 확인했다(#3에서 넣은 잠금이 동작함).
- 저장소의 Go 파일은 체크아웃 시 CRLF로 바뀌어(`core.autocrlf=true`) 컨테이너의 `gofmt -l`이 모든 파일을 나열한다. CR을 지우고 검사하면 차이가 없다. 이번 변경과 무관한 기존 상태이다.

## 검증

- `go build ./...`, `go vet ./...`: ✅
- `go test ./...`: ✅ 기존 테스트(server 3, scheduler) + `internal/db` 2개 테스트(하위 17개) 통과
  - 왕복: 빈 DB → up(버전 4, 테이블 13개) → `DownTo(0)`(테이블 0개, `pg_trgm` 없음) → up
  - 제약: 가격 중복, 음수 가격, 잘못된 대표값 필드, 세트·카드 언어 불일치, 영어 카드의 한글판 표시, 세트 안 번호 중복, 알 수 없는 변형, 자기 연결, 외부 ID 중복, 같은 통화쌍 환율, 이메일 대소문자 중복, 수량 0, 알 수 없는 상태 등급, 사용자 데이터가 있는 변형 삭제 → 모두 거부
  - 가격 소스 5행, 사용자 ID UUID 버전 7, 카드 삭제 시 변형·가격 이력 함께 삭제
- `TEST_DATABASE_URL`을 비우면 DB 테스트 2개가 `SKIP`: ✅
- 개발 DB: `docker compose up -d backend` 후 로그에 `migration applied` 3건(00002~00004), `/healthz` → `{"db":"ok","status":"ok"}`: ✅
- 개발 DB: `migrate down` 3회 → 앱 테이블 0개 → `migrate up` → 버전 4: ✅
- 재현성: 데이터가 없는 것(행 0)을 확인한 뒤 DB 볼륨(`pokecardprice_db-data`)만 지우고 `docker compose up -d --build` → 마이그레이션 1~4 적용, `/healthz` 200, 다음 `go test` 실행 때 `cardmarket_test` 자동 생성: ✅
  - 구현 계획서의 `docker compose down -v` 대신 DB 볼륨만 지웠다. `-v`는 Go 모듈·빌드 캐시와 `node_modules` 볼륨까지 지운다.

## 다음 단계

4단계: 금액 Go 타입을 생성 코드로 비교해 확정하고 `sqlc.yaml` `overrides`에 반영한다. 기본 쿼리(`catalog.sql`, `prices.sql`, `users.sql`)와 생성 코드, 쿼리 테스트를 작성한다. `db_schema.md`, `backend_stack.md`, `CLAUDE.md`를 갱신한다.

## 커밋

승인 후 기록
