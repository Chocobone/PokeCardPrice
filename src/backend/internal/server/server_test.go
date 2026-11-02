package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func serve(t *testing.T, db Pinger, method, path string) (*httptest.ResponseRecorder, map[string]string) {
	t.Helper()
	rec := httptest.NewRecorder()
	New(db).ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return rec, body
}

func TestHealthz(t *testing.T) {
	rec, body := serve(t, fakeDB{}, http.MethodGet, "/healthz")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q", ct)
	}
	if body["status"] != "ok" || body["db"] != "ok" {
		t.Errorf("body = %v, want status=ok db=ok", body)
	}
}

func TestHealthzDBDown(t *testing.T) {
	rec, body := serve(t, fakeDB{err: errors.New("connection refused")}, http.MethodGet, "/healthz")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if body["status"] != "degraded" || body["db"] != "error" {
		t.Errorf("body = %v, want status=degraded db=error", body)
	}
}

func TestHealthzMethodNotAllowed(t *testing.T) {
	rec, body := serve(t, fakeDB{}, http.MethodPost, "/healthz")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if body["error"] == "" {
		t.Errorf("body = %v, want error field", body)
	}
}

func TestUnknownPath(t *testing.T) {
	rec, body := serve(t, fakeDB{}, http.MethodGet, "/nope")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if body["error"] == "" {
		t.Errorf("body = %v, want error field", body)
	}
}

func TestErrorWrittenOnce(t *testing.T) {
	rec := httptest.NewRecorder()
	New(fakeDB{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))

	// RequestLogger 와 프레임워크가 오류 처리기를 두 번 호출해도 응답은 한 번만 쓰여야 한다.
	if n := strings.Count(rec.Body.String(), `"error"`); n != 1 {
		t.Fatalf("error body written %d times: %q", n, rec.Body.String())
	}
}
