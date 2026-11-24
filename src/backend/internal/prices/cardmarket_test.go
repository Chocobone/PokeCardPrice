package prices

import (
	"context"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/tcgdex"
)

func f64(v float64) *float64 { return &v }

func TestCardmarketRows(t *testing.T) {
	vs := func(codes ...string) []cardVariant {
		out := make([]cardVariant, len(codes))
		for i, c := range codes {
			out[i] = cardVariant{ID: int64(i + 1), Variant: c}
		}
		return out
	}
	full := func() *tcgdex.CardmarketPricing {
		return &tcgdex.CardmarketPricing{
			Updated: "2026-11-23T03:12:41.577Z", Unit: "EUR",
			Avg: f64(0.17), Low: f64(0.02), Trend: f64(0.16), Avg1: f64(0.15), Avg7: f64(0.15), Avg30: f64(0.2),
			AvgHolo: f64(0.36), LowHolo: f64(0.05), TrendHolo: f64(0.37), Avg1Holo: f64(0.4), Avg7Holo: f64(0.35), Avg30Holo: f64(0.4),
		}
	}
	none := func(int64) bool { return false }

	type want struct {
		rows  []string // "변형:대표값:기준"
		skips []string
	}
	cases := []struct {
		name     string
		variants []cardVariant
		cm       *tcgdex.CardmarketPricing
		linked   func(int64) bool
		want     want
	}{
		{"일반+리버스", vs("normal", "reverse_holofoil"), full(), none,
			want{[]string{"normal:0.16:trend", "reverse_holofoil:0.37:trend"}, nil}},
		{"홀로+리버스", vs("holofoil", "reverse_holofoil"), full(), none,
			want{[]string{"holofoil:0.16:trend", "reverse_holofoil:0.37:trend"}, nil}},
		{"일반만, -holo 값 있음 → 버림", vs("normal"), full(), none,
			want{[]string{"normal:0.16:trend"}, []string{cmSkipNoVariant}}},
		{"초판 세트 → 무제한판, 초판에는 넣지 않음", vs("first_edition", "unlimited"), full(), none,
			want{[]string{"unlimited:0.16:trend"}, []string{cmSkipNoVariant}}},
		{"초판 홀로 세트 → 무제한 홀로", vs("first_edition_holofoil", "unlimited_holofoil"), full(), none,
			want{[]string{"unlimited_holofoil:0.16:trend"}, []string{cmSkipNoVariant}}},
		{"초판만 있는 카드", vs("first_edition"), full(), none,
			want{nil, []string{cmSkipNoVariant, cmSkipNoVariant}}},
		{"Q9: TCGplayer 연결이 홀로에만", vs("normal", "holofoil"), full(), func(id int64) bool { return id == 2 },
			want{[]string{"holofoil:0.16:trend"}, []string{cmSkipNoVariant}}},
		{"Q9 아님: 일반에도 연결", vs("normal", "holofoil"), full(), func(int64) bool { return true },
			want{[]string{"normal:0.16:trend"}, []string{cmSkipNoVariant}}},
		{"가격 없음", vs("normal"), nil, none, want{nil, []string{cmSkipNoPricing}}},
		{"통화 USD", vs("normal"), &tcgdex.CardmarketPricing{Updated: "2026-11-23T03:12:41Z", Unit: "USD", Trend: f64(1)}, none,
			want{nil, []string{cmSkipBadUnit}}},
		{"updated 없음", vs("normal"), &tcgdex.CardmarketPricing{Unit: "EUR", Trend: f64(1)}, none,
			want{nil, []string{cmSkipBadDate}}},
		{"trend 0 → avg, -holo 0 → 후보 아님", vs("normal", "reverse_holofoil"),
			&tcgdex.CardmarketPricing{Updated: "2026-11-23T03:12:41Z", Unit: "EUR", Trend: f64(0), Avg: f64(1.5), TrendHolo: f64(0)}, none,
			want{[]string{"normal:1.5:avg"}, nil}},
		{"기본 값 없음", vs("normal", "reverse_holofoil"),
			&tcgdex.CardmarketPricing{Updated: "2026-11-23T03:12:41Z", Unit: "EUR", AvgHolo: f64(2)}, none,
			want{[]string{"reverse_holofoil:2:avg"}, []string{cmSkipNoValue}}},
	}
	for _, c := range cases {
		rows, skips := cardmarketRows(targetCard{Language: "en", TcgdexID: "x", Variants: c.variants}, c.cm, 2, c.linked)
		var got []string
		for _, r := range rows {
			code := c.variants[slices.IndexFunc(c.variants, func(v cardVariant) bool { return v.ID == r.CardVariantID })].Variant
			got = append(got, code+":"+r.Price.String()+":"+r.PriceBasis)
			if r.SourceID != 2 || r.PriceDate.Format(time.DateOnly) != "2026-11-23" || r.SourceUpdatedAt == nil {
				t.Errorf("%s: row = %+v", c.name, r)
			}
		}
		if !slices.Equal(got, c.want.rows) || !slices.Equal(skips, c.want.skips) {
			t.Errorf("%s: rows=%v skips=%v, want %v %v", c.name, got, skips, c.want.rows, c.want.skips)
		}
	}

	// 0 인 보조 필드는 NULL, 나머지는 그대로
	cm := full()
	cm.Avg1 = f64(0)
	rows, _ := cardmarketRows(targetCard{Variants: vs("normal")}, cm, 2, none)
	r := rows[0]
	if r.Avg1 != nil || r.Low.String() != "0.02" || r.Avg.String() != "0.17" || r.Avg30.String() != "0.2" || r.Market != nil {
		t.Fatalf("fields = %+v", r)
	}
}

func TestCardmarketCollectIsIdempotent(t *testing.T) {
	pool := testPool(t)
	seedCatalog(t, pool)
	ft := &fakeTCGdex{cards: map[string]string{
		"/en/cards/sv03.5-001": `{"id":"sv03.5-001","pricing":{"cardmarket":{"updated":"2026-11-23T03:12:41.577Z","unit":"EUR","avg":0.17,"trend":0.16,"avg-holo":0.36,"trend-holo":0.37},"tcgplayer":null}}`,
		"/en/cards/sv03.5-002": `{"id":"sv03.5-002","pricing":{"cardmarket":{"updated":"2026-11-23T03:12:41.577Z","unit":"EUR","avg":0.2,"trend":0}}}`,
		"/en/cards/sv03.5-205": `{"id":"sv03.5-205","pricing":{"cardmarket":{"updated":"2026-11-22T03:00:00Z","unit":"EUR","avg":20,"trend":21.5}}}`,
		"/en/cards/A1-001":     `{"id":"A1-001","pricing":null}`,
	}}
	srv := httptest.NewServer(ft)
	defer srv.Close()
	src := NewTCGdex(pool, tcgdex.New(tcgdex.WithBaseURL(srv.URL), tcgdex.WithMinInterval(0), tcgdex.WithRetry(0, 0)))
	opt := Options{Languages: []string{"en"}}

	for i := range 2 {
		res, err := src.Collect(context.Background(), opt)
		// /en/series/tcgp 가 404 이면 TCG Pocket 을 빼지 않고 A1-001 도 받는다
		if err != nil || res.Fetched != 4 || res.Saved != 4 || res.Skipped != 1 {
			t.Fatalf("run %d: res=%+v err=%v", i+1, res, err)
		}
		got := priceRows(t, pool, CardmarketCode)
		want := map[string]priceRow{
			"sv03.5-001/normal":           {"sv03.5-001/normal", "2026-11-23", "0.16", "trend"},
			"sv03.5-001/reverse_holofoil": {"sv03.5-001/reverse_holofoil", "2026-11-23", "0.37", "trend"},
			"sv03.5-002/normal":           {"sv03.5-002/normal", "2026-11-23", "0.20", "avg"},
			"sv03.5-205/holofoil":         {"sv03.5-205/holofoil", "2026-11-22", "21.50", "trend"},
		}
		if len(got) != len(want) {
			t.Fatalf("run %d: rows = %+v", i+1, got)
		}
		for k, w := range want {
			if got[k] != w {
				t.Errorf("run %d: %s = %+v, want %+v", i+1, k, got[k], w)
			}
		}
	}
}
