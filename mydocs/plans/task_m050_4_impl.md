# Task #4: DB 스키마 설계 및 마이그레이션 — 구현 계획서

수행 계획서: `mydocs/plans/task_m050_4.md`

1~2단계는 설계 단계라 문서만 작성하고 저장소 소스는 바꾸지 않는다. 2단계 설계를 작업지시자가 승인한 뒤 3~4단계에서 `src/backend/`, `Docker/`, `CLAUDE.md`를 변경한다.

## 공통 규칙

### 설계 확인용 임시 DB

- 1~2단계에서 SQL 동작(열거 타입, 제약, 인덱스 크기, 실행 계획)을 확인할 때는 compose의 `db` 서비스에 **임시 데이터베이스 `poc_task4`**를 만들어 쓰고, 단계가 끝나면 삭제한다. 개발 DB에는 손대지 않는다.
- 확인용 SQL과 데이터 생성 스크립트는 세션 임시 폴더에 두고, 결과 문서에는 측정값과 필요한 발췌만 남긴다.

### 테스트용 DB (3~4단계)

- 마이그레이션·쿼리 테스트는 **별도 데이터베이스 `pokecard_test`**에서 실행한다. 테스트가 데이터를 지우고 다시 만들기 때문에 개발 DB와 분리한다.
- 접속 정보는 환경 변수 `TEST_DATABASE_URL`로 받는다. 값이 없으면 DB 테스트는 `t.Skip`으로 건너뛰어, DB 없이도 `go test ./...`가 실패하지 않게 한다.
- `pokecard_test` 생성 방법(compose 초기화 스크립트 또는 테스트 헬퍼가 `CREATE DATABASE` 실행)은 3단계에서 둘 중 하나로 정하고 보고서에 남긴다.

### 결과 문서

**`mydocs/tech/db_schema.md`** (신규, `tech_investigation_template.md` 형식). 1단계에서 엔터티·결정 사항, 2단계에서 상세 정의를 쓰고, 3~4단계에서 실제 마이그레이션·쿼리와 맞춰 갱신한다.

### 명명 규칙

- 테이블은 복수형 snake_case(`cards`, `price_history`는 예외로 이력 성격을 살려 단수 유지 여부를 1단계에서 정한다), 컬럼은 snake_case.
- 시각은 `timestamptz`, 기준일·거래일은 `date`.
- 제약 이름은 `{테이블}_{컬럼}_{종류}` (예: `cards_set_id_fkey`, `price_history_uniq`).

---

## 1단계: 요구 사항 정리와 엔터티 설계

### 목표

#4에 넘어온 요구 사항을 빠짐없이 엔터티로 옮기고, 이후 상세 설계를 좌우하는 결정 사항의 선택지를 비교해 선정안을 낸다.

### 구현

**요구 사항 추적표.** 다음 출처의 요구를 한 표로 모으고, 각 요구가 어느 테이블·컬럼으로 반영되는지 적는다. 반영하지 않는 요구는 사유와 담당 타스크를 적는다.

| 출처 | 내용 |
|------|------|
| 이슈 #4 본문 | 세트, 카드, 가격 소스, 날짜별 시세 이력(가격 성격 구분), 환율, 사용자, 컬렉션, 시계열 인덱스 |
| `price_data_sources.md` "후속 타스크 설계 입력" | 카드 언어, 소스별 외부 ID(TCGdex ID, TCGplayer `productId`+`subTypeName`, Cardmarket 상품 ID), 가격 이력 컬럼, 가격 성격 값 4종, 환율 컬럼, 사용자 거래가 컬럼, 판매자 정보 미저장 |
| 이슈 #5 / #6 / #7 | 한글판 → 일본판 연결, 소스 갱신일이 오래된 가격 구분, 환산 참고가 저장, 관리자 입력가의 출처·확인일 |
| 이슈 #8 / #10 | 이름·세트·번호 검색, 기간별 추이·변동률·급등락 순위, 보유 수량, 총 평가액, 사용자 거래가 검증 상태 |

**엔터티 초안 (ERD).** Mermaid `erDiagram`으로 그린다. 현재 후보는 다음과 같으며 1단계에서 확정한다.

| 엔터티 | 역할 |
|--------|------|
| `sets` | 세트 (언어별, TCGdex 세트 ID·세트 코드) |
| `cards` | 카드 (세트, 수록 번호, 이름, 언어, 희귀도, 이미지 URL) |
| `card_variants` | 카드 변형 (Normal, Holofoil, Reverse Holofoil 등). 가격의 기준 단위 |
| `card_links` 또는 `cards` 컬럼 | 한글판 → 일본판 등 카드 간 연결 (자동/수동 구분) |
| `price_sources` | 가격 소스 (TCGCSV, TCGdex Cardmarket, 일본판 환산, 사용자 등록 집계, 관리자 입력) |
| `external_ids` | 소스별 외부 ID와 카드 변형 연결 |
| `price_history` | 날짜별 시세 이력 |
| `fx_rates` | 환율 |
| `users` | 사용자 (최소 컬럼) |
| `collection_items` | 사용자 보유 카드·수량 |
| `user_trades` | 사용자 거래가 (검증 상태 포함, 원본 입력) |

**결정 사항 비교.** 각 항목의 선택지와 장단점, 선정안을 쓴다.

| 결정 사항 | 선택지 | 확인 방법 |
|-----------|--------|-----------|
| 열거 값 표현 (언어, 통화, 가격 성격, 검증 상태 등) | PostgreSQL `enum` / `text` + `CHECK` / 참조 테이블 | `poc_task4`에서 값 추가 마이그레이션 작성 비용과 sqlc 생성 타입 비교 |
| 기본 키 | `bigint generated always as identity` / `uuid` | 사용자처럼 외부 노출되는 ID는 `uuid` 검토 (PostgreSQL 18 `uuidv7()`) |
| 가격 이력 형태 | 긴 형태(가격 필드 종류를 행으로) / 넓은 형태(필드별 컬럼) | 소스마다 필드가 다른 점(TCGCSV `low`·`mid`·`high`·`market`·`directLow`, Cardmarket `avg`·`low`·`trend`·`avg1`·`avg7`·`avg30`)과 조회 쿼리 형태로 판단 |
| 사용자 거래가 저장 위치 | `user_trades`만 두고 집계 결과를 `price_history`에 저장 / `price_history`에 함께 저장 | 검증 상태·사용자 소유 여부, #7 집계 흐름으로 판단 |
| 관리자 입력가의 출처·확인일 | `price_history`에 선택 컬럼 / 별도 테이블 | 다른 소스 행에 빈 컬럼이 생기는 정도로 판단 |
| 금액 Go 타입 (sqlc `overrides`) | `pgtype.Numeric` / `string` / decimal 라이브러리(`shopspring/decimal` 등) | 4단계에서 생성 코드로 최종 확인. 1단계에서는 후보와 판단 기준만 정함 |

**`mydocs/working/task_m050_4_stage1.md`** — 단계별 완료 보고서

### 검증

- 요구 사항 추적표의 모든 항목이 테이블·컬럼에 대응되거나, 반영하지 않는 사유가 적혀 있다.
- 결정 사항마다 선정안과 근거가 있다. 열거 값 표현은 `poc_task4`에서 확인한 결과를 근거로 한다.

---

## 2단계: 상세 설계와 시세 이력 규모 추정

### 목표

테이블별 컬럼·타입·제약·인덱스를 확정하고, 시세 이력의 행 수와 인덱스 크기를 추정해 파티셔닝 필요 여부를 판단한다. **이 단계 결과를 작업지시자가 승인하면 3단계에서 SQL을 작성한다.**

### 구현

**테이블 정의.** 테이블마다 컬럼(이름, 타입, NULL 허용, 기본값, 설명), 기본 키, 외래 키와 삭제 동작(`CASCADE`/`RESTRICT`), 유일 제약, `CHECK` 제약을 표로 쓴다. 다음 제약은 반드시 포함한다.

| 제약 | 목적 |
|------|------|
| `price_history` 유일 제약 (카드 변형, 소스, 가격 필드, 기준일) | 수집기 재실행 시 중복 방지 (`ON CONFLICT ... DO UPDATE`) |
| `external_ids` 유일 제약 (소스, 외부 ID) | 같은 외부 상품이 두 변형에 연결되지 않게 함 |
| `fx_rates` 유일 제약 (기준 통화, 대상 통화, 기준일, 제공처) | 환율 중복 방지 |
| `collection_items` 유일 제약 (사용자, 카드 변형, 상태 등급) | 보유 카드는 수량으로 관리 |
| 금액 `CHECK (price >= 0)`, 수량 `CHECK (quantity > 0)` | 잘못된 값 방지 |

**인덱스 설계.** #8·#10의 대표 쿼리를 먼저 SQL로 적고, 각 쿼리가 쓸 인덱스를 정한다.

| 대표 쿼리 | 예상 인덱스 |
|-----------|-------------|
| 카드 변형·소스별 기간 추이 (7일~1년) | `price_history (card_variant_id, source_id, price_field, price_date)` (유일 제약과 겸용 여부 확인) |
| 카드별 최신가 | 위 인덱스의 역순 스캔 또는 `DISTINCT ON` 실행 계획 확인 |
| 날짜별 전체 변동률·급등락 순위 | `price_history (price_date, ...)` 필요 여부 확인 |
| 이름 검색 (부분 일치, 한글 포함) | `pg_trgm` GIN 인덱스 / `lower(name)` B-tree 비교 |
| 세트·번호 검색 | `cards (set_id, number)` |
| 사용자별 컬렉션·거래가 | `collection_items (user_id)`, `user_trades (card_variant_id, trade_date)` |

**규모 추정.** 카드 수(영어·일본어·한글), 카드당 평균 변형 수, 소스별 저장 가격 필드 수, 하루 행 수와 1년 행 수를 계산한다. `poc_task4`에 추정치 규모(최소 1년치)의 가짜 데이터를 넣고 테이블·인덱스 크기와 대표 쿼리의 `EXPLAIN ANALYZE`를 기록한다.
- 행 수가 크면 **저장할 가격 필드를 줄이는 방안**(예: 소스별 대표 필드만 저장)을 파티셔닝보다 먼저 검토한다.
- 파티셔닝이 필요하다고 판단되면 근거와 함께 제안만 하고, 도입 여부는 작업지시자가 정한다.

**마이그레이션 파일 구성안.** `00001_init.sql`은 그대로 두고 이어서 추가한다.

| 파일 | 내용 |
|------|------|
| `00002_catalog.sql` | 확장(`pg_trgm` 등 필요 시), 열거 타입, `sets`, `cards`, `card_variants`, 카드 연결 |
| `00003_prices.sql` | `price_sources`(초기 행 포함), `external_ids`, `price_history`, `fx_rates` |
| `00004_users.sql` | `users`, `collection_items`, `user_trades` |

**`mydocs/tech/db_schema.md`** — 상세 정의, 인덱스, 규모 추정 절 추가
**`mydocs/working/task_m050_4_stage2.md`** — 단계별 완료 보고서

### 검증

- 모든 테이블 정의가 `poc_task4`에서 `CREATE` 되고, 대표 쿼리가 오류 없이 실행된다.
- 대표 쿼리의 실행 계획이 설계한 인덱스를 사용한다 (`EXPLAIN ANALYZE` 결과 첨부).
- 작업지시자 승인 후 `poc_task4`를 삭제한다.

---

## 3단계: 마이그레이션 작성과 검증

### 목표

승인된 설계대로 goose 마이그레이션을 작성하고, 적용·되돌리기·재적용이 모두 되는 것을 테스트로 확인한다.

### 구현

**`src/backend/migrations/00002_catalog.sql`, `00003_prices.sql`, `00004_users.sql`** — 신규. 각 파일에 `-- +goose Up`과 `-- +goose Down`을 두고, `Down`은 `Up`의 역순으로 객체를 삭제한다. 함수·`DO` 블록처럼 세미콜론이 여러 개인 구문은 `-- +goose StatementBegin/End`로 감싼다.

**`src/backend/internal/db/migrate_test.go`** — 신규. `TEST_DATABASE_URL`이 없으면 건너뛴다.

```go
// 빈 DB에서 전체 up → 버전 0까지 down → 다시 up 이 오류 없이 끝나는지 확인한다.
func TestMigrationsUpDownUp(t *testing.T)

// up 이후 주요 제약이 동작하는지 확인한다 (price_history 중복 삽입 거부, 음수 가격 거부 등).
func TestSchemaConstraints(t *testing.T)
```

**`Docker/` 변경** — 테스트 DB 생성 방법과 `TEST_DATABASE_URL` 주입 (`backend` 서비스 환경 변수). `.env.example`에 필요한 값이 있으면 추가한다.

**`mydocs/working/task_m050_4_stage3.md`** — 단계별 완료 보고서

### 검증

- 개발 DB: `docker compose restart backend` 후 로그에 `migration applied` 3건(00002~00004)이 찍히고 `/healthz`가 200.
- `go run ./cmd/migrate`로 상태 확인, `down` 3회 → `up` 수행 후 다시 버전 4.
- `go test ./internal/db/...` 통과 (테스트 DB 대상).
- `docker compose down -v` → `up -d --build`로 빈 DB에서 처음부터 적용되는지 재현성 확인.

---

## 4단계: sqlc 타입 설정·기본 쿼리·문서 정리

### 목표

금액 Go 타입을 확정해 sqlc 설정에 반영하고, #5·#6·#7이 바로 쓸 기본 쿼리와 생성 코드를 만든다. 문서와 CLAUDE.md를 실제 결과에 맞춘다.

### 구현

**금액 타입 확정.** 1단계 후보로 `price_history` 조회 쿼리의 생성 코드를 각각 만들어 보고(세션 임시 폴더), 사용 편의·JSON 직렬화·의존성 기준으로 하나를 고른다.

**`src/backend/sqlc.yaml`** — 선택한 타입의 `overrides` 추가 (`numeric`, 필요하면 `uuid`·`timestamptz`)

**`src/backend/internal/db/queries/*.sql`** — 신규. 수집기와 API의 출발점이 되는 최소 쿼리만 둔다.

| 파일 | 쿼리 |
|------|------|
| `catalog.sql` | 세트 upsert, 카드 upsert, 변형 upsert, 카드 단건 조회 |
| `prices.sql` | 외부 ID로 변형 찾기, 가격 upsert(일괄), 변형·소스별 기간 조회, 환율 upsert·기준일 조회 |
| `users.sql` | 사용자 생성·이메일 조회, 컬렉션 항목 upsert·목록, 사용자 거래가 등록 |

**`src/backend/internal/db/dbq/`** — `go tool sqlc generate`로 생성 (생성 코드는 커밋한다)

**`src/backend/internal/db/queries_test.go`** — 신규. 카드·변형 저장 → 가격 upsert 두 번(같은 키) → 1행만 존재하고 값이 갱신되는지, 기간 조회 결과가 날짜순인지 확인한다.

**문서**
- `mydocs/tech/db_schema.md`: 최종 ERD, 마이그레이션·쿼리 파일 목록, 금액 타입 결정, 후속 타스크(#5~#11) 인계 사항
- `mydocs/tech/backend_stack.md`: "금액 타입은 #4에서 정한다" 부분을 결정 결과로 갱신
- `CLAUDE.md` `빌드 및 실행`: DB 테스트 실행 방법(`TEST_DATABASE_URL`) 추가

**`mydocs/working/task_m050_4_stage4.md`** — 단계별 완료 보고서

### 검증

- `go tool sqlc generate` 후 `git diff`에 의도하지 않은 변경이 없다 (다시 생성해도 결과가 같음).
- `go build ./...`, `go vet ./...`, `go test ./...` 통과.

---

## 검증 (전체)

- `docker compose exec backend go build ./...` 통과
- `docker compose exec backend go vet ./...` 통과
- `docker compose exec backend go test ./...` 통과 (기존 테스트 + 마이그레이션·제약·쿼리 테스트)
- 빈 DB에서 `docker compose up -d --build` 후 마이그레이션 4개 적용, `/healthz` 200
- `migrate down`을 반복해 버전 0까지 되돌린 뒤 `up`으로 복구 가능
- `db_schema.md`의 요구 사항 추적표가 모두 반영 또는 사유 기재 상태

## 예상 단계: 4단계
