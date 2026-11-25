package server

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/api"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbq"
)

const defaultPeriod = "30d"

// parseCardID 는 경로의 카드 ID(양의 정수)이다.
func parseCardID(c *echo.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, invalidParam("카드 ID 는 양의 정수여야 합니다")
	}
	return id, nil
}

// getCard 는 카드 상세 행이다. 없으면 404 이다.
func (h *cardHandlers) getCard(ctx context.Context, id int64) (dbq.GetCardDetailRow, error) {
	card, err := h.q.GetCardDetail(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return card, notFound("카드를 찾을 수 없습니다")
	}
	if err != nil {
		return card, fmt.Errorf("get card %d: %w", id, err)
	}
	return card, nil
}

// detail 은 GET /cards/{id} 카드 상세이다.
func (h *cardHandlers) detail(c *echo.Context) error {
	id, err := parseCardID(c)
	if err != nil {
		return err
	}
	ctx := c.Request().Context()
	card, err := h.getCard(ctx, id)
	if err != nil {
		return err
	}
	variants, err := h.q.ListCardVariants(ctx, id)
	if err != nil {
		return fmt.Errorf("list card variants: %w", err)
	}
	prices, err := h.latestPrices(ctx, []int64{id})
	if err != nil {
		return err
	}
	linkedRows, err := h.q.ListLinkedCards(ctx, id)
	if err != nil {
		return fmt.Errorf("list linked cards: %w", err)
	}
	linkedIn := make([]summaryInput, len(linkedRows))
	for i, r := range linkedRows {
		linkedIn[i] = summaryInput{
			ID: r.ID, Language: r.Language, TcgdexID: r.TcgdexID, Name: r.Name, LocalID: r.LocalID,
			SetTcgdexID: r.SetTcgdexID, SetName: r.SetName, ImageURL: r.ImageUrl,
			HasKoEdition: r.HasKoEdition, SpeciesNameKo: r.SpeciesNameKo,
		}
	}
	linked, err := h.summaries(ctx, linkedIn)
	if err != nil {
		return err
	}

	cardPrices := prices[id]
	byVariant := make(map[int64][]api.LatestPrice)
	for _, p := range cardPrices {
		byVariant[p.VariantID] = append(byVariant[p.VariantID], p)
	}
	vps := make([]api.VariantPrices, len(variants))
	for i, v := range variants {
		ps := byVariant[v.ID]
		api.SortBySource(ps)
		out := make([]api.Price, len(ps))
		for j, p := range ps {
			out[j] = api.NewPrice(p)
		}
		vps[i] = api.VariantPrices{Variant: v.Variant, Prices: out}
	}

	var releaseDate *string
	if card.SetReleaseDate != nil {
		s := api.FormatDate(*card.SetReleaseDate)
		releaseDate = &s
	}
	dexIDs := card.DexIds
	if dexIDs == nil {
		dexIDs = []int32{}
	}
	return c.JSON(http.StatusOK, api.CardDetail{
		ID: card.ID, Language: card.Language, TcgdexID: card.TcgdexID, Name: card.Name, LocalID: card.LocalID,
		Set: api.SetDetail{
			TcgdexID: card.SetTcgdexID, Name: card.SetName, ReleaseDate: releaseDate,
			LogoURL: card.SetLogoUrl, CardCount: card.SetCardCount,
		},
		ImageURL: card.ImageUrl, HasKoEdition: card.HasKoEdition, SpeciesNameKo: card.SpeciesNameKo,
		Price:  api.PickRepresentative(card.Language, cardPrices),
		Rarity: card.Rarity, DexIDs: dexIDs, Linked: linked, Variants: vps,
	})
}

// prices 는 GET /cards/{id}/prices 시세 추이이다.
func (h *cardHandlers) prices(c *echo.Context) error {
	id, err := parseCardID(c)
	if err != nil {
		return err
	}
	period := c.QueryParam("period")
	if period == "" {
		period = defaultPeriod
	}
	days, ok := api.PeriodDays[period]
	if !ok {
		return invalidParam("period 는 7d, 30d, 90d, 1y 중 하나여야 합니다")
	}
	ctx := c.Request().Context()
	if _, err := h.getCard(ctx, id); err != nil {
		return err
	}
	from := api.PeriodStart(h.now(), days)
	rows, err := h.q.ListCardPriceHistory(ctx, dbq.ListCardPriceHistoryParams{CardID: id, FromDate: from, FxProvider: api.FxProvider})
	if err != nil {
		return fmt.Errorf("list card price history: %w", err)
	}

	// 행은 변형 id → 소스 id → 날짜 순이다. 변형·소스가 바뀔 때마다 계열을 끊는다.
	type key struct {
		variantID int64
		source    string
	}
	var groups [][]api.LatestPrice
	var last key
	for i, r := range rows {
		k := key{r.CardVariantID, r.Source}
		if i == 0 || k != last {
			groups = append(groups, nil)
			last = k
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], api.LatestPrice{
			VariantID: r.CardVariantID, Variant: r.Variant, Source: r.Source, Currency: r.Currency,
			Date: r.PriceDate, Amount: r.Price, UpdatedAt: r.SourceUpdatedAt, FxDate: r.FxRateDate, FxRate: r.FxRate,
		})
	}
	// 같은 변형 안에서는 소스 우선순위 순으로 둔다 (SQL 은 소스 id 순이다). 변형 순서(id 순)는 유지한다.
	slices.SortStableFunc(groups, func(a, b []api.LatestPrice) int {
		if c := cmp.Compare(a[0].VariantID, b[0].VariantID); c != 0 {
			return c
		}
		return api.SourceRank(a[0].Source) - api.SourceRank(b[0].Source)
	})
	series := make([]api.PriceSeries, len(groups))
	for i, g := range groups {
		points := make([]api.PricePoint, len(g))
		for j, p := range g {
			points[j] = api.PricePoint{Date: api.FormatDate(p.Date), Amount: api.FormatAmount(p.Amount), KRW: api.ToKRW(p.Amount, p.Currency, p.FxRate)}
		}
		series[i] = api.PriceSeries{Variant: g[0].Variant, Source: g[0].Source, Currency: g[0].Currency, Points: points, Summary: api.Summarize(g)}
	}
	return c.JSON(http.StatusOK, api.PriceHistoryResponse{CardID: id, Period: period, From: api.FormatDate(from), Series: series})
}
