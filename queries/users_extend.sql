-- name: ListUsers :many
SELECT id, email, name, password, avatar, role, google_id, email_verified, created_at, updated_at
FROM users
ORDER BY created_at DESC;

-- name: UpdateUserRole :exec
UPDATE users SET role = ?, updated_at = ? WHERE id = ?;

-- name: GetUserRole :one
SELECT role FROM users WHERE id = ?;
