package tcgdex

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"time"
)

// 응답 타입은 카탈로그 적재에 쓰는 필드만 정의한다 (mydocs/tech/catalog_collector.md 2절).
// 응답에 없을 수 있는 필드는 포인터로 받는다.

// CardCount 는 세트의 카드 수이다. Official 은 공식 번호 수(시크릿 카드 제외)이다.
type CardCount struct {
	Total    int `json:"total"`
	Official int `json:"official"`
}

// SetBrief 는 세트 목록(/{lang}/sets)의 항목이다. Logo 는 영어판에만 있다.
type SetBrief struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Logo      *string   `json:"logo"`
	CardCount CardCount `json:"cardCount"`
}

// Set 은 세트 상세(/{lang}/sets/{id})이다.
type Set struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Logo        *string     `json:"logo"`
	ReleaseDate *string     `json:"releaseDate"` // "2006-01-02"
	CardCount   CardCount   `json:"cardCount"`
	Serie       *SerieBrief `json:"serie"`
	Cards       []CardBrief `json:"cards"`
}

// SerieBrief 는 세트가 속한 시리즈이다.
type SerieBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Released 는 발매일을 날짜로 돌려준다. 없거나 형식이 다르면 nil 이다.
func (s *Set) Released() *time.Time {
	if s.ReleaseDate == nil {
		return nil
	}
	t, err := time.Parse(time.DateOnly, *s.ReleaseDate)
	if err != nil {
		return nil
	}
	return &t
}

// CardBrief 는 세트 상세에 들어 있는 카드 요약이다. 희귀도·도감 번호·변형은 없다.
type CardBrief struct {
	ID      string  `json:"id"`
	LocalID string  `json:"localId"`
	Name    string  `json:"name"`
	Image   *string `json:"image"` // 확장자 없는 기본 주소. 화면에서 "/high.webp" 등을 붙인다
}

// Card 는 카드 상세(/{lang}/cards/{id})이다.
type Card struct {
	ID       string  `json:"id"`
	LocalID  string  `json:"localId"`
	Name     string  `json:"name"`
	Image    *string `json:"image"`
	Category string  `json:"category"` // Pokemon, Trainer, Energy
	Rarity   *string `json:"rarity"`
	// DexID 는 도감 번호이다. 일부 카드는 384.1 처럼 소수로 오므로 실수로 받는다 (DexIDs 로 정수화).
	DexID    []float64 `json:"dexId"`
	Variants *Variants `json:"variants"`
	Set      struct {
		ID string `json:"id"`
	} `json:"set"`
	// Pricing 은 가격 정보 원본이다 (#6). 카탈로그 적재가 가격 필드 형식 변화로 실패하지 않도록
	// 여기서는 해석하지 않고, 시세 수집기가 TCGPlayerPricing 등으로 따로 해석한다.
	Pricing json.RawMessage `json:"pricing"`
}

// TCGPlayerPrice 는 TCGplayer 상품·하위 유형 하나의 가격이다 (USD).
type TCGPlayerPrice struct {
	ProductID int      `json:"productId"`
	Low       *float64 `json:"lowPrice"`
	Mid       *float64 `json:"midPrice"`
	High      *float64 `json:"highPrice"`
	Market    *float64 `json:"marketPrice"`
	DirectLow *float64 `json:"directLowPrice"`
}

// TCGPlayerPricing 은 카드 상세 pricing.tcgplayer 이다. 키는 TCGplayer 하위 유형이다
// (normal, holofoil, reverse-holofoil, 1st-edition, 1st-edition-holofoil, unlimited, unlimited-holofoil).
type TCGPlayerPricing struct {
	Unit     string
	Updated  string
	Variants map[string]TCGPlayerPrice
}

// TCGPlayer 는 pricing.tcgplayer 를 해석한다. 값이 없으면 nil 이다.
// unit·updated 외의 키 중 객체가 아닌 값은 건너뛴다.
func (c *Card) TCGPlayer() (*TCGPlayerPricing, error) {
	if len(c.Pricing) == 0 || string(c.Pricing) == "null" {
		return nil, nil
	}
	var p struct {
		TCGPlayer map[string]json.RawMessage `json:"tcgplayer"`
	}
	if err := json.Unmarshal(c.Pricing, &p); err != nil {
		return nil, fmt.Errorf("pricing: %w", err)
	}
	if p.TCGPlayer == nil {
		return nil, nil
	}
	out := &TCGPlayerPricing{Variants: map[string]TCGPlayerPrice{}}
	for k, raw := range p.TCGPlayer {
		switch k {
		case "unit":
			_ = json.Unmarshal(raw, &out.Unit)
		case "updated":
			_ = json.Unmarshal(raw, &out.Updated)
		default:
			if len(raw) == 0 || raw[0] != '{' {
				continue
			}
			var v TCGPlayerPrice
			if err := json.Unmarshal(raw, &v); err != nil {
				return nil, fmt.Errorf("pricing.tcgplayer.%s: %w", k, err)
			}
			out.Variants[k] = v
		}
	}
	return out, nil
}

// CardmarketPricing 은 카드 상세 pricing.cardmarket 이다 (EUR).
// 기본 필드와 -holo 필드가 있다. -holo 는 Cardmarket 상품 안의 리버스 홀로 속성 가격이다 (price_collector.md 4절).
// 거래가 없는 필드는 0 으로 오는 경우가 있다.
type CardmarketPricing struct {
	Updated   string   `json:"updated"`
	Unit      string   `json:"unit"`
	IDProduct *int64   `json:"idProduct"`
	Avg       *float64 `json:"avg"`
	Low       *float64 `json:"low"`
	Trend     *float64 `json:"trend"`
	Avg1      *float64 `json:"avg1"`
	Avg7      *float64 `json:"avg7"`
	Avg30     *float64 `json:"avg30"`
	AvgHolo   *float64 `json:"avg-holo"`
	LowHolo   *float64 `json:"low-holo"`
	TrendHolo *float64 `json:"trend-holo"`
	Avg1Holo  *float64 `json:"avg1-holo"`
	Avg7Holo  *float64 `json:"avg7-holo"`
	Avg30Holo *float64 `json:"avg30-holo"`
}

// Cardmarket 은 pricing.cardmarket 을 해석한다. 값이 없으면 nil 이다.
func (c *Card) Cardmarket() (*CardmarketPricing, error) {
	if len(c.Pricing) == 0 || string(c.Pricing) == "null" {
		return nil, nil
	}
	var p struct {
		Cardmarket *CardmarketPricing `json:"cardmarket"`
	}
	if err := json.Unmarshal(c.Pricing, &p); err != nil {
		return nil, fmt.Errorf("pricing.cardmarket: %w", err)
	}
	return p.Cardmarket, nil
}

// Serie 는 시리즈 상세(/{lang}/series/{id})이다.
type Serie struct {
	ID   string     `json:"id"`
	Name string     `json:"name"`
	Sets []SetBrief `json:"sets"`
}

// DexIDs 는 도감 번호를 정수로 돌려준다. 소수는 정수 부분만 쓰고(384.1 → 384), 0 이하는 버리며,
// 중복은 한 번만 넣는다. adjusted 는 소수·0 이하·중복을 고쳤는지 알려 준다 (호출 측에서 로그).
func (c *Card) DexIDs() (ids []int32, adjusted bool) {
	for _, v := range c.DexID {
		n := int32(math.Floor(v))
		if float64(n) != v || n <= 0 || slices.Contains(ids, n) {
			adjusted = true
			if n <= 0 || slices.Contains(ids, n) {
				continue
			}
		}
		ids = append(ids, n)
	}
	return ids, adjusted
}

// Variants 는 카드에 있는 변형 종류이다.
// FirstEdition 이 참이면 "초판만"이 아니라 "초판과 무제한판이 모두 있음"을 뜻한다 (catalog_collector.md 5절).
type Variants struct {
	Normal       bool `json:"normal"`
	Reverse      bool `json:"reverse"`
	Holo         bool `json:"holo"`
	FirstEdition bool `json:"firstEdition"`
	WPromo       bool `json:"wPromo"`
}
