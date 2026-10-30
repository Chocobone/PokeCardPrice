# Task #1 Stage 4: 문서 정비 — 단계별 완료 보고서

> **작성일**: 2026-10-30
> **브랜치**: `local/task1`

## 작업 내용

하이퍼-워터폴 템플릿 설명이던 README를 Card-Market 프로젝트 소개로 바꾸고,
CLAUDE.md `빌드 및 실행` 절의 자리표시자를 compose 기반 명령으로 채웠다.

### 변경 파일

| 파일 | 내용 |
|------|------|
| `README.md` | 프로젝트 소개, 주요 기능(계획), 기술 스택, 저장소 구조, 실행 방법·서비스 주소, 개발 명령, 개발 방식·마일스톤 |
| `CLAUDE.md` | `빌드 및 실행` 절: 최초 설정, 실행·상태 확인, 소스 변경 반영, 빌드, 테스트, 의존성 추가, 중지 명령. Git Bash 경로 변환(`MSYS_NO_PATHCONV=1`)과 `/api` 프록시 규칙 메모 |

### 메모

- 템플릿 README에 있던 방법론 적용 안내는 `mydocs/manual/hyper_waterfall.md`에 같은 내용이 있어 README에서는 링크만 남겼다.
- README의 기능 설명은 계획 단계임을 표시했다. 기능이 구현되면 해당 타스크에서 갱신한다.

## 검증 결과

문서의 명령을 그대로 실행해 Stage 1~3 결과를 다시 확인했다. 컨테이너는 내렸다가 이미지를 다시 빌드해 기동했다(볼륨은 유지).

```
$ docker compose down && docker compose up -d --build
$ docker compose ps
backend   Up 14 seconds
db        Up 14 seconds (healthy)
frontend  Up 13 seconds

$ docker compose exec backend go build ./...          # 성공
$ docker compose exec frontend npm run build          # ✓ built in 696ms
$ docker compose exec backend go vet ./...            # 경고 없음
$ docker compose exec backend go test ./...
ok  	github.com/Chocobone/PokeCardPrice/src/backend/internal/server	0.009s

$ curl -s localhost:8080/healthz                      # {"status":"ok"}
$ curl -s localhost:5173/api/healthz                  # {"status":"ok"}  (프록시 경유)
$ curl -s localhost:5173/ | grep -o '<title>.*</title>'
<title>Card-Market</title>

$ docker compose restart backend && docker compose logs --tail 2 backend
shutting down
listening on :8080
```

## 다음 단계

최종 결과보고서 작성 (`mydocs/report/task_m010_1_report.md`)
