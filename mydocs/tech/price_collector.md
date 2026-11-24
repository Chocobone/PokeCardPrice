# 기술 조사: 해외 시세 수집기

> **작성일**: 2026-11-19
> **관련 타스크**: [#6](https://github.com/Chocobone/PokeCardPrice/issues/6) 시세 수집 스케줄러 (해외 소스)
> **상태**: 1단계(사전 확인과 측정) 승인, Q1·Q2 확정안과 Q5~Q7 제안대로 결정 (2026-11-19). 2단계 승인(2026-11-23), 3단계(TCGCSV 어댑터) 승인(Q8·Q9 제안대로), 4단계(Cardmarket 어댑터) 승인(2026-11-23). 5단계에서 전체 실측값을 추가한다.

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

## 구현 결과

### 패키지 구성 (2단계)

| 패키지 | 역할 |
|--------|------|
| `internal/httpx` | 외부 소스 공용 HTTP 클라이언트. 요청 간 최소 간격(클라이언트 전체 공유), 전용 User-Agent, 네트워크 오류·429·5xx만 지수 백오프 재시도(기본 3회, 첫 대기 2초, `Retry-After` 초 단위 존중, 최대 60초), 404는 `ErrNotFound`, 그 밖의 4xx는 본문 앞 200바이트를 담아 바로 실패. `GetJSON`, `GetText` |
| `internal/fx` | Frankfurter v2 클라이언트. `Rates(ctx, base, quotes, date)`. 요청 간 1초. 환율이 양수가 아니면 오류 |
| `internal/prices` | `Source` 인터페이스, `Runner`(작업 동시 실행), `FX`(환율 작업). 3단계: `TCGCSV`(TCGplayer 가격), `TCGdex`(카드 상세 순회, 방법 A), 연결 규칙(`link.go`). 4단계: 같은 순회에서 Cardmarket 가격 저장, 배분 규칙(`cardmarket.go`) |
| `internal/tcgcsv` | TCGCSV 클라이언트 (3단계). `LastUpdated`, `Groups`, `Products`, `Prices`. 요청 간 120 ms, `httpx` 재시도. 응답 `success`가 거짓이면 오류 |

- `internal/tcgdex`는 #5 코드 그대로 두었다. 같은 재시도 로직이 `httpx`와 중복되지만, 상시 수집기가 쓰는 코드라 이번 타스크에서 옮기지 않는다.

### 작업 실행 (`prices.Runner`)

- `Source`는 `Code()`와 `Collect(ctx, Options) (Result, error)` 두 메서드이다. 가격 소스의 `Code`는 `price_sources.code`(`tcgcsv`, `tcgdex_cardmarket`), 환율 작업은 `fx`이다.
- `Runner.Run`은 `Options.Sources`에 해당하는 작업을 모두 고루틴으로 동시에 실행하고 기다린다. 작업 하나의 오류·패닉은 그 작업의 오류로만 남고, 다른 작업은 끝까지 실행된다. 반환 오류는 실패한 작업 오류를 합친 것(`errors.Join`)이다.
- 작업마다 시작·종료 로그를 남긴다. 종료 로그에는 `fetched`, `linked`, `saved`, `skipped`, `failed`, `duration`과 실패 항목 앞 20개가 들어간다 (Q3: 로그로만 기록).

### 환율 작업 (`prices.FX`)

- `USD→KRW`, `EUR→KRW`를 최신 값으로 1회씩 조회해 `UpsertFxRate`(provider `frankfurter`)로 저장한다. `rate_date`는 응답의 `date`이다.
- 빈 응답이면 저장하지 않고 경고만 남긴다(`skipped`). 한 쌍이 실패해도 다른 쌍은 저장한다.
- 개발 DB 실행 (2026-11-19 18:04 KST): `USD/KRW 2026-11-19 1,332.93`, `EUR/KRW 2026-11-19 1,464.90`, 1.4초.

### TCGCSV 수집 (3단계)

**실행 흐름.** 한 실행 안에서 TCGCSV 작업과 TCGdex 작업(`tcgdex_cardmarket`)이 동시에 돈다. `Runner`가 작업마다 "끝남" 채널을 만들어 `Options`로 넘기고, TCGCSV는 2번에서 TCGdex 작업이 끝나기를 기다린다 (같은 실행에 없으면 바로 진행).

| 순서 | TCGCSV 작업 | TCGdex 작업 |
|------|-------------|-------------|
| 1 | `last-updated.txt` → `price_date`(UTC 날짜)·`source_updated_at`. 대상 그룹의 상품·가격을 받는다 (실패 그룹은 끝에 한 번 더). 저장된 `external_ids`로 연결되는 행을 그룹 단위 트랜잭션으로 바로 저장 | 대상 카드(`en`·`ja`, TCG Pocket 제외)의 상세를 받아 `pricing.tcgplayer` 키별 `productId`를 모으고, 모든 카드를 받은 뒤 `external_ids`를 갱신 (방법 A). 값이 같으면 다시 쓰지 않고, 여러 카드가 같은 상품 ID를 주면 바꾸지 않는다 (5단계). 동시성 2, 요청 간격은 `tcgdex.Client` 공유. 실패 카드는 끝에 한 번 더 |
| 2 | 남은 행이 있으면 TCGdex 작업 종료를 기다린다 | (끝) |
| 3 | `external_ids`를 다시 읽어 남은 행을 연결하고, 그래도 남으면 그룹 ↔ 세트 + 번호로 연결 (방법 B). 방법 B 연결은 `external_ids`에 저장해 다음 날부터 1번에서 바로 쓴다 | |

- 빈 DB 첫 실행에서도 같은 날 모든 연결이 끝난다. 평소에는 대부분 1번에서 저장되고, 2번 대기 후에는 새 카드·새 상품만 남는다.
- 상품 정보는 이름과 정규화한 번호만 남겨 두 시간 넘게 기다리는 동안의 메모리를 줄였다.
- TCG Pocket 세트는 실행마다 TCGdex `/{lang}/series/tcgp`(언어당 1회)로 목록을 받아 뺀다 (Q6). 조회에 실패하면 경고만 남기고 빼지 않는다.

**연결 규칙 (구현).**

| 항목 | 규칙 |
|------|------|
| 그룹 ↔ 세트 | ① 수동 대응표(`manualGroupSets`, 현재 비어 있음) ② 그룹 상품 중 `external_ids`로 연결된 상품의 세트 — 가장 많은 세트가 90% 이상이고 언어가 같을 때 ③ 그룹 이름 앞 코드 = 같은 언어의 TCGdex 세트 ID (대소문자 무시, 하나뿐일 때) |
| 번호 비교 | `/` 앞만, 대문자, 숫자로 시작하면 숫자 부분의 앞자리 0 제거 (`001/165` → `1`, `050A` → `50A`, `TG01/TG30` → `TG01`). 접미사 처리는 5단계에서 추가 |
| **저장된 연결 확인** | 저장된 연결이 가리키는 카드 번호가 상품 번호와 다르면 연결과 그 변형의 이 소스 가격을 지우고 번호 대조로 다시 연결한다 (1번·3번 모두, 5단계) |
| 기본 상품 | 같은 번호의 상품이 하나면 그것, 여럿이면 이름에 괄호(`(`·`[`)가 없는 하나. 정할 수 없으면 번호 대조를 하지 않는다 |
| **특별 상품의 저장된 연결** | 같은 번호의 기본 상품이 그룹에 따로 있는 특별 상품은 **저장된 연결(방법 A 포함)이 있어도 쓰지 않는다** (아래 발견 사항) |
| 변형 대체 | 같은 변형 → 초판 세트 `Normal`→`unlimited`, `Holofoil`→`unlimited_holofoil` → 일본판 단일 변형 (1절 규칙 그대로) |
| Q5 변형 추가 | 영어판, 괄호 없는 기본 상품, 번호가 카드 하나에만 대응할 때 `normal`·`holofoil`·`reverse_holofoil`만 추가한다. 초판·무제한판은 판 구성을 바꾸므로 추가하지 않는다 |
| 모호한 연결 | 여러 가격 행이 한 변형에 연결되면(이번 실행에 이미 저장한 변형 포함) 모두 저장하지 않는다 (`ambiguous`) |
| 대표값 | `marketPrice`, 없거나 0 이하면 `midPrice`. 둘 다 없으면 가격 행을 만들지 않는다 (연결은 저장) |

- 연결하지 않은 행은 분류별 건수를 요약 로그(`tcgcsv skipped rows`)에 남긴다: `non_card`, `group_unmapped`, `special_product`, `card_not_found`, `variant_missing`, `ambiguous`, `no_price`. 상품별 내용은 디버그 로그. 같은 줄의 `links_removed`는 번호가 맞지 않아 지운 연결 수이다 (상품별로 `tcgcsv link number mismatch`).
- 같은 날 재수집을 막는 장치(기준일 행이 이미 있으면 건너뛰기)는 두지 않았다. 그룹 일부가 실패한 날을 구분하려면 실행 기록이 필요한데 Q3에서 로그만 남기기로 했다. 정기 실행은 하루 1회이고, 수동 `-once` 반복은 개발용 범위로 한다.
- 개발용 범위: `Options.TCGCSVGroups`(그룹 ID). `SetIDs`는 그룹 이름 코드가 같은 그룹만 고르므로(일본판 위주), 영어판 그룹은 ID로 지정한다.
- `tcgdex.Card.Pricing`은 원본 JSON(`json.RawMessage`)으로 받고 `Card.TCGPlayer()`로 따로 해석한다. 가격 필드 형식이 바뀌어도 카탈로그 적재(#5)는 실패하지 않는다.

**개발 DB 표본 수집 (2026-11-23 13:46 KST, TCGCSV 기준일 2026-11-22).**

| 세트 (TCGCSV 그룹) | 변형 | 가격 저장 | 1단계 "A 또는 B" | 차이 원인 |
|--------------------|------|-----------|------------------|-----------|
| `en` `sv03.5` (23237) | 362 | 360 | 360 | |
| `en` `svp` (22872) | 227 | 212 | 212 | |
| `ja` `SV2a` (23599, 코드 대응) | 363 | 210 | 210 | |
| `ja` `SV4K` (23610, 코드 대응) | 95 | 94 | 95 | `SV4K-095` 기본 악 에너지: 오늘 TCGCSV에 가격 행 없음 |
| `en` `sm1` (1863) | 172 → 325 | 263 | 110 | Q5로 리버스 126·홀로 27 변형 추가 |

| 항목 | 값 |
|------|----|
| 소요 시간 (표본 4세트) | 카드 상세 738장 3분 5초, TCGCSV 그룹 4개 상품·가격 8회 호출. TCGCSV만 재실행 2.5초 |
| 그룹 ↔ 세트 대응 | 방법 A 결과 2, 코드 2, 대응 없음 0 (`sm1` 포함 시 방법 A 1 추가) |
| 건너뛴 행 (표본 4세트) | 비카드 46, 특별 상품 370, 변형 없음 2 (`svp` `Cosmos Holo`·`Illustration Contest` 상품의 `Normal`) |
| 재실행 | 2·3회째 1번 단계에서 876행 저장, 행 수 불변 (`external_ids` 877, `price_history` 876, `card_variants` 48,930) |
| 원본 대조 | 저장 가격 667건(그룹 23237·22872·23610)의 `low`·`mid`·`high`·`market`·`direct_low`·대표값이 TCGCSV 원본과 모두 같음 |

**발견 사항**

1. **TCGdex가 특별 상품 ID를 주는 경우가 있다.** `svp-173` Eevee의 `pricing.tcgplayer.holofoil.productId`는 `610757 Eevee - 173 (Pokemon Center Exclusive)`이고, 기본 상품은 `610758 Eevee - 173`이다 (시세 61.22달러 대 8.00달러). 첫 구현에서는 이 연결로 한정판 가격이 저장되었다. 같은 번호의 기본 상품이 있으면 특별 상품의 저장된 연결을 쓰지 않도록 고쳤다. 1단계 "A·B 결과가 다른 건수 0"은 같은 가격 행끼리의 비교라 이런 경우(두 행이 한 변형을 다툼)는 세지 않았다.
2. **`(Full Art)`처럼 이름에 괄호가 있는 단독 상품**은 Q5 조건(괄호 없는 기본 상품)에 걸려 변형을 추가하지 않는다. `sm1`에서 26행 (`Lurantis GX (Full Art)` 등, TCGdex는 `normal`만 있음).
3. **Q5로 홀로 변형을 추가한 카드에는 가격 없는 `normal`이 남는다.** `sm1` 홀로 레어 27장은 TCGdex가 `normal`만 두었으나 TCGplayer에는 `Holofoil`만 있다. 추가한 `holofoil`에 가격이 붙고 `normal`은 비어 있다. 4단계 Cardmarket 기본 필드 규칙(`normal` 우선)을 그대로 쓰면 이 카드는 TCGplayer 가격(`holofoil`)과 Cardmarket 가격(`normal`)이 다른 변형에 붙는다. → Q9 결정(2026-11-23): TCGplayer 가격이 `holofoil`에만 있으면 Cardmarket 기본 필드도 `holofoil`에 넣는다 (4단계).
   - 2번은 Q8 결정(2026-11-23)으로 현행 유지, 5단계 전체 수집 건수를 보고 다시 판단한다.

### Cardmarket 수집 (4단계)

TCGdex 카드 상세 순회 작업(`tcgdex_cardmarket`)이 같은 응답 하나로 Cardmarket 가격 저장과 TCGplayer 상품 ID 연결(3단계)을 함께 한다. 대상은 `en`·`ja` 전체 카드(TCG Pocket 제외)이고 매일 실행한다 (Q2).

**배분 규칙 (Q7·Q9 구현).**

| 필드 | 넣을 변형 |
|------|-----------|
| 기본 (`low`, `avg`, `trend`, `avg1`, `avg7`, `avg30`) | `normal` → `holofoil` → `unlimited_holofoil` → `unlimited` 중 먼저 있는 것. 초판(`first_edition`, `first_edition_holofoil`)에는 넣지 않는다. **Q9**: `normal`·`holofoil`이 모두 있고 TCGplayer 연결(`external_ids`)이 `holofoil`에만 있으면 `holofoil` |
| `-holo` | `reverse_holofoil`. 없으면 버린다. 값이 없으면(대부분의 홀로 카드) 후보로 세지 않는다 |

- 대표값은 `trend`, 없으면 `avg`이다. `0` 이하는 거래가 없는 것으로 보고 모든 필드를 NULL로 둔다. 대표값이 없으면 그 변형의 행을 만들지 않는다.
- `price_date`는 `updated`의 UTC 날짜, `source_updated_at`은 `updated`이다. `unit`이 `EUR`가 아니거나 `updated`를 해석할 수 없으면 그 카드는 저장하지 않는다. 시작할 때 `price_sources.currency`가 `EUR`인지도 확인한다.
- Cardmarket 상품 ID(`idProduct`)는 같은 카드의 변형들이 공유하므로 `external_ids`에 저장하지 않는다 (1단계 결론).
- 저장은 500행 단위 트랜잭션이다. 저장에 실패하면 그 묶음의 카드를 실패로 두고, 조회 실패 카드와 함께 끝에 한 번 더 받는다.
- 저장하지 않은 이유는 요약 로그(`cardmarket skipped`)에 남긴다: `no_pricing`, `unit_not_eur`, `updated_invalid`, `no_value`, `variant_missing`.
- Q9는 TCGplayer 연결이 이미 있어야 적용된다. 빈 DB 첫 실행에서는 Q5 변형(`holofoil`)이 같은 실행의 TCGCSV 3번 단계에서 만들어지므로, 그런 카드의 첫날 Cardmarket 가격은 `normal`에 붙고 다음 날부터 `holofoil`에 붙는다.

**개발 DB 표본 수집 (2026-11-23 14:07 KST).** 대상: 3단계 5세트(`sv03.5`, `svp`, `SV2a`, `SV4K`, `sm1`) + 초판 세트 `base1` + 가격 없는 `ja` `PMCG2`.

| 항목 | 값 |
|------|----|
| 카드 | 1,060장, 4분 27초 (초당 약 4장, 1단계 계산과 같음), 실패 카드 0 |
| 저장 | 1,444행, 모두 `trend` 기준, `price_date` 2026-11-23 |
| 건너뜀 | 가격 없음 48장 (`PMCG2` 전체), 변형 없음 145 (리버스가 없는 카드의 `-holo` 값 등) |
| TCGplayer 연결 (같은 순회) | 새로 연결·변경 101, 그대로 830, 건너뜀 28 |
| `base1` | `unlimited` 86 / 86, `unlimited_holofoil` 16 / 16, 초판 0 (규칙대로) |
| `sm1` Q9 | `normal`+`holofoil` 카드 27장 모두 Cardmarket이 `holofoil`에 붙음, `normal` 0 |
| `ja` `SV2a` 리버스 | 153 / 153 (TCGCSV는 패턴 상품이라 연결 안 됨, Cardmarket `-holo`로만 가격이 있음) |
| 원본 대조 | `sv03.5-001`, `sm1-12`, `base1-4`, `SV2a-001`, `svp-173`의 모든 필드가 API 응답과 같음 |
| 재실행 (`sv03.5`) | 207장 53초, `price_history`·`external_ids`·`card_variants` 행 수 불변 |

### 전체 수집 실측 (5단계)

개발 DB, 2026-11-24 14:27~17:03 KST (2시간 36분), `-once` 전체 범위. 11-23에 시작한 첫 시도는 16:41부터 DNS 조회 실패(네트워크 끊김)로 카드 726장이 실패한 뒤 17:28에 끊겨 다시 실행했다.

| 작업 | 받은 것 | 저장 | 건너뜀 | 최종 실패 | 소요 시간 |
|------|---------|------|--------|-----------|-----------|
| 카탈로그 | 세트 511 (건너뜀 9) | 새 카드 0 | | 0 | 2분 46초 |
| 환율 (`fx`) | 2쌍 | 2 (기준일 11-24) | 0 | 0 | 1.9초 |
| TCGCSV | 가격 행 75,143 (그룹 685) | 42,476 (56.5%) | 32,667 | 0 | 2시간 33분 (1번 단계 5분, 나머지는 TCGdex 대기) |
| Cardmarket | 카드 34,588 | 40,434행 | 12,800 | 0 | 2시간 33분 |

- TCGCSV 건너뜀: `group_unmapped` 17,295, `non_card` 6,869, `special_product` 3,697, `card_not_found` 3,505, `variant_missing` 1,300, `ambiguous` 1. Q5 변형 추가 4,363.
- TCGCSV 그룹 ↔ 세트: 방법 A 결과 93, 코드 126, 대응 없음 320 (남은 행이 있는 그룹 기준).
- Cardmarket 건너뜀: `variant_missing` 8,832 (`-holo` 값이 있으나 리버스 변형 없음), `no_pricing` 3,559, `no_value` 409.
- TCGdex 요청: 카드 상세 34,588 + 재시도 84회 (HTTP 503). 실패 5장은 끝 재시도에서 모두 성공. 초당 약 3.8장으로 1단계 계산(2시간 25분)과 비슷하다.
- TCGCSV 요청: `last-updated` 1 + 그룹 목록 2 + 상품·가격 1,370.
- 행 수: `price_history` 34,829 → 117,739, `external_ids` 28,208 → 42,518, `card_variants` 49,083 → 53,446.
- 네트워크가 끊긴 동안 수집기는 카드마다 재시도 3번(약 7초) 후 실패로 넘기며 계속 진행했다. 실패 카드는 끝에 다시 받으므로 짧은 끊김은 스스로 복구된다.

### TCGdex 상품 ID 중복과 저장된 연결 확인 (5단계)

**발견.** 전체 수집 로그에 `tcgplayer link changed`가 92건 있었다. TCGdex가 **서로 다른 카드에 같은 TCGplayer 상품 ID**를 주어(30개 상품), 나중에 처리한 카드가 연결을 덮어쓰고 있었다. 동시성 2라 실행마다 결과가 달라질 수 있고, 가격이 다른 카드에 붙는다. TCGCSV 원본 대조:

| 상품 | TCGCSV 상품 | TCGdex가 이 ID를 준 카드 | 맞는 카드 |
|------|-------------|---------------------------|-----------|
| 89381 | Snivy `6/113` (Legendary Treasures) | `bw11-6`, `bw11-RC1` | `bw11-6` (`RC1`은 89382) |
| 89079 | Servine `7/113` | `bw11-7`, `bw11-RC2` | `bw11-7` (`RC2`는 89080) |
| 87775 | Ninetales `3` (Rumble) | `ru1-5` Starmie, `ru1-6` Gyarados | 둘 다 아님 (`ru1-3`) |
| 84971 | Drowzee `074A` (Aquapolis) | `ecard2-74a`, `ecard2-74b` | `ecard2-74a` |

대상: `bw11` ↔ Radiant Collection, `swsh9` ↔ `swsh9tg`, `cel25` ↔ `cel25cc`, `ecard2` a/b, 트레이너 키트(`tk-*`), `ru1`, `np`.

**처리 (2026-11-24 승인).**

1. TCGdex 작업은 연결 후보를 모아 실행 끝에 저장한다. 한 실행에서 둘 이상의 카드가 같은 (상품, 하위 유형)을 주면 저장된 연결을 바꾸지 않는다 (`duplicate_product`, 상품별 `tcgplayer duplicate product` 로그에 카드 목록).
2. TCGCSV 작업은 저장된 연결을 쓸 때 연결된 카드 번호와 상품 번호를 비교한다. 다르면 연결과 그 변형의 TCGCSV 가격(이번 실행에 이미 저장한 값은 남김)을 지우고, 번호 대조(방법 B)로 다시 연결한다. 1번 대조만 두면 이미 틀린 연결이 남고, 방법 B로 바로잡은 연결을 다음 날 TCGdex가 다시 덮어쓸 수 있어 함께 두었다.
3. 번호 정규화가 숫자+문자 번호의 앞자리 0을 떼지 않아(`050A` ≠ `50A`) 확인 단계에서 맞는 연결 2건(`ecard2-50a`, `ecard2-95b`)을 지웠다. 숫자로 시작하는 번호는 숫자 부분의 0을 떼도록 고쳤고, 방법 B에서도 `ecard2` a/b 카드 11행이 새로 연결되었다.

**축소 검증** (영어판 15세트 1,042장 + TCGCSV 그룹 13개: 중복이 나온 세트와 오탐 확인용 `svp`, 2026-11-24).

| 항목 | 결과 |
|------|------|
| 1회차 | `duplicate_product` 46, 지운 연결 37 (오탐 2건은 3번으로 수정), `link changed` 0 |
| 2회차 | 지운 연결 0, `link changed` 0, 행 수 불변 (`price_history` 117,735, `external_ids` 42,512, `card_variants` 53,446) |
| `ecard2` (정규화 수정 후 2회) | 1회차 11행 새로 연결, 2회차 모두 1번 단계에서 저장, 지운 연결 0 |
| 최종 연결 | 89381 → `bw11-6`, 89079 → `bw11-7`, 87775 → `ru1-3`, 84971 → `ecard2-74a`, 85813 → `ecard2-50a`, 87591 → `ecard2-95b` |
| `svp` 오탐 | 0 |

**남은 한계.**

- 번호도 같은 중복(트레이너 키트 `tk-sm-l-19` ↔ `tk-sm-r-19`처럼 한 TCGCSV 그룹에 두 세트가 섞임)은 저장된 연결을 그대로 둔다. 두 세트가 한 그룹이라 그룹 ↔ 세트 대응도 되지 않는다 (`group maps to several sets` 경고). 저장된 연결이 틀렸을 수 있다.
- 카드 한 장만 틀린 상품 ID를 주면 중복으로 잡히지 않는다. 이 경우 매일 TCGdex가 틀린 연결을 쓰고 TCGCSV가 번호 확인으로 지우는 일이 되풀이되며(`link changed`·`link number mismatch` 로그), 가격은 맞는 카드에 붙는다.
- 범위를 줄인 개발용 실행에서는 범위 밖 카드와의 중복을 알 수 없다. 번호 확인이 대부분을 걸러 낸다.
- `Lightning Energy (#3)`(TCGCSV 번호 `2`)처럼 TCGCSV 번호가 이름과 다른 상품은 번호 확인이 연결을 지운다. 원본끼리 맞지 않아 어느 쪽이 맞는지 알 수 없다.

### 원화 환산 조회 (5단계)

`ListPriceHistoryKRW`(`internal/db/queries/prices.sql`): 변형 1개·소스 1개의 기간 추이에 그 날짜의 환율을 붙여 돌려준다. #8이 그대로 쓰거나 바탕으로 삼는다.

- 환율은 `price_date` 이전 가장 가까운 값이다 (주말·공휴일·수집 누락일은 직전 값). `fx_rate_date`로 어느 날 환율인지 함께 준다.
- 첫 환율보다 이른 날, 소스 통화가 `KRW`인 소스(사용자·관리자 가격)는 `fx_rate_date`·`fx_rate`가 NULL이다. KRW 소스는 `price`를 그대로 쓴다.
- 원화 값은 호출 측에서 `decimal`로 `price × fx_rate`를 원 단위 반올림한다. SQL에서 계산하면 sqlc가 `pgtype.Numeric`으로 만들어 프로젝트 규칙(`decimal`)과 맞지 않아서 그렇게 했다.
- 실행 계획 (개발 DB, 환율 2년치 1,496행을 트랜잭션 안에 넣고 확인 후 롤백): 직전 날짜는 `fx_rates_pkey` 역순 Index Only Scan(`LIMIT 1`), 값은 같은 키 4열 전체로 Index Scan. 가격 행마다 인덱스 두 번이라 환율 행 수와 무관하다.
  - 처음에는 `LEFT JOIN fx_rates ... AND rate_date = (서브쿼리)` 형태였으나, 값 조회가 날짜 조건 없이 통화의 모든 환율 행을 훑어 `LATERAL`로 날짜를 먼저 구하게 고쳤다. `LEFT JOIN LATERAL`의 열은 sqlc가 NULL 불가로 만들어 값은 테이블 `LEFT JOIN`으로 가져온다.

### 수집기 연결과 운영 (5단계)

- `collect-daily`(05:30 KST)는 카탈로그 → 시세 순으로 실행한다. 카탈로그가 실패해도 시세는 이미 적재된 카드로 수집하고, 두 오류를 합쳐 돌려준다.
- 카탈로그와 시세가 TCGdex 클라이언트 하나를 같이 써서 요청 간격(250 ms)을 나눠 쓴다. 두 작업은 차례로 돌아 동시에 부르지 않는다.
- 시세 작업 셋(`fx`, `tcgcsv`, `tcgdex_cardmarket`)은 동시에 돈다. `tcgcsv`는 남은 행을 처리하기 전에 `tcgdex_cardmarket`이 끝나기를 기다린다.

| 환경 변수 (개발용) | 의미 |
|--------------------|------|
| `PRICES_SOURCES` | 실행할 작업 (`fx`, `tcgcsv`, `tcgdex_cardmarket`, 쉼표 구분). 비우면 전체, `none`이면 시세 수집을 하지 않는다 |
| `PRICES_TCGCSV_GROUPS` | TCGCSV 그룹 ID. 영어판 그룹은 이름 코드가 세트 ID와 달라 따로 지정한다 |
| `CATALOG_LANGUAGES`, `CATALOG_SETS` | 카탈로그와 같은 언어·세트 범위를 시세에도 쓴다 (`ko`는 시세 대상이 아니라 무시된다) |

- 별도 장비 실행: `scripts/collector.sh`가 `PRICES_*`를 넘기고, `status`에 소스별 최근 기준일·행 수와 환율을 보여 준다 (`mydocs/manual/collector_remote.md`).
- 로그: 작업마다 `price source start`·`price source done`(`fetched`, `linked`, `saved`, `skipped`, `failed`, `duration`). 연결하지 않은 이유는 `tcgcsv skipped rows`, `cardmarket skipped`, 그룹 ↔ 세트 대응은 `tcgcsv group set mapping`.

## 후속 타스크 인계

| 타스크 | 인계 사항 |
|--------|-----------|
| #7 국내 거래가·일본판 참고가 | 일본판 TCGplayer 가격은 TCGCSV `Pokemon Japan`(그룹 코드 대응)으로, Cardmarket 가격은 `-holo` 포함 TCGdex로 들어온다. 일본판 리버스(`SV2a` 등 패턴 상품)는 TCGCSV에 연결되지 않고 Cardmarket에만 있다. 구판 일본 카드(번호 없음, `PMCG` 등)는 두 소스 모두 가격이 없다 |
| #8 시세 조회 API | 원화 추이는 `ListPriceHistoryKRW`(변형·소스·기간)를 쓰거나 바탕으로 삼는다. 원화 값은 호출 측에서 `price × fx_rate`를 원 단위 반올림한다. `source_updated_at`으로 오래된 가격을 구분한다. 소스 기준일이 다르다 (TCGCSV는 UTC 전날, Cardmarket은 `updated` 날짜) |
| #8 | 한 카드의 변형별 가격은 소스마다 붙는 변형이 다를 수 있다 (Q9 첫날, 초판에는 Cardmarket 없음). 변형 단위로 보여 주고, 카드 대표가를 만들 때는 규칙을 정해야 한다 |
| #18 과거 이력 백필 | 연결은 `external_ids`를 그대로 쓰고, 상품 번호 확인(5단계) 규칙을 같이 적용한다 |
| #17 TCG Pocket | 시세 수집은 실행마다 TCGdex `tcgp` 시리즈로 빼고 있다. 카탈로그에서 빼면 이 처리는 지워도 된다 |
| 운영 | 매일 `tcgcsv skipped rows`의 `links_removed`, `tcgdex tcgplayer ids`의 `duplicate_product`, `price source done`의 `failed`를 본다. `links_removed`가 갑자기 늘면 번호 형식 변화(오탐)를 의심한다 |

## 참고 자료

- TCGCSV: https://tcgcsv.com/ , https://tcgcsv.com/docs , https://tcgcsv.com/last-updated.txt
- TCGdex: https://tcgdex.dev/markets-prices , https://tcgdex.dev/reference/card
- Frankfurter: https://frankfurter.dev/docs/ , https://api.frankfurter.dev/v2/providers
- 관련 문서: `price_data_sources.md`(#2), `db_schema.md` 6·8·10절(#4), `catalog_collector.md` 5·8절(#5)
