package tcgdex

import (
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
