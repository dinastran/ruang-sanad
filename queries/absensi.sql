-- name: GetAbsensiByPertemuan :many
SELECT a.*, s.nama AS santri_nama, s.id_mahasantri
FROM absensi a
JOIN santri s ON a.santri_id = s.id
WHERE a.pertemuan_id = ?
ORDER BY s.nama;

-- name: GetAbsensiBySantri :many
SELECT a.*, p.pertemuan_ke, p.tanggal, p.materi
FROM absensi a
JOIN pertemuan p ON a.pertemuan_id = p.id
WHERE a.santri_id = ?
ORDER BY p.tanggal DESC;

-- name: GetAbsensiByPertemuanAndSantri :one
SELECT * FROM absensi WHERE pertemuan_id = ? AND santri_id = ?;

-- name: GetAbsensiByID :one
SELECT * FROM absensi WHERE id = ?;

-- name: CreateAbsensi :exec
INSERT INTO absensi (pertemuan_id, santri_id, status, catatan, batas_materi, dibuat_oleh)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateAbsensi :exec
UPDATE absensi SET status = ?, catatan = ?, batas_materi = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateAbsensiByPertemuanAndSantri :exec
UPDATE absensi SET status = ?, catatan = ?, batas_materi = ?, updated_at = CURRENT_TIMESTAMP
WHERE pertemuan_id = ? AND santri_id = ?;

-- name: CreateOrUpdateAbsensi :exec
INSERT INTO absensi (pertemuan_id, santri_id, status, catatan, batas_materi, dibuat_oleh)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(pertemuan_id, santri_id)
DO UPDATE SET status = excluded.status, catatan = excluded.catatan, batas_materi = excluded.batas_materi, updated_at = CURRENT_TIMESTAMP;

-- name: CountAbsensiByStatus :many
SELECT a.status, COUNT(*) AS total
FROM absensi a
JOIN pertemuan p ON a.pertemuan_id = p.id
WHERE p.kelas_id = ?
GROUP BY a.status;

-- name: CountAbsensiSantriByKelas :many
SELECT a.santri_id, s.nama AS santri_nama,
    COUNT(*) AS total,
    SUM(CASE WHEN a.status = 'hadir' THEN 1 ELSE 0 END) AS total_hadir,
    SUM(CASE WHEN a.status = 'izin' THEN 1 ELSE 0 END) AS total_izin,
    SUM(CASE WHEN a.status = 'sakit' THEN 1 ELSE 0 END) AS total_sakit,
    SUM(CASE WHEN a.status = 'alpa' THEN 1 ELSE 0 END) AS total_alpa,
    SUM(CASE WHEN a.status = 'telat' THEN 1 ELSE 0 END) AS total_telat
FROM absensi a
JOIN pertemuan p ON a.pertemuan_id = p.id
JOIN santri s ON a.santri_id = s.id
WHERE p.kelas_id = ? AND p.status = 'selesai'
GROUP BY a.santri_id
ORDER BY s.nama;

-- name: GetRekapAbsensiKelas :many
SELECT
    s.id AS santri_id,
    s.nama AS santri_nama,
    s.id_mahasantri
FROM santri s
WHERE s.kelas_id = ? AND s.status = 'aktif'
ORDER BY s.nama;

-- name: GetPertemuanByKelasForRekap :many
SELECT id, pertemuan_ke, pertemuan_level_ke, level_nama, tanggal
FROM pertemuan
WHERE kelas_id = ?
ORDER BY pertemuan_ke;

-- name: GetAbsensiByPertemuanIDs :many
SELECT a.pertemuan_id, a.santri_id, a.status, s.nama AS santri_nama
FROM absensi a
JOIN santri s ON a.santri_id = s.id
WHERE a.pertemuan_id IN (sqlc.slice('pertemuan_ids'))
ORDER BY s.nama;

-- name: GetAbsensiTerakhirBySantri :one
SELECT a.*, p.pertemuan_ke, p.tanggal, p.materi
FROM absensi a
JOIN pertemuan p ON a.pertemuan_id = p.id
WHERE a.santri_id = ?
ORDER BY p.tanggal DESC
LIMIT 1;

-- name: CountSantriAlpaBerturutByKelas :many
SELECT s.id, s.nama, COUNT(*) AS total_alpa
FROM absensi a
JOIN pertemuan p ON a.pertemuan_id = p.id
JOIN santri s ON a.santri_id = s.id
WHERE p.kelas_id = ?
  AND a.status = 'alpa'
  AND p.tanggal >= date('now', '-30 days')
GROUP BY s.id
HAVING COUNT(*) >= 3
ORDER BY total_alpa DESC;
