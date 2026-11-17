# Task #12 단계2 완료 보고서: API 계약 초안, API 클라이언트, 목업

## 수행 내용

- **API 계약 초안** (`mydocs/tech/api_contract.md`): `GET /cards`(검색)와 `GET /rankings/movers`(급등·급락)의 요청·응답·오류 형식, `CardSummary`·`Price`·`Mover` 타입, 목업 규칙과 표본 추출 쿼리를 정리했다. 대표가·순위 산출 기준과 검색 정렬은 #8에서 정하도록 남겼다.
- **타입** (`src/api/types.ts`): 계약 문서와 같은 타입.
- **API 클라이언트** (`src/api/client.ts`):
  - `apiGet<T>(path, { params, signal, timeoutMs })`: `/api` 기준, 쿼리에서 빈 값 제외, 제한 시간 기본 10초.
  - 오류는 `ApiError(status, code, message)`로 통일한다. 응답 오류는 본문의 `error.code·message`를 쓰고, 연결 실패는 `status 0`·`network`, 제한 시간 초과는 `status 0`·`timeout`이다. 호출 측 취소는 `AbortError` 그대로 던진다(`isAbortError`).
  - `errorMessage(e)`: 화면용 한국어 문구 (5xx는 "잠시 후 다시 시도").
- **엔드포인트 함수** (`src/api/cards.ts`): `searchCards`, `getMovers`. `VITE_API_MOCK=true`면 목업 모듈을 동적으로 불러 쓰고, 아니면 실제 API를 부른다.
- **목업** (`src/api/mock/`):
  - `data.ts`: 개발 DB 표본 36장 (영어판 `sv03.5` 20장·`base1` 4장, 일본판 12장). 자동 생성 파일.
  - `handlers.ts`: 계약대로 검색(낱말별 이름·한글 종 이름·세트 이름 부분 일치, 세트 ID·번호 일치, 언어 필터, 쪽 넘김)과 순위(기간별 변동률, 방향별 정렬)를 흉내 낸다. 가격·변동률은 카드 ID로 만든 고정 난수라 같은 카드는 늘 같은 값이다. 응답 지연 300ms, 요청 취소 지원, `__error` 검색어는 500 오류.
- **`.env.development`**: `VITE_API_MOCK=true` (개발 서버에만 적용).

## 변경 파일

| 파일 | 변경 내용 |
|------|----------|
| `mydocs/tech/api_contract.md` | 신규. API 계약 초안, 목업 규칙, 표본 추출 쿼리 |
| `src/frontend/src/api/types.ts` | 신규 |
| `src/frontend/src/api/client.ts` | 신규 |
| `src/frontend/src/api/cards.ts` | 신규 |
| `src/frontend/src/api/mock/handlers.ts` | 신규 |
| `src/frontend/src/api/mock/data.ts` | 신규 (자동 생성) |
| `src/frontend/.env.development` | 신규 |
| `mydocs/orders/20261117.md` | #12 절 갱신 |

## 발견 사항

- **목업 표본에 한글판 카드와 SV 시대 일본판 카드가 아직 없다.** #5 전체 적재가 진행 중이라 추출 시점에 일본판은 SM 시대까지, 한글판은 없었다. SM 시대 일본판 카드는 TCGdex에 이미지가 없어 표본의 일본판 12장은 모두 이미지가 없다. 적재가 끝나면 같은 쿼리로 `data.ts`를 다시 만들어 한글판·이미지 있는 일본판을 넣는다 (4단계 전).
- **임시 표시 코드 대신 브라우저에서 모듈을 직접 불러 확인했다.** 계획서는 홈 화면에 임시로 결과 건수를 띄우는 방식이었으나, 헤드리스 Chromium에서 개발 서버의 `src/api/*.ts`를 직접 import해 호출 결과를 확인했다. 화면 코드에 지울 코드를 넣지 않아도 된다.
- **운영 빌드에서 목업이 빠지는지는 3단계에서 다시 확인한다.** 지금은 어떤 화면도 `src/api`를 불러오지 않아 빌드 결과에 API 코드 자체가 없다. 화면이 API를 쓰는 3단계에서 목업 코드가 번들에 없는지 확인한다.
- 목업 검색은 세트 이름도 부분 일치로 찾는다. `151`은 세트 이름 "151"과 번호 151에 모두 맞아 20장이 나온다. 실제 검색의 일치 범위는 #8에서 정한다.

## 검증

- `npm run typecheck`, `npm run lint`, `npm run build`: ✅
- `npm audit`: 취약점 0건
- 헤드리스 Chromium에서 개발 서버 모듈 직접 호출 (목업 모드):

| 확인 | 결과 |
|------|------|
| `피카츄` 검색 | 5장 (영어 3, 일본어 2), 가격 통화가 언어별로 USD·JPY |
| `피카츄` + `lang=ja` | 일본판 2장만 |
| `sv03.5 25` | `sv03.5-025` 1장 (번호 앞 0 무시) |
| `151`, `size=5`, `page=2` | `page 2`, `size 5`, `total 20`, 항목 5개 |
| 없는 검색어 | `total 0` |
| `__error` | `ApiError` 500 `internal`, 화면 문구 "서버에 문제가 생겼습니다…" |
| 요청 직후 취소 | `AbortError`, `isAbortError` 참 |
| 급등 7일 | 변동률 내림차순 (0.575, 0.564, …), 이전 가격 기준일 7일 전 |
| 급락 7일 | 음수, 가장 많이 내린 순 (-0.435, -0.383, …) |
| 급등 30일 | 7일과 다른 값 |
| 실제 API 경로 (`/healthz`) | 백엔드 없는 #12 프로젝트라 `ApiError` 502 `http_502` |
| `timeoutMs: 1` | `ApiError` 0 `timeout` |
| 쿼리 문자열 | `undefined`·빈 값 제외, 한글 인코딩 |

- 계약 문서의 타입과 `types.ts`가 같다: ✅

## 다음 단계

3단계: 검색창(`SearchBox`), 급등·급락 스토어와 목록, 공용 컴포넌트(카드 이미지, 언어 배지, 상태 표시), 홈 화면.

## 커밋

(승인 후 커밋)
