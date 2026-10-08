-- ============ Kunjungan Kelas ============

-- name: CreateKunjungan :one
INSERT INTO kunjungan_kelas (
    guru_id, kelas_id, target_mulai, target_selesai, tanggal, jam, status, catatan,
    nilai_kedisiplinan, nilai_materi, nilai_metode, nilai_interaksi
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id;

-- name: UpdateKunjungan :execrows
UPDATE kunjungan_kelas
SET guru_id = ?, kelas_id = ?, target_mulai = ?, target_selesai = ?, tanggal = ?, jam = ?,
    status = ?, catatan = ?, nilai_kedisiplinan = ?, nilai_materi = ?, nilai_metode = ?,
    nilai_interaksi = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND dikirim_at IS NULL;

-- name: DeleteKunjungan :execrows
DELETE FROM kunjungan_kelas WHERE id = ? AND dikirim_at IS NULL;

-- name: ListKunjungan :many
SELECT kk.id, kk.guru_id, kk.kelas_id, kk.target_mulai, kk.target_selesai, kk.tanggal, kk.jam,
       kk.status, kk.catatan, kk.nilai_kedisiplinan, kk.nilai_materi, kk.nilai_metode,
       kk.nilai_interaksi, kk.dikirim_at, kk.dibaca_at, kk.tanggapan_guru, kk.tanggapan_at,
       g.nama AS guru_nama, g.no_wa AS guru_no_wa, g.user_id AS guru_user_id,
       COALESCE(k.nama_kelas, '') AS kelas_nama
FROM kunjungan_kelas kk
JOIN guru g ON g.id = kk.guru_id
LEFT JOIN kelas k ON k.id = kk.kelas_id
ORDER BY COALESCE(kk.tanggal, kk.target_mulai) DESC, kk.id DESC;

-- name: GetKunjungan :one
SELECT kk.id, kk.guru_id, kk.kelas_id, kk.target_mulai, kk.target_selesai, kk.tanggal, kk.jam,
       kk.status, kk.catatan, kk.nilai_kedisiplinan, kk.nilai_materi, kk.nilai_metode,
       kk.nilai_interaksi, kk.dikirim_at, kk.dibaca_at, kk.tanggapan_guru, kk.tanggapan_at,
       g.nama AS guru_nama, g.no_wa AS guru_no_wa, g.user_id AS guru_user_id,
       COALESCE(k.nama_kelas, '') AS kelas_nama
FROM kunjungan_kelas kk
JOIN guru g ON g.id = kk.guru_id
LEFT JOIN kelas k ON k.id = kk.kelas_id
WHERE kk.id = ?;

-- name: ListKunjunganTerkirimByGuru :many
SELECT kk.id, kk.guru_id, kk.kelas_id, kk.target_mulai, kk.target_selesai, kk.tanggal, kk.jam,
       kk.status, kk.catatan, kk.nilai_kedisiplinan, kk.nilai_materi, kk.nilai_metode,
       kk.nilai_interaksi, kk.dikirim_at, kk.dibaca_at, kk.tanggapan_guru, kk.tanggapan_at,
       g.nama AS guru_nama, g.no_wa AS guru_no_wa, g.user_id AS guru_user_id,
       COALESCE(k.nama_kelas, '') AS kelas_nama
FROM kunjungan_kelas kk
JOIN guru g ON g.id = kk.guru_id
LEFT JOIN kelas k ON k.id = kk.kelas_id
WHERE kk.guru_id = ? AND kk.dikirim_at IS NOT NULL
ORDER BY kk.tanggal DESC, kk.id DESC;

-- name: KirimKunjungan :exec
UPDATE kunjungan_kelas
SET dikirim_at = ?, dikirim_oleh = ?, dibaca_at = NULL, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: BukaKunciKunjungan :exec
UPDATE kunjungan_kelas
SET dikirim_at = NULL, dikirim_oleh = NULL, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: TandaiKunjunganDibaca :exec
UPDATE kunjungan_kelas
SET dibaca_at = ?
WHERE id = ? AND guru_id = ? AND dikirim_at IS NOT NULL AND dibaca_at IS NULL;

-- name: SimpanTanggapanKunjungan :execrows
UPDATE kunjungan_kelas
SET tanggapan_guru = sqlc.arg(tanggapan_guru), tanggapan_at = sqlc.arg(waktu),
    dibaca_at = COALESCE(dibaca_at, sqlc.arg(waktu)), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id) AND guru_id = sqlc.arg(guru_id) AND dikirim_at IS NOT NULL;

-- ============ Tindak Lanjut ============

-- name: CreateTindakLanjut :one
INSERT INTO kunjungan_tindak_lanjut (kunjungan_id, jenis, catatan, target_tanggal, dibuat_oleh)
VALUES (?, ?, ?, ?, ?) RETURNING id;

-- name: GetTindakLanjut :one
SELECT id, kunjungan_id, jenis, catatan, target_tanggal, status, selesai_at,
       kunjungan_berikutnya_id, dibuat_oleh, created_at, updated_at
FROM kunjungan_tindak_lanjut WHERE id = ?;

-- name: ListTindakLanjut :many
SELECT tl.id, tl.kunjungan_id, tl.jenis, tl.catatan, tl.target_tanggal, tl.status, tl.selesai_at,
       tl.kunjungan_berikutnya_id, kk.guru_id, g.nama AS guru_nama,
       COALESCE(k.nama_kelas, '') AS kelas_nama, kk.tanggal AS kunjungan_tanggal
FROM kunjungan_tindak_lanjut tl
JOIN kunjungan_kelas kk ON kk.id = tl.kunjungan_id
JOIN guru g ON g.id = kk.guru_id
LEFT JOIN kelas k ON k.id = kk.kelas_id
ORDER BY tl.status = 'selesai', tl.target_tanggal IS NULL, tl.target_tanggal, tl.id;

-- name: ListTindakLanjutByKunjungan :many
SELECT id, kunjungan_id, jenis, catatan, target_tanggal, status, selesai_at,
       kunjungan_berikutnya_id, dibuat_oleh, created_at, updated_at
FROM kunjungan_tindak_lanjut WHERE kunjungan_id = ?
ORDER BY id;

-- name: SetTindakLanjutKunjunganBerikutnya :exec
UPDATE kunjungan_tindak_lanjut
SET kunjungan_berikutnya_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateTindakLanjutStatus :exec
UPDATE kunjungan_tindak_lanjut
SET status = ?, selesai_at = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: SelesaikanMonitoringOlehKunjungan :exec
UPDATE kunjungan_tindak_lanjut
SET status = 'selesai', selesai_at = ?, updated_at = CURRENT_TIMESTAMP
WHERE kunjungan_berikutnya_id = ? AND status = 'terbuka';

-- name: DeleteTindakLanjut :exec
DELETE FROM kunjungan_tindak_lanjut WHERE id = ?;
