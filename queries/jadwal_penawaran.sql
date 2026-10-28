-- name: ListJadwalPenawaran :many
SELECT * FROM jadwal_penawaran ORDER BY urutan, id;

-- name: GetJadwalPenawaranByID :one
SELECT * FROM jadwal_penawaran WHERE id = ?;

-- name: GetJadwalPenawaranByKode :one
SELECT * FROM jadwal_penawaran WHERE kode = ?;

-- name: CreateJadwalPenawaran :one
INSERT INTO jadwal_penawaran (kode, hari, jam_mulai, jam_selesai, level, gender, kuota_per_gender, is_buka, urutan, catatan, periode_mulai, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateJadwalPenawaran :exec
UPDATE jadwal_penawaran SET
    hari = ?, jam_mulai = ?, jam_selesai = ?, level = ?, gender = ?, kuota_per_gender = ?,
    is_buka = ?, urutan = ?, catatan = ?, updated_at = ?
WHERE id = ?;

-- name: ResetPeriodeJadwalPenawaran :exec
UPDATE jadwal_penawaran SET periode_mulai = ?, updated_at = ? WHERE id = ?;

-- name: DeleteJadwalPenawaran :execrows
DELETE FROM jadwal_penawaran WHERE id = ?;

-- name: CountLeadsByJadwalKode :one
SELECT COUNT(*) FROM leads WHERE jadwal_kode = ?;

-- name: ListVerifiedLeadSeats :many
-- Kursi terpakai: lead dengan jadwal yang pembayarannya sudah diverifikasi
-- dan tidak batal. Pembanding periode dilakukan di service.
SELECT jadwal_kode, jenis_kelamin, verified_at FROM leads
WHERE jadwal_kode != '' AND verified_at IS NOT NULL AND status IN ('open', 'won');
