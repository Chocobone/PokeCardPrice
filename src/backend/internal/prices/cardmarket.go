package prices

import (
	"slices"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/catalog"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbq"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/tcgdex"
)

// Cardmarket(TCGdex) 가격 → 카드 변형 배분 규칙 (Q7·Q9, price_collector.md 4절).

// cardmarketUnit 은 Cardmarket 가격 통화이다 (price_sources.currency 와 같아야 한다).
const cardmarketUnit = "EUR"

// cardmarketBaseOrder 는 기본 필드를 넣을 변형 순서이다 (먼저 있는 것). 초판에는 넣지 않는다.
var cardmarketBaseOrder = []string{
	catalog.VariantNormal, catalog.VariantHolofoil, catalog.VariantUnlimitedHolofoil, catalog.VariantUnlimited,
}

// Cardmarket 가격을 저장하지 않은 이유 (요약 로그).
const (
	cmSkipNoPricing = "no_pricing"      // pricing.cardmarket 없음 (카드)
	cmSkipBadUnit   = "unit_not_eur"    // 통화가 EUR 가 아님 (카드)
	cmSkipBadDate   = "updated_invalid" // updated 없음·형식 오류 (카드)
	cmSkipNoValue   = "no_value"        // 기본 필드 trend·avg 모두 없음
	cmSkipNoVariant = "variant_missing" // 값은 있으나 넣을 변형 없음 (기본: 초판만 있는 카드, -holo: 리버스 없음)
)

// cardmarketBaseVariant 는 기본 필드를 넣을 변형이다.
// Q9: normal·holofoil 이 모두 있고 TCGplayer 연결(linked)이 holofoil 에만 있으면 holofoil 이다.
// TCGdex 가 normal 만 둔 홀로 레어에 TCGCSV 근거로 holofoil 을 추가한 경우(Q5)로, 두 소스 가격을 같은 변형에 둔다.
func cardmarketBaseVariant(vs []cardVariant, linked func(int64) bool) (cardVariant, bool) {
	find := func(code string) (cardVariant, bool) {
		i := slices.IndexFunc(vs, func(v cardVariant) bool { return v.Variant == code })
		if i < 0 {
			return cardVariant{}, false
		}
		return vs[i], true
	}
	n, hasN := find(catalog.VariantNormal)
	h, hasH := find(catalog.VariantHolofoil)
	if hasN && hasH && linked(h.ID) && !linked(n.ID) {
		return h, true
	}
	for _, code := range cardmarketBaseOrder {
		if v, ok := find(code); ok {
			return v, true
		}
	}
	return cardVariant{}, false
}

// cardmarketRows 는 카드 하나의 Cardmarket 가격을 price_history 행으로 바꾼다. skips 는 저장하지 않은 이유이다.
// 대표값은 trend, 없으면 avg 이다. 0 이하는 거래가 없는 것으로 보고 NULL 로 둔다.
func cardmarketRows(c targetCard, cm *tcgdex.CardmarketPricing, srcID int16, linked func(int64) bool) (rows []dbq.UpsertPriceHistoryParams, skips []string) {
	if cm == nil {
		return nil, []string{cmSkipNoPricing}
	}
	if cm.Unit != cardmarketUnit {
		return nil, []string{cmSkipBadUnit}
	}
	updated, err := time.Parse(time.RFC3339Nano, cm.Updated)
	if err != nil {
		return nil, []string{cmSkipBadDate}
	}
	updated = updated.UTC()
	date := time.Date(updated.Year(), updated.Month(), updated.Day(), 0, 0, 0, 0, time.UTC)

	add := func(v cardVariant, low, avg, trend, avg1, avg7, avg30 *float64) {
		p := dbq.UpsertPriceHistoryParams{
			CardVariantID: v.ID, SourceID: srcID, PriceDate: date, SourceUpdatedAt: &updated,
			Low: dec(low), Avg: dec(avg), Trend: dec(trend), Avg1: dec(avg1), Avg7: dec(avg7), Avg30: dec(avg30),
		}
		switch {
		case p.Trend != nil:
			p.Price, p.PriceBasis = *p.Trend, "trend"
		case p.Avg != nil:
			p.Price, p.PriceBasis = *p.Avg, "avg"
		default:
			return
		}
		rows = append(rows, p)
	}

	// 기본 필드
	if dec(cm.Trend) == nil && dec(cm.Avg) == nil {
		skips = append(skips, cmSkipNoValue)
	} else if v, ok := cardmarketBaseVariant(c.Variants, linked); ok {
		add(v, cm.Low, cm.Avg, cm.Trend, cm.Avg1, cm.Avg7, cm.Avg30)
	} else {
		skips = append(skips, cmSkipNoVariant)
	}
	// -holo 필드 → 리버스 홀로. 값이 없으면 후보가 아니다 (홀로 카드는 대부분 비어 있음).
	if dec(cm.TrendHolo) != nil || dec(cm.AvgHolo) != nil {
		i := slices.IndexFunc(c.Variants, func(v cardVariant) bool { return v.Variant == catalog.VariantReverseHolofoil })
		if i >= 0 {
			add(c.Variants[i], cm.LowHolo, cm.AvgHolo, cm.TrendHolo, cm.Avg1Holo, cm.Avg7Holo, cm.Avg30Holo)
		} else {
			skips = append(skips, cmSkipNoVariant)
		}
	}
	return rows, skips
}

// dec 는 가격 값을 decimal 로 바꾼다. 없거나 0 이하면 nil 이다.
func dec(f *float64) *decimal.Decimal {
	if f == nil || *f <= 0 {
		return nil
	}
	d := decimal.NewFromFloat(*f)
	return &d
}
