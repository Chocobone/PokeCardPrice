# Task #12 최종 결과보고서: 프론트엔드 기반 및 홈·검색 화면

> **작성일**: 2026-11-17
> **브랜치**: `local/task12` (`local/task5` `21970a8`에서 분기, worktree `../PokeCardPrice-task12`)
> **마일스톤**: M100 (v1.0.0)

## 개요

#1의 임시 화면을 실제 앱 골격으로 바꾸고, 홈 화면과 검색 결과 화면을 만들었다.
- 앱 골격: TypeScript, vue-router, Pinia, ESLint, API 클라이언트, 반응형 레이아웃.
- 홈 화면: 검색창, 7일·30일 급등·급락.
- 검색 결과 화면: 언어 필터, 카드 격자, 쪽 넘김.

검색·순위 API(#8)가 아직 없어, 화면이 따를 **API 계약 초안**을 먼저 정하고 **목업 모드**로 화면을 완성했다(Q1). 목업은 #5가 적재한 실제 카탈로그 표본(60장)을 쓴다. #8이 계약대로 API를 만들면 `VITE_API_MOCK=false`로 바꾸기만 하면 된다.

#5 전체 적재와 병행하려고 `local/task5`에서 분기한 별도 worktree와 별도 compose 프로젝트(`pokecardprice-t12`, 포트 5174, 프론트엔드만)에서 작업했다. #5와 병합 시 충돌이 없음을 확인했다.

## 수행 결과

| 단계 | 내용 | 커밋 |
|------|------|------|
| 1단계 | TypeScript 전환, vue-router·Pinia·ESLint 도입, 라우트 4개, 레이아웃(헤더·푸터), 디자인 변수 | `9f3de6d` |
| 2단계 | API 계약 초안(`api_contract.md`), API 클라이언트(시간 초과·취소·오류 형식), 목업(개발 DB 표본) | `38c3f49` |
| 3단계 | 홈 화면: 큰 검색창, 헤더 검색창, 급등·급락(7일·30일, 결과 보관), 공용 컴포넌트, 빈·오류 상태 | `8abef4c` |
| 4단계 | 검색 결과 화면(URL 동기화, 언어 필터, 카드 격자, 쪽 넘김), 목업 다시 생성(60장), 구조 문서 | `9beacf1` |

## 산출물

### 화면

| 경로 | 화면 |
|------|------|
| `/` | 큰 검색창, 급등·급락 (기간 탭 7일·30일, 상승·하락 각 10위) |
| `/search?q=&lang=&page=` | 검색 결과. 언어 필터(전체·영어판·일본판·한글판), 카드 격자(모바일 2열 ~ 데스크톱 6열), 쪽 넘김(한 쪽 24장) |
| `/cards/:id` | 자리 화면 (#13에서 구현) |
| 그 밖 | 찾을 수 없음 |

### 구조

| 영역 | 파일 |
|------|------|
| API | `api/types.ts`(계약 타입), `api/client.ts`(`apiGet`, `ApiError`, 시간 초과 10초), `api/cards.ts`(엔드포인트, 목업 전환 지점), `api/mock/` |
| 상태 | `stores/movers.ts`(기간·방향별 결과 보관), `stores/search.ts`(같은 질의 건너뜀, 이전 요청 취소) |
| 공용 컴포넌트 | `CardImage`, `LangBadge`, `StateView`, `PageNav`, `SearchBox` |
| 기능 컴포넌트 | `movers/`(`MoversPanel`, `MoverList`, `MoverRow`), `cards/`(`CardGrid`, `CardTile`) |
| 스타일 | `styles/tokens.css`(색·간격·글꼴 변수, 어두운 모드), `base.css` |

자세한 구조·규칙·인계 사항은 `mydocs/tech/frontend_structure.md`, API 계약은 `mydocs/tech/api_contract.md`에 있다.

### 작업지시자 결정 사항

| # | 결정 |
|---|------|
| Q1 | #8 의존: API 계약 초안 + 목업 모드로 화면 완성, #8이 계약을 따른다 |
| Q2 | UI 프레임워크 없음 (CSS 변수, 직접 작성 컴포넌트) |
| Q3 | 단위 테스트 도구는 도입하지 않고 #15에서 정한다 (대안) |
| Q4 | TypeScript로 전환 (대안). `typescript-eslint` 지원 범위 때문에 TypeScript 6.0으로 고정 |

### 사용 방법

```bash
# Docker/ 에서 (병합 후에는 기본 frontend 컨테이너, http://localhost:5173)
docker compose exec frontend npm run typecheck   # 타입 검사
docker compose exec frontend npm run lint        # ESLint
docker compose exec frontend npm run build       # 타입 검사 + 운영 빌드
```

- 목업 모드: `src/frontend/.env.development`의 `VITE_API_MOCK=true` (현재 기본). `?mock=empty`·`?mock=error`로 빈·오류 화면, 검색어 `__error`로 검색 오류 화면을 볼 수 있다.
- `vite.config.ts`·`tsconfig*.json`을 바꾸면 `docker compose restart frontend`.

## 발견 사항

- **URL 정규화 비교 버그 (4단계에서 수정)**: 고친 주소를 `fullPath` 문자열로 비교하면, 공백이 있는 검색어(`sv03.5 25`)에서 공백 인코딩(`%20`·`+`)만 달라 같은 위치로 `replace`하게 된다. 라우터가 이를 중복 이동으로 무시해 검색이 실행되지 않았다. 쿼리 값을 비교하도록 바꿨다.
- **최신 일본판·한글판 카드는 TCGdex에 이미지가 없다.** 화면은 "이미지 없음" 자리 표시를 보여 준다. 한글판은 연결된 일본판 이미지를 쓸 수 있다 (#13 인계).
- **Vite 개발 서버 특성**:
  - 설정 파일 교체는 HMR로 반영되지 않아 재시작이 필요하다.
  - 낯선 호스트 이름(`host.docker.internal`)은 403으로 막는다. 헤드리스 확인은 호스트 IP로 접속했다.
  - 파일을 고친 뒤에는 모듈이 `?t=` 주소로 다시 불려 인스턴스가 나뉜다. 목업 호출 수는 `globalThis.__mockStats`로 읽는다.
- **ESLint 10은 TS 설정 파일에 `jiti`가 필요하다.** 의존성을 늘리지 않으려고 `eslint.config.js`로 두었다.
- **운영 빌드 크기**: 앱 번들 110 kB (gzip 43 kB). 목업 코드는 동적 import로 운영 빌드에서 빠진다.
- **CRLF**: worktree에서 `git stash`를 쓰면 작업 파일이 CRLF로 바뀐다(`core.autocrlf=true`). 커밋 내용에는 영향이 없다.

## 검증

| 항목 | 결과 |
|------|------|
| `npm run typecheck` (`vue-tsc`) | ✅ |
| `npm run lint` | ✅ 오류·경고 0 |
| `npm run build` | ✅ |
| `npm audit` | 취약점 0 |
| 운영 빌드에 목업 코드 없음 | ✅ |
| 홈 화면 (360px·1280px) | 가로 스크롤 없음, 기간 탭 전환 시 결과 보관(재요청 없음), 빈·오류 상태, 검색 이동 ✅ (3단계) |
| 검색 화면 (360px·1280px) | 쪽 넘김·언어 필터·뒤로 가기·새로고침, 한글 종 이름 검색, 빠른 검색어 변경 시 마지막 결과만, 잘못된 값 정정, 빈·오류·검색어 없음 ✅ (4단계) |
| 홈 → 검색 → 필터·쪽 넘김 → 카드 자리 화면 흐름 | ✅ |
| API 계약 문서와 `src/api/types.ts` 일치 | ✅ |
| 백엔드·수집기·`Docker/` 변경 없음 | ✅ 분기점 대비 `src/frontend`·`mydocs` 밖은 `CLAUDE.md` 1줄 |
| `local/task5`와 병합 | 충돌 없음 (`git merge-tree`) ✅ |
| 작업지시자 직접 확인 | http://localhost:5174 (4단계) |

화면 확인은 단위 테스트 대신 헤드리스 Chromium(Playwright 1.49 컨테이너)으로 했다 (Q3).

## 계획 대비 차이

| 계획 | 실제 | 사유 |
|------|------|------|
| `eslint.config.ts` | `eslint.config.js` | TS 설정 파일은 `jiti` 의존성이 필요 |
| 2단계 확인: 홈에 임시 결과 건수 표시 | 헤드리스 브라우저에서 API 모듈을 직접 호출 | 화면에 지울 코드를 넣지 않기 위해 |
| `Pagination.vue` | `PageNav.vue` | ESLint `vue/multi-word-component-names` |
| 목업 표본: 일본판 종별 최신 2장 | 이미지가 있는 카드 중 최신 2장 + 한글판과 연결된 일본판 카드 (36장 → 60장) | 최신 일본판은 이미지가 없고, 한글판 배지 카드가 없었다 |

## 후속 조치

| 타스크 | 인계 사항 |
|--------|-----------|
| #8 | `api_contract.md`대로 `GET /cards`, `GET /rankings/movers` 구현. 완료 후 `VITE_API_MOCK=false`. 검색 일치 범위·정렬, 대표가·순위 산출 기준, 통화 표시 결정 |
| #13 | `/cards/:id` 자리 화면 교체. `CardImage`·`LangBadge`·`formatPrice` 재사용, 한글판 카드는 연결된 일본판 이미지 사용, 카드 상세 API 계약 추가 |
| #14 | `/login`·`/collection` 라우트, 헤더 메뉴 |
| #15 | 단위 테스트 도구 결정 (스토어, `format.ts`, `PageNav` 쪽 계산, `SearchView` URL 정규화가 우선 대상) |
| #16 | SPA 경로 처리(모든 경로에 `index.html`), `/api` 전달 규칙을 개발 서버와 맞춤 |

- 모바일(360px)에서 홈 검색창 안내 문구가 잘린다 (3단계 발견). 입력하면 사라지는 문구라 그대로 두었다.
- 병합 순서: #5를 먼저 `local/devel`에 병합한 뒤 #12를 병합한다. 병합 후 #12 worktree와 `pokecardprice-t12` 컨테이너는 정리한다.

## 이슈

- https://github.com/Chocobone/PokeCardPrice/issues/12
