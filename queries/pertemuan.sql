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

-- name: GetNextPertemuanLevelKe :one
-- Nomor pertemuan berikutnya dihitung dari awal level kelas saat ini.
SELECT CAST(MAX(1, MAX(COALESCE((SELECT MAX(p.pertemuan_ke) FROM pertemuan p WHERE p.kelas_id = kelas.id), 0), kelas.pertemuan_terakhir) + 1 - kelas.level_pertemuan_awal) AS INTEGER) AS next_level_ke
FROM kelas
WHERE kelas.id = ?;

-- name: CreatePertemuan :execresult
-- level_nama & pertemuan_level_ke are snapshotted from the class so meeting
-- history stays labelled with the level it was taught at.
INSERT INTO pertemuan (
    kelas_id, pertemuan_ke, tanggal, jam_mulai, jam_selesai,
    materi, catatan, is_reschedule, jadwal_semula, alasan_reschedule,
    is_badal, guru_pengganti_id, alasan_badal, status, dibuat_oleh,
    level_nama, pertemuan_level_ke
) VALUES (
    sqlc.arg(kelas_id), sqlc.arg(pertemuan_ke), sqlc.arg(tanggal), sqlc.arg(jam_mulai), sqlc.arg(jam_selesai),
    sqlc.arg(materi), sqlc.arg(catatan), sqlc.arg(is_reschedule), sqlc.arg(jadwal_semula), sqlc.arg(alasan_reschedule),
    sqlc.arg(is_badal), sqlc.arg(guru_pengganti_id), sqlc.arg(alasan_badal), sqlc.arg(status), sqlc.arg(dibuat_oleh),
    COALESCE((
        SELECT COALESCE(NULLIF(l.nama, ''), k.level)
        FROM kelas k
        LEFT JOIN level l ON l.kode = k.level
        WHERE k.id = sqlc.arg(kelas_id)
    ), ''),
    MAX(1, sqlc.arg(pertemuan_ke) - COALESCE((SELECT k.level_pertemuan_awal FROM kelas k WHERE k.id = sqlc.arg(kelas_id)), 0))
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
