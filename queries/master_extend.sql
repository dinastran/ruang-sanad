-- name: DeleteAngkatan :exec
DELETE FROM angkatan WHERE id = ?;

-- name: DeleteLevel :exec
DELETE FROM level WHERE id = ?;

-- name: DeleteJadwal :exec
DELETE FROM jadwal WHERE id = ?;

-- name: DeleteGuru :exec
DELETE FROM guru WHERE id = ?;

-- name: UpdateJadwal :exec
UPDATE jadwal SET nama = ? WHERE id = ?;
