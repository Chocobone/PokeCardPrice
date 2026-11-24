package prices

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
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

// cardmarketBatchSize 는 Cardmarket 가격을 한 번에 저장하는 행 수이다.
const cardmarketBatchSize = 500

// TCGdex 는 TCGdex 카드 상세를 순회한다 (작업 이름 tcgdex_cardmarket). 카드 상세 응답 하나로 두 가지를 한다.
//
//   - pricing.cardmarket 을 카드 변형에 나눠 price_history(소스 tcgdex_cardmarket)에 저장한다 (Q7·Q9, cardmarket.go).
//   - pricing.tcgplayer 의 TCGplayer 상품 ID 로 external_ids(소스 tcgcsv)를 갱신한다 (Q1 방법 A).
//     같은 실행의 TCGCSV 작업은 이 작업이 끝나기를 기다렸다가 새 연결로 남은 가격을 다시 연결한다.
//
// 대상은 en·ja 전체 카드(TCG Pocket 제외)이고 매일 실행한다 (Q2).
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

// tcgdexRun 은 실행 한 번의 상태이다. 작업자 고루틴들이 함께 쓰므로 mu 로 보호한다.
type tcgdexRun struct {
	pool      *pgxpool.Pool
	q         *dbq.Queries
	tcgcsvSrc int16
	cmSrc     int16

	mu      sync.Mutex
	links   map[linkKey]int64            // 저장된 TCGplayer 외부 ID → 변형
	claims  map[linkKey]map[int64]string // 이번 실행에서 카드 상세가 준 TCGplayer 외부 ID → 변형 → 카드
	linked  map[int64]bool               // TCGplayer 연결이 있는 변형 (Q9)
	res     *Result
	skips   map[string]int // Cardmarket 저장하지 않은 이유
	tp      struct{ linked, unchanged, skipped, duplicate int }
	pending []pendingRows // 저장 대기 중인 Cardmarket 행
	nRows   int
	failed  map[string]bool // 실패한 카드 (재시도 대상)
}

type pendingRows struct {
	card string
	rows []dbq.UpsertPriceHistoryParams
}

// Collect 는 대상 카드의 상세를 받아 Cardmarket 가격을 저장하고 TCGplayer 상품 ID 를 연결한다.
// 실패한 카드(상세 조회·저장)는 끝에 한 번 더 시도하고, 남은 실패는 Result.Failed 에 담는다.
func (s *TCGdex) Collect(ctx context.Context, opt Options) (Result, error) {
	var res Result
	q := dbq.New(s.pool)
	tcgcsvSrc, err := q.GetPriceSourceByCode(ctx, TCGCSVCode)
	if err != nil {
		return res, fmt.Errorf("가격 소스 %s: %w", TCGCSVCode, err)
	}
	cmSrc, err := q.GetPriceSourceByCode(ctx, CardmarketCode)
	if err != nil {
		return res, fmt.Errorf("가격 소스 %s: %w", CardmarketCode, err)
	}
	if cmSrc.Currency != cardmarketUnit {
		return res, fmt.Errorf("가격 소스 %s 통화 %s (%s 이어야 함)", CardmarketCode, cmSrc.Currency, cardmarketUnit)
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
	byKey := make(map[string]targetCard, len(rows))
	for i, r := range rows {
		cards[i] = targetCard{Language: r.Language, TcgdexID: r.TcgdexID, Variants: zipVariants(r.VariantIds, r.Variants)}
		byKey[cards[i].key()] = cards[i]
	}

	links, err := loadLinks(ctx, q, tcgcsvSrc.ID)
	if err != nil {
		return res, err
	}
	run := &tcgdexRun{
		pool: s.pool, q: q, tcgcsvSrc: tcgcsvSrc.ID, cmSrc: cmSrc.ID,
		links: links, claims: map[linkKey]map[int64]string{}, linked: make(map[int64]bool, len(links)),
		res: &res, skips: map[string]int{}, failed: map[string]bool{},
	}
	for _, v := range links {
		run.linked[v] = true
	}
	slog.Info("tcgdex details start", "cards", len(cards), "excluded_sets", len(exclude), "known_links", len(links))

	s.process(ctx, run, cards)
	if len(run.failed) > 0 && ctx.Err() == nil {
		retry := make([]targetCard, 0, len(run.failed))
		for _, k := range slices.Sorted(maps.Keys(run.failed)) {
			retry = append(retry, byKey[k])
		}
		slog.Info("tcgdex details retry failed cards", "cards", len(retry))
		run.failed = map[string]bool{}
		s.process(ctx, run, retry)
	}
	if err := ctx.Err(); err != nil {
		return res, err
	}
	res.Failed = slices.Sorted(maps.Keys(run.failed))
	if err := run.applyLinks(ctx); err != nil {
		return res, err
	}

	reasons := slices.Sorted(maps.Keys(run.skips))
	attrs := make([]any, 0, 2*len(reasons))
	for _, r := range reasons {
		attrs = append(attrs, r, run.skips[r])
	}
	slog.Info("cardmarket skipped", attrs...)
	slog.Info("tcgdex tcgplayer ids", "linked_new_or_changed", run.tp.linked, "unchanged", run.tp.unchanged,
		"skipped", run.tp.skipped, "duplicate_product", run.tp.duplicate)
	if len(res.Failed) > 0 {
		return res, fmt.Errorf("카드 %d장 실패", len(res.Failed))
	}
	return res, nil
}

// process 는 카드들을 작업자 고루틴으로 처리하고, 남은 Cardmarket 행을 저장한다.
func (s *TCGdex) process(ctx context.Context, run *tcgdexRun, cards []targetCard) {
	ch := make(chan targetCard)
	var wg sync.WaitGroup
	for range s.workers {
		wg.Go(func() {
			for c := range ch {
				if err := s.collectCard(ctx, run, c); err != nil {
					if ctx.Err() == nil {
						slog.Warn("tcgdex card failed", "card", c.key(), "err", err)
					}
					run.mu.Lock()
					run.failed[c.key()] = true
					run.mu.Unlock()
				}
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
	run.mu.Lock()
	run.flush(ctx)
	run.mu.Unlock()
}

// collectCard 는 카드 하나의 상세를 받아 TCGplayer 상품 ID 를 연결하고 Cardmarket 행을 저장 대기열에 넣는다.
func (s *TCGdex) collectCard(ctx context.Context, run *tcgdexRun, c targetCard) error {
	card, err := s.client.GetCard(ctx, c.Language, c.TcgdexID)
	if errors.Is(err, tcgdex.ErrNotFound) {
		// 카탈로그에는 있으나 TCGdex 에서 사라진 카드. 다음 카탈로그 갱신 때까지 건너뛴다.
		slog.Warn("tcgdex card not found", "card", c.key())
		return nil
	}
	if err != nil {
		return err
	}
	run.claimTCGPlayer(c, card)

	// 가격 형식 오류는 다시 받아도 같으므로 실패로 두지 않는다.
	cm, err := card.Cardmarket()
	if err != nil {
		slog.Warn("tcgdex cardmarket pricing invalid", "card", c.key(), "err", err)
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	rows, skips := cardmarketRows(c, cm, run.cmSrc, func(id int64) bool { return run.linked[id] })
	run.res.Fetched++
	run.res.Linked += len(rows)
	run.res.Skipped += len(skips)
	for _, r := range skips {
		run.skips[r]++
	}
	if len(rows) > 0 {
		run.pending = append(run.pending, pendingRows{card: c.key(), rows: rows})
		run.nRows += len(rows)
		if run.nRows >= cardmarketBatchSize {
			run.flush(ctx)
		}
	}
	return nil
}

// flush 는 대기 중인 Cardmarket 행을 한 트랜잭션으로 저장한다. 실패하면 그 카드들을 실패로 둔다 (끝에 다시 받음).
// run.mu 를 잡은 채로 부른다.
func (run *tcgdexRun) flush(ctx context.Context) {
	if len(run.pending) == 0 {
		return
	}
	batch := run.pending
	run.pending, run.nRows = nil, 0
	var params []dbq.UpsertPriceHistoryParams
	for _, p := range batch {
		params = append(params, p.rows...)
	}
	err := pgx.BeginFunc(ctx, run.pool, func(tx pgx.Tx) error {
		var errs []error
		run.q.WithTx(tx).UpsertPriceHistory(ctx, params).Exec(func(_ int, err error) {
			if err != nil {
				errs = append(errs, err)
			}
		})
		return errors.Join(errs...)
	})
	if err != nil {
		if ctx.Err() == nil {
			slog.Warn("cardmarket save failed", "cards", len(batch), "rows", len(params), "err", err)
		}
		// 다시 받을 카드이므로 집계에서 뺀다.
		run.res.Fetched -= len(batch)
		run.res.Linked -= len(params)
		for _, p := range batch {
			run.failed[p.card] = true
		}
		return
	}
	run.res.Saved += len(params)
}

// claimTCGPlayer 는 카드 상세 pricing.tcgplayer 의 상품 ID 를 이번 실행의 연결 후보로 모은다 (방법 A).
// 저장은 모든 카드를 받은 뒤 applyLinks 가 한다 (같은 상품 ID 를 여러 카드가 주는지 알아야 해서).
func (run *tcgdexRun) claimTCGPlayer(c targetCard, card *tcgdex.Card) {
	tp, err := card.TCGPlayer()
	if err != nil {
		slog.Warn("tcgdex tcgplayer pricing invalid", "card", c.key(), "err", err)
		return
	}
	if tp == nil {
		return
	}
	for _, k := range slices.Sorted(maps.Keys(tp.Variants)) {
		p := tp.Variants[k]
		sub, ok := tcgdexTCGPlayerSubType[k]
		if !ok || p.ProductID <= 0 {
			slog.Debug("tcgdex tcgplayer key skipped", "card", c.key(), "key", k, "product", p.ProductID)
			run.countTP(func() { run.tp.skipped++ })
			continue
		}
		v, ok := resolveVariant(c.Language, c.Variants, sub)
		if !ok {
			slog.Debug("tcgdex tcgplayer variant missing", "card", c.key(), "key", k)
			run.countTP(func() { run.tp.skipped++ })
			continue
		}
		lk := linkKey{ProductID: strconv.Itoa(p.ProductID), SubType: sub}
		run.mu.Lock()
		if run.claims[lk] == nil {
			run.claims[lk] = map[int64]string{}
		}
		run.claims[lk][v.ID] = c.key()
		// Q9 판단(Cardmarket 기본 필드 배분)에는 바로 쓴다. 중복 상품이어도 그 카드에 TCGplayer 상품이 있다는 뜻은 같다.
		run.linked[v.ID] = true
		run.mu.Unlock()
	}
}

// applyLinks 는 모은 연결 후보를 external_ids 에 저장한다. 저장된 연결과 같으면 다시 쓰지 않는다.
//
// TCGdex 는 서로 다른 카드에 같은 TCGplayer 상품 ID 를 주는 경우가 있다 (bw11-6 과 bw11-RC1 이 같은 상품 등).
// 그런 상품은 어느 카드가 맞는지 알 수 없어 저장된 연결을 바꾸지 않는다 (duplicate_product).
// 저장된 연결이 틀렸으면 TCGCSV 작업이 상품 번호로 걸러 내고 번호 대조로 다시 연결한다 (tcgcsv.go).
func (run *tcgdexRun) applyLinks(ctx context.Context) error {
	for _, lk := range slices.SortedFunc(maps.Keys(run.claims), compareLinkKey) {
		vs := run.claims[lk]
		if len(vs) > 1 {
			cards := slices.Sorted(maps.Values(vs))
			slog.Info("tcgplayer duplicate product", "product", lk.ProductID, "sub_type", lk.SubType, "cards", cards)
			run.tp.duplicate++
			continue
		}
		var v int64
		for id := range vs {
			v = id
		}
		old, had := run.links[lk]
		if had && old == v {
			run.tp.unchanged++
			continue
		}
		if err := run.q.UpsertExternalID(ctx, dbq.UpsertExternalIDParams{
			SourceID: run.tcgcsvSrc, ExternalID: lk.ProductID, ExternalSubID: lk.SubType, CardVariantID: v,
		}); err != nil {
			return fmt.Errorf("external_ids 저장: %w", err)
		}
		if had {
			slog.Info("tcgplayer link changed", "product", lk.ProductID, "sub_type", lk.SubType, "from", old, "to", v, "card", vs[v])
		}
		run.links[lk] = v
		run.tp.linked++
	}
	return nil
}

func compareLinkKey(a, b linkKey) int {
	if c := strings.Compare(a.ProductID, b.ProductID); c != 0 {
		return c
	}
	return strings.Compare(a.SubType, b.SubType)
}

func (run *tcgdexRun) countTP(f func()) {
	run.mu.Lock()
	f()
	run.mu.Unlock()
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
