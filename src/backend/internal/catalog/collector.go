// Package catalog 는 TCGdex 의 세트·카드·카드 변형을 카탈로그 테이블에 적재한다 (#5).
//
// 흐름 (언어마다):
//  1. 세트 목록을 받는다.
//  2. 세트마다 세트 상세(카드 요약 포함)를 받는다.
//  3. DB 에 없는 카드만 카드 상세를 받는다 (Full 이면 모든 카드). 희귀도·도감 번호·변형은 카드 상세에만 있다.
//  4. 세트 하나를 트랜잭션 하나로 저장한다. 세트가 실패하면 그 세트만 롤백하고 다음 세트로 넘어간다.
//
// 외부 호출은 트랜잭션 밖에서 끝내, 느린 응답 때문에 트랜잭션이 오래 열려 있지 않게 한다.
// 소스에서 사라진 세트·카드는 지우지 않는다 (가격·컬렉션이 참조할 수 있다).
// 설계 근거: mydocs/tech/catalog_collector.md
package catalog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbq"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/tcgdex"
)

// Languages 는 기본 수집 언어이다.
var Languages = []string{"en", "ja", "ko"}

// DefaultConcurrency 는 카드 상세 동시 요청 수이다 (Q1 결정). 요청 간격은 클라이언트가 지킨다.
const DefaultConcurrency = 2

// Options 는 수집 범위를 정한다.
type Options struct {
	// Languages 가 비어 있으면 Languages(en, ja, ko) 전체.
	Languages []string
	// SetIDs 가 비어 있으면 전체 세트. 개발 중 일부만 적재할 때 쓴다 (TCGdex 세트 ID, 언어 공통).
	SetIDs []string
	// Full 이면 이미 있는 카드도 카드 상세를 다시 받아 희귀도·도감 번호·변형을 갱신한다 (월 1회 전체 갱신).
	Full bool
	// Concurrency 는 카드 상세 동시 요청 수. 0 이면 DefaultConcurrency.
	Concurrency int
}

// Result 는 수집 결과 집계이다.
type Result struct {
	Sets          int      // 저장한 세트
	SetsSkipped   int      // 세트 상세가 404 라 건너뛴 세트
	CardsNew      int      // 새로 넣은 카드
	CardsDetailed int      // 카드 상세로 갱신한 기존 카드 (Full)
	CardsBrief    int      // 요약 필드(번호·이름·이미지)가 바뀐 기존 카드
	CardsSkipped  int      // 신규인데 카드 상세가 404 라 넣지 못한 카드
	Variants      int      // upsert 한 변형 (기존 포함)
	FailedSets    []string // 실패한 세트 ("{lang}/{id}"). 다음 실행에서 다시 시도된다
	Duration      time.Duration
}

func (r *Result) add(o Result) {
	r.Sets += o.Sets
	r.SetsSkipped += o.SetsSkipped
	r.CardsNew += o.CardsNew
	r.CardsDetailed += o.CardsDetailed
	r.CardsBrief += o.CardsBrief
	r.CardsSkipped += o.CardsSkipped
	r.Variants += o.Variants
}

// LogValue 는 slog 로 결과를 남길 때 쓴다.
func (r Result) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("sets", r.Sets), slog.Int("sets_skipped", r.SetsSkipped),
		slog.Int("cards_new", r.CardsNew), slog.Int("cards_detailed", r.CardsDetailed),
		slog.Int("cards_brief", r.CardsBrief), slog.Int("cards_skipped", r.CardsSkipped),
		slog.Int("variants", r.Variants), slog.Any("failed_sets", r.FailedSets),
		slog.Duration("duration", r.Duration),
	)
}

// Collector 는 카탈로그 수집기이다.
type Collector struct {
	pool *pgxpool.Pool
	api  *tcgdex.Client
}

// New 는 수집기를 만든다.
func New(pool *pgxpool.Pool, api *tcgdex.Client) *Collector {
	return &Collector{pool: pool, api: api}
}

// Run 은 opt 범위의 세트·카드를 적재한다.
// 세트 단위 실패는 Result.FailedSets 에 모으고 계속 진행한다. 세트 목록을 하나도 받지 못했거나
// 모든 세트가 실패하면 오류를 돌려준다. ctx 가 취소되면 진행 중인 세트를 롤백하고 멈춘다.
func (c *Collector) Run(ctx context.Context, opt Options) (Result, error) {
	start := time.Now()
	langs := opt.Languages
	if len(langs) == 0 {
		langs = Languages
	}
	var res Result
	var listErrs []error
	attempted := 0
	for _, lang := range langs {
		sets, err := c.api.ListSets(ctx, lang)
		if err != nil {
			if ctx.Err() != nil {
				return res, ctx.Err()
			}
			slog.Error("catalog: list sets failed", "lang", lang, "err", err)
			listErrs = append(listErrs, fmt.Errorf("%s 세트 목록: %w", lang, err))
			continue
		}
		for _, sb := range sets {
			if len(opt.SetIDs) > 0 && !slices.Contains(opt.SetIDs, sb.ID) {
				continue
			}
			attempted++
			r, err := c.collectSet(ctx, lang, sb, opt)
			if err != nil {
				if ctx.Err() != nil {
					res.Duration = time.Since(start)
					return res, ctx.Err()
				}
				slog.Error("catalog: set failed", "lang", lang, "set", sb.ID, "err", err)
				res.FailedSets = append(res.FailedSets, lang+"/"+sb.ID)
				continue
			}
			res.add(r)
		}
	}
	res.Duration = time.Since(start)
	if len(listErrs) == len(langs) {
		return res, errors.Join(listErrs...)
	}
	if attempted > 0 && len(res.FailedSets) == attempted {
		return res, fmt.Errorf("모든 세트(%d개) 적재 실패", attempted)
	}
	return res, nil
}

// collectSet 은 세트 하나를 받아 저장한다.
func (c *Collector) collectSet(ctx context.Context, lang string, sb tcgdex.SetBrief, opt Options) (Result, error) {
	var res Result
	set, err := c.api.GetSet(ctx, lang, sb.ID)
	if errors.Is(err, tcgdex.ErrNotFound) {
		// 목록에는 있지만 상세가 없는 세트 (예: ko SM1+). 카드가 없어 건너뛴다.
		slog.Warn("catalog: set detail not found, skipped", "lang", lang, "set", sb.ID)
		res.SetsSkipped = 1
		return res, nil
	}
	if err != nil {
		return res, err
	}

	q := dbq.New(c.pool)
	known, err := q.ListCardTcgdexIDsBySet(ctx, dbq.ListCardTcgdexIDsBySetParams{Language: lang, SetTcgdexID: set.ID})
	if err != nil {
		return res, err
	}
	isKnown := make(map[string]bool, len(known))
	for _, id := range known {
		isKnown[id] = true
	}
	var need []string
	for _, cb := range set.Cards {
		if opt.Full || !isKnown[cb.ID] {
			need = append(need, cb.ID)
		}
	}
	details, err := c.fetchCards(ctx, lang, need, opt.Concurrency)
	if err != nil {
		return res, err
	}

	logo := set.Logo
	if logo == nil {
		logo = sb.Logo
	}
	cardCount := int32(set.CardCount.Total)

	err = pgx.BeginFunc(ctx, c.pool, func(tx pgx.Tx) error {
		q := q.WithTx(tx)
		setID, err := q.UpsertSet(ctx, dbq.UpsertSetParams{
			Language: lang, TcgdexID: set.ID, Name: set.Name,
			ReleaseDate: set.Released(), CardCount: &cardCount, LogoUrl: logo,
		})
		if err != nil {
			return fmt.Errorf("세트 저장: %w", err)
		}
		for _, cb := range set.Cards {
			d, ok := details[cb.ID]
			switch {
			case ok:
				n, err := upsertCardDetail(ctx, q, setID, lang, cb, d)
				if err != nil {
					return fmt.Errorf("카드 %s: %w", cb.ID, err)
				}
				res.Variants += n
				if isKnown[cb.ID] {
					res.CardsDetailed++
				} else {
					res.CardsNew++
				}
			case isKnown[cb.ID]:
				n, err := q.UpdateCardBrief(ctx, dbq.UpdateCardBriefParams{
					Language: lang, TcgdexID: cb.ID, LocalID: cb.LocalID, Name: cb.Name, ImageUrl: cb.Image,
				})
				if err != nil {
					return fmt.Errorf("카드 %s 요약 갱신: %w", cb.ID, err)
				}
				res.CardsBrief += int(n)
			default:
				// 신규 카드인데 상세가 404 였다. 다음 실행에서 다시 시도된다.
				res.CardsSkipped++
			}
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	res.Sets = 1
	slog.Info("catalog: set done", "lang", lang, "set", set.ID, "cards", len(set.Cards),
		"new", res.CardsNew, "detailed", res.CardsDetailed, "brief", res.CardsBrief, "skipped", res.CardsSkipped)
	return res, nil
}

// upsertCardDetail 은 카드 요약(세트 상세)과 카드 상세로 카드와 변형을 저장하고, upsert 한 변형 수를 돌려준다.
// 번호·이름·이미지는 세트 상세 값을 쓴다 (매일 갱신하는 값과 같은 출처를 쓰기 위해).
func upsertCardDetail(ctx context.Context, q *dbq.Queries, setID int64, lang string, cb tcgdex.CardBrief, d *tcgdex.Card) (int, error) {
	var dex []int32
	if len(d.DexID) > 0 {
		dex = d.DexID
	}
	cardID, err := q.UpsertCard(ctx, dbq.UpsertCardParams{
		SetID: setID, Language: lang, TcgdexID: cb.ID, LocalID: cb.LocalID, Name: cb.Name,
		Rarity: d.Rarity, ImageUrl: cb.Image, DexIds: dex,
	})
	if err != nil {
		return 0, err
	}
	codes, fallback := MapVariants(d.Variants)
	if fallback {
		slog.Warn("catalog: card has no variants, using normal", "lang", lang, "card", cb.ID)
	}
	if d.Variants != nil && d.Variants.WPromo {
		slog.Warn("catalog: wPromo variant is not stored", "lang", lang, "card", cb.ID)
	}
	for _, code := range codes {
		if _, err := q.UpsertCardVariant(ctx, dbq.UpsertCardVariantParams{CardID: cardID, Variant: code}); err != nil {
			return 0, fmt.Errorf("변형 %s: %w", code, err)
		}
	}
	return len(codes), nil
}

// fetchCards 는 카드 상세를 동시에 최대 concurrency 개씩 받는다. 404 인 카드는 결과에서 빠진다.
// 그 밖의 오류가 하나라도 나면 나머지 요청을 멈추고 오류를 돌려준다.
func (c *Collector) fetchCards(ctx context.Context, lang string, ids []string, concurrency int) (map[string]*tcgdex.Card, error) {
	out := make(map[string]*tcgdex.Card, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	var mu sync.Mutex
	var wg sync.WaitGroup
	jobs := make(chan string)
	for range min(concurrency, len(ids)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				card, err := c.api.GetCard(ctx, lang, id)
				if errors.Is(err, tcgdex.ErrNotFound) {
					slog.Warn("catalog: card detail not found", "lang", lang, "card", id)
					continue
				}
				if err != nil {
					cancel(fmt.Errorf("카드 %s 상세: %w", id, err))
					continue
				}
				mu.Lock()
				out[id] = card
				mu.Unlock()
			}
		}()
	}
send:
	for _, id := range ids {
		select {
		case jobs <- id:
		case <-ctx.Done():
			break send
		}
	}
	close(jobs)
	wg.Wait()
	if err := context.Cause(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
