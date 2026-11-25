package api

import "github.com/shopspring/decimal"

// MoverSource 는 급등·급락 순위의 소스이다 (#8 Q3).
const MoverSource = "tcgcsv"

// MoverMinPrice 는 순위에 넣을 최소 가격이다. 기준일·비교일 가격이 모두 이 값 이상이어야 한다 (USD, #8 Q3).
var MoverMinPrice = decimal.NewFromInt(1)

// MoverPeriodDays 는 순위 기간 이름 → 일수이다.
var MoverPeriodDays = map[string]int{"7d": 7, "30d": 30}

// Mover 는 순위 항목이다. card.price 는 카드 대표가이고, current·previous 는 순위를 낸 변형의 가격이다.
type Mover struct {
	Card       CardSummary `json:"card"`
	Variant    string      `json:"variant"`
	Current    Price       `json:"current"`
	Previous   Price       `json:"previous"`
	ChangeRate float64     `json:"changeRate"`
}

// MoversResponse 는 GET /rankings/movers 응답이다.
type MoversResponse struct {
	Period    string  `json:"period"`
	Direction string  `json:"direction"`
	AsOf      *string `json:"asOf"`
	Items     []Mover `json:"items"`
}
