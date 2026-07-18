-- name: CreateTransactionEvent :one
INSERT INTO transaction_events (transaction_id, event, payload)
VALUES ($1, $2, $3) RETURNING *;

-- name: ListTransactionEvent :many
SELECT * FROM transaction_events WHERE transaction_id = $1 ORDER BY created_at DESC;