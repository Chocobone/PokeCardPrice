package server

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
)

// apiError 는 계약 형식({"error": {"code", "message"}})으로 응답할 오류이다.
// message 는 화면에 그대로 보여 줄 수 있는 한국어 문장이다.
type apiError struct {
	Status  int
	Code    string
	Message string
}

func (e *apiError) Error() string { return e.Code + ": " + e.Message }

func invalidQuery(msg string) *apiError {
	return &apiError{Status: http.StatusBadRequest, Code: "invalid_query", Message: msg}
}

func invalidParam(msg string) *apiError {
	return &apiError{Status: http.StatusBadRequest, Code: "invalid_param", Message: msg}
}

func notFound(msg string) *apiError {
	return &apiError{Status: http.StatusNotFound, Code: "not_found", Message: msg}
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// toAPIError 는 핸들러·프레임워크 오류를 계약 오류로 바꾼다. 알 수 없는 오류는 500 internal 이다.
func toAPIError(err error) *apiError {
	var ae *apiError
	if errors.As(err, &ae) {
		return ae
	}
	status := http.StatusInternalServerError
	var he *echo.HTTPError
	var sc echo.HTTPStatusCoder
	switch {
	case errors.As(err, &he):
		status = he.Code
	case errors.As(err, &sc):
		status = sc.StatusCode()
	}
	switch {
	case status == http.StatusNotFound:
		return notFound("요청한 주소를 찾을 수 없습니다")
	case status == http.StatusMethodNotAllowed:
		return &apiError{Status: status, Code: "method_not_allowed", Message: "허용하지 않는 요청 방식입니다"}
	case status >= 400 && status < 500:
		return &apiError{Status: status, Code: "invalid_param", Message: "요청이 올바르지 않습니다"}
	default:
		return &apiError{Status: http.StatusInternalServerError, Code: "internal", Message: "서버 오류가 발생했습니다. 잠시 후 다시 시도해 주세요"}
	}
}

// errorHandler 는 모든 오류를 계약 형식으로 응답한다. 500 의 원인은 로그에만 남긴다.
func errorHandler(c *echo.Context, err error) {
	// RequestLogger 가 오류 처리기를 먼저 호출하므로, 이미 응답했으면 다시 쓰지 않는다.
	if r, _ := echo.UnwrapResponse(c.Response()); r != nil && r.Committed {
		return
	}
	ae := toAPIError(err)
	if ae.Status >= http.StatusInternalServerError {
		slog.Error("request error", "err", err, "path", c.Request().URL.Path)
	}
	_ = c.JSON(ae.Status, errorBody{Error: errorDetail{Code: ae.Code, Message: ae.Message}})
}
