# 기술 조사: 프론트엔드 구조

> **작성일**: 2026-11-17
> **관련 타스크**: #12 (작성), #13 (카드 상세 추가), #14·#8·#16 (인계)

---

## 배경

#12에서 #1의 임시 화면을 실제 앱 골격(TypeScript, 라우터, Pinia, API 클라이언트, 반응형 레이아웃)과 홈·검색 화면으로 바꿨다. 이후 화면 타스크(#13 카드 상세, #14 로그인·컬렉션)가 같은 구조 위에서 이어 갈 수 있도록 구조와 규칙을 정리한다.

## 조사 내용

### 1. 기술 구성

| 항목 | 선택 | 비고 |
|------|------|------|
| 언어 | TypeScript ~6.0 (`strict`, `noUncheckedIndexedAccess`) | 7.x 는 `typescript-eslint`가 아직 지원하지 않아 6.0으로 고정 (#12 Q4) |
| 프레임워크 | Vue 3.5 (`<script setup lang="ts">`) | |
| 라우터 | vue-router 5 (`createWebHistory`) | |
| 상태 | Pinia 4 (setup 스토어) | |
| 빌드 | Vite 8 (`vue-tsc -b && vite build`) | |
| 정적 검사 | ESLint 10 + `eslint-plugin-vue` `flat/recommended` + `@vue/eslint-config-typescript` | 설정 파일은 `eslint.config.js` (TS 설정 파일은 `jiti` 추가 필요) |
| UI | 프레임워크 없음, CSS 변수 + 직접 작성 컴포넌트 (#12 Q2) | |
| 단위 테스트 | 없음 (#15에서 정함, #12 Q3) | 화면 확인은 헤드리스 Chromium |

### 2. 폴더 구조 (`src/frontend/src/`)

| 경로 | 내용 |
|------|------|
| `main.ts` | 앱 생성, Pinia·라우터 등록, 전역 스타일 |
| `App.vue` | 레이아웃 (헤더 + `RouterView` + 푸터) |
| `router/index.ts` | 라우트, 탭 제목(`meta.title`), 스크롤 복원 |
| `api/types.ts` | API 계약 타입 (`mydocs/tech/api_contract.md`와 같아야 함) |
| `api/client.ts` | `apiGet`, `ApiError`, `errorMessage`, `isAbortError` |
| `api/cards.ts` | 엔드포인트 함수 (`searchCards`, `getMovers`, `getCard`, `getCardPrices`). 목업 전환 지점 |
| `api/mock/` | 목업 (`handlers.ts`, 자동 생성 `data.ts`) |
| `stores/` | Pinia 스토어 (`movers.ts`, `search.ts`, `cardDetail.ts`) |
| `views/` | 라우트 화면 |
| `components/layout/` | 헤더·푸터 |
| `components/common/` | 공용: `CardImage`, `LangBadge`, `StateView`, `PageNav` |
| `components/cards/`, `components/movers/` | 기능별 컴포넌트 |
| `components/card-detail/` | 카드 상세 (#13): `CardHeader`, `PriceHeadline`, `PriceTable`, `LinkedCards`, `PriceHistory`, `PriceChart`, `PriceSummary` |
| `utils/format.ts` | 금액·원화·변동률·날짜 표시 형식, 오래된 가격 판정(`isStale`), 소스·변형 표시 이름, 출처 문구(`SOURCE_ATTRIBUTION`) |
| `utils/chart.ts` | 차트 축 범위·눈금, 좌표, 끊긴 선 경로 (#13) |
| `styles/tokens.css`, `base.css` | 디자인 변수, 기본 스타일 |

### 3. 라우트

| 경로 | 이름 | 화면 | 상태 |
|------|------|------|------|
| `/` | `home` | 큰 검색창, 급등·급락 (7일·30일) | #12 |
| `/search?q=&lang=&page=` | `search` | 검색 결과 (언어 필터, 카드 격자, 쪽 넘김) | #12 |
| `/cards/:id` | `card` | 카드 상세 (정보·대표가·가격 비교 표·연결 카드·시세 추이) | #13 |
| 그 밖 | `not-found` | 찾을 수 없음 | #12 |

- 검색 화면은 **URL이 상태의 기준**이다. 필터·쪽 변경은 `router.push`로 URL만 바꾸고, URL 감시가 검색을 부른다. 잘못된 값(`page=abc`, 모르는 `lang`)은 `router.replace`로 고친다.
- 헤더 검색창은 홈에서 숨긴다 (홈에 큰 검색창이 있음).

### 4. 상태 관리 규칙

- 스토어는 **API 결과와 로딩·오류만** 보관한다. 화면 선택 상태(기간 탭 등)는 컴포넌트에, 공유가 필요한 질의는 URL에 둔다.
- 같은 질의는 다시 부르지 않는다 (`movers`: 기간·방향별 보관, `search`: 마지막 질의와 같으면 건너뜀). "다시 시도"는 `force`.
- 새 검색은 이전 요청을 취소한다 (`AbortController`). 취소된 요청의 오류는 화면에 보이지 않는다 (`isAbortError`).
- `cardDetail`: 카드 상세는 `id`별, 추이는 `id:기간`별로 보관한다. 다른 화면에 갔다 돌아오거나 기간 탭을 오가도 다시 부르지 않는다. 최근 카드 20장(`MAX_CARDS`)까지 보관하고, 넘으면 가장 오래 쓰지 않은 카드의 상세·추이를 함께 지운다. 404는 `notFound`로 구분해 찾을 수 없음 화면을 보인다.

### 5. API 클라이언트와 목업

- 모든 호출은 `/api` 기준이다. 개발 서버는 Vite 프록시가 `/api`를 떼어 `BACKEND_URL`로 넘긴다. 배포 구성(#16)에서 같은 규칙을 맞춘다.
- 오류는 `ApiError(status, code, message)`. `status 0`은 연결 실패(`network`)·시간 초과(`timeout`, 기본 10초).
- 목업: `src/frontend/.env.development`의 `VITE_API_MOCK=true`. #8 이후 기본값은 `false`(실제 API, Vite 프록시 → 백엔드)이고, 백엔드 없이 화면을 볼 때만 `true`로 바꾼다(프론트엔드 재시작 필요). 화면·스토어는 목업 여부를 모른다. 운영 빌드에는 목업 코드가 들어가지 않는다 (#12 3단계 확인).
- 목업 개발 도구: `?mock=empty`·`?mock=error`(빈 결과·오류 화면 확인), 검색어 `__error`, `globalThis.__mockStats`(호출 수). 자세한 내용은 `api_contract.md` 4절.

### 6. 스타일 규칙

- 색·간격·글꼴은 `styles/tokens.css` 변수만 쓴다. 어두운 모드는 `prefers-color-scheme`로 변수만 바꾼다.
- 상승은 빨강(`--color-up`), 하락은 파랑(`--color-down`) — 국내 시세 표기 관례.
- 반응형 기준: 모바일 기본, `640px`·`1024px` 이상에서 넓힌다. 본문 폭은 `.container`(최대 1200px, 좌우 여백 16px/24px).
- 컴포넌트 이름은 두 단어 이상 (`vue/multi-word-component-names`).

### 6-1. 카드 상세 화면 규칙 (#13)

- **가격 표시**: 원화를 크게, 원래 금액(소스 통화)을 작게 쓴다. 원화가 없으면 "환율 없음", 환율 날짜가 가격 기준일과 다르면 "환율 N월 N일"을 덧붙인다. 카드 타일·순위 줄은 원래 금액 뒤에 `≈₩655`를 덧붙인다.
- **오래된 가격**: 가격 기준일이 오늘(KST)보다 3일(`STALE_DAYS`) 넘게 이전이면 "오래된 가격" 표시.
- **가격 비교 표**: 변형 × 소스별 최신가를 섞지 않고 줄마다 보인다. 대표가 줄은 강조. 640px 미만은 카드형 목록.
- **출처**: 표 아래와 푸터에 소스별 출처 문구. `SOURCE_ATTRIBUTION`은 DB `price_sources.attribution`과 같은 문구라 마이그레이션 문구가 바뀌면 함께 고친다.
- **차트** (`PriceChart`, 패키지 없음): 원화 기준, 소스별 선(첫 선 `--color-accent`, 둘째 `--color-series-2`). 선은 SVG `viewBox 0~1000`을 늘려 그리고(`vector-effect: non-scaling-stroke`), 축 글자·점은 HTML을 백분율 위치에 겹친다. 빠진 날·원화 없는 날은 선을 끊는다. 가로축 끝은 `from + 기간`이다.
- **세로축** (`valueAxis`): 위아래 8% 여백, 눈금 간격 1·2·5 × 10ⁿ, 최소=최대면 ±5%, 0 아래로 내리지 않는다. **가로축** (`dateTicks`): 90일 이하는 날짜 5개(`11/3`), 1년은 매월(`3월`, 1월은 `2026년 1월`).
- **점 정보**: 마우스·터치로 가장 가까운 날짜, 키보드 ←/→·Home/End, Esc로 닫기. 값은 차트 위 한 줄에 쓴다(`aria-live`, 떠 있는 상자는 360px에서 넘쳐 쓰지 않음). 터치는 `touch-action: pan-y`로 세로 스크롤을 막지 않는다.
- **기간 요약** (`PriceSummary`): 평균·최저·최고는 API가 소스 통화로만 주므로 환산하지 않는다. 첫 가격이 기간 시작보다 늦으면 "이력이 짧음". 올해가 아니면 연도까지 쓴다.
- **화면 상태**: 기간(기본 30일)·변형(기본 대표가 변형)은 컴포넌트 상태다. 새 기간을 받는 동안 이전 기간 추이를 흐리게 유지한다.

### 7. 개발·확인 방법

```bash
# Docker/ 에서
docker compose exec frontend npm run typecheck   # 타입 검사
docker compose exec frontend npm run lint        # ESLint
docker compose exec frontend npm run build       # 타입 검사 + 운영 빌드
```

- `vite.config.ts`·`tsconfig*.json`을 바꾸면 개발 서버를 다시 시작한다 (`docker compose restart frontend`). 설정 파일 교체는 HMR로 반영되지 않는다.
- 의존성을 추가하면 컨테이너 안에서 `npm install`한다 (`node_modules`는 Docker 볼륨).
- 화면 확인(헤드리스 Chromium): Playwright 컨테이너에서 개발 서버에 접속한다. Vite는 낯선 호스트 이름(`host.docker.internal`, 서비스 이름 `frontend`)을 막으므로(403) **IP**로 접속한다.
- 목업 화면 확인 (`.env.development`는 그대로 둔다): 컨테이너 안에서 목업 개발 서버를 하나 더 띄우고, Playwright 컨테이너를 같은 Docker 네트워크(`pokecardprice_default`)에 붙여 frontend 컨테이너 IP로 접속한다.

```bash
docker compose exec -d -e VITE_API_MOCK=true frontend npx vite --host 0.0.0.0 --port 5174 --strictPort
docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' pokecardprice-frontend-1   # 예: 172.18.0.5
# 확인이 끝나면 docker compose restart frontend 로 목업 서버를 정리한다
```

## 결론

1. 화면은 API 계약 타입과 `src/api/cards.ts`에만 의존하고, 목업은 그 한 곳에서 바뀐다.
2. 검색 질의는 URL이 기준이라 새로고침·뒤로 가기·링크 공유가 그대로 된다.
3. 스타일은 CSS 변수와 공용 컴포넌트(`CardImage`, `LangBadge`, `StateView`, `PageNav`)로 통일한다.

## 적용 방향 (인계)

| 타스크 | 인계 사항 |
|--------|-----------|
| #8 | `api_contract.md`대로 구현, `VITE_API_MOCK=false`로 전환 (완료) |
| #13 | 카드 상세 화면 구현 (완료, 6-1절) |
| #14 | 라우트 `/login`, `/collection` 추가, 헤더에 메뉴 자리 |
| #15 | 단위 테스트 도구 결정 (스토어·`format.ts`·`chart.ts`·`PageNav` 계산이 우선 대상) |
| #16 | `createWebHistory`라 웹 서버가 모든 경로에 `index.html`을 돌려줘야 한다. `/api` 경로 전달 규칙을 개발 서버와 맞춘다 |

## 참고 자료

- `mydocs/tech/api_contract.md`
- `mydocs/plans/task_m100_12.md`, `task_m100_12_impl.md`
- `mydocs/plans/task_m100_13.md`, `task_m100_13_impl.md`
- `mydocs/working/task_m100_12_stage1.md` ~ `stage4.md`
