-- jumlah_santri is computed live as the count of santri holding a seat
-- (aktif + cuti); nonaktif and tidak_lanjut release their seat.

-- name: FindKelasByKunci :many
SELECT id, kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, guru_id, kapasitas, pertemuan_terakhir, materi_individual,
    (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = kelas.id AND s.status IN ('aktif', 'cuti')) AS jumlah_santri,
    created_at, is_aktif
FROM kelas
WHERE kunci_kelas = ? AND is_aktif = 1
ORDER BY sub_index ASC;

-- name: GetNextKelasSubIndexByKunci :one
SELECT COALESCE(MAX(sub_index), 0) + 1 FROM kelas WHERE kunci_kelas = ?;

-- name: GetKelasByID :one
SELECT id, kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, guru_id, kapasitas, pertemuan_terakhir, materi_individual,
    (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = kelas.id AND s.status IN ('aktif', 'cuti')) AS jumlah_santri,
    created_at, is_aktif
FROM kelas WHERE kelas.id = ?;

-- name: CreateKelas :execresult
INSERT INTO kelas (
    kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi,
    jadwal, sub_index, nama_kelas, guru_id, kapasitas, jumlah_santri,
    created_at
) VALUES (
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?,
    ?
);

-- name: IncrementJumlahSantri :exec
UPDATE kelas SET jumlah_santri = jumlah_santri + 1 WHERE id = ?;

-- name: DecrementJumlahSantri :exec
UPDATE kelas SET jumlah_santri = jumlah_santri - 1 WHERE id = ?;

-- name: ListKelasByAngkatan :many
SELECT id, kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, guru_id, kapasitas, pertemuan_terakhir, materi_individual,
    (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = kelas.id AND s.status IN ('aktif', 'cuti')) AS jumlah_santri,
    created_at, is_aktif
FROM kelas
WHERE (@angkatan = '' OR angkatan = @angkatan)
ORDER BY angkatan, tipe, jenis_kelamin, level, sub_index;

-- name: ListKelasAll :many
SELECT id, kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, guru_id, kapasitas, pertemuan_terakhir, materi_individual,
    (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = kelas.id AND s.status IN ('aktif', 'cuti')) AS jumlah_santri,
    created_at, is_aktif
FROM kelas
ORDER BY angkatan, tipe, jenis_kelamin, level, sub_index;

-- name: AssignGuru :exec
UPDATE kelas SET guru_id = ? WHERE id = ?;

-- name: UpdateKelasPertemuanTerakhir :execresult
-- If the class already started a new level before any meeting was recorded,
-- that level begins right after the anchor, so level_pertemuan_awal follows it.
UPDATE kelas
SET pertemuan_terakhir = sqlc.arg(pertemuan_terakhir),
    level_pertemuan_awal = CASE
        WHEN level_pertemuan_awal > 0
          OR EXISTS (SELECT 1 FROM kelas_perubahan kp WHERE kp.kelas_id = kelas.id AND kp.jenis = 'level_kelas')
        THEN sqlc.arg(pertemuan_terakhir)
        ELSE level_pertemuan_awal
    END
WHERE kelas.id = sqlc.arg(id)
  AND NOT EXISTS (SELECT 1 FROM pertemuan WHERE kelas_id = sqlc.arg(id));

-- name: SetKelasAktif :exec
UPDATE kelas SET is_aktif = ? WHERE id = ?;

-- name: SetKelasMateriIndividual :exec
UPDATE kelas SET materi_individual = ? WHERE id = ?;

-- name: DeleteKelas :exec
DELETE FROM kelas WHERE id = ?;

-- name: CountKelasByAngkatan :many
SELECT angkatan, COUNT(*) as total FROM kelas
GROUP BY angkatan
ORDER BY angkatan;

-- name: CountKelasTotal :one
SELECT COUNT(*) FROM kelas;
