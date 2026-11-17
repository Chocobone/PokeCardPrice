package main

import (
	"slices"
	"testing"
	"time"
)

func TestCatalogConfigFromEnv(t *testing.T) {
	t.Setenv("CATALOG_LANGUAGES", " en, ja ,")
	t.Setenv("CATALOG_SETS", "sv03.5,SV5K")
	t.Setenv("CATALOG_FULL", "true")
	cfg, err := catalogConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cfg.opt.Languages, []string{"en", "ja"}) || !slices.Equal(cfg.opt.SetIDs, []string{"sv03.5", "SV5K"}) || !cfg.forceFull {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestCatalogConfigDefaults(t *testing.T) {
	for _, k := range []string{"CATALOG_LANGUAGES", "CATALOG_SETS", "CATALOG_FULL"} {
		t.Setenv(k, "")
	}
	cfg, err := catalogConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.opt.Languages != nil || cfg.opt.SetIDs != nil || cfg.forceFull {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestCatalogConfigInvalid(t *testing.T) {
	t.Setenv("CATALOG_LANGUAGES", "en,fr")
	if _, err := catalogConfigFromEnv(); err == nil {
		t.Error("지원하지 않는 언어인데 오류가 없음")
	}
	t.Setenv("CATALOG_LANGUAGES", "")
	t.Setenv("CATALOG_FULL", "yes")
	if _, err := catalogConfigFromEnv(); err == nil {
		t.Error("CATALOG_FULL 형식 오류인데 오류가 없음")
	}
}

func TestIsFullRefreshDay(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Seoul")
	if !isFullRefreshDay(time.Date(2026, 12, 1, 5, 30, 0, 0, loc)) {
		t.Error("12월 1일이 전체 갱신 날이 아님")
	}
	if isFullRefreshDay(time.Date(2026, 12, 2, 5, 30, 0, 0, loc)) {
		t.Error("12월 2일이 전체 갱신 날임")
	}
}
