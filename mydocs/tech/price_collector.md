# 기술 조사: 해외 시세 수집기

> **작성일**: 2026-11-19
> **관련 타스크**: [#6](https://github.com/Chocobone/PokeCardPrice/issues/6) 시세 수집 스케줄러 (해외 소스)
> **상태**: 1단계(사전 확인과 측정) 승인, Q1·Q2 확정안과 Q5~Q7 제안대로 결정 (2026-11-19). 2~5단계에서 구현 결과와 실측값을 추가한다.

---

## 배경

#6은 TCGplayer 가격(TCGCSV), Cardmarket 가격(TCGdex), 환율(Frankfurter v2)을 매일 수집해 `external_ids`, `price_history`, `fx_rates`에 저장한다.
구현 전에 다음을 확인해야 했다 (#2·#4·#5 인계 사항).

- TCGCSV 상품·가격 응답 구조와 소스 기준일
- **TCGCSV 상품을 어떤 규칙으로 카드 변형에 연결할지** (수행 계획서 Q1)
- TCGdex Cardmarket 필드, 특히 `-holo` 필드를 어느 변형에 넣을지 (#4 인계)
- Cardmarket 수집에 드는 호출 수와 시간 (수행 계획서 Q2)
- Frankfurter v2 응답 형식, 주말·공휴일 동작, 호출 한도 (#2 미해결 사항)

- 확인일은 별도 표시가 없으면 모두 2026-11-19(KST)이다.
- 호출 조건: 동시성 1, User-Agent `PokeCardPrice-collector/0.1`. TCGCSV 요청 간 120 ms, TCGdex 요청 간 500 ms. 조사 중 호출은 TCGCSV 약 30회, TCGdex 약 2,090회, Frankfurter 약 15회였다.
- 확인 상태: `응답 확인`, `문서 확인`, `미확인`, `확인 불가`

### 표본

카탈로그(#5)가 적재된 개발 DB에서 세트 14개, 카드 2,061장, 카드 변형 3,169개를 골랐다. 시대(초판·ex·SM·SWSH·SV·ME), 프로모, 일본판 구·신 세트를 섞었다.

| 언어 | TCGdex 세트 | TCGCSV 그룹 | 카드 |
|------|-------------|-------------|------|
| `en` | `base1` Base Set | 604 Base Set | 102 |
| `en` | `base2` Jungle | 635 Jungle | 64 |
| `en` | `neo1` Neo Genesis | 1396 Neo Genesis | 111 |
| `en` | `ex3` Dragon | 1376 EX Dragon | 100 |
| `en` | `sm1` Sun & Moon | 1863 SM Base Set | 172 |
| `en` | `swsh1` Sword & Shield | 2585 SWSH01: Sword & Shield Base Set | 216 |
| `en` | `sv01` Scarlet & Violet | 22873 SV01: Scarlet & Violet Base Set | 258 |
| `en` | `svp` SVP Black Star Promos | 22872 SV: Scarlet & Violet Promo Cards | 226 |
| `en` | `sv03.5` 151 | 23237 SV: Scarlet & Violet 151 | 207 |
| `en` | `me05` Pitch Black | 24688 ME05: Pitch Black | 120 |
| `ja` | `PMCG2` ポケモンジャングル | 23722 Pokemon Jungle | 48 |
| `ja` | `SV2a` ポケモンカード151 | 23599 SV2a: Pokemon Card 151 | 210 |
| `ja` | `SV4K` 古代の咆哮 | 23610 SV4K: Ancient Roar | 95 |
| `ja` | `SV9` バトルパートナーズ | 24173 SV9: Battle Partners | 132 |

## 조사 내용

### 1. TCGCSV 응답

| 항목 | 내용 | 상태 |
|------|------|------|
| 카테고리 | `Pokemon` 3, `Pokemon Japan` 85 (전체 94개) | 응답 확인 |
| 그룹 수 | `Pokemon` 221개, `Pokemon Japan` 462개 | 응답 확인 |
| 그룹 필드 | `groupId`, `name`, `abbreviation`, `isSupplemental`, `publishedOn`, `modifiedOn`, `categoryId` | 응답 확인 |
| 상품 필드 | `productId`, `name`, `cleanName`, `imageUrl`, `groupId`, `url`, `modifiedOn`, `extendedData[]`(`Number`, `Rarity`, `Card Type`, `HP` 등) | 응답 확인 |
| 가격 필드 | `productId`, `subTypeName`, `lowPrice`, `midPrice`, `highPrice`, `marketPrice`, `directLowPrice` (USD) | 응답 확인 |
| 카드가 아닌 상품 | 박스·팩·컬렉션은 `extendedData`에 `Number`가 없다. 예: `151 Elite Trainer Box`. 표본 가격 행 3,640개 중 313개(8.6%) | 응답 확인 |
| 갱신 시각 | `/last-updated.txt`: `2026-11-18T20:05:56+0000`. 응답 헤더 `Last-Modified`도 같은 시각 | 응답 확인 |
| `marketPrice` NULL | 표본 그룹 6개 중 0~7행. `marketPrice`·`midPrice`가 모두 NULL인 행은 0 | 응답 확인 |
| 하루 호출 수 | 그룹 683개 × (상품 + 가격) = 1,366회 + 목록 3회. 24시간 1만 회 한도의 약 14% | 계산 |

- **소스 기준일**은 `last-updated.txt` 시각의 UTC 날짜(2026-11-18)로 한다. KST로는 11-19 05:05이므로, 05:30 KST 수집은 그날 갱신분을 받는다.
- 상품 번호는 `001/165`(세트), `014`(SV 프로모), `1/149`(SM) 등 형식이 섞여 있다. 앞자리 0을 떼고 `/` 앞만 비교하면 TCGdex `localId`(`001`, `1`)와 맞는다.
- 같은 번호의 상품이 여러 개인 경우가 있다: `Charizard (Black Dot Error)`, `Mew ex - 205/165 (151 Metal Card)`, `Koraidon - 014 (Pokemon Center Exclusive)`, `Bulbasaur - 001/165 (Poke Ball Pattern)` 등. **이름에 괄호가 없는 상품이 기본 카드**이다.

### 2. 상품 ↔ 카드 변형 연결 측정 (Q1)

**두 방법**

| 방법 | 내용 |
|------|------|
| A. TCGdex 상품 ID | TCGdex 카드 상세 `pricing.tcgplayer`의 키(`normal`, `reverse-holofoil`, `holofoil`, `1st-edition`, `1st-edition-holofoil`, `unlimited`, `unlimited-holofoil`)별 `productId`. (`productId`, 키에 대응하는 `subTypeName`) → (카드, 변형) |
| B. 번호 대조 | TCGCSV 그룹 ↔ TCGdex 세트, 상품 `Number` ↔ 카드 `localId`, `subTypeName` → 변형. 같은 번호의 상품이 여럿이면 괄호 없는 상품 하나만 쓴다 |

`subTypeName` → 변형은 7종이 1:1로 대응한다 (#5 `catalog_collector.md` 5절). 단, 카드에 그 변형이 없으면 다음 규칙으로 대신한다 (측정 결과로 정함).

| 경우 | 규칙 | 근거 (표본 건수) |
|------|------|-----------------|
| 초판이 있는 세트 (`first_edition`·`unlimited` 계열) | `Normal` → `unlimited`, `Holofoil` → `unlimited_holofoil` | TCGplayer 기본 그룹은 무제한판이다. 초판은 다른 그룹(`Base Set (Shadowless)` 등)에 있다. A·B 모두 101건 |
| 일본판 카드의 변형이 1개 | `subTypeName`과 상관없이 그 변형 | TCGCSV 일본판은 홀로 카드의 `subTypeName`이 `Holofoil`이나 TCGdex는 `normal`로 둔다 (`SV9` 101~132번 등). 34건 |
| 그 밖에 카드에 없는 변형 | 연결하지 않음 | 아래 "연결 안 된 행" 참고 |

**결과 (가격 행 기준)**

| 구분 | 행 수 | 비율 |
|------|-------|------|
| TCGCSV 가격 행 | 3,640 | |
| 카드가 아닌 상품 | 313 | 8.6% |
| 카드 상품 가격 행 | 3,327 | 100% |
| A·B 모두 연결, 결과 같음 | 2,285 | 68.7% |
| **A·B 모두 연결, 결과 다름** | **0** | **0%** |
| A만 연결 | 29 | 0.9% |
| B만 연결 | 461 | 13.9% |
| 연결 안 됨 | 552 | 16.6% |
| **A 또는 B로 연결** | **2,775** | **83.4%** |

**결과 (세트별, 카드 변형 기준)**

| 세트 | 변형 | A | B | A 또는 B | 메모 |
|------|------|---|---|----------|------|
| `en` `base1` | 204 | 101 | 101 | 101 | 초판 102개는 다른 그룹 (연결 안 함) |
| `en` `base2` | 130 | 126 | 128 | 128 | |
| `en` `neo1` | 222 | 216 | 222 | 222 | A는 TCGdex 가격 없는 카드 3장 누락 |
| `en` `ex3` | 191 | 186 | 186 | 186 | |
| `en` `sm1` | 172 | 108 | 110 | 110 | **TCGdex에 리버스·홀로 변형 없음** (3절) |
| `en` `swsh1` | 380 | 368 | 380 | 380 | |
| `en` `sv01` | 444 | 444 | 444 | 444 | |
| `en` `svp` | 227 | 211 | 184 | 212 | 번호 중복 상품(포켓몬센터 한정 등) 많음. A가 유리 |
| `en` `sv03.5` | 362 | 360 | 360 | 360 | |
| `en` `me05` | 199 | 194 | 194 | 194 | |
| `ja` `PMCG2` | 48 | 0 | 0 | 0 | TCGCSV 상품에 `Number` 없음 (구판 일본 카드는 번호 없음) |
| `ja` `SV2a` | 363 | 0 | 210 | 210 | 리버스 153개는 `Poke Ball Pattern`·`Master Ball Pattern` 두 상품으로 나뉨 (연결 안 함) |
| `ja` `SV4K` | 95 | 0 | 95 | 95 | |
| `ja` `SV9` | 132 | 0 | 132 | 132 | |
| **합계** | **3,169** | **2,314** | **2,746** | **2,774 (87.5%)** | |

- **방법 A는 일본판에 쓸 수 없다.** TCGdex 일본판 카드는 `pricing.tcgplayer`가 모두 `null`이고, `variants_detailed`에도 TCGplayer ID가 거의 없다 (`ja` 743개 중 227개, 대부분 같은 카드의 영어판 ID로 보이나 확인하지 않음).
- **영어판에서 두 방법은 한 번도 어긋나지 않았다** (2,285건 일치). A는 프로모처럼 번호가 겹치는 세트에서, B는 TCGdex에 가격 정보가 없는 카드에서 더 많이 연결한다.

**연결 안 된 카드 상품 행 552개**

| 원인 | 행 | 처리 |
|------|----|------|
| `ja` 패턴 상품 (`Poke Ball Pattern`, `Master Ball Pattern`) | 305 | 변형 7종으로 두 상품을 구분할 수 없음. 연결하지 않음 |
| `en` `sm1` 리버스 홀로 (TCGdex에 변형 없음) | 179 | **Q5** |
| `en` 특별 상품 (`Pokemon Center Exclusive`, `[Staff]`, `Black Dot Error`, `Metal Card`) | 67 | 기본 카드가 아니므로 연결하지 않음 |
| 기타 (`swsh1` Evolution Incense 리버스, TCGdex에 변형 없음) | 1 | Q5와 같음 |

**그룹 ↔ 세트 대응 (방법 B의 전제)**

| 방법 | `en` | `ja` |
|------|------|------|
| 그룹 이름 앞 코드 (`SV2a: Pokemon Card 151` → `SV2a`)가 TCGdex 세트 ID와 같음 | 그룹 24개 | 세트 120 / 197개, 카드 10,003 / 13,156장 (76%) |
| 발매일이 같은 세트가 하나뿐 | 100개 (오연결 가능, 검증 안 함) | - |
| 자동 대응 불가 | 97개 | 세트 77개 (구판 `PMCG`·`neo`·`E`·`PCG` 시리즈, `SV-P`·`M-P` 프로모, 카드 0장 세트 다수) |

- `en`은 이름·약어 체계가 TCGdex와 달라 이름 규칙만으로는 대응시키기 어렵다. 대신 **방법 A가 연결한 상품의 그룹**으로 세트 ↔ 그룹을 알 수 있다 (TCGdex 카드 상세 → `productId` → TCGCSV 그룹).
- `ja`는 코드 대응을 기본으로 하고, 남는 세트는 수동 대응표로 보완한다. 구판 일본 카드(번호 없음)는 대응해도 번호로 연결할 수 없다.

### 3. TCGdex 카탈로그의 변형 누락

`en` 세트 중 2011년 이후 발매, 카드 30장 이상인데 리버스 홀로 변형이 하나도 없는 세트가 53개다 (개발 DB, 2026-11-19).

| 구분 | 세트 | 원인 |
|------|------|------|
| 실물 세트, 리버스·홀로 누락 | `bw8`~`xy12`, `g1`, `sm1`~`sm12`, `sm115`, `dc1`, `mfb`, `me06` 등 약 30개 | TCGdex `variants`가 `normal`만 참. 예: `sm1-1` Caterpie는 `normal`만 있으나 TCGplayer에는 `Reverse Holofoil` 상품 가격이 있음 |
| 프로모·특수 세트 | `bwp`, `xyp`, `smp`, `swshp`, `mep`, `sma`, `swsh4.5sv` 등 | 원래 리버스가 없는 세트일 수 있음 |
| **TCG Pocket (디지털 게임)** | `A1`~`A4a`, `B1`~`B2a`, `P-A` (시리즈 `tcgp`) | 실물 카드가 아님. 2,480장. 가격 없음 (`A1-001` `pricing` 모두 `null`) |

- 실물 세트의 누락은 TCGdex 데이터의 한계이다. 이 세트들은 TCGplayer·Cardmarket에 리버스 홀로 가격이 있어도 붙일 변형이 없다 → **Q5**.
- TCG Pocket 세트는 #5 카탈로그에 들어와 있다. 시세 수집 대상에서 빼야 하고, 검색 결과에 섞이는 문제가 있다 → **Q6**.

### 4. TCGdex Cardmarket

| 항목 | 내용 | 상태 |
|------|------|------|
| 위치 | 카드 상세 `pricing.cardmarket` (최상위). `variants_detailed[].pricing.cardmarket`에도 변형·도장별로 있으나 문서에 없는 필드 | 응답 확인 |
| 필드 | `updated`, `unit`(`EUR`), `idProduct`, `avg`, `low`, `trend`, `avg1`, `avg7`, `avg30`, 같은 6개의 `-holo` 필드 | 응답 확인 |
| 갱신 | 표본 전체 `updated`가 `2026-11-19T03:12Z` 전후 (KST 12:12). 05:30 KST 수집은 전날 갱신분을 받는다 | 응답 확인 |
| 가격 있는 카드 | `en` 1,573 / 1,576 (99.8%), `ja` 437 / 485 (90.1%, `PMCG2` 48장 없음) | 응답 확인 |
| `unit`이 `EUR`가 아님 | 0건 | 응답 확인 |
| 값 `0` | `trend-holo: 0`처럼 거래가 없는 필드가 `0`으로 오는 경우가 있음 (`sv03.5-001`의 도장 변형 등) | 응답 확인 |

**`-holo` 필드의 의미.** Cardmarket은 같은 상품 안에서 리버스 홀로를 "holo" 속성으로 구분한다. 변형 구성별로 기본 필드(`avg`·`trend`)와 `-holo` 필드(`avg-holo`·`trend-holo`)에 값이 있는 카드 수를 셌다.

| 카드 변형 구성 | 카드 | 기본 필드 있음 | `-holo` 있음 | 해석 |
|----------------|------|----------------|--------------|------|
| `normal`+`reverse_holofoil` | 704 | 704 | 703 | 기본 = 일반, `-holo` = 리버스 |
| `holofoil`+`reverse_holofoil` | 105 | 105 | 105 | 기본 = 홀로, `-holo` = 리버스 |
| `holofoil` | 551 | 551 | 59 | 기본 = 홀로. `-holo`는 대부분 비어 있음 |
| `normal` | 363 | 363 | 173 | 기본 = 일반. `-holo`가 있는 카드는 실제로 리버스가 있으나 TCGdex에 변형이 없는 경우 (`sm1` 등, 3절) |
| `first_edition`+`unlimited` 계열 | 277 | 277 | 237 | 초판·무제한판을 나누지 않은 가격. `-holo`는 `avg-holo`가 비고 `trend-holo`만 있는 등 불규칙 |
| `holofoil`+`normal`(+`reverse_holofoil`) | 10 | 10 | 9 | 드묾. `ex3` 일부와 `sv03.5-004` |

**`-holo` 필드 연결 규칙 (제안)**

| 필드 | 넣을 변형 (먼저 있는 것) |
|------|--------------------------|
| 기본 필드 (`avg`, `low`, `trend`, `avg1`, `avg7`, `avg30`) | `normal` → `holofoil` → `unlimited_holofoil` → `unlimited` |
| `-holo` 필드 | `reverse_holofoil` (없으면 버림) |

- 초판(`first_edition`, `first_edition_holofoil`)에는 Cardmarket 가격을 붙이지 않는다. Cardmarket은 초판을 같은 상품의 속성으로 다루어 기본 필드가 어느 판의 가격인지 알 수 없다. 무제한판에 붙이는 쪽이 거래량상 더 가깝다고 판단했으나 원본 대조는 하지 못했다(`cardmarket.com` 조회 금지, #2).
- 대표값은 #4 규칙대로 `trend`, 없으면 `avg`이다. `0`은 값이 없는 것으로 본다(가격 `0`을 시세로 저장하지 않음). 둘 다 없으면 그 변형의 행을 만들지 않는다.
- 같은 카드의 두 변형이 같은 `idProduct`를 공유하므로 `external_ids`에는 저장하지 않는다(키가 겹침). Cardmarket은 카드 ID(TCGdex)로 바로 찾는다.

### 5. Cardmarket 호출 수와 소요 시간 (Q2)

| 항목 | 값 |
|------|----|
| 대상 카드 | `en` 23,912 + `ja` 13,156 = 37,068장. TCG Pocket 2,480장을 빼면 **34,588장** |
| 표본 응답 시간 | 2,087회, 평균 약 0.40초 (요청 대기 제외), 오류 0회 |
| 수집기 설정 (#5) | 요청 간 250 ms, 동시성 2 → 초당 최대 4회 |
| 하루 소요 시간 | 34,588 ÷ 4 ≈ **약 2시간 25분** (요청 간격이 응답 시간보다 짧아 간격이 상한) |
| 일정 (평소) | 05:30 카탈로그(약 2분) → TCGCSV(약 16분) → Cardmarket → **약 08:15 종료** |
| 일정 (매월 1일) | 카탈로그 전체 갱신 약 3시간 → **약 11:00 종료** |

**대상을 줄이는 방안 비교**

| 방안 | 절감 | 단점 |
|------|------|------|
| TCG Pocket 제외 | 2,480장 (6.7%) | 없음 |
| 가격이 없었던 카드 제외 (`ja` 약 10%) | 약 1,300장 (3.5%) | 새로 가격이 생긴 카드를 놓침. 주기적 전체 확인 필요 |
| 매일은 최근 세트만, 주 1회 전체 | 큼 | 구판 가격이 최대 7일 늦음. 날짜별 이력에 구멍 |
| 매월 1일 카탈로그 상세 응답에서 가격도 저장 | 그날만 3만 7천 회 절약 | 카탈로그·시세 코드가 얽힘 |

### 6. Frankfurter v2

| 항목 | 내용 | 상태 |
|------|------|------|
| 경로 | `GET /v2/rates?base=USD&quotes=KRW` (`date=YYYY-MM-DD`, `from`·`to` 기간 가능) | 응답 확인 |
| 응답 | `[{"date":"2026-11-19","base":"USD","quote":"KRW","rate":1333.27}]` 배열 | 응답 확인 |
| 2026-11-19 값 | USD→KRW 1,333.27, EUR→KRW 1,478.66(11-15) 등 | 응답 확인 |
| 주말 | 혼합값은 토·일 날짜로 값을 줌 (2026-11-15 일요일) | 응답 확인 |
| 공휴일 | 한국 한글날(2026-10-09), 유럽 노동절(2026-05-01)에도 그 날짜로 값을 줌. `providers=ECB`는 05-01에 직전 영업일(04-30) 값 | 응답 확인 |
| 미래 날짜 | `date=2026-11-20` → 최신 값(`date: 2026-11-19`)을 돌려줌. **응답의 `date`를 기준일로 저장해야 함** | 응답 확인 |
| 잘못된 통화 | HTTP 422 `{"status":422,"message":"invalid currency: XXX"}` | 응답 확인 |
| 없는 제공처 | `providers=BOK` → HTTP 200, 빈 배열 | 응답 확인 |
| 캐시 | `cache-control: public, max-age=…` (당일 값은 약 18시간) | 응답 확인 |
| 호출 한도 | 응답 헤더에 한도 정보 없음. 문서는 "일·월 한도 없음" (#2) | 응답 확인, 문서 확인 |
| 한국 기관 | KRW를 다루는 제공처 56곳 중 한국 기관 없음. 혼합값은 해외 기관 값의 중간값 | 응답 확인 |

- 수집은 하루 1회 `base=USD&quotes=KRW`, `base=EUR&quotes=KRW` 두 번 호출한다. 05:30 KST(20:30 UTC)에 받으면 대개 전날(UTC) 날짜 값이 온다. 응답의 `date`로 저장하므로 문제없다.
- 빈 배열이면 저장하지 않고 경고 로그만 남긴다. 원화 환산은 `GetFxRateOnOrBefore`로 직전 값을 쓴다.

## 결론

1. **Q1 연결 규칙: 방법 A를 우선 쓰고 B로 보완한다.** 영어판에서 둘은 한 번도 어긋나지 않았고(2,285건), 함께 쓰면 카드 상품 가격 행의 83.4%, 카드 변형의 87.5%가 연결된다. 일본판은 A가 없어 B만 쓴다.
2. 연결되지 않는 주요 원인은 일본판 패턴 상품(변형 7종으로 구분 불가), TCGdex의 변형 누락(Q5), 특별 상품이다. 셋 다 "틀린 연결보다 빈 연결" 원칙에 따라 연결하지 않는다.
3. **Q2 Cardmarket 수집: 매일 전체(TCG Pocket 제외) 34,588장, 약 2시간 25분**이다. 평소 08:15, 매월 1일 11:00 무렵 끝난다.
4. Cardmarket 기본 필드는 일반(없으면 홀로·무제한 홀로·무제한), `-holo` 필드는 리버스 홀로에 넣는다. 초판에는 붙이지 않는다.
5. Frankfurter는 주말·공휴일·미래 날짜에도 값을 준다. 응답의 `date`를 기준일로 저장한다.

## 적용 방향 (승인 후 2~5단계)

- 연결: 매일 TCGdex 카드 상세(Cardmarket 수집 시 받는 응답)의 `pricing.tcgplayer` `productId`로 `external_ids`를 갱신하고(A), 그 결과로 세트 ↔ 그룹 대응을 만든 뒤 남은 상품을 번호로 연결한다(B). `ja`는 코드 대응 + 수동 대응표.
  - 이 순서라면 Cardmarket(TCGdex) 수집이 TCGCSV 연결보다 먼저 끝나야 한다. 3·4단계 순서와 실행 흐름은 구현 중 조정하고 단계 보고서에 적는다. 이미 저장된 `external_ids`가 있으면 TCGCSV는 바로 수집할 수 있다.
- 저장: TCGCSV는 `last-updated` UTC 날짜, Cardmarket은 `updated` UTC 날짜를 `price_date`로 한다.
- 환율: 하루 2회 호출, 응답 `date`로 저장.

## 참고 자료

- TCGCSV: https://tcgcsv.com/ , https://tcgcsv.com/docs , https://tcgcsv.com/last-updated.txt
- TCGdex: https://tcgdex.dev/markets-prices , https://tcgdex.dev/reference/card
- Frankfurter: https://frankfurter.dev/docs/ , https://api.frankfurter.dev/v2/providers
- 관련 문서: `price_data_sources.md`(#2), `db_schema.md` 6·8·10절(#4), `catalog_collector.md` 5·8절(#5)
