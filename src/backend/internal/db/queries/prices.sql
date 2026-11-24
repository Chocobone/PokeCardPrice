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

-- name: ListExternalIDLinks :many
-- 소스의 외부 ID 전체와 연결된 변형의 세트·카드 번호
-- (#6 TCGCSV: 저장된 연결 재사용, 상품 번호 확인, 그룹 ↔ 세트 대응).
SELECT e.external_id, e.external_sub_id, e.card_variant_id, s.language, s.tcgdex_id AS set_tcgdex_id,
       c.tcgdex_id AS card_tcgdex_id, c.local_id
FROM external_ids e
JOIN card_variants v ON v.id = e.card_variant_id
JOIN cards c ON c.id = v.card_id
JOIN sets s ON s.id = c.set_id
WHERE e.source_id = @source_id;

-- name: DeleteExternalIDLink :execrows
-- 외부 ID 연결을 지운다. 그사이 다른 변형으로 바뀌었으면 지우지 않는다 (#6 상품 번호가 맞지 않는 연결).
DELETE FROM external_ids
WHERE source_id = @source_id AND external_id = @external_id AND external_sub_id = @external_sub_id
  AND card_variant_id = @card_variant_id;

-- name: DeleteVariantSourcePrices :execrows
-- 변형 하나의 소스 가격 중 before_date 보다 이른 날을 지운다 (#6 잘못된 연결로 저장된 가격 정리).
DELETE FROM price_history
WHERE card_variant_id = @card_variant_id AND source_id = @source_id AND price_date < @before_date;

-- name: ListSetCardVariants :many
-- 세트의 카드와 변형 (#6 TCGCSV 번호 대조).
SELECT c.id AS card_id, c.local_id, v.id AS card_variant_id, v.variant
FROM cards c
JOIN sets s ON s.id = c.set_id
JOIN card_variants v ON v.card_id = c.id
WHERE s.language = @language AND s.tcgdex_id = @set_tcgdex_id
ORDER BY c.id, v.id;

-- name: ListSetTcgdexIDs :many
-- 언어의 세트 ID 전체 (#6 TCGCSV 그룹 코드 대응).
SELECT tcgdex_id FROM sets WHERE language = @language ORDER BY tcgdex_id;

-- name: ListPriceTargetCards :many
-- 시세 수집 대상 카드와 변형 (#6 TCGdex 카드 상세 순회).
-- set_tcgdex_ids 가 비어 있으면 모든 세트. exclude_sets 는 "언어:세트ID" 목록이다 (TCG Pocket 제외, Q6).
SELECT c.language, c.tcgdex_id, s.tcgdex_id AS set_tcgdex_id,
       array_agg(v.variant ORDER BY v.id)::text[] AS variants,
       array_agg(v.id ORDER BY v.id)::bigint[] AS variant_ids
FROM cards c
JOIN sets s ON s.id = c.set_id
JOIN card_variants v ON v.card_id = c.id
WHERE c.language = ANY(@languages::text[])
  AND (cardinality(@set_tcgdex_ids::text[]) = 0 OR s.tcgdex_id = ANY(@set_tcgdex_ids::text[]))
  AND NOT ((c.language || ':' || s.tcgdex_id) = ANY(@exclude_sets::text[]))
GROUP BY c.id, s.tcgdex_id
ORDER BY c.language, s.tcgdex_id, c.id;

-- name: ListPriceHistoryKRW :many
-- 변형 1개·소스 1개의 기간 추이를 원화로 환산한다 (#6, #8 시세 조회의 출발점).
-- 환율은 price_date 이전 가장 가까운 값이다 (주말·공휴일·수집 누락일은 직전 값, fx_rates 기본 키 역순 스캔).
-- 환율이 없으면(첫 환율보다 이른 날, 소스 통화가 KRW) fx_* 가 NULL 이다.
-- 원화 값은 호출 측에서 decimal 로 price * fx_rate 를 원 단위 반올림한다 (SQL 계산 결과는 sqlc 가 decimal 로 만들지 않는다).
SELECT p.price_date, p.price, p.price_basis, s.currency, p.source_updated_at,
       f.rate_date AS fx_rate_date, f.rate AS fx_rate
FROM price_history p
JOIN price_sources s ON s.id = p.source_id
-- 직전 환율 날짜는 LATERAL(기본 키 역순 스캔)로 구하고, 값은 그 날짜로 기본 키 조회한다.
-- LATERAL 결과 열은 sqlc 가 NULL 불가로 만들어서, 값은 테이블 LEFT JOIN 으로 가져온다.
LEFT JOIN LATERAL (
  SELECT f2.rate_date AS on_date
  FROM fx_rates f2
  WHERE f2.base_currency = s.currency AND f2.quote_currency = 'KRW'
    AND f2.provider = @fx_provider AND f2.rate_date <= p.price_date
  ORDER BY f2.rate_date DESC
  LIMIT 1
) last ON true
LEFT JOIN fx_rates f
  ON f.base_currency = s.currency AND f.quote_currency = 'KRW' AND f.provider = @fx_provider
 AND f.rate_date = last.on_date
WHERE p.card_variant_id = @card_variant_id AND p.source_id = @source_id AND p.price_date >= @from_date
ORDER BY p.price_date;
