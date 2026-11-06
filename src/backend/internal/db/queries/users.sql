-- 사용자·컬렉션 기본 쿼리. 인증(#9)과 컬렉션 API(#10)의 출발점이다.

-- name: CreateUser :one
INSERT INTO users (email, password_hash, display_name)
VALUES (@email, @password_hash, @display_name)
RETURNING *;

-- name: GetUserByEmail :one
-- 이메일은 대소문자를 구분하지 않는다 (users_email_key 인덱스 사용).
SELECT * FROM users WHERE lower(email) = lower(@email);

-- name: UpsertCollectionItem :one
-- 같은 변형·상태 등급이면 수량을 새 값으로 바꾼다.
INSERT INTO collection_items (user_id, card_variant_id, condition, quantity)
VALUES (@user_id, @card_variant_id, @condition, @quantity)
ON CONFLICT (user_id, card_variant_id, condition) DO UPDATE
SET quantity = EXCLUDED.quantity,
    updated_at = now()
RETURNING *;

-- name: ListCollectionItems :many
SELECT * FROM collection_items WHERE user_id = @user_id ORDER BY id;

-- name: CreateUserTrade :one
INSERT INTO user_trades (user_id, card_variant_id, price_krw, trade_date, condition)
VALUES (@user_id, @card_variant_id, @price_krw, @trade_date, @condition)
RETURNING *;
