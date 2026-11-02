# Task #3 최종 결과보고서: 백엔드 기술 선정

> **작성일**: 2026-11-02
> **브랜치**: `local/task3`
> **마일스톤**: M010 (v0.1.0)

## 개요

웹 프레임워크, DB 접근 방식, 마이그레이션 도구, 스케줄러 방식을 후보별 예제로 실제 빌드·실행해 비교하고 선정했다. 선정한 기술은 #1의 백엔드 골격에 적용했다.

| 항목 | 선정 | 버전 |
|------|------|------|
| 웹 프레임워크 | Echo v5 | v5.4.4 |
| DB 접근 | sqlc + pgx | sqlc v1.31.1, pgx v5.12.0 |
| 마이그레이션 | goose | v3.29.0 |
| 스케줄러 실행 위치 | 별도 수집 프로세스 (`cmd/collector`, compose `collector`) | - |
| 일정 표현 | gocron v2 | v2.23.0 |

이제 백엔드는 시작할 때 DB에 연결하고 마이그레이션을 적용한다. `/healthz`는 DB 상태를 함께 보고한다. 수집기는 매일 05:30 KST에 작업을 실행할 준비가 되어 있다. #4는 마이그레이션 파일과 sqlc 쿼리를 추가하고, #5·#6은 수집기에 작업을 등록하면 된다.

## 수행 결과

| 단계 | 내용 | 커밋 |
|------|------|------|
| 계획 | 수행 계획서, 구현 계획서 | `505fc2e` |
| 1단계 | 웹 프레임워크 비교 (`net/http`, Gin, Echo v5): 같은 기능, 같은 테스트 8개 | `78d0aac` |
| 2단계 | DB 접근 비교 (pgx, sqlc, GORM): 같은 테스트 6개. 마이그레이션 비교 (golang-migrate, goose) | `9e07b1f` |
| 3단계 | 스케줄러 비교 (`time.Ticker`, robfig/cron, gocron v2), 선정안 확정 | `6750a38` |
| 4단계 | 선정 기술을 백엔드 골격에 적용, compose·문서 갱신 | `f7a8e62` |

## 산출물

### 선정 근거 요약

| 항목 | 선정 이유 | 탈락 이유 |
|------|-----------|-----------|
| Echo v5 | 로그·패닉 복구·CORS 내장, 오류 응답을 한 곳에서 처리, 바이너리 의존 모듈 2개 | Gin: 의존 모듈 19개·바이너리 20.6 MiB, 검증 메시지 변환 필요. `net/http`: 미들웨어와 404·405 JSON 처리를 직접 유지 |
| sqlc + pgx | SQL 오류를 코드 생성 시 발견, 타입 있는 함수 생성, `COPY` 대량 저장, 마이그레이션 파일을 스키마로 읽음 | GORM: 시계열 집계를 결국 SQL로 작성, 대량 저장이 `INSERT` |
| goose | 실패해도 이전 버전 유지, 적용할 것이 없어도 오류 없음, up·down 한 파일 | golang-migrate: 실패 후 dirty 상태로 수동 복구 필요 |
| 별도 수집 프로세스 | API 재시작과 수집이 무관, 장애 격리, 수동 실행 쉬움 | API 내장: 개발 중 재시작마다 수집이 끊김 |
| gocron v2 | 겹침 방지·시간대·종료 대기·취소 전달 내장, 유지 중 | robfig/cron: 2020년 이후 릴리스 없음. `time.Ticker`: 종료 경합 등을 직접 처리 |

상세 비교는 `mydocs/tech/backend_stack.md`에 있다.

### 백엔드 구조

```
src/backend/
├── cmd/
│   ├── server/        ← API 서버 (DB 연결 → 마이그레이션 → Echo)
│   ├── collector/     ← 수집기 (매일 05:30 KST, -once, COLLECTOR_INTERVAL)
│   └── migrate/       ← 마이그레이션 관리 (status, up, down)
├── internal/
│   ├── server/        ← Echo 라우트, 공통 오류 처리, /healthz
│   ├── db/            ← 연결 풀, goose 실행 (세션 잠금)
│   └── scheduler/     ← gocron 래퍼
├── migrations/        ← goose SQL (실행 파일에 포함)
└── sqlc.yaml          ← 스키마: migrations/, 쿼리·생성 코드는 #4부터
```

### 서비스

| 서비스 | 변경 |
|--------|------|
| `backend` | 엔트리포인트가 `APP`(기본 `server`)에 해당하는 `cmd/`를 빌드·실행 |
| `collector` | 신규. backend와 같은 이미지·소스, `APP=collector` |

### 사용 방법

```bash
cd Docker
docker compose up -d --build
docker compose exec backend go run ./cmd/migrate          # 마이그레이션 상태
docker compose exec backend go run ./cmd/collector -once  # 수집 즉시 1회
docker compose exec backend go tool sqlc generate         # sqlc 코드 생성
```

전체 명령은 CLAUDE.md `빌드 및 실행` 절에 있다.

## 발견 사항

- **Echo v5의 사용자 정의 오류 처리기는 응답 커밋 여부를 확인해야 한다.** 확인하지 않으면 오류 응답이 두 번 쓰인다. 적용 코드에 반영했고 테스트(`TestErrorWrittenOnce`)로 고정했다.
- **sqlc 생성기는 CGO 없이 개발 컨테이너에서 실행된다.** 첫 실행 85초, 이후 2.5초이다.
- **golang-migrate는 실패 후 실제 스키마가 롤백되었는데도 dirty 버전으로 남는다.** goose는 이전 버전을 유지한다.
- **`time.Ticker` 직접 구현에서 종료 경합 버그를 확인했다.** 종료 신호와 틱이 함께 준비되면 작업을 하나 더 시작했다.
- **goose CLI를 도구로 넣으면 `go.mod`가 크게 늘어난다.** 모든 DB 드라이버가 따라 들어와 56줄에서 112줄이 되었다. 그래서 라이브러리로 `cmd/migrate`를 만들었다.
- **서버와 수집기가 동시에 시작해도 마이그레이션은 한 번만 적용된다** (goose 세션 잠금).
- 예제 비교는 모두 Go 1.27.3 컨테이너에서 했다. DB 예제는 임시 데이터베이스 `poc_task3`에서 실행하고 삭제했으며, 예제 코드는 저장소에 넣지 않았다.

## 검증

| 항목 | 결과 |
|------|------|
| 이슈 범위: Gin vs Echo | ✅ `net/http`를 기준선으로 함께 비교, Echo v5 선정 |
| 이슈 범위: DB 접근 방식, 마이그레이션 도구 | ✅ sqlc + pgx, goose 선정 |
| 이슈 범위: 스케줄러 방식 | ✅ 별도 수집 프로세스 + gocron v2 선정 |
| 빌드·정적 검사 | ✅ `go build`, `go vet` 통과, `gofmt` 결과 없음 |
| 테스트 | ✅ 8 passed (server 6, scheduler 2), 0 failed |
| 기동 | ✅ db healthy, backend·collector·frontend Up |
| `/healthz` | ✅ 정상 200, DB 중지 시 503, 복구 시 200. 프록시 경유 `/api/healthz` 200 |
| 마이그레이션 | ✅ status → down → up |
| 수집기 | ✅ 짧은 간격 실행, 정상 종료(종료 코드 0), `-once` |
| 재현성 | ✅ `down` → `up --build` 후 재확인 (볼륨 유지 상태) |
| 비교 예제가 저장소에 없음 | ✅ |

## 계획 대비 차이

- **Echo는 v5로 비교했다.** 이슈에는 버전 표기가 없었고, v4도 계속 릴리스되고 있으나 신규 프로젝트에는 최신 메이저 버전이 맞다고 판단했다.
- **동시 마이그레이션은 계획의 2개 대신 3개를 10회 반복했다.** 1회로는 경합이 드러나지 않을 수 있어서이다.
- **goose CLI를 도구로 넣지 않고 `cmd/migrate`를 만들었다** (3단계 적용안에서 변경).
- **sqlc 생성 코드는 만들지 않았다.** 테이블이 없어 쿼리가 없다. 설정과 실행 확인까지 했다.
- **CORS 미들웨어는 넣지 않았다.** 현재는 Vite 프록시로 같은 출처에서 호출한다.
- **기존 `/healthz` 테스트의 Content-Type 검사를 `application/json` 접두어 확인으로 완화했다.** Echo가 `charset`을 붙이지 않기 때문이다.
- **중복 실행 방지용 advisory lock은 넣지 않았다.** 수집기를 1개만 띄우는 구성이다.
- 볼륨까지 지운 새 환경(`down -v`)에서의 기동은 검증하지 않았다.

## 후속 타스크로 넘기는 사항

| 타스크 | 넘기는 사항 |
|--------|-------------|
| #4 DB 스키마 | `migrations/`에 goose 형식(`00002_...sql`)으로 테이블 추가, `internal/db/queries/`에 첫 쿼리 작성 후 `go tool sqlc generate`. 금액(`numeric`)의 Go 타입과 sqlc `overrides` 결정 |
| #5, #6, #7 수집기 | `cmd/collector`의 `collect-prices` 작업에 실제 수집을 연결. 작업은 `ctx` 취소를 확인해 중간에 멈출 수 있게 작성. 수집기를 여러 개 띄우게 되면 advisory lock 추가 |
| #8 조회 API | Echo 라우트 추가, 오류는 `echo.NewHTTPError`로 반환. 입력 검증 방식(직접 작성 또는 go-playground/validator 연결) 결정 |
| #16 배포 | 운영 이미지에서 server·collector 두 실행 파일 구성, CORS 필요 여부, 시간대 데이터(수집기는 `time/tzdata` 포함) 확인 |

## 이슈

- https://github.com/Chocobone/PokeCardPrice/issues/3
