package api

import (
	"time"

	"github.com/shopspring/decimal"
)

// FxProvider 는 원화 환산에 쓰는 환율 제공자(fx_rates.provider)이다.
const FxProvider = "frankfurter"

// kst 는 시각을 응답할 때 쓰는 시간대이다. 실행 환경에 시간대 데이터가 없으면 고정 +09:00 을 쓴다.
var kst = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Seoul"); err == nil {
		return loc
	}
	return time.FixedZone("KST", 9*60*60)
}()

// SourcePriority 는 대표가를 고를 때의 소스 순서이다 (#8 Q1).
var SourcePriority = []string{"tcgcsv", "tcgdex_cardmarket"}

// VariantPriority 는 대표가를 고를 때의 변형 순서이다 (#8 Q1).
var VariantPriority = []string{
	"normal", "holofoil", "unlimited_holofoil", "unlimited",
	"reverse_holofoil", "first_edition_holofoil", "first_edition",
}

// LatestPrice 는 변형 하나·소스 하나의 가격과 환산용 환율이다 (DB 행을 옮겨 담는다).
type LatestPrice struct {
	CardID    int64
	VariantID int64
	Variant   string
	Source    string
	Currency  string
	Date      time.Time
	Amount    decimal.Decimal
	UpdatedAt *time.Time
	FxDate    *time.Time
	FxRate    *decimal.Decimal
}

// FormatDate 는 날짜를 YYYY-MM-DD 로 쓴다.
func FormatDate(t time.Time) string { return t.Format(time.DateOnly) }

// FormatAmount 는 소스 통화 금액을 소수 둘째 자리까지 쓴다 (numeric(12,2)).
func FormatAmount(d decimal.Decimal) string { return d.StringFixed(2) }

// ToKRW 는 금액을 원화로 바꿔 원 단위로 반올림한다 (0.5 는 0 에서 먼 쪽으로).
// 소스 통화가 KRW 이면 그대로, 환율이 없으면 nil 이다.
func ToKRW(amount decimal.Decimal, currency string, rate *decimal.Decimal) *string {
	var v decimal.Decimal
	switch {
	case currency == "KRW":
		v = amount
	case rate == nil:
		return nil
	default:
		v = amount.Mul(*rate)
	}
	s := v.Round(0).StringFixed(0)
	return &s
}

// NewPrice 는 응답용 가격을 만든다.
func NewPrice(lp LatestPrice) Price {
	p := Price{
		Amount:   FormatAmount(lp.Amount),
		Currency: lp.Currency,
		Source:   lp.Source,
		Date:     FormatDate(lp.Date),
		KRW:      ToKRW(lp.Amount, lp.Currency, lp.FxRate),
	}
	if lp.Currency != "KRW" && lp.FxRate != nil && lp.FxDate != nil {
		s := FormatDate(*lp.FxDate)
		p.FxDate = &s
	}
	if lp.UpdatedAt != nil {
		s := lp.UpdatedAt.In(kst).Format(time.RFC3339)
		p.UpdatedAt = &s
	}
	return p
}

// SourceRank 는 소스 우선순위 순번이다. 목록에 없으면 가장 뒤이다.
func SourceRank(source string) int { return rank(SourcePriority, source) }

func rank(list []string, v string) int {
	for i, s := range list {
		if s == v {
			return i
		}
	}
	return len(list)
}

// PickRepresentative 는 카드 하나의 변형 × 소스 최신가 중 대표가를 고른다 (#8 Q1).
// 소스 우선순위가 먼저이고, 같은 소스 안에서는 변형 우선순위가 높은 것을 쓴다.
// 한글판 카드와 가격이 없는 카드는 nil 이다 (한글판 가격은 #7).
func PickRepresentative(language string, prices []LatestPrice) *Price {
	if language == "ko" {
		return nil
	}
	var best *LatestPrice
	for i := range prices {
		lp := &prices[i]
		if rank(SourcePriority, lp.Source) == len(SourcePriority) {
			continue
		}
		if best == nil || less(lp, best) {
			best = lp
		}
	}
	if best == nil {
		return nil
	}
	p := NewPrice(*best)
	return &p
}

// less 는 a 가 b 보다 대표가로 앞서는지이다.
func less(a, b *LatestPrice) bool {
	if sa, sb := rank(SourcePriority, a.Source), rank(SourcePriority, b.Source); sa != sb {
		return sa < sb
	}
	if va, vb := rank(VariantPriority, a.Variant), rank(VariantPriority, b.Variant); va != vb {
		return va < vb
	}
	return a.VariantID < b.VariantID
}
