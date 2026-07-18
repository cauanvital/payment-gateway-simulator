-- name: CreateTerminal :one
INSERT INTO terminals (merchant_id, serial)
VALUES ($1, $2) RETURNING *;

-- name: GetTerminalByID :one
SELECT * FROM terminals WHERE id = $1;

-- name: GetTerminalBySerial :one
SELECT * FROM terminals WHERE serial = $1;

-- name: ListTerminals :many
SELECT * FROM terminals WHERE merchant_id = $1 ORDER BY created_at DESC;

-- name: UpdateTerminalStatus :one
UPDATE terminals SET status = $2 WHERE id = $1 RETURNING *;