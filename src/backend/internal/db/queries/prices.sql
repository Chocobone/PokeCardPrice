-- 가격·환율 기본 쿼리. 시세 수집기(#6), 국내 거래가(#7), 시세 API(#8)의 출발점이다.

-- name: GetPriceSourceByCode :one
SELECT * FROM price_sources WHERE code = @code;

-- name: UpsertExternalID :exec
INSERT INTO external_ids (source_id, external_id, external_sub_id, card_variant_id)
VALUES (@source_id, @external_id, @external_sub_id, @card_variant_id)
ON CONFLICT (source_id, external_id, external_sub_id) DO UPDATE
SET card_variant_id = EXCLUDED.card_variant_id,
    updated_at = now();

-- name: FindVariantByExternalID :one
SELECT card_variant_id FROM external_ids
WHERE source_id = @source_id AND external_id = @external_id AND external_sub_id = @external_sub_id;

-- name: UpsertPriceHistory :batchexec
-- 변형·소스·소스 기준일이 같으면 값을 갱신한다. 수집기를 다시 실행해도 행이 늘지 않는다.
INSERT INTO price_history (
  card_variant_id, source_id, price_date, price, price_basis, sample_count, source_updated_at,
  low, mid, high, market, direct_low, avg, trend, avg1, avg7, avg30
) VALUES (
  @card_variant_id, @source_id, @price_date, @price, @price_basis, @sample_count, @source_updated_at,
  @low, @mid, @high, @market, @direct_low, @avg, @trend, @avg1, @avg7, @avg30
)
ON CONFLICT (card_variant_id, source_id, price_date) DO UPDATE
SET price = EXCLUDED.price,
    price_basis = EXCLUDED.price_basis,
    sample_count = EXCLUDED.sample_count,
    source_updated_at = EXCLUDED.source_updated_at,
    collected_at = now(),
    low = EXCLUDED.low,
    mid = EXCLUDED.mid,
    high = EXCLUDED.high,
    market = EXCLUDED.market,
    direct_low = EXCLUDED.direct_low,
    avg = EXCLUDED.avg,
    trend = EXCLUDED.trend,
    avg1 = EXCLUDED.avg1,
    avg7 = EXCLUDED.avg7,
    avg30 = EXCLUDED.avg30;

-- name: ListPriceHistory :many
-- 변형 1개·소스 1개의 기간 추이 (기본 키 범위 스캔).
SELECT price_date, price, price_basis
FROM price_history
WHERE card_variant_id = @card_variant_id AND source_id = @source_id AND price_date >= @from_date
ORDER BY price_date;

-- name: UpsertFxRate :exec
INSERT INTO fx_rates (base_currency, quote_currency, rate_date, provider, rate)
VALUES (@base_currency, @quote_currency, @rate_date, @provider, @rate)
ON CONFLICT (base_currency, quote_currency, rate_date, provider) DO UPDATE
SET rate = EXCLUDED.rate,
    fetched_at = now();

-- name: GetFxRateOnOrBefore :one
-- 주말·공휴일처럼 그날 값이 없으면 직전 값을 돌려준다.
SELECT rate_date, rate
FROM fx_rates
WHERE base_currency = @base_currency AND quote_currency = @quote_currency
  AND provider = @provider AND rate_date <= @on_date
ORDER BY rate_date DESC
LIMIT 1;
