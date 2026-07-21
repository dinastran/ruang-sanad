-- name: ListAngkatan :many
SELECT * FROM angkatan ORDER BY kode DESC;

-- name: GetAngkatanByKode :one
SELECT * FROM angkatan WHERE kode = ?;

-- name: CreateAngkatan :exec
INSERT INTO angkatan (kode, keterangan, is_aktif, created_at)
VALUES (?, ?, ?, CURRENT_TIMESTAMP);

-- name: UpdateAngkatan :exec
UPDATE angkatan SET keterangan = ?, is_aktif = ? WHERE id = ?;

-- name: ListLevel :many
SELECT * FROM level ORDER BY urutan ASC;

-- name: GetLevelByKode :one
SELECT * FROM level WHERE kode = ?;

-- name: CreateLevel :exec
INSERT INTO level (kode, nama, urutan) VALUES (?, ?, ?);

-- name: UpdateLevel :exec
UPDATE level SET nama = ?, urutan = ? WHERE id = ?;

-- name: ListJadwal :many
SELECT * FROM jadwal ORDER BY nama ASC;

-- name: GetJadwalByNama :one
SELECT * FROM jadwal WHERE nama = ?;

-- name: CreateJadwal :exec
INSERT INTO jadwal (nama) VALUES (?);

-- name: ListGuru :many
SELECT * FROM guru WHERE is_aktif = 1 ORDER BY nama ASC;

-- name: ListGuruAll :many
SELECT * FROM guru ORDER BY nama ASC;

-- name: GetGuruByID :one
SELECT * FROM guru WHERE id = ?;

-- name: CreateGuru :exec
INSERT INTO guru (nama, jenis_kelamin, is_aktif) VALUES (?, ?, ?);

-- name: UpdateGuru :exec
UPDATE guru SET nama = ?, jenis_kelamin = ?, is_aktif = ? WHERE id = ?;
