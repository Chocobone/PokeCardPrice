package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/labstack/echo/v5"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/api"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbq"
)

const (
	maxQueryLen   = 100
	maxQueryTerms = 10
	defaultSize   = 24
	maxSize       = 60
	maxPage       = 10000
)

type cardHandlers struct {
	q   *dbq.Queries
	now func() time.Time
}

// search 는 GET /cards 카드 검색이다.
func (h *cardHandlers) search(c *echo.Context) error {
	q := strings.TrimSpace(c.QueryParam("q"))
	if n := utf8.RuneCountInString(q); n < 1 || n > maxQueryLen {
		return invalidQuery(fmt.Sprintf("검색어는 1~%d자여야 합니다", maxQueryLen))
	}
	terms := strings.Fields(strings.ToLower(q))
	if len(terms) > maxQueryTerms {
		return invalidQuery(fmt.Sprintf("검색어는 낱말 %d개까지 입력할 수 있습니다", maxQueryTerms))
	}
	lang, err := parseLanguage(c.QueryParam("lang"))
	if err != nil {
		return err
	}
	page, err := parseIntParam(c, "page", 1, 1, maxPage)
	if err != nil {
		return err
	}
	size, err := parseIntParam(c, "size", defaultSize, 1, maxSize)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	params := searchParams(terms, lang, page, size)
	rows, err := h.q.SearchCards(ctx, params)
	if err != nil {
		return fmt.Errorf("search cards: %w", err)
	}
	var total int64
	if len(rows) > 0 {
		total = rows[0].Total
	} else if page > 1 {
		// 마지막 쪽을 넘으면 행이 없어 전체 건수를 알 수 없다. 첫 행만 다시 읽어 건수를 구한다.
		params.Off, params.Lim = 0, 1
		first, err := h.q.SearchCards(ctx, params)
		if err != nil {
			return fmt.Errorf("search cards total: %w", err)
		}
		if len(first) > 0 {
			total = first[0].Total
		}
	}

	cards := make([]summaryInput, len(rows))
	for i, r := range rows {
		cards[i] = summaryInput{
			ID: r.ID, Language: r.Language, TcgdexID: r.TcgdexID, Name: r.Name, LocalID: r.LocalID,
			SetTcgdexID: r.SetTcgdexID, SetName: r.SetName, ImageURL: r.ImageUrl,
			HasKoEdition: r.HasKoEdition, SpeciesNameKo: r.SpeciesNameKo,
		}
	}
	items, err := h.summaries(ctx, cards)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.Page[api.CardSummary]{Items: items, Page: page, Size: size, Total: total})
}

// searchParams 는 낱말 목록으로 검색 쿼리 매개변수를 만든다.
// 후보를 찾을 낱말은 가장 긴 것(글자 수, 같으면 앞의 것)이다. 길수록 후보가 적고 trigram 인덱스를 쓸 수 있다.
func searchParams(terms []string, lang string, page, size int) dbq.SearchCardsParams {
	key := terms[0]
	for _, t := range terms[1:] {
		if utf8.RuneCountInString(t) > utf8.RuneCountInString(key) {
			key = t
		}
	}
	return dbq.SearchCardsParams{
		Terms:      terms,
		KeyTerm:    key,
		KeyPattern: "%" + escapeLike(key) + "%",
		Query:      strings.Join(terms, " "),
		Language:   lang,
		Lim:        int32(size),
		Off:        int32((page - 1) * size),
	}
}

// escapeLike 는 LIKE 의 특수 문자(\, %, _)를 문자 그대로 찾도록 바꾼다 (기본 이스케이프 문자 \).
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// summaryInput 은 CardSummary 를 만드는 데 필요한 카드 정보이다 (쿼리마다 행 타입이 달라 옮겨 담는다).
type summaryInput struct {
	ID            int64
	Language      string
	TcgdexID      string
	Name          string
	LocalID       string
	SetTcgdexID   string
	SetName       string
	ImageURL      *string
	HasKoEdition  bool
	SpeciesNameKo *string
}

// summaries 는 카드들의 대표가를 한 번에 조회해 CardSummary 목록을 만든다. 순서는 입력 순서이다.
func (h *cardHandlers) summaries(ctx context.Context, cards []summaryInput) ([]api.CardSummary, error) {
	items := make([]api.CardSummary, len(cards))
	if len(cards) == 0 {
		return items, nil
	}
	prices, err := h.latestPrices(ctx, cardIDs(cards))
	if err != nil {
		return nil, err
	}
	for i, cd := range cards {
		items[i] = api.CardSummary{
			ID: cd.ID, Language: cd.Language, TcgdexID: cd.TcgdexID, Name: cd.Name, LocalID: cd.LocalID,
			Set:      api.SetRef{TcgdexID: cd.SetTcgdexID, Name: cd.SetName},
			ImageURL: cd.ImageURL, HasKoEdition: cd.HasKoEdition, SpeciesNameKo: cd.SpeciesNameKo,
			Price: api.PickRepresentative(cd.Language, prices[cd.ID]),
		}
	}
	return items, nil
}

func cardIDs(cards []summaryInput) []int64 {
	ids := make([]int64, len(cards))
	for i, cd := range cards {
		ids[i] = cd.ID
	}
	return ids
}

// latestPrices 는 카드별 변형 × 소스 최신가이다.
func (h *cardHandlers) latestPrices(ctx context.Context, ids []int64) (map[int64][]api.LatestPrice, error) {
	rows, err := h.q.ListLatestPrices(ctx, dbq.ListLatestPricesParams{CardIds: ids, FxProvider: api.FxProvider})
	if err != nil {
		return nil, fmt.Errorf("list latest prices: %w", err)
	}
	m := make(map[int64][]api.LatestPrice, len(ids))
	for _, r := range rows {
		m[r.CardID] = append(m[r.CardID], api.LatestPrice{
			CardID: r.CardID, VariantID: r.CardVariantID, Variant: r.Variant, Source: r.Source, Currency: r.Currency,
			Date: r.PriceDate, Amount: r.Price, UpdatedAt: r.SourceUpdatedAt, FxDate: r.FxRateDate, FxRate: r.FxRate,
		})
	}
	return m, nil
}

// parseLanguage 는 lang 매개변수이다. 비어 있으면 모든 언어("")이다.
func parseLanguage(s string) (string, error) {
	switch s {
	case "", "en", "ja", "ko":
		return s, nil
	}
	return "", invalidParam("lang 은 en, ja, ko 중 하나여야 합니다")
}

// parseIntParam 은 정수 쿼리 매개변수를 읽는다. 비어 있으면 def 이다.
func parseIntParam(c *echo.Context, name string, def, lo, hi int) (int, error) {
	s := c.QueryParam(name)
	if s == "" {
		return def, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < lo || v > hi {
		return 0, invalidParam(fmt.Sprintf("%s 는 %d~%d 사이의 정수여야 합니다", name, lo, hi))
	}
	return v, nil
}
