# Task #13: 카드 상세 화면 — 구현 계획서

수행 계획서: `mydocs/plans/task_m100_13.md` (Q1~Q5 모두 제안으로 결정)

## 공통 규칙

### 변경 범위

- `src/frontend/`와 문서만 바꾼다. 백엔드·수집기·`Docker/`는 바꾸지 않는다. API 계약을 바꿔야 하면 작업지시자에게 먼저 묻는다.
- 새 패키지는 넣지 않는다 (Q1).
- #12 규칙을 따른다: `<script setup lang="ts">`, CSS 변수(`tokens.css`)만, 두 단어 이상 컴포넌트 이름, 스토어에는 API 결과·로딩·오류만, 화면 선택 상태는 컴포넌트, 공유할 상태는 URL.

### 개발 환경

```bash
# Docker/ 에서
docker compose exec frontend npm run typecheck
docker compose exec frontend npm run lint
docker compose exec frontend npm run build
```

- 실제 API: 기본(`VITE_API_MOCK=false`). 개발 DB 이력은 2~3일치.
- 목업 확인: `.env.development`를 바꾸지 않고, 4단계에서 `VITE_API_MOCK=true`를 환경 변수로 준 별도 Vite 프로세스(다른 포트)를 컨테이너 안에서 잠시 띄워 확인한다. 커밋하는 `.env.development`는 `false` 그대로 둔다.
- 화면 확인: #8과 같은 Playwright 컨테이너(`mcr.microsoft.com/playwright:v1.49.0-noble`), 호스트 IP로 접속. 스크립트·캡처는 세션 임시 폴더에만 둔다.

## 화면 구성

```
┌ 데스크톱 (1024px 이상) ─────────────────────────────────────────┐
│ [이미지 high]   이름 / 한글 종 이름 / 언어·한글판 배지          │
│                 세트 로고·이름 · 번호 · 희귀도 · 발매일          │
│                 대표가 ₩16,441  ($12.30 · TCGplayer · 11-23)    │
│                 연결 카드 (한글판 ↔ 일본판 작은 타일)            │
├─────────────────────────────────────────────────────────────────┤
│ 가격 비교 표: 변형 | 소스 | 원화 | 원래 금액 | 기준일 | 상태    │
│ 출처 문구                                                       │
├─────────────────────────────────────────────────────────────────┤
│ 시세 추이  [7일][30일][90일][1년]   변형 [normal ▾]             │
│ ┌ SVG 차트 (소스별 선, 원화) ┐  ┌ 기간 요약 (소스별) ┐          │
│ └────────────────────────────┘  └───────────────────┘          │
└─────────────────────────────────────────────────────────────────┘
모바일(360px): 이미지 → 정보 → 대표가 → 표(가로 줄 대신 카드형 목록) → 차트 → 요약 세로 배치
```

## 파일 구성

| 파일 | 내용 | 단계 |
|------|------|------|
| `views/CardDetailView.vue` | 라우트 화면. `id` 검증, 스토어 호출, 상태 분기, 탭 제목 | 1 |
| `views/CardPlaceholderView.vue` | 삭제 | 1 |
| `stores/cardDetail.ts` | 카드 상세(`id`별)·추이(`id:period`별) 보관, 로딩·오류·404 구분, 요청 취소 | 1 |
| `components/card-detail/CardHeader.vue` | 이미지·이름·세트·번호·희귀도·배지·대표가 | 1·2 |
| `components/card-detail/PriceTable.vue` | 변형 × 소스 가격 표 (Q2·Q4), 출처 (Q5) | 2 |
| `components/card-detail/LinkedCards.vue` | 연결 카드 타일, 한글판 안내 | 2 |
| `components/card-detail/PriceHistory.vue` | 기간 탭·변형 선택·차트·요약 묶음 | 3 |
| `components/card-detail/PriceChart.vue` | SVG 선 그래프 (Q1·Q3) | 3 |
| `components/card-detail/PriceSummary.vue` | 소스별 기간 요약 | 3 |
| `utils/format.ts` | `formatKrw`, `formatShortDate`, `isStale`, 변형·소스 표시 이름 | 2 |
| `utils/chart.ts` | 축 범위·눈금 계산, 점 → 좌표, 끊긴 선 경로 | 3 |
| `components/cards/CardTile.vue`, `movers/MoverRow.vue` | `≈₩655` 덧붙임 (Q2) | 2 |
| `components/layout/AppFooter.vue` | 가격 출처 (Q5) | 2 |
| `router/index.ts` | `card` 라우트를 `CardDetailView`로 | 1 |
| `api/mock/handlers.ts` | 목업 상세에 변형 2개·소스 2개, 추이에 기간별 점(7·30·90·365)과 일부 원화 null | 3 |

## 세부 규칙

### 스토어 (`cardDetail.ts`)

- `cards: Record<id, CardDetail>`, `histories: Record<"id:period", PriceHistoryResponse>`, 각각 `loading`·`errors` (오류는 `{ status, message }`로 404를 구분).
- 같은 키에 결과가 있으면 다시 부르지 않는다. "다시 시도"는 `force`.
- 카드를 바꾸면(라우트 `id` 변경) 진행 중인 상세·추이 요청을 취소한다 (`AbortController`, 취소 오류는 무시).
- 보관 개수 제한: 카드 20장. 넘으면 오래된 카드의 상세·추이를 함께 지운다 (1년 추이 약 78 KB).

### 화면 상태

| 상태 | 표시 |
|------|------|
| `id`가 양의 정수가 아님 | 찾을 수 없음 (요청하지 않음) |
| 상세 로딩 | `StateView` 로딩 (이미지 자리 유지) |
| 404 | "카드를 찾을 수 없습니다" + 홈·검색 링크 |
| 그 밖 오류 | 오류 문구 + 다시 시도 |
| 시세 없음 (`variants`의 가격이 모두 비어 있음) | 가격 표·차트 대신 "아직 시세가 없습니다". 한글판이면 "한글판 시세는 준비 중입니다"와 연결된 일본판 링크 |
| 추이 로딩·오류 | 차트 영역 안에서만 (상세는 그대로 보임) |
| 점 1개 | 점 하나와 "이력이 쌓이면 추이가 표시됩니다" |

- 탭 제목: 상세를 받으면 `{이름} ({세트 이름} {번호}) · Card-Market`, 그 전에는 `카드 상세`.

### 원화 표시 (Q2)

- 상세: 원화를 크게(`₩16,441`), 원래 금액·소스·기준일을 작게(`$12.30 · TCGplayer · 11월 23일`). 원화가 없으면(환율 없음) 원래 금액을 크게 쓰고 "환율 정보 없음".
- 홈·검색 타일·순위 줄: 지금 표시 뒤에 작은 글씨 `≈₩655`. 원화가 없으면 덧붙이지 않는다.
- 환율 기준일이 가격 기준일과 다르면 표에 "환율 11월 21일 기준"을 작게 쓴다.

### 오래된 가격 (Q4)

- 가격 기준일 < 오늘(KST) − 3일이면 표의 상태 칸에 "오래된 가격"(흐린 색), 대표가 옆에도 같은 표시.
- 오늘(KST)은 브라우저 시각을 `Asia/Seoul`로 바꿔 구한다 (`Intl.DateTimeFormat`).

### 표시 이름

| 코드 | 화면 |
|------|------|
| `tcgcsv` | TCGplayer |
| `tcgdex_cardmarket` | Cardmarket |
| `normal` · `holofoil` · `reverse_holofoil` | 일반 · 홀로 · 리버스 홀로 |
| `first_edition` · `first_edition_holofoil` | 1st 에디션 · 1st 에디션 홀로 |
| `unlimited` · `unlimited_holofoil` | 언리미티드 · 언리미티드 홀로 |
| 그 밖 | 코드 그대로 |

출처 문구(Q5)는 `price_sources.attribution`과 같은 문구를 프론트엔드 상수로 둔다 ("TCGplayer 가격 (TCGCSV 경유)", "Cardmarket 가격 (TCGdex 경유)"). API에 없는 값이라 상수로 두며, 바뀌면 함께 고친다(문서에 적음).

### 추이 차트 (Q1·Q3)

- 기간 탭 기본 30일, 변형 기본은 대표가 변형(없으면 첫 계열의 변형). 선택은 컴포넌트 상태(URL에 두지 않음).
- 선택 변형의 계열(소스별 최대 2개)을 원화(`krw`)로 그린다. `krw`가 null인 점은 선을 끊는다. 모든 점이 null이면 "환율 정보가 없어 원화 추이를 그릴 수 없습니다"와 소스 통화 요약만.
- 가로축: 기간 시작일(`from`) ~ 오늘(KST), 날짜 비례. 눈금 4~6개 (`11/1` 형식, 1년은 월).
- 세로축: 선택 계열 원화 최소~최대에 여백 8%, 보기 좋은 눈금 4~5개(1·2·5 × 10ⁿ). 최소=최대면 위아래로 ±5%.
- 선 색: TCGplayer `--color-accent`, Cardmarket 새 변수 `--color-series-2` (밝은·어두운 모드 각각). 범례는 차트 위. 상승·하락 색은 선에 쓰지 않는다(요약 변동률에만).
- 점 정보: 마우스 이동·터치 → 가장 가까운 날짜의 세로선과 소스별 값(원화·원래 금액). 키보드: 차트에 포커스 후 ←/→로 날짜 이동, Esc로 닫기. SVG에 `role="img"`·`aria-label`(기간·현재가 요약).
- 크기: `viewBox` 고정 비율, 폭은 부모 100%, 높이 모바일 200px·데스크톱 280px. `ResizeObserver` 없이 `viewBox`와 `preserveAspectRatio`로 늘리되 글자는 HTML 겹침으로 그려 늘어나지 않게 한다.

### 기간 요약

- 소스별 줄: 현재가(원화·원래 금액), 평균, 최저, 최고(원래 통화), 변동률(상승 빨강·하락 파랑, null이면 "—"), "11월 22일부터" (`summary.from`이 `from`보다 늦으면 "이력이 짧음").
- 평균·최저·최고는 API가 소스 통화로만 주므로 원래 통화로 표시한다 (화면이 환산하지 않는다).

## 단계

### 1단계: 스토어와 화면 골격

- `stores/cardDetail.ts`, `views/CardDetailView.vue`, `CardHeader.vue`(대표가 제외), 라우트 교체, 자리 화면 삭제.
- 상태 분기(잘못된 id, 로딩, 404, 오류·다시 시도), 탭 제목, 카드 간 이동 시 요청 취소.
- 확인: typecheck·lint·build, 실제 API로 영어·일본·한글판 카드, `/cards/999999`, `/cards/abc`.

### 2단계: 가격 비교

- `format.ts` 보강, `PriceTable.vue`, `LinkedCards.vue`, `CardHeader` 대표가, 한글판 안내, 시세 없음.
- `CardTile`·`MoverRow` 원화 덧붙임, `AppFooter` 가격 출처.
- 확인: 영어판(변형 2개·소스 2개), 일본판(Cardmarket만), 한글판(연결·안내), 트레이너스(시세 없음), 원화 null(환율 없는 경우는 목업으로).

### 3단계: 추이 차트

- `utils/chart.ts`, `PriceChart.vue`, `PriceSummary.vue`, `PriceHistory.vue`, 목업 추이 보강, `--color-series-2`.
- 확인: 목업(7·30·90·365점, 끊긴 선, 최소=최대), 실제 API(점 1~3개), 키보드·터치 동작.

### 4단계: 마무리·확인·문서

- 반응형(360·768·1280px), 어두운 모드, 긴 이름·긴 세트 이름, 이미지 없음.
- 헤드리스 확인: 실제 API와 목업, 콘솔 오류·가로 스크롤 없음, 탭 전환 시 재요청 없음(요청 수 확인).
- 문서: `frontend_structure.md`(라우트·스토어·컴포넌트·차트 규칙·출처 상수), `api_contract.md` 4절 목업 변경, `CLAUDE.md`는 필요 시 한 줄.

각 단계 끝에 단계별 완료 보고서(`mydocs/working/task_m100_13_stage{N}.md`)를 쓰고 승인 후 소스와 함께 커밋한다.
