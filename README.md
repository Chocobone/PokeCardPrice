# Card-Market (PokeCardPrice)

포켓몬 카드 시세 비교 웹 애플리케이션.

여러 거래처(TCGplayer, Cardmarket, 국내 거래가 등)의 포켓몬 카드 가격을 원화로 환산해 한 화면에서 비교하고,
날짜별 시세를 쌓아 과거와 현재 가격의 추이를 보여 준다.

> 현재 개발 초기 단계다. 개발 환경과 서버·화면 골격까지 구성되어 있다.

## 주요 기능 (계획)

- **홈·검색**: 카드 이름·세트·번호 검색, 급등·급락 순위
- **카드 상세**: 거래처별 원화 가격, 평균가, 기간별(7일/30일/90일/1년) 시세 추이 차트
- **내 컬렉션**: 로그인 사용자별 보유 카드 관리, 총 평가액과 기간별 증감

## 기술 스택

| 영역 | 기술 |
|------|------|
| 프론트엔드 | Vite + Vue 3 |
| 백엔드 | Go (`net/http`) |
| DB | PostgreSQL 18 |
| 개발 환경 | Docker Compose |

## 저장소 구조

```
PokeCardPrice/
├── Docker/                 ← 개발 컨테이너 정의
│   ├── docker-compose.yml  ← db, backend, frontend 서비스
│   ├── .env.example        ← 환경 변수 예시 (.env로 복사해 사용)
│   ├── backend/            ← Go 개발 이미지
│   └── frontend/           ← Node.js 개발 이미지
├── src/
│   ├── backend/            ← Go API 서버
│   │   ├── cmd/server/     ← 진입점
│   │   └── internal/       ← 내부 패키지
│   └── frontend/           ← Vue 웹 앱
├── docs/                   ← 텀프로젝트 계획서
└── mydocs/                 ← 작업 문서 (계획서, 보고서, 기술 조사)
```

## 실행 방법

호스트에는 [Docker](https://www.docker.com/)만 있으면 된다. Go와 Node.js는 컨테이너 안에서 실행한다.

```bash
cd Docker
cp .env.example .env            # 최초 1회, POSTGRES_PASSWORD 변경
docker compose up -d --build    # db → backend → frontend 기동
docker compose ps               # 상태 확인
```

| 서비스 | 주소 |
|--------|------|
| 프론트엔드 | http://localhost:5173 |
| 백엔드 | http://localhost:8080/healthz |
| PostgreSQL | localhost:5432 |

포트는 `Docker/.env`에서 바꿀 수 있다.

### 개발 명령

```bash
# Docker/ 폴더에서 실행
docker compose exec backend go test ./...     # 백엔드 테스트
docker compose exec backend go build ./...    # 백엔드 빌드
docker compose restart backend                # Go 소스 변경 반영
docker compose exec frontend npm run build    # 프론트엔드 빌드 (화면은 HMR로 자동 반영)
docker compose down                           # 중지 (데이터 유지)
docker compose down -v                        # 중지 + DB 초기화
```

## 개발 방식

GitHub Issues를 타스크 단위로 사용하는 하이퍼-워터폴 방법론으로 진행한다.
상세 내용은 [`mydocs/manual/hyper_waterfall.md`](mydocs/manual/hyper_waterfall.md)를 참고한다.

| 마일스톤 | 내용 |
|----------|------|
| M010 (v0.1.0) | 기반 구축 및 기술 조사 |
| M050 (v0.5.0) | 백엔드 및 데이터 수집 |
| M100 (v1.0.0) | 프론트엔드, 검증, 릴리즈 |
