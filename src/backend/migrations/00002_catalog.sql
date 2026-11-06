-- 카탈로그: 세트, 카드, 카드 변형, 카드 간 연결, 포켓몬 종 이름.
-- 설계 근거: mydocs/tech/db_schema.md 6절

-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 언어별 세트. 일본판·한글판은 TCGdex 세트 ID가 같다(SV5K).
CREATE TABLE sets (
  id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  language     text NOT NULL CONSTRAINT sets_language_check CHECK (language IN ('en', 'ja', 'ko')),
  tcgdex_id    text NOT NULL,
  name         text NOT NULL,
  release_date date,
  card_count   integer,
  logo_url     text,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT sets_language_tcgdex_id_key UNIQUE (language, tcgdex_id),
  -- cards 의 복합 외래 키 대상
  CONSTRAINT sets_id_language_key UNIQUE (id, language)
);

-- 카드. language 는 세트와 같아야 한다(복합 외래 키).
CREATE TABLE cards (
  id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  set_id         bigint NOT NULL,
  language       text NOT NULL,
  tcgdex_id      text NOT NULL,
  local_id       text NOT NULL,
  name           text NOT NULL,
  rarity         text,
  image_url      text,
  dex_ids        integer[],
  -- 일본판 카드에 같은 세트 코드·번호의 한글판이 있음을 표시 (TCGdex ko 미수록 한글판)
  has_ko_edition boolean NOT NULL DEFAULT false,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT cards_set_id_language_fkey FOREIGN KEY (set_id, language) REFERENCES sets (id, language) ON DELETE RESTRICT,
  CONSTRAINT cards_language_tcgdex_id_key UNIQUE (language, tcgdex_id),
  CONSTRAINT cards_set_id_local_id_key UNIQUE (set_id, local_id),
  CONSTRAINT cards_has_ko_edition_check CHECK (NOT has_ko_edition OR language = 'ja')
);
CREATE INDEX cards_name_trgm_idx ON cards USING gin (name gin_trgm_ops);
CREATE INDEX cards_dex_ids_idx ON cards USING gin (dex_ids);

-- 카드 변형. 가격의 기준 단위.
CREATE TABLE card_variants (
  id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  card_id    bigint NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
  variant    text NOT NULL CONSTRAINT card_variants_variant_check CHECK (variant IN (
               'normal', 'holofoil', 'reverse_holofoil', 'first_edition',
               'first_edition_holofoil', 'unlimited', 'unlimited_holofoil')),
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT card_variants_card_id_variant_key UNIQUE (card_id, variant)
);

-- 한글판 카드 → 일본판 카드 연결. 방향(ko → ja)은 수집기가 확인한다.
CREATE TABLE card_links (
  card_id        bigint PRIMARY KEY REFERENCES cards (id) ON DELETE CASCADE,
  linked_card_id bigint NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
  method         text NOT NULL CONSTRAINT card_links_method_check CHECK (method IN ('auto', 'manual')),
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT card_links_not_self_check CHECK (card_id <> linked_card_id)
);
CREATE INDEX card_links_linked_card_id_idx ON card_links (linked_card_id);

-- 포켓몬 종 이름. 한글 종 이름으로 cards.dex_ids 를 찾는다. 데이터 적재는 #5.
CREATE TABLE pokemon_species (
  dex_id     integer PRIMARY KEY CONSTRAINT pokemon_species_dex_id_check CHECK (dex_id > 0),
  name_en    text NOT NULL,
  name_ja    text,
  name_ko    text,
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX pokemon_species_name_ko_trgm_idx ON pokemon_species USING gin (name_ko gin_trgm_ops);

-- +goose Down
DROP TABLE pokemon_species;
DROP TABLE card_links;
DROP TABLE card_variants;
DROP TABLE cards;
DROP TABLE sets;
DROP EXTENSION IF EXISTS pg_trgm;
