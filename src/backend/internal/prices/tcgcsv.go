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
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Chocobone/PokeCardPrice/src/backend/internal/catalog"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/db/dbq"
	"github.com/Chocobone/PokeCardPrice/src/backend/internal/tcgcsv"
)

// tcgcsvCategories 는 수집할 TCGplayer 카테고리와 카드 언어이다.
var tcgcsvCategories = []struct {
	ID       int
	Language string
}{
	{tcgcsv.CategoryPokemon, "en"},
	{tcgcsv.CategoryPokemonJapan, "ja"},
}

// manualGroupSets 는 이름 코드로 대응시킬 수 없는 그룹 → TCGdex 세트 ID 이다 (Q1 수동 대응표).
// 확인한 그룹만 넣는다. 영어판은 방법 A 결과로 대응하므로 대부분 필요 없다.
var manualGroupSets = map[int]string{}

// groupSetMinShare 는 방법 A 결과로 그룹 ↔ 세트를 정할 때 가장 많은 세트가 차지해야 하는 비율이다.
const groupSetMinShare = 0.9

// q5Variants 는 TCGdex 에 없어도 TCGCSV 근거로 추가하는 변형이다 (Q5).
// 초판·무제한판은 판 구성 자체를 바꾸므로 추가하지 않는다.
var q5Variants = []string{catalog.VariantNormal, catalog.VariantHolofoil, catalog.VariantReverseHolofoil}

// 연결하지 않은 가격 행의 분류 (요약 로그).
const (
	skipNonCard   = "non_card"        // 카드가 아닌 상품 (번호 없음)
	skipNoSet     = "group_unmapped"  // 그룹 ↔ 세트 대응 없음
	skipSpecial   = "special_product" // 같은 번호의 특별 상품 (괄호)
	skipNoCard    = "card_not_found"  // 번호에 맞는 카드 없음 또는 여럿
	skipNoVariant = "variant_missing" // 카드에 맞는 변형 없음
	skipAmbiguous = "ambiguous"       // 여러 가격 행이 한 변형에 연결됨
	skipNoPrice   = "no_price"        // market·mid 모두 없음
)

// TCGCSV 는 TCGplayer 가격(TCGCSV)을 카드 변형에 연결해 price_history 에 저장한다.
//
//  1. 그룹별 상품·가격을 받고, 저장된 external_ids 로 연결되는 행을 바로 저장한다.
//  2. 같은 실행의 TCGdex 작업(방법 A 연결 갱신)이 끝나기를 기다린다.
//  3. 남은 행을 새 external_ids 로, 그다음 그룹 ↔ 세트 + 카드 번호(방법 B)로 연결해 저장한다.
//     방법 B 연결은 external_ids 에 저장해 다음 날부터 1번에서 바로 쓴다.
//
// price_date 는 TCGCSV 갱신 시각의 UTC 날짜, source_updated_at 은 갱신 시각이다.
// 대표값은 market, 없으면 mid 이다. 연결 규칙은 link.go, 근거는 price_collector.md 2절.
type TCGCSV struct {
	pool   *pgxpool.Pool
	client *tcgcsv.Client
}

// NewTCGCSV 는 TCGCSV 가격 작업을 만든다.
func NewTCGCSV(pool *pgxpool.Pool, client *tcgcsv.Client) *TCGCSV {
	return &TCGCSV{pool: pool, client: client}
}

// Code 는 작업 이름이다.
func (s *TCGCSV) Code() string { return TCGCSVCode }

// tcgcsvGroup 은 그룹 하나의 받은 데이터이다. 상품은 연결에 필요한 값만 남긴다 (2시간 넘게 들고 있어서).
type tcgcsvGroup struct {
	Category int
	Language string
	Group    tcgcsv.Group
	Products map[int]tcgcsvProduct
	Prices   []tcgcsv.Price
	pending  []int          // 아직 연결하지 못한 Prices 인덱스
	base     map[string]int // 번호 → 기본 상품 ID (baseProduct)
}

func (g *tcgcsvGroup) key() string { return g.Language + "/" + strconv.Itoa(g.Group.GroupID) }

// baseProduct 는 번호 num 의 기본 상품 ID 이다: 상품이 하나뿐이면 그것, 여럿이면 괄호 없는 하나. 없거나 정할 수 없으면 0.
func (g *tcgcsvGroup) baseProduct(num string) int {
	if g.base == nil {
		byNumber := map[string][]int{}
		for id, p := range g.Products {
			if p.Number != "" {
				byNumber[p.Number] = append(byNumber[p.Number], id)
			}
		}
		g.base = make(map[string]int, len(byNumber))
		for n, ids := range byNumber {
			found := 0
			if len(ids) == 1 {
				found = ids[0]
			} else {
				for _, id := range ids {
					if isSpecialProduct(g.Products[id].Name) {
						continue
					}
					if found != 0 {
						found = 0
						break
					}
					found = id
				}
			}
			g.base[n] = found
		}
	}
	return g.base[num]
}

// shadowed 는 같은 번호의 기본 상품이 따로 있는 상품(특별 상품)인지 알려 준다.
func (g *tcgcsvGroup) shadowed(productID int) bool {
	b := g.baseProduct(g.Products[productID].Number)
	return b != 0 && b != productID
}

type tcgcsvProduct struct {
	Name   string
	Number string // 정규화한 번호. 카드가 아니면 빈 문자열
}

// assignment 는 가격 행 하나를 변형에 연결한 결과이다.
type assignment struct {
	group *tcgcsvGroup
	price tcgcsv.Price
	// VariantID 가 0 이면 CardID 카드에 Variant 변형을 새로 만든다 (Q5).
	VariantID int64
	CardID    int64
	Variant   string
	// newLink 이면 external_ids 에 저장한다 (방법 B).
	newLink bool
}

func (a assignment) target() string {
	if a.VariantID != 0 {
		return "v" + strconv.FormatInt(a.VariantID, 10)
	}
	return "c" + strconv.FormatInt(a.CardID, 10) + ":" + a.Variant
}

// tcgcsvRun 은 실행 한 번의 상태이다.
type tcgcsvRun struct {
	q         *dbq.Queries
	pool      *pgxpool.Pool
	srcID     int16
	priceDate time.Time
	updatedAt time.Time
	res       *Result
	skips     map[string]int
	used      map[string]bool // 이번 실행에 저장한 변형 (assignment.target)
	created   int             // Q5 로 추가한 변형 수
}

// Collect 는 TCGCSV 가격을 받아 연결·저장한다. 그룹 단위로 실패를 가둔다.
func (s *TCGCSV) Collect(ctx context.Context, opt Options) (Result, error) {
	var res Result
	q := dbq.New(s.pool)
	src, err := q.GetPriceSourceByCode(ctx, TCGCSVCode)
	if err != nil {
		return res, fmt.Errorf("가격 소스 %s: %w", TCGCSVCode, err)
	}
	updated, err := s.client.LastUpdated(ctx)
	if err != nil {
		return res, err
	}
	run := &tcgcsvRun{
		q: q, pool: s.pool, srcID: src.ID,
		priceDate: time.Date(updated.Year(), updated.Month(), updated.Day(), 0, 0, 0, 0, time.UTC),
		updatedAt: updated,
		res:       &res,
		skips:     map[string]int{},
		used:      map[string]bool{},
	}
	slog.Info("tcgcsv start", "price_date", run.priceDate.Format(time.DateOnly), "updated_at", updated)

	groups, failed := s.fetch(ctx, opt)
	if err := ctx.Err(); err != nil {
		return res, err
	}
	res.Failed = append(res.Failed, failed...)
	for _, g := range groups {
		res.Fetched += len(g.Prices)
	}

	// 1. 저장된 연결로 저장
	links, err := loadLinks(ctx, q, src.ID)
	if err != nil {
		return res, err
	}
	var as []assignment
	for _, g := range groups {
		for i, p := range g.Prices {
			prod, ok := g.Products[p.ProductID]
			if !ok || prod.Number == "" {
				run.skip(skipNonCard)
				continue
			}
			if g.shadowed(p.ProductID) {
				// 같은 번호의 기본 상품이 따로 있는 특별 상품. 저장된 연결(방법 A 포함)이 있어도 쓰지 않는다.
				// TCGdex 가 특별 상품 ID 를 주는 경우가 있다 (svp-173 → Pokemon Center Exclusive).
				slog.Debug("tcgcsv special product skipped", "group", g.key(), "product", p.ProductID, "name", prod.Name)
				run.skip(skipSpecial)
				continue
			}
			if v, ok := links[linkKey{strconv.Itoa(p.ProductID), p.SubTypeName}]; ok {
				as = append(as, assignment{group: g, price: p, VariantID: v})
				continue
			}
			g.pending = append(g.pending, i)
		}
	}
	run.save(ctx, as)
	pending := 0
	for _, g := range groups {
		pending += len(g.pending)
	}
	slog.Info("tcgcsv phase 1 done", "groups", len(groups), "saved", res.Saved, "pending", pending)

	// 2. TCGdex 작업이 external_ids 를 갱신하기를 기다린다
	if pending > 0 {
		ran, err := opt.waitFor(ctx, CardmarketCode)
		if err != nil {
			return res, err
		}
		if ran {
			slog.Info("tcgcsv phase 2 after tcgdex details")
		}
	}

	// 3. 남은 행을 새 연결과 번호 대조로 저장
	if pending > 0 {
		as, err := run.linkPending(ctx, groups)
		if err != nil {
			return res, err
		}
		run.save(ctx, as)
	}

	reasons := slices.Sorted(maps.Keys(run.skips))
	attrs := make([]any, 0, 2*len(reasons)+2)
	for _, r := range reasons {
		attrs = append(attrs, r, run.skips[r])
	}
	attrs = append(attrs, "variants_created", run.created)
	slog.Info("tcgcsv skipped rows", attrs...)
	if len(res.Failed) > 0 {
		return res, fmt.Errorf("그룹 %d개 실패", len(res.Failed))
	}
	return res, nil
}

// fetch 는 대상 그룹의 상품·가격을 받는다. 실패한 그룹은 끝에 한 번 더 받고, 남은 실패를 돌려준다.
func (s *TCGCSV) fetch(ctx context.Context, opt Options) (groups []*tcgcsvGroup, failed []string) {
	var todo []*tcgcsvGroup
	for _, cat := range tcgcsvCategories {
		if !opt.wantLanguage(cat.Language) {
			continue
		}
		gs, err := s.client.Groups(ctx, cat.ID)
		if err != nil {
			slog.Warn("tcgcsv groups failed", "category", cat.ID, "err", err)
			failed = append(failed, cat.Language+"/groups")
			continue
		}
		for _, g := range gs {
			if wantGroup(opt, g) {
				todo = append(todo, &tcgcsvGroup{Category: cat.ID, Language: cat.Language, Group: g})
			}
		}
	}
	slog.Info("tcgcsv groups", "groups", len(todo))

	for attempt := 0; attempt < 2 && len(todo) > 0 && ctx.Err() == nil; attempt++ {
		if attempt > 0 {
			slog.Info("tcgcsv retry failed groups", "groups", len(todo))
		}
		var retry []*tcgcsvGroup
		for _, g := range todo {
			if ctx.Err() != nil {
				break
			}
			if err := s.fetchGroup(ctx, g); err != nil {
				slog.Warn("tcgcsv group failed", "group", g.key(), "err", err)
				retry = append(retry, g)
				continue
			}
			groups = append(groups, g)
		}
		todo = retry
	}
	for _, g := range todo {
		failed = append(failed, g.key())
	}
	return groups, failed
}

func (s *TCGCSV) fetchGroup(ctx context.Context, g *tcgcsvGroup) error {
	cat := g.Category
	products, err := s.client.Products(ctx, cat, g.Group.GroupID)
	if err != nil {
		return err
	}
	prices, err := s.client.Prices(ctx, cat, g.Group.GroupID)
	if err != nil {
		return err
	}
	g.Products = make(map[int]tcgcsvProduct, len(products))
	for _, p := range products {
		num := ""
		if n := p.Number(); n != "" {
			num = normalizeNumber(n)
		}
		g.Products[p.ProductID] = tcgcsvProduct{Name: p.Name, Number: num}
	}
	g.Prices = prices
	return nil
}

// wantGroup 은 개발용 범위(TCGCSVGroups, SetIDs)에 맞는 그룹인지 알려 준다. 범위가 없으면 모두.
func wantGroup(opt Options, g tcgcsv.Group) bool {
	if len(opt.SetIDs) == 0 && len(opt.TCGCSVGroups) == 0 {
		return true
	}
	if slices.Contains(opt.TCGCSVGroups, g.GroupID) {
		return true
	}
	code := g.Code()
	return code != "" && slices.ContainsFunc(opt.SetIDs, func(id string) bool { return strings.EqualFold(id, code) })
}

func (r *tcgcsvRun) skip(reason string) {
	r.skips[reason]++
	r.res.Skipped++
}

// linkPending 은 1단계에서 남은 행을 다시 읽은 external_ids, 그다음 번호 대조로 연결한다.
func (r *tcgcsvRun) linkPending(ctx context.Context, groups []*tcgcsvGroup) ([]assignment, error) {
	rows, err := r.q.ListExternalIDLinks(ctx, r.srcID)
	if err != nil {
		return nil, fmt.Errorf("external_ids: %w", err)
	}
	links := make(map[linkKey]int64, len(rows))
	productSet := make(map[string]string, len(rows)) // productId → "언어:세트ID"
	for _, row := range rows {
		links[linkKey{row.ExternalID, row.ExternalSubID}] = row.CardVariantID
		productSet[row.ExternalID] = row.Language + ":" + row.SetTcgdexID
	}
	groupSets, err := r.mapGroupSets(ctx, groups, productSet)
	if err != nil {
		return nil, err
	}

	var as []assignment
	setCards := map[string]map[string][]setCard{} // "언어:세트ID" → 번호 → 카드
	for _, g := range groups {
		if len(g.pending) == 0 {
			continue
		}
		set := groupSets[g.Group.GroupID]
		var cards map[string][]setCard
		if set != "" {
			if cards = setCards[set]; cards == nil {
				lang, id, _ := strings.Cut(set, ":")
				cards, err = r.loadSetCards(ctx, lang, id)
				if err != nil {
					return nil, err
				}
				setCards[set] = cards
			}
		}

		for _, i := range g.pending {
			p := g.Prices[i]
			if v, ok := links[linkKey{strconv.Itoa(p.ProductID), p.SubTypeName}]; ok {
				as = append(as, assignment{group: g, price: p, VariantID: v})
				continue
			}
			if set == "" {
				r.skip(skipNoSet)
				continue
			}
			prod := g.Products[p.ProductID]
			if g.baseProduct(prod.Number) != p.ProductID {
				r.skip(skipSpecial)
				continue
			}
			cs := cards[prod.Number]
			if len(cs) != 1 {
				slog.Debug("tcgcsv card not found", "group", g.key(), "product", p.ProductID, "name", prod.Name, "cards", len(cs))
				r.skip(skipNoCard)
				continue
			}
			c := cs[0]
			if v, ok := resolveVariant(g.Language, c.Variants, p.SubTypeName); ok {
				as = append(as, assignment{group: g, price: p, VariantID: v.ID, newLink: true})
				continue
			}
			// Q5: TCGdex 에 없는 변형을 추가한다 (영어판, 괄호 없는 기본 상품, 번호가 카드 하나에만 대응).
			code := subTypeVariant[p.SubTypeName]
			if g.Language == "en" && !isSpecialProduct(prod.Name) && slices.Contains(q5Variants, code) {
				as = append(as, assignment{group: g, price: p, CardID: c.CardID, Variant: code, newLink: true})
				continue
			}
			slog.Debug("tcgcsv variant missing", "group", g.key(), "product", p.ProductID, "name", prod.Name, "sub_type", p.SubTypeName)
			r.skip(skipNoVariant)
		}
	}
	return as, nil
}

// setCard 는 번호 대조용 카드이다.
type setCard struct {
	CardID   int64
	Variants []cardVariant
}

func (r *tcgcsvRun) loadSetCards(ctx context.Context, lang, setID string) (map[string][]setCard, error) {
	rows, err := r.q.ListSetCardVariants(ctx, dbq.ListSetCardVariantsParams{Language: lang, SetTcgdexID: setID})
	if err != nil {
		return nil, fmt.Errorf("세트 %s:%s 카드: %w", lang, setID, err)
	}
	byID := map[int64]*setCard{}
	var order []int64
	numbers := map[int64]string{}
	for _, row := range rows {
		c, ok := byID[row.CardID]
		if !ok {
			c = &setCard{CardID: row.CardID}
			byID[row.CardID] = c
			order = append(order, row.CardID)
			numbers[row.CardID] = normalizeNumber(row.LocalID)
		}
		c.Variants = append(c.Variants, cardVariant{ID: row.CardVariantID, Variant: row.Variant})
	}
	out := map[string][]setCard{}
	for _, id := range order {
		out[numbers[id]] = append(out[numbers[id]], *byID[id])
	}
	return out, nil
}

// mapGroupSets 는 그룹 → "언어:세트ID" 이다. 대응이 없는 그룹은 맵에 없다.
//
//  1. 수동 대응표
//  2. 그룹 상품 중 external_ids 로 연결된 상품의 세트 (가장 많은 세트가 90% 이상일 때)
//  3. 그룹 이름 앞 코드가 같은 언어의 세트 ID 와 같음 (대소문자 무시, 하나뿐일 때)
func (r *tcgcsvRun) mapGroupSets(ctx context.Context, groups []*tcgcsvGroup, productSet map[string]string) (map[int]string, error) {
	setIDs := map[string][]string{} // 언어 → 세트 ID
	out := map[int]string{}
	var byManual, byLinks, byCode, none int
	for _, g := range groups {
		if len(g.pending) == 0 {
			continue
		}
		if id, ok := manualGroupSets[g.Group.GroupID]; ok {
			out[g.Group.GroupID] = g.Language + ":" + id
			byManual++
			continue
		}
		counts := map[string]int{}
		total := 0
		for id := range g.Products {
			if set, ok := productSet[strconv.Itoa(id)]; ok {
				counts[set]++
				total++
			}
		}
		if total > 0 {
			top := ""
			for set, n := range counts {
				if top == "" || n > counts[top] || (n == counts[top] && set < top) {
					top = set
				}
			}
			if float64(counts[top]) >= groupSetMinShare*float64(total) && strings.HasPrefix(top, g.Language+":") {
				out[g.Group.GroupID] = top
				byLinks++
				continue
			}
			slog.Warn("tcgcsv group maps to several sets", "group", g.key(), "name", g.Group.Name, "sets", counts)
		}
		if code := g.Group.Code(); code != "" {
			ids, ok := setIDs[g.Language]
			if !ok {
				var err error
				if ids, err = r.q.ListSetTcgdexIDs(ctx, g.Language); err != nil {
					return nil, fmt.Errorf("세트 목록: %w", err)
				}
				setIDs[g.Language] = ids
			}
			var match []string
			for _, id := range ids {
				if strings.EqualFold(id, code) {
					match = append(match, id)
				}
			}
			if len(match) == 1 {
				out[g.Group.GroupID] = g.Language + ":" + match[0]
				byCode++
				continue
			}
		}
		none++
		slog.Debug("tcgcsv group unmapped", "group", g.key(), "name", g.Group.Name)
	}
	slog.Info("tcgcsv group set mapping", "manual", byManual, "links", byLinks, "code", byCode, "unmapped", none)
	return out, nil
}

// save 는 연결 결과를 그룹 단위 트랜잭션으로 저장한다.
// 여러 행이 한 변형에 연결되면(이번 실행에 이미 저장한 변형 포함) 어느 값이 맞는지 알 수 없어 모두 저장하지 않는다.
func (r *tcgcsvRun) save(ctx context.Context, as []assignment) {
	count := map[string]int{}
	for _, a := range as {
		count[a.target()]++
	}
	byGroup := map[*tcgcsvGroup][]assignment{}
	var order []*tcgcsvGroup
	for _, a := range as {
		t := a.target()
		if count[t] > 1 || r.used[t] {
			slog.Debug("tcgcsv ambiguous", "group", a.group.key(), "product", a.price.ProductID, "sub_type", a.price.SubTypeName, "target", t)
			r.skip(skipAmbiguous)
			continue
		}
		r.res.Linked++
		if _, ok := byGroup[a.group]; !ok {
			order = append(order, a.group)
		}
		byGroup[a.group] = append(byGroup[a.group], a)
	}
	for _, g := range order {
		if ctx.Err() != nil {
			return
		}
		saved, noPrice, created, err := r.saveGroup(ctx, byGroup[g])
		if err != nil {
			slog.Warn("tcgcsv group save failed", "group", g.key(), "err", err)
			r.res.Failed = append(r.res.Failed, g.key())
			continue
		}
		for _, a := range byGroup[g] {
			r.used[a.target()] = true
		}
		r.res.Saved += saved
		r.created += created
		for range noPrice {
			r.skip(skipNoPrice)
		}
	}
}

func (r *tcgcsvRun) saveGroup(ctx context.Context, as []assignment) (saved, noPrice, created int, err error) {
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		saved, noPrice, created = 0, 0, 0
		q := r.q.WithTx(tx)
		var params []dbq.UpsertPriceHistoryParams
		for _, a := range as {
			vid := a.VariantID
			if vid == 0 {
				id, err := q.UpsertCardVariant(ctx, dbq.UpsertCardVariantParams{CardID: a.CardID, Variant: a.Variant})
				if err != nil {
					return fmt.Errorf("변형 추가: %w", err)
				}
				vid = id
				created++
				slog.Debug("tcgcsv variant created", "card", a.CardID, "variant", a.Variant, "product", a.price.ProductID)
			}
			if a.newLink {
				if err := q.UpsertExternalID(ctx, dbq.UpsertExternalIDParams{
					SourceID: r.srcID, ExternalID: strconv.Itoa(a.price.ProductID), ExternalSubID: a.price.SubTypeName, CardVariantID: vid,
				}); err != nil {
					return fmt.Errorf("external_ids 저장: %w", err)
				}
			}
			price, basis, ok := representative(a.price)
			if !ok {
				noPrice++
				continue
			}
			updated := r.updatedAt
			params = append(params, dbq.UpsertPriceHistoryParams{
				CardVariantID: vid, SourceID: r.srcID, PriceDate: r.priceDate,
				Price: price, PriceBasis: basis, SourceUpdatedAt: &updated,
				Low: a.price.LowPrice, Mid: a.price.MidPrice, High: a.price.HighPrice,
				Market: a.price.MarketPrice, DirectLow: a.price.DirectLowPrice,
			})
		}
		var errs []error
		q.UpsertPriceHistory(ctx, params).Exec(func(_ int, err error) {
			if err != nil {
				errs = append(errs, err)
			}
		})
		if err := errors.Join(errs...); err != nil {
			return fmt.Errorf("price_history 저장: %w", err)
		}
		saved = len(params)
		return nil
	})
	return saved, noPrice, created, err
}

// representative 는 대표값이다: market, 없으면 mid (db_schema.md 6절). 0 이하는 값이 없는 것으로 본다.
func representative(p tcgcsv.Price) (decimal.Decimal, string, bool) {
	if p.MarketPrice != nil && p.MarketPrice.IsPositive() {
		return *p.MarketPrice, "market", true
	}
	if p.MidPrice != nil && p.MidPrice.IsPositive() {
		return *p.MidPrice, "mid", true
	}
	return decimal.Decimal{}, "", false
}
