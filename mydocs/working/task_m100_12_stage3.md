# Task #12 단계3 완료 보고서: 홈 화면 (검색창, 급등·급락)

## 수행 내용

- **검색창** (`components/SearchBox.vue`): 헤더용(`compact`)·홈용(`hero`) 두 모양. 제출하면 `/search?q=...`로 이동하고, 앞뒤 공백을 뺀 값이 비면 버튼이 비활성이라 이동하지 않는다. 검색 화면에서는 URL의 `q`를 입력값으로 보여 주고, 언어 필터(`lang`)는 유지한 채 쪽을 처음으로 돌린다. 최대 100자.
- **헤더 검색창**: 홈 외 화면의 헤더에 붙였다 (홈은 큰 검색창이 있어 숨김). 모바일은 다음 줄 전체 폭, 640px 이상은 이름 옆.
- **급등·급락 스토어** (`stores/movers.ts`): 기간·방향별(`7d:up` 등)로 결과·로딩·오류를 보관한다. 이미 있거나 불러오는 중이면 다시 부르지 않고, "다시 시도"는 `force`로 다시 부른다.
- **급등·급락 화면** (`components/movers/`): 기간 탭(7일·30일), 급등·급락 두 목록(1024px 이상 두 열, 그 아래는 위아래). 행은 순위·썸네일·이름·언어 배지·세트·번호·현재가·변동률(▲ 빨강 / ▼ 파랑). 행을 누르면 `/cards/:id`. 기준일 표시. 로딩·빈 목록("시세 데이터가 쌓이면 표시됩니다")·오류("다시 시도") 상태.
- **공용 컴포넌트·함수**:
  - `CardImage`: TCGdex 이미지(`/low.webp`·`/high.webp`), 지연 로딩, 이미지가 없거나 실패하면 "이미지 없음" 자리 표시.
  - `LangBadge`: EN·JP·KR 배지(화면 낭독기에는 "영어판" 등), 일본판에 한글판이 있으면 "한글판" 배지.
  - `StateView`: 로딩·빈 결과·오류 공통 표시 (`role=status`·`alert`).
  - `utils/format.ts`: 통화별 금액(`$135.57`, `¥847`, `₩12,000`), 변동률(`+12.5%`), 날짜(`2026년 11월 16일`).
- **목업 개발 도구**: 주소에 `?mock=empty`·`?mock=error`를 붙이면 빈 결과·서버 오류로 바뀐다. 호출 수는 `globalThis.__mockStats`. 모두 목업 코드라 운영 빌드에는 없다 (`api_contract.md` 4절에 기록).

## 변경 파일

| 파일 | 변경 내용 |
|------|----------|
| `src/frontend/src/components/SearchBox.vue` | 신규 |
| `src/frontend/src/components/layout/AppHeader.vue` | 헤더 검색창 연결, 모바일 줄바꿈 |
| `src/frontend/src/components/movers/MoversPanel.vue`, `MoverList.vue`, `MoverRow.vue` | 신규 |
| `src/frontend/src/components/common/CardImage.vue`, `LangBadge.vue`, `StateView.vue` | 신규 |
| `src/frontend/src/stores/movers.ts` | 신규 |
| `src/frontend/src/utils/format.ts` | 신규 |
| `src/frontend/src/views/HomeView.vue` | 큰 검색창 + 급등·급락 |
| `src/frontend/src/api/mock/handlers.ts` | 상황 전환(`?mock=`), 호출 수 |
| `mydocs/tech/api_contract.md` | 목업 상황 전환·호출 수 설명 |
| `mydocs/orders/20261117.md` | #12 절 갱신 |

## 발견 사항

- **목업 호출 수는 전역 변수로 읽는다.** 처음에는 확인 스크립트가 `handlers.ts` 모듈을 직접 불러 호출 수를 읽었으나 0이 나왔다. 파일을 고친 뒤에는 Vite가 앱 쪽 모듈을 `?t=` 붙은 주소로 다시 불러 서로 다른 모듈 인스턴스가 되기 때문이다. `globalThis.__mockStats`로 바꿔 해결했다.
- **모바일(360px)에서 홈 검색창 안내 문구가 잘린다** ("포켓몬 이름(피카츄), 카…"). 입력하면 사라지는 안내 문구라 그대로 두었다. 필요하면 4단계에서 모바일용 짧은 문구로 바꾼다.
- 목업 일본판 카드는 모두 이미지가 없어 "이미지 없음"으로 보인다 (2단계 보고서 참고, 4단계 전에 표본을 다시 만든다).

## 검증

- `npm run typecheck`, `npm run lint`, `npm run build`: ✅ (기본 JS 110 kB / gzip 43 kB)
- `npm audit`: 취약점 0건
- **운영 빌드에서 목업 제외**: 운영 빌드 결과에 API 코드(`rankings/movers`)는 있고 목업 표본·`__mockStats`는 없다. 대조로 목업 모드 빌드를 하면 `handlers-*.js` 청크가 생긴다: ✅
- 화면 (헤드리스 Chromium, 목업 모드):

| 확인 | 360px | 1280px |
|------|-------|--------|
| 가로 스크롤 | 없음 | 없음 |
| 콘솔 오류 | 없음 | 없음 |
| 홈 헤더 검색창 숨김 | ✅ | ✅ |
| 급등·급락 각 10행, 기준일 표시 | ✅ | ✅ |
| 이미지 14장 모두 불러옴, 나머지 6행은 "이미지 없음" | ✅ | ✅ |

| 동작 (1280px) | 결과 |
|---------------|------|
| 첫 화면 목업 호출 | 순위 2회 (급등·급락) |
| 30일 탭 | 목록이 바뀜, 탭 선택 표시, 호출 4회 |
| 다시 7일 탭 | 추가 호출 없음 (4회) |
| 빈 검색창 | 검색 버튼 비활성 |
| `  피카츄 ` 입력 후 Enter | `/search?q=피카츄` (공백 제거), 헤더 검색창에 `피카츄` |
| 헤더 검색창에 `Mew` | `/search?q=Mew` |
| 이름 눌러 홈으로 돌아옴 | 추가 호출 없음 (4회) |
| 급등 1위 행 클릭 | `/cards/17078`, 탭 제목 "카드 상세 · Card-Market" |
| `?mock=empty` | 두 목록 모두 "시세 데이터가 쌓이면 표시됩니다." |
| `?mock=error` | 두 목록 모두 오류 문구 + "다시 시도" 버튼 (`role=alert`) |

## 다음 단계

4단계: 목업 표본 재생성(한글판·이미지 있는 일본판), 검색 스토어와 검색 결과 화면(URL 동기화, 언어 필터, 카드 격자, 쪽 넘김), 프론트엔드 구조 문서, `CLAUDE.md`.

## 커밋

(승인 후 커밋)
