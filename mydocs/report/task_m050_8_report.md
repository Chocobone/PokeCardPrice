# Task #8 최종 결과보고서: 카드 검색·시세 조회 API

> **작성일**: 2026-11-25
> **브랜치**: `local/task8`
> **마일스톤**: M050 (v0.5.0)

## 개요

#5 카탈로그와 #6 해외 시세를 읽어 카드 검색, 카드 상세, 시세 추이, 급등·급락 순위를 HTTP API로 제공한다. #12에서 정한 API 계약을 확정하고, 프론트엔드를 목업에서 실제 API로 전환했다.

| 엔드포인트 | 내용 |
|------------|------|
| `GET /cards?q=&lang=&page=&size=` | 이름(영·일·한)·한글 종 이름·세트 이름·세트 ID·번호(앞 0 무시)로 낱말 AND 검색, 카드별 대표가 |
| `GET /cards/{id}` | 카드·세트 정보, 한글판 ↔ 일본판 연결 카드, 변형 × 소스별 최신가 |
| `GET /cards/{id}/prices?period=7d\|30d\|90d\|1y` | 변형 × 소스별 날짜 계열과 요약(현재가·평균·최저·최고·변동률) |
| `GET /rankings/movers?period=7d\|30d&direction=up\|down&limit=` | TCGCSV 변형 단위 변동률 순위 ($1 이상, 카드당 1개) |

모든 가격은 소스 통화 금액(문자열)에 원화 환산값(`krw`, 직전 환율, 원 단위 반올림)을 함께 보내고, 오류는 `{"error": {"code", "message"}}`로 통일했다.

## 수행 결과

| 단계 | 내용 | 커밋 |
|------|------|------|
| 계획 | 수행 계획서(Q1~Q6), 구현 계획서 (1단계 커밋에 포함) | `a88ba44` |
| 1단계 | 계약 확정(상세·추이 엔드포인트, 원화 필드, EUR), 쿼리 측정(개발 DB, 1년치 복제 DB `poc_task8`), 검색 인덱스·순위 쿼리 형태 결정, S1~S3 | `a88ba44` |
| 2단계 | 계약 오류 형식, 매개변수 검증, 금액·원화 변환, 대표가 선택(Q1), `GET /cards`, 마이그레이션 `00005` | `89a1346` |
| 3단계 | `GET /cards/{id}`, `GET /cards/{id}/prices`, 기간 요약(Q5) | `472a8e6` |
| 4단계 | `GET /rankings/movers`(Q3), 1년치 규모에서 API 서버 측정, `poc_task8` 삭제 | `ff520f3` |
| 5단계 | 프론트엔드 실제 API 전환(`types.ts`·목업·`VITE_API_MOCK=false`), 화면 확인, 응답 시간 실측, 문서, F1 | `af48c2e` |

## 산출물

### 코드

| 위치 | 내용 |
|------|------|
| `src/backend/internal/server/` | `cards.go`(검색·`CardSummary` 조립), `detail.go`(상세·추이), `rankings.go`(순위), `errors.go`(계약 오류 형식), `server.go`(라우트) |
| `src/backend/internal/api/` (신규) | 응답 타입, `ToKRW`·`NewPrice`, `PickRepresentative`(Q1), `Summarize`·`ChangeRate`(Q5), 기간·순위 상수 |
| `src/backend/internal/db/queries/cards.sql` (신규) | `SearchCards`, `ListLatestPrices`, `GetCardDetail`, `ListLinkedCards`, `ListCardPriceHistory`, `GetLatestPriceDate`, `ListMovers` (+ 생성 코드) |
| `src/backend/migrations/00005_search_indexes.sql` | `cards_name_lower_trgm_idx`, `cards_local_number_idx` 추가, `cards_name_trgm_idx` 삭제 |
| `src/frontend/src/api/` | `types.ts` 확정 계약, `cards.ts`에 `getCard`·`getCardPrices`, 목업 갱신 |
| `src/frontend/.env.development` | `VITE_API_MOCK=false` |

### 문서

- `mydocs/tech/api_contract.md`: 확정 계약 (규칙·타입·오류·목업).
- 갱신: `CLAUDE.md`(카드 API 한 줄), `mydocs/tech/frontend_structure.md`, `mydocs/tech/db_schema.md` 7절(#8 쿼리 측정값).

### 사용 방법

```bash
# Docker/ 에서
curl "localhost:8080/cards?q=sv03.5+25"
curl "localhost:8080/cards/16970"
curl "localhost:8080/cards/16970/prices?period=1y"
curl "localhost:8080/rankings/movers?period=7d&direction=up&limit=10"
# 프론트엔드는 /api/* 로 같은 경로를 부른다 (http://localhost:5173)
# 백엔드 없이 화면만 볼 때: src/frontend/.env.development 의 VITE_API_MOCK=true 후 docker compose restart frontend
```

### 작업지시자 결정 사항

| # | 결정 |
|---|------|
| Q1 | 대표가: TCGCSV → Cardmarket, 변형 `normal` → `holofoil` → … 순. 한글판 `null` |
| Q2 | `Price`에 `krw`·`fxDate`(·`updatedAt`) 추가, 통화에 `EUR` |
| Q3 | 순위: TCGCSV 변형 단위, 두 날짜 모두 $1 이상, 카드당 1개 |
| Q4 | 검색 정렬: 세트 ID·번호 일치 → 이름 시작·종 이름 일치 → 나머지, 같은 순위는 발매일 최신순 |
| Q5 | 추이 요약: 기간 안 첫·마지막 가격으로 변동률, 산술 평균, 이력이 짧으면 있는 만큼 |
| Q6 | 집계 테이블 없이 조회 시 계산, 목표 p95 100 ms(순위 300 ms), 캐시는 #11 |
| S1 | 대소문자 구분 이름 인덱스 삭제 |
| S2 | 순위 비교일은 순위 전체에 하나 |
| S3 | `poc_task8`은 4단계 끝에 삭제 |
| F1 | 원화 표시 방식은 #13에서 정함 (홈·검색 화면은 소스 통화 그대로) |
| — | 11-25 누락된 상시 수집은 수동 실행하지 않음 |

## 발견 사항

- **검색 NULL 결함 (1단계, 수정).** 처음 쿼리에서 트레이너스·에너지(`dex_ids` NULL)가 종 이름 조건을 NULL로 통과해 모든 낱말에 맞는 것으로 나왔다(`피카츄 ex` 6,885장 → 50장). `coalesce`로 막고 회귀 테스트를 넣었다.
- **대소문자 무시 비교가 느리다.** 이 DB(Alpine, `en_US.utf8`)에서 `lower()`·`ILIKE`로 카드 3만 7천 장을 훑으면 조건 하나에 약 25~35 ms가 들어, 낱말 배열 OR 검색이 100~200 ms였다. 가장 긴 낱말로 인덱스 후보를 찾는 방식과 인덱스 2개로 약 10 ms가 됐다.
- **순위 쿼리 플래너 선택.** 기준일·비교일 조인을 플래너가 기본 키 중첩 루프로 풀어 1년치에서 약 240 ms였다. 두 날짜를 `MATERIALIZED` CTE로 따로 읽어 해시 조인하게 하자 60~90 ms가 됐다.
- **1년치 규모에서 첫 조회가 느리다.** 가격 이력 1년치는 약 4.7 GB인데 `shared_buffers`가 128 MB(기본값)라, 카드마다 처음 읽는 1년 추이는 p95 97 ms(캐시 후 약 45 ms), 순위 첫 요청은 최대 247 ms였다. 1년 추이 응답은 변형 2개 카드가 약 78 KB다.
- **3천만 행 생성 방식.** 외래 키·인덱스를 둔 채 넣으면 1시간이 넘어 취소했고, 떼고 넣은 뒤 다시 만들면 약 3분이었다. 측정용 대량 데이터는 이 방식으로 만든다.
- **같은 변형도 소스 사이 가격 차이가 크다** (예: `ex6-74` 리버스 홀로 TCGCSV $317.99, Cardmarket €90.75). 상세·추이를 변형 × 소스로 나눠 보여 주는 이유다. 연결 결함인지는 판단하지 않았다.
- **11-25 상시 수집 누락.** Docker Desktop이 꺼져 있어 05:30 실행이 빠졌다. 지금 TCGCSV 최신일이 11-23이라 7d 순위는 비교일이 없어 빈 목록이다. 매일 수집이 이어지면 11-30 무렵부터 나온다.

## 검증

| 항목 | 결과 |
|------|------|
| 백엔드 빌드 (`go build`, `go vet`, 새 파일 `gofmt`) | ✅ |
| 백엔드 테스트 (`go test ./...`, DB 테스트 포함) | PASS 160 (하위 테스트 포함, 이 타스크 `internal/api`·`internal/server` 54), FAIL·SKIP 0 |
| 프론트엔드 `typecheck`·`lint`·`build` | ✅ |
| 계약 일치 | 4개 엔드포인트 응답 형식·오류 형식이 `api_contract.md`·`types.ts`와 같음 ✅ |
| 화면 (헤드리스 Chromium, 실제 API) | 홈·검색 4화면, 콘솔 오류·가로 스크롤 없음 ✅ |
| 응답 시간 (개발 DB, 100회) | 검색 p95 15~36 ms, 상세·추이 p95 2~3 ms, 순위 p95 1 ms ✅ |
| 응답 시간 (1년치 복제 DB) | 순위 p50 67~79 ms·p95 최대 247 ms, 1년 추이 p50 45 ms·p95 97 ms, 상세 p95 9 ms ✅ |
| 마이그레이션 | `00005` 개발 DB 적용, up/down 테스트 통과 ✅ |
| 회귀 | 수집기·카탈로그·DB 테스트 통과, `/healthz` 응답 유지 (오류 응답만 계약 형식으로 바뀜) |

## 계획 대비 차이

| 계획 | 실제 | 사유 |
|------|------|------|
| 측정용 임시 DB는 1단계 후 삭제 | 4단계 순위 재측정까지 유지 후 삭제 | 다시 만드는 비용 (S3) |
| 1단계 측정은 SQL만 | 4단계에서 같은 코드의 API 서버를 측정용 DB에 따로 띄워 HTTP 단위로도 측정 | 실제 처리 시간 확인 |
| 순위 비교일 "그날 없으면 이전 가장 가까운 날" | 순위 전체에 비교일 하나 | 변형마다 찾으면 30d 약 1초 (S2) |
| `page` 1 이상 | 1~10000 | 정수 넘침 방지 (계약에 반영) |
| 홈·검색 화면 원화 표시 확인 | 화면 변경 없음 | F1, #13에서 정함 |
| — | 오류 코드 `405 method_not_allowed` 추가, `CardDetail.dexIds`·`set.cardCount` 추가 | 계약 확정 중 보완 |

## 후속 조치

| 타스크 | 인계 사항 |
|--------|-----------|
| #13 카드 상세 화면 | `getCard`·`getCardPrices`와 `CardDetail`·`PriceHistoryResponse` 타입을 쓴다. 원화 표시 방식(F1)을 정한다. 1년 추이 응답이 수십 KB라 기간 전환 시 캐시를 고려한다 |
| #11 캐시·성능 | 1년치에서 첫 조회가 느림(`shared_buffers` 128 MB). 1자 검색 120~190 ms. 순위는 하루 한 번 바뀌므로 캐시 효과가 크다 |
| #7 국내 거래가 | 한글판 대표가는 지금 `null`. 일본판 참고가를 더할 때 대표가 규칙(`api_contract.md` 2절, `internal/api.PickRepresentative`)을 갱신한다 |
| #16 배포 | 꼬리말에 가격 출처(TCGplayer·Cardmarket)·이용 조건 표기가 없다 |
| #10 내 컬렉션 | 목록 항목은 `CardSummary`와 `summaries`(대표가 일괄 조회)를 다시 쓸 수 있다 |
| 운영 | 상시 수집이 하루 빠지면 7d 순위가 빈다. 다음 실행(11-26 05:30) 로그에서 #6 후속 확인 항목도 함께 본다 |

## 이슈

- https://github.com/Chocobone/PokeCardPrice/issues/8
