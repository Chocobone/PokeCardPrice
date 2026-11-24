package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/fx"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/prices"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/tcgcsv"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/tcgdex"
)

// pricesSourceNone 은 시세 수집을 끄는 PRICES_SOURCES 값이다 (카탈로그만 돌릴 때).
const pricesSourceNone = "none"

// pricesConfig 는 해외 시세 수집(#6) 설정이다. 환경 변수로 바꾼다 (모두 개발용, 비우면 전체).
//
//   - PRICES_SOURCES: 실행할 작업 (쉼표 구분: fx, tcgcsv, tcgdex_cardmarket). none 이면 시세 수집을 하지 않는다
//   - PRICES_TCGCSV_GROUPS: TCGCSV 그룹 ID (쉼표 구분, 예: 23237). 영어판 그룹은 이름 코드가 세트 ID 와 달라 따로 지정한다
//   - 언어·세트 범위는 카탈로그와 같은 CATALOG_LANGUAGES·CATALOG_SETS 를 쓴다
type pricesConfig struct {
	opt      prices.Options
	disabled bool
}

var priceSourceCodes = []string{prices.FXCode, prices.TCGCSVCode, prices.CardmarketCode}

func pricesConfigFromEnv(cat catalogConfig) (pricesConfig, error) {
	var cfg pricesConfig
	srcs := splitList(os.Getenv("PRICES_SOURCES"))
	if slices.Equal(srcs, []string{pricesSourceNone}) {
		cfg.disabled = true
		return cfg, nil
	}
	for _, s := range srcs {
		if !slices.Contains(priceSourceCodes, s) {
			return cfg, fmt.Errorf("PRICES_SOURCES: 알 수 없는 작업 %q (%v 또는 %s)", s, priceSourceCodes, pricesSourceNone)
		}
	}
	cfg.opt.Sources = srcs
	for _, g := range splitList(os.Getenv("PRICES_TCGCSV_GROUPS")) {
		n, err := strconv.Atoi(g)
		if err != nil || n <= 0 {
			return cfg, fmt.Errorf("PRICES_TCGCSV_GROUPS: 그룹 ID %q", g)
		}
		cfg.opt.TCGCSVGroups = append(cfg.opt.TCGCSVGroups, n)
	}
	cfg.opt.Languages = cat.opt.Languages
	cfg.opt.SetIDs = cat.opt.SetIDs
	return cfg, nil
}

// newPriceRunner 는 시세 수집 작업들을 만든다. td 는 카탈로그 수집과 같은 TCGdex 클라이언트이다 (요청 간격 공유).
func newPriceRunner(pool *pgxpool.Pool, td *tcgdex.Client) *prices.Runner {
	return prices.NewRunner(
		prices.NewFX(pool, fx.New()),
		prices.NewTCGCSV(pool, tcgcsv.New()),
		prices.NewTCGdex(pool, td),
	)
}

// collectPrices 는 시세 수집을 한 번 실행한다. 작업별 결과는 Runner 가 로그로 남긴다.
func collectPrices(ctx context.Context, runner *prices.Runner, cfg pricesConfig) error {
	if cfg.disabled {
		slog.Info("prices: disabled (PRICES_SOURCES=none)")
		return nil
	}
	slog.Info("prices: start", "sources", cfg.opt.Sources, "languages", cfg.opt.Languages, "sets", cfg.opt.SetIDs, "tcgcsv_groups", cfg.opt.TCGCSVGroups)
	if _, err := runner.Run(ctx, cfg.opt); err != nil {
		return fmt.Errorf("시세 수집: %w", err)
	}
	slog.Info("prices: done")
	return nil
}
