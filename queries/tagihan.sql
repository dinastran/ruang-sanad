-- name: CreateTagihan :execresult
INSERT INTO tagihan (
    santri_id, kelas_id, pertemuan_id, bulan_ke, pertemuan_ke, nominal,
    tanggal_tagih, jatuh_tempo, angkatan_kelas
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(santri_id, bulan_ke) DO NOTHING;

-- name: GetTagihanByID :one
SELECT t.*, s.nama AS santri_nama, s.id_mahasantri, s.no_wa, s.angkatan,
       s.frekuensi, k.nama_kelas, COALESCE(g.nama, '') AS guru_nama
FROM tagihan t
JOIN santri s ON s.id = t.santri_id
LEFT JOIN kelas k ON k.id = t.kelas_id
LEFT JOIN guru g ON g.id = k.guru_id
WHERE t.id = ?;

-- name: ListTagihan :many
SELECT t.*, s.nama AS santri_nama, s.id_mahasantri, s.no_wa, s.angkatan,
       s.frekuensi, s.jenis_kelamin, s.level, k.nama_kelas,
       COALESCE(g.nama, '') AS guru_nama
FROM tagihan t
JOIN santri s ON s.id = t.santri_id
LEFT JOIN kelas k ON k.id = t.kelas_id
LEFT JOIN guru g ON g.id = k.guru_id
WHERE (sqlc.narg('status') IS NULL OR t.status = sqlc.narg('status'))
  AND (sqlc.narg('kelas_id') IS NULL OR t.kelas_id = sqlc.narg('kelas_id'))
  AND (sqlc.arg('angkatan_kelas') = '' OR t.angkatan_kelas = sqlc.arg('angkatan_kelas'))
  AND (sqlc.narg('guru_id') IS NULL OR k.guru_id = sqlc.narg('guru_id'))
  AND (sqlc.arg('frekuensi') = '' OR s.frekuensi = sqlc.arg('frekuensi'))
  AND (sqlc.arg('level') = '' OR s.level = sqlc.arg('level'))
  AND (sqlc.arg('gender') = '' OR s.jenis_kelamin = sqlc.arg('gender'))
  AND (sqlc.narg('bulan_ke') IS NULL OR t.bulan_ke = sqlc.narg('bulan_ke'))
  AND (sqlc.narg('tanggal_dari') IS NULL OR t.tanggal_tagih >= sqlc.narg('tanggal_dari'))
  AND (sqlc.narg('tanggal_sampai') IS NULL OR t.tanggal_tagih <= sqlc.narg('tanggal_sampai'))
  AND (sqlc.arg('search') = '' OR s.nama LIKE '%' || sqlc.arg('search') || '%' OR s.id_mahasantri LIKE '%' || sqlc.arg('search') || '%')
ORDER BY t.tanggal_tagih DESC, t.id DESC;

-- name: MarkTagihanLunas :exec
UPDATE tagihan
SET status = 'lunas', tanggal_bayar = ?, metode = ?, catatan = ?, dicatat_oleh = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = 'belum_bayar';

-- name: BatalkanTagihan :exec
UPDATE tagihan
SET status = 'batal', catatan = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status != 'lunas';

-- name: TouchFollowUp :exec
UPDATE tagihan
SET fu_terakhir = CURRENT_TIMESTAMP, fu_count = fu_count + 1, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: RingkasanTagihanPeriode :one
SELECT COUNT(*) AS total_tagihan,
       COALESCE(SUM(nominal), 0) AS nominal_tagihan,
       COALESCE(SUM(CASE WHEN status = 'lunas' THEN 1 ELSE 0 END), 0) AS total_lunas,
       COALESCE(SUM(CASE WHEN status = 'lunas' THEN nominal ELSE 0 END), 0) AS nominal_lunas,
       COALESCE(SUM(CASE WHEN status = 'belum_bayar' THEN 1 ELSE 0 END), 0) AS total_belum_bayar,
       COALESCE(SUM(CASE WHEN status = 'belum_bayar' THEN nominal ELSE 0 END), 0) AS nominal_belum_bayar,
       COALESCE(SUM(CASE WHEN status = 'belum_bayar' AND jatuh_tempo < DATE('now') THEN 1 ELSE 0 END), 0) AS total_terlambat
FROM tagihan
WHERE tanggal_tagih >= ? AND tanggal_tagih <= ?;

-- name: ListTagihanBelumBayar :many
SELECT t.*, s.nama AS santri_nama, s.id_mahasantri, s.no_wa, s.angkatan,
       s.frekuensi, k.nama_kelas, COALESCE(g.nama, '') AS guru_nama
FROM tagihan t
JOIN santri s ON s.id = t.santri_id
LEFT JOIN kelas k ON k.id = t.kelas_id
LEFT JOIN guru g ON g.id = k.guru_id
WHERE t.status = 'belum_bayar' AND t.tanggal_tagih >= ? AND t.tanggal_tagih <= ?
ORDER BY t.tanggal_tagih DESC, t.id DESC;

-- name: ListTagihanLunas :many
SELECT t.*, s.nama AS santri_nama, s.id_mahasantri, s.no_wa, s.angkatan,
       s.frekuensi, k.nama_kelas, COALESCE(g.nama, '') AS guru_nama
FROM tagihan t
JOIN santri s ON s.id = t.santri_id
LEFT JOIN kelas k ON k.id = t.kelas_id
LEFT JOIN guru g ON g.id = k.guru_id
WHERE t.status = 'lunas' AND t.tanggal_tagih >= ? AND t.tanggal_tagih <= ?
ORDER BY t.tanggal_tagih DESC, t.id DESC;

-- name: ListPertemuanSelesai :many
SELECT * FROM pertemuan WHERE status = 'selesai' ORDER BY tanggal, id;
