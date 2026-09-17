-- name: ListRiayahByGuru :many
SELECT cr.*,
    COALESCE(s.nama, '') AS target_nama,
    COALESCE(s.id_mahasantri, '') AS target_id_mahasantri,
    COALESCE(author.name, u.name) AS penulis_nama
FROM catatan_riayah cr
LEFT JOIN santri s ON cr.target_type = 'santri' AND cr.target_id = s.id
LEFT JOIN guru g ON cr.guru_id = g.id
LEFT JOIN users u ON g.user_id = u.id
LEFT JOIN users author ON cr.author_user_id = author.id
WHERE cr.guru_id = ?
ORDER BY cr.created_at DESC;

-- name: ListRiayahByTarget :many
SELECT cr.*, COALESCE(author.name, u.name) AS penulis_nama
FROM catatan_riayah cr
LEFT JOIN guru g ON cr.guru_id = g.id
LEFT JOIN users u ON g.user_id = u.id
LEFT JOIN users author ON cr.author_user_id = author.id
WHERE cr.target_type = ? AND cr.target_id = ?
ORDER BY cr.created_at DESC;

-- name: GetRiayahByID :one
SELECT * FROM catatan_riayah WHERE id = ?;

-- name: CreateRiayah :execresult
INSERT INTO catatan_riayah (guru_id, author_user_id, target_type, target_id, catatan)
VALUES (?, ?, ?, ?, ?);

-- name: CreateUserRiayah :execresult
INSERT INTO catatan_riayah (author_user_id, target_type, target_id, catatan)
VALUES (?, ?, ?, ?);

-- name: UpdateRiayah :exec
UPDATE catatan_riayah
SET catatan = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND guru_id = ?;

-- name: UpdateRiayahByID :exec
UPDATE catatan_riayah
SET catatan = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteRiayah :exec
DELETE FROM catatan_riayah WHERE id = ? AND guru_id = ?;

-- name: DeleteRiayahByID :exec
DELETE FROM catatan_riayah WHERE id = ?;

-- name: DeleteRiayahByTarget :exec
DELETE FROM catatan_riayah WHERE target_type = ? AND target_id = ?;

-- name: CountRiayahByGuru :one
SELECT COUNT(*) FROM catatan_riayah WHERE guru_id = ?;
