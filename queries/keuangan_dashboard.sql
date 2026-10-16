-- Dashboard keuangan. Date columns are stored by the driver as
-- "YYYY-MM-DD 00:00:00 +0000 UTC", so months/days are compared with substr.

-- name: KeuanganKasMasukSppPerBulan :many
-- Cash basis: SPP paid, grouped by the month it was paid.
SELECT CAST(substr(tanggal_bayar, 1, 7) AS TEXT) AS bulan,
       COUNT(*) AS jumlah,
       CAST(COALESCE(SUM(nominal), 0) AS INTEGER) AS nominal
FROM tagihan
WHERE status = 'lunas' AND tanggal_bayar IS NOT NULL
  AND substr(tanggal_bayar, 1, 7) >= CAST(sqlc.arg(dari_bulan) AS TEXT)
  AND substr(tanggal_bayar, 1, 7) <= CAST(sqlc.arg(sampai_bulan) AS TEXT)
GROUP BY substr(tanggal_bayar, 1, 7);

-- name: KeuanganKolektibilitasPerBulan :many
-- Billing basis: tagihan issued in a month and how much of it is paid.
SELECT CAST(substr(tanggal_tagih, 1, 7) AS TEXT) AS bulan,
       COUNT(*) AS jumlah,
       CAST(COALESCE(SUM(nominal), 0) AS INTEGER) AS nominal,
       CAST(COALESCE(SUM(CASE WHEN status = 'lunas' THEN 1 ELSE 0 END), 0) AS INTEGER) AS jumlah_lunas,
       CAST(COALESCE(SUM(CASE WHEN status = 'lunas' THEN nominal ELSE 0 END), 0) AS INTEGER) AS nominal_lunas
FROM tagihan
WHERE status != 'batal'
  AND substr(tanggal_tagih, 1, 7) >= CAST(sqlc.arg(dari_bulan) AS TEXT)
  AND substr(tanggal_tagih, 1, 7) <= CAST(sqlc.arg(sampai_bulan) AS TEXT)
GROUP BY substr(tanggal_tagih, 1, 7);

-- name: KeuanganUjrohDibayarPerBulan :many
-- Cash out: ujroh marked paid, grouped by the month it was paid.
SELECT CAST(substr(dibayar_at, 1, 7) AS TEXT) AS bulan,
       COUNT(*) AS jumlah,
       CAST(COALESCE(SUM(total), 0) AS INTEGER) AS nominal
FROM ujroh_guru
WHERE dibayar_at IS NOT NULL
  AND substr(dibayar_at, 1, 7) >= CAST(sqlc.arg(dari_bulan) AS TEXT)
  AND substr(dibayar_at, 1, 7) <= CAST(sqlc.arg(sampai_bulan) AS TEXT)
GROUP BY substr(dibayar_at, 1, 7);

-- name: KeuanganUjrohTerkunciBelumDibayar :one
SELECT COUNT(*) AS jumlah_guru,
       CAST(COALESCE(SUM(g.total), 0) AS INTEGER) AS nominal
FROM ujroh_guru g
JOIN ujroh_bulan b ON b.bulan = g.bulan
WHERE b.dikunci_at IS NOT NULL AND g.dibayar_at IS NULL;

-- name: KeuanganAgingPiutang :one
-- Outstanding tagihan as of hari_ini, bucketed by days past due date
-- (tanggal_tagih when jatuh_tempo is empty).
SELECT
  CAST(COALESCE(SUM(CASE WHEN hari <= 0 THEN 1 ELSE 0 END), 0) AS INTEGER) AS belum_tempo_jumlah,
  CAST(COALESCE(SUM(CASE WHEN hari <= 0 THEN nominal ELSE 0 END), 0) AS INTEGER) AS belum_tempo_nominal,
  CAST(COALESCE(SUM(CASE WHEN hari BETWEEN 1 AND 30 THEN 1 ELSE 0 END), 0) AS INTEGER) AS telat_30_jumlah,
  CAST(COALESCE(SUM(CASE WHEN hari BETWEEN 1 AND 30 THEN nominal ELSE 0 END), 0) AS INTEGER) AS telat_30_nominal,
  CAST(COALESCE(SUM(CASE WHEN hari BETWEEN 31 AND 60 THEN 1 ELSE 0 END), 0) AS INTEGER) AS telat_60_jumlah,
  CAST(COALESCE(SUM(CASE WHEN hari BETWEEN 31 AND 60 THEN nominal ELSE 0 END), 0) AS INTEGER) AS telat_60_nominal,
  CAST(COALESCE(SUM(CASE WHEN hari > 60 THEN 1 ELSE 0 END), 0) AS INTEGER) AS telat_lebih_jumlah,
  CAST(COALESCE(SUM(CASE WHEN hari > 60 THEN nominal ELSE 0 END), 0) AS INTEGER) AS telat_lebih_nominal
FROM (
  SELECT nominal,
         CAST(julianday(CAST(sqlc.arg(hari_ini) AS TEXT)) - julianday(substr(COALESCE(jatuh_tempo, tanggal_tagih), 1, 10)) AS INTEGER) AS hari
  FROM tagihan
  WHERE status = 'belum_bayar'
);

-- name: KeuanganPrioritasFollowUp :many
-- Overdue tagihan to chase first: longest overdue, then largest, then the
-- ones followed up least recently (never first).
SELECT t.*, s.nama AS santri_nama, s.id_mahasantri, s.no_wa, s.angkatan,
       s.frekuensi, k.nama_kelas, COALESCE(g.nama, '') AS guru_nama,
       CAST(julianday(CAST(sqlc.arg(hari_ini) AS TEXT)) - julianday(substr(COALESCE(t.jatuh_tempo, t.tanggal_tagih), 1, 10)) AS INTEGER) AS hari_telat
FROM tagihan t
JOIN santri s ON s.id = t.santri_id
LEFT JOIN kelas k ON k.id = t.kelas_id
LEFT JOIN guru g ON g.id = k.guru_id
WHERE t.status = 'belum_bayar'
  AND substr(COALESCE(t.jatuh_tempo, t.tanggal_tagih), 1, 10) < CAST(sqlc.arg(hari_ini) AS TEXT)
ORDER BY hari_telat DESC, t.nominal DESC, t.fu_terakhir IS NOT NULL, t.fu_terakhir ASC, t.id
LIMIT sqlc.arg('batas');

-- name: KeuanganPiutangPerKelas :many
SELECT t.kelas_id, CAST(COALESCE(k.nama_kelas, 'Tanpa kelas') AS TEXT) AS label,
       COUNT(DISTINCT t.santri_id) AS jumlah_santri,
       COUNT(*) AS jumlah_tagihan,
       CAST(COALESCE(SUM(t.nominal), 0) AS INTEGER) AS nominal
FROM tagihan t
LEFT JOIN kelas k ON k.id = t.kelas_id
WHERE t.status = 'belum_bayar'
GROUP BY t.kelas_id
ORDER BY nominal DESC, jumlah_tagihan DESC
LIMIT sqlc.arg('batas');

-- name: KeuanganPiutangPerAngkatan :many
SELECT CAST(t.angkatan_kelas AS TEXT) AS label,
       COUNT(DISTINCT t.santri_id) AS jumlah_santri,
       COUNT(*) AS jumlah_tagihan,
       CAST(COALESCE(SUM(t.nominal), 0) AS INTEGER) AS nominal
FROM tagihan t
WHERE t.status = 'belum_bayar'
GROUP BY t.angkatan_kelas
ORDER BY nominal DESC, jumlah_tagihan DESC
LIMIT sqlc.arg('batas');

-- name: KeuanganPenjualanProdukBulan :one
SELECT COUNT(DISTINCT o.id) AS jumlah_order,
       CAST(COALESCE(SUM(i.qty), 0) AS INTEGER) AS jumlah_qty,
       CAST(COALESCE(SUM(i.qty * i.harga_satuan), 0) AS INTEGER) AS omzet
FROM product_orders o
LEFT JOIN product_order_items i ON i.order_id = o.id
WHERE o.status = 'aktif' AND substr(o.tanggal, 1, 7) = CAST(sqlc.arg(bulan) AS TEXT);

-- name: KeuanganOmzetProdukPerBulan :many
-- Active product orders, grouped by order month; treated as cash in.
SELECT CAST(substr(o.tanggal, 1, 7) AS TEXT) AS bulan,
       CAST(COALESCE(SUM(i.qty * i.harga_satuan), 0) AS INTEGER) AS nominal
FROM product_orders o
JOIN product_order_items i ON i.order_id = o.id
WHERE o.status = 'aktif'
  AND substr(o.tanggal, 1, 7) >= CAST(sqlc.arg(dari_bulan) AS TEXT)
  AND substr(o.tanggal, 1, 7) <= CAST(sqlc.arg(sampai_bulan) AS TEXT)
GROUP BY substr(o.tanggal, 1, 7);
