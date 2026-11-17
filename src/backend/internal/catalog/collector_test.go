package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbtest"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/tcgdex"
)

// fakeAPI 는 TCGdex 를 흉내 내는 테스트 서버이다. 테스트 중에 응답을 바꿀 수 있다.
type fakeAPI struct {
	mu          sync.Mutex
	sets        map[string][]tcgdex.SetBrief // lang → 목록
	setDetail   map[string]*tcgdex.Set       // "lang/id"
	cards       map[string]*tcgdex.Card      // "lang/id"
	failSet     map[string]bool              // "lang/id" → 500
	cardCalls   int
	setNotFound map[string]bool
}

func newFakeAPI() *fakeAPI {
	return &fakeAPI{
		sets: map[string][]tcgdex.SetBrief{}, setDetail: map[string]*tcgdex.Set{},
		cards: map[string]*tcgdex.Card{}, failSet: map[string]bool{}, setNotFound: map[string]bool{},
	}
}

func ptr[T any](v T) *T { return &v }

// addSet 은 세트와 카드들을 등록한다. cards 의 LocalID·Name·Image 로 세트 요약도 만든다.
func (f *fakeAPI) addSet(lang, id, name string, cards ...*tcgdex.Card) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := &tcgdex.Set{ID: id, Name: name, ReleaseDate: ptr("2023-09-22"), CardCount: tcgdex.CardCount{Total: len(cards), Official: len(cards)}}
	for _, c := range cards {
		c.Set.ID = id
		s.Cards = append(s.Cards, tcgdex.CardBrief{ID: c.ID, LocalID: c.LocalID, Name: c.Name, Image: c.Image})
		f.cards[lang+"/"+c.ID] = c
	}
	f.sets[lang] = append(f.sets[lang], tcgdex.SetBrief{ID: id, Name: name, CardCount: s.CardCount})
	f.setDetail[lang+"/"+id] = s
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/") // 경로는 디코딩된 값 (exu-%3F)
	var v any
	switch {
	case len(parts) == 2 && parts[1] == "sets":
		v = f.sets[parts[0]]
	case len(parts) == 3 && parts[1] == "sets":
		key := parts[0] + "/" + parts[2]
		if f.failSet[key] {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		s, ok := f.setDetail[key]
		if !ok || f.setNotFound[key] {
			http.NotFound(w, r)
			return
		}
		v = s
	case len(parts) == 3 && parts[1] == "cards":
		f.cardCalls++
		c, ok := f.cards[parts[0]+"/"+parts[2]]
		if !ok {
			http.NotFound(w, r)
			return
		}
		v = c
	default:
		http.NotFound(w, r)
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func (f *fakeAPI) resetCardCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := f.cardCalls
	f.cardCalls = 0
	return n
}

func card(id, localID, name, rarity string, dex []int32, v tcgdex.Variants) *tcgdex.Card {
	return &tcgdex.Card{
		ID: id, LocalID: localID, Name: name, Rarity: ptr(rarity), DexID: dex, Variants: &v,
		Image: ptr("https://assets.tcgdex.net/x/" + id), Category: "Pokemon",
	}
}

type env struct {
	ctx  context.Context
	api  *fakeAPI
	pool *pgxpool.Pool
	c    *Collector
}

func setup(t *testing.T) *env {
	t.Helper()
	pool := dbtest.Pool(t, "catalog")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	api := newFakeAPI()
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)
	client := tcgdex.New(tcgdex.WithBaseURL(srv.URL), tcgdex.WithMinInterval(0), tcgdex.WithRetry(1, time.Millisecond))
	return &env{ctx: ctx, api: api, pool: pool, c: New(pool, client)}
}

func (e *env) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(e.ctx, sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (e *env) run(t *testing.T, opt Options) Result {
	t.Helper()
	res, err := e.c.Run(e.ctx, opt)
	if err != nil {
		t.Fatalf("Run: %v (result %+v)", err, res)
	}
	return res
}

// 표준 데이터: 영어 2세트(신규 세트, 초판 있는 구세트 + 특수 문자 ID), 일본어 1세트.
func (e *env) seed() {
	e.api.addSet("en", "sv03.5", "151",
		card("sv03.5-001", "001", "Bulbasaur", "Common", []int32{1}, tcgdex.Variants{Normal: true, Reverse: true}),
		card("sv03.5-002", "002", "Ivysaur", "Common", []int32{2}, tcgdex.Variants{Normal: true, Reverse: true}),
	)
	e.api.addSet("en", "base2", "Jungle",
		card("base2-1", "1", "Clefable", "Rare", []int32{36}, tcgdex.Variants{Holo: true, FirstEdition: true}),
		card("exu-%3F", "%3F", "Unown", "Rare", []int32{201}, tcgdex.Variants{Holo: true}),
	)
	e.api.addSet("ja", "SV4K", "古代の咆哮",
		card("SV4K-001", "001", "ヤナップ", "Common", []int32{511}, tcgdex.Variants{Normal: true}),
	)
}

func TestRunLoadsCatalogAndIsIdempotent(t *testing.T) {
	e := setup(t)
	e.seed()

	res := e.run(t, Options{Languages: []string{"en", "ja"}})
	if res.Sets != 3 || res.CardsNew != 5 || res.Variants != 8 || len(res.FailedSets) != 0 {
		t.Fatalf("first run = %+v", res)
	}
	if n := e.api.resetCardCalls(); n != 5 {
		t.Fatalf("card detail calls = %d, want 5", n)
	}

	counts := func() [3]int {
		return [3]int{
			e.count(t, "SELECT count(*) FROM sets"),
			e.count(t, "SELECT count(*) FROM cards"),
			e.count(t, "SELECT count(*) FROM card_variants"),
		}
	}
	if got := counts(); got != [3]int{3, 5, 8} {
		t.Fatalf("rows = %v", got)
	}

	// 변형 매핑과 컬럼 값
	var variants []string
	rows, err := e.pool.Query(e.ctx, `SELECT v.variant FROM card_variants v JOIN cards c ON c.id = v.card_id WHERE c.tcgdex_id = 'base2-1' ORDER BY v.variant`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		variants = append(variants, s)
	}
	if !slices.Equal(variants, []string{"first_edition_holofoil", "unlimited_holofoil"}) {
		t.Fatalf("base2-1 variants = %v", variants)
	}
	if n := e.count(t, `SELECT count(*) FROM cards c JOIN sets s ON s.id = c.set_id
		WHERE c.tcgdex_id = 'SV4K-001' AND c.language = 'ja' AND s.language = 'ja' AND c.rarity = 'Common'
		  AND c.dex_ids = '{511}' AND c.local_id = '001' AND s.card_count = 1 AND s.release_date = '2023-09-22'`); n != 1 {
		t.Fatal("ja card columns not stored as expected")
	}
	if n := e.count(t, `SELECT count(*) FROM cards WHERE tcgdex_id = 'exu-%3F' AND local_id = '%3F'`); n != 1 {
		t.Fatal("card id with %3F not stored")
	}

	// 두 번째 실행: 카드 상세를 다시 받지 않고 행도 늘지 않는다.
	res = e.run(t, Options{Languages: []string{"en", "ja"}})
	if res.Sets != 3 || res.CardsNew != 0 || res.CardsBrief != 0 || res.Variants != 0 {
		t.Fatalf("second run = %+v", res)
	}
	if n := e.api.resetCardCalls(); n != 0 {
		t.Fatalf("second run card detail calls = %d, want 0", n)
	}
	if got := counts(); got != [3]int{3, 5, 8} {
		t.Fatalf("rows after second run = %v", got)
	}
}

func TestBriefUpdateKeepsDetailFields(t *testing.T) {
	e := setup(t)
	e.seed()
	e.run(t, Options{Languages: []string{"en"}, SetIDs: []string{"sv03.5"}})

	e.api.mu.Lock()
	e.api.setDetail["en/sv03.5"].Cards[0].Name = "Bulbasaur (renamed)"
	e.api.mu.Unlock()

	res := e.run(t, Options{Languages: []string{"en"}, SetIDs: []string{"sv03.5"}})
	if res.CardsBrief != 1 || res.CardsNew != 0 {
		t.Fatalf("result = %+v", res)
	}
	if n := e.count(t, `SELECT count(*) FROM cards WHERE tcgdex_id = 'sv03.5-001'
		AND name = 'Bulbasaur (renamed)' AND rarity = 'Common' AND dex_ids = '{1}'`); n != 1 {
		t.Fatal("brief update changed detail fields or did not rename")
	}
}

func TestFullRefreshUpdatesDetailFields(t *testing.T) {
	e := setup(t)
	e.seed()
	opt := Options{Languages: []string{"en"}, SetIDs: []string{"sv03.5"}}
	e.run(t, opt)
	e.api.resetCardCalls()

	e.api.mu.Lock()
	e.api.cards["en/sv03.5-001"].Rarity = ptr("Uncommon")
	e.api.cards["en/sv03.5-001"].Variants.Holo = true
	e.api.mu.Unlock()

	// 평소 실행은 카드 상세를 받지 않으므로 바뀌지 않는다.
	e.run(t, opt)
	if n := e.count(t, `SELECT count(*) FROM cards WHERE tcgdex_id = 'sv03.5-001' AND rarity = 'Common'`); n != 1 {
		t.Fatal("normal run should not refresh rarity")
	}

	opt.Full = true
	res := e.run(t, opt)
	if res.CardsDetailed != 2 || res.CardsNew != 0 || e.api.resetCardCalls() != 2 {
		t.Fatalf("full run = %+v", res)
	}
	if n := e.count(t, `SELECT count(*) FROM cards WHERE tcgdex_id = 'sv03.5-001' AND rarity = 'Uncommon'`); n != 1 {
		t.Fatal("full run did not refresh rarity")
	}
	if n := e.count(t, `SELECT count(*) FROM card_variants v JOIN cards c ON c.id = v.card_id WHERE c.tcgdex_id = 'sv03.5-001'`); n != 3 {
		t.Fatalf("variants after full run = %d, want 3", n)
	}
}

func TestFailedSetDoesNotBlockOthers(t *testing.T) {
	e := setup(t)
	e.seed()
	e.api.failSet["en/base2"] = true

	res := e.run(t, Options{Languages: []string{"en", "ja"}})
	if !slices.Equal(res.FailedSets, []string{"en/base2"}) || res.Sets != 2 {
		t.Fatalf("result = %+v", res)
	}
	if n := e.count(t, "SELECT count(*) FROM sets WHERE tcgdex_id = 'base2'"); n != 0 {
		t.Fatal("failed set should not be stored")
	}

	// 다음 실행에서 다시 시도되어 적재된다.
	delete(e.api.failSet, "en/base2")
	res = e.run(t, Options{Languages: []string{"en"}})
	if len(res.FailedSets) != 0 || res.CardsNew != 2 {
		t.Fatalf("retry run = %+v", res)
	}
}

func TestAllSetsFailedIsError(t *testing.T) {
	e := setup(t)
	e.seed()
	e.api.failSet["en/sv03.5"] = true
	e.api.failSet["en/base2"] = true
	res, err := e.c.Run(e.ctx, Options{Languages: []string{"en"}})
	if err == nil || len(res.FailedSets) != 2 {
		t.Fatalf("err = %v, result = %+v", err, res)
	}
}

func TestNotFoundSetAndCardAreSkipped(t *testing.T) {
	e := setup(t)
	e.seed()
	e.api.setNotFound["ja/SV4K"] = true
	e.api.mu.Lock()
	delete(e.api.cards, "en/sv03.5-002") // 세트 요약에는 있지만 카드 상세가 404
	e.api.mu.Unlock()

	res := e.run(t, Options{Languages: []string{"en", "ja"}})
	if res.SetsSkipped != 1 || res.CardsSkipped != 1 || res.CardsNew != 3 || len(res.FailedSets) != 0 {
		t.Fatalf("result = %+v", res)
	}
	if n := e.count(t, "SELECT count(*) FROM sets WHERE language = 'ja'"); n != 0 {
		t.Fatal("404 set should not be stored")
	}
}

func TestCardDetailErrorFailsSet(t *testing.T) {
	e := setup(t)
	e.seed()
	// 카드 상세가 500 이면 세트 전체를 실패로 처리한다 (일부 카드만 들어간 세트를 남기지 않는다).
	srvBroken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/cards/base2-1") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		e.api.ServeHTTP(w, r)
	}))
	defer srvBroken.Close()
	c := New(e.pool, tcgdex.New(tcgdex.WithBaseURL(srvBroken.URL), tcgdex.WithMinInterval(0), tcgdex.WithRetry(1, time.Millisecond)))

	res, err := c.Run(e.ctx, Options{Languages: []string{"en"}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.FailedSets, []string{"en/base2"}) {
		t.Fatalf("result = %+v", res)
	}
	if n := e.count(t, "SELECT count(*) FROM cards WHERE tcgdex_id IN ('base2-1', 'exu-%3F')"); n != 0 {
		t.Fatal("partial set stored")
	}
}

func TestCanceledContextStops(t *testing.T) {
	e := setup(t)
	e.seed()
	ctx, cancel := context.WithCancel(e.ctx)
	cancel()
	if _, err := e.c.Run(ctx, Options{Languages: []string{"en"}}); err == nil {
		t.Fatal("want error on canceled context")
	}
	if n := e.count(t, "SELECT count(*) FROM sets"); n != 0 {
		t.Fatal("nothing should be stored")
	}
}
