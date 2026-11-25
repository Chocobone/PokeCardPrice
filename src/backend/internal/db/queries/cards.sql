-- 카드 검색·시세 조회 API(#8) 쿼리. 규칙: mydocs/tech/api_contract.md, 측정: mydocs/working/task_m050_8_stage1.md

-- name: SearchCards :many
-- 모든 낱말이 맞는 카드를 찾는다. 낱말이 맞는 조건은 카드 이름·세트 이름 부분 일치, 세트 ID·카드 번호(앞 0 무시) 일치,
-- 한글 종 이름 부분 일치(도감 번호) 중 하나이다.
-- terms 는 소문자 낱말, key_term 은 그중 후보를 찾을 낱말(가장 긴 것), key_pattern 은 key_term 을 LIKE 용으로 감싼 값,
-- query 는 낱말을 공백 하나로 이은 전체 검색어이다. language 가 '' 이면 모든 언어.
-- 정렬: ① 세트 ID·번호가 낱말과 같음 ② 이름이 검색어로 시작하거나 첫 도감 번호의 한글 종 이름이 낱말과 같음 ③ 나머지,
-- 그 안에서 세트 발매일 최신순 → 세트 ID → 번호.
WITH t AS (
  SELECT term, ltrim(term, '0') AS num,
         (SELECT coalesce(array_agg(ps.dex_id), '{}')::int[] FROM pokemon_species ps WHERE strpos(ps.name_ko, term) > 0) AS dex,
         (SELECT coalesce(array_agg(st.id), '{}')::bigint[] FROM sets st WHERE strpos(lower(st.name), term) > 0 OR lower(st.tcgdex_id) = term) AS set_ids
  FROM unnest(@terms::text[]) AS term
),
k AS (SELECT * FROM t WHERE t.term = @key_term::text),
cand AS (
  SELECT c1.id FROM cards c1, k WHERE lower(c1.name) LIKE @key_pattern::text
  UNION SELECT c2.id FROM cards c2, k WHERE c2.set_id = ANY(k.set_ids)
  UNION SELECT c3.id FROM cards c3, k WHERE k.num <> '' AND ltrim(c3.local_id, '0') = k.num
  UNION SELECT c4.id FROM cards c4, k WHERE c4.dex_ids && k.dex
)
SELECT c.id, c.language, c.tcgdex_id, c.name, c.local_id, c.image_url, c.has_ko_edition,
       s.tcgdex_id AS set_tcgdex_id, s.name AS set_name, p.name_ko AS species_name_ko,
       count(*) OVER () AS total
FROM cand
JOIN cards c ON c.id = cand.id
JOIN sets s ON s.id = c.set_id
LEFT JOIN pokemon_species p ON p.dex_id = c.dex_ids[1]
WHERE (@language::text = '' OR c.language = @language::text)
  AND NOT EXISTS (
    SELECT 1 FROM t
    WHERE NOT (
      strpos(lower(c.name), t.term) > 0
      OR c.set_id = ANY(t.set_ids)
      OR (t.num <> '' AND ltrim(c.local_id, '0') = t.num)
      -- 트레이너스·에너지는 dex_ids 가 NULL 이다. NULL 이 NOT 을 거쳐 '맞음'이 되지 않게 막는다.
      OR coalesce(c.dex_ids && t.dex, false)
    )
  )
ORDER BY
  CASE
    WHEN EXISTS (SELECT 1 FROM t WHERE lower(s.tcgdex_id) = t.term OR (t.num <> '' AND ltrim(c.local_id, '0') = t.num)) THEN 0
    WHEN strpos(lower(c.name), @query::text) = 1 OR EXISTS (SELECT 1 FROM t WHERE p.name_ko = t.term) THEN 1
    ELSE 2
  END,
  s.release_date DESC NULLS LAST, s.tcgdex_id, c.local_id
LIMIT @lim OFFSET @off;

-- name: ListLatestPrices :many
-- 카드들의 변형 × 판매가 소스별 최신가와 원화 환산용 직전 환율 (대표가·카드 상세).
-- 최신 날짜는 LATERAL(기본 키 역순 스캔)로 구하고 값은 기본 키로 다시 읽는다.
-- LATERAL 결과 열은 sqlc 가 NULL 불가로 만들어서, NULL 이 될 수 있는 열은 테이블 JOIN 으로 가져온다 (ListPriceHistoryKRW 와 같은 방식).
SELECT v.card_id, v.id AS card_variant_id, v.variant, src.code AS source, src.currency,
       ph.price_date, ph.price, ph.source_updated_at,
       f.rate_date AS fx_rate_date, f.rate AS fx_rate
FROM card_variants v
CROSS JOIN price_sources src
JOIN LATERAL (
  SELECT p2.price_date AS on_date FROM price_history p2
  WHERE p2.card_variant_id = v.id AND p2.source_id = src.id
  ORDER BY p2.price_date DESC
  LIMIT 1
) latest ON true
JOIN price_history ph ON ph.card_variant_id = v.id AND ph.source_id = src.id AND ph.price_date = latest.on_date
LEFT JOIN LATERAL (
  SELECT f2.rate_date AS on_date FROM fx_rates f2
  WHERE f2.base_currency = src.currency AND f2.quote_currency = 'KRW'
    AND f2.provider = @fx_provider AND f2.rate_date <= ph.price_date
  ORDER BY f2.rate_date DESC
  LIMIT 1
) fxd ON true
LEFT JOIN fx_rates f
  ON f.base_currency = src.currency AND f.quote_currency = 'KRW' AND f.provider = @fx_provider
 AND f.rate_date = fxd.on_date
WHERE v.card_id = ANY(@card_ids::bigint[]) AND src.price_kind = 'sale'
ORDER BY v.card_id, v.id, src.id;
