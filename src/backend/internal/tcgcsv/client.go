// Package tcgcsv 는 TCGCSV(https://tcgcsv.com) 클라이언트이다.
//
// TCGCSV 는 TCGplayer 상품·가격을 매일 한 번(약 20:00 UTC) JSON 으로 내려 주는 비공식 미러이다.
// 운영 규칙(하루 1회 동기화, 요청 간 100 ms 이상, 24시간 1만 회 미만, 전용 User-Agent)을 지키도록
// 기본 요청 간격을 120 ms 로 두고, 클라이언트 하나를 순서대로(동시성 1) 쓴다.
// 조사 근거: mydocs/tech/price_collector.md 1절
package tcgcsv

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/httpx"
)

const (
	// DefaultBaseURL 은 TCGCSV 주소이다.
	DefaultBaseURL = "https://tcgcsv.com"
	// DefaultMinInterval 은 요청 간 최소 간격이다 (운영 규칙 100 ms 이상).
	DefaultMinInterval = 120 * time.Millisecond
)

// TCGplayer 카테고리 ID (price_collector.md 1절).
const (
	CategoryPokemon      = 3  // 영어판 (en)
	CategoryPokemonJapan = 85 // 일본판 (ja)
)

// lastUpdatedLayout 은 /last-updated.txt 형식이다 (예: 2026-11-22T20:05:55+0000).
const lastUpdatedLayout = "2006-01-02T15:04:05-0700"

// Group 은 TCGplayer 그룹(세트·프로모 묶음)이다.
type Group struct {
	GroupID        int    `json:"groupId"`
	Name           string `json:"name"`
	Abbreviation   string `json:"abbreviation"`
	IsSupplemental bool   `json:"isSupplemental"`
	PublishedOn    string `json:"publishedOn"`
	CategoryID     int    `json:"categoryId"`
}

// Code 는 그룹 이름 앞의 세트 코드이다 ("SV2a: Pokemon Card 151" → "SV2a"). 없으면 빈 문자열이다.
func (g Group) Code() string {
	code, _, ok := strings.Cut(g.Name, ": ")
	if !ok || strings.ContainsAny(code, " ()") {
		return ""
	}
	return code
}

// Product 는 TCGplayer 상품이다. 카드가 아닌 상품(박스·팩)은 ExtendedData 에 Number 가 없다.
type Product struct {
	ProductID    int            `json:"productId"`
	Name         string         `json:"name"`
	GroupID      int            `json:"groupId"`
	ExtendedData []ExtendedData `json:"extendedData"`
}

// ExtendedData 는 상품 부가 정보 한 항목이다 (Number, Rarity, Card Type 등).
type ExtendedData struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Number 는 카드 번호(예: "001/165", "014")이다. 카드가 아닌 상품이면 빈 문자열이다.
func (p Product) Number() string {
	for _, e := range p.ExtendedData {
		if e.Name == "Number" {
			return strings.TrimSpace(e.Value)
		}
	}
	return ""
}

// Price 는 상품·하위 유형(subTypeName) 하나의 가격이다 (USD). 값이 없으면 nil 이다.
type Price struct {
	ProductID      int              `json:"productId"`
	SubTypeName    string           `json:"subTypeName"`
	LowPrice       *decimal.Decimal `json:"lowPrice"`
	MidPrice       *decimal.Decimal `json:"midPrice"`
	HighPrice      *decimal.Decimal `json:"highPrice"`
	MarketPrice    *decimal.Decimal `json:"marketPrice"`
	DirectLowPrice *decimal.Decimal `json:"directLowPrice"`
}

// Client 는 TCGCSV 클라이언트이다.
type Client struct {
	http    *httpx.Client
	baseURL string
}

// Option 은 Client 설정을 바꾼다.
type Option func(*Client)

// WithBaseURL 은 주소를 바꾼다 (테스트용).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") } }

// WithHTTP 는 HTTP 클라이언트를 바꾼다 (테스트용 간격·재시도 설정 등).
func WithHTTP(h *httpx.Client) Option { return func(c *Client) { c.http = h } }

// New 는 기본 클라이언트(요청 간 120 ms)를 만든다.
func New(opts ...Option) *Client {
	c := &Client{
		http:    httpx.New("tcgcsv", httpx.WithMinInterval(DefaultMinInterval)),
		baseURL: DefaultBaseURL,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// LastUpdated 는 TCGCSV 가 마지막으로 갱신된 시각이다 (가격의 소스 기준 시각).
func (c *Client) LastUpdated(ctx context.Context) (time.Time, error) {
	s, err := c.http.GetText(ctx, c.baseURL+"/last-updated.txt")
	if err != nil {
		return time.Time{}, fmt.Errorf("tcgcsv: %w", err)
	}
	t, err := time.Parse(lastUpdatedLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("tcgcsv: last-updated %q: %w", s, err)
	}
	return t.UTC(), nil
}

// Groups 는 카테고리의 그룹 목록이다.
func (c *Client) Groups(ctx context.Context, category int) ([]Group, error) {
	return getResults[Group](ctx, c, "/tcgplayer/"+strconv.Itoa(category)+"/groups")
}

// Products 는 그룹의 상품 목록이다.
func (c *Client) Products(ctx context.Context, category, group int) ([]Product, error) {
	return getResults[Product](ctx, c, "/tcgplayer/"+strconv.Itoa(category)+"/"+strconv.Itoa(group)+"/products")
}

// Prices 는 그룹의 가격 목록이다.
func (c *Client) Prices(ctx context.Context, category, group int) ([]Price, error) {
	return getResults[Price](ctx, c, "/tcgplayer/"+strconv.Itoa(category)+"/"+strconv.Itoa(group)+"/prices")
}

// envelope 은 TCGCSV JSON 응답의 공통 형식이다.
type envelope[T any] struct {
	Success bool     `json:"success"`
	Errors  []string `json:"errors"`
	Results []T      `json:"results"`
}

func getResults[T any](ctx context.Context, c *Client, path string) ([]T, error) {
	var env envelope[T]
	if err := c.http.GetJSON(ctx, c.baseURL+path, &env); err != nil {
		return nil, fmt.Errorf("tcgcsv: %w", err)
	}
	if !env.Success {
		return nil, fmt.Errorf("tcgcsv: %s: success=false: %s", path, strings.Join(env.Errors, "; "))
	}
	return env.Results, nil
}
