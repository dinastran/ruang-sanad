-- name: CreateJadwalPertemuan :one
INSERT INTO jadwal_pertemuan (kelas_id, tanggal, jam_mulai, catatan, dibuat_oleh)
VALUES (?, ?, ?, ?, ?)
RETURNING id;

-- name: GetJadwalPertemuanByID :one
SELECT * FROM jadwal_pertemuan WHERE id = ? AND kelas_id = ?;

-- name: CountDueJadwalPertemuanByKelas :one
SELECT COUNT(*) FROM jadwal_pertemuan
WHERE kelas_id = ? AND status = 'dijadwalkan' AND tanggal <= ?;

-- name: ListDueJadwalPertemuanByKelas :many
SELECT jp.*, COALESCE(gb.nama, '') AS guru_pengganti_nama
FROM jadwal_pertemuan jp
LEFT JOIN guru gb ON gb.id = jp.guru_pengganti_id
WHERE jp.kelas_id = ? AND jp.status = 'dijadwalkan' AND jp.tanggal <= ?
ORDER BY jp.tanggal, jp.jam_mulai, jp.id;

-- name: ListJadwalPertemuanForGuru :many
SELECT
    jp.*,
    k.nama_kelas,
    k.guru_id AS guru_utama_id,
    COALESCE(gu.nama, '') AS guru_utama_nama,
    COALESCE(gb.nama, '') AS guru_pengganti_nama
FROM jadwal_pertemuan jp
JOIN kelas k ON k.id = jp.kelas_id
LEFT JOIN guru gu ON gu.id = k.guru_id
LEFT JOIN guru gb ON gb.id = jp.guru_pengganti_id
WHERE jp.status IN ('dijadwalkan', 'dimulai')
  AND (
      sqlc.narg(guru_id) IS NULL
      OR k.guru_id = sqlc.narg(guru_id)
      OR jp.guru_pengganti_id = sqlc.narg(guru_id)
  )
ORDER BY jp.tanggal, jp.jam_mulai, jp.id;

-- name: UpdateJadwalPertemuanReschedule :execrows
UPDATE jadwal_pertemuan
SET tanggal = ?, jam_mulai = ?, is_reschedule = 1, jadwal_semula = ?,
    alasan_reschedule = ?, jadwal_kelas_berubah = 0, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND kelas_id = ? AND status = 'dijadwalkan';

-- name: UpdateJadwalPertemuanBadal :execrows
UPDATE jadwal_pertemuan
SET guru_pengganti_id = ?, alasan_badal = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND kelas_id = ? AND status = 'dijadwalkan';

-- name: CancelJadwalPertemuan :execrows
UPDATE jadwal_pertemuan
SET status = 'dibatalkan', alasan_batal = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND kelas_id = ? AND status = 'dijadwalkan';

-- name: ClaimJadwalPertemuan :execrows
UPDATE jadwal_pertemuan
SET status = 'diproses', updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND kelas_id = ? AND status = 'dijadwalkan';

-- name: CompleteJadwalPertemuanStart :exec
UPDATE jadwal_pertemuan
SET status = 'dimulai', pertemuan_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND kelas_id = ? AND status = 'diproses';

-- name: MarkJadwalPertemuanSusulan :exec
UPDATE jadwal_pertemuan
SET is_susulan = 1, alasan_susulan = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND kelas_id = ?;

-- name: ListKoordinatorUserIDs :many
SELECT id FROM users WHERE role = 'koordinator_guru' ORDER BY id;

-- name: MarkJadwalPertemuanSelesai :exec
UPDATE jadwal_pertemuan
SET status = 'selesai', updated_at = CURRENT_TIMESTAMP
WHERE pertemuan_id = ? AND status = 'dimulai';

-- name: MarkJadwalPertemuanKelasBerubah :exec
UPDATE jadwal_pertemuan
SET jadwal_kelas_berubah = 1, updated_at = CURRENT_TIMESTAMP
WHERE kelas_id = ? AND status = 'dijadwalkan';

-- name: ListJadwalDibatalkanByKelas :many
-- Riwayat pembatalan kelas: tidak punya pertemuan sehingga tidak memakai
-- nomor pertemuan. Pembatal diambil dari log aktivitas terakhir.
SELECT
    jp.id,
    jp.tanggal,
    jp.jam_mulai,
    jp.alasan_batal,
    jp.updated_at,
    CAST(COALESCE((
        SELECT u.name FROM schedule_activity_logs l
        JOIN users u ON u.id = l.created_by
        WHERE l.jadwal_pertemuan_id = jp.id AND l.action = 'jadwal_dibatalkan'
        ORDER BY l.created_at DESC, l.id DESC LIMIT 1
    ), '') AS TEXT) AS dibatalkan_oleh,
    CAST(COALESCE((
        SELECT u.role FROM schedule_activity_logs l
        JOIN users u ON u.id = l.created_by
        WHERE l.jadwal_pertemuan_id = jp.id AND l.action = 'jadwal_dibatalkan'
        ORDER BY l.created_at DESC, l.id DESC LIMIT 1
    ), '') AS TEXT) AS dibatalkan_oleh_role
FROM jadwal_pertemuan jp
WHERE jp.kelas_id = ? AND jp.status = 'dibatalkan'
ORDER BY jp.tanggal DESC, jp.jam_mulai DESC, jp.id DESC;

-- name: ListSusulanByKelas :many
SELECT pertemuan_id, alasan_susulan
FROM jadwal_pertemuan
WHERE kelas_id = ? AND is_susulan = 1 AND pertemuan_id IS NOT NULL;
