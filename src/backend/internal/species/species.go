// Package species 는 포켓몬 종(도감 번호)과 영어·일본어·한국어 이름 데이터이다 (#5, Q5 결정).
//
// 데이터는 PokéAPI 원본 CSV 에서 필요한 열만 뽑아 저장소에 넣었다. 출처·라이선스: LICENSE.pokeapi.md
// 종 이름은 새 세대가 나올 때만 바뀌므로 수집기가 매번 외부를 호출하지 않고 이 파일을 적재한다.
// 갱신 방법: mydocs/tech/catalog_collector.md "종 이름 데이터 갱신"
package species

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"fmt"
	"strconv"
)

//go:embed pokemon_species_names.csv
var namesCSV []byte

// Species 는 포켓몬 종 하나이다. 이름이 없으면 빈 문자열이다 (영어 이름은 항상 있다).
type Species struct {
	DexID  int32
	NameEn string
	NameJa string
	NameKo string
}

// All 은 내장 데이터 전체를 도감 번호 순으로 돌려준다.
func All() ([]Species, error) {
	r := csv.NewReader(bytes.NewReader(namesCSV))
	r.FieldsPerRecord = 4
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("species: CSV 해석: %w", err)
	}
	if len(rows) == 0 || rows[0][0] != "dex_id" {
		return nil, fmt.Errorf("species: 머리글이 없습니다")
	}
	out := make([]Species, 0, len(rows)-1)
	for i, row := range rows[1:] {
		id, err := strconv.ParseInt(row[0], 10, 32)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("species: %d행 도감 번호 %q", i+2, row[0])
		}
		if row[1] == "" {
			return nil, fmt.Errorf("species: %d행 영어 이름 없음", i+2)
		}
		out = append(out, Species{DexID: int32(id), NameEn: row[1], NameJa: row[2], NameKo: row[3]})
	}
	return out, nil
}
