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
    alasan_reschedule = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND kelas_id = ? AND status = 'dijadwalkan';

-- name: UpdateJadwalPertemuanBadal :execrows
UPDATE jadwal_pertemuan
SET guru_pengganti_id = ?, alasan_badal = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND kelas_id = ? AND status = 'dijadwalkan';

-- name: CancelJadwalPertemuan :execrows
UPDATE jadwal_pertemuan
SET status = 'dibatalkan', updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND kelas_id = ? AND status = 'dijadwalkan';

-- name: ClaimJadwalPertemuan :execrows
UPDATE jadwal_pertemuan
SET status = 'diproses', updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND kelas_id = ? AND status = 'dijadwalkan';

-- name: CompleteJadwalPertemuanStart :exec
UPDATE jadwal_pertemuan
SET status = 'dimulai', pertemuan_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND kelas_id = ? AND status = 'diproses';

-- name: MarkJadwalPertemuanSelesai :exec
UPDATE jadwal_pertemuan
SET status = 'selesai', updated_at = CURRENT_TIMESTAMP
WHERE pertemuan_id = ? AND status = 'dimulai';
