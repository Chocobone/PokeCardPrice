// Package server 는 HTTP 라우팅과 핸들러를 구성한다.
// 응답 형식은 mydocs/tech/api_contract.md 를 따른다.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbq"
)

// Pinger 는 /healthz 가 확인하는 DB 상태이다. *pgxpool.Pool 이 만족한다.
type Pinger interface {
	Ping(ctx context.Context) error
}

// New 는 API 라우터를 반환한다. q 는 카드 API 가 쓰는 쿼리이다 (읽기 전용).
func New(db Pinger, q *dbq.Queries) http.Handler {
	return newRouter(db, q, time.Now)
}

// newRouter 는 New 와 같고, 시세 기간 계산에 쓸 현재 시각을 바꿀 수 있다 (테스트용).
func newRouter(db Pinger, q *dbq.Queries, now func() time.Time) http.Handler {
	e := echo.New()
	e.HTTPErrorHandler = errorHandler
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.GET("/healthz", healthz(db))

	h := &cardHandlers{q: q, now: now}
	e.GET("/cards", h.search)
	e.GET("/cards/:id", h.detail)
	e.GET("/cards/:id/prices", h.prices)
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
