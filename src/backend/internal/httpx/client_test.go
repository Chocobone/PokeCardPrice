package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fastClient(opts ...Option) *Client {
	return New("test", append([]Option{WithRetry(2, time.Millisecond)}, opts...)...)
}

func TestGetJSONRetriesServerErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != DefaultUserAgent {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{"a":1}`))
	}))
	defer srv.Close()

	var out struct{ A int }
	if err := fastClient().GetJSON(context.Background(), srv.URL, &out); err != nil {
		t.Fatal(err)
	}
	if out.A != 1 || calls.Load() != 3 {
		t.Fatalf("out=%+v calls=%d", out, calls.Load())
	}
}

func TestGetJSONGivesUpAfterMaxRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	err := fastClient().GetJSON(context.Background(), srv.URL, &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "2회 재시도 후 실패") {
		t.Fatalf("err = %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
}

func TestClientErrorsAreNotRetried(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"status":422,"message":"invalid currency: XXX"}`))
	}))
	defer srv.Close()

	c := fastClient()
	if err := c.GetJSON(context.Background(), srv.URL+"/missing", &struct{}{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("404 err = %v", err)
	}
	err := c.GetJSON(context.Background(), srv.URL+"/bad", &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "HTTP 422") || !strings.Contains(err.Error(), "invalid currency") {
		t.Fatalf("422 err = %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2 (재시도 없음)", calls.Load())
	}
}

func TestMalformedJSONIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"a":"x"}`))
	}))
	defer srv.Close()

	var out struct{ A int }
	if err := fastClient().GetJSON(context.Background(), srv.URL, &out); err == nil {
		t.Fatal("want error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestGetTextTrims(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("2026-11-18T20:05:56+0000\n"))
	}))
	defer srv.Close()

	s, err := fastClient().GetText(context.Background(), srv.URL)
	if err != nil || s != "2026-11-18T20:05:56+0000" {
		t.Fatalf("s=%q err=%v", s, err)
	}
}

func TestMinIntervalIsShared(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := fastClient(WithMinInterval(30 * time.Millisecond))
	start := time.Now()
	done := make(chan error, 4)
	for range 4 {
		go func() { done <- c.GetJSON(context.Background(), srv.URL, &struct{}{}) }()
	}
	for range 4 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	// 4건이면 첫 요청 뒤 간격 3번 = 90ms 이상
	if d := time.Since(start); d < 90*time.Millisecond {
		t.Fatalf("elapsed %v, want >= 90ms", d)
	}
}

func TestContextCancelStopsRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := fastClient().GetJSON(ctx, srv.URL, &struct{}{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
}

func TestParseRetryAfter(t *testing.T) {
	for in, want := range map[string]time.Duration{"5": 5 * time.Second, "": 0, "0": 0, "x": 0, "Wed, 21 Oct 2015 07:28:00 GMT": 0} {
		if got := parseRetryAfter(in); got != want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", in, got, want)
		}
	}
}
