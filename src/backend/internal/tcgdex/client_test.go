package tcgdex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func serveFile(t *testing.T, name string) http.HandlerFunc {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}
}

func newTestClient(srv *httptest.Server, opts ...Option) *Client {
	base := []Option{WithBaseURL(srv.URL), WithMinInterval(0), WithRetry(3, time.Millisecond)}
	return New(append(base, opts...)...)
}

func TestGetSetAndCard(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /en/sets/sv03.5", serveFile(t, "set_en_sv03.5.json"))
	mux.Handle("GET /en/cards/sv03.5-001", serveFile(t, "card_en_sv03.5-001.json"))
	mux.Handle("GET /ja/cards/SV4K-001", serveFile(t, "card_ja_SV4K-001.json"))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := newTestClient(srv)
	ctx := context.Background()

	s, err := c.GetSet(ctx, "en", "sv03.5")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "151" || s.CardCount.Total != 207 || s.CardCount.Official != 165 || len(s.Cards) != 2 {
		t.Fatalf("set = %+v", s)
	}
	if r := s.Released(); r == nil || r.Format(time.DateOnly) != "2023-09-22" {
		t.Fatalf("Released() = %v", r)
	}
	if s.Cards[0].ID != "sv03.5-001" || s.Cards[0].LocalID != "001" || s.Cards[0].Image == nil {
		t.Fatalf("card brief = %+v", s.Cards[0])
	}

	card, err := c.GetCard(ctx, "en", "sv03.5-001")
	if err != nil {
		t.Fatal(err)
	}
	if card.Rarity == nil || *card.Rarity != "Common" || len(card.DexID) != 1 || card.DexID[0] != 1 {
		t.Fatalf("card = %+v", card)
	}
	if v := card.Variants; v == nil || !v.Normal || !v.Reverse || v.Holo || v.FirstEdition {
		t.Fatalf("variants = %+v", card.Variants)
	}

	ja, err := c.GetCard(ctx, "ja", "SV4K-001")
	if err != nil {
		t.Fatal(err)
	}
	if ja.Name != "ヤナップ" || ja.Set.ID != "SV4K" || ja.DexID[0] != 511 {
		t.Fatalf("ja card = %+v", ja)
	}
}

// TCGdex 카드 ID 에는 "%3F" 가 문자 그대로 들어 있어, 경로에서는 "%253F" 로 보내야 한다.
func TestCardIDIsPathEscaped(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.EscapedPath()
		_, _ = w.Write([]byte(`{"id":"exu-%3F","localId":"%3F","name":"Unown"}`))
	}))
	defer srv.Close()
	card, err := newTestClient(srv).GetCard(context.Background(), "en", "exu-%3F")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/en/cards/exu-%253F" {
		t.Fatalf("path = %q", got)
	}
	if card.LocalID != "%3F" {
		t.Fatalf("localId = %q", card.LocalID)
	}
}

func TestNotFoundIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.NotFound(w, r)
	}))
	defer srv.Close()
	_, err := newTestClient(srv).GetSet(context.Background(), "ko", "SM1+")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("calls = %d, want 1", n)
	}
}

func TestRetryOnServerErrorThenSuccess(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.WriteHeader(http.StatusBadGateway)
		case 2:
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			_, _ = w.Write([]byte(`[{"id":"base1","name":"Base Set","cardCount":{"total":102,"official":102}}]`))
		}
	}))
	defer srv.Close()
	sets, err := newTestClient(srv).ListSets(context.Background(), "en")
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 1 || sets[0].ID != "base1" || calls.Load() != 3 {
		t.Fatalf("sets = %+v, calls = %d", sets, calls.Load())
	}
}

func TestRetryGivesUp(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	_, err := newTestClient(srv, WithRetry(2, time.Millisecond)).ListSets(context.Background(), "en")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if n := calls.Load(); n != 3 { // 첫 시도 + 재시도 2회
		t.Fatalf("calls = %d, want 3", n)
	}
}

func TestClientErrorIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	if _, err := newTestClient(srv).ListSets(context.Background(), "xx"); err == nil {
		t.Fatal("want error")
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("calls = %d, want 1", n)
	}
}

func TestUserAgentHeader(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	if _, err := newTestClient(srv).ListSets(context.Background(), "en"); err != nil {
		t.Fatal(err)
	}
	if ua != DefaultUserAgent {
		t.Fatalf("User-Agent = %q", ua)
	}
}

// 여러 고루틴이 동시에 호출해도 요청 시작 간격이 최소 간격 이상이어야 한다.
func TestMinIntervalSharedAcrossGoroutines(t *testing.T) {
	const interval = 40 * time.Millisecond
	var mu sync.Mutex
	var starts []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		starts = append(starts, time.Now())
		mu.Unlock()
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	c := newTestClient(srv, WithMinInterval(interval))

	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.ListSets(context.Background(), "en"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if len(starts) != 5 {
		t.Fatalf("requests = %d", len(starts))
	}
	// 서버가 받은 시각이므로 약간의 오차를 허용한다.
	if total := starts[len(starts)-1].Sub(starts[0]); total < 4*interval-10*time.Millisecond {
		t.Fatalf("5 requests took %v, want >= %v", total, 4*interval)
	}
}

func TestContextCancelStopsWaiting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	c := newTestClient(srv, WithRetry(5, time.Hour))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.ListSets(ctx, "en")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("did not stop on context cancel")
	}
}

func TestFractionalDexID(t *testing.T) {
	// TCGdex 는 일부 카드의 도감 번호를 384.1 처럼 소수로 준다 (ja PCG2-067 등).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"PCG2-067","localId":"067","name":"レックウザ","dexId":[384.1,384,0]}`))
	}))
	defer srv.Close()
	card, err := newTestClient(srv).GetCard(context.Background(), "ja", "PCG2-067")
	if err != nil {
		t.Fatal(err)
	}
	ids, adjusted := card.DexIDs()
	if len(ids) != 1 || ids[0] != 384 || !adjusted {
		t.Fatalf("DexIDs() = %v, %v", ids, adjusted)
	}
	if ids, adjusted := (&Card{DexID: []float64{25, 26}}).DexIDs(); len(ids) != 2 || adjusted {
		t.Fatalf("정수 도감 번호 DexIDs() = %v, %v", ids, adjusted)
	}
	if ids, _ := (&Card{}).DexIDs(); ids != nil {
		t.Fatalf("도감 번호 없음 DexIDs() = %v, want nil", ids)
	}
}

func TestTruncatedBodyIsRetried(t *testing.T) {
	// 본문을 읽다 연결이 끊기면(시간 초과·끊김) 전송 오류로 보고 재시도한다.
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Content-Length", "200")
			_, _ = w.Write([]byte(`{"id":"ex9-84","name":`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"ex9-84","localId":"84","name":"Pikachu"}`))
	}))
	defer srv.Close()
	card, err := newTestClient(srv).GetCard(context.Background(), "en", "ex9-84")
	if err != nil {
		t.Fatal(err)
	}
	if card.Name != "Pikachu" || calls.Load() != 2 {
		t.Fatalf("card = %+v, calls = %d", card, calls.Load())
	}
}

func TestInvalidJSONIsNotRetried(t *testing.T) {
	// 형식·타입 오류는 다시 받아도 같으므로 재시도하지 않는다.
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer srv.Close()
	_, err := newTestClient(srv).GetCard(context.Background(), "en", "x")
	if err == nil || calls.Load() != 1 {
		t.Fatalf("err = %v, calls = %d (want 오류, 1회)", err, calls.Load())
	}
}

func TestTCGPlayerPricing(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /en/cards/sv03.5-001", serveFile(t, "card_en_sv03.5-001_pricing.json"))
	mux.Handle("GET /ja/cards/SV4K-001", serveFile(t, "card_ja_SV4K-001.json"))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := newTestClient(srv)

	card, err := c.GetCard(context.Background(), "en", "sv03.5-001")
	if err != nil {
		t.Fatal(err)
	}
	p, err := card.TCGPlayer()
	if err != nil {
		t.Fatal(err)
	}
	if p == nil || p.Unit != "USD" || len(p.Variants) != 2 {
		t.Fatalf("pricing = %+v", p)
	}
	n, r := p.Variants["normal"], p.Variants["reverse-holofoil"]
	if n.ProductID != 502552 || r.ProductID != 502552 || n.Market == nil || *n.Market != 0.27 {
		t.Fatalf("normal=%+v reverse=%+v", n, r)
	}

	// pricing 이 null 인 카드
	card, err = c.GetCard(context.Background(), "ja", "SV4K-001")
	if err != nil {
		t.Fatal(err)
	}
	if p, err := card.TCGPlayer(); p != nil || err != nil {
		t.Fatalf("ja pricing = %+v, %v", p, err)
	}
	// tcgplayer 가 null 이거나 형식이 다른 경우
	for raw, wantErr := range map[string]bool{
		`{"tcgplayer":null,"cardmarket":{}}`:                      false,
		`{"tcgplayer":{"unit":"USD","normal":{"productId":"x"}}}`: true,
	} {
		card := Card{Pricing: []byte(raw)}
		p, err := card.TCGPlayer()
		if (err != nil) != wantErr || (!wantErr && p != nil) {
			t.Errorf("%s: p=%+v err=%v", raw, p, err)
		}
	}
}
