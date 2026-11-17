// Package tcgdex 는 TCGdex REST API(https://tcgdex.dev) 클라이언트이다.
//
// TCGdex 는 호출 한도를 공개하지 않고 "배려해서 쓰라"고만 안내한다.
// 그래서 클라이언트 전체에서 요청 간 최소 간격을 지키고, 일시 오류만 제한 횟수로 재시도한다.
// 조사 근거: mydocs/tech/catalog_collector.md
package tcgdex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	// DefaultBaseURL 은 TCGdex REST API v2 주소이다.
	DefaultBaseURL = "https://api.tcgdex.net/v2"
	// DefaultUserAgent 는 소스 운영자가 수집기를 구분할 수 있도록 보내는 전용 User-Agent 이다.
	DefaultUserAgent = "PokeCardPrice-collector/0.1"
	// DefaultMinInterval 은 요청 간 최소 간격이다 (Q1 결정, 2026-11-10).
	DefaultMinInterval = 250 * time.Millisecond

	maxRetryAfter = 30 * time.Second
)

// ErrNotFound 는 요청한 세트·카드가 없을 때(HTTP 404) 돌려준다. 재시도하지 않는다.
var ErrNotFound = errors.New("tcgdex: not found")

// Client 는 TCGdex REST API 클라이언트이다. 여러 고루틴에서 함께 써도 안전하며,
// 요청 간 최소 간격은 클라이언트 전체에서 공유한다.
type Client struct {
	hc          *http.Client
	baseURL     string
	userAgent   string
	minInterval time.Duration
	maxRetries  int
	backoff     time.Duration

	mu   sync.Mutex
	next time.Time // 다음 요청을 보낼 수 있는 가장 이른 시각
}

// Option 은 Client 설정을 바꾼다.
type Option func(*Client)

// WithBaseURL 은 API 주소를 바꾼다 (테스트용).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithHTTPClient 는 HTTP 클라이언트를 바꾼다.
func WithHTTPClient(hc *http.Client) Option { return func(c *Client) { c.hc = hc } }

// WithUserAgent 는 User-Agent 를 바꾼다.
func WithUserAgent(ua string) Option { return func(c *Client) { c.userAgent = ua } }

// WithMinInterval 은 요청 간 최소 간격을 바꾼다. 0 이면 간격을 두지 않는다.
func WithMinInterval(d time.Duration) Option { return func(c *Client) { c.minInterval = d } }

// WithRetry 는 최대 재시도 횟수와 첫 대기 시간을 바꾼다. 대기 시간은 시도마다 두 배가 된다.
func WithRetry(maxRetries int, backoff time.Duration) Option {
	return func(c *Client) { c.maxRetries, c.backoff = maxRetries, backoff }
}

// New 는 기본 설정(요청 간격 250ms, 재시도 3회, 첫 대기 1초, 요청 제한 시간 60초) 클라이언트를 만든다.
func New(opts ...Option) *Client {
	c := &Client{
		hc:          &http.Client{Timeout: 60 * time.Second},
		baseURL:     DefaultBaseURL,
		userAgent:   DefaultUserAgent,
		minInterval: DefaultMinInterval,
		maxRetries:  3,
		backoff:     time.Second,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// ListSets 는 언어(en, ja, ko)의 세트 목록을 돌려준다.
func (c *Client) ListSets(ctx context.Context, lang string) ([]SetBrief, error) {
	var out []SetBrief
	err := c.get(ctx, "/"+url.PathEscape(lang)+"/sets", &out)
	return out, err
}

// GetSet 은 세트 상세(카드 요약 목록 포함)를 돌려준다.
func (c *Client) GetSet(ctx context.Context, lang, id string) (*Set, error) {
	var out Set
	if err := c.get(ctx, "/"+url.PathEscape(lang)+"/sets/"+url.PathEscape(id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCard 는 카드 상세를 돌려준다.
// TCGdex 카드 ID 에는 "%3F" 같은 문자가 그대로 들어 있어(exu-%3F) 반드시 경로 인코딩해서 보낸다.
func (c *Client) GetCard(ctx context.Context, lang, id string) (*Card, error) {
	var out Card
	if err := c.get(ctx, "/"+url.PathEscape(lang)+"/cards/"+url.PathEscape(id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// get 은 path 를 GET 으로 호출해 JSON 을 out 에 담는다.
// 네트워크 오류, 429, 5xx 만 재시도한다. 404 는 ErrNotFound 이다.
func (c *Client) get(ctx context.Context, path string, out any) error {
	u := c.baseURL + path
	var lastErr error
	for attempt := 0; ; attempt++ {
		retryAfter, err := c.do(ctx, u, out)
		if err == nil {
			return nil
		}
		var re *retryableError
		if !errors.As(err, &re) || attempt >= c.maxRetries {
			if errors.As(err, &re) {
				return fmt.Errorf("%s: %d회 재시도 후 실패: %w", u, attempt, re.err)
			}
			return err
		}
		lastErr = re.err
		wait := c.backoff << attempt
		if retryAfter > 0 {
			wait = min(retryAfter, maxRetryAfter)
		}
		slog.Warn("tcgdex request retry", "url", u, "attempt", attempt+1, "wait", wait, "err", lastErr)
		if err := sleep(ctx, wait); err != nil {
			return err
		}
	}
}

type retryableError struct{ err error }

func (e *retryableError) Error() string { return e.err.Error() }

// do 는 요청 한 번을 보낸다. 재시도할 오류는 *retryableError 로 감싸고, Retry-After 값을 함께 돌려준다.
func (c *Client) do(ctx context.Context, u string, out any) (time.Duration, error) {
	if err := c.wait(ctx); err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, &retryableError{err}
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			if ctx.Err() != nil {
				return 0, ctx.Err()
			}
			// JSON 형식·타입 오류는 다시 받아도 같으므로 재시도하지 않는다.
			// 그 밖(본문을 읽다 끊김·시간 초과)은 전송 오류라 재시도한다.
			var syntaxErr *json.SyntaxError
			var typeErr *json.UnmarshalTypeError
			if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
				return 0, fmt.Errorf("%s: 응답 해석: %w", u, err)
			}
			return 0, &retryableError{fmt.Errorf("응답 본문 읽기: %w", err)}
		}
		return 0, nil
	case resp.StatusCode == http.StatusNotFound:
		_, _ = io.Copy(io.Discard, resp.Body)
		return 0, fmt.Errorf("%s: %w", u, ErrNotFound)
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		_, _ = io.Copy(io.Discard, resp.Body)
		return parseRetryAfter(resp.Header.Get("Retry-After")), &retryableError{fmt.Errorf("HTTP %d", resp.StatusCode)}
	default:
		_, _ = io.Copy(io.Discard, resp.Body)
		return 0, fmt.Errorf("%s: HTTP %d", u, resp.StatusCode)
	}
}

// wait 는 이전 요청과 최소 간격이 지날 때까지 기다린다. 동시에 호출되어도 순서대로 간격을 나눠 갖는다.
func (c *Client) wait(ctx context.Context) error {
	if c.minInterval <= 0 {
		return ctx.Err()
	}
	c.mu.Lock()
	now := time.Now()
	at := c.next
	if at.Before(now) {
		at = now
	}
	c.next = at.Add(c.minInterval)
	c.mu.Unlock()
	return sleep(ctx, time.Until(at))
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// parseRetryAfter 는 초 단위 Retry-After 만 해석한다. 날짜 형식이나 잘못된 값이면 0 이다.
func parseRetryAfter(v string) time.Duration {
	s, err := strconv.Atoi(v)
	if err != nil || s <= 0 {
		return 0
	}
	return time.Duration(s) * time.Second
}
