-- 가격: 가격 소스, 외부 ID, 시세 이력(넓은 형태), 환율.
-- 설계 근거: mydocs/tech/db_schema.md 6~8절

-- +goose Up
-- 가격 소스. 가격 성격과 통화는 소스마다 하나로 정해진다.
CREATE TABLE price_sources (
  id          smallint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  code        text NOT NULL CONSTRAINT price_sources_code_key UNIQUE,
  name        text NOT NULL,
  price_kind  text NOT NULL CONSTRAINT price_sources_price_kind_check CHECK (price_kind IN ('sale', 'jp_reference', 'user', 'admin')),
  currency    text NOT NULL CONSTRAINT price_sources_currency_check CHECK (currency IN ('USD', 'EUR', 'KRW')),
  attribution text
);

-- jp_reference 는 조회 시 계산하므로 price_history 에 행을 만들지 않는다(표시용).
INSERT INTO price_sources (code, name, price_kind, currency, attribution) VALUES
  ('tcgcsv',            'TCGplayer (TCGCSV)',  'sale',         'USD', 'TCGplayer 가격 (TCGCSV 경유)'),
  ('tcgdex_cardmarket', 'Cardmarket (TCGdex)', 'sale',         'EUR', 'Cardmarket 가격 (TCGdex 경유)'),
  ('jp_reference',      '일본판 환산 참고가',  'jp_reference', 'KRW', '일본판 가격 원화 환산 (Frankfurter 환율)'),
  ('user_aggregate',    '사용자 등록가 집계',  'user',         'KRW', NULL),
  ('admin',             '관리자 입력가',       'admin',        'KRW', NULL);

-- 소스별 외부 상품 ID → 카드 변형. TCGplayer 는 productId + subTypeName.
CREATE TABLE external_ids (
  source_id       smallint NOT NULL REFERENCES price_sources (id),
  external_id     text NOT NULL,
  external_sub_id text NOT NULL DEFAULT '',
  card_variant_id bigint NOT NULL REFERENCES card_variants (id) ON DELETE CASCADE,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (source_id, external_id, external_sub_id)
);
CREATE INDEX external_ids_card_variant_id_idx ON external_ids (card_variant_id);

-- 날짜별 시세 이력. 변형·소스·소스 기준일마다 1행.
-- price 는 대표값이고 price_basis 는 그 값을 가져온 필드이다. 나머지 필드는 원래 값 그대로 둔다.
CREATE TABLE price_history (
  card_variant_id   bigint NOT NULL REFERENCES card_variants (id) ON DELETE CASCADE,
  price_date        date NOT NULL,
  source_id         smallint NOT NULL REFERENCES price_sources (id),
  sample_count      integer,
  collected_at      timestamptz NOT NULL DEFAULT now(),
  source_updated_at timestamptz,
  price             numeric(12,2) NOT NULL,
  price_basis       text NOT NULL,
  low               numeric(12,2),
  mid               numeric(12,2),
  high              numeric(12,2),
  market            numeric(12,2),
  direct_low        numeric(12,2),
  avg               numeric(12,2),
  trend             numeric(12,2),
  avg1              numeric(12,2),
  avg7              numeric(12,2),
  avg30             numeric(12,2),
  PRIMARY KEY (card_variant_id, source_id, price_date),
  CONSTRAINT price_history_price_basis_check CHECK (price_basis IN ('market', 'mid', 'low', 'trend', 'avg', 'median', 'manual')),
  CONSTRAINT price_history_price_check CHECK (price >= 0),
  CONSTRAINT price_history_sample_count_check CHECK (sample_count IS NULL OR sample_count > 0)
);
-- 날짜별 순위·변동률 조회용 (Index Only Scan)
CREATE INDEX price_history_source_id_price_date_idx ON price_history (source_id, price_date) INCLUDE (card_variant_id, price);

-- 환율. 값이 없는 날은 기본 키 역순 스캔으로 직전 값을 쓴다.
CREATE TABLE fx_rates (
  base_currency  text NOT NULL CONSTRAINT fx_rates_base_currency_check CHECK (base_currency IN ('USD', 'EUR', 'KRW')),
  quote_currency text NOT NULL CONSTRAINT fx_rates_quote_currency_check CHECK (quote_currency IN ('USD', 'EUR', 'KRW')),
  rate_date      date NOT NULL,
  provider       text NOT NULL CONSTRAINT fx_rates_provider_check CHECK (provider IN ('frankfurter', 'koreaexim')),
  rate           numeric(18,8) NOT NULL CONSTRAINT fx_rates_rate_check CHECK (rate > 0),
  fetched_at     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (base_currency, quote_currency, rate_date, provider),
  CONSTRAINT fx_rates_pair_check CHECK (base_currency <> quote_currency)
);

-- +goose Down
DROP TABLE fx_rates;
DROP TABLE price_history;
DROP TABLE external_ids;
DROP TABLE price_sources;
