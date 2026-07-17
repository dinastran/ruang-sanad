-- name: CreateUser :one
INSERT INTO users (email, name, password, role, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: CreateUserWithGoogleID :one
INSERT INTO users (email, name, google_id, avatar, email_verified, role, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: GetUserByID :one
SELECT id, email, name, password, avatar, role, google_id, email_verified, created_at, updated_at
FROM users
WHERE id = ?;

-- name: GetUserByEmail :one
SELECT id, email, name, password, avatar, role, google_id, email_verified, created_at, updated_at
FROM users
WHERE email = ?;

-- name: GetUserByGoogleID :one
SELECT id, email, name, password, avatar, role, google_id, email_verified, created_at, updated_at
FROM users
WHERE google_id = ?;

-- name: UpdateUser :execrows
UPDATE users
SET name = ?, avatar = ?, email_verified = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateUserPassword :execrows
UPDATE users
SET password = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateUserAvatar :execrows
UPDATE users
SET avatar = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteUser :execrows
DELETE FROM users
WHERE id = ?;

-- name: SetUserRoleAdmin :execrows
UPDATE users
SET role = ?, updated_at = ?
WHERE id = ?;
