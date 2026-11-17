package species

import "testing"

func TestAll(t *testing.T) {
	all, err := All()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1025 {
		t.Fatalf("종 수 = %d, want 1025", len(all))
	}
	for i, s := range all {
		if s.DexID != int32(i+1) {
			t.Fatalf("%d번째 도감 번호 = %d (연속이 아님)", i, s.DexID)
		}
		if s.NameEn == "" || s.NameJa == "" || s.NameKo == "" {
			t.Errorf("%d: 이름 누락 %+v", s.DexID, s)
		}
	}
	if p := all[24]; p.NameEn != "Pikachu" || p.NameJa != "ピカチュウ" || p.NameKo != "피카츄" {
		t.Errorf("25번 = %+v", p)
	}
}
