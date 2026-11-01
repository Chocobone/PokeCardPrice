# 기술 조사: 시세 데이터 소스

> **작성일**: 2026-11-01
> **관련 타스크**: [#2](https://github.com/Chocobone/PokeCardPrice/issues/2) 시세 데이터 소스 조사
> **상태**: 1단계(카드·해외 시세 API) 작성. 환율은 2단계, 소스 확정은 4단계에서 추가한다.

---

## 배경

카드 카탈로그와 해외 시세를 수집할 소스를 정하기 위해 계획서에 있는 TCGdex와 pokemontcg.io를 같은 기준으로 비교한다.
조사 중 pokemontcg.io의 지원 종료와 TCGdex의 TCGplayer 가격 누락을 확인해, 대체 후보인 Scrydex와 TCGCSV도 추가로 확인했다.

- 확인일은 별도 표시가 없으면 모두 2026-11-01(UTC)이다.
- 확인 상태: `응답 확인`, `문서 확인`, `미확인`, `확인 불가`

## 조사 내용

### 1. TCGdex와 pokemontcg.io 비교

| 비교 항목 | TCGdex | pokemontcg.io | 상태 |
|-----------|--------|---------------|------|
| 카드 필드 | `id`, `localId`, `name`, `image`, `rarity`, `set`, `dexId`, `variants`, `variants_detailed` 등 | `id`, `number`, `name`, `images`, `rarity`, `set`, `nationalPokedexNumbers`, `tcgplayer`, `cardmarket` 등 | 응답 확인 |
| 세트 수 | 영어 220개 (`/v2/en/sets`) | 176개 (`totalCount`) | 응답 확인 |
| 페이지네이션 | `pagination:page`, `pagination:itemsPerPage`. 기본은 전체 반환 | `page`, `pageSize` (최대 250) | 문서 확인 |
| TCGplayer 가격 | 필드는 있으나 **표본 8종 모두 `null`** | `low`, `mid`, `high`, `market`, `directLow` (USD). 신규 세트 표본은 URL만 있고 가격 없음 | 응답 확인 |
| Cardmarket 가격 | `avg`, `low`, `trend`, `avg1`, `avg7`, `avg30`, `-holo` 계열 (EUR) | `averageSellPrice`, `lowPrice`, `trendPrice`, `avg1`, `avg7`, `avg30` 등 (EUR). 신규 세트 표본은 객체 없음 | 응답 확인 |
| 가격 갱신일 | Cardmarket 2026-10-31 (문서상 매일) | TCGplayer 2026/10/15, Cardmarket **2026/07/01** | 응답 확인 |
| 과거 가격 이력 | 없음. 1·7·30일 평균 필드만 제공 | 없음. 1·7·30일 평균 필드만 제공 | 문서 확인 |
| 호출 한도 | 공식 문서에 기재 없음 | 키 없음: 1,000회/일, 30회/분. 키 있음: 20,000회/일 | TCGdex `미확인`, pokemontcg.io 문서 확인 |
| API 키 | 문서에 요구 사항 없고 무키 호출 성공 | 무키 호출 가능. **신규 가입 중단** | 응답 확인, 문서 확인 |
| 이용 조건 | 데이터베이스 저장소 MIT 라이선스. 가격 데이터와 이미지에 적용되는지는 기재 없음 | 라이선스·저장·재배포 규정을 찾지 못함 | 문서 확인, `미확인` |
| 한글판 | 카드 239장 / 9,521장 (2.51%), 이미지 없음 | 언어 지원 기재 없음. 한글 이름 검색은 HTTP 400 | 응답 확인, `미확인` |
| 운영 상태 | 활동 중인 공개 저장소, 상태 페이지 운영 | **지원 종료 공지**. 표본 호출 7회 중 3회 500/502 | 문서 확인, 응답 확인 |

### 2. 샘플 카드 응답 비교

호출 시각은 2026-11-01 09:17~09:25 UTC이다. 세 소스의 TCGplayer 값은 TCGdex의 `thirdParty.tcgplayer` 상품 ID로 연결했다.

| 카드 | TCGdex ID | pokemontcg.io ID | TCGplayer 상품 ID |
|------|-----------|------------------|-------------------|
| 최신 세트: Lurantis ex (Pitch Black) | `me05-004` | `me5-4` | 704761 |
| 구세트: Charizard (Base Set) | `base1-4` | `base1-4` | 42382 |
| 한글판 존재 세트: Bulbasaur (151) | `sv03.5-001` | `sv3pt5-1` | 502552 |

**TCGplayer 시장가 (USD)**

| 카드 (변형) | TCGdex | pokemontcg.io (갱신 10/15) | TCGCSV (갱신 10/31) |
|-------------|--------|----------------------------|---------------------|
| Lurantis ex (Holofoil) | `null` | 가격 없음 | 0.48 |
| Charizard (Holofoil) | `null` | 954.65 | 987.13 |
| Bulbasaur (Normal) | `null` | 0.29 | 0.20 |
| Bulbasaur (Reverse Holofoil) | `null` | 0.67 | 0.37 |

**Cardmarket 평균 판매가 (EUR)**

| 카드 | TCGdex `avg` (갱신 10/31) | pokemontcg.io `averageSellPrice` (갱신 07/01) | 차이 |
|------|---------------------------|-----------------------------------------------|------|
| Lurantis ex | 0.43 | 가격 없음 | - |
| Charizard | 458.10 | 1,531.00 | 약 3.3배 |
| Bulbasaur | 0.13 | 79.89 | 약 615배 |

- pokemontcg.io의 Bulbasaur 값은 일반 카드 시세로 보기 어렵다. TCGdex에는 같은 카드의 도장(`set-logo`) 변형이 평균 67.51로 따로 있어, pokemontcg.io가 다른 상품과 연결했을 가능성이 있다. 이는 추정이며 확인하지 못했다.
- Charizard의 차이는 4개월의 시점 차이, 상품 연결 차이, 또는 거래량이 적은 카드의 변동일 수 있다. 원인은 확인하지 못했다.

**원본 사이트 대조**

| 사이트 | 결과 | 상태 |
|--------|------|------|
| TCGplayer | 상품 페이지 3건 모두 제목만 반환(클라이언트 렌더링). 가격을 읽지 못함 | `미확인` |
| Cardmarket | `robots.txt`가 ClaudeBot 등 AI 크롤러를 전면 차단하므로 상품 페이지를 조회하지 않음 | `확인 불가` |

두 사이트 모두 원본 값과 대조하지 못했다. 위 값들은 소스 간 비교일 뿐이며, 어느 쪽이 정확한지는 입증되지 않았다.

### 3. TCGdex 응답에서 확인한 사항

문서에 없거나 문서와 다른 동작이 있어 수집기 구현 전에 알아야 한다.

**TCGplayer 가격 누락**

- 영어 카드 표본 8종(`base1-4`, `base1-2`, `me05-004`, `me01-001`, `sv08-001`, `sv03.5-001`, `sv01-001`, `swsh3-136`) 모두 `pricing.tcgplayer`가 `null`이었다.
- 공식 문서는 TCGplayer 가격을 "Hourly to daily"로 갱신한다고 설명한다. 일시적 장애인지 중단인지는 `미확인`이다.

**`variants_detailed` (공식 문서에 없음)**

```json
{
  "type": "normal",
  "thirdParty": { "cardmarket": 733596, "tcgplayer": 502552, "cardtrader": 261065 },
  "variantId": "...",
  "pricing": { "cardmarket": { "updated": "2026-10-31T10:29:52.912Z", "unit": "EUR", "avg": 0.13 }, "tcgplayer": null }
}
```

- 변형별 외부 상품 ID를 제공해, TCGplayer·Cardmarket 데이터를 카드와 연결하는 데 쓸 수 있다.
- 문서에 없는 필드라 예고 없이 바뀔 수 있고, `sv01-001`처럼 값이 없는 카드도 있다.
- 변형을 구분하는 신뢰도에 한계가 있다. `base1-4`의 초판 shadowless와 일반 shadowless는 같은 상품 ID(660224)와 같은 가격을 반환했다.
- `pricing.tcgplayer`의 변형 키도 문서끼리 다르다. 시세 안내 페이지는 `normal`, `reverse`, `holo`이고, 레퍼런스는 `holofoil`, `reverse-holofoil` 등이다. 값이 모두 `null`이어서 실제 키는 `미확인`이다.

### 4. 한글판 지원

| 항목 | 결과 | 상태 |
|------|------|------|
| TCGdex 한글 세트 | 95개, 카드 수 합계 7,933장 | 응답 확인 |
| TCGdex 한글 카드 | 239장 / 9,521장 (2.51%). 일본어는 12,781장 / 18,031장 (70.88%) | 응답 확인 |
| 세트 메타데이터만 있는 경우 | `ko/sets/SV2a`(포켓몬 카드 151)는 세트 정보만 있고 `cards`가 비어 있음. `ko/cards/SV2a-001`은 404 | 응답 확인 |
| 한글 이미지 | `ko/cards/SV5K-001`에 `image` 없음 | 응답 확인 (표본 1종) |
| 언어 간 연결 | `SV5K-001`은 한글·일본어 ID가 같음. 영어판은 체계가 다름 (`SV2a` ↔ `sv03.5`) | 응답 확인 (표본), 공식 보장 없음 |
| 한글 카드의 가격 | `ko/SV5K-001`과 `ja/SV5K-001`의 Cardmarket 상품 ID(752743)와 가격 필드가 같음 | 응답 확인 (표본 1종) |
| TCGCSV | 카테고리는 `Pokemon`, `Pokemon Japan`뿐이며 한글판 없음 | 응답 확인 |

- **한글판 카탈로그를 해외 API만으로 구성할 수 없다.** 한글판 수록 카드 대부분이 TCGdex에 없고, 나머지 두 소스도 한글판을 제공하지 않는다.
- **TCGdex 한글 카드의 가격은 국내 시세가 아니다.** 표본에서는 일본판과 같은 유럽 시장 상품의 가격이었다.
- 한글판의 세트 코드와 번호가 일본판과 같다면 일본판 데이터로 보완할 수 있다. 일치 범위는 3단계와 #5에서 추가로 확인해야 한다.

### 5. 추가 후보

#### pokemontcg.io의 지원 종료

문서 전체에 다음 공지가 있다(`문서 확인`).

> The Pokémon TCG API is deprecated. New account registrations are no longer available. Existing API keys will continue to function through March 1, 2027.

인증 안내 페이지에는 무료 가입 안내가 남아 있으나, 지원 종료 공지가 최신 내용으로 보인다. 2027-03-01 이후 무키 호출이 유지되는지는 `미확인`이다.

#### Scrydex (pokemontcg.io의 후속 서비스)

| 항목 | 내용 | 상태 |
|------|------|------|
| 요금 | 무료 플랜 안내 없음. 최저 Starter $29/월 (5,000 크레딧), 초과 크레딧당 $0.006 | 문서 확인 |
| 학생·비상업 요금 | 찾지 못함 | `미확인` |
| 가격 이력 | 제공 (유료 플랜 전체, 요청당 3크레딧) | 문서 확인 |
| 가격 범위 | TCGplayer 판매처 정보 포함. 시세 산출 출처는 명시되지 않음. Cardmarket은 찾지 못함, 유로 시장은 "future update" | 문서 확인 |
| 언어 | 일본어 지원 (JPY). 한글판은 찾지 못함 | 문서 확인 |
| 데이터 보관 | 캐시와 로컬 저장 권장. 재판매·재배포는 서면 허가 필요 | 문서 확인 |

호출하려면 유료 계정이 필요해 응답은 확인하지 않았다.

#### TCGCSV (TCGplayer 가격 일별 공개본)

| 항목 | 내용 | 상태 |
|------|------|------|
| 데이터 | TCGplayer API에서 가져온 상품·가격. 필드는 `lowPrice`, `midPrice`, `highPrice`, `marketPrice`, `directLowPrice`, `subTypeName` (USD) | 응답 확인 |
| 갱신 | 매일 약 20:00 UTC. 확인 시점 최종 갱신 2026-10-31T20:05:59Z | 응답 확인 |
| 키 | 불필요 | 응답 확인 |
| 과거 가격 | 2024-02-08부터 일별 압축 파일 제공 | 문서 확인 (파일은 미호출) |
| 호출 규칙 | 하루 1회 동기화, 요청 간 100ms, 24시간 10,000회 초과 시 차단 가능, 사용자 대면 앱에서 직접 호출 금지, 전용 User-Agent 설정 | 문서 확인 |
| 이용 조건 | "Permission is implicit", 표기 의무 없음. 명시적 라이선스, 재배포·상업적 이용 규정은 찾지 못함 | 문서 확인, `미확인` |
| 운영 주체 | 1인 취미 프로젝트 ("a hobby and a one-man show"). TCGplayer와의 공식 제휴 언급 없음 | 문서 확인 |
| 키 구조 | `productId`만으로는 고유하지 않음. `subTypeName`과 복합 키 필요 | 문서 확인 |

TCGCSV가 TCGplayer 데이터를 재공개할 권한이 있는지, TCGplayer 약관이 이 사용을 허용하는지는 확인하지 못했다. 4단계 이용 조건 판단에서 검토한다.

### 6. robots.txt 확인 결과

| 사이트 | 결과 |
|--------|------|
| `api.tcgdex.net` | `Disallow: /`. 단 주석에 크롤러 대상이며 API 사용은 가능하다고 명시 |
| `tcgdex.dev` | `/api`, `/js`, 쿼리 문자열 경로 금지 |
| `api.pokemontcg.io`, `docs.pokemontcg.io`, `pokemontcg.io` | 주석만 있고 차단 규칙 없음 |
| `scrydex.com`, `tcgcsv.com` | 전체 허용 |
| `tcgplayer.com` | 상품 경로 허용, `Crawl-Delay: 10` |
| `cardmarket.com` | ClaudeBot, GPTBot 등 AI 크롤러 전면 차단 |

## 결론

1차 소스 확정은 4단계에서 한다. 아래는 1단계 근거에 따른 잠정 판단이다.

1. **카드 카탈로그는 TCGdex가 유력하다.** 무키로 쓸 수 있고 pokemontcg.io보다 세트 수가 많으며(220개, 176개) 저장소 라이선스가 명시되어 있다. 다만 가격·이미지의 이용 조건과 호출 한도는 미확인이다.
2. **Cardmarket 시세는 TCGdex가 유력하다.** 전날 값이 갱신되어 있고, pokemontcg.io는 4개월 전 값에 머물러 있다. 다만 원본 대조는 하지 못했다.
3. **TCGplayer 시세는 TCGCSV가 유력한 후보이지만, 이용 조건 검토가 남았다.** TCGdex는 표본 전체가 `null`이고, pokemontcg.io는 값이 2주 이상 지났으며 신규 세트 가격이 없다.
4. **pokemontcg.io는 1차 소스로 부적합하다.** 지원 종료 공지, 신규 키 발급 중단, 오래된 가격, 잦은 서버 오류가 근거이다.
5. **Scrydex는 유료 대안으로 남긴다.** 이력까지 제공하지만 월 비용이 들고 Cardmarket이 없으며, 재배포 제한의 적용 범위를 검토해야 한다.
6. **한글판 카드는 해외 API로 해결되지 않는다.** 국내 거래가 소스 판단(3단계)과 함께 카드 식별 방법을 정해야 한다.

## 적용 방향 (4단계에서 확정)

- **#4 DB 스키마**: 소스별 외부 상품 ID 저장 구조가 필요하다. 가격은 카드가 아니라 변형 단위로 저장하고, 카드 언어를 구분한다.
- **#5 카탈로그 수집기**: 언어별 ID 체계가 다르므로, 영어판과 일본·한글판의 연결을 ID 일치로 가정하지 않는다.
- **#6 해외 시세 수집기**: 소스별 갱신일을 저장해 오래된 가격을 현재가로 표시하지 않는다. 상품 연결 오류를 잡기 위한 이상치 점검이 필요하다.
- **초기 이력**: TCGCSV 압축 파일을 쓸 수 있으면 TCGplayer 이력을 2024-02-08까지 소급할 수 있다. 사용 여부는 이용 조건 검토 후 결정한다.

## 참고 자료

- TCGdex 문서: https://tcgdex.dev/ , https://tcgdex.dev/markets-prices , https://tcgdex.dev/reference/card , https://tcgdex.dev/reference/set , https://tcgdex.dev/rest/filtering-sorting-pagination , https://tcgdex.dev/status , https://tcgdex.dev/assets
- TCGdex 진행 현황: https://api.tcgdex.net/status
- TCGdex 저장소·라이선스: https://github.com/tcgdex/cards-database , https://raw.githubusercontent.com/tcgdex/cards-database/master/LICENSE
- pokemontcg.io 문서: https://docs.pokemontcg.io/ , https://docs.pokemontcg.io/getting-started/rate-limits , https://docs.pokemontcg.io/getting-started/authentication , https://docs.pokemontcg.io/api-reference/cards/card-object , https://docs.pokemontcg.io/api-reference/cards/search-cards
- Scrydex: https://scrydex.com/pricing , https://scrydex.com/terms , https://scrydex.com/docs/getting-started/prices , https://scrydex.com/docs/getting-started/best-practices , https://scrydex.com/docs/pokemon/price-history
- TCGCSV: https://tcgcsv.com/ , https://tcgcsv.com/faq , https://tcgcsv.com/docs
