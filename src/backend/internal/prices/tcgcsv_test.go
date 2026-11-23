package prices

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbq"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/httpx"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/tcgcsv"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/tcgdex"
)

func TestResolveVariant(t *testing.T) {
	v := func(codes ...string) []cardVariant {
		out := make([]cardVariant, len(codes))
		for i, c := range codes {
			out[i] = cardVariant{ID: int64(i + 1), Variant: c}
		}
		return out
	}
	cases := []struct {
		name, lang string
		variants   []cardVariant
		sub        string
		want       string // 빈 문자열이면 연결 안 됨
	}{
		{"같은 변형", "en", v("normal", "reverse_holofoil"), "Reverse Holofoil", "reverse_holofoil"},
		{"초판 세트 Normal → 무제한판", "en", v("first_edition", "unlimited"), "Normal", "unlimited"},
		{"초판 세트 Holofoil → 무제한 홀로", "en", v("first_edition_holofoil", "unlimited_holofoil"), "Holofoil", "unlimited_holofoil"},
		{"초판 세트 1st Edition", "en", v("first_edition", "unlimited"), "1st Edition", "first_edition"},
		{"영어판 없는 변형", "en", v("normal"), "Reverse Holofoil", ""},
		{"일본판 단일 변형", "ja", v("normal"), "Holofoil", "normal"},
		{"일본판 변형 여럿", "ja", v("normal", "reverse_holofoil"), "Holofoil", ""},
		{"모르는 하위 유형", "ja", v("normal"), "Foil Etched", ""},
	}
	for _, c := range cases {
		got, ok := resolveVariant(c.lang, c.variants, c.sub)
		if (c.want == "") == ok || (ok && got.Variant != c.want) {
			t.Errorf("%s: got %+v ok=%v, want %q", c.name, got, ok, c.want)
		}
	}
}

func TestNormalizeNumber(t *testing.T) {
	for in, want := range map[string]string{
		"001/165": "1", "014": "14", "1/149": "1", "TG01/TG30": "TG01", "swsh001": "SWSH001", "000": "0", " 25 ": "25", "": "",
	} {
		if got := normalizeNumber(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
	if !isSpecialProduct("Mew ex - 205/165 (151 Metal Card)") || !isSpecialProduct("Pikachu [Staff]") || isSpecialProduct("Mew ex - 205/165") {
		t.Error("isSpecialProduct")
	}
}

func TestWantGroup(t *testing.T) {
	g151 := tcgcsv.Group{GroupID: 23237, Name: "SV: Scarlet & Violet 151"}
	gSV4K := tcgcsv.Group{GroupID: 23610, Name: "SV4K: Ancient Roar"}
	opt := Options{SetIDs: []string{"sv4k"}, TCGCSVGroups: []int{23237}}
	if !wantGroup(opt, g151) || !wantGroup(opt, gSV4K) || wantGroup(opt, tcgcsv.Group{GroupID: 1, Name: "SV2a: Pokemon Card 151"}) {
		t.Fatal("범위 선택")
	}
	if !wantGroup(Options{}, g151) {
		t.Fatal("범위가 없으면 모두")
	}
}

// 테스트 카탈로그 (가격 연결 대상)
//
//	en sv03.5: 001(normal, reverse_holofoil), 002(normal), 205(holofoil)
//	en A1 (TCG Pocket): 001(normal)
//	ja SV4K:   001(normal), 002(normal)
type testCatalog struct {
	variants map[string]int64 // "카드ID/변형" → card_variants.id
}

func seedCatalog(t *testing.T, pool *pgxpool.Pool) *testCatalog {
	t.Helper()
	ctx := context.Background()
	q := dbq.New(pool)
	tc := &testCatalog{variants: map[string]int64{}}
	sets := []struct {
		lang, set string
		cards     map[string][]string // localId → 변형
	}{
		{"en", "sv03.5", map[string][]string{"001": {"normal", "reverse_holofoil"}, "002": {"normal"}, "205": {"holofoil"}}},
		{"en", "A1", map[string][]string{"001": {"normal"}}},
		{"ja", "SV4K", map[string][]string{"001": {"normal"}, "002": {"normal"}}},
	}
	for _, s := range sets {
		setID, err := q.UpsertSet(ctx, dbq.UpsertSetParams{Language: s.lang, TcgdexID: s.set, Name: s.set})
		if err != nil {
			t.Fatal(err)
		}
		for local, variants := range s.cards {
			id := s.set + "-" + local
			cardID, err := q.UpsertCard(ctx, dbq.UpsertCardParams{SetID: setID, Language: s.lang, TcgdexID: id, LocalID: local, Name: id})
			if err != nil {
				t.Fatal(err)
			}
			for _, v := range variants {
				vid, err := q.UpsertCardVariant(ctx, dbq.UpsertCardVariantParams{CardID: cardID, Variant: v})
				if err != nil {
					t.Fatal(err)
				}
				tc.variants[id+"/"+v] = vid
			}
		}
	}
	return tc
}

// fakeTCGCSV 는 경로별 JSON 을 돌려준다. 없는 경로는 404 이다.
type fakeTCGCSV struct {
	mu    sync.Mutex
	body  map[string]string
	paths []string
}

func (f *fakeTCGCSV) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.paths = append(f.paths, r.URL.Path)
	body, ok := f.body[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	_, _ = w.Write([]byte(body))
}

func ok(results string) string { return `{"success":true,"errors":[],"results":[` + results + `]}` }

func product(id int, name, number string) string {
	ext := ""
	if number != "" {
		ext = `{"name":"Number","value":"` + number + `"}`
	}
	return `{"productId":` + strconv.Itoa(id) + `,"name":"` + name + `","extendedData":[` + ext + `]}`
}

func price(id int, sub, market, mid string) string {
	return `{"productId":` + strconv.Itoa(id) + `,"subTypeName":"` + sub + `","lowPrice":0.01,"midPrice":` + mid +
		`,"highPrice":99,"marketPrice":` + market + `,"directLowPrice":null}`
}

// newFakeTCGCSV 는 다음 그룹을 내려 준다.
//
//	en 23237 "SV: Scarlet & Violet 151" (코드가 TCGdex 세트 ID 와 다름 → 방법 A 결과로 대응)
//	  501999 ETB(번호 없음), 502552 001(Normal·Reverse), 502553 002(Normal 값 없음·Reverse → Q5),
//	  517051 205 Holofoil(market 없음 → mid), 519481 205 Metal Card(특별 상품)
//	en 999 "Base Set" (대응 없음)
//	ja 23610 "SV4K: Ancient Roar" (코드 대응)
//	  565756 001 Holofoil(단일 변형 → normal), 565757 002 Normal·Holofoil(한 변형에 둘 → 모호)
func newFakeTCGCSV() *fakeTCGCSV {
	return &fakeTCGCSV{body: map[string]string{
		"/last-updated.txt":    "2026-11-22T20:05:55+0000",
		"/tcgplayer/3/groups":  ok(`{"groupId":23237,"name":"SV: Scarlet & Violet 151","categoryId":3},{"groupId":999,"name":"Base Set","categoryId":3}`),
		"/tcgplayer/85/groups": ok(`{"groupId":23610,"name":"SV4K: Ancient Roar","categoryId":85}`),
		"/tcgplayer/3/23237/products": ok(strings.Join([]string{
			product(501999, "151 Elite Trainer Box", ""),
			product(502552, "Bulbasaur - 001/165", "001/165"),
			product(502553, "Ivysaur - 002/165", "002/165"),
			product(517051, "Mew ex - 205/165", "205/165"),
			product(519481, "Mew ex - 205/165 (151 Metal Card)", "205/165"),
		}, ",")),
		"/tcgplayer/3/23237/prices": ok(strings.Join([]string{
			price(501999, "Normal", "1097.54", "1300"),
			price(502552, "Normal", "0.27", "0.26"),
			price(502552, "Reverse Holofoil", "0.32", "0.37"),
			price(502553, "Normal", "null", "null"),
			price(502553, "Reverse Holofoil", "0.45", "0.48"),
			price(517051, "Holofoil", "null", "27.63"),
			price(519481, "Normal", "23.73", "26"),
		}, ",")),
		"/tcgplayer/3/999/products":    ok(product(700, "Alakazam", "1/102")),
		"/tcgplayer/3/999/prices":      ok(price(700, "Holofoil", "300", "310")),
		"/tcgplayer/85/23610/products": ok(product(565756, "Pansage", "001/066") + "," + product(565757, "Simisage", "002/066")),
		"/tcgplayer/85/23610/prices": ok(strings.Join([]string{
			price(565756, "Holofoil", "0.5", "0.6"),
			price(565757, "Normal", "0.1", "0.2"),
			price(565757, "Holofoil", "0.3", "0.4"),
		}, ",")),
	}}
}

func newTCGCSVSource(t *testing.T, pool *pgxpool.Pool, f *fakeTCGCSV) *TCGCSV {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return NewTCGCSV(pool, tcgcsv.New(tcgcsv.WithBaseURL(srv.URL), tcgcsv.WithHTTP(httpx.New("test", httpx.WithRetry(0, 0)))))
}

type priceRow struct {
	Variant, Date, Price, Basis string
}

func priceRows(t *testing.T, pool *pgxpool.Pool) map[string]priceRow {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT c.tcgdex_id || '/' || v.variant, p.price_date::text, p.price::text, p.price_basis
		FROM price_history p JOIN card_variants v ON v.id = p.card_variant_id JOIN cards c ON c.id = v.card_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]priceRow{}
	for rows.Next() {
		var r priceRow
		if err := rows.Scan(&r.Variant, &r.Date, &r.Price, &r.Basis); err != nil {
			t.Fatal(err)
		}
		out[r.Variant] = r
	}
	return out
}

func countRows(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func seedLink(t *testing.T, pool *pgxpool.Pool, product, sub string, variantID int64) {
	t.Helper()
	ctx := context.Background()
	q := dbq.New(pool)
	src, err := q.GetPriceSourceByCode(ctx, TCGCSVCode)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.UpsertExternalID(ctx, dbq.UpsertExternalIDParams{SourceID: src.ID, ExternalID: product, ExternalSubID: sub, CardVariantID: variantID}); err != nil {
		t.Fatal(err)
	}
}

// wantTCGCSVPrices 는 고정 응답으로 저장되어야 하는 가격이다.
var wantTCGCSVPrices = map[string]priceRow{
	"sv03.5-001/normal":           {"sv03.5-001/normal", "2026-11-22", "0.27", "market"},
	"sv03.5-001/reverse_holofoil": {"sv03.5-001/reverse_holofoil", "2026-11-22", "0.32", "market"},
	"sv03.5-002/reverse_holofoil": {"sv03.5-002/reverse_holofoil", "2026-11-22", "0.45", "market"}, // Q5 로 추가한 변형
	"sv03.5-205/holofoil":         {"sv03.5-205/holofoil", "2026-11-22", "27.63", "mid"},
	"SV4K-001/normal":             {"SV4K-001/normal", "2026-11-22", "0.50", "market"},
}

func checkTCGCSVPrices(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	got := priceRows(t, pool)
	if len(got) != len(wantTCGCSVPrices) {
		t.Errorf("price_history = %+v", got)
	}
	for k, want := range wantTCGCSVPrices {
		if got[k] != want {
			t.Errorf("%s: got %+v, want %+v", k, got[k], want)
		}
	}
}

func TestTCGCSVCollectLinksAndIsIdempotent(t *testing.T) {
	pool := testPool(t)
	tc := seedCatalog(t, pool)
	// 방법 A 로 이미 저장된 연결 하나 (그룹 23237 ↔ sv03.5 대응의 근거)
	seedLink(t, pool, "502552", "Normal", tc.variants["sv03.5-001/normal"])
	// 특별 상품(Metal Card)을 가리키는 연결. 같은 번호의 기본 상품(517051)이 있으므로 쓰지 않아야 한다.
	seedLink(t, pool, "519481", "Normal", tc.variants["sv03.5-205/holofoil"])
	f := newFakeTCGCSV()
	src := newTCGCSVSource(t, pool, f)
	ctx := context.Background()

	res, err := src.Collect(ctx, Options{})
	if err != nil {
		t.Fatal(err)
	}
	// 가격 행 11 = 연결 6(저장 5 + 값 없음 1) + 건너뜀 5(비카드·특별·대응 없음·모호 2)
	if res.Fetched != 11 || res.Linked != 6 || res.Saved != 5 || res.Skipped != 6 || len(res.Failed) != 0 {
		t.Fatalf("res = %+v", res)
	}
	checkTCGCSVPrices(t, pool)
	// 외부 ID: 기존 2 + 방법 B 5 (값 없는 행도 연결은 저장)
	if n := countRows(t, pool, "external_ids"); n != 7 {
		t.Fatalf("external_ids = %d", n)
	}
	// Q5: sv03.5-002 에 reverse_holofoil 추가 (기존 변형 7 + 1)
	if n := countRows(t, pool, "card_variants"); n != 8 {
		t.Fatalf("card_variants = %d", n)
	}

	// 다시 실행해도 행이 늘지 않는다. 이번에는 저장된 연결로 1단계에서 모두 연결된다.
	res, err = src.Collect(ctx, Options{})
	if err != nil || res.Linked != 6 || res.Saved != 5 {
		t.Fatalf("2nd res=%+v err=%v", res, err)
	}
	checkTCGCSVPrices(t, pool)
	if countRows(t, pool, "external_ids") != 7 || countRows(t, pool, "card_variants") != 8 || countRows(t, pool, "price_history") != 5 {
		t.Fatal("재실행으로 행 수가 바뀜")
	}
}

func TestTCGCSVGroupFailureIsIsolated(t *testing.T) {
	pool := testPool(t)
	seedCatalog(t, pool)
	f := newFakeTCGCSV()
	delete(f.body, "/tcgplayer/3/23237/prices") // 404 → 재시도 후에도 실패
	res, err := newTCGCSVSource(t, pool, f).Collect(context.Background(), Options{})
	if err == nil || !slices.Equal(res.Failed, []string{"en/23237"}) {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	// 다른 그룹(ja SV4K)은 저장된다
	got := priceRows(t, pool)
	if len(got) != 1 || got["SV4K-001/normal"].Price != "0.50" {
		t.Fatalf("price_history = %+v", got)
	}
	n := 0
	for _, p := range f.paths {
		if p == "/tcgplayer/3/23237/prices" {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("실패 그룹은 끝에 한 번 더 받아야 함: %d회", n)
	}
}

// fakeTCGdex 는 카드 상세를 돌려주는 TCGdex 이다. 응답을 늦춰 TCGCSV 가 기다리는지 확인한다.
type fakeTCGdex struct {
	mu    sync.Mutex
	paths []string
	delay time.Duration
	cards map[string]string // 경로 → JSON, 값이 "503" 이면 서버 오류
}

func (f *fakeTCGdex) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.paths = append(f.paths, r.URL.Path)
	body, ok := f.cards[r.URL.Path]
	f.mu.Unlock()
	time.Sleep(f.delay)
	switch {
	case !ok:
		http.NotFound(w, r)
	case body == "503":
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		_, _ = w.Write([]byte(body))
	}
}

func TestRunnerTCGdexThenTCGCSV(t *testing.T) {
	pool := testPool(t)
	seedCatalog(t, pool)
	card001, err := os.ReadFile("../tcgdex/testdata/card_en_sv03.5-001_pricing.json")
	if err != nil {
		t.Fatal(err)
	}
	ft := &fakeTCGdex{delay: 50 * time.Millisecond, cards: map[string]string{
		"/en/series/tcgp":      `{"id":"tcgp","name":"Pokémon TCG Pocket","sets":[{"id":"A1","name":"Genetic Apex"}]}`,
		"/en/cards/sv03.5-001": string(card001),
		"/en/cards/sv03.5-002": "503",
		"/en/cards/sv03.5-205": `{"id":"sv03.5-205","localId":"205","pricing":null}`,
		"/ja/cards/SV4K-001":   `{"id":"SV4K-001","localId":"001"}`,
		"/ja/cards/SV4K-002":   `{"id":"SV4K-002","localId":"002","pricing":{"cardmarket":null,"tcgplayer":null}}`,
	}}
	tsrv := httptest.NewServer(ft)
	defer tsrv.Close()
	td := NewTCGdex(pool, tcgdex.New(tcgdex.WithBaseURL(tsrv.URL), tcgdex.WithMinInterval(0), tcgdex.WithRetry(1, time.Millisecond)))
	tcsv := newTCGCSVSource(t, pool, newFakeTCGCSV())

	results, err := NewRunner(&FX{}, tcsv, td).Run(context.Background(), Options{Sources: []string{TCGCSVCode, CardmarketCode}})
	if err == nil || !strings.Contains(err.Error(), CardmarketCode) || strings.Contains(err.Error(), TCGCSVCode+":") {
		t.Fatalf("TCGdex 실패만 오류여야 함: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %+v", results)
	}
	csvRes, tdRes := results[0], results[1]
	// TCGdex: 5장 중 1장 실패(끝에 다시 시도), 상품 ID 2개 연결, TCG Pocket(A1) 카드는 부르지 않음
	if tdRes.Fetched != 4 || tdRes.Linked != 2 || !slices.Equal(tdRes.Failed, []string{"en/sv03.5-002"}) {
		t.Fatalf("tcgdex = %+v", tdRes)
	}
	ft.mu.Lock()
	for _, p := range ft.paths {
		if strings.Contains(p, "A1-") {
			t.Errorf("TCG Pocket 카드 요청: %s", p)
		}
	}
	ft.mu.Unlock()
	// TCGCSV: 방법 A 연결이 끝난 뒤 그룹 23237 을 sv03.5 에 대응시켜 같은 결과를 낸다
	if csvRes.Saved != 5 || len(csvRes.Failed) != 0 {
		t.Fatalf("tcgcsv = %+v", csvRes)
	}
	checkTCGCSVPrices(t, pool)
}
