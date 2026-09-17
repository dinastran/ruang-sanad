-- ============ Kriteria ============

-- name: ListTsiKriteria :many
SELECT * FROM tsi_kriteria ORDER BY
    CASE kategori WHEN 'kompetensi' THEN 1 WHEN 'kepuasan' THEN 2 WHEN 'kedisiplinan' THEN 3 ELSE 4 END,
    urutan ASC;

-- ============ Periode ============

-- name: GetTsiPeriode :one
SELECT * FROM tsi_periode WHERE guru_id = ? AND bulan = ?;

-- name: GetTsiPeriodeByID :one
SELECT * FROM tsi_periode WHERE id = ?;

-- name: CreateTsiPeriode :one
INSERT INTO tsi_periode (guru_id, bulan, status) VALUES (?, ?, 'draft') RETURNING id;

-- name: SetTsiPeriodeStatus :exec
UPDATE tsi_periode SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: ListTsiPeriodeByBulan :many
SELECT tp.*, g.nama AS guru_nama, g.status AS guru_status
FROM tsi_periode tp JOIN guru g ON g.id = tp.guru_id
WHERE tp.bulan = ? ORDER BY g.nama ASC;

-- ============ Nilai ============

-- name: ListTsiNilaiByPeriode :many
SELECT * FROM tsi_nilai WHERE periode_id = ?;

-- name: GetTsiNilai :one
SELECT * FROM tsi_nilai WHERE periode_id = ? AND kriteria_id = ?;

-- name: UpsertTsiNilai :exec
INSERT INTO tsi_nilai (periode_id, kriteria_id, nilai, is_override, alasan_override, catatan, updated_at)
VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(periode_id, kriteria_id) DO UPDATE SET
    nilai = excluded.nilai,
    is_override = excluded.is_override,
    alasan_override = excluded.alasan_override,
    catatan = excluded.catatan,
    updated_at = CURRENT_TIMESTAMP;

-- name: InsertTsiAudit :exec
INSERT INTO tsi_audit (periode_id, kriteria_id, nilai_lama, nilai_baru, alasan, oleh)
VALUES (?, ?, ?, ?, ?, ?);

-- ============ Aggregates for automatic indicators (range = [start, end]) ============

-- name: TsiCountPertemuanSelesai :one
SELECT COUNT(*) FROM pertemuan p JOIN kelas k ON k.id = p.kelas_id
WHERE k.guru_id = ? AND p.status = 'selesai' AND p.tanggal >= ? AND p.tanggal <= ?;

-- name: TsiCountPertemuanWithAbsensi :one
SELECT COUNT(*) FROM pertemuan p JOIN kelas k ON k.id = p.kelas_id
WHERE k.guru_id = ? AND p.status = 'selesai' AND p.tanggal >= ? AND p.tanggal <= ?
  AND EXISTS (SELECT 1 FROM absensi a WHERE a.pertemuan_id = p.id);

-- name: TsiCountReschedule :one
SELECT COUNT(*) FROM pertemuan p JOIN kelas k ON k.id = p.kelas_id
WHERE k.guru_id = ? AND p.is_reschedule = 1 AND p.tanggal >= ? AND p.tanggal <= ?;

-- name: TsiCountBadal :one
SELECT COUNT(*) FROM pertemuan p JOIN kelas k ON k.id = p.kelas_id
WHERE k.guru_id = ? AND p.is_badal = 1 AND p.tanggal >= ? AND p.tanggal <= ?;

-- name: TsiCountPertemuanTotal :one
SELECT COUNT(*) FROM pertemuan p JOIN kelas k ON k.id = p.kelas_id
WHERE k.guru_id = ? AND p.tanggal >= ? AND p.tanggal <= ?;

-- name: TsiCountSantriTotal :one
SELECT COUNT(*) FROM santri s JOIN kelas k ON k.id = s.kelas_id WHERE k.guru_id = ?;

-- name: TsiCountSantriTidakLanjut :one
SELECT COUNT(*) FROM santri s JOIN kelas k ON k.id = s.kelas_id WHERE k.guru_id = ? AND s.status = 'tidak_lanjut';

-- name: CountRapatTerlaksana :one
SELECT COUNT(*) FROM rapat_guru WHERE status = 'terlaksana' AND tanggal >= ? AND tanggal <= ?;

-- name: CountKalamInRange :one
SELECT COUNT(*) FROM kalam_bersanad WHERE tanggal >= ? AND tanggal <= ?;
