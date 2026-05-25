-- name: CreateWallet :one
INSERT INTO wallets (user_id, currency)
VALUES ($1, $2)
RETURNING id, user_id, balance, currency, created_at;

-- name: GetWalletByUserID :one
SELECT id, user_id, balance, currency, created_at
FROM wallets
WHERE user_id = $1 AND currency = $2;

-- name: UpdateWalletBalance :one
UPDATE wallets
SET balance = $1
WHERE id = $2
RETURNING id, user_id, balance, currency, created_at;