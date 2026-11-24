// Package httpx 는 외부 소스 호출용 HTTP 클라이언트이다 (TCGCSV, Frankfurter 등).
//
// 소스마다 호출 예절이 정해져 있어(요청 간 최소 간격, 전용 User-Agent) 클라이언트 전체에서 간격을 공유하고,
// 일시 오류(네트워크 오류, 429, 5xx)만 지수 백오프로 제한 횟수 재시도한다.
// 4xx 는 다시 보내도 같으므로 바로 실패한다. 404 는 ErrNotFound 로 구분한다.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent 는 소스 운영자가 수집기를 구분할 수 있도록 보내는 전용 User-Agent 이다.
const DefaultUserAgent = "PokeCardPrice-collector/0.1"

const (
	maxRetryAfter = 60 * time.Second
	maxErrBody    = 200 // 오류 메시지에 담을 응답 본문 최대 길이
)

// ErrNotFound 는 HTTP 404 이다. 재시도하지 않는다.
var ErrNotFound = errors.New("not found")

// Client 는 여러 고루틴에서 함께 써도 안전하다. 요청 간 최소 간격은 클라이언트 전체에서 공유한다.
type Client struct {
	name        string // 로그 구분용 (tcgcsv, frankfurter)
	hc          *http.Client
	userAgent   string
	minInterval time.Duration
	maxRetries  int
	backoff     time.Duration

	mu   sync.Mutex
	next time.Time // 다음 요청을 보낼 수 있는 가장 이른 시각
}

// Option 은 Client 설정을 바꾼다.
type Option func(*Client)

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

// New 는 기본 설정(간격 없음, 재시도 3회, 첫 대기 2초, 요청 제한 시간 60초) 클라이언트를 만든다.
func New(name string, opts ...Option) *Client {
	c := &Client{
		name:       name,
		hc:         &http.Client{Timeout: 60 * time.Second},
		userAgent:  DefaultUserAgent,
		maxRetries: 3,
		backoff:    2 * time.Second,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// GetJSON 은 url 을 GET 으로 호출해 JSON 을 out 에 담는다.
func (c *Client) GetJSON(ctx context.Context, url string, out any) error {
	return c.get(ctx, url, func(r io.Reader) error {
		if err := json.NewDecoder(r).Decode(out); err != nil {
			var syntaxErr *json.SyntaxError
			var typeErr *json.UnmarshalTypeError
			if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
				// 형식 오류는 다시 받아도 같다.
				return fmt.Errorf("응답 해석: %w", err)
			}
			// 그 밖(본문을 읽다 끊김·시간 초과)은 전송 오류라 재시도한다.
			return &retryableError{fmt.Errorf("응답 본문 읽기: %w", err)}
		}
		return nil
	})
}

// GetText 는 url 을 GET 으로 호출해 본문을 문자열로 돌려준다 (앞뒤 공백 제거).
func (c *Client) GetText(ctx context.Context, url string) (string, error) {
	var s string
	err := c.get(ctx, url, func(r io.Reader) error {
		b, err := io.ReadAll(r)
		if err != nil {
			return &retryableError{fmt.Errorf("응답 본문 읽기: %w", err)}
		}
		s = strings.TrimSpace(string(b))
		return nil
	})
	return s, err
}

func (c *Client) get(ctx context.Context, url string, read func(io.Reader) error) error {
	for attempt := 0; ; attempt++ {
		retryAfter, err := c.do(ctx, url, read)
		if err == nil {
			return nil
		}
		var re *retryableError
		if !errors.As(err, &re) {
			return fmt.Errorf("%s: %w", url, err)
		}
		if attempt >= c.maxRetries {
			return fmt.Errorf("%s: %d회 재시도 후 실패: %w", url, attempt, re.err)
		}
		wait := c.backoff << attempt
		if retryAfter > 0 {
			wait = min(retryAfter, maxRetryAfter)
		}
		slog.Warn("http request retry", "source", c.name, "url", url, "attempt", attempt+1, "wait", wait, "err", re.err)
		if err := Sleep(ctx, wait); err != nil {
			return err
		}
	}
}

type retryableError struct{ err error }

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

// do 는 요청 한 번을 보낸다. 재시도할 오류는 *retryableError 로 감싸고, Retry-After 값을 함께 돌려준다.
func (c *Client) do(ctx context.Context, url string, read func(io.Reader) error) (time.Duration, error) {
	if err := c.wait(ctx); err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", c.userAgent)

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
		err := read(resp.Body)
		if err != nil && ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, err
	case resp.StatusCode == http.StatusNotFound:
		_, _ = io.Copy(io.Discard, resp.Body)
		return 0, ErrNotFound
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		_, _ = io.Copy(io.Discard, resp.Body)
		return parseRetryAfter(resp.Header.Get("Retry-After")), &retryableError{fmt.Errorf("HTTP %d", resp.StatusCode)}
	default:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody))
		_, _ = io.Copy(io.Discard, resp.Body)
		return 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
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
	return Sleep(ctx, time.Until(at))
}

// Sleep 은 d 만큼 기다린다. ctx 가 끝나면 바로 ctx.Err() 를 돌려준다.
func Sleep(ctx context.Context, d time.Duration) error {
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
