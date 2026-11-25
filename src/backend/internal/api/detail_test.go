package api

import (
	"testing"
	"time"
)

func TestPeriodStart(t *testing.T) {
	// 2026-11-24 15:30 UTC = 2026-11-25 00:30 KST → 오늘은 11-25
	now := time.Date(2026, 11, 24, 15, 30, 0, 0, time.UTC)
	if got := FormatDate(PeriodStart(now, 7)); got != "2026-11-18" {
		t.Fatalf("7d = %s", got)
	}
	if got := FormatDate(PeriodStart(now, 365)); got != "2025-11-25" {
		t.Fatalf("1y = %s", got)
	}
}

func TestSummarize(t *testing.T) {
	pt := func(d, amount string) LatestPrice {
		return LatestPrice{Source: "tcgcsv", Currency: "USD", Date: day(d), Amount: dec(amount)}
	}

	s := Summarize([]LatestPrice{pt("2026-11-20", "10.00"), pt("2026-11-21", "8.00"), pt("2026-11-22", "12.01")})
	if s.Current.Amount != "12.01" || s.Current.Date != "2026-11-22" || s.From != "2026-11-20" {
		t.Fatalf("summary = %+v", s)
	}
	// 평균 (10 + 8 + 12.01) / 3 = 10.0033… → 10.00
	if s.Average != "10.00" || s.Min != "8.00" || s.Max != "12.01" {
		t.Fatalf("average/min/max = %s %s %s", s.Average, s.Min, s.Max)
	}
	if s.ChangeRate == nil || *s.ChangeRate != 0.201 {
		t.Fatalf("changeRate = %v", s.ChangeRate)
	}

	// 가격 하나: 변동률 없음
	s = Summarize([]LatestPrice{pt("2026-11-23", "1.5")})
	if s.ChangeRate != nil || s.Average != "1.50" || s.Min != "1.50" || s.Max != "1.50" {
		t.Fatalf("single = %+v", s)
	}
	// 첫 가격 0: 변동률 없음
	s = Summarize([]LatestPrice{pt("2026-11-22", "0"), pt("2026-11-23", "1")})
	if s.ChangeRate != nil {
		t.Fatalf("zero first = %v", *s.ChangeRate)
	}
	// 하락, 평균 반올림 (0.5 올림): (1 + 0.25) / 2 = 0.625 → 0.63
	s = Summarize([]LatestPrice{pt("2026-11-22", "1"), pt("2026-11-23", "0.25")})
	if s.ChangeRate == nil || *s.ChangeRate != -0.75 || s.Average != "0.63" {
		t.Fatalf("down = %+v rate %v", s, s.ChangeRate)
	}
}

func TestChangeRateRounding(t *testing.T) {
	// 1/3 은 소수 여섯째 자리까지
	if got := ChangeRate(dec("3"), dec("4")); got != 0.333333 {
		t.Fatalf("ChangeRate = %v", got)
	}
}

func TestSortBySource(t *testing.T) {
	ps := []LatestPrice{{Source: "tcgdex_cardmarket"}, {Source: "admin"}, {Source: "tcgcsv"}}
	SortBySource(ps)
	if ps[0].Source != "tcgcsv" || ps[1].Source != "tcgdex_cardmarket" || ps[2].Source != "admin" {
		t.Fatalf("order = %+v", ps)
	}
}
