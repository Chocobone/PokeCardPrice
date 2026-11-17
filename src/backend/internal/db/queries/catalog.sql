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

-- name: UpsertPokemonSpecies :execrows
-- 종 이름 전체를 한 번에 upsert 한다 (#5, Q5). 값이 같은 행은 갱신하지 않는다.
INSERT INTO pokemon_species (dex_id, name_en, name_ja, name_ko)
-- 빈 이름은 NULL 로 넣는다. 배열 길이는 호출 측에서 같게 맞춘다.
SELECT unnest(@dex_ids::int[]), unnest(@names_en::text[]), NULLIF(unnest(@names_ja::text[]), ''), NULLIF(unnest(@names_ko::text[]), '')
ON CONFLICT (dex_id) DO UPDATE
SET name_en = EXCLUDED.name_en,
    name_ja = EXCLUDED.name_ja,
    name_ko = EXCLUDED.name_ko,
    updated_at = now()
WHERE (pokemon_species.name_en, pokemon_species.name_ja, pokemon_species.name_ko)
      IS DISTINCT FROM (EXCLUDED.name_en, EXCLUDED.name_ja, EXCLUDED.name_ko);

-- name: UpsertAutoCardLinks :execrows
-- 한글판 자동 연결 (#5, Q2): ko 카드와 세트 ID·localId 가 같고 dexId 가 같은(둘 다 없으면 같음) ja 카드를 잇는다.
-- 연결 방향은 항상 ko → ja 이다. manual 연결은 덮어쓰지 않는다. 바뀐 행이 없으면 0 이다.
INSERT INTO card_links (card_id, linked_card_id, method)
SELECT ko.id, ja.id, 'auto'
FROM cards ko
JOIN sets ks ON ks.id = ko.set_id
JOIN sets js ON js.language = 'ja' AND js.tcgdex_id = ks.tcgdex_id
JOIN cards ja ON ja.set_id = js.id AND ja.local_id = ko.local_id
WHERE ko.language = 'ko'
  AND ko.dex_ids IS NOT DISTINCT FROM ja.dex_ids
ON CONFLICT (card_id) DO UPDATE
SET linked_card_id = EXCLUDED.linked_card_id,
    updated_at = now()
WHERE card_links.method = 'auto'
  AND card_links.linked_card_id <> EXCLUDED.linked_card_id;

-- name: DeleteStaleAutoCardLinks :execrows
-- 규칙에 더는 맞지 않는 auto 연결을 지운다 (일본판 카드의 도감 번호가 바뀐 경우 등). manual 연결은 두지 않는다.
DELETE FROM card_links l
USING cards ko
WHERE l.card_id = ko.id
  AND l.method = 'auto'
  AND NOT EXISTS (
    SELECT 1
    FROM cards ja
    JOIN sets js ON js.id = ja.set_id
    JOIN sets ks ON ks.id = ko.set_id
    WHERE ja.id = l.linked_card_id
      AND ko.language = 'ko'
      AND ja.language = 'ja'
      AND js.tcgdex_id = ks.tcgdex_id
      AND ja.local_id = ko.local_id
      AND ko.dex_ids IS NOT DISTINCT FROM ja.dex_ids);

-- name: ListUnlinkedKoCards :many
-- 연결이 없는 ko 카드 (#7 수동 연결 대상).
SELECT c.tcgdex_id
FROM cards c
WHERE c.language = 'ko'
  AND NOT EXISTS (SELECT 1 FROM card_links l WHERE l.card_id = c.id)
ORDER BY c.tcgdex_id;

-- name: SetHasKoEdition :execrows
-- has_ko_edition 표시 (#5, Q4): 같은 ID 의 ko 세트가 있는 ja 세트에서, 번호(숫자) ≤ ko 세트 카드 수인 카드만 참.
-- 그 밖의 ja 카드는 거짓으로 되돌린다. 값이 바뀌는 행만 갱신한다.
UPDATE cards c
SET has_ko_edition = x.flag,
    updated_at = now()
FROM (
  SELECT ja.id,
         COALESCE(CASE WHEN ja.local_id ~ '^[0-9]{1,6}$' THEN ja.local_id::int END <= ks.card_count, false) AS flag
  FROM cards ja
  JOIN sets js ON js.id = ja.set_id
  LEFT JOIN sets ks ON ks.language = 'ko' AND ks.tcgdex_id = js.tcgdex_id
  WHERE ja.language = 'ja'
) x
WHERE c.id = x.id
  AND c.has_ko_edition IS DISTINCT FROM x.flag;

-- name: GetKoLinkStats :one
-- 한글판 연결 현황 집계 (수집 결과 로그용).
SELECT
  (SELECT count(*) FROM cards WHERE language = 'ko')::int AS ko_cards,
  (SELECT count(*) FROM card_links l JOIN cards c ON c.id = l.card_id WHERE c.language = 'ko' AND l.method = 'auto')::int AS auto_links,
  (SELECT count(*) FROM card_links l JOIN cards c ON c.id = l.card_id WHERE c.language = 'ko' AND l.method = 'manual')::int AS manual_links,
  (SELECT count(*) FROM cards WHERE has_ko_edition)::int AS has_ko_edition;

-- name: ListCardsByKoSpeciesName :many
-- 한글 종 이름으로 카드를 찾는다 (#8 검색의 출발점, 3단계 검증용).
SELECT c.*
FROM pokemon_species p
JOIN cards c ON c.dex_ids @> ARRAY[p.dex_id]
WHERE p.name_ko = @name_ko
ORDER BY c.language, c.tcgdex_id;
