# CLAUDE.md

이 파일은 클로드 코드가 이 저장소에서 작업할 때 참고하는 지시서이다.

## 프로젝트 개요

<!-- TODO: 프로젝트 목표와 기술 스택을 여기에 기술한다 -->
**목표**: [프로젝트 목표]
- 포켓몬 카드의 실시간 가격 모니터링 웹 어플리케이션
- 각 카드의 가격 정보를 여러 플랫폼에서 가져와 비교할 수 있는 사이트
- 과거의 시세와 현재의 가격을 시계열 분석으로 한눈에 비교 가능하도록 구성

**기술 스택**: [사용 언어/프레임워크]
- frontend: Vite + Vue 
- backend: go
- DB : PostgreSQL

## 클로드 코드 사용 시 주의사항

이 프로젝트는 **하이퍼-워터폴(Hyper-Waterfall)** 방법론을 적용한다.
클로드 코드의 기본 동작(빠른 실행, 자율 수정)과 충돌이 발생할 수 있으므로 반드시 숙지한다.

**핵심 규칙:**
- 소스 수정 전 반드시 작업지시자 승인 요청
- 이슈 → 브랜치 → 할일 → 계획서 → 구현 순서 절대 생략 금지
- 각 단계 완료 후 승인 없이 다음 단계 진행 금지
- 이슈 클로즈는 작업지시자 승인 후에만 수행

상세 방법론: `mydocs/manual/hyper_waterfall.md`

---

## 문서 생성 규칙

모든 문서는 **한국어**로 작성한다.

### 폴더 구조 (`mydocs/` 하위)

| 폴더 | 용도 |
|------|------|
| `orders/` | 오늘 할일 (`yyyymmdd.md`만 허용) |
| `plans/` | 수행 계획서, 구현 계획서 |
| `plans/archives/` | 완료된 계획서 보관 |
| `working/` | 단계별 완료 보고서 (`_stage{N}.md`) |
| `report/` | 최종 결과보고서 (`_report.md`) + 기타 장기 보관 보고서 |
| `feedback/` | 작업지시자 피드백, 코드 리뷰 의견 |
| `tech/` | 기술 조사·분석 |
| `manual/` | 매뉴얼, 가이드 |
| `troubleshootings/` | 트러블슈팅 |

### 파일명 규칙

| 문서 | 형식 | 예시 |
|------|------|------|
| 오늘 할일 | `yyyymmdd.md` | `20260427.md` |
| 수행 계획서 | `task_{milestone}_{번호}.md` | `task_m100_1.md` |
| 구현 계획서 | `task_{milestone}_{번호}_impl.md` | `task_m100_1_impl.md` |
| 단계별 완료 보고서 | `task_{milestone}_{번호}_stage{N}.md` | `task_m100_1_stage1.md` |
| 최종 결과보고서 | `task_{milestone}_{번호}_report.md` | `task_m100_1_report.md` |

**규칙:**
- `task_` 접두어 필수
- 마일스톤은 `m{숫자}` 형식 (예: `m100`, `m200`)
- 후속 수정은 `_v2`, `_v3` 접미어 사용

### 폴더 역할 (엄격 준수)

- `plans/` — 계획서만. `_stage{N}`, `_report`는 여기 두지 않는다.
- `working/` — 단계별 보고서(`_stage{N}.md`)만. 최종 보고서는 여기 두지 않는다.
- `report/` — **최종 보고서(`_report.md`)는 반드시 이 폴더**.

---

## 빌드 및 실행

개발 환경은 컨테이너로 관리한다. 호스트에는 Docker만 있으면 되고, `go`·`npm`은 컨테이너 안에서 실행한다.
아래 명령은 모두 `Docker/` 폴더에서 실행한다.

```bash
# 최초 1회: 환경 변수 파일 생성 후 POSTGRES_PASSWORD 변경
cp .env.example .env

# 실행 (db → backend → frontend 순으로 기동)
docker compose up -d --build
docker compose ps                     # db healthy, backend·frontend Up 확인
# 백엔드 http://localhost:8080/healthz, 프론트엔드 http://localhost:5173

# 소스 변경 반영
docker compose restart backend        # Go 소스 변경 시 (자동 재시작 없음)
                                      # 프론트엔드는 Vite HMR로 자동 반영

# 빌드
docker compose exec backend go build ./...
docker compose exec frontend npm run build

# 테스트
docker compose exec backend go vet ./...
docker compose exec backend go test ./...

# 의존성 추가
docker compose exec backend go get <모듈>
docker compose exec frontend npm install <패키지>

# 중지
docker compose down                   # 데이터 유지
docker compose down -v                # 볼륨 삭제 (DB 초기화)
```

- Git Bash에서 컨테이너 안의 절대 경로(`/app` 등)를 인자로 넘길 때는 `MSYS_NO_PATHCONV=1`을 앞에 붙인다.
- 프론트엔드는 `/api/*`로 호출하고, Vite 프록시가 `/api`를 떼어 백엔드로 넘긴다(`/api/healthz` → `/healthz`).

---

## 워크플로우

### 브랜치 관리

| 브랜치 | 용도 |
|--------|------|
| `main` | 최종 릴리즈 |
| `devel` | 개발 통합 |
| `local/devel` | devel의 로컬 작업 브랜치 |
| `local/task{num}` | 타스크별 작업 |

### Git 워크플로우

```
local/task{N} ──커밋──커밋──┐
                             ├─→ local/devel merge
                             ├─→ devel merge + push
                             └─→ main PR (릴리즈 시점)
```

- 타스크 브랜치: `local/task{이슈번호}` (예: `local/task1`)
- `local/devel`, `local/task` 브랜치는 원격 push 금지 (로컬 유지)
- `devel`만 push

### 타스크 번호 관리

- GitHub Issues를 타스크 번호로 사용
- 마일스톤 표기: `M{버전}` (예: M100=v1.0.0)
- 커밋 메시지: `Task #N: 내용`

### 타스크 진행 절차

1. GitHub Issue에 타스크 등록
2. `local/task{이슈번호}` 브랜치 생성
3. 수행 계획서 작성 → **승인 요청**
4. 구현 계획서 작성 (최소 3단계, 최대 6단계) → **승인 요청**
5. 단계별 구현 진행
6. 각 단계 완료 후 단계별 완료보고서 작성 → **승인 요청**
7. **단계별 완료보고서는 해당 단계 소스 커밋과 함께 타스크 브랜치에서 커밋**
8. 모든 단계 완료 시 최종 결과보고서 작성 → **승인 요청**
9. **최종 결과보고서와 오늘할일 갱신도 타스크 브랜치에서 커밋. merge 전 `git status`로 미커밋 파일 없는지 확인**
10. 최종 결과보고서 작성 후 오늘할일 해당 타스크 상태 갱신

### 작업 규칙

- 작업 시작과 종료는 작업지시자가 결정한다. 클로드가 임의로 작업 종료를 제안하거나 시간을 한정하지 않는다.
