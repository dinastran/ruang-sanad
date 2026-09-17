-- name: ListRiwayatPertemuanByKelas :many
SELECT * FROM pertemuan
WHERE kelas_id = ? AND status = 'selesai'
ORDER BY pertemuan_ke DESC;

-- name: GetPertemuanByID :one
SELECT * FROM pertemuan WHERE id = ?;

-- name: GetActivePertemuanByKelas :one
SELECT * FROM pertemuan
WHERE kelas_id = ? AND status = 'berlangsung'
ORDER BY id DESC
LIMIT 1;

-- name: GetLastPertemuanByKelas :one
SELECT * FROM pertemuan
WHERE kelas_id = ? AND status = 'selesai'
ORDER BY pertemuan_ke DESC
LIMIT 1;

-- name: GetNextPertemuanKe :one
SELECT MAX(COALESCE((SELECT MAX(p.pertemuan_ke) FROM pertemuan p WHERE p.kelas_id = kelas.id), 0), kelas.pertemuan_terakhir) + 1 AS next_ke
FROM kelas
WHERE kelas.id = ?;

-- name: CreatePertemuan :execresult
INSERT INTO pertemuan (
    kelas_id, pertemuan_ke, tanggal, jam_mulai, jam_selesai,
    materi, catatan, is_reschedule, jadwal_semula, alasan_reschedule,
    is_badal, guru_pengganti_id, alasan_badal, status, dibuat_oleh
) VALUES (
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?
);

-- name: ClaimPertemuanCompletion :execrows
UPDATE pertemuan
SET status = 'menyelesaikan', updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND kelas_id = ? AND status = 'berlangsung';

-- name: UpdatePertemuanSelesai :execrows
UPDATE pertemuan
SET jam_selesai = ?, materi = ?, catatan = ?, status = 'selesai', updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND kelas_id = ? AND status = 'menyelesaikan';

-- name: UpdatePertemuanStatus :exec
UPDATE pertemuan SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: CountPertemuanByKelas :one
SELECT COUNT(*) FROM pertemuan WHERE kelas_id = ?;

-- name: CountPertemuanReschedule :one
SELECT COUNT(*) FROM pertemuan WHERE kelas_id = ? AND is_reschedule = 1;

-- name: CountPertemuanBadal :one
SELECT COUNT(*) FROM pertemuan WHERE kelas_id = ? AND is_badal = 1;

-- name: CountPertemuanByGuruID :many
SELECT k.id AS kelas_id, k.nama_kelas, COUNT(p.id) AS total_pertemuan
FROM kelas k
LEFT JOIN pertemuan p ON p.kelas_id = k.id
WHERE k.guru_id = ?
GROUP BY k.id
ORDER BY k.nama_kelas;
