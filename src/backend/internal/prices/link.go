package prices

import (
	"slices"
	"strings"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/catalog"
)

// TCGCSV 상품 ↔ 카드 변형 연결 규칙 (Q1, price_collector.md 2절).

// subTypeVariant 는 TCGCSV subTypeName → card_variants.variant 이다 (7종 1:1).
var subTypeVariant = map[string]string{
	"Normal":               catalog.VariantNormal,
	"Holofoil":             catalog.VariantHolofoil,
	"Reverse Holofoil":     catalog.VariantReverseHolofoil,
	"1st Edition":          catalog.VariantFirstEdition,
	"1st Edition Holofoil": catalog.VariantFirstEditionHolofoil,
	"Unlimited":            catalog.VariantUnlimited,
	"Unlimited Holofoil":   catalog.VariantUnlimitedHolofoil,
}

// tcgdexTCGPlayerSubType 는 TCGdex pricing.tcgplayer 키 → TCGCSV subTypeName 이다 (방법 A).
var tcgdexTCGPlayerSubType = map[string]string{
	"normal":               "Normal",
	"holofoil":             "Holofoil",
	"reverse-holofoil":     "Reverse Holofoil",
	"1st-edition":          "1st Edition",
	"1st-edition-holofoil": "1st Edition Holofoil",
	"unlimited":            "Unlimited",
	"unlimited-holofoil":   "Unlimited Holofoil",
}

// cardVariant 는 카드에 있는 변형 하나이다.
type cardVariant struct {
	ID      int64
	Variant string
}

// resolveVariant 는 subTypeName 에 해당하는 카드 변형을 고른다. 없으면 ok 가 거짓이다.
//
//  1. 같은 변형이 있으면 그것.
//  2. 초판이 있는 카드(first_edition·unlimited 계열)는 Normal → unlimited, Holofoil → unlimited_holofoil.
//     TCGplayer 기본 그룹은 무제한판이고 초판은 다른 그룹에 있다.
//  3. 일본판 카드의 변형이 하나뿐이면 그 변형. TCGCSV 일본판 subTypeName 은 홀로 여부일 뿐 별도 판이 아니다.
func resolveVariant(language string, variants []cardVariant, subTypeName string) (cardVariant, bool) {
	code, ok := subTypeVariant[subTypeName]
	if !ok {
		return cardVariant{}, false
	}
	find := func(code string) (cardVariant, bool) {
		i := slices.IndexFunc(variants, func(v cardVariant) bool { return v.Variant == code })
		if i < 0 {
			return cardVariant{}, false
		}
		return variants[i], true
	}
	if v, ok := find(code); ok {
		return v, true
	}
	switch code {
	case catalog.VariantNormal:
		if v, ok := find(catalog.VariantUnlimited); ok {
			return v, true
		}
	case catalog.VariantHolofoil:
		if v, ok := find(catalog.VariantUnlimitedHolofoil); ok {
			return v, true
		}
	}
	if language == "ja" && len(variants) == 1 {
		return variants[0], true
	}
	return cardVariant{}, false
}

// normalizeNumber 는 카드 번호를 비교용으로 바꾼다.
// "/" 앞만 쓰고 대문자로 바꾸며, 숫자만이면 앞자리 0 을 뗀다 ("001/165" → "1", "TG01/TG30" → "TG01").
func normalizeNumber(s string) string {
	s, _, _ = strings.Cut(strings.TrimSpace(s), "/")
	s = strings.ToUpper(strings.TrimSpace(s))
	if s != "" && strings.Trim(s, "0123456789") == "" {
		s = strings.TrimLeft(s, "0")
		if s == "" {
			s = "0"
		}
	}
	return s
}

// isSpecialProduct 는 이름에 괄호가 있는 상품이다 (Poke Ball Pattern, Metal Card, Pokemon Center Exclusive 등).
// 같은 번호의 상품이 여럿이면 괄호 없는 상품이 기본 카드이다.
func isSpecialProduct(name string) bool { return strings.ContainsAny(name, "([") }
