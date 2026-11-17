# Task #5: 카드 카탈로그 수집기 — 구현 계획서

수행 계획서: `mydocs/plans/task_m050_5.md`

1단계는 조사·검증 단계라 문서만 작성하고 저장소 소스는 바꾸지 않는다. 1단계 결과(연결 규칙, 변형 매핑, 종 이름 소스)를 작업지시자가 승인한 뒤 2~4단계에서 `src/backend/`를 변경한다.

## 공통 규칙

### 외부 호출

- 조사 단계의 호출도 수집기와 같은 예절을 지킨다: 동시성 1, 요청 간 최소 200ms, 전용 User-Agent(`PokeCardPrice-collector/0.1` 형식, 개인 정보 미포함).
- 조사용 스크립트와 원본 응답은 세션 임시 폴더에 두고, 결과 문서에는 측정값과 필요한 발췌만 남긴다.
- 확인 상태 표기는 #2와 같이 `응답 확인`, `문서 확인`, `미확인`, `확인 불가`를 쓴다.

### 테스트

- 외부 API에 의존하는 테스트는 만들지 않는다. 클라이언트는 `httptest.Server`와 저장한 고정 응답(`testdata/*.json`, 표본 몇 건만)으로 검증한다.
- DB 테스트는 기존 방식대로 `TEST_DATABASE_URL`(테스트 DB)에서 실행한다. 현재 테스트 DB 헬퍼(`testPool`)는 `internal/db` 패키지의 `_test.go` 안에 있어 다른 패키지에서 쓸 수 없으므로, 2단계에서 `internal/db/dbtest` 패키지로 옮겨 공유한다(기존 테스트 동작은 바꾸지 않는다).

### 결과 문서

**`mydocs/tech/catalog_collector.md`** (신규, `tech_investigation_template.md` 형식). 1단계에서 조사·검증 결과와 규칙을 쓰고, 2~4단계에서 실제 구현·실측값과 맞춰 갱신한다.

---

## 1단계: 사전 확인과 검증

### 목표

구현을 좌우하는 사실(이용 조건, 응답 필드, 호출 수, 한글판 일치 범위, 변형 매핑, 종 이름 소스)을 확인하고, 연결 규칙과 매핑 규칙의 선정안을 낸다. **이 단계 결과를 작업지시자가 승인하면 2단계부터 소스를 작성한다.**

### 구현

**TCGdex 이용 조건·호출 한도 재확인** (#2 미해결 사항). 공식 문서·저장소에서 라이선스, 데이터·이미지 이용 조건, 호출 한도 기재 여부를 다시 확인하고, 응답 헤더(`RateLimit-*`, `Cache-Control` 등)를 기록한다.

**응답 필드 확인.** 세 언어 각각에 대해 다음을 확인한다.

| 엔드포인트 | 확인 항목 |
|------------|-----------|
| `/v2/{lang}/sets` | 세트 수, 목록 항목 필드 (`id`, `name`, `cardCount`, `logo` 등) |
| `/v2/{lang}/sets/{id}` | `releaseDate`, `cardCount`, 카드 요약(`cards[]`)의 필드 (`id`, `localId`, `name`, `image`) |
| `/v2/{lang}/cards/{id}` | `rarity`, `variants`, `dexId`, `set` |
| `/v2/{lang}/cards?...` | 목록 조회에 `rarity`·`variants`·`dexId`가 포함되는지, 필터·페이지네이션 동작 |

- `cards` 테이블 컬럼 중 `rarity`, `dex_ids`와 `card_variants`는 카드 상세에만 있을 가능성이 높다. 이 경우 **카드 상세 호출 수**(언어별 카드 수 합계)와 요청 간격별 전체 소요 시간을 계산하고, 줄이는 방안을 비교한다.

| 방안 | 내용 |
|------|------|
| 전체 상세 조회 | 매 실행마다 모든 카드 상세 호출 |
| 신규·변경분만 상세 조회 | 세트 요약의 카드 수·카드 목록이 바뀐 세트, DB에 없는 카드만 상세 호출. 나머지는 주기적(예: 주 1회) 전체 갱신 |
| 목록 API 필드 활용 | 목록 응답에 필요한 필드가 있으면 상세 호출 생략 |

**한글판·일본판 일치 범위 검증.** TCGdex `ko` 세트 95개 전체의 세트 ID가 `ja`에 있는지 확인하고, 카드가 있는 `ko` 세트에서 카드별로 `ja`의 같은 세트 ID·`localId` 카드가 있는지, 이름(포켓몬 종)·`dexId`가 같은지 비교한다.

| 측정 | 의미 |
|------|------|
| `ko` 세트 중 같은 ID의 `ja` 세트가 있는 비율 | `has_ko_edition` 표시 대상 판단 근거 |
| `ko` 카드 239장 중 같은 세트 ID·번호의 `ja` 카드가 있는 비율, `dexId` 일치 비율 | `card_links` 자동 연결 규칙의 신뢰도 |
| 세트 ID는 같지만 카드 수가 다른 세트 | 번호 체계가 어긋날 수 있는 세트 목록 |

결과를 바탕으로 자동 연결 규칙(예: "세트 ID·`localId` 일치 + `dexId` 일치일 때만 `auto` 연결")과 `has_ko_edition` 표시 규칙(예: "같은 ID의 `ko` 세트가 있는 `ja` 세트의 카드 중, 번호가 `ko` 세트 `cardCount` 이하인 카드")을 제안한다.

**변형 매핑.** 세트 시대별 표본 카드(초판이 있는 구세트, 일반 세트, 최신 세트, 일본판)의 `variants` 값을 모아 `card_variants.variant` 7종(`normal`, `holofoil`, `reverse_holofoil`, `first_edition`, `first_edition_holofoil`, `unlimited`, `unlimited_holofoil`)으로 바꾸는 표를 만든다. `variants` 조합(예: `firstEdition`+`holo`)이 7종으로 표현되지 않는 경우와 `variants`가 비어 있는 카드의 처리 방안을 함께 적는다. #6이 TCGCSV `subTypeName`(`Normal`, `Holofoil`, `Reverse Holofoil`, `1st Edition Holofoil` 등)과 연결하기 쉬운 형태인지도 확인한다.

**종 이름 데이터 소스 비교.**

| 후보 | 확인 항목 |
|------|-----------|
| PokeAPI (REST, GraphQL, 원본 CSV 저장소) | 한·일·영 이름 제공 여부, 라이선스, 호출 규칙, 일괄 다운로드 가능 여부 |
| TCGdex 자체 데이터 | `ko`·`ja` 카드 이름에서 종 이름을 얻을 수 있는지 (범위가 좁을 것으로 예상) |
| 그 밖에 조사 중 발견한 후보 | 같은 기준 |

- 적재 방식(수집기가 매번 호출 / 저장소에 정적 데이터 파일로 포함해 마이그레이션 또는 시작 시 적재)도 함께 비교한다. 종 데이터는 거의 바뀌지 않는다.
- 적합한 소스가 없으면 선택지를 정리해 작업지시자 결정을 받는다.

**`ko` 미수록 한글판 카드 이름** (#2 미해결 사항). 해외 공개 소스로 얻을 수 있는지 가능 여부만 조사해 기록한다(구현은 범위 밖).

**`mydocs/working/task_m050_5_stage1.md`** — 단계별 완료 보고서. 결정이 필요한 사항은 질문 목록(Q1~)으로 정리한다.

### 검증

- 결정 사항(호출 방안, 연결 규칙, `has_ko_edition` 규칙, 변형 매핑, 종 이름 소스와 적재 방식)마다 측정 근거와 선정안이 있다.
- 측정값은 확인일·확인 상태와 함께 기록되어 있다.

---

## 2단계: TCGdex 클라이언트와 세트·카드·변형 적재 (en·ja)

### 목표

TCGdex 클라이언트를 만들고, 영어·일본어 세트·카드·변형을 DB에 멱등 적재한다.

### 구현

**`src/backend/internal/tcgdex/`** (신규) — TCGdex HTTP 클라이언트:

```go
type Client struct { /* http.Client, baseURL, userAgent, 요청 간격 제한기 */ }

func New(opts ...Option) *Client // WithBaseURL, WithHTTPClient, WithMinInterval, WithUserAgent
func (c *Client) ListSets(ctx context.Context, lang string) ([]SetBrief, error)
func (c *Client) GetSet(ctx context.Context, lang, id string) (*Set, error)
func (c *Client) GetCard(ctx context.Context, lang, id string) (*Card, error)
```

- 요청 간격은 클라이언트 전체에서 공유하는 제한기로 지킨다(표준 라이브러리 `time.Ticker` 또는 `golang.org/x/time/rate`. 의존성 추가가 필요하면 보고서에 근거를 남긴다).
- 재시도: 네트워크 오류·`429`·`5xx`만, 지수 백오프(예: 1s→2s→4s, 최대 3회), `Retry-After` 존중. `404`는 재시도하지 않고 구분 가능한 오류(`ErrNotFound`)로 돌려준다.
- 응답 타입은 1단계에서 확인한 필드만 정의하고, 없는 필드는 포인터로 받아 `nil`을 허용한다.

**`src/backend/internal/catalog/`** (신규) — 적재 로직:

```go
type Collector struct { /* tcgdex 클라이언트, pgxpool, 설정 */ }

type Options struct {
    Languages []string // 기본 en, ja, ko
    SetIDs    []string // 비우면 전체. 개발용 일부 적재
}

type Result struct { Sets, Cards, Variants int; FailedSets []string }

func (c *Collector) Run(ctx context.Context, opt Options) (Result, error)
func MapVariants(v tcgdex.Variants) []string // 1단계 매핑 표 구현
```

- 세트 단위로 트랜잭션을 연다: `UpsertSet` → 카드마다 `UpsertCard` → `UpsertCardVariant`. 세트 하나가 실패하면 그 세트만 롤백하고 `FailedSets`에 넣은 뒤 다음 세트로 넘어간다.
- 카드 상세 호출 범위는 1단계에서 승인한 방안을 따른다. "신규·변경분만" 방안이면 비교에 필요한 조회 쿼리(예: 세트별 저장된 카드 수·`tcgdex_id` 목록)를 `catalog.sql`에 추가한다.
- `ctx`가 취소되면 진행 중인 세트를 롤백하고 멈춘다(수집기 종료 시).
- 사라진 세트·카드는 지우지 않는다.

**`src/backend/internal/db/queries/catalog.sql`**, **`internal/db/dbq/`** — 필요한 조회 쿼리 추가, `go tool sqlc generate`로 생성 코드 갱신.

**`src/backend/internal/db/dbtest/`** (신규) — 기존 `testPool` 헬퍼를 옮겨 다른 패키지 테스트에서도 쓰게 한다. `internal/db`의 기존 테스트는 이를 호출하도록 바꾼다.

### 검증

- `tcgdex` 단위 테스트(`httptest`): 정상 응답 해석, `404` → `ErrNotFound`, `429`·`5xx` 재시도 후 성공/실패, 요청 간격 준수, User-Agent 헤더.
- `MapVariants` 표 기반 테스트: 1단계 매핑 표의 모든 행.
- `catalog` DB 테스트(고정 응답 서버 + 테스트 DB): 두 번 실행해도 행 수가 같다, 카드 이름이 바뀌면 갱신된다, 한 세트의 응답 오류가 다른 세트 적재를 막지 않는다.
- 실제 TCGdex로 표본 세트 2~3개(`SetIDs` 지정)를 개발 DB에 적재해 행 수를 확인한다.
- `go vet ./...`, `go test ./...` 통과.

---

## 3단계: 한글판 연결과 포켓몬 종 이름 적재

### 목표

`ko` 카드 적재와 일본판 연결(`card_links`), `has_ko_edition` 표시, `pokemon_species` 적재를 구현한다.

### 구현

**`src/backend/internal/catalog/`** — 한글판 처리:

- `ko` 세트·카드는 2단계와 같은 흐름으로 적재한다(`cards.language = 'ko'`).
- 연결 작업은 언어별 적재가 끝난 뒤 DB 기준으로 한 번에 실행한다(적재 순서에 의존하지 않도록).
  - `ko` 카드마다 1단계에서 승인한 규칙으로 `ja` 카드를 찾아 `card_links`(method `auto`)를 upsert한다. 연결 방향(`ko` → `ja`)은 코드에서 확인한다(DB 제약 없음, #4 인계).
  - 이미 `manual` 연결이 있는 카드는 덮어쓰지 않는다.
  - 규칙에 맞는 `ja` 카드가 없으면 연결하지 않고 건수만 로그로 남긴다(#7 수동 연결 대상).
  - `has_ko_edition`은 승인한 규칙으로 `ja` 카드에 설정하고, 규칙에서 벗어난 카드는 `false`로 되돌린다.
- 결과에 연결 수, 미연결 수, `has_ko_edition` 표시 수를 추가한다.

**`src/backend/internal/db/queries/catalog.sql`** — 추가 쿼리 (이름은 구현 시 확정):

| 쿼리 | 내용 |
|------|------|
| `UpsertAutoCardLink` | `ON CONFLICT (card_id) DO UPDATE ... WHERE card_links.method = 'auto'` (수동 연결 보호) |
| `ListUnlinkedKoCards` 등 | 연결 대상 조회 |
| `SetHasKoEdition` | 규칙에 따른 일괄 갱신 (집합 기반 `UPDATE`) |
| `UpsertPokemonSpecies` | `:batchexec` 또는 `:copyfrom` (1단계에서 정한 적재 방식에 맞춤) |

**종 이름 적재** — 1단계에서 승인한 소스·방식으로 구현한다. 수집기 작업이면 `internal/species/`(가칭)에, 정적 데이터면 데이터 파일과 적재 코드를 둔다. 출처·라이선스 표기가 필요하면 데이터 파일 옆에 함께 둔다.

### 검증

- DB 테스트: 일치하는 `ko`·`ja` 카드가 `auto`로 연결된다, 규칙에 맞지 않으면 연결되지 않는다, `manual` 연결은 바뀌지 않는다, 재실행해도 결과가 같다, `has_ko_edition`이 규칙대로 설정·해제된다.
- 종 이름 적재 테스트: 재실행 시 행 수 불변, 한글 이름으로 `pokemon_species` → `cards.dex_ids` 조회가 된다(예: `피카츄`).
- 실제 데이터로 `ko` 전체를 적재해 연결 수가 1단계 측정값과 일치하는지 확인한다.

---

## 4단계: 수집기 연결, 전체 적재 실측, 문서 정리

### 목표

카탈로그 수집을 수집기 일정에 연결하고, 전체 적재를 실제로 돌려 소요 시간·행 수를 기록한 뒤 문서를 정리한다.

### 구현

**`src/backend/cmd/collector/main.go`** — 작업 등록:

- 기존 자리 작업(`collect-prices`)을 카탈로그 수집 작업(`collect-catalog`)으로 바꾸거나 별도 작업으로 추가한다. #6의 시세 수집은 카드가 있어야 하므로, 같은 일정 안에서 **카탈로그 → (이후 #6) 시세** 순서로 실행되도록 구성한다.
- 개발용 필터를 환경 변수로 받는다(예: `CATALOG_LANGUAGES=en,ja`, `CATALOG_SETS=sv03.5,SV5K`). 이름은 구현 시 확정해 `CLAUDE.md`·`.env.example`에 반영한다.
- 실행 결과(세트·카드·변형·연결 수, 실패 세트, 소요 시간)를 `slog`로 남긴다. 실패 세트가 있으면 작업은 오류 없이 끝내되 `WARN`으로 기록한다. 전체가 실패하면 오류를 반환한다.

**전체 적재 실측.** 빈 개발 DB에서 `-once`로 전체 적재를 실행하고, 이어서 한 번 더 실행해(갱신 실행) 두 실행의 소요 시간, 호출 수, 언어별 행 수, 실패 세트를 기록한다.

**문서.**
- `mydocs/tech/catalog_collector.md`: 구현 결과와 실측값, #6·#7·#8 인계 사항.
- `mydocs/tech/price_data_sources.md` 미해결 사항 표: #5 담당 항목의 결과 반영.
- `mydocs/tech/db_schema.md` 10절 쿼리 표: 추가한 쿼리 반영.
- `CLAUDE.md` 빌드 및 실행: 카탈로그 수집 명령·환경 변수 추가.

### 검증

- `docker compose exec backend go build ./...`, `go vet ./...`, `go test ./...` 통과.
- `docker compose exec backend go run ./cmd/collector -once`로 전체 적재 성공, 재실행 시 행 수 불변.
- `COLLECTOR_INTERVAL`로 짧은 간격 실행 시 작업이 겹쳐 실행되지 않는다(기존 singleton 동작 확인).

---

## 검증 (전체)

- `docker compose exec backend go build ./...` 통과
- `docker compose exec backend go vet ./...`, `go test ./...` 통과 (DB 테스트 포함)
- 전체 적재 후 언어별 세트·카드·변형 수가 TCGdex 응답과 일치하고, 재실행해도 행이 늘지 않는다
- `ko` 카드의 `card_links` 연결 수와 `has_ko_edition` 표시 수가 1단계 검증 결과와 일치한다
- 한글 종 이름(`피카츄`)으로 일본판·영어판 카드를 찾을 수 있다

## 예상 단계: 4단계
