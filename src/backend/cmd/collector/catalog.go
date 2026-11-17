package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/catalog"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/tcgdex"
)

// catalogConfig 는 카탈로그 수집 설정이다. 환경 변수로 바꾼다 (모두 개발용, 비우면 기본값).
//
//   - CATALOG_LANGUAGES: 수집 언어 (쉼표 구분, 예: en,ja). 기본 en,ja,ko
//   - CATALOG_SETS: 수집할 TCGdex 세트 ID (쉼표 구분, 예: sv03.5,SV5K). 기본 전체
//   - CATALOG_FULL: true 이면 매 실행을 전체 갱신으로 한다. 기본은 일정 실행에서 매월 1일만 전체 갱신
type catalogConfig struct {
	opt       catalog.Options
	forceFull bool
}

func catalogConfigFromEnv() (catalogConfig, error) {
	var cfg catalogConfig
	cfg.opt.Languages = splitList(os.Getenv("CATALOG_LANGUAGES"))
	for _, l := range cfg.opt.Languages {
		if l != "en" && l != "ja" && l != "ko" {
			return cfg, fmt.Errorf("CATALOG_LANGUAGES: 지원하지 않는 언어 %q (en, ja, ko)", l)
		}
	}
	cfg.opt.SetIDs = splitList(os.Getenv("CATALOG_SETS"))
	if v := os.Getenv("CATALOG_FULL"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return cfg, fmt.Errorf("CATALOG_FULL: %w", err)
		}
		cfg.forceFull = b
	}
	return cfg, nil
}

func splitList(v string) []string {
	var out []string
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// isFullRefreshDay 는 정기 전체 갱신 날(매월 1일)인지 알려 준다 (Q1).
// 전체 갱신을 별도 작업으로 두면 같은 시각의 매일 작업과 겹쳐 같은 테이블에 동시에 쓰게 되므로,
// 매일 작업 하나가 날짜를 보고 범위를 정한다.
func isFullRefreshDay(now time.Time) bool { return now.Day() == 1 }

// collectCatalog 는 카탈로그 수집을 한 번 실행하고 결과를 로그로 남긴다.
// 일부 세트만 실패하면 WARN 으로 남기고 성공으로 끝낸다 (다음 실행에서 다시 시도된다).
func collectCatalog(ctx context.Context, pool *pgxpool.Pool, cfg catalogConfig, full bool) error {
	opt := cfg.opt
	opt.Full = full || cfg.forceFull
	slog.Info("catalog: start", "languages", opt.Languages, "sets", opt.SetIDs, "full", opt.Full)
	res, err := catalog.New(pool, tcgdex.New()).Run(ctx, opt)
	if err != nil {
		slog.Error("catalog: failed", "result", res, "err", err)
		return fmt.Errorf("카탈로그 수집: %w", err)
	}
	if len(res.FailedSets) > 0 {
		slog.Warn("catalog: done with failed sets", "result", res)
		return nil
	}
	slog.Info("catalog: done", "result", res)
	return nil
}
