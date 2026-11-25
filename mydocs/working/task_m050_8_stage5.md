# Task #8: 카드 검색·시세 조회 API — 5단계 완료 보고서

## 단계 목표

프론트엔드를 실제 API로 전환하고, 개발 DB에서 응답 시간을 실측하고, 문서를 정리한다.

## 변경 파일

| 파일 | 내용 |
|------|------|
| `src/frontend/src/api/types.ts` | `Currency` = `USD`·`EUR`·`KRW`(JPY 제거), `Price`에 `krw`·`fxDate`·`updatedAt`, `Period`·`SetDetail`·`VariantPrices`·`CardDetail`·`PricePoint`·`PriceSeries`·`PriceHistoryResponse` 추가 (#13 대비) |
| `src/frontend/src/api/cards.ts` | `getCard`, `getCardPrices` 추가 |
| `src/frontend/src/api/mock/handlers.ts` | 통화·소스를 실제 API처럼(영어판 USD `tcgcsv`, 일본판 EUR `tcgdex_cardmarket`, 한글판 대표가 `null`), 원화 필드, 상세·추이 목업 추가 |
| `src/frontend/src/utils/format.ts` | 통화 소수 자릿수 `EUR: 2` (JPY 제거) |
| `src/frontend/.env.development` | `VITE_API_MOCK=false` |
| `mydocs/tech/api_contract.md` | `page` 1~10000, 목업 절(기본값·흉내 범위·호출 수), 적용 방향 |
| `mydocs/tech/frontend_structure.md` | API 함수 목록, 목업 기본값 `false` |
| `mydocs/tech/db_schema.md` 7절 | #8 쿼리 4종의 인덱스·측정값, 인덱스 변경(`00005`) |
| `CLAUDE.md` | 카드 API 목록·패키지 위치·오류 형식 한 줄, 목업 기본값 |

화면(`.vue`)은 바꾸지 않았다. 가격 표시는 지금처럼 소스 통화(`$0.49`, `€395.10`)이고, 원화는 응답에만 있다(아래 확인 사항 F1).

## 검증

| 항목 | 결과 |
|------|------|
| 프론트엔드 `npm run typecheck`·`lint`·`build` | 통과 (`noUncheckedIndexedAccess` 지적 1건은 목업에서 고침) |
| 백엔드 `go vet`·`go test ./...`·`go build` | 통과 (4단계 이후 백엔드 변경 없음) |

**화면 확인** (헤드리스 Chromium, Playwright 1.49 컨테이너, 실제 API, #12와 같은 방식)

| 화면 | 폭 | API 호출 | 결과 |
|------|----|----------|------|
| 홈 `/` | 1280 | `/api/rankings/movers` 상승·하락 200 | "2026년 11월 23일 기준", 급등·급락 모두 "시세 데이터가 쌓이면 표시됩니다" (비교일 없음) |
| 검색 `피카츄` | 1280 | `/api/cards` 200 | 348장, 카드 이미지·한글 종 이름·`$0.49` 등 |
| 검색 `리자몽` 일본판 | 360 | `/api/cards?lang=ja` 200 | 45장, `$131.98`(TCGCSV)·`€395.10`(Cardmarket) 함께 표시 |
| 검색 결과 없음 | 360 | 200 | 빈 상태 안내 |

모든 화면에서 콘솔 오류·가로 스크롤이 없었다. 캡처와 스크립트는 세션 임시 폴더에만 두었다.

**응답 시간 실측** (개발 DB, 엔드포인트마다 100회, 서버 로그의 처리 시간, 단위 ms)

| 요청 | 횟수 | p50 | p95 | 최대 |
|------|-----:|----:|----:|-----:|
| `/cards?q=pikachu` | 100 | 12.2 | 17.7 | 22.2 |
| `/cards?q=피카츄` | 100 | 11.9 | 16.8 | 20.9 |
| `/cards?q=sv03.5 25` | 100 | 9.8 | 15.4 | 22.1 |
| `/cards?q=ピカ&lang=ja` (2자) | 100 | 26.1 | 36.1 | 48.9 |
| `/cards?q=charizard ex&page=2` | 100 | 11.4 | 17.9 | 49.7 |
| `/cards/16970` (상세) | 100 | 1.9 | 3.1 | 13.7 |
| `/cards/36319` (한글판 상세, 연결 1장) | 100 | 2.4 | 3.1 | 4.3 |
| `/cards/16970/prices` (30d) | 100 | 1.2 | 2.2 | 3.9 |
| `/cards/16970/prices?period=1y` | 100 | 1.2 | 1.9 | 2.5 |
| `/rankings/movers` 7d 상승 | 100 | 0.9 | 1.1 | 1.9 |
| `/rankings/movers` 30d 하락 | 100 | 0.9 | 1.3 | 1.7 |

- Q6 목표(검색·상세·추이 p95 100 ms, 순위 300 ms)를 모두 만족한다.
- 개발 DB 이력은 3일치라 추이·순위는 작게 나온다. 1년치 규모 값은 4단계 측정(순위 p50 67~79 ms·p95 최대 247 ms, 1년 추이 p50 45 ms·p95 97 ms)을 기준으로 본다.
- 개발 DB의 검색 시간은 1단계 SQL 측정과 같은 수준이다(인덱스 적용 후).

## 확인 사항 (2026-11-25 작업지시자 결정: F1 제안대로)

| # | 사항 | 제안 | 대안 |
|---|------|------|------|
| F1 | 홈·검색 카드에 원화를 함께 보일지 | **이 타스크에서는 바꾸지 않는다.** 원화 표시 방식(병기·전환·환율 기준일 안내)은 #13 카드 상세 화면에서 함께 정한다 | 지금 카드 타일·순위 줄에 `≈ ₩655` 병기 |

## 발견 사항

- 검색 `피카츄`의 첫 쪽은 최신 세트(`30th Celebration`, 영어판) 피카츄가 차지한다. 모든 피카츄 카드가 같은 순위(②)라 발매일 최신순이 되기 때문이다(Q4 규칙대로). 언어 필터로 좁힐 수 있다.
- 사이트 꼬리말 출처에 가격 출처(TCGplayer·Cardmarket)가 없다. 실제 가격을 보이기 시작했으므로 공개 배포 전에 출처·이용 조건 표기를 넣어야 한다(#16, #6 이용 조건 재확인과 함께).
- `.env.development` 변경은 Vite HMR로 반영되지 않아 프론트엔드 컨테이너를 다시 시작했다(`frontend_structure.md`에 적음).

## 다음 단계

최종 결과보고서 작성.
