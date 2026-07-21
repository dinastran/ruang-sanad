-- name: ListKodeKelas :many
SELECT * FROM kode_kelas ORDER BY urutan ASC;

-- name: GetKodeKelasByKode :one
SELECT * FROM kode_kelas WHERE kode = ?;

-- name: CreateKodeKelas :exec
INSERT INTO kode_kelas (kode, tipe, frekuensi, urutan) VALUES (?, ?, ?, ?);

-- name: UpdateKodeKelas :exec
UPDATE kode_kelas SET tipe = ?, frekuensi = ?, urutan = ? WHERE id = ?;

-- name: DeleteKodeKelas :exec
DELETE FROM kode_kelas WHERE id = ?;
