# Task #1 Stage 3: 프론트엔드 골격 — 단계별 완료 보고서

> **작성일**: 2026-10-30
> **브랜치**: `local/task1`

## 작업 내용

프론트엔드 컨테이너 안에서 `create-vite`의 `vue` 템플릿(JavaScript)으로 프로젝트를 만들었다.
데모 파일은 지우고, 백엔드 `/healthz` 결과를 보여 주는 임시 화면, `/api` 프록시, 파일 감시 polling을 설정했다.

### 추가·변경 파일

| 파일 | 내용 |
|------|------|
| `src/frontend/package.json`, `package-lock.json` | 이름 `card-market-frontend`. Vue 3.5.44, Vite 8.3.5, `@vitejs/plugin-vue` 6.0.10 |
| `src/frontend/vite.config.js` | `/api/*` → `BACKEND_URL`(없으면 `http://localhost:8080`) 프록시, `/api` 접두어 제거. `watch.usePolling` 300ms |
| `src/frontend/index.html` | `lang="ko"`, 제목 `Card-Market` |
| `src/frontend/src/App.vue` | 화면이 열리면 `/api/healthz`를 호출해 "백엔드 상태: ok" 또는 실패 사유 표시 |
| `src/frontend/src/style.css` | 데모 스타일을 기본 글꼴·색상만 남긴 최소 스타일로 교체 |
| `src/frontend/src/main.js`, `public/favicon.svg`, `.gitignore` | 템플릿 기본값 유지 |
| 삭제 | `HelloWorld.vue`, `src/assets/*`, `public/icons.svg`, 템플릿 README, `.vscode/`, `src/frontend/.gitkeep` |

### 설계 메모

- `/app`에는 `node_modules` 네임드 볼륨이 마운트되어 있어 비어 있지 않다. 이 상태에서는 `create-vite`가 덮어쓰기 여부를 묻는다.
  그래서 컨테이너의 `/tmp/scaffold`에 프로젝트를 만든 뒤 `/app`으로 복사했다. 생성 명령에는 `--no-interactive`를 지정했다.
- 프록시는 `/api` 접두어를 떼고 백엔드로 넘긴다. 백엔드 라우트는 접두어 없이(`/healthz`) 유지하고, 프론트엔드는 항상 `/api/...`로 호출한다.
  배포 시 리버스 프록시도 같은 규칙을 따르면 된다(#16에서 확정).
- Windows 바인드 마운트에서는 호스트의 파일 변경 이벤트가 컨테이너로 전달되지 않는다. 그래서 Vite 파일 감시를 polling으로 설정했다.
- TypeScript, 라우터, Pinia, ESLint는 이번 단계에서 넣지 않았다. #12(프론트엔드 기반)에서 함께 도입한다.
- 현재 화면은 연결 확인용 임시 화면이다. #12에서 실제 홈 화면으로 바꾼다.

## 검증 결과

```
$ docker compose exec frontend sh -c 'cd /app && npm run build'
dist/index.html                  0.46 kB │ gzip:  0.29 kB
dist/assets/index-DecCAcv0.css   0.27 kB │ gzip:  0.21 kB
dist/assets/index-C_Js5doL.js   62.94 kB │ gzip: 24.94 kB
✓ built in 463ms

$ docker compose logs frontend
VITE v8.3.5  ready in 877 ms
➜  Local:   http://localhost:5173/

$ curl -s localhost:5173/api/healthz          # 프록시 경유 백엔드 응답
{"status":"ok"}

$ curl -s localhost:5173/ | grep -o '<title>.*</title>'
<title>Card-Market</title>

# 호스트에서 App.vue 수정 → 컨테이너가 변경 감지
[vite] (client) hmr update /src/App.vue
```

- `npm install` 결과 취약점 0건.
- 개발 서버가 보내는 컴파일된 `App.vue`에 `fetch('/api/healthz')`와 상태 표시 문구가 들어 있는 것을 확인했다.
- **화면 확인 대기**: 브라우저 도구를 쓰지 않아 실제 렌더링은 확인하지 못했다. 작업지시자가 `http://localhost:5173`에 접속해 "백엔드 상태: ok"(녹색)가 보이는지 확인해야 한다.

## 다음 단계

Stage 4 — 문서 정비: README를 프로젝트 소개·구조·실행 방법으로 교체, CLAUDE.md `빌드 및 실행` 절에 compose 기반 명령 기입
