# 기술 조사: 카드 검색·시세 조회 API 계약

> **작성일**: 2026-11-17 (초안, #12) / 2026-11-25 확정 (#8 1단계)
> **관련 타스크**: #12 (초안·목업), #8 (확정·구현), #13·#10 (사용 예정)

---

## 배경

#12(프론트엔드 홈·검색 화면)는 #8(카드 검색·시세 조회 API)보다 먼저 진행돼, 작업지시자 결정(#12 Q1)에 따라 API 형식을 먼저 정하고 화면은 이 계약대로 동작하는 목업으로 완성했다.
#8 1단계에서 이 계약을 확정했다. 초안이 #8로 미룬 항목(대표가·순위·정렬)을 정하고, 카드 상세·시세 추이 엔드포인트와 원화 환산 필드를 더했다 (#8 Q1~Q6, 2026-11-25 작업지시자 결정).
바꿀 필요가 있으면 이 문서와 `src/frontend/src/api/types.ts`를 같은 커밋에서 고친다.

### 초안 대비 바뀐 점 (#8)

| 항목 | 초안 (#12) | 확정 (#8) |
|------|-----------|-----------|
| `Currency` | `USD`·`JPY`·`KRW` | `USD`·`EUR`·`KRW` (Cardmarket은 EUR. 일본판도 소스 통화는 USD·EUR) |
| `Price` | `amount`·`currency`·`source`·`date` | `krw`·`fxDate`·`updatedAt` 추가 (Q2) |
| 엔드포인트 | 검색·순위 | 카드 상세 `GET /cards/{id}`, 시세 추이 `GET /cards/{id}/prices` 추가 |
| 오류 코드 | 400·500 | `404 not_found`, `405 method_not_allowed` 추가 |
| 대표가·순위·정렬 | #8에서 정함 | 2절에 규칙 명시 (Q1·Q3·Q4) |

## 조사 내용

### 1. 공통 규칙

| 항목 | 규칙 |
|------|------|
| 경로 | 백엔드 기준(`/cards`). 프론트엔드는 `/api/cards`로 부르고 Vite 프록시(개발)·배포 서버가 `/api`를 뗀다 |
| 형식 | JSON (UTF-8). 필드 이름은 camelCase. 값이 없으면 `null`(필드를 빼지 않는다), 목록이 없으면 `[]` |
| 금액 | **문자열** (`"12.34"`). 백엔드 `decimal` 정밀도를 지킨다. 화면은 표시할 때만 숫자로 바꾼다 |
| 원화 | 문자열, 원 단위 정수 (`"16602"`). 가격 기준일 이전 가장 가까운 Frankfurter 환율로 `금액 × 환율`을 원 단위 반올림(0.5는 올림). 환율이 없으면 `null` |
| 변동률 | 숫자. `0.125` = 12.5%, 음수는 하락. 계산할 수 없으면 `null` |
| 날짜 | `YYYY-MM-DD` 문자열. 시각(`updatedAt`)은 RFC 3339 (`2026-11-24T05:05:58+09:00`) |
| 기간 | `7d`·`30d`·`90d`·`1y`(=365일). 시작일 = 오늘(KST) − 기간 일수, 시작일 포함 |
| 쪽 넘김 | 요청 `page`(1부터)·`size`, 응답 `{"items", "page", "size", "total"}` |
| 오류 | HTTP 상태 + `{"error": {"code": "invalid_query", "message": "검색어는 1~100자여야 합니다"}}`. `message`는 화면에 그대로 보여 줄 수 있는 한국어 문장 |

오류 코드:

| HTTP | `code` | 상황 |
|------|--------|------|
| 400 | `invalid_query` | 검색어가 비었거나 너무 김 |
| 400 | `invalid_param` | `id`·`lang`·`page`·`size`·`period`·`direction`·`limit` 값이 잘못됨 |
| 404 | `not_found` | 카드가 없음, 없는 경로 |
| 405 | `method_not_allowed` | 허용하지 않는 메서드 |
| 500 | `internal` | 서버 오류 (원인은 서버 로그에만 남긴다) |

### 2. 엔드포인트

#### `GET /cards` — 카드 검색

| 매개변수 | 필수 | 값 | 기본 |
|----------|------|----|------|
| `q` | ✅ | 앞뒤 공백을 뺀 1~100자 | - |
| `lang` | | `en`·`ja`·`ko` | 전체 |
| `page` | | 1~10000 | 1 |
| `size` | | 1~60 | 24 |

- 검색어는 공백으로 나눈 낱말(최대 10개)로 찾고, **모든 낱말이 맞는 카드**만 돌려준다 (예: `sv03.5 25`, `피카츄 ex`). 대소문자는 구분하지 않는다.
- 낱말이 맞는 조건 (하나라도 맞으면 됨):
  - 카드 이름(영·일·한)에 낱말이 들어 있음 (부분 일치)
  - 세트 이름에 낱말이 들어 있음, 또는 세트 ID가 낱말과 같음
  - 카드 번호가 낱말과 같음. 앞의 0은 무시한다 (`25` = `025`)
  - 한글 종 이름(`pokemon_species.name_ko`)에 낱말이 들어 있고, 카드의 도감 번호(`cards.dex_ids`)에 그 종이 있음
- 정렬 (Q4): ① 세트 ID 또는 카드 번호가 낱말과 정확히 같은 카드 → ② 카드 이름이 검색어 전체로 시작하거나, 첫 도감 번호의 한글 종 이름이 낱말과 같은 카드 → ③ 나머지. 같은 순위 안에서는 세트 발매일 최신순 → 세트 ID → 카드 번호.
- 응답: `Page<CardSummary>`

#### `GET /cards/{id}` — 카드 상세

- `id`: `cards.id` (양의 정수). 아니면 `400 invalid_param`, 없으면 `404 not_found`.
- 응답: `CardDetail`. 변형마다 소스별 최신가를 담는다. 시세가 없는 변형은 `prices: []`.
- `linked`: 한글판 카드면 연결된 일본판, 일본판 카드면 연결된 한글판들 (`card_links`).

#### `GET /cards/{id}/prices` — 시세 추이

| 매개변수 | 필수 | 값 | 기본 |
|----------|------|----|------|
| `period` | | `7d`·`30d`·`90d`·`1y` | `30d` |

- 응답: `PriceHistoryResponse`. 변형 × 소스마다 계열(`series`) 하나. 기간 안에 가격이 없는 변형·소스는 계열을 만들지 않는다.
- 요약(`summary`, Q5): `current` = 기간 안 마지막 가격, `changeRate` = (마지막 − 첫) / 첫 (가격이 하나뿐이거나 첫 가격이 0이면 `null`), `average` = 날짜별 가격의 산술 평균(소수 둘째 자리 반올림), `min`·`max`, `from` = 기간 안 첫 가격의 날짜. 이력이 기간보다 짧으면 있는 만큼으로 계산하고 `from`이 요청 시작일보다 늦어진다.

#### `GET /rankings/movers` — 급등·급락 순위

| 매개변수 | 필수 | 값 | 기본 |
|----------|------|----|------|
| `period` | ✅ | `7d`·`30d` | - |
| `direction` | ✅ | `up`·`down` | - |
| `limit` | | 1~50 | 10 |

- 응답: `{"period", "direction", "asOf", "items": Mover[]}`. `up`은 변동률 큰 순(0보다 큰 것만), `down`은 작은 순(0보다 작은 것만, 가장 많이 내린 순).
- 산출 기준 (Q3):
  - 소스는 **TCGCSV(USD)**, 단위는 변형.
  - 기준일 `asOf` = TCGCSV 최신 가격 기준일. 비교일 = 기준일 − 7일(30일)과 같거나 그 이전 중 가장 가까운 TCGCSV 가격 기준일. 비교일은 순위 전체에 하나다 (1단계 측정 결과).
  - 기준일·비교일 가격이 **모두 $1 이상**인 변형만.
  - 같은 카드는 변동률이 가장 큰(하락은 가장 작은) 변형 1개만.
- 시세 데이터가 없으면 `asOf: null`, 비교일이 없으면 `asOf`는 기준일. 두 경우 모두 `items: []` (오류 아님).

#### 대표가 (`CardSummary.price`, Q1)

- 소스 우선순위: TCGCSV(USD) → Cardmarket(EUR).
- 변형 우선순위: `normal` → `holofoil` → `unlimited_holofoil` → `unlimited` → `reverse_holofoil` → `first_edition_holofoil` → `first_edition`.
- 소스 우선순위가 먼저다. 첫 소스에서 우선순위가 가장 높은 변형의 최신가를 쓰고, 그 소스에 가격이 하나도 없으면 다음 소스로 간다.
- 한글판 카드는 `null` (일본판 참고가는 #7).

### 3. 타입

프론트엔드 `src/frontend/src/api/types.ts`와 같다.

```ts
type Language = 'en' | 'ja' | 'ko'
type Currency = 'USD' | 'EUR' | 'KRW'
type Period = '7d' | '30d' | '90d' | '1y'

interface Price {
  amount: string                // "12.34" (소스 통화)
  currency: Currency
  source: string                // price_sources 코드 ("tcgcsv", "tcgdex_cardmarket")
  date: string                  // 가격 기준일 "2026-11-23"
  krw: string | null            // 원화 환산 "16602". 환율이 없으면 null
  fxDate: string | null         // 환산에 쓴 환율 날짜
  updatedAt: string | null      // 소스 갱신 시각 (오래된 가격 판단용)
}

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
  price: Price | null           // 대표가 (2절). 시세가 없으면 null
}

interface CardDetail extends Omit<CardSummary, 'set'> {
  rarity: string | null
  dexIds: number[]
  set: { tcgdexId: string; name: string; releaseDate: string | null; logoUrl: string | null; cardCount: number | null }
  linked: CardSummary[]
  variants: { variant: string; prices: Price[] }[]   // 변형 id 순, prices 는 소스 우선순위 순
}

interface PriceSeries {
  variant: string
  source: string
  currency: Currency
  points: { date: string; amount: string; krw: string | null }[]   // 날짜 오름차순
  summary: {
    current: Price
    average: string
    min: string
    max: string
    changeRate: number | null
    from: string
  }
}

interface PriceHistoryResponse {
  cardId: number
  period: Period
  from: string                  // 요청 시작일
  series: PriceSeries[]         // 변형 id 순 → 소스 우선순위 순
}

interface Page<T> { items: T[]; page: number; size: number; total: number }

interface Mover {
  card: CardSummary
  variant: string               // card_variants.variant
  current: Price                // 기준일 가격
  previous: Price               // 비교일 가격
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

### 4. 목업 (`src/frontend/src/api/mock/`)

- `.env.development`의 `VITE_API_MOCK=true`이면 `src/api/cards.ts`가 목업을 쓴다. #8 이후 기본값은 `false`(실제 API)이고, 백엔드 없이 화면을 볼 때만 `true`로 바꾼다(Vite 재시작 필요). 화면·스토어는 목업 여부를 모른다. 운영 빌드(`.env.development` 미적용)에서는 목업 코드가 번들에 들어가지 않는다.
- 목업은 검색·순위·카드 상세·시세 추이 네 함수를 흉내 낸다. 통화는 실제 API처럼 영어판 USD(`tcgcsv`), 일본판 EUR(`tcgdex_cardmarket`)이고 한글판 대표가는 `null`이다. 원화는 고정 환율(USD 1,400·EUR 1,500)로 만든다. 상세는 변형 하나·연결 없음, 추이는 계열 하나다.
- 응답 지연 300ms, 요청 취소(`AbortSignal`) 지원. 검색어 `__error`는 500 오류를 흉내 낸다.
- 상황 전환 (개발용): 주소에 `?mock=empty`를 붙이면 모든 목업 응답이 빈 결과, `?mock=error`면 500 오류가 된다. 빈 상태·오류 화면 확인용이다.
- 호출 수: `globalThis.__mockStats` (`{ search, movers, card, prices }`). 캐시 동작 확인용이다.
- 카드 정보는 개발 DB 표본, 가격·변동률은 카드 ID로 만든 임의 값이다(같은 카드는 늘 같은 값). 약 12%의 카드는 시세 없음(`price: null`)으로 둔다.

#### 목업 데이터

`src/api/mock/data.ts`는 개발 DB에서 아래 기준으로 뽑은 표본을 변환한 자동 생성 파일이다.

| 기준 | 내용 |
|------|------|
| 영어판 `sv03.5`(151) | 이상해씨·파이리·리자몽·꼬부기·피카츄·푸린·팬텀·이브이·잠만보·뮤츠·뮤 카드와 트레이너스 1장 |
| 영어판 `base1` | 리자몽·파이리·피카츄·뮤츠 |
| 일본판 | 리자몽·피카츄·팬텀·이브이·뮤츠·뮤 종별 이미지가 있는 최신 2장, 한글판 표본(각 세트 1~4번)과 연결된 일본판 카드 |
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
    WHERE c.language = 'ja' AND c.image_url IS NOT NULL AND c.dex_ids[1] IN (6,25,94,133,150,151)
  ) j WHERE rn <= 2
  UNION
  SELECT id FROM (
    SELECT c.id, row_number() OVER (PARTITION BY s.tcgdex_id ORDER BY c.local_id) rn
    FROM cards c JOIN sets s ON s.id = c.set_id WHERE c.language = 'ko'
  ) k WHERE rn <= 4
  UNION
  SELECT l.linked_card_id FROM card_links l JOIN cards c ON c.id = l.card_id
  WHERE c.local_id IN ('001', '002', '003', '004')
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

1. 검색(`GET /cards`), 카드 상세(`GET /cards/{id}`), 시세 추이(`GET /cards/{id}/prices`), 급등·급락(`GET /rankings/movers`) 네 엔드포인트를 둔다.
2. 금액은 소스 통화 문자열에 원화 환산(`krw`)을 함께 보내고, 변동률은 숫자, 오류는 `{"error": {code, message}}`로 통일한다.
3. 대표가는 TCGCSV → Cardmarket, 변형 우선순위로 고르고, 순위는 TCGCSV 변형 단위·$1 이상·카드당 1개로 낸다.

## 적용 방향

- #8: 백엔드가 이 계약대로 응답한다. `types.ts`·목업을 이 계약에 맞췄고, `.env.development`의 `VITE_API_MOCK`을 `false`로 바꿔 실제 API로 전환했다. 프론트엔드 API 함수: `searchCards`, `getMovers`, `getCard`, `getCardPrices` (`src/api/cards.ts`).
- #13: 카드 상세 화면은 `CardDetail`·`PriceHistoryResponse`를 쓴다.
- #7: 한글판 카드의 대표가(현재 `null`)와 일본판 참고가 소스를 더할 때 대표가 규칙(2절)을 갱신한다.

## 참고 자료

- `mydocs/tech/db_schema.md` (#4, 7절 대표 쿼리)
- `mydocs/tech/catalog_collector.md` (#5, `has_ko_edition`·한글판 연결 규칙)
- `mydocs/tech/price_collector.md` (#6, 소스 기준일·변형별 소스 차이)
- `mydocs/plans/task_m100_12.md`, `task_m050_8.md`, `mydocs/working/task_m050_8_stage1.md`
