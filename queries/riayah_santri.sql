-- name: ListSantriRiayahScope :many
-- Santri aktif yang menjadi tanggung jawab riayah. guru_id NULL = semua santri.
SELECT s.id, s.nama, s.id_mahasantri, s.no_wa, s.jenis_kelamin,
    s.mulai_belajar, s.tanggal_daftar,
    k.id AS kelas_id, k.nama_kelas, k.level, k.jadwal,
    COALESCE(g.nama, '') AS guru_nama
FROM santri s
JOIN kelas k ON k.id = s.kelas_id
LEFT JOIN guru g ON g.id = k.guru_id
WHERE s.status = 'aktif'
  AND (sqlc.narg('guru_id') IS NULL OR k.guru_id = sqlc.narg('guru_id'))
ORDER BY k.nama_kelas, s.nama;

-- name: ListAbsensiTerakhirRiayah :many
-- 12 absensi terakhir per santri dalam scope, terbaru dulu.
SELECT santri_id, status, batas_materi, tanggal, urutan
FROM (
    SELECT a.santri_id, a.status, a.batas_materi, p.tanggal,
        ROW_NUMBER() OVER (PARTITION BY a.santri_id ORDER BY p.tanggal DESC, p.id DESC) AS urutan
    FROM absensi a
    JOIN pertemuan p ON p.id = a.pertemuan_id
    JOIN santri s ON s.id = a.santri_id
    JOIN kelas k ON k.id = s.kelas_id
    WHERE p.status = 'selesai'
      AND s.status = 'aktif'
      AND (sqlc.narg('guru_id') IS NULL OR k.guru_id = sqlc.narg('guru_id'))
)
WHERE urutan <= 12
ORDER BY santri_id, urutan;

-- name: RekapKehadiranRiayahSejak :many
-- Kehadiran (hadir + telat) per santri sejak tanggal tertentu.
SELECT a.santri_id,
    COUNT(*) AS total,
    SUM(CASE WHEN a.status IN ('hadir', 'telat') THEN 1 ELSE 0 END) AS total_hadir
FROM absensi a
JOIN pertemuan p ON p.id = a.pertemuan_id
JOIN santri s ON s.id = a.santri_id
JOIN kelas k ON k.id = s.kelas_id
WHERE p.status = 'selesai'
  AND s.status = 'aktif'
  AND p.tanggal >= CAST(sqlc.arg('sejak') AS TEXT)
  AND (sqlc.narg('guru_id') IS NULL OR k.guru_id = sqlc.narg('guru_id'))
GROUP BY a.santri_id;

-- name: ListTimelineAbsensiSantri :many
SELECT a.id, a.status, a.catatan, a.batas_materi,
    p.id AS pertemuan_id, p.tanggal, p.pertemuan_level_ke, p.materi,
    k.nama_kelas
FROM absensi a
JOIN pertemuan p ON p.id = a.pertemuan_id
JOIN kelas k ON k.id = p.kelas_id
WHERE a.santri_id = ? AND p.status = 'selesai'
ORDER BY p.tanggal DESC, p.id DESC
LIMIT 200;
