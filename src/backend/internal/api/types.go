// Package api 는 카드 검색·시세 조회 API(#8)의 응답 타입과 계산 규칙이다.
// 응답 형식은 mydocs/tech/api_contract.md 와 프론트엔드 src/frontend/src/api/types.ts 와 같아야 한다.
// DB·HTTP 와 무관한 순수 함수만 두어 단위 테스트로 규칙을 확인한다.
package api

// SetRef 는 목록 항목의 세트 정보이다.
type SetRef struct {
	TcgdexID string `json:"tcgdexId"`
	Name     string `json:"name"`
}

// Price 는 소스 통화 금액과 원화 환산값이다. 금액은 정밀도를 지키려고 문자열이다.
type Price struct {
	Amount    string  `json:"amount"`
	Currency  string  `json:"currency"`
	Source    string  `json:"source"`
	Date      string  `json:"date"`
	KRW       *string `json:"krw"`
	FxDate    *string `json:"fxDate"`
	UpdatedAt *string `json:"updatedAt"`
}

// CardSummary 는 목록에서 카드 하나를 알아보는 데 필요한 최소 정보이다.
type CardSummary struct {
	ID            int64   `json:"id"`
	Language      string  `json:"language"`
	TcgdexID      string  `json:"tcgdexId"`
	Name          string  `json:"name"`
	LocalID       string  `json:"localId"`
	Set           SetRef  `json:"set"`
	ImageURL      *string `json:"imageUrl"`
	HasKoEdition  bool    `json:"hasKoEdition"`
	SpeciesNameKo *string `json:"speciesNameKo"`
	Price         *Price  `json:"price"`
}

// Page 는 쪽 넘김 응답이다.
type Page[T any] struct {
	Items []T   `json:"items"`
	Page  int   `json:"page"`
	Size  int   `json:"size"`
	Total int64 `json:"total"`
}
