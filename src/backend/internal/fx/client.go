// Package fx 는 Frankfurter v2 환율 API(https://frankfurter.dev) 클라이언트이다.
//
// 키가 필요 없고 주말·공휴일에도 그 날짜의 혼합값(여러 기관 값의 중간값)을 준다.
// 아직 값이 없는 날짜(미래 등)를 물으면 최신 값을 돌려주므로, 저장할 때는 응답의 date 를 기준일로 쓴다.
// 조사 근거: mydocs/tech/price_collector.md 6절
package fx

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/httpx"
)

// DefaultBaseURL 은 Frankfurter API 주소이다.
const DefaultBaseURL = "https://api.frankfurter.dev"

// Provider 는 fx_rates.provider 에 저장하는 값이다.
const Provider = "frankfurter"

// Rate 는 base 1 단위가 quote 몇 단위인지이다.
type Rate struct {
	Date  time.Time // 값 기준일 (UTC 자정)
	Base  string
	Quote string
	Rate  decimal.Decimal
}

// Client 는 Frankfurter v2 클라이언트이다.
type Client struct {
	http    *httpx.Client
	baseURL string
}

// Option 은 Client 설정을 바꾼다.
type Option func(*Client)

// WithBaseURL 은 API 주소를 바꾼다 (테스트용).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") } }

// WithHTTP 는 HTTP 클라이언트를 바꾼다 (테스트용 재시도 설정 등).
func WithHTTP(h *httpx.Client) Option { return func(c *Client) { c.http = h } }

// New 는 기본 클라이언트를 만든다. 하루 몇 번만 호출하므로 요청 간격은 1초로 넉넉히 둔다.
func New(opts ...Option) *Client {
	c := &Client{
		http:    httpx.New("frankfurter", httpx.WithMinInterval(time.Second)),
		baseURL: DefaultBaseURL,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

type rateJSON struct {
	Date  string          `json:"date"`
	Base  string          `json:"base"`
	Quote string          `json:"quote"`
	Rate  decimal.Decimal `json:"rate"`
}

// Rates 는 base 기준 quotes 환율을 돌려준다. date 가 0 이면 최신 값이다.
// 결과가 비어 있을 수 있다 (예: 해당 통화를 다루는 제공처가 없음). 호출 측에서 확인한다.
func (c *Client) Rates(ctx context.Context, base string, quotes []string, date time.Time) ([]Rate, error) {
	q := url.Values{}
	q.Set("base", base)
	q.Set("quotes", strings.Join(quotes, ","))
	if !date.IsZero() {
		q.Set("date", date.Format(time.DateOnly))
	}
	var raw []rateJSON
	if err := c.http.GetJSON(ctx, c.baseURL+"/v2/rates?"+q.Encode(), &raw); err != nil {
		return nil, fmt.Errorf("frankfurter: %w", err)
	}
	out := make([]Rate, 0, len(raw))
	for _, r := range raw {
		d, err := time.Parse(time.DateOnly, r.Date)
		if err != nil {
			return nil, fmt.Errorf("frankfurter: 날짜 %q: %w", r.Date, err)
		}
		if !r.Rate.IsPositive() {
			return nil, fmt.Errorf("frankfurter: %s→%s 환율이 양수가 아님: %s", r.Base, r.Quote, r.Rate)
		}
		out = append(out, Rate{Date: d, Base: r.Base, Quote: r.Quote, Rate: r.Rate})
	}
	return out, nil
}
