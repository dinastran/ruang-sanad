-- name: CreatePasswordReset :exec
INSERT INTO password_resets (token, user_id, email, expires_at, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetPasswordReset :one
SELECT * FROM password_resets WHERE token = ? AND used = 0 AND expires_at > ?;

-- name: MarkPasswordResetUsed :exec
UPDATE password_resets SET used = 1 WHERE token = ?;

-- name: DeleteExpiredPasswordResets :exec
DELETE FROM password_resets WHERE expires_at < CURRENT_TIMESTAMP;
