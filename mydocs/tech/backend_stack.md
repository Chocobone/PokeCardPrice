# 기술 조사: 백엔드 기술 스택

> **작성일**: 2026-11-02
> **관련 타스크**: [#3](https://github.com/Chocobone/PokeCardPrice/issues/3) 백엔드 기술 선정
> **상태**: 1단계(웹 프레임워크), 2단계(DB 접근·마이그레이션) 작성. 스케줄러와 선정 결과는 3단계에서 추가한다.

---

## 배경

#1에서 만든 백엔드 골격은 표준 라이브러리 `net/http`만 사용한다. #4 이후 타스크가 쓸 웹 프레임워크, DB 접근 방식, 마이그레이션 도구, 스케줄러 방식을 실제 예제로 비교해 정한다.

- 확인일은 별도 표시가 없으면 모두 2026-11-02이다.
- 예제는 `golang:1.27-alpine3.24`(Go 1.27.3) 컨테이너에서 빌드·실행했다. 예제 코드는 세션 임시 폴더에 두었고 저장소에는 넣지 않았다.

## 조사 내용

### 1. 웹 프레임워크

#### 후보와 버전

| 후보 | 버전 | 릴리스 | 비고 |
|------|------|--------|------|
| 표준 라이브러리 `net/http` | Go 1.27.3 | - | Go 1.22부터 `ServeMux`가 메서드·경로 변수 패턴(`GET /cards/{id}`)을 지원 |
| Gin (`github.com/gin-gonic/gin`) | v1.12.0 | 2026-02-28 | 직전 릴리스 v1.11.0은 2025-09-20 |
| Echo (`github.com/labstack/echo/v5`) | v5.4.4 | 2026-10-30 | v5.0.0은 2026-01-18. v4도 v4.16.3(2026-10-30)으로 함께 관리됨 |

Echo는 최신 메이저인 v5를 대상으로 했다. 릴리스 날짜는 Go 모듈 프록시(`go list -m -json`)로 확인했다.

#### 예제 기능

세 후보로 같은 기능을 구현하고, **같은 테스트 파일**(테스트 8개)을 그대로 적용했다. 테스트는 `newApp() http.Handler`만 호출한다.

| 기능 | 테스트 내용 |
|------|-------------|
| `GET /healthz` | 200, `{"status":"ok"}`, JSON Content-Type |
| `GET /cards/{id}` | 경로 변수, 없는 카드면 404와 `{"error":...}` |
| `GET /cards?q=&page=` | `page` 기본값 1, 0·문자는 400, `q` 101자 이상은 400 |
| `POST /collections` | JSON 본문, 필수 값 누락·수량 0·잘못된 JSON은 400 |
| `/me/*` 그룹 | 인증 미들웨어. 토큰이 없으면 401, 있으면 200 |
| 패닉 복구 | 핸들러 패닉 시 500과 JSON 오류 |
| CORS | `http://localhost:5173` 사전 요청 허용, 다른 출처는 허용 헤더 없음 |
| 404·405 | 없는 경로 404, 허용하지 않는 메서드 405, 둘 다 JSON 오류 |

정상 종료는 세 후보 모두 `http.Server`에 핸들러로 넣는 같은 `main` 구조를 사용했다. 실행 중 SIGTERM을 보내 종료 로그와 종료 코드 0을 확인했다.

#### 측정 결과

| 항목 | `net/http` | Gin v1.12.0 | Echo v5.4.4 |
|------|-----------|-------------|-------------|
| `go vet`, `gofmt` | 통과 | 통과 | 통과 |
| 공통 테스트 8개 | 통과 | 통과 | 통과 (오류 처리기 수정 후, 아래 참고) |
| SIGTERM 정상 종료 | 확인 | 확인 | 확인 |
| 모듈 그래프 (`go list -m all`, 자신 포함) | 1 | 56 | 9 |
| 바이너리에 포함된 의존 모듈 (`go version -m`) | 0 | 19 | 2 |
| 바이너리 크기 (`-ldflags="-s -w"`) | 6.4 MiB | 20.6 MiB | 7.0 MiB |
| 예제 코드 (`main.go`, 빈 줄 제외) | 173줄 | 107줄 | 103줄 |

- 측정은 각 1회이다. 바이너리 크기는 Go 버전과 빌드 옵션에 따라 달라진다.
- 성능 벤치마크는 구현 계획서에 따라 하지 않았다.

#### 기능별 비교

| 항목 | `net/http` | Gin | Echo v5 |
|------|-----------|-----|---------|
| 경로 변수 | `r.PathValue("id")` | `c.Param("id")` | `c.Param("id")` |
| 쿼리·본문 바인딩 | 직접 파싱 | 구조체 태그로 바인딩 (`form`, `json`, 기본값 지정 가능) | `c.Bind`로 바인딩 |
| 검증 | 직접 작성 | 내장 (`binding:"required,min=1"`, go-playground/validator) | 내장 검증기 없음. `Validator` 인터페이스에 외부 검증기를 연결하거나 직접 작성 |
| 핸들러 오류 처리 | 핸들러마다 응답 작성 | 핸들러마다 응답 작성 | 핸들러가 `error`를 반환하고 중앙 오류 처리기가 응답 |
| 로그·복구·CORS 미들웨어 | 직접 작성 (약 50줄) | 복구 내장, CORS는 별도 모듈(`gin-contrib/cors`) 또는 직접 작성, 로그는 직접 작성(`slog` 사용 시) | 셋 다 내장 (`RequestLogger`, `Recover`, `CORS`) |
| 404·405 JSON 응답 | 기본 응답이 텍스트. 응답 래퍼를 직접 작성해야 함 (약 30줄) | `NoRoute`, `NoMethod`로 지정. 405는 `HandleMethodNotAllowed = true` 필요 | 중앙 오류 처리기 하나로 처리 |
| 라우트 그룹 | 하위 `ServeMux`를 만들어 미들웨어로 감쌈 | `r.Group("/me", auth())` | `e.Group("/me", auth)` |
| 테스트 | `httptest`로 바로 가능 | `httptest`로 바로 가능 | `httptest`로 바로 가능 |
| 로그 형식 | `slog` 직접 사용 | 기본 로거는 텍스트. 예제는 `slog` 미들웨어를 직접 작성 | `RequestLogger`가 `slog` JSON 로그 출력 |

#### 확인한 주의점

- **Echo v5: 사용자 정의 오류 처리기는 응답이 이미 나갔는지 확인해야 한다.** 처음 구현에서 오류 응답이 두 번 쓰였다(`{"error":"Not Found"}`가 본문에 2회). `RequestLogger`가 오류 처리기를 먼저 호출하고, 프레임워크가 다시 호출하기 때문이다. 기본 처리기처럼 `echo.UnwrapResponse(c.Response())`의 `Committed`를 확인해 해결했다.
- **Gin: 검증 오류 메시지가 내부 구조체 이름을 드러낸다.** 예: `Key: 'collectionReq.CardID' Error:Field validation for 'CardID' failed on the 'required' tag`. 사용자에게 보여 주려면 메시지 변환이 필요하다.
- **Gin: `required`는 0을 누락으로 본다.** 수량 0을 보내면 `min` 대신 `required` 위반으로 보고된다. 0이 의미 있는 값인 필드에는 `required`를 쓰지 않거나 포인터 타입을 써야 한다.
- **Gin: 의존성이 많다.** 모듈 그래프에 QUIC, MongoDB 드라이버, protobuf 등이 포함된다(바이너리에는 19개만 들어감). 바이너리가 다른 후보의 약 3배이다.
- **`net/http`: 기본 404·405 응답이 텍스트다.** JSON으로 바꾸려면 응답 래퍼가 필요하고, 래퍼는 상태 코드와 Content-Type으로 프레임워크 응답인지 판단해야 해서 깨지기 쉽다.
- **Echo v5는 v5.0.0 이후 약 10개월 된 메이저 버전이다.** 검색되는 예제 상당수가 v4 기준(`echo.Context` 인터페이스)이라 v5(`*echo.Context` 구조체)와 다르다.

### 1단계 잠정 판단

선정은 3단계에서 확정한다. 1단계 결과만으로 본 판단은 다음과 같다.

1. **Echo v5가 가장 유력하다.** 필요한 미들웨어가 내장되어 있고, 핸들러가 오류를 반환하는 구조라 #8~#11의 오류 응답을 한 곳에서 맞출 수 있다. 의존성(바이너리 2개 모듈)과 바이너리 크기도 표준 라이브러리와 비슷하다.
2. **`net/http`는 의존성이 없다는 장점이 크지만, 공통 기능을 직접 유지해야 한다.** 미들웨어와 404·405 JSON 처리에 약 80줄이 들었고, 검증도 모두 직접 작성해야 한다.
3. **Gin은 검증 내장이 장점이지만, 의존성과 바이너리 크기가 가장 크다.** 검증 메시지 변환도 추가로 필요해, 이 프로젝트 규모에서는 이점이 작다.
4. Echo를 고르면 검증은 직접 작성하거나 go-playground/validator를 `Validator`에 연결해야 한다. 이 선택은 #8 구현 계획에서 정해도 된다.

### 2. DB 접근 방식

#### 후보와 버전

| 후보 | 버전 | 릴리스 | 방식 |
|------|------|--------|------|
| pgx | `github.com/jackc/pgx/v5` v5.12.0 | 2026-10-31 | SQL을 직접 작성하고 결과를 직접 스캔 |
| sqlc | `github.com/sqlc-dev/sqlc` v1.31.1 (생성기) + pgx v5.12.0 | 2026-04-22 | SQL 파일에서 타입이 있는 Go 코드를 생성. 런타임은 pgx |
| GORM | `gorm.io/gorm` v1.31.2, `gorm.io/driver/postgres` v1.6.3 | 2026-06-22, 2026-09-14 | 구조체 모델 기반 ORM. 드라이버는 pgx(`database/sql` 경유) |

#### 예제

#2 설계 입력을 줄인 테이블 2개(`cards`, `price_history`: 복합 기본 키, `numeric(12,2)` 가격, NULL 허용)로 같은 저장소 인터페이스를 구현하고, **같은 테스트 6개**를 compose `db`(PostgreSQL 18.6)의 임시 데이터베이스 `poc_task3`에서 실행했다. 후보마다 스키마를 따로 만들어 서로 간섭하지 않게 했다.

| 테스트 | 내용 |
|--------|------|
| 단건 조회 | 없는 카드는 공통 `ErrNotFound`로 구분 |
| 일괄 저장 | 가격 1,000건 저장, 3회 측정 |
| 업서트 | 같은 카드·소스·날짜면 가격 갱신, 행 수 1 유지 |
| 주별 평균 | `date_trunc('week')`, `avg`, 기간 조건, NULL 제외 |
| 트랜잭션 | 카드와 가격을 한 트랜잭션으로 저장, 오류 시 둘 다 롤백 |
| NULL·정밀도 | NULL 가격, `1234567890.12`를 오차 없이 왕복, 날짜·통화 |

#### 측정 결과

| 항목 | pgx | sqlc | GORM |
|------|-----|------|------|
| 테스트 6개 | 통과 | 통과 | 통과 |
| 일괄 저장 1,000건 (중앙값) | 57.0 ms (`COPY`) | 51.8 ms (`COPY`, `:copyfrom`) | 62.3 ms (`INSERT` 일괄) |
| 모듈 그래프 | 16 | 16 | 22 |
| 테스트 바이너리의 의존 모듈 | 6 | 6 | 10 |
| 작성한 코드 | `db.go` 102줄 | `db.go` 96줄 + `queries.sql` 20줄 (+ 생성 코드 243줄) | `db.go` 115줄 |

- 일괄 저장 시간은 3회 측정의 중앙값이다. 세 후보 모두 하루 1회 수집에는 충분하고, 차이는 판단 근거로 삼지 않았다.
- 작성 코드에는 공통 인터페이스로 맞추는 변환 코드가 포함되어 있다.

#### 기능별 비교

| 항목 | pgx | sqlc | GORM |
|------|-----|------|------|
| SQL 오류 발견 시점 | 실행할 때 | **코드 생성할 때** (없는 컬럼 `nam` → `column "nam" does not exist`) | 실행할 때 |
| 결과 타입 | 직접 스캔. 오타·순서 실수는 실행 시 발견 | 쿼리마다 매개변수·결과 구조체 생성 | 모델 구조체. 집계 결과는 `Raw().Scan()`으로 별도 구조체 |
| 복잡한 집계 SQL | 그대로 작성 | 그대로 작성 (`sqlc.arg`로 매개변수 이름 지정) | `Raw`로 SQL 직접 작성 |
| 대량 저장 | `CopyFrom` | `:copyfrom` 주석으로 `COPY` 함수 생성 | `CreateInBatches` (`INSERT` 여러 행) |
| 업서트 | SQL `ON CONFLICT` | SQL `ON CONFLICT` | `clause.OnConflict` 빌더 |
| 트랜잭션 | `pgx.BeginFunc` | `pgx.BeginFunc` + `q.WithTx(tx)` | `db.Transaction` |
| not found | `pgx.ErrNoRows` | `pgx.ErrNoRows` | `gorm.ErrRecordNotFound` |
| `numeric`·`date` 타입 | `*string`, `time.Time`로 바로 스캔 가능 | 기본값은 `pgtype.Numeric`, `pgtype.Date`. `sqlc.yaml` `overrides`로 `*string`, `time.Time` 지정 | 모델 태그 `type:numeric(12,2)`, `*string` |
| 추가 도구 | 없음 | 생성기 실행 필요 | 없음 |

#### 확인한 주의점

- **sqlc 생성기는 CGO 없이 컨테이너에서 실행된다.** `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`가 `golang:1.27-alpine3.24`에서 동작했다. 첫 실행은 내려받기와 컴파일로 85초, 이후에는 2.5초였다(`go-build` 캐시 볼륨 사용).
- **sqlc는 마이그레이션 파일을 스키마로 읽는다.** golang-migrate 형식(`.up.sql`만 사용)과 goose 형식(`-- +goose Down` 이후 무시) 모두 `schema`에 디렉터리를 지정하면 마지막 상태(`set_code` 컬럼 포함)로 코드를 생성했다. 스키마 파일을 따로 관리하지 않아도 된다.
- **sqlc 기본 타입은 `pgtype`이다.** 금액(`numeric`)을 `pgtype.Numeric`으로 다루기 불편하면 `overrides`로 타입을 지정해야 한다. 금액 타입(문자열, 정수 단위, decimal 라이브러리)은 #4에서 정한다.
- **GORM은 이 프로젝트의 주요 쿼리(시계열 집계)를 결국 SQL로 쓴다.** ORM 이점은 단순 CRUD에 한정된다. 대량 저장도 `COPY`가 아닌 `INSERT`이다.

### 3. 마이그레이션 도구

#### 후보와 버전

| 후보 | 버전 | 릴리스 | 이력 테이블 |
|------|------|--------|-------------|
| golang-migrate | `github.com/golang-migrate/migrate/v4` v4.20.1 | 2026-09-09 | `schema_migrations` (버전, dirty) |
| goose | `github.com/pressly/goose/v3` v3.29.0 | 2026-10-11 | `goose_db_version` (적용 이력) |

#### 예제

같은 마이그레이션 2개(1: 테이블 생성, 2: 컬럼 추가)와 실패용 3번(컬럼 추가 후 `SELECT 1/0`)을 두 도구 형식으로 만들고, `go:embed`로 프로그램에 넣어 Go 라이브러리로 실행했다.

| 형식 | golang-migrate | goose |
|------|----------------|-------|
| 파일 | `000001_init.up.sql`, `000001_init.down.sql` (방향별 파일) | `00001_init.sql` 한 파일에 `-- +goose Up`, `-- +goose Down` |

#### 결과

| 시나리오 | golang-migrate | goose |
|----------|----------------|-------|
| up → down 1 → up | 성공 (버전 2 → 1 → 2) | 성공 |
| 적용할 것이 없을 때 up | `ErrNoChange` 오류를 반환하므로 호출 측에서 무시해야 함 | 결과 0건, 오류 없음 |
| 3개 동시 up × 10회 | 오류 0, 최종 버전 정상. 내장 advisory lock 사용 | 오류 0, 최종 버전 정상 (세션 잠금 사용·미사용 모두) |
| 실패 마이그레이션 | 오류 반환, **버전 3, dirty=true**로 남음. 변경은 롤백됨 | 오류 반환, **버전 2 유지**. 트랜잭션으로 롤백됨 |
| 실패 후 재실행 | `Dirty database version 3. Fix and force version.` → `Force(2)`로 수동 복구 필요 | 같은 오류로 다시 실패. 파일을 고치면 그대로 재실행 가능 |
| 단독 사용 시 바이너리 의존 모듈 | 8 | 10 |
| 단독 사용 시 모듈 그래프 | 212 | 100 |
| CLI 설치 (`go install`) | 4.2초, 15.0 MB (`-tags pgx5` 필요) | 31.5초, 62.8 MB |

- goose 동시 실행은 잠금 없이도 이번 시험에서 오류가 없었다. 원인은 확인하지 않았고, 우연일 수 있다. 쓰게 되면 공식 권장대로 세션 잠금(`lock.NewPostgresSessionLocker`)을 켠다.
- 두 도구 모두 `go:embed`로 마이그레이션을 바이너리에 넣어 서버·수집기 시작 시 적용할 수 있다.

#### 확인한 주의점

- **golang-migrate는 실패 시 dirty 상태가 되어 수동 복구가 필요하다.** PostgreSQL에서는 실패한 파일이 롤백되므로 실제 스키마는 이전 상태인데도 버전 표시는 3으로 남는다. `Force`로 버전을 되돌려야 한다.
- **goose는 마이그레이션마다 트랜잭션을 쓴다.** `CREATE INDEX CONCURRENTLY`처럼 트랜잭션 안에서 실행할 수 없는 문장은 `-- +goose NO TRANSACTION` 표시가 필요하다고 알려져 있다(`미확인`, 이번에 시험·문서 확인하지 않음).
- **golang-migrate CLI는 빌드 태그로 DB 드라이버를 고른다.** 태그 없이 설치하면 PostgreSQL을 쓸 수 없다.

### 2단계 잠정 판단

1. **DB 접근은 sqlc가 가장 유력하다.** 시계열 집계처럼 SQL로 써야 하는 쿼리가 많고, sqlc는 그 SQL을 코드 생성 시점에 검사해 타입이 있는 함수로 만든다. 런타임은 pgx 그대로라 `COPY`와 트랜잭션도 pgx 방식으로 쓴다. 생성기는 CGO 없이 컨테이너에서 실행된다.
2. **pgx 직접 사용은 sqlc의 대안이다.** 의존성이 같고 도구가 필요 없지만, 쿼리가 늘수록 스캔 코드와 실수 가능성이 커진다. sqlc를 써도 생성 코드로 표현하기 어려운 쿼리는 pgx로 직접 쓸 수 있다.
3. **GORM은 제외한다.** 주요 쿼리를 결국 SQL로 써야 하고, 대량 저장이 `COPY`가 아니며, 의존성이 가장 많다.
4. **마이그레이션은 goose가 유력하다.** 실패해도 dirty 상태가 남지 않아 수동 복구가 필요 없고, 적용할 것이 없을 때 오류를 반환하지 않으며, 한 파일에 up·down을 둔다. sqlc도 goose 형식을 읽는다.


## 참고 자료

- Go `net/http` ServeMux 패턴: https://pkg.go.dev/net/http#ServeMux
- Gin: https://github.com/gin-gonic/gin , https://pkg.go.dev/github.com/gin-gonic/gin@v1.12.0
- Echo v5: https://github.com/labstack/echo , https://pkg.go.dev/github.com/labstack/echo/v5@v5.4.4
- pgx: https://github.com/jackc/pgx , https://pkg.go.dev/github.com/jackc/pgx/v5@v5.12.0
- sqlc: https://docs.sqlc.dev/ (설정, `overrides`, `:copyfrom`, 마이그레이션 디렉터리 스키마)
- GORM: https://gorm.io/docs/
- golang-migrate: https://github.com/golang-migrate/migrate
- goose: https://github.com/pressly/goose , https://pressly.github.io/goose/
