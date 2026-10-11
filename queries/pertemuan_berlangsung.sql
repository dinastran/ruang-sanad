-- Pertemuan yang masih berstatus berlangsung (belum ditutup guru).

-- name: ListPertemuanBerlangsung :many
SELECT p.id, p.kelas_id, p.pertemuan_ke, p.pertemuan_level_ke, p.tanggal, p.jam_mulai,
       p.created_at, p.guru_pengganti_id, p.pengingat_lupa_at,
       k.nama_kelas, k.guru_id,
       COALESCE(gu.nama, '') AS guru_nama,
       COALESCE(gb.nama, '') AS guru_pengganti_nama
FROM pertemuan p
JOIN kelas k ON k.id = p.kelas_id
LEFT JOIN guru gu ON gu.id = k.guru_id
LEFT JOIN guru gb ON gb.id = p.guru_pengganti_id
WHERE p.status = 'berlangsung'
ORDER BY p.created_at, p.id;

-- name: ListPertemuanLupaBelumDiingatkan :many
SELECT p.id, p.kelas_id, p.tanggal, p.jam_mulai, p.created_at, k.nama_kelas,
       gu.user_id AS guru_user_id, gb.user_id AS guru_pengganti_user_id
FROM pertemuan p
JOIN kelas k ON k.id = p.kelas_id
LEFT JOIN guru gu ON gu.id = k.guru_id
LEFT JOIN guru gb ON gb.id = p.guru_pengganti_id
WHERE p.status = 'berlangsung'
  AND p.pengingat_lupa_at IS NULL;

-- name: MarkPengingatLupaPertemuan :exec
UPDATE pertemuan SET pengingat_lupa_at = ? WHERE id = ?;

-- name: ResetJadwalPertemuanDariPertemuan :exec
-- Jadwal yang memulai pertemuan yang dibatalkan kembali bisa dimulai.
UPDATE jadwal_pertemuan
SET status = 'dijadwalkan', pertemuan_id = NULL, updated_at = CURRENT_TIMESTAMP
WHERE pertemuan_id = ? AND status = 'dimulai';

-- name: GetJadwalIDByPertemuan :one
SELECT id FROM jadwal_pertemuan WHERE pertemuan_id = ? LIMIT 1;

-- name: DeletePertemuanBerlangsung :execrows
DELETE FROM pertemuan WHERE id = ? AND kelas_id = ? AND status = 'berlangsung';
