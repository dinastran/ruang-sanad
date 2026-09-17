-- name: CreateSantri :execresult
INSERT INTO santri (
    id_mahasantri, kelas_kode, nama, jenis_kelamin, nominal, tanggal_daftar,
    angkatan, angkatan_kelas, usia, domisili, no_wa, email, tipe, frekuensi, is_lengkap, kelas_id,
    status, created_by, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?
);

-- name: GetSantriByID :one
SELECT * FROM santri WHERE id = ?;

-- name: ListSantri :many
SELECT * FROM santri
WHERE (@angkatan_pendaftaran = '' OR angkatan = @angkatan_pendaftaran)
  AND (@angkatan_kelas = '' OR angkatan_kelas = @angkatan_kelas)
  AND (@level = '' OR level = @level)
  AND (@tipe = '' OR tipe = @tipe)
  AND (@jadwal = '' OR jadwal = @jadwal)
  AND (@gender = '' OR jenis_kelamin = @gender)
  AND (@status = '' OR status = @status)
  AND (@kelas_id IS NULL OR kelas_id = @kelas_id)
  AND (@lengkap = -1 OR is_lengkap = @lengkap)
  AND (@id_bermasalah = 0 OR (
      substr(id_mahasantri, 1, 4) != 'MHS.'
      OR length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6 <= 0
      OR substr(id_mahasantri, 5, length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6)
          != trim(substr(id_mahasantri, 5, length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6), ' ' || char(9) || char(10) || char(11) || char(12) || char(13))
      OR substr(id_mahasantri, 5 + length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6)
          != ('.' || printf('%04d', id) || '.' || substr(id_mahasantri, -6))
      OR substr(id_mahasantri, -6) NOT GLOB '[0-9][0-9][0-9][0-9][0-9][0-9]'
      OR CAST(substr(id_mahasantri, -6, 2) AS INTEGER) NOT BETWEEN 1 AND 12
      OR EXISTS (SELECT 1 FROM santri duplicate WHERE duplicate.id_mahasantri = santri.id_mahasantri AND duplicate.id != santri.id)
  ))
  AND (@search = '' OR nama LIKE '%' || @search || '%' OR id_mahasantri LIKE '%' || @search || '%')
ORDER BY id DESC
LIMIT @lim OFFSET @off;

-- name: CountSantri :one
SELECT COUNT(*) FROM santri
WHERE (@angkatan_pendaftaran = '' OR angkatan = @angkatan_pendaftaran)
  AND (@angkatan_kelas = '' OR angkatan_kelas = @angkatan_kelas)
  AND (@level = '' OR level = @level)
  AND (@tipe = '' OR tipe = @tipe)
  AND (@jadwal = '' OR jadwal = @jadwal)
  AND (@gender = '' OR jenis_kelamin = @gender)
  AND (@status = '' OR status = @status)
  AND (@kelas_id IS NULL OR kelas_id = @kelas_id)
  AND (@lengkap = -1 OR is_lengkap = @lengkap)
  AND (@id_bermasalah = 0 OR (
      substr(id_mahasantri, 1, 4) != 'MHS.'
      OR length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6 <= 0
      OR substr(id_mahasantri, 5, length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6)
          != trim(substr(id_mahasantri, 5, length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6), ' ' || char(9) || char(10) || char(11) || char(12) || char(13))
      OR substr(id_mahasantri, 5 + length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6)
          != ('.' || printf('%04d', id) || '.' || substr(id_mahasantri, -6))
      OR substr(id_mahasantri, -6) NOT GLOB '[0-9][0-9][0-9][0-9][0-9][0-9]'
      OR CAST(substr(id_mahasantri, -6, 2) AS INTEGER) NOT BETWEEN 1 AND 12
      OR EXISTS (SELECT 1 FROM santri duplicate WHERE duplicate.id_mahasantri = santri.id_mahasantri AND duplicate.id != santri.id)
  ))
  AND (@search = '' OR nama LIKE '%' || @search || '%' OR id_mahasantri LIKE '%' || @search || '%');

-- name: UpdateSantriCSMutable :exec
UPDATE santri
SET kelas_kode = ?, nama = ?, jenis_kelamin = ?, nominal = ?,
    usia = ?, domisili = ?, no_wa = ?, email = ?,
    updated_at = ?
WHERE id = ?;

-- name: IssueSantriRegistrationIdentity :exec
UPDATE santri
SET id_mahasantri = ?, angkatan = ?, tanggal_daftar = ?, updated_at = ?
WHERE id = ?;

-- name: CorrectSantriRegistrationIdentity :exec
UPDATE santri
SET id_mahasantri = ?, angkatan = ?, tanggal_daftar = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateSantriVoiceNote :exec
UPDATE santri
SET voice_note_url = ?, keterangan_vn = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateSantriAdminKelas :exec
UPDATE santri
SET fu = ?, tanggal_vn = ?, hasil_vn = ?, masuk_grup = ?,
    mulai_belajar = ?, jumlah = ?, angkatan_kelas = ?, level = ?, jadwal = ?,
    guru = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateSantriKeuangan :exec
UPDATE santri
SET infaq_terakhir = ?, keterangan_tidak_lanjut = ?,
    status = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateSantriIdMahasantri :exec
UPDATE santri SET id_mahasantri = ? WHERE id = ?;

-- name: UpdateSantriKelas :exec
UPDATE santri
SET kelas_id = ?, angkatan_kelas = ?, kelas_kode = ?, level = ?, jadwal = ?,
    tipe = ?, frekuensi = ?, is_lengkap = 1, updated_at = ?
WHERE id = ?;

-- name: ClearSantriKelasByKelasID :exec
UPDATE santri SET kelas_id = NULL, angkatan_kelas = '', is_lengkap = 0, updated_at = ? WHERE kelas_id = ?;

-- name: UpdateSantriPertemuanAwal :exec
UPDATE santri SET pertemuan_awal = ?, updated_at = ? WHERE id = ?;

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

-- name: ListDuplicateIDMahasantri :many
SELECT id_mahasantri FROM santri
WHERE id_mahasantri != ''
GROUP BY id_mahasantri
HAVING COUNT(*) > 1;
