# Task #1 최종 결과보고서: 저장소 구조 및 개발 환경 구성

> **작성일**: 2026-10-30
> **브랜치**: `local/task1`
> **마일스톤**: M010 (v0.1.0)

## 개요

호스트에 Docker만 있으면 `docker compose up` 한 번으로 PostgreSQL, Go 백엔드, Vite + Vue 프론트엔드가 뜨는 개발 환경을 구성했다.
백엔드는 `/healthz`만 응답하는 최소 서버이고, 프론트엔드는 Vite 프록시를 거쳐 백엔드 상태를 표시하는 임시 화면이다.
빌드·테스트·실행 명령은 README와 CLAUDE.md에 정리했다. 이후 타스크(#4 DB 스키마, #8 조회 API, #12 프론트엔드)는 이 골격 위에서 바로 진행할 수 있다.

## 수행 결과

| 단계 | 내용 | 커밋 |
|------|------|------|
| 1단계 | 컨테이너 개발 환경 구성 (`Docker/` compose·Dockerfile·엔트리포인트, `.gitignore`, `.gitattributes`) | `4216ac5` |
| 2단계 | 백엔드 골격 (Go 모듈, `/healthz`, 단위 테스트 3개, 엔트리포인트 정상 종료 수정) | `88a2d6d` |
| 3단계 | 프론트엔드 골격 (Vite + Vue, `/api` 프록시, polling 파일 감시, 상태 표시 화면) | `c4b408c` |
| 4단계 | 문서 정비 (README 교체, CLAUDE.md 빌드·실행 명령) | `953cefd` |

## 산출물

### 저장소 구조

```
PokeCardPrice/
├── Docker/
│   ├── docker-compose.yml      ← db, backend, frontend
│   ├── .env.example            ← .env로 복사해 사용 (.env는 커밋 제외)
│   ├── backend/                ← golang:1.27-alpine3.24 개발 이미지, 엔트리포인트
│   └── frontend/               ← node:26-alpine3.24 개발 이미지, 엔트리포인트
├── src/
│   ├── backend/                ← go.mod, cmd/server/main.go, internal/server/
│   └── frontend/               ← package.json, vite.config.js, src/App.vue
├── .gitignore
└── .gitattributes              ← *.sh, Dockerfile LF 고정
```

### 서비스

| 서비스 | 이미지 | 주소 | 비고 |
|--------|--------|------|------|
| `db` | `postgres:18-alpine3.24` | `localhost:5432` | `pg_isready` 헬스체크, `db-data` 볼륨 |
| `backend` | Go 1.27.3 | `http://localhost:8080/healthz` | db가 healthy가 된 뒤 기동, `DATABASE_URL` 주입 |
| `frontend` | Node 26.13.0, Vite 8.3.5, Vue 3.5.44 | `http://localhost:5173` | `/api/*` → 백엔드 프록시, HMR |

### 사용 방법

```bash
cd Docker
cp .env.example .env            # 최초 1회, POSTGRES_PASSWORD 변경
docker compose up -d --build
docker compose exec backend go test ./...
docker compose restart backend  # Go 소스 변경 반영
```

전체 명령은 CLAUDE.md `빌드 및 실행` 절과 README 참조.

## 발견 사항

- **PostgreSQL 18 데이터 경로 변경**: 18 이미지는 데이터를 `/var/lib/postgresql/18/docker`에 둔다. 볼륨은 상위 경로 `/var/lib/postgresql`에 마운트해야 한다.
- **`go run`은 종료 신호를 전달하지 않는다**: `exec go run`으로 띄우면 SIGTERM을 받은 `go run`만 종료되고 서버는 정상 종료 절차 없이 끝난다. `go build` 후 `exec`로 바이너리를 PID 1로 실행하도록 바꿨다.
- **Windows 바인드 마운트 파일 감시**: 호스트의 파일 변경 이벤트가 컨테이너로 전달되지 않아 Vite `watch.usePolling`이 필요하다.
- **Git Bash 경로 변환**: `docker compose exec`에 `/app` 같은 절대 경로를 넘기면 Windows 경로로 바뀐다. `MSYS_NO_PATHCONV=1`로 막는다.
- **네임드 볼륨이 마운트된 디렉터리에서 프로젝트 생성**: `node_modules` 볼륨 때문에 `/app`이 비어 있지 않아 `create-vite`가 덮어쓰기 여부를 묻는다. 임시 경로에서 만든 뒤 복사했다.
- **줄바꿈 경고**: 커밋할 때 `LF will be replaced by CRLF` 경고가 난다(`core.autocrlf` 설정 때문). `.gitattributes`로 `*.sh`와 `Dockerfile`은 LF로 고정했고, 엔트리포인트는 이미지 빌드 때 CR을 한 번 더 제거하므로 동작에는 영향이 없다.

## 검증

| 항목 | 결과 |
|------|------|
| 컨테이너 기동 | ✅ 3개 서비스 Up, db healthy |
| 백엔드 빌드·정적 검사 | ✅ `go build`, `go vet`, `gofmt` 통과 |
| 단위 테스트 | 3 passed (`/healthz` 200·JSON, POST 405, 없는 경로 404) |
| 프론트엔드 빌드 | ✅ `npm run build`, 취약점 0건 |
| 기능 검증 | ✅ `/healthz`, 프록시 경유 `/api/healthz` 응답, HMR 반영, 백엔드 정상 종료 |
| 화면 | ✅ 작업지시자가 브라우저에서 확인 후 Stage 3 승인 |
| 재현성 | ✅ 문서 명령만으로 down → up --build 후 전 항목 재확인 (볼륨 유지 상태) |

## 계획 대비 차이

- 수행 계획서 개정(2026-10-29): 작업지시자 지시로 개발 환경을 컨테이너(`Docker/`)로 관리하고, 모든 개발 코드를 `src/` 아래에 두었다. 구현 계획서는 개정된 계획을 기준으로 작성했다.
- Stage 2에서 Stage 1 산출물인 `Docker/backend/entrypoint.sh`를 수정했다(`go run` → `go build` + `exec`). 이유는 발견 사항 참조.
- 프론트엔드 화면 렌더링은 Claude가 브라우저 도구 없이 진행해 직접 확인하지 못했다. 작업지시자가 확인한 뒤 승인했다.
- 볼륨까지 지운 완전히 새로운 환경(`down -v`)에서의 기동은 검증하지 않았다.

## 후속 타스크로 넘기는 사항

- #3: Go 웹 프레임워크, DB 드라이버, 마이그레이션 도구 선정 후 `/healthz`에 DB 상태 확인 추가
- #12: 라우터, Pinia, ESLint 도입 및 임시 화면 교체
- #16: 운영용 이미지(멀티 스테이지 빌드)와 배포 시 `/api` 리버스 프록시 규칙 확정

## 이슈

- https://github.com/Chocobone/PokeCardPrice/issues/1
