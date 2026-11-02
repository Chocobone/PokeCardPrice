# 기술 조사: 백엔드 기술 스택

> **작성일**: 2026-11-02
> **관련 타스크**: [#3](https://github.com/Chocobone/PokeCardPrice/issues/3) 백엔드 기술 선정
> **상태**: 1단계(웹 프레임워크) 작성. DB 접근·마이그레이션은 2단계, 스케줄러와 선정 결과는 3단계에서 추가한다.

---

## 배경

#1에서 만든 백엔드 골격은 표준 라이브러리 `net/http`만 사용한다. #4 이후 타스크가 쓸 웹 프레임워크, DB 접근 방식, 마이그레이션 도구, 스케줄러 방식을 실제 예제로 비교해 정한다.

- 확인일은 별도 표시가 없으면 모두 2026-11-02이다.
- 예제는 `golang:1.27-alpine3.24`(Go 1.27.3) 컨테이너에서 빌드·실행했다. 예제 코드는 세션 임시 폴더에 두었고 저장소에는 넣지 않았다.

## 조사 내용

### 1. 웹 프레임워크

#### 후보와 버전

| 후보 | 버전 | 릴리스 | 비고 |
|------|------|--------|------|
| 표준 라이브러리 `net/http` | Go 1.27.3 | - | Go 1.22부터 `ServeMux`가 메서드·경로 변수 패턴(`GET /cards/{id}`)을 지원 |
| Gin (`github.com/gin-gonic/gin`) | v1.12.0 | 2026-02-28 | 직전 릴리스 v1.11.0은 2025-09-20 |
| Echo (`github.com/labstack/echo/v5`) | v5.4.4 | 2026-10-30 | v5.0.0은 2026-01-18. v4도 v4.16.3(2026-10-30)으로 함께 관리됨 |

Echo는 최신 메이저인 v5를 대상으로 했다. 릴리스 날짜는 Go 모듈 프록시(`go list -m -json`)로 확인했다.

#### 예제 기능

세 후보로 같은 기능을 구현하고, **같은 테스트 파일**(테스트 8개)을 그대로 적용했다. 테스트는 `newApp() http.Handler`만 호출한다.

| 기능 | 테스트 내용 |
|------|-------------|
| `GET /healthz` | 200, `{"status":"ok"}`, JSON Content-Type |
| `GET /cards/{id}` | 경로 변수, 없는 카드면 404와 `{"error":...}` |
| `GET /cards?q=&page=` | `page` 기본값 1, 0·문자는 400, `q` 101자 이상은 400 |
| `POST /collections` | JSON 본문, 필수 값 누락·수량 0·잘못된 JSON은 400 |
| `/me/*` 그룹 | 인증 미들웨어. 토큰이 없으면 401, 있으면 200 |
| 패닉 복구 | 핸들러 패닉 시 500과 JSON 오류 |
| CORS | `http://localhost:5173` 사전 요청 허용, 다른 출처는 허용 헤더 없음 |
| 404·405 | 없는 경로 404, 허용하지 않는 메서드 405, 둘 다 JSON 오류 |

정상 종료는 세 후보 모두 `http.Server`에 핸들러로 넣는 같은 `main` 구조를 사용했다. 실행 중 SIGTERM을 보내 종료 로그와 종료 코드 0을 확인했다.

#### 측정 결과

| 항목 | `net/http` | Gin v1.12.0 | Echo v5.4.4 |
|------|-----------|-------------|-------------|
| `go vet`, `gofmt` | 통과 | 통과 | 통과 |
| 공통 테스트 8개 | 통과 | 통과 | 통과 (오류 처리기 수정 후, 아래 참고) |
| SIGTERM 정상 종료 | 확인 | 확인 | 확인 |
| 모듈 그래프 (`go list -m all`, 자신 포함) | 1 | 56 | 9 |
| 바이너리에 포함된 의존 모듈 (`go version -m`) | 0 | 19 | 2 |
| 바이너리 크기 (`-ldflags="-s -w"`) | 6.4 MiB | 20.6 MiB | 7.0 MiB |
| 예제 코드 (`main.go`, 빈 줄 제외) | 173줄 | 107줄 | 103줄 |

- 측정은 각 1회이다. 바이너리 크기는 Go 버전과 빌드 옵션에 따라 달라진다.
- 성능 벤치마크는 구현 계획서에 따라 하지 않았다.

#### 기능별 비교

| 항목 | `net/http` | Gin | Echo v5 |
|------|-----------|-----|---------|
| 경로 변수 | `r.PathValue("id")` | `c.Param("id")` | `c.Param("id")` |
| 쿼리·본문 바인딩 | 직접 파싱 | 구조체 태그로 바인딩 (`form`, `json`, 기본값 지정 가능) | `c.Bind`로 바인딩 |
| 검증 | 직접 작성 | 내장 (`binding:"required,min=1"`, go-playground/validator) | 내장 검증기 없음. `Validator` 인터페이스에 외부 검증기를 연결하거나 직접 작성 |
| 핸들러 오류 처리 | 핸들러마다 응답 작성 | 핸들러마다 응답 작성 | 핸들러가 `error`를 반환하고 중앙 오류 처리기가 응답 |
| 로그·복구·CORS 미들웨어 | 직접 작성 (약 50줄) | 복구 내장, CORS는 별도 모듈(`gin-contrib/cors`) 또는 직접 작성, 로그는 직접 작성(`slog` 사용 시) | 셋 다 내장 (`RequestLogger`, `Recover`, `CORS`) |
| 404·405 JSON 응답 | 기본 응답이 텍스트. 응답 래퍼를 직접 작성해야 함 (약 30줄) | `NoRoute`, `NoMethod`로 지정. 405는 `HandleMethodNotAllowed = true` 필요 | 중앙 오류 처리기 하나로 처리 |
| 라우트 그룹 | 하위 `ServeMux`를 만들어 미들웨어로 감쌈 | `r.Group("/me", auth())` | `e.Group("/me", auth)` |
| 테스트 | `httptest`로 바로 가능 | `httptest`로 바로 가능 | `httptest`로 바로 가능 |
| 로그 형식 | `slog` 직접 사용 | 기본 로거는 텍스트. 예제는 `slog` 미들웨어를 직접 작성 | `RequestLogger`가 `slog` JSON 로그 출력 |

#### 확인한 주의점

- **Echo v5: 사용자 정의 오류 처리기는 응답이 이미 나갔는지 확인해야 한다.** 처음 구현에서 오류 응답이 두 번 쓰였다(`{"error":"Not Found"}`가 본문에 2회). `RequestLogger`가 오류 처리기를 먼저 호출하고, 프레임워크가 다시 호출하기 때문이다. 기본 처리기처럼 `echo.UnwrapResponse(c.Response())`의 `Committed`를 확인해 해결했다.
- **Gin: 검증 오류 메시지가 내부 구조체 이름을 드러낸다.** 예: `Key: 'collectionReq.CardID' Error:Field validation for 'CardID' failed on the 'required' tag`. 사용자에게 보여 주려면 메시지 변환이 필요하다.
- **Gin: `required`는 0을 누락으로 본다.** 수량 0을 보내면 `min` 대신 `required` 위반으로 보고된다. 0이 의미 있는 값인 필드에는 `required`를 쓰지 않거나 포인터 타입을 써야 한다.
- **Gin: 의존성이 많다.** 모듈 그래프에 QUIC, MongoDB 드라이버, protobuf 등이 포함된다(바이너리에는 19개만 들어감). 바이너리가 다른 후보의 약 3배이다.
- **`net/http`: 기본 404·405 응답이 텍스트다.** JSON으로 바꾸려면 응답 래퍼가 필요하고, 래퍼는 상태 코드와 Content-Type으로 프레임워크 응답인지 판단해야 해서 깨지기 쉽다.
- **Echo v5는 v5.0.0 이후 약 10개월 된 메이저 버전이다.** 검색되는 예제 상당수가 v4 기준(`echo.Context` 인터페이스)이라 v5(`*echo.Context` 구조체)와 다르다.

### 1단계 잠정 판단

선정은 3단계에서 확정한다. 1단계 결과만으로 본 판단은 다음과 같다.

1. **Echo v5가 가장 유력하다.** 필요한 미들웨어가 내장되어 있고, 핸들러가 오류를 반환하는 구조라 #8~#11의 오류 응답을 한 곳에서 맞출 수 있다. 의존성(바이너리 2개 모듈)과 바이너리 크기도 표준 라이브러리와 비슷하다.
2. **`net/http`는 의존성이 없다는 장점이 크지만, 공통 기능을 직접 유지해야 한다.** 미들웨어와 404·405 JSON 처리에 약 80줄이 들었고, 검증도 모두 직접 작성해야 한다.
3. **Gin은 검증 내장이 장점이지만, 의존성과 바이너리 크기가 가장 크다.** 검증 메시지 변환도 추가로 필요해, 이 프로젝트 규모에서는 이점이 작다.
4. Echo를 고르면 검증은 직접 작성하거나 go-playground/validator를 `Validator`에 연결해야 한다. 이 선택은 #8 구현 계획에서 정해도 된다.

## 참고 자료

- Go `net/http` ServeMux 패턴: https://pkg.go.dev/net/http#ServeMux
- Gin: https://github.com/gin-gonic/gin , https://pkg.go.dev/github.com/gin-gonic/gin@v1.12.0
- Echo v5: https://github.com/labstack/echo , https://pkg.go.dev/github.com/labstack/echo/v5@v5.4.4
