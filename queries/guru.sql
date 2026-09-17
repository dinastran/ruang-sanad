-- name: GuruGetByUserID :one
SELECT * FROM guru WHERE user_id = ?;

-- name: GuruGetByID :one
SELECT * FROM guru WHERE id = ?;

-- name: GuruListAll :many
SELECT * FROM guru ORDER BY nama ASC;

-- name: ListGuruByStatus :many
SELECT * FROM guru WHERE status = ? ORDER BY nama ASC;

-- name: UpdateGuruUserID :exec
UPDATE guru SET user_id = ? WHERE id = ?;

-- name: CreateGuruFull :one
INSERT INTO guru (nama, gelar, jenis_kelamin, status, no_wa, email, tanggal_gabung, foto, is_aktif)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: ListLinkableUsers :many
-- Accounts not yet linked to any guru record, eligible for linking.
SELECT id, name, email, role
FROM users
WHERE id NOT IN (SELECT user_id FROM guru WHERE user_id IS NOT NULL)
ORDER BY name ASC;

-- name: CountGuruByUserID :one
SELECT COUNT(*) FROM guru WHERE user_id = ?;

-- name: ListKelasByGuruID :many
SELECT kelas.id, kelas.kunci_kelas, kelas.angkatan, kelas.tipe, kelas.jenis_kelamin, kelas.level, kelas.frekuensi, kelas.jadwal, kelas.sub_index, kelas.nama_kelas, kelas.guru_id, guru.nama AS guru_nama, kelas.kapasitas,
    (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = kelas.id AND s.status != 'tidak_lanjut') AS jumlah_santri,
    kelas.created_at, kelas.is_aktif
FROM kelas
LEFT JOIN guru ON guru.id = kelas.guru_id
WHERE (sqlc.narg(guru_id) IS NULL OR kelas.guru_id = sqlc.narg(guru_id)) AND kelas.is_aktif = 1
ORDER BY COALESCE(guru.nama, ''), angkatan, tipe, level, sub_index;

-- name: CountKelasByGuruID :one
SELECT COUNT(*) FROM kelas WHERE guru_id = ? AND is_aktif = 1;

-- name: CountSantriAktifByGuruID :one
SELECT COUNT(*) FROM santri s
JOIN kelas k ON s.kelas_id = k.id
WHERE k.guru_id = ? AND s.status = 'aktif' AND k.is_aktif = 1;

-- name: CountPertemuanHariIniByGuruID :one
SELECT COUNT(*) FROM pertemuan p
JOIN kelas k ON p.kelas_id = k.id
WHERE k.guru_id = ? AND p.tanggal = date('now');

-- name: CountPertemuanMingguIniByGuruID :one
SELECT COUNT(*) FROM pertemuan p
JOIN kelas k ON p.kelas_id = k.id
WHERE k.guru_id = ? AND p.tanggal >= date('now', '-7 days') AND p.tanggal <= date('now');

-- name: GetJadwalMengajarHariIni :many
SELECT k.id, k.nama_kelas, k.jadwal, k.level, k.tipe, k.jenis_kelamin,
    k.frekuensi, k.kapasitas,
    COALESCE((SELECT COUNT(*) FROM santri s WHERE s.kelas_id = k.id AND s.status = 'aktif'), 0) AS jumlah_santri_aktif
FROM kelas k
WHERE (sqlc.narg(guru_id) IS NULL OR k.guru_id = sqlc.narg(guru_id)) AND k.is_aktif = 1
ORDER BY k.jadwal;

-- name: GetKelasBelumAbsen :many
SELECT k.id, k.nama_kelas, k.jadwal, k.level, k.tipe,
    COALESCE(lp.pertemuan_terakhir, 0) AS pertemuan_terakhir,
    COALESCE(lp.tanggal_terakhir, '') AS tanggal_terakhir
FROM kelas k
LEFT JOIN (
    SELECT kelas_id, MAX(pertemuan_ke) AS pertemuan_terakhir, MAX(tanggal) AS tanggal_terakhir
    FROM pertemuan
    GROUP BY kelas_id
) lp ON lp.kelas_id = k.id
WHERE (sqlc.narg(guru_id) IS NULL OR k.guru_id = sqlc.narg(guru_id)) AND k.is_aktif = 1
  AND (lp.tanggal_terakhir IS NULL OR lp.tanggal_terakhir < date('now', '-1 days'))
ORDER BY lp.tanggal_terakhir ASC;

-- name: GetGuruStats :one
SELECT
    COUNT(DISTINCT k.id) AS total_kelas,
    COUNT(DISTINCT s.id) AS total_santri,
    COUNT(DISTINCT CASE WHEN s.status = 'aktif' THEN s.id END) AS santri_aktif,
    COUNT(DISTINCT p.id) AS total_pertemuan
FROM kelas k
LEFT JOIN santri s ON s.kelas_id = k.id
LEFT JOIN pertemuan p ON p.kelas_id = k.id
WHERE k.is_aktif = 1
  AND (sqlc.narg(guru_id) IS NULL OR k.guru_id = sqlc.narg(guru_id));

-- name: ListGuruByIDs :many
SELECT * FROM guru WHERE id IN (sqlc.slice('ids')) ORDER BY nama;

-- name: UpdateGuruDirectory :exec
UPDATE guru
SET nama = ?, gelar = ?, jenis_kelamin = ?, status = ?, no_wa = ?, email = ?,
    tanggal_gabung = ?, foto = ?, is_aktif = ?
WHERE id = ?;
