-- ============ Guru (helpers) ============

-- name: ListGuruAktifSimple :many
SELECT id, nama, status, no_wa FROM guru WHERE is_aktif = 1 ORDER BY nama ASC;

-- name: CountGuruAktifTotal :one
SELECT COUNT(*) FROM guru WHERE is_aktif = 1;

-- name: CountGuruByStatusAktif :one
SELECT COUNT(*) FROM guru WHERE is_aktif = 1 AND status = ?;

-- name: CountKelasAktifTotal :one
SELECT COUNT(*) FROM kelas WHERE is_aktif = 1;

-- name: CountSantriAktifTotal :one
SELECT COUNT(*) FROM santri WHERE status = 'aktif';

-- name: ListKelasAktifSimple :many
SELECT id, nama_kelas, guru_id FROM kelas WHERE is_aktif = 1 ORDER BY nama_kelas ASC;

-- name: ListGuruBelumAbsenPekanIni :many
SELECT DISTINCT g.id, g.nama, g.no_wa
FROM guru g
JOIN kelas k ON k.guru_id = g.id AND k.is_aktif = 1
WHERE g.is_aktif = 1
  AND NOT EXISTS (
    SELECT 1 FROM pertemuan p WHERE p.kelas_id = k.id AND p.tanggal >= date('now', '-7 days')
  )
ORDER BY g.nama ASC;

-- ============ Kompetensi ============

-- name: GetGuruKompetensi :one
SELECT * FROM guru_kompetensi WHERE guru_id = ?;

-- name: UpsertGuruKompetensi :exec
INSERT INTO guru_kompetensi (
    guru_id, hafalan_quran, hafalan_tuhfah, hafalan_jazariy, hafalan_khaqaniy,
    hafalan_syakhawiy, sanad_qiroah, bahasa_arab_pasif, bahasa_arab_aktif, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(guru_id) DO UPDATE SET
    hafalan_quran = excluded.hafalan_quran,
    hafalan_tuhfah = excluded.hafalan_tuhfah,
    hafalan_jazariy = excluded.hafalan_jazariy,
    hafalan_khaqaniy = excluded.hafalan_khaqaniy,
    hafalan_syakhawiy = excluded.hafalan_syakhawiy,
    sanad_qiroah = excluded.sanad_qiroah,
    bahasa_arab_pasif = excluded.bahasa_arab_pasif,
    bahasa_arab_aktif = excluded.bahasa_arab_aktif,
    updated_at = CURRENT_TIMESTAMP;

-- ============ Pembinaan ============

-- name: CreatePembinaan :one
INSERT INTO pembinaan (tanggal, bulan, pekan_ke, topik, keterangan, status)
VALUES (?, ?, ?, ?, ?, ?) RETURNING id;

-- name: ListPembinaan :many
SELECT * FROM pembinaan ORDER BY tanggal DESC;

-- name: GetPembinaan :one
SELECT * FROM pembinaan WHERE id = ?;

-- name: UpdatePembinaan :exec
UPDATE pembinaan SET tanggal = ?, bulan = ?, pekan_ke = ?, topik = ?, keterangan = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeletePembinaan :exec
DELETE FROM pembinaan WHERE id = ?;

-- name: UpsertPembinaanAbsen :exec
INSERT INTO pembinaan_absen (pembinaan_id, guru_id, hadir, kehadiran, jam_masuk, keterangan, alasan, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(pembinaan_id, guru_id) DO UPDATE SET hadir = excluded.hadir, kehadiran = excluded.kehadiran, jam_masuk = excluded.jam_masuk, keterangan = excluded.keterangan, alasan = excluded.alasan, updated_at = CURRENT_TIMESTAMP;

-- name: ListGuruWithPembinaanAbsen :many
SELECT g.id AS guru_id, g.nama, g.status,
    COALESCE(pa.hadir, 0) AS hadir, COALESCE(pa.kehadiran, '') AS kehadiran, COALESCE(pa.jam_masuk, '') AS jam_masuk,
    COALESCE(pa.keterangan, '') AS keterangan, COALESCE(pa.alasan, '') AS alasan
FROM guru g
LEFT JOIN pembinaan_absen pa ON pa.guru_id = g.id AND pa.pembinaan_id = ?
WHERE g.is_aktif = 1
ORDER BY g.nama ASC;

-- name: ListPembinaanAbsenTersimpan :many
SELECT g.id AS guru_id, g.nama, g.status, pa.hadir, pa.kehadiran, pa.jam_masuk, pa.keterangan, pa.alasan
FROM pembinaan_absen pa
JOIN guru g ON g.id = pa.guru_id
WHERE pa.pembinaan_id = ?
ORDER BY g.nama ASC;

-- name: CountPembinaanTerlaksana :one
SELECT COUNT(*) FROM pembinaan WHERE status = 'terlaksana' AND tanggal >= ? AND tanggal <= ?;

-- name: CountPembinaanHadirByGuru :one
SELECT COUNT(*) FROM pembinaan_absen pa
JOIN pembinaan p ON p.id = pa.pembinaan_id
WHERE pa.guru_id = ? AND pa.hadir = 1 AND p.status = 'terlaksana' AND p.tanggal >= ? AND p.tanggal <= ?;

-- name: ListKehadiranPembinaanTerlaksana :many
-- Bahan analisis alfa berturut-turut: semua absen guru aktif di sesi terlaksana, terbaru dulu.
SELECT pa.guru_id, p.id AS kegiatan_id, p.tanggal, pa.kehadiran
FROM pembinaan_absen pa
JOIN pembinaan p ON p.id = pa.pembinaan_id
JOIN guru g ON g.id = pa.guru_id
WHERE p.status = 'terlaksana' AND g.is_aktif = 1
ORDER BY pa.guru_id ASC, p.tanggal DESC, p.id DESC;

-- ============ Rapat Guru ============

-- name: CreateRapat :one
INSERT INTO rapat_guru (tanggal, judul, catatan, status) VALUES (?, ?, ?, ?) RETURNING id;

-- name: ListRapat :many
SELECT * FROM rapat_guru ORDER BY tanggal DESC;

-- name: GetRapat :one
SELECT * FROM rapat_guru WHERE id = ?;

-- name: UpdateRapat :exec
UPDATE rapat_guru SET tanggal = ?, judul = ?, catatan = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteRapat :exec
DELETE FROM rapat_guru WHERE id = ?;

-- name: UpsertRapatAbsen :exec
INSERT INTO rapat_absen (rapat_id, guru_id, hadir, kehadiran, jam_masuk, keterangan, alasan, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(rapat_id, guru_id) DO UPDATE SET hadir = excluded.hadir, kehadiran = excluded.kehadiran, jam_masuk = excluded.jam_masuk, keterangan = excluded.keterangan, alasan = excluded.alasan, updated_at = CURRENT_TIMESTAMP;

-- name: ListGuruWithRapatAbsen :many
SELECT g.id AS guru_id, g.nama, g.status,
    COALESCE(ra.hadir, 0) AS hadir, COALESCE(ra.kehadiran, '') AS kehadiran, COALESCE(ra.jam_masuk, '') AS jam_masuk,
    COALESCE(ra.keterangan, '') AS keterangan, COALESCE(ra.alasan, '') AS alasan
FROM guru g
LEFT JOIN rapat_absen ra ON ra.guru_id = g.id AND ra.rapat_id = ?
WHERE g.is_aktif = 1
ORDER BY g.nama ASC;

-- name: ListRapatAbsenTersimpan :many
SELECT g.id AS guru_id, g.nama, g.status, ra.hadir, ra.kehadiran, ra.jam_masuk, ra.keterangan, ra.alasan
FROM rapat_absen ra
JOIN guru g ON g.id = ra.guru_id
WHERE ra.rapat_id = ?
ORDER BY g.nama ASC;

-- name: CountRapatHadirByGuru :one
SELECT COUNT(*) FROM rapat_absen ra
JOIN rapat_guru r ON r.id = ra.rapat_id
WHERE ra.guru_id = ? AND ra.hadir = 1 AND r.status = 'terlaksana' AND r.tanggal >= ? AND r.tanggal <= ?;

-- name: ListKehadiranRapatTerlaksana :many
-- Bahan analisis alfa berturut-turut: semua absen guru aktif di rapat terlaksana, terbaru dulu.
SELECT ra.guru_id, r.id AS kegiatan_id, r.tanggal, ra.kehadiran
FROM rapat_absen ra
JOIN rapat_guru r ON r.id = ra.rapat_id
JOIN guru g ON g.id = ra.guru_id
WHERE r.status = 'terlaksana' AND g.is_aktif = 1
ORDER BY ra.guru_id ASC, r.tanggal DESC, r.id DESC;

-- name: ListRiwayatAbsensiGuru :many
SELECT pa.guru_id, g.nama AS guru_nama, 'pembinaan' AS kegiatan, p.id AS kegiatan_id,
       p.tanggal, p.topik AS judul, pa.hadir, pa.kehadiran, pa.jam_masuk, pa.keterangan, pa.alasan
FROM pembinaan_absen pa
JOIN pembinaan p ON p.id = pa.pembinaan_id
JOIN guru g ON g.id = pa.guru_id
UNION ALL
SELECT ra.guru_id, g.nama AS guru_nama, 'rapat' AS kegiatan, r.id AS kegiatan_id,
       r.tanggal, r.judul, ra.hadir, ra.kehadiran, ra.jam_masuk, ra.keterangan, ra.alasan
FROM rapat_absen ra
JOIN rapat_guru r ON r.id = ra.rapat_id
JOIN guru g ON g.id = ra.guru_id
UNION ALL
SELECT k.guru_id, g.nama AS guru_nama, 'kunjungan' AS kegiatan, k.id AS kegiatan_id,
       k.tanggal, COALESCE(kl.nama_kelas, '') AS judul,
       CASE WHEN k.status = 'terlaksana' THEN 1 ELSE 0 END AS hadir,
       '' AS kehadiran,
       k.jam AS jam_masuk, k.catatan AS keterangan,
       CASE WHEN k.status = 'terlaksana' THEN '' ELSE k.status END AS alasan
FROM kunjungan_kelas k
JOIN guru g ON g.id = k.guru_id
LEFT JOIN kelas kl ON kl.id = k.kelas_id
WHERE k.tanggal IS NOT NULL
ORDER BY tanggal DESC, guru_nama ASC;

-- ============ Tilawah Harian ============

-- name: CreateTilawah :exec
INSERT INTO tilawah_harian (guru_id, tanggal) VALUES (?, ?) ON CONFLICT(guru_id, tanggal) DO NOTHING;

-- name: DeleteTilawah :exec
DELETE FROM tilawah_harian WHERE guru_id = ? AND tanggal = ?;

-- name: ExistsTilawah :one
SELECT COUNT(*) FROM tilawah_harian WHERE guru_id = ? AND tanggal = ?;

-- name: CountTilawahByGuruRange :one
SELECT COUNT(*) FROM tilawah_harian WHERE guru_id = ? AND tanggal >= ? AND tanggal <= ?;

-- name: ListTilawahByGuruRange :many
SELECT tanggal FROM tilawah_harian WHERE guru_id = ? AND tanggal >= ? AND tanggal <= ? ORDER BY tanggal ASC;

-- ============ Kalam Bersanad ============

-- name: CreateKalam :one
INSERT INTO kalam_bersanad (tanggal, topik, kitab, keterangan) VALUES (?, ?, ?, ?) RETURNING id;

-- name: ListKalam :many
SELECT kb.*, (SELECT COUNT(*) FROM kalam_share_log ksl WHERE ksl.kalam_id = kb.id) AS total_share
FROM kalam_bersanad kb ORDER BY kb.tanggal DESC;

-- name: GetKalam :one
SELECT * FROM kalam_bersanad WHERE id = ?;

-- name: UpdateKalam :exec
UPDATE kalam_bersanad SET tanggal = ?, topik = ?, kitab = ?, keterangan = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteKalam :exec
DELETE FROM kalam_bersanad WHERE id = ?;

-- name: ToggleKalamShareOn :exec
INSERT INTO kalam_share_log (kalam_id, guru_id) VALUES (?, ?) ON CONFLICT(kalam_id, guru_id) DO NOTHING;

-- name: ToggleKalamShareOff :exec
DELETE FROM kalam_share_log WHERE kalam_id = ? AND guru_id = ?;

-- name: ListGuruWithKalamShare :many
SELECT g.id AS guru_id, g.nama,
    CASE WHEN ksl.id IS NULL THEN 0 ELSE 1 END AS sudah_share
FROM guru g
LEFT JOIN kalam_share_log ksl ON ksl.guru_id = g.id AND ksl.kalam_id = ?
WHERE g.is_aktif = 1
ORDER BY g.nama ASC;

-- name: CountKalamShareByGuruRange :one
SELECT COUNT(*) FROM kalam_share_log ksl
JOIN kalam_bersanad kb ON kb.id = ksl.kalam_id
WHERE ksl.guru_id = ? AND kb.tanggal >= ? AND kb.tanggal <= ?;

-- ============ WA Template ============

-- name: CreateWaTemplate :one
INSERT INTO wa_template (nama, target_type, body, is_aktif) VALUES (?, ?, ?, ?) RETURNING id;

-- name: ListWaTemplate :many
SELECT * FROM wa_template ORDER BY target_type, nama ASC;

-- name: GetWaTemplate :one
SELECT * FROM wa_template WHERE id = ?;

-- name: UpdateWaTemplate :exec
UPDATE wa_template SET nama = ?, target_type = ?, body = ?, is_aktif = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteWaTemplate :exec
DELETE FROM wa_template WHERE id = ?;

-- ============ Dashboard harian ============

-- name: ListDashboardTindakLanjut :many
SELECT tl.id, tl.kunjungan_id, tl.jenis, tl.catatan, tl.target_tanggal, g.nama AS guru_nama
FROM kunjungan_tindak_lanjut tl
JOIN kunjungan_kelas kk ON kk.id = tl.kunjungan_id
JOIN guru g ON g.id = kk.guru_id
WHERE tl.status = 'terbuka'
  AND tl.target_tanggal IS NOT NULL
  AND substr(tl.target_tanggal, 1, 10) <= substr(sqlc.arg(until_date), 1, 10)
ORDER BY substr(tl.target_tanggal, 1, 10) ASC, tl.id ASC;

-- name: ListDashboardTodo :many
SELECT id, judul, deadline, pic, status
FROM todo_koordinator
WHERE status IN ('belum', 'proses')
  AND deadline IS NOT NULL
  AND substr(deadline, 1, 10) <= substr(sqlc.arg(until_date), 1, 10)
ORDER BY substr(deadline, 1, 10) ASC, id ASC;

-- name: ListDashboardPembinaan :many
SELECT id, tanggal, topik
FROM pembinaan
WHERE status = 'dijadwalkan'
  AND substr(tanggal, 1, 10) >= substr(sqlc.arg(start_date), 1, 10)
  AND substr(tanggal, 1, 10) <= substr(sqlc.arg(end_date), 1, 10)
ORDER BY substr(tanggal, 1, 10) ASC, id ASC;

-- name: ListDashboardRapat :many
SELECT id, tanggal, judul
FROM rapat_guru
WHERE status = 'dijadwalkan'
  AND substr(tanggal, 1, 10) >= substr(sqlc.arg(start_date), 1, 10)
  AND substr(tanggal, 1, 10) <= substr(sqlc.arg(end_date), 1, 10)
ORDER BY substr(tanggal, 1, 10) ASC, id ASC;

-- name: ListDashboardKunjungan :many
SELECT kk.id, kk.tanggal, kk.jam, g.nama AS guru_nama, COALESCE(k.nama_kelas, '') AS nama_kelas
FROM kunjungan_kelas kk
JOIN guru g ON g.id = kk.guru_id
LEFT JOIN kelas k ON k.id = kk.kelas_id
WHERE kk.status = 'dijadwalkan'
  AND kk.tanggal IS NOT NULL
  AND substr(kk.tanggal, 1, 10) >= substr(sqlc.arg(start_date), 1, 10)
  AND substr(kk.tanggal, 1, 10) <= substr(sqlc.arg(end_date), 1, 10)
ORDER BY substr(kk.tanggal, 1, 10) ASC, kk.jam ASC, kk.id ASC;

-- name: CountGuruTsiBelumFinal :one
SELECT COUNT(*)
FROM guru g
WHERE g.is_aktif = 1
  AND NOT EXISTS (
    SELECT 1 FROM tsi_periode tp
    WHERE tp.guru_id = g.id AND tp.bulan = sqlc.arg(bulan) AND tp.status = 'final'
  );
