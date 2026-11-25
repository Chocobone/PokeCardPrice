package api

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func decPtr(s string) *decimal.Decimal {
	d := dec(s)
	return &d
}

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestToKRW(t *testing.T) {
	cases := []struct {
		name     string
		amount   string
		currency string
		rate     *decimal.Decimal
		want     *string
	}{
		{"반올림 0.5 올림", "1.50", "USD", decPtr("1001"), strPtr("1502")},  // 1501.5
		{"반올림 내림", "12.34", "USD", decPtr("1336.65"), strPtr("16494")}, // 16494.261
		{"EUR", "3.58", "EUR", decPtr("1474.12"), strPtr("5277")},      // 5277.3496
		{"원화는 그대로", "15000.00", "KRW", nil, strPtr("15000")},
		{"환율 없음", "1.00", "USD", nil, nil},
		{"0원", "0.00", "USD", decPtr("1336.65"), strPtr("0")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ToKRW(dec(c.amount), c.currency, c.rate)
			if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
				t.Fatalf("ToKRW = %v, want %v", deref(got), deref(c.want))
			}
		})
	}
}

func TestNewPrice(t *testing.T) {
	updated := time.Date(2026, 11, 23, 20, 5, 58, 0, time.UTC)
	p := NewPrice(LatestPrice{
		Source: "tcgcsv", Currency: "USD", Date: day("2026-11-23"), Amount: dec("12.3"),
		UpdatedAt: &updated, FxDate: ptr(day("2026-11-21")), FxRate: decPtr("1336.65"),
	})
	if p.Amount != "12.30" || p.Currency != "USD" || p.Source != "tcgcsv" || p.Date != "2026-11-23" {
		t.Fatalf("price = %+v", p)
	}
	if deref(p.KRW) != "16441" || deref(p.FxDate) != "2026-11-21" {
		t.Fatalf("krw = %s, fxDate = %s", deref(p.KRW), deref(p.FxDate))
	}
	if deref(p.UpdatedAt) != "2026-11-24T05:05:58+09:00" {
		t.Fatalf("updatedAt = %s", deref(p.UpdatedAt))
	}

	// 환율이 없으면 krw·fxDate 모두 nil
	p = NewPrice(LatestPrice{Source: "tcgcsv", Currency: "USD", Date: day("2026-11-23"), Amount: dec("1")})
	if p.KRW != nil || p.FxDate != nil || p.UpdatedAt != nil {
		t.Fatalf("price = %+v", p)
	}
}

func TestPickRepresentative(t *testing.T) {
	lp := func(id int64, variant, source, amount string) LatestPrice {
		return LatestPrice{VariantID: id, Variant: variant, Source: source, Currency: "USD", Date: day("2026-11-23"), Amount: dec(amount)}
	}
	cases := []struct {
		name     string
		language string
		prices   []LatestPrice
		want     string // amount, 빈 문자열이면 nil
	}{
		{"가격 없음", "en", nil, ""},
		{"TCGCSV 가 Cardmarket 보다 먼저", "en", []LatestPrice{lp(1, "normal", "tcgdex_cardmarket", "1"), lp(2, "reverse_holofoil", "tcgcsv", "2")}, "2.00"},
		{"같은 소스에서는 변형 순서", "en", []LatestPrice{lp(1, "reverse_holofoil", "tcgcsv", "1"), lp(2, "holofoil", "tcgcsv", "2"), lp(3, "normal", "tcgcsv", "3")}, "3.00"},
		{"초판보다 무제한", "en", []LatestPrice{lp(1, "first_edition_holofoil", "tcgcsv", "1"), lp(2, "unlimited_holofoil", "tcgcsv", "2")}, "2.00"},
		{"TCGCSV 가 없으면 Cardmarket", "ja", []LatestPrice{lp(1, "normal", "tcgdex_cardmarket", "4")}, "4.00"},
		{"판매가가 아닌 소스는 무시", "en", []LatestPrice{lp(1, "normal", "admin", "9")}, ""},
		{"한글판은 nil", "ko", []LatestPrice{lp(1, "normal", "tcgcsv", "1")}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := PickRepresentative(c.language, c.prices)
			if c.want == "" {
				if got != nil {
					t.Fatalf("got %+v, want nil", got)
				}
				return
			}
			if got == nil || got.Amount != c.want {
				t.Fatalf("got %+v, want amount %s", got, c.want)
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func ptr[T any](v T) *T { return &v }

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
