package server

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/api"
)

// moversSQL 은 기본 고정 데이터(TCGCSV 기준일 11-23)에 비교일 11-16 가격을 더한다.
//
//	sv03.5-025 normal          1.00 → 1.23  (+23%)
//	sv03.5-025 reverse_holofoil 4.00 → 5.00  (+25%, 같은 카드에서 더 큼)
//	sv03.5-006 normal          2.00 → 1.00  (−50%)
//	sv03.5-166 normal          0.50 → 3.00  (비교일 $1 미만이라 제외)
//	base1-58   first_edition 100.00 → 150.00 (+50%)
//	SV4K-025   normal          9.99 → 9.99  (변동 없음, 양쪽 모두 제외)
//
// 30d 비교일은 11-23 − 30일 이하 가장 가까운 10-20 이고, sv03.5-025 normal 2.00 → 1.23 (−38.5%) 만 있다.
const moversSQL = `
INSERT INTO price_history (card_variant_id, source_id, price_date, price, price_basis, source_updated_at)
SELECT v.id, src.id, x.d::date, x.price, 'market', (x.d::date + time '20:00') AT TIME ZONE 'UTC'
FROM (VALUES
  ('sv03.5-025', 'normal', '2026-11-16', 1.00),
  ('sv03.5-025', 'reverse_holofoil', '2026-11-16', 4.00),
  ('sv03.5-006', 'normal', '2026-11-16', 2.00),
  ('sv03.5-006', 'normal', '2026-11-23', 1.00),
  ('sv03.5-166', 'normal', '2026-11-16', 0.50),
  ('sv03.5-166', 'normal', '2026-11-23', 3.00),
  ('base1-58', 'first_edition', '2026-11-16', 100.00),
  ('base1-58', 'first_edition', '2026-11-23', 150.00),
  ('SV4K-025', 'normal', '2026-11-16', 9.99)
) AS x(card, variant, d, price)
JOIN cards c ON c.tcgdex_id = x.card
JOIN card_variants v ON v.card_id = c.id AND v.variant = x.variant
JOIN price_sources src ON src.code = 'tcgcsv';
`

func getMovers(t *testing.T, h http.Handler, query string) api.MoversResponse {
	t.Helper()
	var r api.MoversResponse
	rec := do(t, h, http.MethodGet, "/rankings/movers?"+query, &r)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /rankings/movers?%s = %d %s", query, rec.Code, rec.Body.String())
	}
	return r
}

func moverList(r api.MoversResponse) string {
	out := ""
	for _, m := range r.Items {
		out += fmt.Sprintf("%s/%s %s→%s %v; ", m.Card.TcgdexID, m.Variant, m.Previous.Amount, m.Current.Amount, m.ChangeRate)
	}
	return out
}

func TestMovers(t *testing.T) {
	h, pool := testServerPool(t)
	if _, err := pool.Exec(context.Background(), moversSQL); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		query, want string
	}{
		// 상승: 0보다 큰 것만 큰 순, 카드당 변형 하나 (sv03.5-025 는 reverse 25% 가 normal 23% 보다 큼)
		{"period=7d&direction=up", "base1-58/first_edition 100.00→150.00 0.5; sv03.5-025/reverse_holofoil 4.00→5.00 0.25; "},
		{"period=7d&direction=down", "sv03.5-006/normal 2.00→1.00 -0.5; "},
		{"period=7d&direction=up&limit=1", "base1-58/first_edition 100.00→150.00 0.5; "},
		{"period=30d&direction=down", "sv03.5-025/normal 2.00→1.23 -0.385; "},
		{"period=30d&direction=up", ""},
	}
	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			r := getMovers(t, h, c.query)
			if got := moverList(r); got != c.want || deref(r.AsOf) != "2026-11-23" || r.Items == nil {
				t.Fatalf("got %q (asOf %s), want %q", got, deref(r.AsOf), c.want)
			}
		})
	}

	r := getMovers(t, h, "period=7d&direction=up")
	if r.Period != "7d" || r.Direction != "up" {
		t.Fatalf("response = %+v", r)
	}
	m := r.Items[0]
	// 기준일 가격: 직전 환율 11-21 로 원화 150 × 1336.65 = 200497.5 → 200498. 비교일 11-16 은 환율이 없어 null
	if m.Current.Date != "2026-11-23" || m.Current.Source != "tcgcsv" || m.Current.Currency != "USD" ||
		deref(m.Current.KRW) != "200498" || deref(m.Current.FxDate) != "2026-11-21" ||
		deref(m.Current.UpdatedAt) != "2026-11-24T05:00:00+09:00" {
		t.Errorf("current = %+v krw=%s fx=%s updated=%s", m.Current, deref(m.Current.KRW), deref(m.Current.FxDate), deref(m.Current.UpdatedAt))
	}
	if m.Previous.Date != "2026-11-16" || m.Previous.KRW != nil || m.Previous.FxDate != nil {
		t.Errorf("previous = %+v", m.Previous)
	}
	// card 는 CardSummary (대표가 포함)
	if m.Card.Set.TcgdexID != "base1" || m.Card.Price == nil || m.Card.Price.Amount != "150.00" || deref(m.Card.SpeciesNameKo) != "피카츄" {
		t.Errorf("card = %+v", m.Card)
	}
}

func TestMoversNoData(t *testing.T) {
	h, pool := testServerPool(t)
	ctx := context.Background()

	// 비교일이 없음: 기준일(11-23)만 알려 주고 빈 목록. 기본 데이터의 30d 비교일(10-20)도 지운다
	if _, err := pool.Exec(ctx, "DELETE FROM price_history WHERE price_date < '2026-11-22'"); err != nil {
		t.Fatal(err)
	}
	r := getMovers(t, h, "period=30d&direction=down")
	if deref(r.AsOf) != "2026-11-23" || r.Items == nil || len(r.Items) != 0 {
		t.Fatalf("no compare date = %+v", r)
	}

	// 시세 없음: asOf null, 빈 목록 (오류 아님)
	if _, err := pool.Exec(ctx, "DELETE FROM price_history"); err != nil {
		t.Fatal(err)
	}
	r = getMovers(t, h, "period=7d&direction=up")
	if r.AsOf != nil || r.Items == nil || len(r.Items) != 0 {
		t.Fatalf("no data = %+v", r)
	}
}

func TestMoversValidation(t *testing.T) {
	h := New(fakeDB{}, nil) // 검증 오류는 DB 에 닿기 전에 반환된다
	for _, p := range []string{
		"direction=up",
		"period=90d&direction=up",
		"period=7d",
		"period=7d&direction=sideways",
		"period=7d&direction=up&limit=0",
		"period=7d&direction=up&limit=51",
		"period=7d&direction=up&limit=x",
	} {
		expectError(t, h, http.MethodGet, "/rankings/movers?"+p, http.StatusBadRequest, "invalid_param")
	}
}
