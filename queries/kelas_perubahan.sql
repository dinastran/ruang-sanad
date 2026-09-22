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
    kelas_id, santri_id, kelas_tujuan_id, jenis, nilai_lama, nilai_baru, pertemuan_ke, dibuat_oleh
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListKelasPerubahanByKelas :many
SELECT
    kp.id, kp.kelas_id, kp.santri_id, kp.kelas_tujuan_id, kp.jenis,
    kp.nilai_lama, kp.nilai_baru, kp.pertemuan_ke, kp.created_at,
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
