# Task #12 단계1 완료 보고서: TypeScript 전환, 개발 도구, 라우터·Pinia·레이아웃 골격

## 수행 내용

- **TypeScript 전환 (Q4)**: `main.js`·`vite.config.js`를 `.ts`로 바꾸고, `tsconfig.json`(참조), `tsconfig.app.json`(`@vue/tsconfig` DOM 설정 확장, `strict`, `noUncheckedIndexedAccess`, `@/*` 별칭), `tsconfig.node.json`(`vite.config.ts`용), `src/env.d.ts`(`VITE_API_MOCK` 선언)를 추가했다. 빌드 전에 `vue-tsc -b`로 타입 검사를 한다.
- **ESLint**: `eslint-plugin-vue` `flat/recommended` + `@vue/eslint-config-typescript` `recommended`. 설정 파일은 `eslint.config.js`로 두었다 (아래 발견 사항).
- **라우터·Pinia**: `vue-router` 5, `pinia` 4를 등록했다. 라우트는 `/`(홈), `/search`(검색), `/cards/:id`(#13 자리 화면), 그 밖 경로(찾을 수 없음). 홈 외 화면은 필요할 때 불러온다(코드 분할). 라우트 `meta.title`로 브라우저 탭 제목을 바꾼다.
- **레이아웃**: 헤더(서비스 이름, 3단계 검색창 자리, 스크롤해도 위에 고정), 본문, 푸터(TCGdex·PokéAPI 출처, 시세 참고용·상표 고지), 본문으로 건너뛰기 링크.
- **스타일**: `style.css`를 `styles/tokens.css`(색·간격·글꼴 변수, 상승 빨강·하락 파랑, 어두운 모드)와 `styles/base.css`(초기화, `.container` 최대 폭 1200px·좌우 여백 16px/640px 이상 24px, 포커스 표시)로 나눴다.
- **개발 환경**: worktree `Docker/.env`(`FRONTEND_PORT=5174`, Git 제외)로 `pokecardprice-t12` 프로젝트의 `frontend`만 띄웠다. #5의 `backend`·`frontend` 컨테이너는 건드리지 않았다.

## 변경 파일

| 파일 | 변경 내용 |
|------|----------|
| `src/frontend/package.json`, `package-lock.json` | 의존성 추가, 스크립트 `build`(타입 검사 포함)·`typecheck`·`lint` |
| `src/frontend/tsconfig.json`, `tsconfig.app.json`, `tsconfig.node.json` | 신규. TypeScript 설정 |
| `src/frontend/vite.config.ts` | `vite.config.js`에서 옮김. `@` 별칭 추가 (프록시·polling 설정은 그대로) |
| `src/frontend/eslint.config.js` | 신규. ESLint 설정 |
| `src/frontend/index.html` | 진입 파일을 `main.ts`로 |
| `src/frontend/src/main.ts` | `main.js`에서 옮김. Pinia·라우터 등록 |
| `src/frontend/src/env.d.ts` | 신규. 환경 변수 타입 |
| `src/frontend/src/router/index.ts` | 신규. 라우트, 탭 제목, 스크롤 복원 |
| `src/frontend/src/App.vue` | 임시 상태 화면 → 레이아웃 |
| `src/frontend/src/components/layout/AppHeader.vue`, `AppFooter.vue` | 신규 |
| `src/frontend/src/views/*.vue` | 신규. 홈·검색 자리 화면, 카드 자리 화면(#13), 찾을 수 없음 |
| `src/frontend/src/styles/tokens.css`, `base.css` | 신규 (`style.css` 삭제) |
| `mydocs/plans/task_m100_12.md`, `task_m100_12_impl.md` | 수행·구현 계획서 (승인됨) |
| `mydocs/orders/20261117.md` | 파일 끝 #12 절 |

백엔드·수집기·`Docker/` 파일은 바꾸지 않았다.

## 설치한 패키지

| 패키지 | 설치 버전 |
|--------|-----------|
| `vue-router` | ^5.5.1 |
| `pinia` | ^4.1.0 |
| `typescript` | ~6.0.3 |
| `vue-tsc` | ^3.3.14 |
| `@vue/tsconfig` | ^0.9.1 |
| `@types/node` | ^26.6.6 |
| `eslint` | ^10.15.0 |
| `eslint-plugin-vue` | ^10.12.1 |
| `typescript-eslint` | ^8.73.0 |
| `@vue/eslint-config-typescript` | ^14.9.0 |

peer 충돌 없음, `npm audit` 취약점 0건.

## 발견 사항

- **ESLint 설정 파일을 `.ts`가 아닌 `.js`로 두었다.** ESLint 10은 TypeScript 설정 파일을 읽으려면 `jiti` 패키지를 추가로 요구한다. 설정 파일 하나 때문에 의존성을 늘리지 않으려고 JS로 두었다. 설정 내용은 같다.
- **TypeScript 6.0에서는 `baseUrl`이 더 이상 권장되지 않아** `paths`만으로 `@/*` 별칭을 정했다.
- **설정 파일을 바꾸면 개발 서버를 다시 시작해야 한다.** `vite.config.js`를 지우고 `.ts`로 새로 만들었을 때 실행 중인 서버가 옛 설정(별칭 없음)으로 남아 500 오류가 났다. #12 `frontend` 컨테이너를 재시작해 해결했다. 4단계 문서에 적는다.
- **화면 확인은 헤드리스 Chromium으로 했다.** 브라우저 확장 대신 Playwright 컨테이너(`mcr.microsoft.com/playwright:v1.49.0-noble`)로 화면을 열어 캡처하고, 제목·가로 스크롤·콘솔 오류를 확인했다. Vite가 IP가 아닌 낯선 호스트 이름(`host.docker.internal`)을 막아(403) 호스트 IP로 접속했다. 확인용 스크립트·캡처는 세션 임시 폴더에만 두었다.

## 검증

- `npm run typecheck`: ✅
- `npm run lint`: ✅ (`.vue` 7개, `.ts` 4개, 설정 파일 포함 12개 파일 검사)
- `npm run build`: ✅ (홈 외 화면은 별도 청크, 기본 JS 93 kB / gzip 37 kB)
- `npm audit`: 취약점 0건
- 화면 (헤드리스 Chromium, 360px·1280px):

| 경로 | 탭 제목 | 가로 스크롤 | 콘솔 오류 |
|------|---------|-------------|-----------|
| `/` | Card-Market | 없음 | 없음 |
| `/search?q=피카츄` | 검색 · Card-Market | 없음 | 없음 |
| `/cards/42` | 카드 상세 · Card-Market | 없음 | 없음 |
| `/nope` | 페이지를 찾을 수 없습니다 · Card-Market | 없음 | 없음 |

- 각 경로를 주소로 직접 열어(새로고침과 같음) 해당 화면이 나온다: ✅
- 작업지시자 확인: http://localhost:5174 (`pokecardprice-t12` 프로젝트)

## 다음 단계

2단계: API 계약 초안(`mydocs/tech/api_contract.md`), API 클라이언트(`src/api/`), 목업 모드.

## 커밋

(승인 후 커밋)
