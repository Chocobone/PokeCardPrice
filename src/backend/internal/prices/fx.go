package prices

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbq"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/fx"
)

// FXCode 는 환율 수집 작업 이름이다 (price_sources 의 코드가 아님).
const FXCode = "fx"

// fxPairs 는 수집할 통화쌍이다. TCGplayer(USD)·Cardmarket(EUR) 가격을 원화로 환산하는 데 쓴다.
var fxPairs = []struct{ Base, Quote string }{
	{"USD", "KRW"},
	{"EUR", "KRW"},
}

// FX 는 Frankfurter 최신 환율을 fx_rates 에 저장한다.
// 응답의 date 를 기준일로 저장하고, 값이 없으면 저장하지 않는다 (조회 시 GetFxRateOnOrBefore 가 직전 값을 쓴다).
type FX struct {
	pool   *pgxpool.Pool
	client *fx.Client
}

// NewFX 는 환율 수집 작업을 만든다.
func NewFX(pool *pgxpool.Pool, client *fx.Client) *FX { return &FX{pool: pool, client: client} }

// Code 는 작업 이름이다.
func (f *FX) Code() string { return FXCode }

// Collect 는 통화쌍마다 최신 환율을 받아 저장한다. 한 쌍이 실패해도 나머지는 저장한다.
func (f *FX) Collect(ctx context.Context, _ Options) (Result, error) {
	var res Result
	var errs []error
	q := dbq.New(f.pool)
	for _, p := range fxPairs {
		name := p.Base + "/" + p.Quote
		rates, err := f.client.Rates(ctx, p.Base, []string{p.Quote}, time.Time{})
		if err != nil {
			res.Failed = append(res.Failed, name)
			errs = append(errs, err)
			continue
		}
		res.Fetched += len(rates)
		if len(rates) == 0 {
			slog.Warn("fx rate missing", "pair", name)
			res.Skipped++
			continue
		}
		for _, r := range rates {
			if r.Base != p.Base || r.Quote != p.Quote {
				// 요청과 다른 쌍은 저장하지 않는다 (fx_rates CHECK 에 없는 통화일 수 있음).
				slog.Warn("fx unexpected pair", "want", name, "got", r.Base+"/"+r.Quote)
				res.Skipped++
				continue
			}
			if err := q.UpsertFxRate(ctx, dbq.UpsertFxRateParams{
				BaseCurrency: r.Base, QuoteCurrency: r.Quote, RateDate: r.Date, Provider: fx.Provider, Rate: r.Rate,
			}); err != nil {
				res.Failed = append(res.Failed, name)
				errs = append(errs, fmt.Errorf("%s 저장: %w", name, err))
				continue
			}
			res.Saved++
			slog.Info("fx rate saved", "pair", name, "date", r.Date.Format(time.DateOnly), "rate", r.Rate.String())
		}
	}
	return res, errors.Join(errs...)
}
