package fx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/httpx"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(WithBaseURL(srv.URL), WithHTTP(httpx.New("test", httpx.WithRetry(1, time.Millisecond))))
}

func TestRates(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/rates" {
			t.Errorf("path = %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("base") != "USD" || q.Get("quotes") != "KRW" || q.Get("date") != "2026-11-15" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		// 실제 응답 (2026-11-19 확인 형식)
		_, _ = w.Write([]byte(`[{"date":"2026-11-15","base":"USD","quote":"KRW","rate":1335.88}]`))
	})
	got, err := c.Rates(context.Background(), "USD", []string{"KRW"}, time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Rate.String() != "1335.88" || got[0].Date.Format(time.DateOnly) != "2026-11-15" ||
		got[0].Base != "USD" || got[0].Quote != "KRW" {
		t.Fatalf("got %+v", got)
	}
}

func TestRatesLatestOmitsDateAndKeepsResponseDate(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("date") {
			t.Errorf("date 가 있으면 안 됨: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`[{"date":"2026-11-18","base":"EUR","quote":"KRW","rate":1478.66}]`))
	})
	got, err := c.Rates(context.Background(), "EUR", []string{"KRW"}, time.Time{})
	if err != nil || len(got) != 1 || got[0].Date.Format(time.DateOnly) != "2026-11-18" {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestRatesEmptyAndErrors(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("base") {
		case "USD":
			_, _ = w.Write([]byte(`[]`))
		case "XXX":
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"status":422,"message":"invalid currency: XXX"}`))
		case "EUR":
			_, _ = w.Write([]byte(`[{"date":"2026-11-18","base":"EUR","quote":"KRW","rate":0}]`))
		}
	})
	ctx := context.Background()
	if got, err := c.Rates(ctx, "USD", []string{"KRW"}, time.Time{}); err != nil || len(got) != 0 {
		t.Fatalf("empty: %+v %v", got, err)
	}
	if _, err := c.Rates(ctx, "XXX", []string{"KRW"}, time.Time{}); err == nil || !strings.Contains(err.Error(), "invalid currency") {
		t.Fatalf("422: %v", err)
	}
	if _, err := c.Rates(ctx, "EUR", []string{"KRW"}, time.Time{}); err == nil {
		t.Fatal("0 환율은 오류여야 함")
	}
}
