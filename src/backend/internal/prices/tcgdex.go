package prices

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbq"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/tcgdex"
)

// 가격 소스 코드 (price_sources.code).
const (
	TCGCSVCode     = "tcgcsv"
	CardmarketCode = "tcgdex_cardmarket"
)

// priceLanguages 는 해외 시세를 수집하는 카드 언어이다. 한글판은 TCGdex·TCGCSV 에 가격이 없다.
var priceLanguages = []string{"en", "ja"}

// tcgPocketSerie 는 TCG Pocket(디지털 게임) 시리즈 ID 이다. 실물 카드가 아니라 시세 수집에서 뺀다 (Q6).
const tcgPocketSerie = "tcgp"

// TCGdex 는 TCGdex 카드 상세를 순회한다.
//
// 3단계: 카드 상세 pricing.tcgplayer 의 TCGplayer 상품 ID 로 external_ids(소스 tcgcsv)를 갱신한다 (Q1 방법 A).
// 4단계에서 같은 응답의 Cardmarket 가격 저장을 더한다. 그래서 작업 이름은 tcgdex_cardmarket 이다.
// 같은 실행의 TCGCSV 작업은 이 작업이 끝나기를 기다렸다가 새 연결로 남은 가격을 다시 연결한다.
type TCGdex struct {
	pool    *pgxpool.Pool
	client  *tcgdex.Client
	workers int
}

// NewTCGdex 는 카드 상세 순회 작업을 만든다. client 는 카탈로그 수집기와 같은 설정(요청 간격 공유)을 쓴다.
func NewTCGdex(pool *pgxpool.Pool, client *tcgdex.Client) *TCGdex {
	return &TCGdex{pool: pool, client: client, workers: 2}
}

// Code 는 작업 이름이다.
func (s *TCGdex) Code() string { return CardmarketCode }

// targetCard 는 순회 대상 카드 하나이다.
type targetCard struct {
	Language string
	TcgdexID string
	Variants []cardVariant
}

func (c targetCard) key() string { return c.Language + "/" + c.TcgdexID }

// tcgdexCounts 는 카드 하나의 처리 결과이다.
type tcgdexCounts struct {
	linked, unchanged, skipped int
}

// Collect 는 대상 카드의 상세를 받아 TCGplayer 상품 ID 를 연결한다.
// 실패한 카드는 끝에 한 번 더 시도하고, 남은 실패는 Result.Failed 에 담는다.
func (s *TCGdex) Collect(ctx context.Context, opt Options) (Result, error) {
	var res Result
	q := dbq.New(s.pool)
	src, err := q.GetPriceSourceByCode(ctx, TCGCSVCode)
	if err != nil {
		return res, fmt.Errorf("가격 소스 %s: %w", TCGCSVCode, err)
	}
	var langs []string
	for _, l := range priceLanguages {
		if opt.wantLanguage(l) {
			langs = append(langs, l)
		}
	}
	if len(langs) == 0 {
		return res, nil
	}
	exclude := s.tcgPocketSets(ctx, langs)
	rows, err := q.ListPriceTargetCards(ctx, dbq.ListPriceTargetCardsParams{
		Languages: langs, SetTcgdexIds: nonNil(opt.SetIDs), ExcludeSets: nonNil(exclude),
	})
	if err != nil {
		return res, fmt.Errorf("대상 카드: %w", err)
	}
	cards := make([]targetCard, len(rows))
	for i, r := range rows {
		cards[i] = targetCard{Language: r.Language, TcgdexID: r.TcgdexID, Variants: zipVariants(r.VariantIds, r.Variants)}
	}

	links, err := loadLinks(ctx, q, src.ID)
	if err != nil {
		return res, err
	}
	slog.Info("tcgdex details start", "cards", len(cards), "excluded_sets", len(exclude), "known_links", len(links))

	var mu sync.Mutex
	var total tcgdexCounts
	process := func(cards []targetCard) (failed []targetCard) {
		ch := make(chan targetCard)
		var wg sync.WaitGroup
		for range s.workers {
			wg.Go(func() {
				for c := range ch {
					n, err := s.collectCard(ctx, q, src.ID, links, &mu, c)
					mu.Lock()
					if err != nil {
						if ctx.Err() == nil {
							slog.Warn("tcgdex card failed", "card", c.key(), "err", err)
						}
						failed = append(failed, c)
					} else {
						res.Fetched++
						total.linked += n.linked
						total.unchanged += n.unchanged
						total.skipped += n.skipped
					}
					mu.Unlock()
				}
			})
		}
		for _, c := range cards {
			if ctx.Err() != nil {
				break
			}
			ch <- c
		}
		close(ch)
		wg.Wait()
		return failed
	}

	failed := process(cards)
	if len(failed) > 0 && ctx.Err() == nil {
		slog.Info("tcgdex details retry failed cards", "cards", len(failed))
		failed = process(failed)
	}
	if err := ctx.Err(); err != nil {
		return res, err
	}
	for _, c := range failed {
		res.Failed = append(res.Failed, c.key())
	}
	slices.Sort(res.Failed)
	res.Linked = total.linked + total.unchanged
	res.Skipped = total.skipped
	slog.Info("tcgdex tcgplayer ids", "linked_new_or_changed", total.linked, "unchanged", total.unchanged, "skipped", total.skipped)
	if len(res.Failed) > 0 {
		return res, fmt.Errorf("카드 %d장 실패", len(res.Failed))
	}
	return res, nil
}

// collectCard 는 카드 하나의 상세를 받아 TCGplayer 상품 ID 를 연결한다.
// links 는 저장된 연결이다. 값이 같으면 다시 쓰지 않는다.
func (s *TCGdex) collectCard(ctx context.Context, q *dbq.Queries, srcID int16, links map[linkKey]int64, mu *sync.Mutex, c targetCard) (tcgdexCounts, error) {
	var n tcgdexCounts
	card, err := s.client.GetCard(ctx, c.Language, c.TcgdexID)
	if errors.Is(err, tcgdex.ErrNotFound) {
		// 카탈로그에는 있으나 TCGdex 에서 사라진 카드. 다음 카탈로그 갱신 때까지 건너뛴다.
		slog.Warn("tcgdex card not found", "card", c.key())
		return n, nil
	}
	if err != nil {
		return n, err
	}
	tp, err := card.TCGPlayer()
	if err != nil {
		// 가격 형식이 바뀐 카드. 다시 받아도 같으므로 실패로 두지 않는다.
		slog.Warn("tcgdex tcgplayer pricing invalid", "card", c.key(), "err", err)
		return n, nil
	}
	if tp == nil {
		return n, nil
	}
	keys := make([]string, 0, len(tp.Variants))
	for k := range tp.Variants {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		p := tp.Variants[k]
		sub, ok := tcgdexTCGPlayerSubType[k]
		if !ok || p.ProductID <= 0 {
			slog.Debug("tcgdex tcgplayer key skipped", "card", c.key(), "key", k, "product", p.ProductID)
			n.skipped++
			continue
		}
		v, ok := resolveVariant(c.Language, c.Variants, sub)
		if !ok {
			slog.Debug("tcgdex tcgplayer variant missing", "card", c.key(), "key", k)
			n.skipped++
			continue
		}
		lk := linkKey{ProductID: strconv.Itoa(p.ProductID), SubType: sub}
		mu.Lock()
		old, had := links[lk]
		mu.Unlock()
		if had && old == v.ID {
			n.unchanged++
			continue
		}
		if err := q.UpsertExternalID(ctx, dbq.UpsertExternalIDParams{
			SourceID: srcID, ExternalID: lk.ProductID, ExternalSubID: lk.SubType, CardVariantID: v.ID,
		}); err != nil {
			return n, fmt.Errorf("external_ids 저장: %w", err)
		}
		if had {
			slog.Info("tcgplayer link changed", "product", lk.ProductID, "sub_type", lk.SubType, "from", old, "to", v.ID)
		}
		mu.Lock()
		links[lk] = v.ID
		mu.Unlock()
		n.linked++
	}
	return n, nil
}

// tcgPocketSets 는 언어별 TCG Pocket 세트를 "언어:세트ID" 로 돌려준다.
// 조회에 실패하면 경고만 남기고 빼지 않는다 (가격이 없는 카드라 호출만 늘어난다).
func (s *TCGdex) tcgPocketSets(ctx context.Context, langs []string) []string {
	var out []string
	for _, l := range langs {
		serie, err := s.client.GetSerie(ctx, l, tcgPocketSerie)
		if errors.Is(err, tcgdex.ErrNotFound) {
			continue
		}
		if err != nil {
			slog.Warn("tcgdex tcg pocket sets unavailable", "lang", l, "err", err)
			continue
		}
		for _, set := range serie.Sets {
			out = append(out, l+":"+set.ID)
		}
	}
	return out
}

// linkKey 는 TCGCSV 가격 행의 외부 ID (productId, subTypeName) 이다.
type linkKey struct {
	ProductID string
	SubType   string
}

// loadLinks 는 소스의 저장된 외부 ID → 카드 변형 ID 이다.
func loadLinks(ctx context.Context, q *dbq.Queries, srcID int16) (map[linkKey]int64, error) {
	rows, err := q.ListExternalIDLinks(ctx, srcID)
	if err != nil {
		return nil, fmt.Errorf("external_ids: %w", err)
	}
	m := make(map[linkKey]int64, len(rows))
	for _, r := range rows {
		m[linkKey{r.ExternalID, r.ExternalSubID}] = r.CardVariantID
	}
	return m, nil
}

func zipVariants(ids []int64, codes []string) []cardVariant {
	out := make([]cardVariant, 0, len(ids))
	for i := range min(len(ids), len(codes)) {
		out = append(out, cardVariant{ID: ids[i], Variant: codes[i]})
	}
	return out
}

// nonNil 은 sqlc 배열 인자로 nil 대신 빈 배열을 넘긴다 (NULL 배열이면 cardinality 가 NULL).
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
