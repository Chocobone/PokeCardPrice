-- 카탈로그 기본 쿼리. 수집기(#5)와 카드 API(#8)의 출발점이다.

-- name: UpsertSet :one
INSERT INTO sets (language, tcgdex_id, name, release_date, card_count, logo_url)
VALUES (@language, @tcgdex_id, @name, @release_date, @card_count, @logo_url)
ON CONFLICT (language, tcgdex_id) DO UPDATE
SET name = EXCLUDED.name,
    release_date = EXCLUDED.release_date,
    card_count = EXCLUDED.card_count,
    logo_url = EXCLUDED.logo_url,
    updated_at = now()
RETURNING id;

-- name: UpsertCard :one
-- has_ko_edition 은 한글판 연결 작업이 따로 갱신하므로 여기서 바꾸지 않는다.
INSERT INTO cards (set_id, language, tcgdex_id, local_id, name, rarity, image_url, dex_ids)
VALUES (@set_id, @language, @tcgdex_id, @local_id, @name, @rarity, @image_url, @dex_ids)
ON CONFLICT (language, tcgdex_id) DO UPDATE
SET set_id = EXCLUDED.set_id,
    local_id = EXCLUDED.local_id,
    name = EXCLUDED.name,
    rarity = EXCLUDED.rarity,
    image_url = EXCLUDED.image_url,
    dex_ids = EXCLUDED.dex_ids,
    updated_at = now()
RETURNING id;

-- name: UpsertCardVariant :one
-- 이미 있으면 기존 ID 를 돌려준다 (DO NOTHING 은 RETURNING 행이 없어 DO UPDATE 를 쓴다).
INSERT INTO card_variants (card_id, variant)
VALUES (@card_id, @variant)
ON CONFLICT (card_id, variant) DO UPDATE SET variant = EXCLUDED.variant
RETURNING id;

-- name: GetCard :one
SELECT * FROM cards WHERE id = @id;

-- name: ListCardVariants :many
SELECT * FROM card_variants WHERE card_id = @card_id ORDER BY id;
