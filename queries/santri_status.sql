-- name: UpdateSantriStatusDetail :exec
UPDATE santri
SET status = ?, status_alasan = ?, cuti_mulai = ?, cuti_selesai = ?, updated_at = ?
WHERE id = ?;

-- name: CreateSantriStatusLog :exec
INSERT INTO santri_status_log (
    santri_id, kelas_id, status_lama, status_baru, alasan, cuti_mulai, cuti_selesai, dibuat_oleh
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListSantriStatusLogByKelas :many
SELECT l.id, l.santri_id, s.nama AS santri_nama, l.status_lama, l.status_baru,
       l.alasan, l.cuti_mulai, l.cuti_selesai, COALESCE(u.name, '') AS dibuat_oleh_nama,
       l.created_at
FROM santri_status_log l
JOIN santri s ON s.id = l.santri_id
LEFT JOIN users u ON u.id = l.dibuat_oleh
WHERE l.kelas_id = ?
ORDER BY l.created_at DESC, l.id DESC
LIMIT 200;

-- name: ListSantriCutiBerakhir :many
-- cuti_selesai adalah hari terakhir cuti (YYYY-MM-DD).
SELECT * FROM santri
WHERE status = 'cuti' AND cuti_selesai != '' AND cuti_selesai < CAST(sqlc.arg(hari_ini) AS TEXT)
ORDER BY id;

-- name: ListSantriCutiTerjadwal :many
-- Santri aktif dengan cuti terjadwal yang tanggal mulainya sudah tiba.
SELECT * FROM santri
WHERE status = 'aktif' AND cuti_mulai != '' AND cuti_mulai <= CAST(sqlc.arg(hari_ini) AS TEXT)
ORDER BY id;

-- name: GetSantriRosterByKelasID :many
-- Semua santri di kelas lintas status, untuk tampilan Detail Kelas.
SELECT * FROM santri
WHERE kelas_id = ?
ORDER BY nama;
