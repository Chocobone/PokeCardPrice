package db

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbq"
)

func date(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func decPtr(s string) *decimal.Decimal {
	d := dec(s)
	return &d
}

// migratedQueries 는 마이그레이션을 적용한 테스트 DB 의 쿼리 객체를 돌려준다.
func migratedQueries(t *testing.T) (context.Context, *dbq.Queries) {
	t.Helper()
	pool := testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return ctx, dbq.New(pool)
}

// newVariant 는 일본판 세트·카드·normal 변형을 만들고 변형 ID 를 돌려준다.
func newVariant(ctx context.Context, t *testing.T, q *dbq.Queries) (cardID, variantID int64) {
	t.Helper()
	setID, err := q.UpsertSet(ctx, dbq.UpsertSetParams{Language: "ja", TcgdexID: "SV5K", Name: "ワイルドフォース"})
	if err != nil {
		t.Fatal(err)
	}
	cardID, err = q.UpsertCard(ctx, dbq.UpsertCardParams{SetID: setID, Language: "ja", TcgdexID: "SV5K-001", LocalID: "001", Name: "ナゾノクサ", DexIds: []int32{43}})
	if err != nil {
		t.Fatal(err)
	}
	variantID, err = q.UpsertCardVariant(ctx, dbq.UpsertCardVariantParams{CardID: cardID, Variant: "normal"})
	if err != nil {
		t.Fatal(err)
	}
	return cardID, variantID
}

func TestCatalogUpsertIsIdempotent(t *testing.T) {
	ctx, q := migratedQueries(t)

	cardID, variantID := newVariant(ctx, t, q)
	cardID2, variantID2 := newVariant(ctx, t, q)
	if cardID != cardID2 || variantID != variantID2 {
		t.Fatalf("다시 저장하면 같은 ID 여야 함: card %d→%d, variant %d→%d", cardID, cardID2, variantID, variantID2)
	}

	// 같은 키로 이름을 바꾸면 갱신된다.
	setID, err := q.UpsertSet(ctx, dbq.UpsertSetParams{Language: "ja", TcgdexID: "SV5K", Name: "ワイルドフォース"})
	if err != nil {
		t.Fatal(err)
	}
	rarity := "Common"
	if _, err := q.UpsertCard(ctx, dbq.UpsertCardParams{SetID: setID, Language: "ja", TcgdexID: "SV5K-001", LocalID: "001", Name: "ナゾノクサ改", Rarity: &rarity}); err != nil {
		t.Fatal(err)
	}
	card, err := q.GetCard(ctx, cardID)
	if err != nil {
		t.Fatal(err)
	}
	if card.Name != "ナゾノクサ改" || card.Rarity == nil || *card.Rarity != "Common" {
		t.Errorf("card = %+v", card)
	}
	if card.ImageUrl != nil {
		t.Errorf("ImageUrl = %v, want nil (NULL)", *card.ImageUrl)
	}

	if _, err := q.UpsertCardVariant(ctx, dbq.UpsertCardVariantParams{CardID: cardID, Variant: "reverse_holofoil"}); err != nil {
		t.Fatal(err)
	}
	vs, err := q.ListCardVariants(ctx, cardID)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 2 || vs[0].Variant != "normal" || vs[1].Variant != "reverse_holofoil" {
		t.Errorf("variants = %+v", vs)
	}
}

func TestPriceHistoryUpsertAndRange(t *testing.T) {
	ctx, q := migratedQueries(t)
	_, variantID := newVariant(ctx, t, q)
	src, err := q.GetPriceSourceByCode(ctx, "tcgcsv")
	if err != nil {
		t.Fatal(err)
	}
	if src.Currency != "USD" || src.PriceKind != "sale" {
		t.Errorf("source = %+v", src)
	}

	if err := q.UpsertExternalID(ctx, dbq.UpsertExternalIDParams{SourceID: src.ID, ExternalID: "502552", ExternalSubID: "Normal", CardVariantID: variantID}); err != nil {
		t.Fatal(err)
	}
	found, err := q.FindVariantByExternalID(ctx, dbq.FindVariantByExternalIDParams{SourceID: src.ID, ExternalID: "502552", ExternalSubID: "Normal"})
	if err != nil || found != variantID {
		t.Fatalf("FindVariantByExternalID = %d, %v; want %d", found, err, variantID)
	}

	upsert := func(rows []dbq.UpsertPriceHistoryParams) {
		t.Helper()
		br := q.UpsertPriceHistory(ctx, rows)
		br.Exec(func(i int, err error) {
			if err != nil {
				t.Errorf("row %d: %v", i, err)
			}
		})
	}
	row := func(day, price string) dbq.UpsertPriceHistoryParams {
		return dbq.UpsertPriceHistoryParams{
			CardVariantID: variantID, SourceID: src.ID, PriceDate: date(day),
			Price: dec(price), PriceBasis: "market", Market: decPtr(price), Low: decPtr("0.10"),
		}
	}

	upsert([]dbq.UpsertPriceHistoryParams{row("2026-11-04", "0.20"), row("2026-11-05", "0.25"), row("2026-10-01", "0.30")})
	// 같은 날짜를 다시 저장하면 행이 늘지 않고 값만 바뀐다.
	upsert([]dbq.UpsertPriceHistoryParams{row("2026-11-05", "0.27")})

	got, err := q.ListPriceHistory(ctx, dbq.ListPriceHistoryParams{CardVariantID: variantID, SourceID: src.ID, FromDate: date("2026-11-01")})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("rows = %d, want 2 (%+v)", len(got), got)
	}
	if !got[0].PriceDate.Equal(date("2026-11-04")) || !got[1].PriceDate.Equal(date("2026-11-05")) {
		t.Errorf("날짜순이 아님: %v, %v", got[0].PriceDate, got[1].PriceDate)
	}
	if got[0].Price.String() != "0.2" || !got[1].Price.Equal(dec("0.27")) {
		t.Errorf("prices = %s, %s; want 0.20, 0.27", got[0].Price, got[1].Price)
	}
}

func TestFxRateOnOrBefore(t *testing.T) {
	ctx, q := migratedQueries(t)
	for _, r := range []struct{ day, rate string }{{"2026-10-29", "1410.5"}, {"2026-10-30", "1418.41702250"}} {
		if err := q.UpsertFxRate(ctx, dbq.UpsertFxRateParams{BaseCurrency: "USD", QuoteCurrency: "KRW", RateDate: date(r.day), Provider: "frankfurter", Rate: dec(r.rate)}); err != nil {
			t.Fatal(err)
		}
	}
	// 2026-11-01(일)은 값이 없으므로 직전 영업일 10/30 값을 쓴다.
	got, err := q.GetFxRateOnOrBefore(ctx, dbq.GetFxRateOnOrBeforeParams{BaseCurrency: "USD", QuoteCurrency: "KRW", Provider: "frankfurter", OnDate: date("2026-11-01")})
	if err != nil {
		t.Fatal(err)
	}
	if !got.RateDate.Equal(date("2026-10-30")) || !got.Rate.Equal(dec("1418.417022500")) {
		t.Errorf("got %v %s, want 2026-10-30 1418.41702250", got.RateDate, got.Rate)
	}
	// 소수점 8자리까지 정확히 보존되고, decimal 로 원화 환산을 계산할 수 있다.
	if krw := dec("0.27").Mul(got.Rate).Round(0); !krw.Equal(dec("383")) {
		t.Errorf("0.27 USD → %s KRW, want 383", krw)
	}
}

func TestUserCollectionAndTrade(t *testing.T) {
	ctx, q := migratedQueries(t)
	_, variantID := newVariant(ctx, t, q)

	u, err := q.CreateUser(ctx, dbq.CreateUserParams{Email: "Trainer@Example.com", PasswordHash: "hash", DisplayName: "trainer"})
	if err != nil {
		t.Fatal(err)
	}
	if u.ID.Version() != 7 || u.Role != "user" {
		t.Errorf("user = %+v", u)
	}
	byEmail, err := q.GetUserByEmail(ctx, "trainer@example.COM")
	if err != nil || byEmail.ID != u.ID {
		t.Fatalf("GetUserByEmail = %v, %v", byEmail.ID, err)
	}

	item, err := q.UpsertCollectionItem(ctx, dbq.UpsertCollectionItemParams{UserID: u.ID, CardVariantID: variantID, Condition: "near_mint", Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	item2, err := q.UpsertCollectionItem(ctx, dbq.UpsertCollectionItemParams{UserID: u.ID, CardVariantID: variantID, Condition: "near_mint", Quantity: 3})
	if err != nil {
		t.Fatal(err)
	}
	if item2.ID != item.ID || item2.Quantity != 3 {
		t.Errorf("수량 갱신: id %d→%d, quantity %d", item.ID, item2.ID, item2.Quantity)
	}
	items, err := q.ListCollectionItems(ctx, u.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("items = %d, %v", len(items), err)
	}

	tr, err := q.CreateUserTrade(ctx, dbq.CreateUserTradeParams{UserID: u.ID, CardVariantID: variantID, PriceKrw: dec("15000"), TradeDate: date("2026-11-05"), Condition: "lightly_played"})
	if err != nil {
		t.Fatal(err)
	}
	if tr.VerificationStatus != "pending" || !tr.PriceKrw.Equal(dec("15000")) {
		t.Errorf("trade = %+v", tr)
	}
}
