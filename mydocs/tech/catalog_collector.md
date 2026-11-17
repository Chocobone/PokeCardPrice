# 기술 조사: 카드 카탈로그 수집기

> **작성일**: 2026-11-10
> **관련 타스크**: [#5](https://github.com/Chocobone/PokeCardPrice/issues/5) 카드 카탈로그 수집기
> **상태**: 1단계(사전 확인과 검증) 승인, Q1~Q5 제안대로 결정 (2026-11-10). 2~4단계에서 구현 결과와 실측값을 추가한다.

---

## 배경

#5는 TCGdex에서 영어(`en`)·일본어(`ja`)·한국어(`ko`) 세트와 카드를 받아 #4의 카탈로그 테이블에 적재한다.
구현 전에 다음을 확인해야 했다 (#2·#4 인계 사항).

- TCGdex 이용 조건과 호출 한도
- 테이블 컬럼을 채우려면 어떤 엔드포인트를 몇 번 호출해야 하는지
- 한글판 세트 코드·카드 번호가 일본판과 어디까지 같은지 (`card_links`, `has_ko_edition` 규칙의 근거)
- TCGdex 변형 값을 `card_variants.variant` 7종으로 바꾸는 규칙
- 한·일·영 포켓몬 종 이름 소스 (`pokemon_species`)

- 확인일은 별도 표시가 없으면 모두 2026-11-10(KST)이다.
- 호출 조건: 동시성 1, 요청 간 최소 250ms, User-Agent `PokeCardPrice-collector/0.1 (research)`. 조사 중 TCGdex 호출은 약 1,100회였다.
- 확인 상태: `응답 확인`, `문서 확인`, `미확인`, `확인 불가`

## 조사 내용

### 1. TCGdex 이용 조건·호출 한도

| 항목 | 내용 | 상태 |
|------|------|------|
| 호출 한도 | FAQ: "There are no published hard rate limits, but please be considerate. For bulk data needs, cache responses locally rather than fetching the same data repeatedly." | 문서 확인 |
| API 키 | FAQ: "No. The TCGdex API is free to use and requires no API key." | 문서 확인 |
| 응답 헤더 | `RateLimit-*`, `Retry-After` 없음. `Cache-Control: no-cache, no-store, must-revalidate`, `ETag` 있음 | 응답 확인 |
| 데이터 라이선스 | `tcgdex/cards-database` MIT ("Copyright (c) 2021 TCGdex"). 저작권·허가 문구를 포함해야 함 | 문서 확인 |
| 이미지·가격 이용 조건 | `assets.tcgdex.net` 이미지와 가격 데이터에 MIT가 적용되는지 기재 없음. 카드 그림·텍스트의 권리는 권리자에게 있고, 저장소는 Nintendo·The Pokémon Company와 무관하다고 밝힘 | `미확인` |
| 언어 지원 | 홈페이지 지원 언어 목록에 한국어는 "Coming Soon". 실제 `ko` 엔드포인트는 응답함 | 문서 확인, 응답 확인 |

- 이미지는 URL만 저장하고 파일을 내려받지 않는다(수행 계획서 범위 밖). 화면 표기는 #13에서 출처(TCGdex)와 함께 다룬다.
- 공개 배포 전 이미지 이용 조건 재확인이 필요하다 (#16).

### 2. 엔드포인트와 응답 필드

| 엔드포인트 | 응답 | 테이블 컬럼 대응 | 상태 |
|------------|------|------------------|------|
| `/v2/{lang}/sets` | `id`, `name`, `cardCount{total, official}`, `logo`(en만), `symbol` | `sets.tcgdex_id`, `name`, `card_count`, `logo_url` | 응답 확인 |
| `/v2/{lang}/sets/{id}` | 위 항목 + `releaseDate`, `serie`, `cards[]{id, localId, name, image}` | `sets.release_date`, `cards.tcgdex_id`, `local_id`, `name`, `image_url` | 응답 확인 |
| `/v2/{lang}/cards/{id}` | 위 항목 + `rarity`, `dexId`, `variants`, `variants_detailed`, `category` 등 | `cards.rarity`, `dex_ids`, `card_variants` | 응답 확인 |
| `/v2/{lang}/cards?...` | 목록 항목은 `id`, `localId`, `name`, `image`뿐. 필터(`set.id=eq:...`)·페이지네이션 동작 | 상세 필드 없음 | 응답 확인 |

- **`rarity`·`dexId`·`variants`는 카드 상세에만 있다.** 목록·세트 상세로는 카드 행과 변형을 다 채울 수 없다.
- 언어별 규모: `en` 세트 221개·카드 24,140장, `ja` 204개·19,268장, `ko` 95개·7,931장(카드 수 합계). `ko`는 실제 카드가 239장뿐이다 (4절).
- 카드 상세 응답 시간: 표본 30건 평균 0.535초, 최대 1.339초 (요청 간격 대기 제외).
- 세트 상세 응답 시간: `ko` 95건 평균 0.406초.

#### GraphQL (`/v2/graphql`)

| 항목 | 결과 | 상태 |
|------|------|------|
| 문서 | "Full documentation in progress"뿐. 인자·언어·페이지네이션 설명 없음 | 문서 확인 |
| 전체 카드 한 번에 조회 | `{ cards { id rarity dexId variants{...} set{id} } }` 1회로 영어 카드 23,912장, 4.4 MB, 8.2초 | 응답 확인 |
| 페이지네이션 | `pagination:{page, itemsPerPage}`를 주면 서버 오류(`value.indexOf is not a function`) | 응답 확인 |
| 언어 지정 | 영어만. `?lang=ja`, `/v2/ja/graphql`(404), `Accept-Language` 모두 일본어 카드를 돌려주지 않음 | 응답 확인 |
| 세트의 카드 | `set(id){ cards{...} }`는 카드 207개 자리가 모두 `null` | 응답 확인 |

- 영어 카드 수가 REST 세트 합계(24,140)보다 228장 적다. 원인은 `미확인`.
- 호출 수를 크게 줄일 수 있지만, 문서가 없고 영어만 되며 오류가 있어 **수집 경로로 쓰지 않는다** (Q1). 변형 분포 분석(5절)에만 사용했다.

### 3. 카드 상세 호출 수

| 방안 | 매 실행 호출 수 | 소요 시간 (추정) | 비고 |
|------|----------------|------------------|------|
| A. 매번 전체 상세 조회 | 세트 520 + 카드 약 43,600 | 동시성 1: 약 6.5시간, 동시성 2: 약 3.3시간 | 매일 하기엔 소스 부담이 큼 |
| B. 신규 카드만 상세 조회 (선정안) | 세트 520 + 신규 카드 수 | 평소 약 4~5분. 첫 적재는 A와 같음 | 기존 카드의 `rarity`·`dex_ids`·변형 변경은 정기 전체 갱신으로 반영 |
| C. GraphQL로 영어 일괄 조회 | 세트 520 + GraphQL 1 + `ja`·`ko` 신규 카드 | 평소 수 분 | 비공식 경로 (2절) |

- 추정식: 호출 1회 = 응답 0.53초 + 대기. 동시성 2는 응답 시간을 겹쳐 절반으로 계산했다.
- B에서 세트 상세로 얻는 `name`·`local_id`·`image_url`은 매일 갱신하고, 카드 상세에서만 얻는 필드는 신규 카드일 때와 정기 전체 갱신 때만 바꾼다.

### 4. 한글판·일본판 일치 범위

`ko` 세트 95개 전체와 `ko` 카드 239장 전체를 같은 ID의 `ja` 세트·카드와 비교했다.

**세트**

| 측정 | 결과 | 상태 |
|------|------|------|
| `ko` 세트 중 같은 ID의 `ja` 세트가 있음 | 91 / 95 | 응답 확인 |
| 같은 ID의 `ja` 세트가 없는 `ko` 세트 | `SM1+` 썬&문, `SM3+` 빛나는 전설, `SM4+` GX 배틀부스트, `SM5+` 울트라포스. 4개 모두 세트 상세가 404 | 응답 확인 |
| `ko` 카드 수(`total`) = `ja` 공식 번호 수(`official`) | 89 / 91 | 응답 확인 |
| 나머지 2개 (`SV4K`, `SV4M`) | `ko`·`ja` 모두 `total` 95, `official` 66. 카드가 실제로 있는 세트 | 응답 확인 |
| `ja`에 카드 목록이 없는 세트 | 31 / 91 (`S1H`, `S1W`, `S2`, `S3`, `CS*` 등). 세트 메타데이터만 있음 | 응답 확인 |
| `ko` 세트 수록 기간 | 2016-12-09 ~ 2024-06-21 (SM·S·SV 시리즈). 이후 한글판 세트(`SV6a`~, `M*`)는 TCGdex `ko`에 없음 | 응답 확인 |
| `ja` 카드 `localId` 형식 | 비교 대상 세트 모두 숫자 문자열 | 응답 확인 |

- 카드 목록이 없는 `ko` 세트의 `cardCount.total`은 대부분 일본판의 **공식 번호 수**와 같다. 공식 번호를 넘는 시크릿 카드(SR·SAR 등)가 한글판에도 있는지는 TCGdex로 알 수 없다 (`미확인`).

**카드 (`ko` 239장: `SV4K` 95, `SV4M` 95, `SV5K` 49)**

| 측정 | 결과 | 상태 |
|------|------|------|
| 같은 ID(세트 ID + `localId`)의 `ja` 카드가 있음 | 239 / 239 | 응답 확인 |
| `dexId` 일치 | 239 / 239 (트레이너스·에너지 등 둘 다 없음 68장 포함) | 응답 확인 |
| `variants` 일치 | 239 / 239 | 응답 확인 |
| `rarity` 일치 | 239 / 239 | 응답 확인 |
| `ko` 카드 이미지 | 0 / 239 (모두 없음) | 응답 확인 |

- 표본이 3개 세트(SV 시리즈)뿐이라, 다른 시리즈의 번호 일치는 보장되지 않는다. 그래서 연결 조건에 `dexId` 일치를 더한다 (Q2).

### 5. 변형 매핑

TCGdex `variants`는 `normal`, `reverse`, `holo`, `firstEdition`, `wPromo` 다섯 개의 참·거짓 값이다. 영어 카드 23,912장(GraphQL)의 조합 분포는 다음과 같다.

| 조합 | 카드 수 | 예 |
|------|---------|-----|
| `normal` | 9,112 | `bw10-1` |
| `normal`+`reverse` | 7,623 | `sv03.5-001` |
| `holo` | 4,686 | `dv1-1` |
| `reverse`+`holo` | 1,053 | `ex3-1` |
| `normal`+`firstEdition` | 756 | `neo3-15` |
| `normal`+`reverse`+`holo` | 326 | `hgss1-1` |
| `holo`+`firstEdition` | 179 | `base2-1` |
| `normal`+`holo` | 143 | `pop5-1` |
| `reverse` | 31 | `bog-1` |
| `normal`+`reverse`+`firstEdition` | 2 | `base2-45` |
| `normal`+`holo`+`firstEdition` | 1 | `base5-15` |
| 모두 거짓 또는 없음 | 0 | |
| `wPromo` 참 | 0 | |

**`firstEdition`의 의미.** `firstEdition`이 참이면 초판만 있다는 뜻이 아니라 **초판과 무제한판이 모두 있다**는 뜻이다. TCGdex 응답에 함께 들어 있는 TCGplayer 가격 키로 확인했다.

| 카드 | `variants` | TCGplayer 가격 키 (`variants_detailed`) |
|------|------------|------------------------------------------|
| `base2-1` | `holo`+`firstEdition` | `1st-edition-holofoil`, `unlimited-holofoil` |
| `neo3-15` | `normal`+`firstEdition` | `1st-edition`, `unlimited` |
| `ex3-1` | `holo`+`reverse` | `holofoil`, `reverse-holofoil` |
| `sv03.5-001` | `normal`+`reverse` | `normal`, `reverse-holofoil` |
| `base1-4` | `holo`+`firstEdition` | `holofoil` (무제한판 상품). 초판은 TCGplayer에서 별도 상품(Shadowless) |

**매핑 표 (선정안, Q3)**

| TCGdex 값 | `firstEdition` 거짓 | `firstEdition` 참 |
|-----------|--------------------|-------------------|
| `normal` | `normal` | `first_edition`, `unlimited` |
| `holo` | `holofoil` | `first_edition_holofoil`, `unlimited_holofoil` |
| `reverse` | `reverse_holofoil` | `reverse_holofoil` |
| `wPromo` | 변형을 만들지 않고 로그만 남김 (현재 0건) | 같음 |
| 모두 거짓 또는 없음 | `normal` 하나를 만들고 로그 (현재 0건) | - |

- 결과 코드 7종은 TCGCSV `subTypeName`(`Normal`, `Holofoil`, `Reverse Holofoil`, `1st Edition`, `1st Edition Holofoil`, `Unlimited`, `Unlimited Holofoil`)과 1:1로 대응해 #6이 연결하기 쉽다.
- 도장(`stamp`: `set-logo`, `pre-release`, `1st-edition`)과 `subtype`(`shadowless` 등)은 `variants_detailed`에만 있고, 7종으로 표현할 수 없다. `base1-4`처럼 TCGplayer가 별도 상품으로 파는 경우는 #6에서 외부 ID 연결 시 다룬다.
- 일본판 표본(`SV4K-001`, `SV2a-001`)도 같은 키 구조였다.

### 6. 포켓몬 종 이름 소스

| 후보 | 한·일·영 이름 | 범위 | 이용 조건 | 적재 방식 | 상태 |
|------|---------------|------|-----------|-----------|------|
| **PokeAPI 원본 CSV** (`PokeAPI/pokeapi` 저장소 `data/v2/csv/pokemon_species_names.csv`) | 있음 (언어 ID: 한국어 3, 영어 9, 일본어 가나 1) | 1~1025번 전체, 언어마다 1,025행 | BSD 3조항형 라이선스 ("Copyright (c) 2013–2023 Paul Hallett and PokéAPI contributors"). 저작권 문구를 포함하면 재배포 가능. 포켓몬 이름은 Nintendo 상표 | 파일 1개, API 호출 없음 | 응답 확인, 문서 확인 |
| PokeAPI REST (`/api/v2/pokemon-species/{id}`) | 있음 | 위와 같음 | 같음. 종마다 1회 호출 (1,025회) | 수집기 호출 | 문서 확인 |
| TCGdex 카드 이름 | `ko`는 239장뿐, `ja`·`en`은 카드 이름 (ex·V 등 접미사 포함) | 좁음 | MIT | 이름 정제가 필요 | 응답 확인 |

- TCGdex `dexId` 최댓값은 1025로, PokeAPI 범위 안에 있다. `dexId`가 둘 이상인 카드(TAG TEAM 등)는 영어 기준 130장이다.
- 일본어는 TCGdex 일본판 카드 이름과 같은 가나 표기(언어 ID 1, `ピカチュウ`)를 쓴다. 언어 ID 11도 이 종에서는 같은 값이었다.
- 예: 25번 `Pikachu` / `ピカチュウ` / `피카츄`.

### 7. TCGdex `ko`에 없는 한글판 카드 이름

| 소스 | 결과 | 상태 |
|------|------|------|
| TCGdex `ko` | 239장 외에는 카드 이름 없음 | 응답 확인 |
| TCGCSV | 한글판 카테고리 없음 (#2) | 응답 확인 |
| PokeAPI | 종 이름만 있음. 카드 이름(ex·V·트레이너스 카드명)은 없음 | 문서 확인 |

해외 공개 소스로는 얻을 수 없다. 포켓몬 카드는 종 이름(6절)으로 한글 검색을 보완한다. 트레이너스·에너지 카드의 한글 이름은 범위 밖으로 남긴다.

### 8. 기타 확인 사항 (#6 인계)

- **TCGdex TCGplayer 가격이 다시 채워져 있다.** #2 조사(2026-11-01)에서는 표본 8종 모두 `pricing.tcgplayer`가 `null`이었으나, 2026-11-10에는 `sv03.5-001`, `base2-1` 등에 `marketPrice` 등이 있었다(갱신 시각 `2026-11-10T03:13:49Z`). #6의 소스 선정(TCGCSV)에는 영향이 없지만, 대조용으로 쓸 수 있다.
- `variants_detailed`의 `thirdParty.tcgplayer`는 같은 카드의 변형들이 같은 상품 ID를 공유한다 (`sv03.5-001` normal·reverse 모두 502552). 변형 구분은 TCGCSV `subTypeName`으로 해야 한다.

## 결론

1. TCGdex는 호출 한도를 공개하지 않고 로컬 캐시를 권장한다. 카드 상세 호출이 전체 비용의 대부분(약 43,600회)이므로 **신규 카드만 상세 조회하고, 정기 전체 갱신을 따로 둔다** (3절 B안).
2. GraphQL은 영어 전체를 한 번에 받을 수 있지만 문서가 없고 영어만 되며 오류가 있어 쓰지 않는다.
3. `ko` 카드 239장은 모두 같은 ID의 `ja` 카드와 `dexId`·변형·희귀도까지 일치했다. **같은 ID + `dexId` 일치**를 자동 연결 조건으로 한다.
4. `ko` 세트 91개는 일본판과 세트 ID가 같고, 카드 수는 일본판 공식 번호 수와 같다. `has_ko_edition`은 **같은 ID의 `ko` 세트가 있는 `ja` 세트에서 번호가 `ko` 카드 수 이하인 카드**에 표시한다. 2024-06 이후 한글판 세트는 TCGdex에 없어 표시되지 않는다.
5. `firstEdition`은 초판·무제한판이 모두 있다는 뜻이다. 매핑 결과 7종은 TCGCSV `subTypeName`과 1:1로 대응한다.
6. 종 이름은 PokeAPI 원본 CSV(BSD 3조항형)에서 필요한 열만 뽑아 저장소에 넣고, 라이선스 문구를 함께 둔다.

## 적용 방향

### 확인 사항 (2026-11-10 작업지시자가 모두 제안대로 결정)

| # | 사항 | 제안 | 대안 |
|---|------|------|------|
| Q1 | 카드 상세 호출 방안 | **B안.** 매일: 세트 목록·세트 상세(약 520회, 약 5분) + DB에 없는 카드만 상세. 정기 전체 갱신: 월 1회(매월 1일) 별도 작업으로 모든 카드 상세 재조회, 동시성 2·요청 간격 250ms (약 3~4시간). 첫 적재도 같은 조건 | 정기 전체 갱신을 주 1회로 / 동시성 1로 낮춤 (약 6.5시간) |
| Q2 | 한글판 자동 연결 규칙 | `ko` 카드와 **세트 ID·`localId`가 같고 `dexId`가 같은**(둘 다 없으면 같다고 봄) `ja` 카드를 `card_links`(`auto`)로 연결. 어긋나면 연결하지 않고 로그 (#7 수동 연결 대상) | 세트 ID·`localId`만 비교 |
| Q3 | 변형 매핑 | 5절 매핑 표 | - |
| Q4 | `has_ko_edition` 표시 범위 | **같은 ID의 `ko` 세트가 있는 `ja` 세트에서 `localId`(숫자) ≤ `ko` 세트 `cardCount.total`인 카드.** 확인된 범위만 표시하므로 한글판 시크릿 카드는 대부분 빠진다 | 같은 ID의 `ko` 세트가 있으면 `ja` 세트의 모든 카드에 표시 (시크릿 카드 포함, 일부 오표시 가능) |
| Q5 | 종 이름 적재 방식 | PokeAPI CSV에서 `dex_id`·영어·일본어(가나)·한국어 열만 뽑은 CSV(1,025행)와 라이선스 문구를 `src/backend/internal/species/`(가칭)에 넣고 `embed`. 카탈로그 작업 시작 시 upsert | PokeAPI REST를 수집기가 호출 (1,025회) |

### 범위 밖으로 남기는 사항

- 2024-06 이후 한글판 세트, TCGdex `ko`의 404 세트 4개(`SM1+` 등 한국 전용 구성 세트)의 표시: 일본판과 대응이 확인되지 않아 이번에는 표시하지 않는다. 한글판 세트 목록을 수동으로 관리할지는 #7에서 정한다.
- `ja`에 카드 목록이 없는 31개 세트: TCGdex가 카드를 추가하면 다음 실행에서 자동으로 표시된다.
- 이미지 이용 조건 재확인: 공개 배포 전 (#16).

### 2단계 구현에 반영할 사항

- 세트 상세의 카드 요약으로 `name`·`local_id`·`image_url`만 갱신하는 쿼리가 필요하다. 기존 `UpsertCard`는 `rarity`·`dex_ids`를 함께 덮어쓰므로, 요약만으로 호출하면 값이 `NULL`로 바뀐다.
- `ko` 카드 이미지는 없으므로 `image_url`은 `NULL`로 둔다. 화면은 연결된 `ja` 카드 이미지를 쓸 수 있다 (#13).
- 응답에 `ETag`가 있으나 `Cache-Control: no-store`라, 조건부 요청(`If-None-Match`)은 쓰지 않는다.

## 구현 결과

### 2단계: TCGdex 클라이언트와 en·ja 적재 (2026-11-17)

| 항목 | 구현 |
|------|------|
| 클라이언트 | `internal/tcgdex`. 요청 간 최소 250ms(클라이언트 전체 공유), 재시도는 네트워크 오류·`429`·`5xx`만 최대 3회(1s→2s→4s, `Retry-After` 초 단위·최대 30초), `404` → `ErrNotFound`, User-Agent `PokeCardPrice-collector/0.1`. 의존성 추가 없음 |
| 상세 호출 범위 | 세트 상세는 매번, 카드 상세는 DB에 없는 카드만 (`ListCardTcgdexIDsBySet`). `Options.Full`이면 전체. 카드 상세 동시성 2 |
| 기존 카드 갱신 | `UpdateCardBrief`로 번호·이름·이미지만, 값이 바뀐 경우에만 갱신 |
| 트랜잭션 | 외부 호출을 마친 뒤 세트 하나 = 트랜잭션 하나. 실패 세트만 롤백하고 `FailedSets`에 기록 |
| 404 | 세트 상세 404 → 세트 건너뜀(`SetsSkipped`), 신규 카드 상세 404 → 카드 건너뜀(`CardsSkipped`). 다음 실행에서 다시 시도 |
| 변형 | `MapVariants` (5절 표). 변형 값이 없으면 `normal` 하나, `wPromo`는 저장하지 않고 로그 |

표본 적재(개발 DB): `en` `base2` 64장·변형 130, `exu` 28장·28, `sv03.5` 207장·362, `ja` `SV4K` 95장·95. 세트별 카드 수가 `cardCount.total`과 같고, 재실행 시 행이 늘지 않았다 (2.5초). `ja` `SV4K`의 67번 이후 29장은 TCGdex에 희귀도가 없다.

### 3단계: 한글판 연결과 종 이름 적재 (2026-11-17)

| 항목 | 구현 |
|------|------|
| 실행 시점 | 종 이름: 작업 시작 시. 연결·`has_ko_edition`: 언어별 적재 후 DB 전체 기준, 한 트랜잭션 |
| 자동 연결 | `UpsertAutoCardLinks` (Q2 규칙, `ko` → `ja`, `manual` 보호) |
| 오래된 연결 | `DeleteStaleAutoCardLinks`: 규칙에 맞지 않게 된 `auto` 연결 삭제 (`manual` 제외) |
| 미연결 | `ListUnlinkedKoCards`로 건수와 앞 20장 ID를 `WARN` 로그 |
| `has_ko_edition` | `SetHasKoEdition` (Q4 규칙). 규칙 밖 `ja` 카드는 거짓으로 되돌림. 숫자가 아닌 번호는 거짓 |
| 종 이름 | `internal/species/pokemon_species_names.csv` (PokeAPI 커밋 `8d8a49d`, 1,025행, 영어·일본어 가나·한국어) + `LICENSE.pokeapi.md`, `embed` 후 `UpsertPokemonSpecies` |

실데이터: `ko` 세트 91개 저장(4개 404), 카드 239장 모두 `auto` 연결(1단계 측정과 일치). `has_ko_edition`은 `SV4K` 95, `SV4M` 95, `SV5K` 71(`ko` 세트 카드 수 71, 수록 카드 49). `ko` 전체 적재 1분 25초.

#### 종 이름 데이터 갱신

새 세대로 종이 추가되면 PokeAPI `data/v2/csv/pokemon_species_names.csv`에서 `local_language_id` 9(`en`)·1(`ja-Hrkt`)·3(`ko`)의 `name`을 도감 번호별로 모아 `dex_id,name_en,name_ja,name_ko` 형식으로 다시 만들고, `LICENSE.pokeapi.md`의 커밋·날짜를 고친다. 다음 카탈로그 작업에서 반영된다.

### 4단계: 수집기 연결과 전체 적재 실측 (2026-11-17)

| 항목 | 구현 |
|------|------|
| 작업 | `cmd/collector`의 `collect-daily` 하나. 매일 05:30 KST, 매월 1일은 전체 갱신(`Options.Full`). `-once`·`COLLECTOR_INTERVAL` 실행은 `CATALOG_FULL=true`일 때만 전체 갱신 |
| 범위 지정 | `CATALOG_LANGUAGES`(en·ja·ko만 허용), `CATALOG_SETS`, `CATALOG_FULL` (개발용, 비우면 전체) |
| 겹침 방지 | gocron 싱글톤 모드(`LimitModeReschedule`). 실행이 간격보다 길면 끝난 뒤 다음 실행 |
| 별도 장비 실행 | `scripts/collector.sh` (`setup`·`start`·`stop`·`restart`·`once [-d]`·`full [-d]`·`status`·`logs`), `COLLECTOR_DATABASE_URL`로 외부 DB 적재. 사용법은 `mydocs/manual/collector_remote.md` |
| 실패 재시도 | 모든 세트를 돈 뒤 30초 기다렸다가 실패한 세트와 세트 목록을 받지 못한 언어를 한 번 더 시도 (`SetsRecovered`). 그래도 실패하면 `FailedSets`에 `{lang}/{id}` 또는 `{lang}/*` |
| 응답 본문 오류 | 본문을 읽다 끊김·시간 초과는 재시도. JSON 형식·타입 오류는 재시도하지 않음 |
| 소수 도감 번호 | TCGdex가 `384.1`처럼 주는 값은 정수 부분만 저장하고 `WARN` 로그 (`ja` `PCG2-067` 등 4장, 모두 레쿠쟈) |

#### 실측 (개발 DB, 2026-11-17)

| 항목 | 1차: 빈 DB 첫 적재 | 2차: 갱신 실행 |
|------|-------------------|----------------|
| 소요 | 2시간 43분 45초 | 14분 9초 |
| 처리 세트 | 408 (실패 12, `ko` 목록 실패) | 511 (건너뜀 9, 1개는 재시도로 회복) |
| 새 카드 | 35,343 | 1,964 (`en` 352, `ja` 1,373, `ko` 239) |
| 변형 upsert | 46,774 | 2,156 |
| 재시도 로그 | 85회 (대부분 `503`) | 12회 |
| 최종 실패 | `en` 4개, `ja` 8개, `ko/*` | 없음 |

- 1차 실패 원인: `503` 3회 재시도 소진(TCGdex 일시 장애), 카드 상세 본문 읽기 중 시간 초과, 소수 도감 번호 해석 오류(`PCG2`·`PCG6`·`PCG7`·`PCG9`), 19:01 DNS 장애로 `ko` 세트 목록 실패. 이 결과로 실패 재시도·본문 오류 재시도·소수 도감 번호 처리를 추가했다.
- 2차 후 DB: `en` 221세트·23,912장, `ja` 197세트·13,156장, `ko` 91세트·239장, 변형 48,930, 종 1,025, `auto` 연결 239/239(미연결 0), `has_ko_edition` 4,871.
- 이후 매일 실행은 새 세트·새 카드만 카드 상세를 부르므로 2차보다 짧다 (세트 상세 약 510회 ≈ 3분 이상). 매월 1일 전체 갱신은 1차와 비슷한 약 3시간이다.

## 참고 자료

- TCGdex FAQ: https://tcgdex.dev/faq
- TCGdex REST 필터·페이지네이션: https://tcgdex.dev/rest/filtering-sorting-pagination
- TCGdex GraphQL: https://tcgdex.dev/graphql
- TCGdex 데이터베이스와 라이선스: https://github.com/tcgdex/cards-database
- PokeAPI 저장소와 라이선스: https://github.com/PokeAPI/pokeapi
- PokeAPI 종 이름 CSV: https://github.com/PokeAPI/pokeapi/blob/master/data/v2/csv/pokemon_species_names.csv
- #2 조사: `mydocs/tech/price_data_sources.md`, #4 스키마: `mydocs/tech/db_schema.md`
