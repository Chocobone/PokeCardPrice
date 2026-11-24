package main

import (
	"slices"
	"testing"
)

func TestPricesConfigFromEnv(t *testing.T) {
	t.Setenv("PRICES_SOURCES", "tcgcsv, tcgdex_cardmarket")
	t.Setenv("PRICES_TCGCSV_GROUPS", "23237,22872")
	cat := catalogConfig{}
	cat.opt.Languages = []string{"en"}
	cat.opt.SetIDs = []string{"sv03.5"}
	cfg, err := pricesConfigFromEnv(cat)
	if err != nil {
		t.Fatal(err)
	}
	o := cfg.opt
	if cfg.disabled || !slices.Equal(o.Sources, []string{"tcgcsv", "tcgdex_cardmarket"}) || !slices.Equal(o.TCGCSVGroups, []int{23237, 22872}) ||
		!slices.Equal(o.Languages, []string{"en"}) || !slices.Equal(o.SetIDs, []string{"sv03.5"}) {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestPricesConfigDefaultsAndNone(t *testing.T) {
	t.Setenv("PRICES_SOURCES", "")
	t.Setenv("PRICES_TCGCSV_GROUPS", "")
	cfg, err := pricesConfigFromEnv(catalogConfig{})
	if err != nil || cfg.disabled || cfg.opt.Sources != nil || cfg.opt.TCGCSVGroups != nil {
		t.Errorf("cfg=%+v err=%v", cfg, err)
	}
	t.Setenv("PRICES_SOURCES", "none")
	if cfg, err := pricesConfigFromEnv(catalogConfig{}); err != nil || !cfg.disabled {
		t.Errorf("none: cfg=%+v err=%v", cfg, err)
	}
}

func TestPricesConfigInvalid(t *testing.T) {
	t.Setenv("PRICES_TCGCSV_GROUPS", "")
	t.Setenv("PRICES_SOURCES", "tcgcsv,cardmarket")
	if _, err := pricesConfigFromEnv(catalogConfig{}); err == nil {
		t.Error("알 수 없는 작업인데 오류가 없음")
	}
	t.Setenv("PRICES_SOURCES", "")
	t.Setenv("PRICES_TCGCSV_GROUPS", "23237,abc")
	if _, err := pricesConfigFromEnv(catalogConfig{}); err == nil {
		t.Error("그룹 ID 형식 오류인데 오류가 없음")
	}
}
