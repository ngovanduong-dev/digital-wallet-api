-- name: CreateWallet :one
INSERT INTO wallets (user_id, currency)
VALUES ($1, $2)
RETURNING *;

-- name: GetWalletByUserID :one
SELECT * FROM wallets
WHERE user_id = $1 AND currency = $2;

-- name: UpdateWalletBalance :one
UPDATE wallets
SET balance = $1
WHERE id = $2
RETURNING *;