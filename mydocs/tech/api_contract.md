# 기술 조사: 카드 검색·순위 API 계약 (초안)

> **작성일**: 2026-11-17
> **관련 타스크**: #12 (작성), #8 (구현)

---

## 배경

#12(프론트엔드 홈·검색 화면)는 #8(카드 검색·시세 조회 API)에 의존하지만 #8은 아직 착수 전이다. 작업지시자 결정(#12 Q1)에 따라 프론트엔드가 쓸 API 형식을 먼저 정하고, 화면은 이 계약대로 동작하는 목업으로 완성한다.
#8은 이 계약을 수행 계획서의 입력으로 삼아 구현한다. 바꿀 필요가 있으면 #8에서 이 문서를 고치고, 프론트엔드는 `src/api/` 한 곳만 맞춘다.

## 조사 내용

### 1. 공통 규칙

| 항목 | 규칙 |
|------|------|
| 경로 | 백엔드 기준(`/cards`). 프론트엔드는 `/api/cards`로 부르고 Vite 프록시(개발)·배포 서버가 `/api`를 뗀다 |
| 형식 | JSON (UTF-8). 필드 이름은 camelCase |
| 금액 | **문자열** (`"12.34"`). 백엔드 `decimal` 정밀도를 지킨다. 화면은 표시할 때만 숫자로 바꾼다 |
| 변동률 | 숫자. `0.125` = 12.5%, 음수는 하락 |
| 날짜 | `YYYY-MM-DD` 문자열 |
| 쪽 넘김 | 요청 `page`(1부터)·`size`, 응답 `{"items", "page", "size", "total"}` |
| 오류 | HTTP 상태 + `{"error": {"code": "invalid_query", "message": "검색어는 1~100자여야 합니다"}}`. `message`는 화면에 그대로 보여 줄 수 있는 한국어 문장 |

오류 코드 (초안):

| HTTP | `code` | 상황 |
|------|--------|------|
| 400 | `invalid_query` | 검색어가 비었거나 너무 김 |
| 400 | `invalid_param` | `lang`·`page`·`size`·`period`·`direction`·`limit` 값이 잘못됨 |
| 500 | `internal` | 서버 오류 |

### 2. 엔드포인트

#### `GET /cards` — 카드 검색

| 매개변수 | 필수 | 값 | 기본 |
|----------|------|----|------|
| `q` | ✅ | 1~100자 | - |
| `lang` | | `en`·`ja`·`ko` | 전체 |
| `page` | | 1 이상 | 1 |
| `size` | | 1~60 | 24 |

- 검색 대상: 카드 이름(영·일·한), 한글 종 이름(`pokemon_species.name_ko` → `cards.dex_ids`), 세트 이름, 세트 ID, 카드 번호. 공백으로 나눈 낱말이 모두 맞는 카드를 찾는다 (예: `sv03.5 25`, `피카츄 ex`).
- 번호는 앞의 0을 무시하고 비교한다 (`25` = `025`).
- 정렬·부분 일치 방식·순위(관련도)는 #8에서 정한다.
- 응답: `Page<CardSummary>`

#### `GET /rankings/movers` — 급등·급락 순위

| 매개변수 | 필수 | 값 | 기본 |
|----------|------|----|------|
| `period` | ✅ | `7d`·`30d` | - |
| `direction` | ✅ | `up`·`down` | - |
| `limit` | | 1~50 | 10 |

- 응답: `{"period", "direction", "asOf", "items": Mover[]}`. `up`은 변동률 큰 순, `down`은 작은 순(가장 많이 내린 순).
- 시세 데이터가 없으면 `asOf: null`, `items: []` (오류 아님).
- 어느 소스·변형·통화로 순위를 낼지는 #8에서 정한다 (#4 `db_schema.md` 7절 S3 참고).

### 3. 타입

프론트엔드 `src/frontend/src/api/types.ts`와 같다.

```ts
type Language = 'en' | 'ja' | 'ko'
type Currency = 'USD' | 'JPY' | 'KRW'

interface CardSummary {
  id: number                    // cards.id
  language: Language
  tcgdexId: string              // "sv03.5-025"
  name: string
  localId: string               // "025"
  set: { tcgdexId: string; name: string }
  imageUrl: string | null       // TCGdex 기본 주소. 화면이 "/low.webp"·"/high.webp" 를 붙인다
  hasKoEdition: boolean         // 일본판 카드에 한글판이 있는지 (#5 규칙)
  speciesNameKo: string | null  // 첫 도감 번호의 한글 종 이름. 트레이너스·에너지는 null
  price: Price | null           // 대표가. 시세가 없으면 null
}

interface Price {
  amount: string                // "12.34"
  currency: Currency
  source: string                // price_sources 코드 (예: "tcgcsv")
  date: string                  // "2026-11-16"
}

interface Page<T> { items: T[]; page: number; size: number; total: number }

interface Mover {
  card: CardSummary
  variant: string               // card_variants.variant
  current: Price
  previous: Price               // 기간 시작 시점 가격
  changeRate: number            // (current - previous) / previous
}

interface MoversResponse {
  period: '7d' | '30d'
  direction: 'up' | 'down'
  asOf: string | null           // 순위 기준일
  items: Mover[]
}
```

- `CardSummary`는 #13(카드 상세)·#10(내 컬렉션)에서도 목록 항목으로 다시 쓸 수 있도록 카드 하나를 알아보는 데 필요한 최소 정보만 담는다.
- 대표가(`price`)를 어느 소스·변형·통화로 고를지는 #8에서 정한다. 원화 환산값을 따로 둘지도 #8에서 정한다 (지금은 원래 통화 그대로).

### 4. 목업 (`src/frontend/src/api/mock/`)

- `.env.development`의 `VITE_API_MOCK=true`이면 `src/api/cards.ts`가 목업을 쓴다. 화면·스토어는 목업 여부를 모른다. 운영 빌드(`.env.development` 미적용)에서는 목업 코드가 번들에 들어가지 않는다.
- 응답 지연 300ms, 요청 취소(`AbortSignal`) 지원. 검색어 `__error`는 500 오류를 흉내 낸다.
- 카드 정보는 개발 DB 표본, 가격·변동률은 카드 ID로 만든 임의 값이다(같은 카드는 늘 같은 값). 약 12%의 카드는 시세 없음(`price: null`)으로 둔다.

#### 목업 데이터

`src/api/mock/data.ts`는 개발 DB에서 아래 기준으로 뽑은 표본을 변환한 자동 생성 파일이다.

| 기준 | 내용 |
|------|------|
| 영어판 `sv03.5`(151) | 이상해씨·파이리·리자몽·꼬부기·피카츄·푸린·팬텀·이브이·잠만보·뮤츠·뮤 카드와 트레이너스 1장 |
| 영어판 `base1` | 리자몽·파이리·피카츄·뮤츠 |
| 일본판 | 리자몽·피카츄·팬텀·이브이·뮤츠·뮤 종별 최신 2장 |
| 한글판 | 세트별 앞 4장 |

카드 ID(`cards.id`)는 개발 DB를 다시 적재하면 바뀌므로 목업 안에서만 의미가 있다.

추출 쿼리 (개발 DB, 읽기 전용). 결과 JSON 배열의 각 항목을 `MOCK_CARDS`의 원소로 옮긴다.

```sql
-- #12 목업 표본 추출 (개발 DB, 읽기 전용). 결과는 CardSummary 형식 JSON 배열.
WITH pick AS (
  SELECT c.id FROM cards c JOIN sets s ON s.id = c.set_id
  WHERE s.language = 'en' AND s.tcgdex_id = 'sv03.5'
    AND (c.dex_ids[1] IN (1,4,6,7,25,39,94,133,143,150,151) OR c.local_id IN ('166','196'))
  UNION
  SELECT c.id FROM cards c JOIN sets s ON s.id = c.set_id
  WHERE s.language = 'en' AND s.tcgdex_id = 'base1' AND c.dex_ids[1] IN (4,6,25,150,151)
  UNION
  SELECT id FROM (
    SELECT c.id, row_number() OVER (PARTITION BY c.dex_ids[1] ORDER BY s.release_date DESC NULLS LAST, c.local_id) rn
    FROM cards c JOIN sets s ON s.id = c.set_id
    WHERE c.language = 'ja' AND c.dex_ids[1] IN (6,25,94,133,150,151)
  ) j WHERE rn <= 2
  UNION
  SELECT id FROM (
    SELECT c.id, row_number() OVER (PARTITION BY s.tcgdex_id ORDER BY c.local_id) rn
    FROM cards c JOIN sets s ON s.id = c.set_id WHERE c.language = 'ko'
  ) k WHERE rn <= 4
)
SELECT coalesce(json_agg(json_build_object(
  'id', c.id, 'language', c.language, 'tcgdexId', c.tcgdex_id, 'name', c.name, 'localId', c.local_id,
  'set', json_build_object('tcgdexId', s.tcgdex_id, 'name', s.name),
  'imageUrl', c.image_url, 'hasKoEdition', c.has_ko_edition, 'speciesNameKo', p.name_ko
) ORDER BY c.language, s.release_date DESC NULLS LAST, c.local_id), '[]')
FROM pick JOIN cards c ON c.id = pick.id JOIN sets s ON s.id = c.set_id
LEFT JOIN pokemon_species p ON p.dex_id = c.dex_ids[1];
```

## 결론

1. 검색(`GET /cards`)과 급등·급락(`GET /rankings/movers`) 두 엔드포인트로 #12 화면을 만든다.
2. 금액은 문자열, 변동률은 숫자, 오류는 `{"error": {code, message}}`로 통일한다.
3. 대표가·순위 산출 기준과 검색 정렬은 #8이 정하고, 계약은 담는 형식만 정한다.

## 적용 방향

- #12: `src/api/types.ts`·`client.ts`·`cards.ts`와 목업이 이 계약을 따른다.
- #8: 수행 계획서에서 이 계약을 검토하고, 바뀌는 부분은 이 문서와 `types.ts`를 함께 고친다. 구현 후 `.env.development`의 `VITE_API_MOCK`을 `false`로 바꿔 실제 API로 전환한다.

## 참고 자료

- `mydocs/tech/db_schema.md` (#4, 7절 대표 쿼리 S1~S4)
- `mydocs/tech/catalog_collector.md` (#5, `has_ko_edition`·한글판 연결 규칙)
- `mydocs/plans/task_m100_12.md`, `task_m100_12_impl.md`
