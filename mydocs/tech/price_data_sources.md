# 기술 조사: 시세 데이터 소스

> **작성일**: 2026-11-01
> **관련 타스크**: [#2](https://github.com/Chocobone/PokeCardPrice/issues/2) 시세 데이터 소스 조사
> **상태**: 1단계(카드·해외 시세 API), 2단계(환율) 작성. 소스 확정은 4단계에서 추가한다.

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

### 7. 환율 API 비교

달러(TCGplayer)와 유로(Cardmarket) 가격을 원화로 환산할 소스를 비교한다. 시작 후보 3개에 한국은행 ECOS를 추가했다.
공식 문서를 읽지 못해 블로그·외부 도구 문서에서만 찾은 내용은 `2차 자료`로 표시하며, 사실로 확정하지 않는다.

| 비교 항목 | Frankfurter v2 | open.er-api.com | 한국수출입은행 | 한국은행 ECOS |
|-----------|----------------|-----------------|----------------|---------------|
| KRW·USD·EUR | 지원 (응답 확인) | 지원 (응답 확인) | `미확인` (공식 문서 판독 불가) | `미확인` |
| 환율 성격 | 106개 기관 혼합 중간값. `providers=ecb`로 ECB 기준환율 선택 가능 (문서 확인) | 산출 방식 `미확인` | 매매기준율, 송금 환율 필드 (`2차 자료`) | `미확인` |
| 갱신 | 매일 (문서 확인) | 하루 1회. 호출 시점 다음 갱신 예정 시각 제공 (응답 확인) | 영업일 오전 11시 전후 (`2차 자료`). 포털 표기는 "실시간" | `미확인` |
| 주말 | 혼합값은 토·일 날짜 값 반환. ECB는 직전 영업일 값 반환 (응답 확인) | 일요일에도 갱신 (응답 확인) | 값 없음 또는 빈 응답 가능 (`2차 자료`) | `미확인` |
| 공휴일 | `미확인` | `미확인` | 값 없음 또는 빈 응답 가능 (`2차 자료`) | `미확인` |
| 과거 환율 | 날짜·기간 조회 가능, 1948년까지 안내 (응답·문서 확인) | 무료 경로 안내 없음 (`미확인`) | `searchdate`로 조회 (`2차 자료`) | 기간 조회 (`2차 자료`) |
| 호출 한도 | 일·월 한도 없음, 남용 방지 제한은 있으나 수치 미기재 (문서 확인) | 수치 미기재. 초과 시 429, 20분 뒤 해제 (문서 확인) | `미확인` | 월 10,000건 (`2차 자료`) |
| API 키 | 불필요 (응답 확인) | 불필요 (응답 확인) | 필요 (`2차 자료`) | 필요 (`2차 자료`) |
| 이용 조건 | 혼합값 무료 사용, 로컬 저장 권장. 개별 제공처 값은 각 제공처 조건 적용 (문서 확인) | **출처 링크 표기 필수, 재배포 금지.** 환율을 프로그램으로 제공하는 제품·서비스에 사용 금지 (문서 확인) | 무료, 이용허락범위 제한 없음 (포털 문서 확인) | `미확인` |
| 호출 결과 | 호출 완료 | 호출 완료 | 키가 없어 호출하지 않음 | 키가 없어 호출하지 않음 |

한국수출입은행은 기존 도메인(`www.koreaexim.go.kr`)이 2026-04-30 종료 예정으로 공지되어, 새 도메인(`oapi.koreaexim.go.kr`)을 써야 한다. 공식 안내 페이지는 빈 응답을 반환해 읽지 못했다.

**실제 호출 값 (2026-11-01 09:28~09:29 UTC, 일요일)**

| 소스 | 값 기준일 | USD→KRW | EUR→KRW |
|------|-----------|---------|---------|
| Frankfurter v2 혼합 | 2026-11-01 | 1,336.38 | 1,493.58 |
| Frankfurter v2 ECB | 2026-10-30 | 1,335.27 | 1,491.76 |
| open.er-api.com | 2026-11-01 00:02 UTC 갱신 | 1,334.80 | 1,491.57 |
| 최대 차이 | | 1.58 (0.12%) | 2.01 (0.13%) |

**과거 날짜 값 (2026-10-15)**

| 소스 | USD→KRW | EUR→KRW |
|------|---------|---------|
| Frankfurter v2 혼합 | 1,343.33 | 1,504.66 |
| Frankfurter ECB | 1,345.90 (v2 `providers=ecb`) | 1,508.48 (v1) |
| 차이 | 2.57 (0.19%) | 3.82 (0.25%) |

- open.er-api.com은 과거 값을 얻지 못했다. 시험한 경로(`/v6/history/USD/2026/10/15`)는 문서에 있는 경로가 아니라 추측한 경로이며 404였다.
- 후보 간 차이는 이번 표본에서 0.25% 이내였다. Charizard(987.13달러)에 대입하면 0.12% 차이는 약 1,560원이다.
- 한국수출입은행·ECOS 값은 호출하지 못해 이 비교에 포함되지 않았다.
- Frankfurter v1은 지원 중단(deprecated) 상태이지만 계속 제공된다고 안내한다. 응답 형식이 v2와 다르므로 구현은 v2 기준으로 한다.

## 결론

1차 소스 확정은 4단계에서 한다. 아래는 1단계 근거에 따른 잠정 판단이다.

1. **카드 카탈로그는 TCGdex가 유력하다.** 무키로 쓸 수 있고 pokemontcg.io보다 세트 수가 많으며(220개, 176개) 저장소 라이선스가 명시되어 있다. 다만 가격·이미지의 이용 조건과 호출 한도는 미확인이다.
2. **Cardmarket 시세는 TCGdex가 유력하다.** 전날 값이 갱신되어 있고, pokemontcg.io는 4개월 전 값에 머물러 있다. 다만 원본 대조는 하지 못했다.
3. **TCGplayer 시세는 TCGCSV가 유력한 후보이지만, 이용 조건 검토가 남았다.** TCGdex는 표본 전체가 `null`이고, pokemontcg.io는 값이 2주 이상 지났으며 신규 세트 가격이 없다.
4. **pokemontcg.io는 1차 소스로 부적합하다.** 지원 종료 공지, 신규 키 발급 중단, 오래된 가격, 잦은 서버 오류가 근거이다.
5. **Scrydex는 유료 대안으로 남긴다.** 이력까지 제공하지만 월 비용이 들고 Cardmarket이 없으며, 재배포 제한의 적용 범위를 검토해야 한다.
6. **한글판 카드는 해외 API로 해결되지 않는다.** 국내 거래가 소스 판단(3단계)과 함께 카드 식별 방법을 정해야 한다.
7. **환율은 Frankfurter v2 혼합값이 유력하다.** 키 없이 쓸 수 있고, 과거 날짜와 주말 값을 제공하며, 로컬 저장을 권장한다. 다만 호출 한도 수치와 공휴일 동작은 미확인이고, 혼합값의 산출 방식은 세부 확인이 필요하다.
8. **환율 대체 소스는 한국수출입은행이 유력하다.** 이용 조건이 가장 명확하지만 키 발급이 필요하고 공식 문서와 실제 응답을 확인하지 못했다. open.er-api.com은 출처 표기 의무와 재배포 금지 조건, 과거 값 부재로 1차·대체 소스에 부적합하다.

## 소스 확정안 (4단계, 2026-11-02)

판단 순서는 구현 계획서를 따랐다: 이용 조건 → 데이터 적합성 → 운영 가능성 → 구현 비용. 국내 거래가는 작업지시자가 권장안(A+B+C)을 채택했다(2026-11-02). 근거는 `domestic_price_sources.md`의 "포케폴리오 제외 및 대안 조사" 절이다.

| 항목 | 1차 소스 | 대체 소스 | 가격 성격 | 위험·조건 |
|------|----------|-----------|-----------|-----------|
| 카드 카탈로그 | TCGdex (`en`, `ja`, `ko`) | 없음 (pokemontcg.io는 지원 종료, Scrydex는 유료) | - | 무키, 저장소 MIT. 가격·이미지의 이용 조건과 호출 한도는 `미확인`. 한글판은 239장(2.51%)뿐이고 이미지 없음 |
| 해외 시세 (TCGplayer) | TCGCSV (`Pokemon`, `Pokemon Japan`) | Scrydex (유료, 필요 시) | 판매가 (`marketPrice` 등, USD) | 재공개 권한과 TCGplayer 약관 허용 여부는 `미확인`이나 작업지시자 결정으로 사용 (2026-11-02). 하루 1회 동기화, 요청 간 100ms, 전용 User-Agent. 사용자 요청마다 직접 호출 금지 |
| 해외 시세 (Cardmarket) | TCGdex `pricing.cardmarket` | 없음 | 판매가 (`avg`, `trend`, `avg7`·`avg30`, EUR) | 원본 대조 미실시. `variants_detailed`는 문서에 없는 필드라 변경 가능 |
| 환율 | Frankfurter v2 혼합값 | 한국수출입은행 (키 발급 필요) | - | 호출 한도 수치·공휴일 동작 `미확인`. 값 기준일·제공처를 함께 저장 |
| 국내 거래가 A | 일본판 참고가: 한글판과 세트 코드·번호가 같은 일본판의 TCGCSV(`Pokemon Japan`)·TCGdex(`ja`) 가격을 원화 환산 | - | 참고가 (일본판 환산) | 국내 실거래가가 아님을 화면에 표시. 세트 코드·번호 일치 범위 확인 필요 |
| 국내 거래가 B | 사용자 거래가 등록 | - | 사용자 등록가 | 외부 약관 무관. 초기 데이터 부족, 이상치·허위 입력 검증 필요 |
| 국내 거래가 C | 작업지시자 수동 입력 (시연 카드) | - | 관리자 입력가 | 자동 수집 아님. 출처·확인일을 함께 입력 |

**제외한 소스**

| 소스 | 사유 |
|------|------|
| pokemontcg.io | 지원 종료 공지, 신규 키 발급 중단, 오래된 가격 |
| open.er-api.com | 출처 표기 의무, 재배포 금지, 과거 값 없음 |
| 포케폴리오 | 작업지시자 결정 (화면 수집은 약관 금지) |
| 번개장터 | 작업지시자 결정 |
| 중고나라, KREAM | 약관이 무단 복제·이용 금지 |
| 네이버 쇼핑 검색 API | 종료 (2차 출처), NAVER API HUB에 없음 |
| Cardmarket 직접 수집 | `robots.txt`가 AI 크롤러 차단, 공개 API 없음 |

## 후속 타스크 설계 입력

| 타스크 | 넘기는 내용 |
|--------|-------------|
| **#4 DB 스키마** | **카드 언어** 컬럼 (`en`, `ja`, `ko`). **소스별 외부 ID 테이블**: TCGdex 카드 ID, TCGplayer `productId`+`subTypeName` 복합 키, Cardmarket 상품 ID. **가격 이력**: 카드 변형 단위, 소스, 통화(`USD`·`EUR`·`KRW`), 원 가격, 가격 필드 종류(`market`·`avg`·`trend` 등), 소스 갱신일, 수집일. **가격 성격 값**: `판매가`, `참고가(일본판 환산)`, `사용자 등록가`, `관리자 입력가`. **환율**: 통화쌍, 값, 값 기준일, 제공처. **사용자 거래가**: 사용자, 카드, 가격(KRW), 거래일, 상태 등급, 검증 상태. 판매자 정보 컬럼은 두지 않는다 |
| **#5 카탈로그 수집기** | TCGdex `/v2/{en,ja,ko}/sets`, `/cards`. 기본은 전체 반환이며 필요 시 `pagination:page`·`pagination:itemsPerPage`. 하루 1회 또는 신규 세트 공지 시 갱신. 한글판은 `ko`에 있으면 사용하고, 없으면 같은 세트 코드·번호의 `ja` 카드에 연결해 한글판 존재만 표시한다. 영어판과 일본·한글판은 ID 체계가 달라 ID 일치로 연결하지 않는다. **세트 코드·번호 일치 범위를 표본 세트로 먼저 검증** |
| **#6 해외 시세 수집기** | TCGCSV: 하루 1회(20:00 UTC 갱신 이후), 요청 간 100ms 이상, 동시성 1, 전용 User-Agent, 24시간 10,000회 미만. TCGdex Cardmarket: 하루 1회, 카드 단위 조회 시 요청 간격을 두고 동시성 상한 2. 환율은 시세 수집 직후 Frankfurter v2로 1회 조회해 같은 기준일로 저장하고, 값이 없으면 직전 값을 사용. 소스 갱신일이 오래된 가격은 현재가로 표시하지 않음. 초기 이력은 TCGCSV 일별 압축 파일(2024-02-08~) 사용 가능 여부를 이용 조건 판단 후 결정 |
| **#7 국내 거래가 산출 및 입력** | 외부 사이트 수집기는 만들지 않는다. **(A)** 한글판 카드 → 일본판 카드 연결 후, #6이 저장한 일본판 가격을 원화로 환산해 `참고가(일본판 환산)`로 저장. **(B)** 사용자 거래가 등록·집계: 상태 등급 구분, 이상치 제거(중앙값 기준 등), 최소 건수 미만이면 시세로 표시하지 않음. **(C)** 관리자 입력: 시연 카드의 가격·출처·확인일 입력. 사용자 거래가 등록 API는 #10이 담당한다. 이슈 #7은 이 범위로 제목·본문을 수정했다 (2026-11-02) |

**영향을 받는 다른 타스크**

- #8 카드 검색·시세 조회 API: 가격 성격별로 구분해 응답하고, 참고가에는 "일본판 기준" 표시 정보를 포함한다.
- #10 내 컬렉션 API: 사용자 거래가 등록·수정·삭제 API를 추가했다 (2026-11-02 작업지시자 결정, 이슈 본문 수정).
- #13 카드 상세 화면: 국내 시세 영역에 가격 성격과 출처를 표시하고, 사용자 거래가 입력 양식이 필요하다.
- 화면에 출처 표기: TCGplayer(TCGCSV 경유), Cardmarket(TCGdex 경유), 환율(Frankfurter).

**미해결 사항**

| 사항 | 담당 |
|------|------|
| TCGCSV 재공개 권한·TCGplayer 약관 허용 여부 | 작업지시자가 사용 결정 (2026-11-02). 공개 배포 전 재확인 권장 |
| TCGdex 가격·이미지 이용 조건, 호출 한도 | #5, #6 구현 전 재확인 |
| TCGdex `pricing.tcgplayer`가 모두 `null`인 원인 | #6 (TCGCSV 사용 시 영향 없음) |
| 한글판·일본판 세트 코드·번호 일치 범위 | #5 |
| 한글판 카드 이름 (TCGdex `ko` 미수록 카드) | #5 |
| Frankfurter 호출 한도·공휴일 동작 | #6 |

## 참고 자료

- TCGdex 문서: https://tcgdex.dev/ , https://tcgdex.dev/markets-prices , https://tcgdex.dev/reference/card , https://tcgdex.dev/reference/set , https://tcgdex.dev/rest/filtering-sorting-pagination , https://tcgdex.dev/status , https://tcgdex.dev/assets
- TCGdex 진행 현황: https://api.tcgdex.net/status
- TCGdex 저장소·라이선스: https://github.com/tcgdex/cards-database , https://raw.githubusercontent.com/tcgdex/cards-database/master/LICENSE
- pokemontcg.io 문서: https://docs.pokemontcg.io/ , https://docs.pokemontcg.io/getting-started/rate-limits , https://docs.pokemontcg.io/getting-started/authentication , https://docs.pokemontcg.io/api-reference/cards/card-object , https://docs.pokemontcg.io/api-reference/cards/search-cards
- Scrydex: https://scrydex.com/pricing , https://scrydex.com/terms , https://scrydex.com/docs/getting-started/prices , https://scrydex.com/docs/getting-started/best-practices , https://scrydex.com/docs/pokemon/price-history
- TCGCSV: https://tcgcsv.com/ , https://tcgcsv.com/faq , https://tcgcsv.com/docs
- Frankfurter: https://frankfurter.dev/ , https://frankfurter.dev/docs/ , https://api.frankfurter.dev/v2/providers
- ExchangeRate-API: https://www.exchangerate-api.com/docs/free , https://www.exchangerate-api.com/terms
- 한국수출입은행 공공데이터포털 안내: https://www.data.go.kr/data/3068846/openapi.do
- 2차 자료 (공식 문서 아님): https://kora1492.tistory.com/79 , https://homey.app/ko-kr/app/com.yuji.exchange/한국-환율/ , https://clawhub.ai/chloepark85/korea-eximbank-exchange-cli , https://bcuts.tistory.com/293 , https://pypi.org/project/ecos-api/
