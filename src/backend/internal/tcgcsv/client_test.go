package tcgcsv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/httpx"
)

// newTestClient 는 testdata 파일을 경로별로 돌려주는 서버와 클라이언트를 만든다.
// 경로 /tcgplayer/3/23237/products → testdata/products_3_23237.json
func newTestClient(t *testing.T, override map[string]string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if body, ok := override[r.URL.Path]; ok {
			_, _ = w.Write([]byte(body))
			return
		}
		var file string
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		switch {
		case r.URL.Path == "/last-updated.txt":
			file = "last-updated.txt"
		case len(parts) == 3 && parts[2] == "groups":
			file = "groups_" + parts[1] + ".json"
		case len(parts) == 4:
			file = parts[3] + "_" + parts[1] + "_" + parts[2] + ".json"
		}
		b, err := os.ReadFile("testdata/" + file)
		if file == "" || err != nil {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return New(WithBaseURL(srv.URL), WithHTTP(httpx.New("tcgcsv", httpx.WithRetry(0, 0))))
}

func TestLastUpdated(t *testing.T) {
	c := newTestClient(t, nil)
	got, err := c.LastUpdated(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 11, 22, 20, 5, 55, 0, time.UTC)
	if !got.Equal(want) || got.Location() != time.UTC {
		t.Fatalf("got %v, want %v", got, want)
	}

	c = newTestClient(t, map[string]string{"/last-updated.txt": "yesterday"})
	if _, err := c.LastUpdated(context.Background()); err == nil {
		t.Fatal("형식 오류를 받아들임")
	}
}

func TestGroupsProductsPrices(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t, nil)

	groups, err := c.Groups(ctx, CategoryPokemon)
	if err != nil {
		t.Fatal(err)
	}
	codes := map[int]string{}
	for _, g := range groups {
		codes[g.GroupID] = g.Code()
	}
	if len(groups) != 3 || codes[24688] != "ME05" || codes[23237] != "SV" || codes[604] != "" {
		t.Fatalf("groups = %+v, codes = %v", groups, codes)
	}

	products, err := c.Products(ctx, CategoryPokemon, 23237)
	if err != nil {
		t.Fatal(err)
	}
	numbers := map[int]string{}
	for _, p := range products {
		numbers[p.ProductID] = p.Number()
	}
	if numbers[502552] != "001/165" || numbers[501999] != "" || len(products) != 5 {
		t.Fatalf("numbers = %v", numbers)
	}

	prices, err := c.Prices(ctx, CategoryPokemon, 23237)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, p := range prices {
		if p.ProductID == 502553 && p.SubTypeName == "Normal" {
			found = true
			if p.MarketPrice == nil || p.MarketPrice.String() != "0.28" || p.DirectLowPrice != nil {
				t.Fatalf("502553 Normal = %+v", p)
			}
		}
	}
	if !found || len(prices) != 7 {
		t.Fatalf("prices = %+v", prices)
	}
}

func TestUnsuccessfulResponse(t *testing.T) {
	c := newTestClient(t, map[string]string{
		"/tcgplayer/3/1/prices": `{"success":false,"errors":["group not found"],"results":[]}`,
	})
	_, err := c.Prices(context.Background(), CategoryPokemon, 1)
	if err == nil || !strings.Contains(err.Error(), "group not found") {
		t.Fatalf("err = %v", err)
	}
	if _, err := c.Prices(context.Background(), CategoryPokemon, 2); err == nil {
		t.Fatal("404 를 받아들임")
	}
}

func TestGroupCode(t *testing.T) {
	for name, want := range map[string]string{
		"SV2a: Pokemon Card 151":              "SV2a",
		"SWSH01: Sword & Shield Base Set":     "SWSH01",
		"Base Set (Shadowless)":               "",
		"Trick or Trade: BOOster Bundle 2024": "",
		"M-P: Promotional Cards":              "M-P",
	} {
		if got := (Group{Name: name}).Code(); got != want {
			t.Errorf("%q: got %q, want %q", name, got, want)
		}
	}
}
