-- name: CreateTransaction :one
INSERT INTO transactions (merchant_id, terminal_id, amount, currency, payment_method)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: AuthorizeTransaction :one
UPDATE transactions SET status = 'AUTHORIZED', authorization_code = $2, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeclineTransaction :one
UPDATE transactions SET status = 'DECLINED', updated_at = now() WHERE id = $1 RETURNING *;

-- name: CaptureTransaction :one
UPDATE transactions SET status = 'CAPTURED', updated_at = now() WHERE id = $1 RETURNING *;

-- name: RefundTransaction :one
UPDATE transactions SET status = 'REFUNDED', updated_at = now() WHERE id = $1 RETURNING *;

-- name: GetTransaction :one
SELECT * FROM transactions WHERE id = $1;

-- name: GetTransactionForUpdate :one
SELECT * FROM transactions WHERE id = $1 FOR UPDATE;
