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

// do 는 요청을 보내고 응답 본문을 v 로 읽는다.
func do(t *testing.T, h http.Handler, method, path string, v any) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	if v != nil {
		if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(v); err != nil {
			t.Fatalf("decode body %q: %v", rec.Body.String(), err)
		}
	}
	return rec
}

// expectError 는 계약 오류 형식과 상태·코드를 확인한다.
func expectError(t *testing.T, h http.Handler, method, path string, status int, code string) {
	t.Helper()
	var body errorBody
	rec := do(t, h, method, path, &body)
	if rec.Code != status || body.Error.Code != code || body.Error.Message == "" {
		t.Fatalf("%s %s = %d %+v, want %d %s", method, path, rec.Code, body, status, code)
	}
}

func TestHealthz(t *testing.T) {
	var body map[string]string
	rec := do(t, New(fakeDB{}, nil), http.MethodGet, "/healthz", &body)

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
	var body map[string]string
	rec := do(t, New(fakeDB{err: errors.New("connection refused")}, nil), http.MethodGet, "/healthz", &body)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if body["status"] != "degraded" || body["db"] != "error" {
		t.Errorf("body = %v, want status=degraded db=error", body)
	}
}

func TestHealthzMethodNotAllowed(t *testing.T) {
	expectError(t, New(fakeDB{}, nil), http.MethodPost, "/healthz", http.StatusMethodNotAllowed, "method_not_allowed")
}

func TestUnknownPath(t *testing.T) {
	expectError(t, New(fakeDB{}, nil), http.MethodGet, "/nope", http.StatusNotFound, "not_found")
}

func TestErrorWrittenOnce(t *testing.T) {
	rec := do(t, New(fakeDB{}, nil), http.MethodGet, "/nope", nil)

	// RequestLogger 와 프레임워크가 오류 처리기를 두 번 호출해도 응답은 한 번만 쓰여야 한다.
	if n := strings.Count(rec.Body.String(), `"error"`); n != 1 {
		t.Fatalf("error body written %d times: %q", n, rec.Body.String())
	}
}

func TestInternalErrorHidesCause(t *testing.T) {
	ae := toAPIError(errors.New("pq: password authentication failed"))
	if ae.Status != http.StatusInternalServerError || ae.Code != "internal" || strings.Contains(ae.Message, "password") {
		t.Fatalf("toAPIError = %+v", ae)
	}
}

func TestSearchParams(t *testing.T) {
	p := searchParams([]string{"피카츄", "ex", "100%_"}, "", 3, 24)
	if p.KeyTerm != "100%_" || p.KeyPattern != `%100\%\_%` {
		t.Fatalf("key = %q pattern = %q", p.KeyTerm, p.KeyPattern)
	}
	if p.Query != "피카츄 ex 100%_" || p.Off != 48 || p.Lim != 24 {
		t.Fatalf("params = %+v", p)
	}
	// 글자 수가 같으면 앞의 낱말 (한글 3자 = 9바이트이지만 글자 수로 비교)
	if p := searchParams([]string{"리자몽", "abcd"}, "", 1, 24); p.KeyTerm != "abcd" {
		t.Fatalf("key = %q", p.KeyTerm)
	}
	if p := searchParams([]string{"리자몽", "abc"}, "", 1, 24); p.KeyTerm != "리자몽" {
		t.Fatalf("key = %q", p.KeyTerm)
	}
}

func TestSearchValidation(t *testing.T) {
	h := New(fakeDB{}, nil) // 검증 오류는 DB 에 닿기 전에 반환된다
	for _, c := range []struct {
		path, code string
	}{
		{"/cards", "invalid_query"},
		{"/cards?q=%20%20", "invalid_query"},
		{"/cards?q=" + strings.Repeat("가", 101), "invalid_query"},
		{"/cards?q=a+b+c+d+e+f+g+h+i+j+k", "invalid_query"},
		{"/cards?q=pikachu&lang=fr", "invalid_param"},
		{"/cards?q=pikachu&page=0", "invalid_param"},
		{"/cards?q=pikachu&page=x", "invalid_param"},
		{"/cards?q=pikachu&size=61", "invalid_param"},
		{"/cards?q=pikachu&page=10001", "invalid_param"},
	} {
		expectError(t, h, http.MethodGet, c.path, http.StatusBadRequest, c.code)
	}
}
