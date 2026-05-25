-- name: CreateTransaction :one
INSERT INTO transactions (wallet_id, type, amount, balance_after, reference_id, note)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, wallet_id, type, amount, balance_after, reference_id, note, created_at;

-- name: ListTransactionsByWalletID :many
SELECT id, wallet_id, type, amount, balance_after, reference_id, note, created_at
FROM transactions
WHERE wallet_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;