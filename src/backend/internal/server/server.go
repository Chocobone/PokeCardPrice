// Package server 는 HTTP 라우팅과 핸들러를 구성한다.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// Pinger 는 /healthz 가 확인하는 DB 상태이다. *pgxpool.Pool 이 만족한다.
type Pinger interface {
	Ping(ctx context.Context) error
}

// New 는 API 라우터를 반환한다.
func New(db Pinger) http.Handler {
	e := echo.New()
	e.HTTPErrorHandler = errorHandler
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.GET("/healthz", healthz(db))
	return e
}

// healthz 는 서버와 DB 상태를 응답한다. DB 확인에 실패하면 503 을 반환한다.
func healthz(db Pinger) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			slog.Warn("healthz: db ping failed", "err", err)
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "degraded", "db": "error"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "db": "ok"})
	}
}

// errorHandler 는 모든 오류를 {"error": "..."} 형식으로 응답한다.
func errorHandler(c *echo.Context, err error) {
	// RequestLogger 가 오류 처리기를 먼저 호출하므로, 이미 응답했으면 다시 쓰지 않는다.
	if r, _ := echo.UnwrapResponse(c.Response()); r != nil && r.Committed {
		return
	}
	code, msg := http.StatusInternalServerError, "internal error"
	var he *echo.HTTPError
	var sc echo.HTTPStatusCoder
	switch {
	case errors.As(err, &he):
		code, msg = he.Code, he.Message
		if msg == "" {
			msg = http.StatusText(code)
		}
	case errors.As(err, &sc):
		code, msg = sc.StatusCode(), http.StatusText(sc.StatusCode())
	}
	if code >= http.StatusInternalServerError {
		slog.Error("request error", "err", err, "path", c.Request().URL.Path)
	}
	_ = c.JSON(code, map[string]string{"error": msg})
}
