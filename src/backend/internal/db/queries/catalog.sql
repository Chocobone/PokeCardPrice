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

-- name: ListCardTcgdexIDsBySet :many
-- 카탈로그 수집기(#5): 이미 적재한 카드를 골라내 신규 카드만 상세 조회한다 (Q1).
SELECT c.tcgdex_id
FROM cards c
JOIN sets s ON s.id = c.set_id
WHERE s.language = @language AND s.tcgdex_id = @set_tcgdex_id;

-- name: UpdateCardBrief :execrows
-- 세트 상세의 카드 요약(번호·이름·이미지)만 갱신한다. 희귀도·도감 번호는 카드 상세에만 있어 건드리지 않는다.
-- 값이 같으면 갱신하지 않아 updated_at 이 실제 변경 시각을 나타낸다.
UPDATE cards
SET local_id = @local_id,
    name = @name,
    image_url = @image_url,
    updated_at = now()
WHERE language = @language
  AND tcgdex_id = @tcgdex_id
  AND (local_id IS DISTINCT FROM @local_id
       OR name IS DISTINCT FROM @name
       OR image_url IS DISTINCT FROM @image_url);
