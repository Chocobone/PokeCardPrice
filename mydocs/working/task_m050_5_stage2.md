# Task #5 단계2 완료 보고서: TCGdex 클라이언트와 세트·카드·변형 적재 (en·ja)

## 수행 내용

- `internal/tcgdex`: TCGdex REST 클라이언트를 만들었다 (`ListSets`, `GetSet`, `GetCard`).
  - 요청 간 최소 간격(기본 250ms)은 클라이언트 전체가 함께 지킨다. 표준 라이브러리만 써서 의존성은 추가하지 않았다.
  - 네트워크 오류·`429`·`5xx`만 재시도한다 (최대 3회, 1s→2s→4s, `Retry-After`는 초 단위만 따르고 최대 30초). `404`는 `ErrNotFound`.
  - 전용 User-Agent `PokeCardPrice-collector/0.1`을 보낸다. 카드 ID는 경로 인코딩한다 (`exu-%3F` 같은 ID 대응).
- `internal/catalog`: 적재 로직(`Collector.Run`)과 변형 매핑(`MapVariants`)을 만들었다.
  - Q1 방안을 따른다: 세트 상세를 매번 받고, DB에 없는 카드만 카드 상세를 받는다. `Options.Full`이면 모든 카드 상세를 다시 받는다 (월 1회 전체 갱신용). 카드 상세는 동시성 2로 받는다.
  - 기존 카드는 세트 요약의 번호·이름·이미지만 갱신한다 (`UpdateCardBrief`). 희귀도·도감 번호는 그대로 둔다.
  - 외부 호출을 마친 뒤 세트 하나를 트랜잭션 하나로 저장한다. 세트가 실패하면 그 세트만 롤백하고 `FailedSets`에 넣는다. 모든 세트가 실패하거나 모든 언어의 세트 목록을 받지 못하면 오류를 돌려준다.
  - 세트 상세·신규 카드 상세가 `404`면 건너뛰고 건수를 센다. 다음 실행에서 다시 시도된다.
  - `ctx` 취소 시 진행 중인 세트를 롤백하고 멈춘다. 사라진 세트·카드는 지우지 않는다.
- `internal/db/queries/catalog.sql`: `ListCardTcgdexIDsBySet`, `UpdateCardBrief` 추가, sqlc 생성 코드 갱신.
- `internal/db/dbtest`: 테스트 DB 헬퍼를 옮겼다. 패키지별로 `suffix`를 주면 별도 DB(`{테스트 DB}_{suffix}`)를 써서, `go test ./...`가 패키지를 동시에 실행해도 서로의 스키마를 지우지 않는다. `internal/db` 테스트는 기존과 같은 DB를 쓴다.
- 실제 TCGdex로 표본 세트 4개를 개발 DB에 적재해 확인했다. 확인용 임시 명령(`cmd/tmp_catalogtry`)은 확인 후 지웠다.

## 변경 파일

| 파일 | 변경 내용 |
|------|----------|
| `src/backend/internal/tcgdex/client.go` | 신규. HTTP 클라이언트, 요청 간격 제한, 재시도 |
| `src/backend/internal/tcgdex/types.go` | 신규. 세트 목록·세트 상세·카드 상세 응답 타입 |
| `src/backend/internal/tcgdex/client_test.go` | 신규. `httptest` 단위 테스트 9개 |
| `src/backend/internal/tcgdex/testdata/*.json` | 신규. 고정 응답 3개 (`sv03.5` 세트, `sv03.5-001`, `SV4K-001`) |
| `src/backend/internal/catalog/collector.go` | 신규. 적재 로직 |
| `src/backend/internal/catalog/variants.go` | 신규. 변형 매핑 (Q3) |
| `src/backend/internal/catalog/collector_test.go` | 신규. 고정 응답 서버 + 테스트 DB 테스트 8개 |
| `src/backend/internal/catalog/variants_test.go` | 신규. 매핑 표 테스트 |
| `src/backend/internal/db/dbtest/dbtest.go` | 신규. 테스트 DB 헬퍼 (`internal/db/testdb_test.go`에서 옮김) |
| `src/backend/internal/db/testdb_test.go` | `dbtest.Pool` 호출로 변경 |
| `src/backend/internal/db/queries/catalog.sql` | 쿼리 2개 추가 |
| `src/backend/internal/db/dbq/catalog.sql.go` | sqlc 생성 코드 |
| `mydocs/tech/catalog_collector.md` | 2단계 구현 결과 절 추가 |
| `mydocs/orders/20261117.md` | 오늘 할일 |

`dbq/` 의 나머지 생성 파일(`batch.go`, `db.go` 등)은 내용 변경 없이 줄바꿈(LF/CRLF)만 다르게 표시되며, 커밋 시 변경으로 잡히지 않는다.

## 표본 적재 결과 (개발 DB, 2026-11-17)

| 언어 | 세트 | `cardCount.total` | 카드 | 변형 | 비고 |
|------|------|------:|-----:|-----:|------|
| `en` | `base2` (Jungle) | 64 | 64 | 130 | 초판·무제한판 분리 확인 (`first_edition` 48, `unlimited` 48, `first_edition_holofoil` 16, `unlimited_holofoil` 16, `reverse_holofoil` 2) |
| `en` | `exu` | 28 | 28 | 28 | `exu-%3F` 등 특수 문자 ID 정상 |
| `en` | `sv03.5` (151) | 207 | 207 | 362 | |
| `ja` | `SV4K` | 95 | 95 | 95 | |

- 이전 세션에서 첫 적재를 해 둔 상태였고, 오늘 재실행 결과는 세트 4·신규 카드 0·요약 변경 0·실패 0(2.5초)으로 행이 늘지 않았다 (멱등 확인). 첫 적재의 소요 시간은 기록이 남아 있지 않아, 4단계 전체 적재 실측에서 빈 DB 기준으로 기록한다.

## 발견 사항

- **`ja` `SV4K`의 67번 이후 카드 29장은 TCGdex에 희귀도가 없다** (`rarity` `NULL`). TCGdex 데이터 공백으로, 적재는 정상이다. 월 1회 전체 갱신 때 채워지면 반영된다.
- `dex_ids`가 `NULL`인 카드(트레이너스·에너지)는 `en` 23장, `ja` 31장이다. `ja` 31장은 희귀도 없는 29장과 겹치는 카드가 있어, 3단계 `dexId` 비교 시 "둘 다 없으면 같다고 봄"(Q2) 규칙이 적용된다.
- `UpsertSet`은 값이 같아도 `updated_at`을 갱신한다 (#4 쿼리). 세트 수가 적어 그대로 두었다.

## 검증

- `go build ./...`, `go vet ./...`: ✅
- `go test ./...` (DB 테스트 포함, `-count=1 -v`로 건너뜀 없이 실행 확인): ✅
  - `tcgdex` 9개: 정상 응답 해석, 카드 ID 경로 인코딩, `404` → `ErrNotFound`(재시도 없음), `5xx` 재시도 후 성공, 재시도 한도 초과 실패, `4xx` 재시도 안 함, User-Agent, 고루틴 간 요청 간격 공유, 취소 시 대기 중단
  - `catalog` 9개: 두 번 실행해도 행 수 불변, 요약 갱신 시 상세 필드 유지, 전체 갱신 시 상세 필드 갱신, 한 세트 오류가 다른 세트를 막지 않음, 전체 실패 시 오류, `404` 세트·카드 건너뜀, 카드 상세 오류 시 세트 롤백, 취소 시 중단, `MapVariants` 매핑 표 전 행
  - `internal/db` 기존 6개 통과 (헬퍼 이동 후 동작 동일)
- `go tool sqlc generate` 재실행 후 생성 코드 차이 없음: ✅
- 실제 TCGdex 표본 적재: 세트별 카드 수가 `cardCount.total`과 일치, 재실행 시 행 불변: ✅

## 다음 단계

3단계: `ko` 세트·카드 적재, `ko`→`ja` 자동 연결(`card_links`, Q2), `has_ko_edition` 표시(Q4), PokeAPI 종 이름 `embed` 적재(Q5).

## 커밋

(승인 후 커밋)
