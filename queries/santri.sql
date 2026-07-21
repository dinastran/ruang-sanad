-- name: CreateSantri :execresult
INSERT INTO santri (
    id_mahasantri, kelas_kode, nama, jenis_kelamin, nominal, tanggal_daftar,
    angkatan, usia, domisili, tipe, frekuensi, is_lengkap, kelas_id,
    status, created_by, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?
);

-- name: GetSantriByID :one
SELECT * FROM santri WHERE id = ?;

-- name: ListSantri :many
SELECT * FROM santri
WHERE (@angkatan = '' OR angkatan = @angkatan)
  AND (@level = '' OR level = @level)
  AND (@tipe = '' OR tipe = @tipe)
  AND (@jadwal = '' OR jadwal = @jadwal)
  AND (@gender = '' OR jenis_kelamin = @gender)
  AND (@status = '' OR status = @status)
  AND (@kelas_id IS NULL OR kelas_id = @kelas_id)
  AND (@lengkap = -1 OR is_lengkap = @lengkap)
  AND (@search = '' OR nama LIKE '%' || @search || '%' OR id_mahasantri LIKE '%' || @search || '%')
ORDER BY id DESC
LIMIT @lim OFFSET @off;

-- name: CountSantri :one
SELECT COUNT(*) FROM santri
WHERE (@angkatan = '' OR angkatan = @angkatan)
  AND (@level = '' OR level = @level)
  AND (@tipe = '' OR tipe = @tipe)
  AND (@jadwal = '' OR jadwal = @jadwal)
  AND (@gender = '' OR jenis_kelamin = @gender)
  AND (@status = '' OR status = @status)
  AND (@kelas_id IS NULL OR kelas_id = @kelas_id)
  AND (@lengkap = -1 OR is_lengkap = @lengkap)
  AND (@search = '' OR nama LIKE '%' || @search || '%' OR id_mahasantri LIKE '%' || @search || '%');

-- name: UpdateSantriCS :exec
UPDATE santri
SET kelas_kode = ?, nama = ?, jenis_kelamin = ?, nominal = ?,
    tanggal_daftar = ?, angkatan = ?, usia = ?, domisili = ?,
    updated_at = ?
WHERE id = ?;

-- name: UpdateSantriAdminKelas :exec
UPDATE santri
SET fu = ?, tanggal_vn = ?, hasil_vn = ?, masuk_grup = ?,
    mulai_belajar = ?, jumlah = ?, level = ?, jadwal = ?,
    guru = ?, tipe = ?, frekuensi = ?, is_lengkap = ?,
    kelas_id = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateSantriKeuangan :exec
UPDATE santri
SET infaq_terakhir = ?, keterangan_tidak_lanjut = ?,
    status = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateSantriIdMahasantri :exec
UPDATE santri SET id_mahasantri = ? WHERE id = ?;

-- name: UpdateSantriKelas :exec
UPDATE santri SET kelas_id = ?, updated_at = ? WHERE id = ?;

-- name: UpdateSantriEngine :exec
UPDATE santri
SET tipe = ?, frekuensi = ?, is_lengkap = ?, kelas_id = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateSantriStatus :exec
UPDATE santri SET status = ?, updated_at = ? WHERE id = ?;

-- name: DeleteSantri :exec
DELETE FROM santri WHERE id = ?;

-- name: CountSantriByAngkatan :many
SELECT angkatan, COUNT(*) as total FROM santri
WHERE status = 'aktif'
GROUP BY angkatan
ORDER BY angkatan;

-- name: CountSantriByGender :many
SELECT jenis_kelamin, COUNT(*) as total FROM santri
WHERE status = 'aktif'
GROUP BY jenis_kelamin;

-- name: CountSantriByStatus :many
SELECT status, COUNT(*) as total FROM santri
GROUP BY status;

-- name: CountSantriByLevel :many
SELECT level, COUNT(*) as total FROM santri
WHERE status = 'aktif' AND level != ''
GROUP BY level
ORDER BY level;

-- name: CountSantriByTipe :many
SELECT tipe, COUNT(*) as total FROM santri
WHERE status = 'aktif' AND tipe != ''
GROUP BY tipe;

-- name: SumNominalByAngkatan :many
SELECT angkatan, SUM(nominal) as total_nominal FROM santri
WHERE status = 'aktif' AND nominal > 0
GROUP BY angkatan
ORDER BY angkatan;

-- name: GetPerluDilengkapi :many
SELECT * FROM santri
WHERE is_lengkap = 0 AND status = 'aktif'
ORDER BY id DESC;

-- name: GetSantriByKelasID :many
SELECT * FROM santri
WHERE kelas_id = ? AND status = 'aktif'
ORDER BY nama;
