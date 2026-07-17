-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, data, expires_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetSessionByID :one
SELECT id, user_id, data, expires_at, created_at, updated_at
FROM sessions
WHERE id = ?;

-- name: GetSessionsByUserID :many
SELECT id, user_id, data, expires_at, created_at, updated_at
FROM sessions
WHERE user_id = ?;

-- name: UpdateSession :execrows
UPDATE sessions
SET data = ?, expires_at = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteSession :execrows
DELETE FROM sessions
WHERE id = ?;

-- name: DeleteSessionsByUserID :exec
DELETE FROM sessions
WHERE user_id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < ?;
