-- name: TryCreateIdempotencyKey :one
INSERT INTO idempotency_keys (key, endpoint, request_hash, response)
VALUES ($1, $2, $3, '{}'::jsonb)
ON CONFLICT (key, endpoint) DO NOTHING
RETURNING key, endpoint, request_hash, status_code, response, created_at;

-- name: GetIdempotencyKey :one
SELECT key, endpoint, request_hash, status_code, response, created_at
FROM idempotency_keys
WHERE key = $1
  AND endpoint = $2;

-- name: CompleteIdempotencyKey :one
UPDATE idempotency_keys
SET status_code = $1,
    response = $2
WHERE key = $3
  AND endpoint = $4
RETURNING key, endpoint, request_hash, status_code, response, created_at;
