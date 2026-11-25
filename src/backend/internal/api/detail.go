package api

import (
	"slices"
	"time"

	"github.com/shopspring/decimal"
)

// SetDetail 은 카드 상세의 세트 정보이다.
type SetDetail struct {
	TcgdexID    string  `json:"tcgdexId"`
	Name        string  `json:"name"`
	ReleaseDate *string `json:"releaseDate"`
	LogoURL     *string `json:"logoUrl"`
	CardCount   *int32  `json:"cardCount"`
}

// VariantPrices 는 변형 하나의 소스별 최신가이다. 시세가 없으면 Prices 는 빈 배열이다.
type VariantPrices struct {
	Variant string  `json:"variant"`
	Prices  []Price `json:"prices"`
}

// CardDetail 은 GET /cards/{id} 응답이다. CardSummary 의 set 을 SetDetail 로 바꾸고 상세 필드를 더했다.
type CardDetail struct {
	ID            int64           `json:"id"`
	Language      string          `json:"language"`
	TcgdexID      string          `json:"tcgdexId"`
	Name          string          `json:"name"`
	LocalID       string          `json:"localId"`
	Set           SetDetail       `json:"set"`
	ImageURL      *string         `json:"imageUrl"`
	HasKoEdition  bool            `json:"hasKoEdition"`
	SpeciesNameKo *string         `json:"speciesNameKo"`
	Price         *Price          `json:"price"`
	Rarity        *string         `json:"rarity"`
	DexIDs        []int32         `json:"dexIds"`
	Linked        []CardSummary   `json:"linked"`
	Variants      []VariantPrices `json:"variants"`
}

// PricePoint 는 추이의 날짜별 가격이다.
type PricePoint struct {
	Date   string  `json:"date"`
	Amount string  `json:"amount"`
	KRW    *string `json:"krw"`
}

// Summary 는 기간 요약이다 (#8 Q5).
type Summary struct {
	Current    Price    `json:"current"`
	Average    string   `json:"average"`
	Min        string   `json:"min"`
	Max        string   `json:"max"`
	ChangeRate *float64 `json:"changeRate"`
	From       string   `json:"from"`
}

// PriceSeries 는 변형 하나·소스 하나의 기간 추이이다.
type PriceSeries struct {
	Variant  string       `json:"variant"`
	Source   string       `json:"source"`
	Currency string       `json:"currency"`
	Points   []PricePoint `json:"points"`
	Summary  Summary      `json:"summary"`
}

// PriceHistoryResponse 는 GET /cards/{id}/prices 응답이다.
type PriceHistoryResponse struct {
	CardID int64         `json:"cardId"`
	Period string        `json:"period"`
	From   string        `json:"from"`
	Series []PriceSeries `json:"series"`
}

// PeriodDays 는 기간 이름 → 일수이다.
var PeriodDays = map[string]int{"7d": 7, "30d": 30, "90d": 90, "1y": 365}

// PeriodStart 는 기간 시작일(오늘 KST − 일수, 시작일 포함)이다. 반환값은 UTC 자정의 날짜이다.
func PeriodStart(now time.Time, days int) time.Time {
	y, m, d := now.In(kst).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -days)
}

// SortBySource 는 가격을 소스 우선순위 순으로 정렬한다 (같으면 원래 순서).
func SortBySource(prices []LatestPrice) {
	slices.SortStableFunc(prices, func(a, b LatestPrice) int {
		return rank(SourcePriority, a.Source) - rank(SourcePriority, b.Source)
	})
}

// Summarize 는 날짜 오름차순 가격으로 기간 요약을 만든다. points 는 1개 이상이어야 한다.
// 변동률은 (마지막 − 첫) / 첫 이고, 가격이 하나뿐이거나 첫 가격이 0이면 nil 이다.
// 평균은 날짜별 가격의 산술 평균을 소수 둘째 자리로 반올림한다.
func Summarize(points []LatestPrice) Summary {
	first, last := points[0], points[len(points)-1]
	sum, lo, hi := decimal.Zero, first.Amount, first.Amount
	for _, p := range points {
		sum = sum.Add(p.Amount)
		lo = decimal.Min(lo, p.Amount)
		hi = decimal.Max(hi, p.Amount)
	}
	s := Summary{
		Current: NewPrice(last),
		Average: sum.Div(decimal.NewFromInt(int64(len(points)))).Round(2).StringFixed(2),
		Min:     FormatAmount(lo),
		Max:     FormatAmount(hi),
		From:    FormatDate(first.Date),
	}
	if len(points) > 1 && !first.Amount.IsZero() {
		r := ChangeRate(first.Amount, last.Amount)
		s.ChangeRate = &r
	}
	return s
}

// ChangeRate 는 (current − previous) / previous 이다. previous 는 0 이 아니어야 한다.
func ChangeRate(previous, current decimal.Decimal) float64 {
	return current.Sub(previous).DivRound(previous, 6).InexactFloat64()
}
