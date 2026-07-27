-- jumlah_santri is computed live as the count of ACTIVE santri (excluding
-- tidak_lanjut) so class occupancy reflects only continuing mahasantri.

-- name: FindKelasByKunci :many
SELECT id, kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, guru_id, kapasitas,
    (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = kelas.id AND s.status != 'tidak_lanjut') AS jumlah_santri,
    created_at, is_aktif
FROM kelas
WHERE kunci_kelas = ?
ORDER BY sub_index ASC;

-- name: GetKelasByID :one
SELECT id, kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, guru_id, kapasitas,
    (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = kelas.id AND s.status != 'tidak_lanjut') AS jumlah_santri,
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
SELECT id, kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, guru_id, kapasitas,
    (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = kelas.id AND s.status != 'tidak_lanjut') AS jumlah_santri,
    created_at, is_aktif
FROM kelas
WHERE (@angkatan = '' OR angkatan = @angkatan)
ORDER BY angkatan, tipe, jenis_kelamin, level, sub_index;

-- name: AssignGuru :exec
UPDATE kelas SET guru_id = ? WHERE id = ?;

-- name: SetKelasAktif :exec
UPDATE kelas SET is_aktif = ? WHERE id = ?;

-- name: DeleteKelas :exec
DELETE FROM kelas WHERE id = ?;

-- name: CountKelasByAngkatan :many
SELECT angkatan, COUNT(*) as total FROM kelas
GROUP BY angkatan
ORDER BY angkatan;

-- name: CountKelasTotal :one
SELECT COUNT(*) FROM kelas;
