package server

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/api"
)

// cardID 는 검색으로 카드 ID 를 찾는다 (cards.id 는 적재 순서에 따라 바뀐다).
func cardID(t *testing.T, h http.Handler, query, tcgdexID string) int64 {
	t.Helper()
	for _, c := range search(t, h, query).Items {
		if c.TcgdexID == tcgdexID {
			return c.ID
		}
	}
	t.Fatalf("%s 를 찾지 못함", tcgdexID)
	return 0
}

func getDetail(t *testing.T, h http.Handler, id int64) api.CardDetail {
	t.Helper()
	var d api.CardDetail
	rec := do(t, h, http.MethodGet, fmt.Sprintf("/cards/%d", id), &d)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /cards/%d = %d %s", id, rec.Code, rec.Body.String())
	}
	return d
}

func getPrices(t *testing.T, h http.Handler, id int64, query string) api.PriceHistoryResponse {
	t.Helper()
	var r api.PriceHistoryResponse
	rec := do(t, h, http.MethodGet, fmt.Sprintf("/cards/%d/prices%s", id, query), &r)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /cards/%d/prices%s = %d %s", id, query, rec.Code, rec.Body.String())
	}
	return r
}

// variantSummary 는 변형별 소스·금액 목록을 문자열로 만든다 (비교용).
func variantSummary(vs []api.VariantPrices) string {
	s := ""
	for _, v := range vs {
		s += v.Variant + "["
		for _, p := range v.Prices {
			s += fmt.Sprintf("%s:%s %s %s,", p.Source, p.Amount, p.Currency, p.Date)
		}
		s += "] "
	}
	return s
}

func TestCardDetail(t *testing.T) {
	h := testServer(t)

	d := getDetail(t, h, cardID(t, h, q("sv03.5 25"), "sv03.5-025"))
	if d.TcgdexID != "sv03.5-025" || d.Name != "Pikachu" || d.Language != "en" || d.LocalID != "025" ||
		fmt.Sprint(d.DexIDs) != "[25]" || d.Rarity != nil || deref(d.SpeciesNameKo) != "피카츄" {
		t.Errorf("card = %+v", d)
	}
	if d.Set.TcgdexID != "sv03.5" || d.Set.Name != "151" || deref(d.Set.ReleaseDate) != "2023-09-22" || d.Set.LogoURL != nil || d.Set.CardCount != nil {
		t.Errorf("set = %+v", d.Set)
	}
	// 변형 id 순, 변형 안에서는 TCGCSV → Cardmarket
	want := "normal[tcgcsv:1.23 USD 2026-11-23,tcgdex_cardmarket:0.90 EUR 2026-11-24,] reverse_holofoil[tcgcsv:5.00 USD 2026-11-23,] "
	if got := variantSummary(d.Variants); got != want {
		t.Errorf("variants = %s\nwant       %s", got, want)
	}
	if d.Price == nil || d.Price.Amount != "1.23" || deref(d.Variants[0].Prices[0].KRW) != "1644" {
		t.Errorf("price = %+v", d.Price)
	}
	if d.Linked == nil || len(d.Linked) != 0 {
		t.Errorf("linked = %v, want []", d.Linked)
	}

	// 트레이너스: 도감 번호 [] (null 아님), 시세 없는 변형은 prices []
	tr := getDetail(t, h, cardID(t, h, q("exp"), "sv03.5-166"))
	if tr.DexIDs == nil || len(tr.DexIDs) != 0 || len(tr.Variants) != 1 || tr.Variants[0].Prices == nil || len(tr.Variants[0].Prices) != 0 || tr.Price != nil {
		t.Errorf("trainer = %+v", tr)
	}

	// 한글판 → 일본판 연결 (일본판 대표가 있음), 일본판 → 한글판 연결 (한글판 대표가 null)
	ko := getDetail(t, h, cardID(t, h, q("피카츄")+"&lang=ko", "SV4K-025"))
	if len(ko.Linked) != 1 || ko.Linked[0].TcgdexID != "SV2a-025" || ko.Linked[0].Price == nil || ko.Linked[0].Price.Amount != "2.00" {
		t.Errorf("ko linked = %+v", ko.Linked)
	}
	if ko.Price != nil || variantSummary(ko.Variants) != "normal[tcgcsv:9.99 USD 2026-11-23,] " {
		t.Errorf("ko price = %+v variants = %s", ko.Price, variantSummary(ko.Variants))
	}
	ja := getDetail(t, h, ko.Linked[0].ID)
	if len(ja.Linked) != 1 || ja.Linked[0].TcgdexID != "SV4K-025" || ja.Linked[0].Price != nil || !ja.HasKoEdition {
		t.Errorf("ja linked = %+v", ja.Linked)
	}
}

func TestCardDetailErrors(t *testing.T) {
	h := testServer(t)
	expectError(t, h, http.MethodGet, "/cards/999999", http.StatusNotFound, "not_found")
	expectError(t, h, http.MethodGet, "/cards/abc", http.StatusBadRequest, "invalid_param")
	expectError(t, h, http.MethodGet, "/cards/0", http.StatusBadRequest, "invalid_param")
	expectError(t, h, http.MethodGet, "/cards/999999/prices", http.StatusNotFound, "not_found")
	expectError(t, h, http.MethodGet, "/cards/-1/prices", http.StatusBadRequest, "invalid_param")
	expectError(t, h, http.MethodGet, "/cards/1/prices?period=2w", http.StatusBadRequest, "invalid_param")
}

// seriesKey 는 계열을 "변형/소스" 로 나타낸다.
func seriesKey(s api.PriceSeries) string { return s.Variant + "/" + s.Source }

func points(s api.PriceSeries) string {
	out := ""
	for _, p := range s.Points {
		out += fmt.Sprintf("%s=%s(%s) ", p.Date, p.Amount, deref(p.KRW))
	}
	return out
}

func TestCardPrices(t *testing.T) {
	h := testServer(t)
	id := cardID(t, h, q("sv03.5 25"), "sv03.5-025")

	// 기본 30d: 2026-10-26 부터. 10-20 가격은 빠진다
	r := getPrices(t, h, id, "")
	if r.CardID != id || r.Period != "30d" || r.From != "2026-10-26" || len(r.Series) != 3 {
		t.Fatalf("response = %+v", r)
	}
	if k := []string{seriesKey(r.Series[0]), seriesKey(r.Series[1]), seriesKey(r.Series[2])}; fmt.Sprint(k) != "[normal/tcgcsv normal/tcgdex_cardmarket reverse_holofoil/tcgcsv]" {
		t.Fatalf("series = %v", k)
	}
	n := r.Series[0]
	// 11-22·11-23 모두 직전 환율 11-21(1336.65): 1.10 → 1470.315 → 1470, 1.23 → 1644.0795 → 1644
	if n.Currency != "USD" || points(n) != "2026-11-22=1.10(1470) 2026-11-23=1.23(1644) " {
		t.Errorf("normal/tcgcsv points = %s", points(n))
	}
	// 평균 1.165 → 1.17, 변동률 0.13 / 1.10 = 0.118182
	if n.Summary.Current.Amount != "1.23" || n.Summary.Average != "1.17" || n.Summary.Min != "1.10" || n.Summary.Max != "1.23" ||
		n.Summary.From != "2026-11-22" || n.Summary.ChangeRate == nil || *n.Summary.ChangeRate != 0.118182 {
		t.Errorf("normal/tcgcsv summary = %+v rate %v", n.Summary, n.Summary.ChangeRate)
	}
	// Cardmarket: EUR 환율이 없어 원화 null, 가격 하나라 변동률 null
	cm := r.Series[1]
	if cm.Currency != "EUR" || points(cm) != "2026-11-24=0.90(<nil>) " || cm.Summary.ChangeRate != nil || cm.Summary.Current.KRW != nil {
		t.Errorf("normal/cardmarket = %s %+v", points(cm), cm.Summary)
	}

	// 90d: 10-20 포함. 첫 환율(11-20)보다 이른 날은 원화 null
	r = getPrices(t, h, id, "?period=90d")
	n = r.Series[0]
	if r.From != "2026-08-27" || points(n) != "2026-10-20=2.00(<nil>) 2026-11-22=1.10(1470) 2026-11-23=1.23(1644) " ||
		n.Summary.From != "2026-10-20" || n.Summary.ChangeRate == nil || *n.Summary.ChangeRate != -0.385 || n.Summary.Max != "2.00" {
		t.Errorf("90d = %s %+v", points(n), n.Summary)
	}

	if r := getPrices(t, h, id, "?period=7d"); r.From != "2026-11-18" || len(r.Series) != 3 {
		t.Errorf("7d = %+v", r)
	}
	if r := getPrices(t, h, id, "?period=1y"); r.From != "2025-11-25" || len(r.Series[0].Points) != 3 {
		t.Errorf("1y = %+v", r)
	}

	// 시세 없는 카드: series 는 [] (null 아님)
	tr := getPrices(t, h, cardID(t, h, q("exp"), "sv03.5-166"), "")
	if tr.Series == nil || len(tr.Series) != 0 {
		t.Errorf("trainer series = %v", tr.Series)
	}
}
