package catalog

import (
	"slices"
	"testing"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/tcgdex"
)

// catalog_collector.md 5절 매핑 표와 영어 카드 변형 조합 11종을 모두 확인한다.
func TestMapVariants(t *testing.T) {
	type V = tcgdex.Variants
	tests := []struct {
		name     string
		in       *V
		want     []string
		fallback bool
	}{
		{"normal (bw10-1)", &V{Normal: true}, []string{"normal"}, false},
		{"normal+reverse (sv03.5-001)", &V{Normal: true, Reverse: true}, []string{"normal", "reverse_holofoil"}, false},
		{"holo (dv1-1)", &V{Holo: true}, []string{"holofoil"}, false},
		{"reverse+holo (ex3-1)", &V{Holo: true, Reverse: true}, []string{"holofoil", "reverse_holofoil"}, false},
		{"normal+firstEdition (neo3-15)", &V{Normal: true, FirstEdition: true}, []string{"first_edition", "unlimited"}, false},
		{"normal+reverse+holo (hgss1-1)", &V{Normal: true, Reverse: true, Holo: true}, []string{"normal", "holofoil", "reverse_holofoil"}, false},
		{"holo+firstEdition (base2-1)", &V{Holo: true, FirstEdition: true}, []string{"first_edition_holofoil", "unlimited_holofoil"}, false},
		{"normal+holo (pop5-1)", &V{Normal: true, Holo: true}, []string{"normal", "holofoil"}, false},
		{"reverse (bog-1)", &V{Reverse: true}, []string{"reverse_holofoil"}, false},
		{"normal+reverse+firstEdition (base2-45)", &V{Normal: true, Reverse: true, FirstEdition: true}, []string{"first_edition", "unlimited", "reverse_holofoil"}, false},
		{"normal+holo+firstEdition (base5-15)", &V{Normal: true, Holo: true, FirstEdition: true}, []string{"first_edition", "unlimited", "first_edition_holofoil", "unlimited_holofoil"}, false},
		{"wPromo only", &V{WPromo: true}, []string{"normal"}, true},
		{"all false", &V{}, []string{"normal"}, true},
		{"missing", nil, []string{"normal"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, fb := MapVariants(tt.in)
			if !slices.Equal(got, tt.want) || fb != tt.fallback {
				t.Fatalf("MapVariants = %v, %v; want %v, %v", got, fb, tt.want, tt.fallback)
			}
		})
	}
}
