# 기술 조사: 프론트엔드 구조

> **작성일**: 2026-11-17
> **관련 타스크**: #12 (작성), #13·#14·#8·#16 (인계)

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
| `stores/` | Pinia 스토어 (`movers.ts`, `search.ts`) |
| `views/` | 라우트 화면 |
| `components/layout/` | 헤더·푸터 |
| `components/common/` | 공용: `CardImage`, `LangBadge`, `StateView`, `PageNav` |
| `components/cards/`, `components/movers/` | 기능별 컴포넌트 |
| `utils/format.ts` | 금액·변동률·날짜 표시 형식 |
| `styles/tokens.css`, `base.css` | 디자인 변수, 기본 스타일 |

### 3. 라우트

| 경로 | 이름 | 화면 | 상태 |
|------|------|------|------|
| `/` | `home` | 큰 검색창, 급등·급락 (7일·30일) | #12 |
| `/search?q=&lang=&page=` | `search` | 검색 결과 (언어 필터, 카드 격자, 쪽 넘김) | #12 |
| `/cards/:id` | `card` | 자리 화면 | #13에서 구현 |
| 그 밖 | `not-found` | 찾을 수 없음 | #12 |

- 검색 화면은 **URL이 상태의 기준**이다. 필터·쪽 변경은 `router.push`로 URL만 바꾸고, URL 감시가 검색을 부른다. 잘못된 값(`page=abc`, 모르는 `lang`)은 `router.replace`로 고친다.
- 헤더 검색창은 홈에서 숨긴다 (홈에 큰 검색창이 있음).

### 4. 상태 관리 규칙

- 스토어는 **API 결과와 로딩·오류만** 보관한다. 화면 선택 상태(기간 탭 등)는 컴포넌트에, 공유가 필요한 질의는 URL에 둔다.
- 같은 질의는 다시 부르지 않는다 (`movers`: 기간·방향별 보관, `search`: 마지막 질의와 같으면 건너뜀). "다시 시도"는 `force`.
- 새 검색은 이전 요청을 취소한다 (`AbortController`). 취소된 요청의 오류는 화면에 보이지 않는다 (`isAbortError`).

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

### 7. 개발·확인 방법

```bash
# Docker/ 에서
docker compose exec frontend npm run typecheck   # 타입 검사
docker compose exec frontend npm run lint        # ESLint
docker compose exec frontend npm run build       # 타입 검사 + 운영 빌드
```

- `vite.config.ts`·`tsconfig*.json`을 바꾸면 개발 서버를 다시 시작한다 (`docker compose restart frontend`). 설정 파일 교체는 HMR로 반영되지 않는다.
- 의존성을 추가하면 컨테이너 안에서 `npm install`한다 (`node_modules`는 Docker 볼륨).
- 화면 확인(헤드리스 Chromium): Playwright 컨테이너에서 개발 서버에 접속한다. Vite는 낯선 호스트 이름(`host.docker.internal`)을 막으므로(403) 호스트 **IP**로 접속한다.

## 결론

1. 화면은 API 계약 타입과 `src/api/cards.ts`에만 의존하고, 목업은 그 한 곳에서 바뀐다.
2. 검색 질의는 URL이 기준이라 새로고침·뒤로 가기·링크 공유가 그대로 된다.
3. 스타일은 CSS 변수와 공용 컴포넌트(`CardImage`, `LangBadge`, `StateView`, `PageNav`)로 통일한다.

## 적용 방향 (인계)

| 타스크 | 인계 사항 |
|--------|-----------|
| #8 | `api_contract.md`대로 구현, `VITE_API_MOCK=false`로 전환 (완료) |
| #13 | `/cards/:id` 자리 화면(`CardPlaceholderView.vue`)을 교체. `CardImage size="high"`, `LangBadge`, `formatPrice` 재사용. 카드 상세 API 계약은 `api_contract.md`에 추가 |
| #14 | 라우트 `/login`, `/collection` 추가, 헤더에 메뉴 자리 |
| #15 | 단위 테스트 도구 결정 (스토어·`format.ts`·`PageNav` 계산이 우선 대상) |
| #16 | `createWebHistory`라 웹 서버가 모든 경로에 `index.html`을 돌려줘야 한다. `/api` 경로 전달 규칙을 개발 서버와 맞춘다 |

## 참고 자료

- `mydocs/tech/api_contract.md`
- `mydocs/plans/task_m100_12.md`, `task_m100_12_impl.md`
- `mydocs/working/task_m100_12_stage1.md` ~ `stage4.md`
