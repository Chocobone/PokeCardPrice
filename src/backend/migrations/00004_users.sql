-- 사용자: 사용자, 보유 카드, 사용자 거래가, 관리자 입력가.
-- 설계 근거: mydocs/tech/db_schema.md 6절. 인증 관련 추가 컬럼·테이블은 #9.

-- +goose Up
CREATE TABLE users (
  id            uuid PRIMARY KEY DEFAULT uuidv7(),
  email         text NOT NULL,
  password_hash text NOT NULL,
  display_name  text NOT NULL,
  role          text NOT NULL DEFAULT 'user' CONSTRAINT users_role_check CHECK (role IN ('user', 'admin')),
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);
-- 대소문자를 구분하지 않고 이메일 중복을 막는다.
CREATE UNIQUE INDEX users_email_key ON users (lower(email));

-- 보유 카드. 같은 변형·상태 등급은 수량으로 관리한다.
-- 상태 등급은 TCGplayer 기준 5단계, 감정 카드는 다루지 않는다.
CREATE TABLE collection_items (
  id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id         uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  card_variant_id bigint NOT NULL REFERENCES card_variants (id) ON DELETE RESTRICT,
  condition       text NOT NULL CONSTRAINT collection_items_condition_check CHECK (condition IN (
                    'near_mint', 'lightly_played', 'moderately_played', 'heavily_played', 'damaged')),
  quantity        integer NOT NULL CONSTRAINT collection_items_quantity_check CHECK (quantity > 0),
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT collection_items_user_variant_condition_key UNIQUE (user_id, card_variant_id, condition)
);

-- 사용자 거래가 원본. #7 이 집계해 price_history(user_aggregate)에 넣는다. 판매자 정보는 저장하지 않는다.
CREATE TABLE user_trades (
  id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id             uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  card_variant_id     bigint NOT NULL REFERENCES card_variants (id) ON DELETE RESTRICT,
  price_krw           numeric(12,0) NOT NULL CONSTRAINT user_trades_price_krw_check CHECK (price_krw > 0),
  trade_date          date NOT NULL,
  condition           text NOT NULL CONSTRAINT user_trades_condition_check CHECK (condition IN (
                        'near_mint', 'lightly_played', 'moderately_played', 'heavily_played', 'damaged')),
  verification_status text NOT NULL DEFAULT 'pending' CONSTRAINT user_trades_verification_status_check CHECK (
                        verification_status IN ('pending', 'accepted', 'outlier', 'rejected')),
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX user_trades_card_variant_id_trade_date_idx ON user_trades (card_variant_id, trade_date);
CREATE INDEX user_trades_user_id_idx ON user_trades (user_id);

-- 관리자 입력가 원본(출처·확인일). 시세 값은 price_history(admin)에도 넣는다.
CREATE TABLE admin_price_entries (
  id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  card_variant_id bigint NOT NULL REFERENCES card_variants (id) ON DELETE RESTRICT,
  price_krw       numeric(12,0) NOT NULL CONSTRAINT admin_price_entries_price_krw_check CHECK (price_krw > 0),
  source_note     text NOT NULL,
  source_url      text,
  checked_date    date NOT NULL,
  created_by      uuid REFERENCES users (id) ON DELETE SET NULL,
  created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX admin_price_entries_card_variant_id_checked_date_idx ON admin_price_entries (card_variant_id, checked_date);

-- +goose Down
DROP TABLE admin_price_entries;
DROP TABLE user_trades;
DROP TABLE collection_items;
DROP TABLE users;
