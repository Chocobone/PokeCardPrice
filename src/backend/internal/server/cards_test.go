package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/api"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbq"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbtest"
)

// fixtureSQL 은 검색·시세 테스트용 카탈로그와 가격이다.
//
//	en sv03.5 "151"        (2023-09-22) 025 Pikachu [normal, reverse_holofoil], 006 Charizard ex, 166 Exp. Share(트레이너스)
//	en base1  "Base"       (1999-01-09) 058 Pikachu [first_edition]
//	ja SV2a   "ポケモンカード151" (2023-06-16) 025 ピカチュウ [normal]
//	ko SV4K   "고대의 포효"   (2023-11-30) 025 피카츄 [normal] → 일본판 SV2a-025 와 연결
const fixtureSQL = `
INSERT INTO pokemon_species (dex_id, name_en, name_ja, name_ko) VALUES
  (6, 'Charizard', 'リザードン', '리자몽'), (25, 'Pikachu', 'ピカチュウ', '피카츄');
INSERT INTO sets (language, tcgdex_id, name, release_date) VALUES
  ('en', 'sv03.5', '151', '2023-09-22'), ('en', 'base1', 'Base', '1999-01-09'),
  ('ja', 'SV2a', 'ポケモンカード151', '2023-06-16'), ('ko', 'SV4K', '고대의 포효', '2023-11-30');
INSERT INTO cards (set_id, language, tcgdex_id, local_id, name, dex_ids, image_url)
SELECT s.id, s.language, x.tcgdex_id, x.local_id, x.name, x.dex_ids, x.image_url
FROM (VALUES
  ('en', 'sv03.5', 'sv03.5-025', '025', 'Pikachu', '{25}'::int[], 'https://assets.tcgdex.net/en/sv/sv03.5/025'),
  ('en', 'sv03.5', 'sv03.5-006', '006', 'Charizard ex', '{6}'::int[], NULL),
  ('en', 'sv03.5', 'sv03.5-166', '166', 'Exp. Share', NULL, NULL),
  ('en', 'base1', 'base1-58', '58', 'Pikachu', '{25}'::int[], NULL),
  ('ja', 'SV2a', 'SV2a-025', '025', 'ピカチュウ', '{25}'::int[], NULL),
  ('ko', 'SV4K', 'SV4K-025', '025', '피카츄', '{25}'::int[], NULL)
) AS x(language, set_id, tcgdex_id, local_id, name, dex_ids, image_url)
JOIN sets s ON s.language = x.language AND s.tcgdex_id = x.set_id;
UPDATE cards SET has_ko_edition = true WHERE tcgdex_id = 'SV2a-025';
INSERT INTO card_links (card_id, linked_card_id, method)
SELECT k.id, j.id, 'auto' FROM cards k, cards j WHERE k.tcgdex_id = 'SV4K-025' AND j.tcgdex_id = 'SV2a-025';
-- 변형 id 순서가 VALUES 순서와 같도록 ORDER BY 로 넣는다 (JOIN 결과 순서는 정해져 있지 않다)
INSERT INTO card_variants (card_id, variant)
SELECT c.id, x.variant FROM (VALUES
  (1, 'sv03.5-025', 'normal'), (2, 'sv03.5-025', 'reverse_holofoil'), (3, 'sv03.5-006', 'normal'), (4, 'sv03.5-166', 'normal'),
  (5, 'base1-58', 'first_edition'), (6, 'SV2a-025', 'normal'), (7, 'SV4K-025', 'normal')
) AS x(ord, card, variant) JOIN cards c ON c.tcgdex_id = x.card
ORDER BY x.ord;
INSERT INTO fx_rates (base_currency, quote_currency, rate_date, provider, rate) VALUES
  ('USD', 'KRW', '2026-11-20', 'frankfurter', 1330.00), ('USD', 'KRW', '2026-11-21', 'frankfurter', 1336.65);
INSERT INTO price_history (card_variant_id, source_id, price_date, price, price_basis, source_updated_at)
SELECT v.id, src.id, x.d::date, x.price, x.basis, '2026-11-23 20:05:58+00'
FROM (VALUES
  ('sv03.5-025', 'normal', 'tcgcsv', '2026-10-20', 2.00, 'market'),
  ('sv03.5-025', 'normal', 'tcgcsv', '2026-11-22', 1.10, 'market'),
  ('sv03.5-025', 'normal', 'tcgcsv', '2026-11-23', 1.23, 'market'),
  ('sv03.5-025', 'reverse_holofoil', 'tcgcsv', '2026-11-23', 5.00, 'market'),
  ('sv03.5-025', 'normal', 'tcgdex_cardmarket', '2026-11-24', 0.90, 'trend'),
  ('base1-58', 'first_edition', 'tcgdex_cardmarket', '2026-11-24', 120.00, 'trend'),
  ('SV2a-025', 'normal', 'tcgdex_cardmarket', '2026-11-24', 2.00, 'trend'),
  ('SV4K-025', 'normal', 'tcgcsv', '2026-11-23', 9.99, 'market')
) AS x(card, variant, source, d, price, basis)
JOIN cards c ON c.tcgdex_id = x.card
JOIN card_variants v ON v.card_id = c.id AND v.variant = x.variant
JOIN price_sources src ON src.code = x.source;
`

// testServer 는 마이그레이션과 고정 데이터를 넣은 테스트 DB 로 서버를 만든다.
func testServer(t *testing.T) http.Handler {
	t.Helper()
	pool := dbtest.Pool(t, "server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, fixtureSQL); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	return newRouter(pool, dbq.New(pool), func() time.Time { return testNow })
}

// testNow 는 테스트 서버의 현재 시각이다 (2026-11-25 10:00 KST). 30d 시작일은 2026-10-26 이다.
var testNow = time.Date(2026, 11, 25, 1, 0, 0, 0, time.UTC)

func search(t *testing.T, h http.Handler, query string) api.Page[api.CardSummary] {
	t.Helper()
	var page api.Page[api.CardSummary]
	rec := do(t, h, http.MethodGet, "/cards?"+query, &page)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /cards?%s = %d %s", query, rec.Code, rec.Body.String())
	}
	return page
}

func ids(items []api.CardSummary) []string {
	out := make([]string, len(items))
	for i, c := range items {
		out[i] = c.TcgdexID
	}
	return out
}

func q(s string) string { return "q=" + url.QueryEscape(s) }

func TestSearchCards(t *testing.T) {
	h := testServer(t)

	cases := []struct {
		name  string
		query string
		want  []string
	}{
		// 이름이 검색어로 시작(순위 ②) → 발매일 최신순
		{"영문 이름", q("PIKACHU"), []string{"sv03.5-025", "base1-58"}},
		// 한글 종 이름으로 모든 언어의 피카츄 (순위 ②, 발매일 최신순)
		{"한글 종 이름", q("피카츄"), []string{"SV4K-025", "sv03.5-025", "SV2a-025", "base1-58"}},
		{"한글 종 이름 + 언어", q("피카츄") + "&lang=ja", []string{"SV2a-025"}},
		// 트레이너스(dex_ids NULL)가 종 이름 낱말을 통과하면 안 된다 (1단계 발견 결함). 후보는 'exp'(긴 낱말)로 찾는다
		{"트레이너스는 종 이름에 맞지 않음", q("피카 exp"), []string{}},
		// 이름이 ex 로 시작하는 Exp. Share(②)가 Charizard ex(③)보다 앞선다
		{"부분 일치", q("ex"), []string{"sv03.5-166", "sv03.5-006"}},
		// 세트 ID + 번호(앞 0 무시), 순위 ①
		{"세트 ID와 번호", q("sv03.5 25"), []string{"sv03.5-025"}},
		{"번호만", q("25") + "&lang=en", []string{"sv03.5-025"}},
		{"번호 앞 0", q("058"), []string{"base1-58"}},
		// 세트 이름 부분 일치: 151 세트 전체 + 일본판 151. 번호 일치가 없으므로 순위 ③, 발매일순
		{"세트 이름", q("151"), []string{"sv03.5-006", "sv03.5-025", "sv03.5-166", "SV2a-025"}},
		// 번호가 정확히 맞는 카드(순위 ①)가 이름 일치(②)보다 앞선다
		{"번호 우선", q("pikachu 58"), []string{"base1-58"}},
		{"LIKE 특수 문자는 문자 그대로", q("%"), []string{}},
		{"밑줄", q("_"), []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			page := search(t, h, c.query)
			got := ids(page.Items)
			if fmt.Sprint(got) != fmt.Sprint(c.want) || page.Total != int64(len(c.want)) {
				t.Fatalf("got %v (total %d), want %v", got, page.Total, c.want)
			}
		})
	}
}

func TestSearchPaging(t *testing.T) {
	h := testServer(t)

	page := search(t, h, q("피카츄")+"&size=1&page=2")
	if fmt.Sprint(ids(page.Items)) != "[sv03.5-025]" || page.Total != 4 || page.Page != 2 || page.Size != 1 {
		t.Fatalf("page 2 = %+v", page)
	}
	// 마지막 쪽을 넘어도 전체 건수는 알려 준다
	page = search(t, h, q("피카츄")+"&size=2&page=10")
	if len(page.Items) != 0 || page.Total != 4 {
		t.Fatalf("page 10 = %+v", page)
	}
	if page.Items == nil {
		t.Fatal("items 는 null 이 아니라 [] 여야 한다")
	}
	page = search(t, h, q("없는카드"))
	if len(page.Items) != 0 || page.Total != 0 || page.Items == nil {
		t.Fatalf("no match = %+v", page)
	}
}

func TestSearchSummaryFields(t *testing.T) {
	h := testServer(t)

	byID := map[string]api.CardSummary{}
	for _, c := range search(t, h, q("피카츄")).Items {
		byID[c.TcgdexID] = c
	}
	for _, c := range search(t, h, q("exp")).Items {
		byID[c.TcgdexID] = c
	}

	// 영어판: TCGCSV normal 최신가(11-23, 1.23), 직전 환율(11-21)로 원화 환산
	en := byID["sv03.5-025"]
	if en.Price == nil || en.Price.Amount != "1.23" || en.Price.Currency != "USD" || en.Price.Source != "tcgcsv" ||
		en.Price.Date != "2026-11-23" || deref(en.Price.KRW) != "1644" || deref(en.Price.FxDate) != "2026-11-21" ||
		deref(en.Price.UpdatedAt) != "2026-11-24T05:05:58+09:00" {
		t.Errorf("sv03.5-025 price = %+v", en.Price)
	}
	if en.Set != (api.SetRef{TcgdexID: "sv03.5", Name: "151"}) || deref(en.SpeciesNameKo) != "피카츄" ||
		deref(en.ImageURL) != "https://assets.tcgdex.net/en/sv/sv03.5/025" || en.LocalID != "025" || en.Language != "en" {
		t.Errorf("sv03.5-025 = %+v", en)
	}
	// TCGCSV 가 없으면 Cardmarket. EUR 환율이 없으면 krw 는 null
	if p := byID["base1-58"].Price; p == nil || p.Source != "tcgdex_cardmarket" || p.Amount != "120.00" || p.Currency != "EUR" || p.KRW != nil || p.FxDate != nil {
		t.Errorf("base1-58 price = %+v", p)
	}
	if ja := byID["SV2a-025"]; !ja.HasKoEdition || ja.Price == nil {
		t.Errorf("SV2a-025 = %+v", ja)
	}
	// 한글판은 가격이 있어도 대표가 null (#7)
	if ko := byID["SV4K-025"]; ko.Price != nil {
		t.Errorf("SV4K-025 price = %+v, want nil", ko.Price)
	}
	// 트레이너스는 종 이름·가격이 없다
	if tr := byID["sv03.5-166"]; tr.SpeciesNameKo != nil || tr.Price != nil {
		t.Errorf("sv03.5-166 = %+v", tr)
	}
}

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
