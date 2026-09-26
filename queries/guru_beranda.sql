-- name: ListKelasAktifBeranda :many
-- Kelas aktif dalam cakupan guru (NULL = semua) beserta jumlah santri aktif.
SELECT k.id, k.nama_kelas, k.level, k.jadwal, k.created_at,
    COALESCE(k.guru_id, 0) AS guru_id,
    COALESCE(g.nama, '') AS guru_nama,
    (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = k.id AND s.status = 'aktif') AS jumlah_santri
FROM kelas k
LEFT JOIN guru g ON g.id = k.guru_id
WHERE k.is_aktif = 1
  AND (sqlc.narg('guru_id') IS NULL OR k.guru_id = sqlc.narg('guru_id'))
ORDER BY k.nama_kelas;

-- name: ListJadwalPertemuanBeranda :many
-- Jadwal pertemuan (reschedule/badal/batal) yang tanggal baru atau tanggal
-- semulanya jatuh di rentang [mulai, selesai).
SELECT jp.id, jp.kelas_id, jp.tanggal, jp.jam_mulai, jp.status,
    jp.is_reschedule, jp.jadwal_semula,
    COALESCE(jp.guru_pengganti_id, 0) AS guru_pengganti_id,
    COALESCE(jp.pertemuan_id, 0) AS pertemuan_id,
    k.nama_kelas, k.level,
    COALESCE(k.guru_id, 0) AS guru_utama_id,
    COALESCE(gu.nama, '') AS guru_utama_nama,
    COALESCE(gb.nama, '') AS guru_pengganti_nama,
    (SELECT COUNT(*) FROM santri s WHERE s.kelas_id = k.id AND s.status = 'aktif') AS jumlah_santri
FROM jadwal_pertemuan jp
JOIN kelas k ON k.id = jp.kelas_id
LEFT JOIN guru gu ON gu.id = k.guru_id
LEFT JOIN guru gb ON gb.id = jp.guru_pengganti_id
WHERE (
        (jp.tanggal >= CAST(sqlc.arg('mulai') AS TEXT) AND jp.tanggal < CAST(sqlc.arg('selesai') AS TEXT))
     OR (jp.jadwal_semula >= CAST(sqlc.arg('mulai') AS TEXT) AND jp.jadwal_semula < CAST(sqlc.arg('selesai') AS TEXT))
  )
  AND (
      sqlc.narg('guru_id') IS NULL
      OR k.guru_id = sqlc.narg('guru_id')
      OR jp.guru_pengganti_id = sqlc.narg('guru_id')
  )
ORDER BY jp.tanggal, jp.jam_mulai, jp.id;

-- name: ListPertemuanKelasSejak :many
SELECT id, kelas_id, tanggal, status
FROM pertemuan
WHERE kelas_id IN (sqlc.slice('kelas_ids'))
  AND tanggal >= CAST(sqlc.arg('sejak') AS TEXT)
  AND status IN ('berlangsung', 'selesai')
ORDER BY tanggal, id;

-- name: GetTsiFinalTerakhir :one
SELECT bulan FROM tsi_periode
WHERE guru_id = ? AND status = 'final'
ORDER BY bulan DESC
LIMIT 1;

-- name: ListAgendaGuruMendatang :many
-- Agenda koordinator terdekat: pembinaan, rapat, dan kalam bersanad.
SELECT jenis, id, CAST(tanggal AS TEXT) AS tanggal, judul FROM (
    SELECT 'pembinaan' AS jenis, id, tanggal, topik AS judul FROM pembinaan
    WHERE status = 'dijadwalkan' AND tanggal >= CAST(sqlc.arg('sejak') AS TEXT)
    UNION ALL
    SELECT 'rapat' AS jenis, id, tanggal, judul FROM rapat_guru
    WHERE status = 'dijadwalkan' AND tanggal >= CAST(sqlc.arg('sejak') AS TEXT)
    UNION ALL
    SELECT 'kalam' AS jenis, id, tanggal, topik AS judul FROM kalam_bersanad
    WHERE tanggal >= CAST(sqlc.arg('sejak') AS TEXT)
)
ORDER BY tanggal, jenis
LIMIT 5;
