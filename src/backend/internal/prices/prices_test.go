package prices

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbtest"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/fx"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/httpx"
)

// testPool 은 마이그레이션을 적용한 패키지 전용 테스트 DB 이다.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := dbtest.Pool(t, "prices")
	if err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

// fakeSource 는 Runner 테스트용 작업이다.
type fakeSource struct {
	code    string
	err     error
	panics  bool
	started chan struct{}
	release chan struct{}
	opt     Options
}

func (f *fakeSource) Code() string { return f.code }

func (f *fakeSource) Collect(ctx context.Context, opt Options) (Result, error) {
	f.opt = opt
	if f.started != nil {
		close(f.started)
	}
	if f.release != nil {
		<-f.release
	}
	if f.panics {
		panic("boom")
	}
	return Result{Saved: 1}, f.err
}

func TestRunnerRunsSourcesConcurrentlyAndIsolatesFailures(t *testing.T) {
	// a 는 b 가 시작해야 끝난다 → 동시에 실행되지 않으면 교착된다.
	bStarted := make(chan struct{})
	a := &fakeSource{code: "a", release: bStarted}
	b := &fakeSource{code: "b", started: bStarted, err: errors.New("b failed")}
	c := &fakeSource{code: "c", panics: true}

	done := make(chan struct{})
	var results []Result
	var err error
	go func() {
		results, err = NewRunner(a, b, c).Run(context.Background(), Options{})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("작업이 동시에 실행되지 않음")
	}

	if len(results) != 3 || results[0].Source != "a" || results[1].Source != "b" || results[2].Source != "c" {
		t.Fatalf("results = %+v", results)
	}
	if results[0].Saved != 1 || results[1].Saved != 1 {
		t.Fatalf("실패한 작업의 결과도 남아야 함: %+v", results)
	}
	if err == nil || !strings.Contains(err.Error(), "b: b failed") || !strings.Contains(err.Error(), "c: panic: boom") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "a:") {
		t.Fatalf("성공한 작업이 오류에 들어감: %v", err)
	}
}

func TestRunnerFiltersSources(t *testing.T) {
	a, b := &fakeSource{code: "a"}, &fakeSource{code: "b"}
	opt := Options{Sources: []string{"b"}, Languages: []string{"ja"}}
	results, err := NewRunner(a, b).Run(context.Background(), opt)
	if err != nil || len(results) != 1 || results[0].Source != "b" {
		t.Fatalf("results=%+v err=%v", results, err)
	}
	if a.opt.Languages != nil || b.opt.Languages[0] != "ja" {
		t.Fatalf("옵션 전달: a=%+v b=%+v", a.opt, b.opt)
	}
}

// fakeFrankfurter 는 base 별 응답을 바꿀 수 있는 Frankfurter 이다.
type fakeFrankfurter struct {
	mu    sync.Mutex
	body  map[string]string // base → JSON
	calls atomic.Int32
}

func (f *fakeFrankfurter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.calls.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	body, ok := f.body[r.URL.Query().Get("base")]
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte(body))
}

func newFX(t *testing.T, pool *pgxpool.Pool, f *fakeFrankfurter) *FX {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return NewFX(pool, fx.New(fx.WithBaseURL(srv.URL), fx.WithHTTP(httpx.New("test", httpx.WithRetry(1, time.Millisecond)))))
}

type fxRow struct {
	Base, Quote, Date, Rate string
}

func fxRows(t *testing.T, pool *pgxpool.Pool) []fxRow {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT base_currency, quote_currency, rate_date::text, rate::text FROM fx_rates ORDER BY 1, 2, 3`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []fxRow
	for rows.Next() {
		var r fxRow
		if err := rows.Scan(&r.Base, &r.Quote, &r.Date, &r.Rate); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

func TestFXCollectIsIdempotent(t *testing.T) {
	pool := testPool(t)
	f := &fakeFrankfurter{body: map[string]string{
		"USD": `[{"date":"2026-11-18","base":"USD","quote":"KRW","rate":1333.6}]`,
		"EUR": `[{"date":"2026-11-18","base":"EUR","quote":"KRW","rate":1478.66}]`,
	}}
	src := newFX(t, pool, f)
	ctx := context.Background()

	res, err := src.Collect(ctx, Options{})
	if err != nil || res.Fetched != 2 || res.Saved != 2 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	// 같은 날 다시 실행하면 값만 갱신되고 행은 늘지 않는다.
	f.mu.Lock()
	f.body["USD"] = `[{"date":"2026-11-18","base":"USD","quote":"KRW","rate":1334.1}]`
	f.mu.Unlock()
	if _, err := src.Collect(ctx, Options{}); err != nil {
		t.Fatal(err)
	}
	got := fxRows(t, pool)
	want := []fxRow{{"EUR", "KRW", "2026-11-18", "1478.66000000"}, {"USD", "KRW", "2026-11-18", "1334.10000000"}}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("rows = %+v", got)
	}

	// 다음 날 값은 새 행으로 쌓인다.
	f.mu.Lock()
	f.body["USD"] = `[{"date":"2026-11-19","base":"USD","quote":"KRW","rate":1333.27}]`
	f.mu.Unlock()
	if _, err := src.Collect(ctx, Options{}); err != nil {
		t.Fatal(err)
	}
	if n := len(fxRows(t, pool)); n != 3 {
		t.Fatalf("rows = %d, want 3", n)
	}
}

func TestFXCollectPartialFailure(t *testing.T) {
	pool := testPool(t)
	// USD 는 서버 오류, EUR 는 빈 응답(제공처 없음)
	f := &fakeFrankfurter{body: map[string]string{"EUR": `[]`}}
	res, err := newFX(t, pool, f).Collect(context.Background(), Options{})
	if err == nil {
		t.Fatal("USD 실패가 오류로 나와야 함")
	}
	if len(res.Failed) != 1 || res.Failed[0] != "USD/KRW" || res.Skipped != 1 || res.Saved != 0 {
		t.Fatalf("res = %+v", res)
	}
	if n := len(fxRows(t, pool)); n != 0 {
		t.Fatalf("rows = %d", n)
	}
}
