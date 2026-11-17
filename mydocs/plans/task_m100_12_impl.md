# Task #12: 프론트엔드 기반 및 홈·검색 화면 — 구현 계획서

수행 계획서: `mydocs/plans/task_m100_12.md` (Q1·Q2 제안, Q3·Q4 대안으로 결정)

## 공통 규칙

### 변경 범위

- `src/frontend/`와 신규 문서만 바꾼다. 백엔드·수집기·`Docker/` 파일은 바꾸지 않는다 (#5와 merge 충돌 방지).
- 공용 파일은 #5가 고치지 않는 위치에만 추가한다. 오늘 할일은 파일 끝의 #12 절, `CLAUDE.md`는 4단계에서 프론트엔드 줄만 추가한다.

### 개발 환경

#5 전체 적재가 `backend` 컨테이너에서 돌고 있고, 기존 `frontend` 컨테이너는 #5 작업 폴더(`PokeCardPrice/src/frontend`)를 마운트한다. #12는 worktree에서 별도 compose 프로젝트로 `frontend`만 띄운다.

```bash
# PokeCardPrice-task12/Docker 에서 (최초 1회 .env 복사 후 FRONTEND_PORT=5174 로 변경)
docker compose -p pokecardprice-t12 up -d --no-deps frontend     # http://localhost:5174
docker compose -p pokecardprice-t12 exec frontend npm install    # 의존성 추가 후
docker compose -p pokecardprice-t12 exec frontend npm run build  # 타입 검사 + 빌드
docker compose -p pokecardprice-t12 exec frontend npm run lint
```

- `backend`가 없는 프로젝트라 `/api` 프록시는 연결되지 않는다. 화면은 목업 모드로 확인한다.
- worktree의 `Docker/.env`는 Git에 올리지 않는다 (기존 `.gitignore`).

### 패키지 버전 (2026-11-17 `npm view` 확인)

| 패키지 | 버전 | 비고 |
|--------|------|------|
| `vue-router` | ^5.5.1 | |
| `pinia` | ^4.1.0 | `vue-router` 5의 선택적 peer (`^3.0.4 \|\| ^4.0.2`) 범위 안 |
| `typescript` | ~6.0.3 | 최신은 7.0.2이나 `typescript-eslint` 8.73이 `<6.1.0`만 지원해 6.0으로 고정 |
| `vue-tsc` | ^3.3.14 | `typescript >=5.0.0` |
| `@vue/tsconfig` | ^0.9.1 | |
| `@types/node` | ^26 | `vite.config.ts`용 |
| `eslint` | ^10.15.0 | |
| `eslint-plugin-vue` | ^10.12.1 | |
| `typescript-eslint` | ^8.73.0 | |
| `@vue/eslint-config-typescript` | ^14.9.0 | |

실제 설치에서 peer 충돌이 나면 버전을 조정하고 단계 보고서에 기록한다.

### 검증 방식 (단위 테스트 없음, Q3)

- 매 단계: `npm run typecheck`(`vue-tsc`), `npm run lint`, `npm run build`, `npm audit`.
- 화면: 목업 모드에서 데스크톱(1280px)·모바일(360px) 폭으로 확인한다. 브라우저 도구로 확인할 수 있으면 화면을 캡처해 보고서에 남기고, 아니면 작업지시자 확인을 요청한다.

---

## 1단계: TypeScript 전환, 개발 도구, 라우터·Pinia·레이아웃 골격

### 목표

TypeScript·ESLint·라우터·Pinia를 갖춘 앱 골격과 반응형 레이아웃을 만들고, 임시 상태 화면을 교체한다.

### 구현

**`src/frontend/package.json`** — 의존성(위 표)과 스크립트:

```json
{
  "scripts": {
    "dev": "vite",
    "build": "vue-tsc -b && vite build",
    "typecheck": "vue-tsc -b",
    "lint": "eslint .",
    "preview": "vite preview"
  }
}
```

**TypeScript 설정** (신규) — `tsconfig.json`(참조만), `tsconfig.app.json`(`@vue/tsconfig/tsconfig.dom.json` 확장, `strict`, 경로 별칭 `@/*` → `src/*`), `tsconfig.node.json`(`vite.config.ts`용), `src/env.d.ts`(`ImportMetaEnv`에 `VITE_API_MOCK` 선언).

**`vite.config.js` → `vite.config.ts`** — 기존 설정(프록시, polling)을 그대로 옮기고 `@` 별칭을 추가한다.

**`eslint.config.js`** (신규) — `eslint-plugin-vue`(`flat/recommended`) + `@vue/eslint-config-typescript`. `dist/`는 제외한다.

**`src/main.ts`** — `createApp(App).use(createPinia()).use(router).mount('#app')`.

**`src/router/index.ts`** (신규):

```ts
const routes: RouteRecordRaw[] = [
  { path: '/', name: 'home', component: HomeView },
  { path: '/search', name: 'search', component: () => import('@/views/SearchView.vue') },
  { path: '/cards/:id', name: 'card', component: () => import('@/views/CardPlaceholderView.vue') }, // #13
  { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('@/views/NotFoundView.vue') },
]
```

- `createWebHistory`를 쓴다. 개발 서버는 모든 경로에 `index.html`을 돌려준다. 배포 시 서버 설정은 #16에 인계한다.
- 화면 제목(`document.title`)은 라우트 `meta.title`로 정한다.

**레이아웃** (신규):

| 파일 | 내용 |
|------|------|
| `src/App.vue` | `AppHeader` + `<RouterView>` + `AppFooter` |
| `src/components/layout/AppHeader.vue` | 서비스 이름(홈 링크), 검색창 자리 (2단계 이후 `SearchBox` 연결) |
| `src/components/layout/AppFooter.vue` | 데이터 출처 표기 (TCGdex, PokéAPI, 시세 소스는 #6 이후 추가), 상표 고지 |
| `src/views/HomeView.vue`, `SearchView.vue` | 자리 화면 (3·4단계에서 구현) |
| `src/views/CardPlaceholderView.vue` | "카드 상세는 준비 중" (#13) |
| `src/views/NotFoundView.vue` | 찾을 수 없음, 홈 링크 |

**`src/styles/`** — `style.css`를 대체:

- `tokens.css`: 색·간격·글꼴·둥근 모서리·그림자 CSS 변수. 상승(빨강)·하락(파랑)은 국내 시세 표기 관례를 따른다. 어두운 모드는 `prefers-color-scheme`로 변수만 바꾼다.
- `base.css`: 초기화, 본문 폭(`max-width` + 좌우 여백 16px), 링크·버튼·입력 기본 스타일, `:focus-visible`.
- 반응형 기준: 모바일 기본, `640px`·`1024px` 이상에서 열 수·여백을 넓힌다. 가로 스크롤이 생기지 않게 한다.

### 검증

- `typecheck`·`lint`·`build` 통과, `npm audit` 결과 기록.
- `/`, `/search`, `/cards/1`, `/없는경로`가 각 화면으로 이동하고, 새로고침해도 같은 화면이다.
- 360px·1280px 폭에서 헤더·본문·푸터에 가로 스크롤이 없다.

---

## 2단계: API 계약 초안, API 클라이언트, 목업

### 목표

#8이 구현할 API 형식을 문서로 정하고, 화면이 쓸 API 클라이언트와 목업 응답을 만든다.

### 구현

**`mydocs/tech/api_contract.md`** (신규, `tech_investigation_template.md` 형식) — 이 타스크가 쓰는 엔드포인트의 계약 초안. #8 수행 계획서의 입력이다.

공통:

- 경로는 백엔드 기준이다(`/cards`). 프론트엔드는 `/api/cards`로 호출하고 Vite 프록시가 `/api`를 뗀다.
- 오류 응답: `{"error": {"code": "invalid_query", "message": "..."}}` + 적절한 HTTP 상태.
- 금액은 정밀도를 지키기 위해 **문자열**(`"12.34"`)로 보낸다 (백엔드 `decimal`). 변동률은 숫자(`0.125` = 12.5%).
- 쪽 넘김 응답: `{"items": [...], "page": 1, "size": 24, "total": 135}`. `page`는 1부터.

| 엔드포인트 | 요청 | 응답 |
|------------|------|------|
| `GET /cards` | `q`(필수, 1~100자), `lang`(`en`·`ja`·`ko`, 선택), `page`(기본 1), `size`(기본 24, 최대 60) | `Page<CardSummary>` |
| `GET /rankings/movers` | `period`(`7d`·`30d`), `direction`(`up`·`down`), `limit`(기본 10, 최대 50) | `{"period", "direction", "asOf", "items": Mover[]}` |

```ts
type Language = 'en' | 'ja' | 'ko'

interface CardSummary {
  id: number              // cards.id
  language: Language
  tcgdexId: string        // "sv03.5-025"
  name: string
  localId: string         // 카드 번호 "025"
  set: { tcgdexId: string; name: string }
  imageUrl: string | null // TCGdex 기본 주소 (화면에서 "/low.webp" 등을 붙임). ko 카드는 null
  hasKoEdition: boolean   // ja 카드에 한글판이 있는지 (#5 규칙)
  speciesNameKo: string | null // 첫 도감 번호의 한글 종 이름 (트레이너스·에너지는 null)
  price: Price | null     // 대표가. 시세가 없으면 null
}

interface Price {
  amount: string          // "12.34"
  currency: 'USD' | 'JPY' | 'KRW'
  source: string          // price_sources 코드 (예: "tcgcsv")
  date: string            // "2026-11-16"
}

interface Mover {
  card: CardSummary
  variant: string         // card_variants.variant
  current: Price
  previous: Price
  changeRate: number      // (current - previous) / previous
}
```

- 대표가·변동률의 산출 기준(어느 소스·변형·통화)은 #8에서 정한다. 계약은 그 결과를 담는 형식만 정한다.
- 검색 대상(`q`): 카드 이름(영·일·한), 한글 종 이름(`pokemon_species.name_ko`), 세트 ID, `세트ID 번호` 형식. 일치 방식(부분 일치, 정렬)은 #8에서 정한다.

**`src/api/types.ts`** — 위 타입.

**`src/api/client.ts`** — `fetch` 래퍼:

```ts
export class ApiError extends Error {
  constructor(public status: number, public code: string, message: string) { super(message) }
}

export interface RequestOptions {
  params?: Record<string, string | number | undefined>
  signal?: AbortSignal
  timeoutMs?: number // 기본 10초
}

export async function apiGet<T>(path: string, opts?: RequestOptions): Promise<T>
```

- 기준 주소 `/api`. `params`의 `undefined`는 빼고 쿼리 문자열을 만든다.
- 응답이 2xx가 아니면 오류 본문을 읽어 `ApiError`로, 네트워크 오류·제한 시간 초과는 `ApiError(0, 'network' | 'timeout', ...)`로 바꾼다. 호출 측 취소(`AbortError`)는 그대로 던져 화면이 무시할 수 있게 한다.

**`src/api/cards.ts`** — `searchCards(params, signal)`, `getMovers(params, signal)`. 목업 여부에 따라 실제 호출과 목업을 고른다(화면·스토어는 모름).

**`src/api/mock/`** — 목업:

- `data.ts`: #5에서 적재한 실제 카탈로그 표본 약 40장 (`en` `sv03.5`, `ja` `SV4K`·`SV5K`, `ko` `SV5K` 일부). 이름·번호·이미지 URL·`has_ko_edition`·한글 종 이름은 개발 DB 값을 옮기고, 가격·변동률은 임의 값이다 (데이터 파일 주석에 명시).
- `handlers.ts`: 계약대로 검색(이름·한글 종 이름·세트 ID·번호 부분 일치, `lang` 필터, 쪽 넘김)과 순위(기간·방향별 정렬)를 흉내 낸다. 응답 지연 300ms. 오류 화면 확인용으로 `q=__error`이면 `ApiError(500)`를 던진다.

**`src/frontend/.env.development`** (신규) — `VITE_API_MOCK=true`. #8 완료 후 `false`로 바꾸거나 지운다.

### 검증

- `typecheck`·`lint`·`build` 통과.
- 목업 모드에서 홈 자리 화면에 임시로 검색·순위 호출 결과 건수를 표시해 확인하고, 3단계에서 지운다.
- 목업을 끄면(`VITE_API_MOCK=false`) `/api/cards` 요청이 나가고, 백엔드가 없을 때 `ApiError`로 처리되는 것을 확인한다.
- 계약 문서의 타입과 `src/api/types.ts`가 같다.

---

## 3단계: 홈 화면 (검색창, 급등·급락)

### 목표

어느 화면에서나 쓰는 검색창과, 홈의 급등·급락 목록을 만든다.

### 구현

**`src/components/SearchBox.vue`** — 검색 입력과 버튼:

- 제출 시 `router.push({ name: 'search', query: { q } })`. 앞뒤 공백을 빼고, 빈 값이면 이동하지 않는다.
- 검색 화면에서는 URL의 `q`를 입력값으로 보여 준다.
- 안내 문구: "포켓몬 이름(피카츄), 카드 이름, 세트 번호로 검색".
- 헤더용(작게)·홈용(크게) 두 모양을 속성으로 고른다.

**`src/stores/movers.ts`** (Pinia):

```ts
type MoverKey = `${'7d' | '30d'}:${'up' | 'down'}`
state: { results: Partial<Record<MoverKey, MoversResponse>>, loading: Set<MoverKey>, errors: Partial<Record<MoverKey, string>> }
actions: { load(period, direction, { force }?) }  // 같은 키는 다시 부르지 않음 (홈 재방문 시 재사용)
```

**`src/components/movers/MoversPanel.vue`**, **`MoverRow.vue`**:

- 기간 탭(7일·30일)과 급등·급락 두 목록. 데스크톱은 두 열, 모바일은 위아래로 쌓는다.
- 행: 순위, 카드 썸네일(`{imageUrl}/low.webp`, 없으면 대체 이미지), 이름·세트·번호, 언어 배지, 현재가, 변동률(▲ 상승 빨강, ▼ 하락 파랑, 부호와 %). 행을 누르면 `/cards/:id`.
- 상태: 로딩(골격 화면), 빈 목록("시세 데이터가 쌓이면 표시됩니다"), 오류(문구 + 다시 시도).
- 기준일(`asOf`)을 목록 위에 표시한다.

**공용 컴포넌트·함수**:

| 파일 | 내용 |
|------|------|
| `src/components/common/CardImage.vue` | 이미지 크기(`low`·`high`), 지연 로딩, 실패·없음 시 대체 이미지 |
| `src/components/common/LangBadge.vue` | 영어·일본어·한국어 배지, 한글판 있음 배지 |
| `src/components/common/StateView.vue` | 로딩·빈 결과·오류 공통 표시 |
| `src/utils/format.ts` | 금액(`Intl.NumberFormat`, 통화별 소수 자리), 변동률, 날짜 |

**`src/views/HomeView.vue`** — 큰 검색창 + `MoversPanel`. 2단계 임시 표시는 지운다.

### 검증

- `typecheck`·`lint`·`build` 통과.
- 목업 모드: 기간 탭 전환 시 목록이 바뀐다, 다시 홈에 오면 다시 요청하지 않는다, 행을 누르면 카드 자리 화면으로 간다.
- 헤더·홈 검색창에서 검색하면 `/search?q=...`로 이동한다. 빈 검색어는 이동하지 않는다.
- 목업 응답을 빈 목록·오류로 바꿔 각 상태 화면을 확인한다.
- 360px·1280px 폭에서 배치가 깨지지 않는다.

---

## 4단계: 검색 결과 화면, 문서 정리

### 목표

URL과 동기화되는 검색 결과 화면을 만들고, 프론트엔드 구조·실행 방법을 문서로 남긴다.

### 구현

**`src/stores/search.ts`** (Pinia):

```ts
state: { query: SearchQuery | null, result: Page<CardSummary> | null, loading: boolean, error: string | null }
actions: { run(query: SearchQuery) } // 이전 요청은 AbortController 로 취소, 같은 질의는 다시 부르지 않음
```

**`src/views/SearchView.vue`**:

- URL 쿼리(`q`, `lang`, `page`)가 기준이다. 라우트 쿼리를 감시해 `search.run()`을 부르고, 필터·쪽 변경은 `router.push`로 URL만 바꾼다. 새로고침·뒤로 가기·링크 공유 시 같은 결과를 보여 준다.
- 잘못된 값(`page=abc`, 모르는 `lang`)은 기본값으로 바꾼다 (`router.replace`).
- 상단: 검색어, 결과 수, 언어 필터(전체·영어·일본어·한국어).
- `q`가 없으면 검색 안내, 결과가 없으면 "검색 결과가 없습니다"와 검색 요령.

**`src/components/cards/CardGrid.vue`**, **`CardTile.vue`**:

- 격자 열 수: 모바일 2열, 640px 이상 3~4열, 1024px 이상 5~6열 (`grid-template-columns: repeat(auto-fill, minmax(...))`).
- 타일: 이미지(비율 고정, 지연 로딩), 이름, 한글 종 이름(이름과 다를 때), 세트 이름·번호, 언어 배지, 한글판 배지, 대표가(없으면 "시세 없음"). 누르면 `/cards/:id`.

**`src/components/common/Pagination.vue`** — 이전·다음, 현재 쪽 주변 번호. 모바일에서는 이전·다음과 "n / N"만 보인다.

**문서**:

- `mydocs/tech/frontend_structure.md` (신규): 폴더 구조, 라우트 표, 스토어, API 클라이언트·목업 전환 방법, 스타일 변수, #8·#13·#14·#16 인계 사항(계약 준수, 카드 상세 경로, 배포 시 SPA 경로 처리).
- `mydocs/tech/api_contract.md`: 구현 중 바뀐 내용 반영.
- `CLAUDE.md` 빌드 및 실행: 프론트엔드 명령(`typecheck`, `lint`)과 목업 모드 설명을 프론트엔드 줄에만 추가한다.

### 검증

- `typecheck`·`lint`·`build` 통과, `npm audit` 결과 기록.
- 목업 모드: `피카츄`·`Pikachu`·`SV5K`로 검색한 결과가 목업 데이터와 맞다, 언어 필터·쪽 넘김이 URL에 반영되고 새로고침·뒤로 가기 후에도 같다, 빠르게 검색어를 바꿔도 마지막 결과만 보인다.
- `q=__error`로 오류 상태, 결과 없는 검색어로 빈 상태를 확인한다.
- 360px·1280px 폭에서 격자·쪽 넘김이 깨지지 않고 가로 스크롤이 없다.

---

## 검증 (전체)

- `npm run typecheck`, `npm run lint`, `npm run build` 통과
- 목업 모드에서 홈 → 검색 → 필터·쪽 넘김 → 카드 자리 화면 흐름이 데스크톱·모바일 폭에서 동작
- API 계약 문서와 `src/api/types.ts`가 일치
- 백엔드·수집기·`Docker/` 파일 변경 없음 (`git diff --stat local/task5...local/task12`로 확인)

## 예상 단계: 4단계
