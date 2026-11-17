package catalog

import "github.com/Chocobone/PokeCardPrice/src/backend/internal/tcgdex"

// card_variants.variant 코드. TCGCSV subTypeName 과 1:1 로 대응한다 (#6 연결용).
const (
	VariantNormal               = "normal"                 // Normal
	VariantHolofoil             = "holofoil"               // Holofoil
	VariantReverseHolofoil      = "reverse_holofoil"       // Reverse Holofoil
	VariantFirstEdition         = "first_edition"          // 1st Edition
	VariantFirstEditionHolofoil = "first_edition_holofoil" // 1st Edition Holofoil
	VariantUnlimited            = "unlimited"              // Unlimited
	VariantUnlimitedHolofoil    = "unlimited_holofoil"     // Unlimited Holofoil
)

// MapVariants 는 TCGdex 변형 값을 card_variants.variant 코드로 바꾼다 (catalog_collector.md 5절, Q3 결정).
//
//   - FirstEdition 이 참이면 초판과 무제한판이 모두 있다는 뜻이라 normal·holo 를 각각 두 변형으로 나눈다.
//   - reverse 는 초판 여부와 관계없이 reverse_holofoil 이다.
//   - wPromo 는 7종으로 표현할 수 없어 변형을 만들지 않는다 (호출 측에서 로그).
//   - 값이 없거나 모두 거짓이면 normal 하나로 본다 (가격을 연결할 변형이 최소 하나는 있어야 한다).
//
// fallback 은 마지막 규칙이 적용되었는지 알려 준다.
func MapVariants(v *tcgdex.Variants) (codes []string, fallback bool) {
	if v == nil {
		return []string{VariantNormal}, true
	}
	if v.Normal {
		if v.FirstEdition {
			codes = append(codes, VariantFirstEdition, VariantUnlimited)
		} else {
			codes = append(codes, VariantNormal)
		}
	}
	if v.Holo {
		if v.FirstEdition {
			codes = append(codes, VariantFirstEditionHolofoil, VariantUnlimitedHolofoil)
		} else {
			codes = append(codes, VariantHolofoil)
		}
	}
	if v.Reverse {
		codes = append(codes, VariantReverseHolofoil)
	}
	if len(codes) == 0 {
		return []string{VariantNormal}, true
	}
	return codes, false
}
