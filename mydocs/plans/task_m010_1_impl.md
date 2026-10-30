# Task #1: 저장소 구조 및 개발 환경 구성 — 구현 계획서

## 전체 구조

```
PokeCardPrice/
├── Docker/
│   ├── docker-compose.yml      ← db, backend, frontend 서비스 정의
│   ├── .env.example            ← 환경 변수 예시 (복사해 .env로 사용, .env는 커밋 금지)
│   ├── backend/
│   │   ├── Dockerfile          ← golang:1.27-alpine3.24 기반 개발 이미지
│   │   └── entrypoint.sh       ← go.mod 있으면 서버 실행, 없으면 대기
│   └── frontend/
│       ├── Dockerfile          ← node:26-alpine3.24 기반 개발 이미지
│       └── entrypoint.sh       ← package.json 있으면 npm install + dev 서버, 없으면 대기
├── src/                        ← 모든 개발 코드 (작업지시자 지시)
│   ├── backend/                ← Go 소스 (바인드 마운트 → 컨테이너 /app)
│   └── frontend/               ← Vue 소스 (바인드 마운트 → 컨테이너 /app)
├── .gitignore
└── .gitattributes              ← *.sh LF 고정 (Windows CRLF로 인한 실행 오류 방지)
```

### 서비스 구성

| 서비스 | 이미지 | 포트(호스트:컨테이너) | 볼륨 | 비고 |
|--------|--------|------------------------|------|------|
| `db` | `postgres:18-alpine3.24` | `5432:5432` | `db-data` (데이터 유지) | `pg_isready` 헬스체크 |
| `backend` | `Docker/backend/Dockerfile` | `8080:8080` | `../src/backend:/app`, `go-mod`, `go-build` 캐시 | db 헬스체크 통과 후 기동 |
| `frontend` | `Docker/frontend/Dockerfile` | `5173:5173` | `../src/frontend:/app`, `node_modules` 네임드 볼륨 | 백엔드 주소를 환경 변수로 전달 |

- 포트는 `.env`에서 바꿀 수 있게 한다.
- 소스가 아직 없는 단계에서도 컨테이너가 죽지 않도록, 엔트리포인트는 소스가 없으면 대기 상태를 유지한다.
  이 상태에서 `docker compose exec backend go mod init ...`처럼 컨테이너 안에서 초기화 명령을 실행한다.
- `node_modules`는 네임드 볼륨에 둔다. Windows 바인드 마운트의 I/O 성능 문제와 호스트·컨테이너 간 바이너리 충돌을 피하기 위해서다.

## 단계별 계획

### Stage 1 — 컨테이너 개발 환경 구성

- `Docker/` 아래 compose, Dockerfile, 엔트리포인트, `.env.example` 작성
- `src/backend/`, `src/frontend/` 빈 디렉터리(`.gitkeep`) 생성
- 루트 `.gitignore`, `.gitattributes` 작성
- 이미지 pull·build 후 `docker compose up -d` → db `healthy`, backend·frontend 대기 상태 확인
- 컨테이너 안 `go version`, `node -v`, `psql` 접속 확인

### Stage 2 — 백엔드 골격

- 컨테이너 안에서 `go mod init`, `cmd/server/main.go`(`net/http`, `/healthz`)
- `DATABASE_URL`로 DB 연결 확인은 `/healthz` 응답에 포함하지 않는다(#3에서 드라이버 선정 후 추가)
- `go test ./...` 통과, 호스트에서 `curl localhost:8080/healthz` 확인

### Stage 3 — 프론트엔드 골격

- 컨테이너 안에서 Vite + Vue 프로젝트 생성
- `vite.config`에 `/api` → 백엔드 프록시, Windows 바인드 마운트용 파일 감시 polling 설정
- 브라우저에서 `localhost:5173` 접속, 백엔드 `/healthz` 호출 결과 표시 확인

### Stage 4 — 문서 정비

- README를 프로젝트 소개, 구조, 실행 방법으로 교체
- CLAUDE.md `빌드 및 실행` 절에 compose 기반 명령 기입
- 최종 보고서 작성

## 검증 방법

| 단계 | 확인 |
|------|------|
| 1 | `docker compose ps`에서 3개 서비스 실행, db `healthy`, 컨테이너 내 도구 버전 출력 |
| 2 | `go test ./...` 통과, `/healthz` 200 응답 |
| 3 | `npm run build` 성공, 브라우저에서 백엔드 응답 표시 |
| 4 | 문서의 명령을 그대로 실행해 1~3 재현 |
