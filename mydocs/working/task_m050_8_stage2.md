# Task #8: 카드 검색·시세 조회 API — 2단계 완료 보고서

## 단계 목표

계약 오류 형식·매개변수 검증·금액/원화 변환·대표가 선택을 만들고, `GET /cards` 카드 검색을 구현한다. 1단계에서 정한 검색 인덱스를 마이그레이션으로 넣는다.

## 변경 파일

| 파일 | 내용 |
|------|------|
| `src/backend/migrations/00005_search_indexes.sql` | `cards_name_lower_trgm_idx`(`lower(name)` trigram), `cards_local_number_idx`(`ltrim(local_id,'0')`) 추가, `cards_name_trgm_idx` 삭제 (S1) |
| `src/backend/internal/db/queries/cards.sql` (신규) | `SearchCards`(1단계 확정 안), `ListLatestPrices`(카드들의 변형 × 판매가 소스 최신가 + 직전 환율) |
| `src/backend/internal/db/dbq/cards.sql.go` (생성) | sqlc 생성 코드 |
| `src/backend/internal/api/` (신규) | 응답 타입(`Price`·`CardSummary`·`Page`), `ToKRW`·`NewPrice`·`PickRepresentative`(Q1)와 단위 테스트 |
| `src/backend/internal/server/errors.go` (신규) | 계약 오류 형식 `{"error": {code, message}}`, 프레임워크 오류(404·405·기타 4xx·5xx) 변환. 500은 고정 문구로 응답하고 원인은 로그에만 남김 |
| `src/backend/internal/server/cards.go` (신규) | `GET /cards` 핸들러, 매개변수 검증, `CardSummary` 조립(`summaries`, 3·4단계에서 재사용) |
| `src/backend/internal/server/server.go` | `New(db Pinger, q *dbq.Queries)`, `/cards` 라우트. 오류 처리기를 `errors.go`로 옮김 |
| `src/backend/cmd/server/main.go` | 쿼리 객체 전달 |
| `src/backend/internal/server/server_test.go`, `cards_test.go` | 새 오류 형식에 맞춘 기존 테스트, 검증·검색 테스트 |

## 구현 내용

### 검색 (`GET /cards`)

- `q`: 앞뒤 공백을 뺀 1~100자(글자 수 기준), 소문자로 바꿔 공백으로 나눈 낱말 최대 10개. 어긋나면 `400 invalid_query`.
- `lang`(`en`·`ja`·`ko`), `page`(1~10000), `size`(1~60, 기본 24). 어긋나면 `400 invalid_param`. `page` 상한은 정수 넘침을 막으려고 두었다(계약 문서는 5단계에서 반영).
- 후보를 찾을 낱말은 **가장 긴 낱말**(글자 수)이다. 이름 패턴은 LIKE 특수 문자(`\`·`%`·`_`)를 이스케이프한다.
- 마지막 쪽을 넘으면(행 없음) 첫 행만 다시 읽어 `total`을 채운다. 결과가 없어도 `items`는 `[]`이다.
- 대표가: 검색 결과 카드 ID로 `ListLatestPrices`를 한 번 부르고 Go에서 고른다(Q1). 한글판은 `null`.

### 금액·원화 (`internal/api`)

- 금액은 `StringFixed(2)`(`"12.30"`), 원화는 `금액 × 환율`을 `Round(0)`(0.5는 0에서 먼 쪽)한 정수 문자열. KRW 소스는 그대로, 환율이 없으면 `krw`·`fxDate`가 `null`.
- `updatedAt`은 KST RFC 3339(`2026-11-24T05:05:58+09:00`). 실행 환경에 시간대 데이터가 없으면 고정 +09:00.

## 검증

| 항목 | 결과 |
|------|------|
| `go vet ./...`, `go build ./...` | 통과 |
| `go test ./...` (DB 테스트 포함, 건너뜀 없음) | 전체 통과 |
| 마이그레이션 | 개발 DB에 `00005` 적용 (285 ms). `internal/db` 마이그레이션 up/down 테스트 통과 |

**테스트 내용** (`cards_test.go`, 테스트 DB에 4개 세트·6장 고정 데이터)

- 검색 12건: 영문 이름(대소문자 무시), 한글 종 이름(4개 언어 카드, 발매일순), 언어 필터, **트레이너스가 종 이름 낱말에 맞지 않음**(1단계 결함 회귀 테스트), 부분 일치와 순위 ②>③, 세트 ID + 번호, 번호만, 번호 앞 0, 세트 이름(순위 ③), 번호 일치(①) 우선, `%`·`_`가 문자 그대로 검색됨.
- 쪽 넘김: 2쪽, 마지막 쪽을 넘은 경우의 `total`, 결과 없음에서 `items: []`.
- 응답 필드: 대표가(TCGCSV normal 최신가, 직전 환율 11-21로 원화 환산, `updatedAt` KST), TCGCSV가 없을 때 Cardmarket(EUR 환율 없음 → `krw: null`), 한글판 `price: null`, 트레이너스 `speciesNameKo: null`.
- 검증 오류 9건(`invalid_query` 4, `invalid_param` 5), 404·405 계약 형식, 500 응답에 원인 문자열이 드러나지 않음.
- 단위 테스트(`internal/api`): 원화 반올림 6건, `NewPrice` 필드, 대표가 우선순위 7건.

**개발 DB 실제 요청** (백엔드 재시작 후, 캐시가 찬 상태)

| 요청 | 서버 처리 시간(로그) | 호스트 `curl` |
|------|---------------------:|-------------:|
| `/healthz` | - | 약 8 ms |
| `/cards?q=pikachu` (24장 + 대표가) | 약 11 ms | 약 22 ms |

- 재시작 직후 첫 요청은 90~300 ms였다(캐시 없음, 1자 검색 `a` 298 ms). 100회 p50·p95 실측은 5단계에서 한다.
- 응답 예: `/cards?q=sv03.5+25` → `sv03.5-025` Pikachu 1건, `price: {"amount":"0.29","currency":"USD","source":"tcgcsv","date":"2026-11-23","krw":"388","fxDate":"2026-11-23",...}`.

## 참고

- `gofmt -l`이 기존 파일 다수를 표시하는데, 체크아웃 줄바꿈(CRLF) 때문이고 내용 변경은 없다. 이번에 만든 파일은 `gofmt`로 정리했다. `dbq/prices.sql.go`는 sqlc 재생성으로 줄바꿈만 바뀌어 `git status`에 나오지만 내용 차이는 없다.
- 프론트엔드는 아직 목업(`VITE_API_MOCK=true`)을 쓴다. 바뀐 오류 형식은 `client.ts`가 이미 읽는 형식이라 영향이 없다. `Price` 필드 추가는 5단계에서 `types.ts`에 반영한다.

## 다음 단계

3단계: `GET /cards/{id}` 카드 상세와 `GET /cards/{id}/prices` 시세 추이.
