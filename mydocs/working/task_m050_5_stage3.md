# Task #5 단계3 완료 보고서: 한글판 연결과 포켓몬 종 이름 적재

## 수행 내용

- **`ko` 적재**: 2단계 흐름을 그대로 쓴다 (기본 수집 언어에 `ko` 포함). 별도 코드는 없다.
- **한글판 자동 연결 (Q2)**: 언어별 적재가 끝난 뒤 DB 전체를 기준으로 한 트랜잭션에서 실행한다 (적재 순서와 무관).
  - `UpsertAutoCardLinks`: `ko` 카드와 세트 ID·`localId`가 같고 `dexId`가 같은(둘 다 없으면 같음) `ja` 카드를 `auto`로 연결한다. 연결 방향은 항상 `ko` → `ja`이다. `manual` 연결은 덮어쓰지 않는다.
  - `DeleteStaleAutoCardLinks`: 규칙에 더는 맞지 않는 `auto` 연결을 지운다 (계획서에 없던 항목, 아래 발견 사항 참고). `manual` 연결은 지우지 않는다.
  - 연결되지 않은 `ko` 카드는 건수와 앞 20장의 ID를 `WARN` 로그로 남긴다 (#7 수동 연결 대상).
- **`has_ko_edition` 표시 (Q4)**: `SetHasKoEdition`이 같은 ID의 `ko` 세트가 있는 `ja` 세트에서 번호(숫자) ≤ `ko` 세트 카드 수인 카드만 참으로, 나머지 `ja` 카드는 거짓으로 맞춘다. 값이 바뀌는 행만 갱신한다. 숫자가 아닌 번호는 거짓이다.
- **종 이름 (Q5)**: `internal/species`에 PokeAPI CSV에서 도감 번호·영어·일본어(가나)·한국어 이름만 뽑은 CSV(1,025행)와 라이선스 문구를 두고 `embed`했다. 카탈로그 작업 시작 시 `UpsertPokemonSpecies`로 한 번에 upsert한다 (배열 `unnest`, 값이 같으면 갱신하지 않음).
- **결과 집계**: `Result`에 `SpeciesChanged`와 `Links`(변경·삭제 연결 수, `ko` 카드 수, `auto`·`manual`·미연결 수, `has_ko_edition` 수와 변경 수)를 추가해 로그에 남긴다.
- **조회 쿼리**: `ListCardsByKoSpeciesName` (한글 종 이름 → 카드). 3단계 검증용이며 #8 검색의 출발점이다.

## 변경 파일

| 파일 | 변경 내용 |
|------|----------|
| `src/backend/internal/catalog/collector.go` | 종 이름 적재, 한글판 연결·`has_ko_edition`, 결과 집계 추가 |
| `src/backend/internal/catalog/collector_test.go` | 한글판 연결·수동 연결 보호·오래된 연결 삭제·종 이름 테스트 4개 추가 |
| `src/backend/internal/species/species.go` | 신규. 내장 종 이름 데이터 해석 |
| `src/backend/internal/species/species_test.go` | 신규. 1,025종 연속성·이름 누락 확인 |
| `src/backend/internal/species/pokemon_species_names.csv` | 신규. PokeAPI 커밋 `8d8a49d`에서 추출 (1,025행) |
| `src/backend/internal/species/LICENSE.pokeapi.md` | 신규. 출처·추출 방법·PokeAPI 라이선스 전문 |
| `src/backend/internal/db/queries/catalog.sql` | 쿼리 7개 추가 |
| `src/backend/internal/db/dbq/catalog.sql.go` | sqlc 생성 코드 |
| `mydocs/tech/catalog_collector.md` | 3단계 구현 결과 추가 |
| `mydocs/orders/20261117.md` | 오늘 할일 갱신 |

## 실데이터 확인 (개발 DB, 2026-11-17)

`ko` 전체를 적재한 뒤, 카드가 있는 `ko` 세트(`SV4K`·`SV4M`·`SV5K`)와 같은 ID의 `ja` 세트를 적재했다. `ja` 전체 적재는 4단계 실측에서 한다.

| 항목 | 결과 | 1단계 측정 |
|------|------|-----------|
| `ko` 세트 | 저장 91, 건너뜀 4 (세트 상세 404) | 91 / 95 일치, 4개 404 |
| `ko` 카드 | 239 (`SV4K` 95, `SV4M` 95, `SV5K` 49) | 239 |
| `auto` 연결 | **239 / 239**, 미연결 0 | 239 / 239 |
| `has_ko_edition` | 261 (`SV4K` 95/95, `SV4M` 95/95, `SV5K` 71/100) | - |
| 종 이름 | 1,025행, 한·일·영 모두 채워짐 | 1,025 |
| `ko` 전체 적재 소요 | 1분 25초 (세트 상세 95회 + 카드 상세 239회) | - |
| 재실행 | 연결 변경 0, `has_ko_edition` 변경 0, 종 변경 0 | - |

- `SV5K`는 `ko` 세트 `cardCount.total`이 71인데 TCGdex에 수록된 `ko` 카드는 49장이다. Q4 규칙대로 `ja` `SV5K`의 1~71번에 표시되었다 (수록 카드 수가 아니라 세트 카드 수 기준).
- 첫 `ko` 적재 시점에는 `ja` `SV4K`만 있어 95장만 연결되고 144장이 미연결 로그에 남았다. `ja` 세트를 적재한 다음 실행에서 144장이 연결되어, 적재 순서와 관계없이 맞춰지는 것을 확인했다.
- 한글 종 이름 조회: `피카츄` → 영어판 3장(현재 적재 범위). `아보크` → `en`·`ja`·`ko` 5장으로 세 언어 모두 찾아진다.

## 발견 사항

- **오래된 `auto` 연결 삭제를 추가했다.** 계획서에는 upsert만 있었으나, `ja` 카드의 도감 번호가 TCGdex에서 수정되면 규칙에 맞지 않는 `auto` 연결이 계속 남는다. `has_ko_edition`을 "규칙에서 벗어나면 되돌린다"로 정한 것과 같은 원칙으로, `auto` 연결도 매 실행 규칙대로 맞춘다. `manual` 연결은 건드리지 않는다.
- sqlc가 여러 인자 `unnest(a, b, ...)`를 해석하지 못해, 선택 목록에 열마다 `unnest`를 두는 형태로 썼다 (PostgreSQL 10 이상에서 같은 길이 배열은 행 단위로 묶인다).
- 종 이름 CSV의 PokeAPI 원본에는 따옴표·쉼표가 들어간 이름이 없어(`Mr. Mime`, `Nidoran♀` 등) 단순 CSV로 충분하다. 1~1,025번 모두 세 언어 이름이 있다.

## 검증

- `go build ./...`, `go vet ./...`, `gofmt`: ✅
- `go test ./...` (DB 테스트 포함): ✅
  - `catalog` 신규 4개: `auto` 연결·`dexId` 불일치 미연결·도감 번호 없음끼리 연결, `has_ko_edition` 설정과 범위 축소 시 해제, 재실행 시 변경 0, `manual` 연결 유지, 규칙에서 벗어난 `auto` 연결 삭제, 종 1,025행 적재·재실행 변경 0, 한글 이름(`이상해씨`)으로 `ja`·`ko` 카드 조회
  - `species` 1개: 1,025종 연속, 이름 누락 없음, `피카츄`
  - 기존 테스트 전부 통과
- 실데이터: `ko` 239장 모두 `auto` 연결 (1단계 측정과 일치), 재실행 시 결과 불변: ✅
- 확인용 임시 명령(`cmd/tmp_catalogtry`)은 확인 후 삭제: ✅

## 다음 단계

4단계: 수집기 일정에 카탈로그 작업 연결(매일 + 월 1회 전체 갱신), 개발용 환경 변수, 빈 DB 전체 적재 실측, 문서 정리.

## 커밋

(승인 후 커밋)
