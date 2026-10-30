# Task #1: 저장소 구조 및 개발 환경 구성 — 수행 계획서

> 개정 (2026-10-29): 작업지시자 지시로 개발 환경을 **컨테이너로 관리**한다.
> PC에 Go·Node.js를 설치하지 않고, `Docker/` 폴더의 docker-compose로 DB·백엔드·프론트엔드 개발 컨테이너를 관리한다.

## 수행 목표

Go 백엔드, Vite + Vue 프론트엔드, PostgreSQL을 **docker-compose 하나로 띄워 개발할 수 있는** 빈 프로젝트
골격을 만든다. 이후 타스크(#4 DB 스키마, #8 조회 API, #12 프론트엔드)가 바로 코드를 얹을 수 있게 하고,
빌드·테스트·실행 명령을 CLAUDE.md에 고정한다.

## 범위

1. **컨테이너 개발 환경** (`Docker/`): PostgreSQL, Go, Node.js 이미지를 pull하고 docker-compose로 묶는다.
   소스는 바인드 마운트로 연결해, 호스트에서 편집하고 컨테이너에서 빌드·실행한다.
2. **백엔드 골격** (`src/backend/`): 컨테이너 안에서 Go 모듈 초기화, `/healthz` 응답만 하는 최소 HTTP 서버, 단위 테스트 1개
3. **프론트엔드 골격** (`src/frontend/`): 컨테이너 안에서 Vite + Vue 프로젝트 생성, 백엔드 `/healthz` 호출 확인(프록시 설정)
4. **저장소 정비**: `.gitignore`, `.gitattributes`(셸 스크립트 LF 고정), README를 Card-Market 소개와 실행 방법으로 교체
5. **CLAUDE.md**: `빌드 및 실행` 절에 compose 기반 빌드·테스트·실행 명령 기입

### 범위 밖

- Go 웹 프레임워크(Gin/Echo), DB 접근 라이브러리, 마이그레이션 도구 선택 → **#3에서 결정**.
  본 타스크의 백엔드는 표준 라이브러리 `net/http`만 사용한다.
- 테이블 생성, 데이터 수집, 화면 구현 → #4 이후
- 라우터·Pinia·차트 라이브러리 도입 → #12
- 운영 배포용 이미지(멀티 스테이지 빌드 등) → #16

## 핵심 설계 원칙

- **호스트 무설치**: 호스트에는 Docker만 있으면 된다. `go`, `npm` 명령은 모두 `docker compose exec`/`run`으로 실행한다.
- **모노레포**: `src/backend/`, `src/frontend/`, `Docker/`를 한 저장소에 두고, 컨테이너 정의는 `Docker/`에 모은다.
- **이미지 버전 고정**: 메이저(·마이너) 버전 태그로 고정해 재현성을 확보한다.
- **설정 분리**: DB 비밀번호 등은 `Docker/.env`로 분리하고 커밋하지 않는다(`.env.example`만 커밋). 공개 저장소이므로 특히 주의한다.
- **재현성**: 새 환경에서 `docker compose up` 한 번으로 DB → 백엔드 → 프론트엔드가 뜬다.

## 기존 구현 현황

신규 구현. 현재 저장소에는 문서만 있다.

- `CLAUDE.md`: 프로젝트 지시서 (빌드 명령 미기입)
- `README.md`: 하이퍼-워터폴 템플릿 설명 (프로젝트 소개로 교체 대상)
- `docs/`, `mydocs/`: 계획서와 방법론 문서

### 개발 환경 확인 결과 (2026-10-29)

| 도구 | 상태 | 처리 |
|------|------|------|
| Docker 29.8.2 / Compose v5.5.1 | 설치됨 | 사용 |
| Go | 미설치 | `golang:1.27-alpine3.24` 컨테이너로 대체 |
| Node.js / npm | 미설치 | `node:26-alpine3.24` (LTS) 컨테이너로 대체 |
| PostgreSQL / psql | 미설치 | `postgres:18-alpine3.24` 컨테이너로 대체 |

## 구현 계획서

`mydocs/plans/task_m010_1_impl.md` 참조

## 브랜치

`local/task1`

## 예상 단계: 4단계

1. 컨테이너 개발 환경 구성 (`Docker/`, 이미지 pull, compose 기동 확인)
2. 백엔드 골격 (컨테이너에서 Go 모듈, `/healthz`, 단위 테스트)
3. 프론트엔드 골격 (컨테이너에서 Vite + Vue, 백엔드 프록시 연동 확인)
4. 문서 정비 (README 교체, CLAUDE.md 빌드·테스트·실행 명령 기입)
