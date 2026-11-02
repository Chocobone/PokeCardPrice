# Task #3 단계4 완료 보고서: 선정 기술을 백엔드 골격에 적용

## 수행 내용

승인된 선정안(Echo v5, sqlc + pgx, goose, 별도 수집 프로세스, gocron v2)을 `src/backend/`에 적용했다.

- **API 서버**: 라우터를 Echo v5로 바꾸고, 공통 JSON 오류 처리기를 넣었다. `/healthz`가 DB 상태를 함께 확인한다. 시작할 때 DB에 연결하고 마이그레이션을 적용한다.
- **DB**: pgx 연결 풀, goose 마이그레이션(실행 파일에 포함, 세션 잠금), 빈 초기 마이그레이션 1개를 만들었다.
- **수집기**: `cmd/collector`를 만들었다. 매일 05:30 KST에 빈 작업을 실행하고, `-once`(즉시 1회)와 `COLLECTOR_INTERVAL`(개발용 간격)을 지원한다.
- **관리 명령**: `cmd/migrate`(status, up, down)를 만들었다.
- **도구**: sqlc를 `go.mod`의 `tool`로 고정했다(`go tool sqlc`).
- **compose**: `collector` 서비스를 추가했다. 엔트리포인트는 `APP` 환경 변수로 실행할 프로그램을 고른다.

## 변경 파일

| 파일 | 변경 내용 |
|------|----------|
| `src/backend/cmd/server/main.go` | DB 연결·마이그레이션 후 Echo 서버 실행, 오류 시 종료 코드 1 |
| `src/backend/cmd/collector/main.go` | 신규. 수집기 진입점 |
| `src/backend/cmd/migrate/main.go` | 신규. 마이그레이션 관리 명령 |
| `src/backend/internal/server/server.go` | Echo 라우터, 공통 오류 처리기, `/healthz` DB 확인 |
| `src/backend/internal/server/server_test.go` | 기존 3개 이식 + 3개 추가 (DB 장애 503, 404 JSON, 오류 응답 1회) |
| `src/backend/internal/db/db.go` | 신규. 연결 풀, goose 실행 |
| `src/backend/internal/scheduler/scheduler.go`, `scheduler_test.go` | 신규. gocron 래퍼와 테스트 2개 |
| `src/backend/migrations/migrations.go`, `00001_init.sql` | 신규. 마이그레이션 포함, 빈 초기 마이그레이션 |
| `src/backend/sqlc.yaml` | 신규. 스키마는 `migrations/`, 쿼리·생성 코드는 #4부터 |
| `src/backend/go.mod`, `go.sum` | Echo, pgx, goose, gocron 추가, sqlc `tool` 등록 |
| `Docker/docker-compose.yml` | `collector` 서비스 추가 |
| `Docker/backend/entrypoint.sh` | `APP` 환경 변수로 `cmd/` 선택 (기본 `server`) |
| `Docker/.env.example` | `COLLECTOR_INTERVAL` 항목 추가 |
| `CLAUDE.md`, `README.md` | 마이그레이션·수집기·sqlc 명령, 기술 스택, 저장소 구조 갱신 |
| `mydocs/tech/backend_stack.md` | 4단계 적용 시 변경 사항 추가 |

## 계획과 달라진 점

1. **goose CLI를 `go.mod` 도구로 넣지 않았다.** 넣어 보니 goose CLI가 지원하는 모든 DB 드라이버가 따라 들어와 `go.mod`가 112줄이 되었다. 대신 goose 라이브러리로 `cmd/migrate`를 만들어 상태 확인·되돌리기를 하게 했다. 최종 `go.mod`는 66줄이다.
2. **sqlc 생성 코드는 만들지 않았다.** 테이블이 없어 생성할 쿼리가 없다. 설정 파일만 두고 `go tool sqlc version`(v1.31.1)으로 실행을 확인했다. 첫 생성은 #4에서 한다.
3. **CORS 미들웨어는 넣지 않았다.** 프론트엔드가 Vite 프록시로 같은 출처에서 호출하므로 지금은 필요 없다.
4. **기존 테스트의 Content-Type 검사를 완화했다.** `application/json; charset=utf-8` 정확히 일치에서 `application/json`으로 시작하는지로 바꿨다. Echo는 `application/json`을 보낸다. 기존 테스트 3개의 나머지 검사(상태 코드, `status` 값)는 유지했다.
5. **서버는 DB에 연결하지 못하면 시작하지 않는다.** 시작 시 마이그레이션을 적용해야 하기 때문이다. compose의 `restart: unless-stopped`로 DB가 준비되면 다시 뜬다. 실행 중 DB가 끊기면 서버는 계속 떠 있고 `/healthz`가 503을 반환한다.

## 발견 사항

- **서버와 수집기가 동시에 시작해도 마이그레이션이 한 번만 적용되었다.** 한쪽은 `migration applied`, 다른 쪽은 `migration done`만 기록했다(goose 세션 잠금).
- **`goose_db_version`은 되돌리면 행을 지운다.** down 후 up 하면 이력이 새로 쌓인다.
- 프론트엔드 임시 화면은 `/healthz`의 `status` 값만 쓰므로, 응답에 `db` 필드가 추가되어도 영향이 없다.

## 검증

| 항목 | 결과 |
|------|------|
| 빌드·정적 검사 | ✅ `go build ./...`, `go vet ./...` 통과, `gofmt -l` 결과 없음 |
| 테스트 | ✅ 8 passed (server 6, scheduler 2), 0 failed |
| 기동 | ✅ `docker compose up -d --build` 후 db healthy, backend·collector·frontend Up |
| `/healthz` | ✅ 정상 200 `{"db":"ok","status":"ok"}` → `stop db` 후 503 `{"db":"error","status":"degraded"}` → `start db` 후 200 |
| 오류 응답 | ✅ 없는 경로 404 `{"error":"Not Found"}`, POST 405 `{"error":"Method Not Allowed"}` |
| 프록시 | ✅ `http://localhost:5173/api/healthz` 200 |
| 마이그레이션 | ✅ status(applied) → down(pending) → up(applied), `goose_db_version` 확인. 개발 DB 테이블은 `goose_db_version`뿐 |
| 스케줄러 | ✅ `COLLECTOR_INTERVAL=5s`로 5초마다 빈 작업 실행 로그, `stop collector` 시 `shutdown complete`, 종료 코드 0. 기본 설정에서 다음 실행 2026-11-03T05:30+09:00 |
| 수동 실행 | ✅ `go run ./cmd/collector -once`가 작업 1회 실행 후 종료 |
| 서버 정상 종료 | ✅ `stop backend` 시 `shutdown complete`, 종료 코드 0 |
| 재현성 | ✅ `docker compose down` → `up -d --build` 후 위 기동·`/healthz`·프록시·수집기 일정 재확인 (볼륨 유지 상태) |
| 범위 | ✅ 테이블 생성 SQL과 수집 로직 없음 (초기 마이그레이션은 `SELECT 1`, 수집 작업은 로그만 남김) |

## 다음 단계

승인 후 최종 결과보고서를 작성한다.

## 커밋

`Task #3: Stage 4 선정 기술을 백엔드 골격에 적용` (본 보고서와 함께 커밋)
