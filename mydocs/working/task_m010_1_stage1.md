# Task #1 Stage 1: 컨테이너 개발 환경 구성 — 단계별 완료 보고서

> **작성일**: 2026-10-29
> **브랜치**: `local/task1`

## 작업 내용

작업지시자 지시(개발 환경을 컨테이너로 관리, `Docker/` 폴더에 docker-compose 작성)에 따라
호스트에 Go·Node.js를 설치하지 않고 DB·백엔드·프론트엔드 개발 컨테이너를 구성했다.

### 추가 파일

| 파일 | 내용 |
|------|------|
| `Docker/docker-compose.yml` | `db`, `backend`, `frontend` 서비스. 네임드 볼륨 `db-data`, `go-mod`, `go-build`, `node-modules` |
| `Docker/.env.example` | DB 계정, 노출 포트 예시. 실제 값은 `Docker/.env`(커밋 제외) |
| `Docker/backend/Dockerfile` | `golang:1.27-alpine3.24` + git, tzdata |
| `Docker/backend/entrypoint.sh` | `go.mod`와 `cmd/server`가 있으면 `go run ./cmd/server`, 없으면 대기 |
| `Docker/frontend/Dockerfile` | `node:26-alpine3.24`(현재 LTS) + tzdata |
| `Docker/frontend/entrypoint.sh` | `package.json`이 있으면 `npm install` 후 dev 서버, 없으면 대기 |
| `src/backend/.gitkeep`, `src/frontend/.gitkeep` | 바인드 마운트 대상 디렉터리 |
| `.gitignore` | `.env`, Go 빌드 산출물, `node_modules`, `dist` 등 |
| `.gitattributes` | `*.sh`, `Dockerfile` LF 고정 (Windows CRLF로 인한 컨테이너 실행 오류 방지) |

### 설계 메모

- PostgreSQL 18 이미지는 데이터 경로가 `/var/lib/postgresql/18/docker`로 바뀌었다. 그래서 볼륨을 상위 경로 `/var/lib/postgresql`에 마운트했다.
- 소스가 아직 없는 단계에서도 컨테이너가 재시작을 반복하지 않도록, 엔트리포인트는 대기(`tail -f /dev/null`) 상태로 둔다.
  Stage 2·3에서 `docker compose exec backend|frontend ...`로 초기화 명령을 실행한다.
- 작업지시자 지시로 모든 개발 코드는 `src/` 아래에서 관리한다. compose는 `../src/backend`, `../src/frontend`를 각 컨테이너의 `/app`에 마운트한다.
  `Docker/`는 컨테이너 정의이므로 루트에 둔다.
- Git Bash에서 `docker compose exec ... /app`처럼 절대 경로를 넘기면 Windows 경로로 변환된다. 이때는 `MSYS_NO_PATHCONV=1`을 지정한다.
- 백엔드에는 `DATABASE_URL`을 compose에서 조립해 주입한다(`db` 서비스 이름으로 접속).
- 프론트엔드에는 `BACKEND_URL=http://backend:8080`을 주입한다. Stage 3에서 Vite 프록시 대상으로 사용한다.
- 엔트리포인트는 Dockerfile에서 `sed`로 CR을 한 번 더 제거해, `.gitattributes`가 적용되기 전 체크아웃에서도 동작하게 했다.

## 검증 결과

```
$ docker compose ps
backend    Up            0.0.0.0:8080->8080/tcp
db         Up (healthy)  0.0.0.0:5432->5432/tcp
frontend   Up            0.0.0.0:5173->5173/tcp

$ docker compose exec backend go version
go version go1.27.3 linux/amd64

$ docker compose exec frontend sh -c 'node -v; npm -v'
v26.13.0
11.20.0

$ docker compose exec db psql ... -c "select version()"
PostgreSQL 18.6 on x86_64-pc-linux-musl ...

$ docker compose exec backend nc -z db 5432
backend→db OK
```

- 백엔드·프론트엔드 로그에 대기 안내 메시지가 출력되는 것을 확인했다(소스 없음, 정상).
- `git check-ignore`로 `Docker/.env`가 커밋 대상에서 빠지는 것을 확인했다.

## 사용 방법

```bash
cd Docker
cp .env.example .env          # 최초 1회, POSTGRES_PASSWORD 변경
docker compose up -d --build  # 기동
docker compose ps             # 상태 확인
docker compose exec backend sh   # Go 컨테이너 접속
docker compose exec frontend sh  # Node 컨테이너 접속
docker compose down           # 중지 (데이터 유지)
docker compose down -v        # 중지 + 볼륨 삭제 (DB 초기화)
```

## 다음 단계

Stage 2 — 백엔드 골격: 컨테이너 안에서 `go mod init`, `cmd/server/main.go`(`/healthz`), 단위 테스트 작성
