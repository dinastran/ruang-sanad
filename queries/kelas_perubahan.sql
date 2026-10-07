-- name: GetKelasPerubahanState :one
SELECT id, kunci_kelas, level, jadwal, sub_index, level_pertemuan_awal
FROM kelas WHERE id = ?;

-- name: UpdateKelasIdentitas :exec
-- Dipakai ganti level / ganti jadwal: level & jadwal adalah bagian dari
-- kunci_kelas, jadi kunci, sub_index, dan nama ikut dibentuk ulang.
UPDATE kelas
SET level = ?, jadwal = ?, kunci_kelas = ?, sub_index = ?, nama_kelas = ?, level_pertemuan_awal = ?
WHERE id = ?;

-- name: SyncSantriLevelJadwalByKelas :exec
UPDATE santri SET level = ?, jadwal = ?, updated_at = ? WHERE kelas_id = ?;

-- name: CreateKelasPerubahan :exec
INSERT INTO kelas_perubahan (
    kelas_id, santri_id, kelas_tujuan_id, jenis, nilai_lama, nilai_baru, pertemuan_ke, alasan, dibuat_oleh
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListKelasPerubahanByKelas :many
SELECT
    kp.id, kp.kelas_id, kp.santri_id, kp.kelas_tujuan_id, kp.jenis,
    kp.nilai_lama, kp.nilai_baru, kp.pertemuan_ke, kp.alasan, kp.created_at,
    COALESCE(s.nama, '') AS santri_nama,
    COALESCE(ka.nama_kelas, '') AS kelas_asal_nama,
    COALESCE(kt.nama_kelas, '') AS kelas_tujuan_nama,
    COALESCE(u.name, 'Sistem') AS dibuat_oleh_nama
FROM kelas_perubahan kp
LEFT JOIN santri s ON s.id = kp.santri_id
LEFT JOIN kelas ka ON ka.id = kp.kelas_id
LEFT JOIN kelas kt ON kt.id = kp.kelas_tujuan_id
LEFT JOIN users u ON u.id = kp.dibuat_oleh
WHERE kp.kelas_id = sqlc.arg(kelas_id) OR kp.kelas_tujuan_id = sqlc.arg(kelas_id)
ORDER BY kp.created_at DESC, kp.id DESC
LIMIT 50;

-- name: SetKelasAwalPertemuan :exec
-- Kelas baru hasil naik level melanjutkan nomor internal kelas asal (untuk
-- periode tagihan) sementara nomor per level tetap mulai dari 1.
UPDATE kelas SET pertemuan_terakhir = ?, level_pertemuan_awal = ?, kapasitas = ? WHERE id = ?;

-- name: GetPertemuanRebaseBounds :one
SELECT
    COUNT(*) AS total,
    CAST(COALESCE(MIN(pertemuan_ke), 0) AS INTEGER) AS min_pertemuan_ke,
    CAST(COALESCE(MAX(pertemuan_ke), 0) AS INTEGER) AS max_pertemuan_ke
FROM pertemuan
WHERE kelas_id = ?;

-- name: GetFirstLevelChangeBoundary :one
SELECT CAST(COALESCE(MIN(pertemuan_ke), 0) AS INTEGER)
FROM kelas_perubahan
WHERE kelas_id = ? AND jenis = 'level_kelas' AND pertemuan_ke > 0;

-- name: GetFirstPertemuanLevelKe :one
SELECT pertemuan_level_ke
FROM pertemuan
WHERE kelas_id = ?
ORDER BY pertemuan_ke ASC, id ASC
LIMIT 1;

-- name: ShiftPertemuanLevelKeThrough :exec
UPDATE pertemuan
SET pertemuan_level_ke = pertemuan_level_ke + sqlc.arg(delta)
WHERE kelas_id = sqlc.arg(kelas_id)
  AND pertemuan_ke <= sqlc.arg(boundary);

-- name: ShiftAllPertemuanLevelKe :exec
UPDATE pertemuan
SET pertemuan_level_ke = pertemuan_level_ke + sqlc.arg(delta)
WHERE kelas_id = sqlc.arg(kelas_id);

-- name: OffsetPertemuanKeForRebase :exec
UPDATE pertemuan
SET pertemuan_ke = pertemuan_ke + sqlc.arg(offset)
WHERE kelas_id = sqlc.arg(kelas_id);

-- name: FinalizePertemuanKeRebase :exec
UPDATE pertemuan
SET pertemuan_ke = pertemuan_ke - sqlc.arg(offset) + sqlc.arg(delta)
WHERE kelas_id = sqlc.arg(kelas_id);

-- name: RebaseKelasMeetingAnchors :exec
UPDATE kelas
SET pertemuan_terakhir = sqlc.arg(new_anchor),
    level_pertemuan_awal = CASE
        WHEN level_pertemuan_awal > 0 THEN level_pertemuan_awal + sqlc.arg(delta)
        ELSE level_pertemuan_awal
    END
WHERE id = sqlc.arg(kelas_id);

-- name: ShiftLevelChangeBoundaries :exec
UPDATE kelas_perubahan
SET pertemuan_ke = pertemuan_ke + sqlc.arg(delta)
WHERE kelas_id = sqlc.arg(kelas_id)
  AND jenis = 'level_kelas'
  AND pertemuan_ke > 0;

-- name: ShiftSantriPertemuanAwalByKelas :exec
UPDATE santri
SET pertemuan_awal = pertemuan_awal + sqlc.arg(delta),
    updated_at = CURRENT_TIMESTAMP
WHERE kelas_id = sqlc.arg(kelas_id)
  AND pertemuan_awal > 0;

-- name: SyncTagihanPertemuanKeByKelas :exec
UPDATE tagihan
SET pertemuan_ke = (
    SELECT p.pertemuan_ke
    FROM pertemuan p
    WHERE p.id = tagihan.pertemuan_id
)
WHERE kelas_id = sqlc.arg(kelas_id)
  AND pertemuan_id IS NOT NULL
  AND EXISTS (
      SELECT 1 FROM pertemuan p
      WHERE p.id = tagihan.pertemuan_id
        AND p.kelas_id = sqlc.arg(kelas_id)
  );
