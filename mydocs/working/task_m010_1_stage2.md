# Task #1 Stage 2: 백엔드 골격 — 단계별 완료 보고서

> **작성일**: 2026-10-30
> **브랜치**: `local/task1`

## 작업 내용

백엔드 컨테이너 안에서 Go 모듈을 초기화하고, 표준 라이브러리 `net/http`만으로 `/healthz` 엔드포인트를 갖춘 API 서버 골격을 만들었다.
웹 프레임워크(Gin/Echo)와 DB 드라이버는 #3 백엔드 기술 선정에서 결정하므로 이번 단계에서는 도입하지 않았다.

### 추가·변경 파일

| 파일 | 내용 |
|------|------|
| `src/backend/go.mod` | 모듈 `github.com/Chocobone/PokeCardPrice/src/backend`, Go 1.27.3 |
| `src/backend/cmd/server/main.go` | 서버 진입점. `PORT` 환경 변수(기본 8080), `ReadHeaderTimeout` 5초, SIGINT/SIGTERM 수신 시 10초 안에 정상 종료 |
| `src/backend/internal/server/server.go` | `New()` 라우터. `GET /healthz` → `{"status":"ok"}` JSON 응답 |
| `src/backend/internal/server/server_test.go` | `/healthz` 200·JSON 본문, POST 405, 없는 경로 404 테스트 |
| `src/backend/.gitkeep` | 삭제 (실제 소스가 생겨 불필요) |
| `Docker/backend/entrypoint.sh` | `go run` 대신 `go build` 후 `exec`로 바이너리 실행 (아래 참고) |

### 설계 메모

- 라우팅은 Go 1.22부터 지원하는 `ServeMux` 메서드 패턴(`"GET /healthz"`)을 사용했다. 메서드가 다르면 405, 경로가 없으면 404가 자동으로 응답된다.
- 핸들러는 `internal/server`에 두어 `httptest`로 서버를 띄우지 않고 테스트할 수 있게 했다.
- `/healthz`에 DB 연결 확인은 넣지 않았다(구현 계획서대로 #3에서 드라이버 선정 후 추가).

### 계획 외 변경: 엔트리포인트 수정

Stage 1 엔트리포인트는 `exec go run ./cmd/server`였다. 검증 중 컨테이너를 재시작해도 서버의 정상 종료 로그(`shutting down`)가 찍히지 않는 것을 발견했다.
`go run`은 SIGTERM을 받으면 자기 프로세스만 종료하고, 자식인 서버 프로세스에는 신호를 전달하지 않기 때문이다.
그래서 `go build -o /tmp/server ./cmd/server` 후 `exec /tmp/server`로 서버가 PID 1이 되도록 바꿨다. 엔트리포인트는 이미지에 복사되므로 이미지를 다시 빌드했다(`docker compose up -d --build backend`).

소스를 고친 뒤 반영하려면 `docker compose restart backend`를 실행한다(자동 재시작 도구는 도입하지 않았다).

## 검증 결과

```
$ docker compose exec backend sh -c 'cd /app && gofmt -l . ; go vet ./... && go test -v ./...'
=== RUN   TestHealthz
--- PASS: TestHealthz (0.00s)
=== RUN   TestHealthzMethodNotAllowed
--- PASS: TestHealthzMethodNotAllowed (0.00s)
=== RUN   TestUnknownPath
--- PASS: TestUnknownPath (0.00s)
PASS
ok  	github.com/Chocobone/PokeCardPrice/src/backend/internal/server	0.009s

$ curl -si localhost:8080/healthz
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8

{"status":"ok"}

$ docker compose restart backend && docker compose logs --tail 3 backend
backend-1  | 14:59:18 listening on :8080
backend-1  | 14:59:25 shutting down
backend-1  | 14:59:26 listening on :8080
```

- `gofmt -l` 출력 없음(포맷 위반 없음), `go vet` 경고 없음.
- 엔트리포인트 수정 후 재시작할 때 정상 종료 로그가 찍히는 것을 확인했다.

## 다음 단계

Stage 3 — 프론트엔드 골격: 컨테이너 안에서 Vite + Vue 프로젝트 생성, `/api` → 백엔드 프록시, 파일 감시 polling 설정, 화면에 `/healthz` 결과 표시
