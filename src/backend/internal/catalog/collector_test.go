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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbq"
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

// 한글판 연결용 데이터: ja SV5K 5장(004 는 트레이너스, 005 는 시크릿), ko SV5K 3장(카드 수 4), ko 세트가 없는 ja SV6K.
func (e *env) seedKo() {
	nv := tcgdex.Variants{Normal: true}
	e.api.addSet("ja", "SV5K", "ワイルドフォース",
		card("SV5K-001", "001", "フシギダネ", "C", []int32{1}, nv),
		card("SV5K-002", "002", "フシギソウ", "C", []int32{2}, nv),
		card("SV5K-003", "003", "フシギバナ", "R", []int32{3}, nv),
		card("SV5K-004", "004", "ネモ", "U", nil, nv),
		card("SV5K-005", "005", "フシギダネ", "AR", []int32{1}, nv),
	)
	e.api.addSet("ja", "SV6K", "変幻の仮面",
		card("SV6K-001", "001", "ピカチュウ", "C", []int32{25}, nv),
	)
	e.api.addSet("ko", "SV5K", "와일드포스",
		card("SV5K-001", "001", "이상해씨", "C", []int32{1}, nv),
		card("SV5K-002", "002", "이상해풀", "C", []int32{99}, nv), // 도감 번호가 달라 연결하지 않는다
		card("SV5K-004", "004", "네모", "U", nil, nv),           // 둘 다 도감 번호가 없으면 같다고 본다
	)
	e.api.setDetail["ko/SV5K"].CardCount.Total = 4
}

func (e *env) linkedTo(t *testing.T, koTcgdexID string) (string, string) {
	t.Helper()
	var ja, method string
	err := e.pool.QueryRow(e.ctx, `SELECT j.tcgdex_id, l.method FROM card_links l
		JOIN cards k ON k.id = l.card_id JOIN cards j ON j.id = l.linked_card_id
		WHERE k.language = 'ko' AND k.tcgdex_id = $1`, koTcgdexID).Scan(&ja, &method)
	if err != nil {
		return "", ""
	}
	return ja, method
}

func (e *env) hasKo(t *testing.T) []string {
	t.Helper()
	rows, err := e.pool.Query(e.ctx, `SELECT tcgdex_id FROM cards WHERE has_ko_edition ORDER BY tcgdex_id`)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

func TestKoLinksAndHasKoEdition(t *testing.T) {
	e := setup(t)
	e.seedKo()

	res := e.run(t, Options{})
	want := LinkResult{Changed: 2, KoCards: 3, Auto: 2, Unlinked: 1, HasKoEdition: 4, HasKoChanged: 4}
	if res.Links != want {
		t.Fatalf("첫 실행 Links = %+v, want %+v", res.Links, want)
	}
	if ja, m := e.linkedTo(t, "SV5K-001"); ja != "SV5K-001" || m != "auto" {
		t.Errorf("SV5K-001 → %q (%s)", ja, m)
	}
	if ja, _ := e.linkedTo(t, "SV5K-002"); ja != "" {
		t.Errorf("도감 번호가 다른 SV5K-002 가 %q 에 연결됨", ja)
	}
	if ja, m := e.linkedTo(t, "SV5K-004"); ja != "SV5K-004" || m != "auto" {
		t.Errorf("SV5K-004 → %q (%s)", ja, m)
	}
	// ja SV5K 001~004 만 참 (005 는 ko 카드 수 4 초과, SV6K 는 ko 세트 없음).
	if got := e.hasKo(t); !slices.Equal(got, []string{"SV5K-001", "SV5K-002", "SV5K-003", "SV5K-004"}) {
		t.Errorf("has_ko_edition = %v", got)
	}

	// 재실행: 바뀌는 것이 없다.
	res = e.run(t, Options{})
	want.Changed, want.HasKoChanged = 0, 0
	if res.Links != want {
		t.Fatalf("재실행 Links = %+v, want %+v", res.Links, want)
	}

	// ko 세트 카드 수가 줄면 범위를 벗어난 카드는 거짓으로 되돌린다.
	e.api.setDetail["ko/SV5K"].CardCount.Total = 2
	res = e.run(t, Options{})
	if res.Links.HasKoChanged != 2 || res.Links.HasKoEdition != 2 {
		t.Errorf("카드 수 2 로 변경 후 Links = %+v", res.Links)
	}
	if got := e.hasKo(t); !slices.Equal(got, []string{"SV5K-001", "SV5K-002"}) {
		t.Errorf("has_ko_edition = %v", got)
	}
}

func TestManualLinkIsKept(t *testing.T) {
	e := setup(t)
	e.seedKo()
	e.run(t, Options{})

	// 작업지시자가 SV5K-001(ko) 을 시크릿 SV5K-005(ja) 로, SV5K-002(ko) 를 SV5K-003(ja) 으로 수동 연결했다.
	_, err := e.pool.Exec(e.ctx, `
		INSERT INTO card_links (card_id, linked_card_id, method)
		SELECT k.id, j.id, 'manual' FROM cards k, cards j
		WHERE k.language = 'ko' AND j.language = 'ja'
		  AND ((k.tcgdex_id = 'SV5K-001' AND j.tcgdex_id = 'SV5K-005') OR (k.tcgdex_id = 'SV5K-002' AND j.tcgdex_id = 'SV5K-003'))
		ON CONFLICT (card_id) DO UPDATE SET linked_card_id = EXCLUDED.linked_card_id, method = 'manual'`)
	if err != nil {
		t.Fatal(err)
	}

	res := e.run(t, Options{})
	if res.Links.Changed != 0 || res.Links.Removed != 0 || res.Links.Manual != 2 || res.Links.Auto != 1 || res.Links.Unlinked != 0 {
		t.Errorf("Links = %+v", res.Links)
	}
	for ko, wantJa := range map[string]string{"SV5K-001": "SV5K-005", "SV5K-002": "SV5K-003"} {
		if ja, m := e.linkedTo(t, ko); ja != wantJa || m != "manual" {
			t.Errorf("%s → %q (%s), want %s (manual)", ko, ja, m, wantJa)
		}
	}
}

func TestStaleAutoLinkIsRemoved(t *testing.T) {
	e := setup(t)
	e.seedKo()
	e.run(t, Options{})

	// 일본판 카드의 도감 번호가 바뀌면(데이터 수정) 규칙에 맞지 않아 연결을 지운다.
	e.api.cards["ja/SV5K-001"].DexID = []int32{4}
	res := e.run(t, Options{Languages: []string{"ja"}, Full: true})
	if res.Links.Removed != 1 || res.Links.Auto != 1 || res.Links.Unlinked != 2 {
		t.Errorf("Links = %+v", res.Links)
	}
	if ja, _ := e.linkedTo(t, "SV5K-001"); ja != "" {
		t.Errorf("SV5K-001 이 아직 %q 에 연결됨", ja)
	}
}

func TestSpeciesLoadedAndSearchable(t *testing.T) {
	e := setup(t)
	e.seedKo()
	res := e.run(t, Options{})
	if res.SpeciesChanged != 1025 {
		t.Errorf("첫 실행 SpeciesChanged = %d, want 1025", res.SpeciesChanged)
	}
	if n := e.count(t, `SELECT count(*) FROM pokemon_species`); n != 1025 {
		t.Errorf("pokemon_species = %d", n)
	}
	if res := e.run(t, Options{}); res.SpeciesChanged != 0 {
		t.Errorf("재실행 SpeciesChanged = %d, want 0", res.SpeciesChanged)
	}

	// 한글 종 이름으로 일본판·한글판 카드를 찾는다.
	cards, err := dbq.New(e.pool).ListCardsByKoSpeciesName(e.ctx, ptr("이상해씨"))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range cards {
		got = append(got, c.Language+"/"+c.TcgdexID)
	}
	if want := []string{"ja/SV5K-001", "ja/SV5K-005", "ko/SV5K-001"}; !slices.Equal(got, want) {
		t.Errorf("이상해씨 카드 = %v, want %v", got, want)
	}
}
