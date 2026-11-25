package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/api"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbq"
)

const (
	defaultMoversLimit = 10
	maxMoversLimit     = 50
)

// farFuture 는 "가장 최근 날짜"를 찾을 때 쓰는 상한이다.
var farFuture = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

// movers 는 GET /rankings/movers 급등·급락 순위이다 (#8 Q3).
func (h *cardHandlers) movers(c *echo.Context) error {
	period := c.QueryParam("period")
	days, ok := api.MoverPeriodDays[period]
	if !ok {
		return invalidParam("period 는 7d, 30d 중 하나여야 합니다")
	}
	direction := c.QueryParam("direction")
	if direction != "up" && direction != "down" {
		return invalidParam("direction 은 up, down 중 하나여야 합니다")
	}
	limit, err := parseIntParam(c, "limit", defaultMoversLimit, 1, maxMoversLimit)
	if err != nil {
		return err
	}
	resp := api.MoversResponse{Period: period, Direction: direction, Items: []api.Mover{}}

	ctx := c.Request().Context()
	src, err := h.q.GetPriceSourceByCode(ctx, api.MoverSource)
	if err != nil {
		return fmt.Errorf("get price source %s: %w", api.MoverSource, err)
	}
	// 기준일: 소스의 최신 가격 기준일. 없으면 시세 데이터가 없는 것이다.
	asOf, ok, err := h.latestPriceDate(ctx, src.ID, farFuture)
	if err != nil || !ok {
		return h.moversDone(c, resp, err)
	}
	s := api.FormatDate(asOf)
	resp.AsOf = &s
	// 비교일: 기준일 − 기간 이하 가장 가까운 가격 기준일 (순위 전체에 하나, 1단계 S2).
	compareOn, ok, err := h.latestPriceDate(ctx, src.ID, asOf.AddDate(0, 0, -days))
	if err != nil || !ok {
		return h.moversDone(c, resp, err)
	}

	rows, err := h.q.ListMovers(ctx, dbq.ListMoversParams{
		SourceID: src.ID, AsOf: asOf, CompareOn: compareOn, Up: direction == "up",
		MinPrice: api.MoverMinPrice, Lim: int32(limit),
	})
	if err != nil {
		return fmt.Errorf("list movers: %w", err)
	}
	if len(rows) == 0 {
		return c.JSON(http.StatusOK, resp)
	}
	curFx, err := h.fxRate(ctx, src.Currency, asOf)
	if err != nil {
		return err
	}
	prevFx, err := h.fxRate(ctx, src.Currency, compareOn)
	if err != nil {
		return err
	}

	cards := make([]summaryInput, len(rows))
	for i, r := range rows {
		cards[i] = summaryInput{
			ID: r.CardID, Language: r.Language, TcgdexID: r.TcgdexID, Name: r.Name, LocalID: r.LocalID,
			SetTcgdexID: r.SetTcgdexID, SetName: r.SetName, ImageURL: r.ImageUrl,
			HasKoEdition: r.HasKoEdition, SpeciesNameKo: r.SpeciesNameKo,
		}
	}
	summaries, err := h.summaries(ctx, cards)
	if err != nil {
		return err
	}
	resp.Items = make([]api.Mover, len(rows))
	for i, r := range rows {
		cur := api.LatestPrice{Source: src.Code, Currency: src.Currency, Date: asOf, Amount: r.CurPrice, UpdatedAt: r.CurUpdatedAt}
		curFx.apply(&cur)
		prev := api.LatestPrice{Source: src.Code, Currency: src.Currency, Date: compareOn, Amount: r.PrevPrice, UpdatedAt: r.PrevUpdatedAt}
		prevFx.apply(&prev)
		resp.Items[i] = api.Mover{
			Card: summaries[i], Variant: r.Variant,
			Current: api.NewPrice(cur), Previous: api.NewPrice(prev),
			ChangeRate: api.ChangeRate(r.PrevPrice, r.CurPrice),
		}
	}
	return c.JSON(http.StatusOK, resp)
}

// moversDone 은 순위를 낼 수 없을 때(데이터 없음) 빈 순위를, 오류면 오류를 돌려준다.
func (h *cardHandlers) moversDone(c *echo.Context, resp api.MoversResponse, err error) error {
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// latestPriceDate 는 소스의 on_or_before 이하 가장 늦은 가격 기준일이다. 없으면 ok 가 false 이다.
func (h *cardHandlers) latestPriceDate(ctx context.Context, sourceID int16, onOrBefore time.Time) (time.Time, bool, error) {
	d, err := h.q.GetLatestPriceDate(ctx, dbq.GetLatestPriceDateParams{SourceID: sourceID, OnOrBefore: onOrBefore})
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("get latest price date: %w", err)
	}
	return d, true, nil
}

// fx 는 원화 환산용 환율이다. 없으면 rate 가 nil 이다.
type fx struct {
	date *time.Time
	rate *decimal.Decimal
}

func (f fx) apply(p *api.LatestPrice) { p.FxDate, p.FxRate = f.date, f.rate }

// fxRate 는 on 이전 가장 가까운 원화 환율이다. 원화 소스는 환율이 필요 없다.
func (h *cardHandlers) fxRate(ctx context.Context, currency string, on time.Time) (fx, error) {
	if currency == "KRW" {
		return fx{}, nil
	}
	r, err := h.q.GetFxRateOnOrBefore(ctx, dbq.GetFxRateOnOrBeforeParams{
		BaseCurrency: currency, QuoteCurrency: "KRW", Provider: api.FxProvider, OnDate: on,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return fx{}, nil
	}
	if err != nil {
		return fx{}, fmt.Errorf("get fx rate %s %s: %w", currency, api.FormatDate(on), err)
	}
	return fx{date: &r.RateDate, rate: &r.Rate}, nil
}
