package queries

import (
	"context"
	"database/sql"
)

// Hand-written like ujroh.go: `sqlc generate` currently fails on an unrelated
// query, so these statements live here instead of queries/*.sql.
//
// Dates are compared on their first 10 characters (YYYY-MM-DD) because the
// driver stores DATE/DATETIME values as full timestamps.

type LaporanPembayaranRow struct {
	TagihanID     int64
	TanggalBayar  string
	SantriID      int64
	IDMahasantri  string
	SantriNama    string
	AngkatanKelas string
	KelasNama     string
	BulanKe       int64
	Nominal       int64
	Metode        string
	Catatan       string
	DicatatOleh   string
}

// ListLaporanPembayaran returns SPP payments (lunas) whose payment date falls
// in [dari, sampai], oldest first.
func (q *Querier) ListLaporanPembayaran(ctx context.Context, dari, sampai string) ([]LaporanPembayaranRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT t.id, substr(t.tanggal_bayar, 1, 10), s.id, s.id_mahasantri, s.nama, COALESCE(NULLIF(t.angkatan_kelas, ''), s.angkatan_kelas),
	COALESCE(k.nama_kelas, ''), t.bulan_ke, t.nominal, t.metode, t.catatan, COALESCE(u.name, '')
FROM tagihan t
JOIN santri s ON s.id = t.santri_id
LEFT JOIN kelas k ON k.id = t.kelas_id
LEFT JOIN users u ON u.id = t.dicatat_oleh
WHERE t.status = 'lunas' AND t.tanggal_bayar IS NOT NULL
	AND substr(t.tanggal_bayar, 1, 10) BETWEEN ? AND ?
ORDER BY substr(t.tanggal_bayar, 1, 10), t.id`, dari, sampai)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LaporanPembayaranRow{}
	for rows.Next() {
		var r LaporanPembayaranRow
		if err := rows.Scan(&r.TagihanID, &r.TanggalBayar, &r.SantriID, &r.IDMahasantri, &r.SantriNama, &r.AngkatanKelas,
			&r.KelasNama, &r.BulanKe, &r.Nominal, &r.Metode, &r.Catatan, &r.DicatatOleh); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type LaporanProdukRow struct {
	JumlahOrder int64
	Omzet       int64
}

// LaporanOmzetProduk sums active product orders dated in [dari, sampai].
func (q *Querier) LaporanOmzetProduk(ctx context.Context, dari, sampai string) (LaporanProdukRow, error) {
	var r LaporanProdukRow
	err := q.Queries.db.QueryRowContext(ctx, `
SELECT COUNT(DISTINCT o.id), CAST(COALESCE(SUM(i.qty * i.harga_satuan), 0) AS INTEGER)
FROM product_orders o
JOIN product_order_items i ON i.order_id = o.id
WHERE o.status = 'aktif' AND substr(o.tanggal, 1, 10) BETWEEN ? AND ?`, dari, sampai).Scan(&r.JumlahOrder, &r.Omzet)
	return r, err
}

type LaporanUjrohDibayarRow struct {
	Bulan     string
	GuruNama  string
	Pokok     int64
	Komponen  int64
	Kasbon    int64
	Diterima  int64
	DibayarAt string
	Catatan   string
}

// ListLaporanUjrohDibayar returns ujroh marked paid in [dari, sampai].
func (q *Querier) ListLaporanUjrohDibayar(ctx context.Context, dari, sampai string) ([]LaporanUjrohDibayarRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT bulan, guru_nama, total, total_komponen, total_kasbon, diterima, substr(dibayar_at, 1, 10), catatan_bayar
FROM ujroh_guru
WHERE dibayar_at IS NOT NULL AND substr(dibayar_at, 1, 10) BETWEEN ? AND ?
ORDER BY substr(dibayar_at, 1, 10), guru_nama`, dari, sampai)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LaporanUjrohDibayarRow{}
	for rows.Next() {
		var r LaporanUjrohDibayarRow
		if err := rows.Scan(&r.Bulan, &r.GuruNama, &r.Pokok, &r.Komponen, &r.Kasbon, &r.Diterima, &r.DibayarAt, &r.Catatan); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type LaporanKasbonRow struct {
	Tanggal    string
	GuruNama   string
	Nominal    int64
	Keterangan string
}

// ListLaporanKasbon returns kasbon handed out in [dari, sampai] (not voided).
func (q *Querier) ListLaporanKasbon(ctx context.Context, dari, sampai string) ([]LaporanKasbonRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT k.tanggal, COALESCE(g.nama, ''), k.nominal, k.keterangan
FROM kasbon k
LEFT JOIN guru g ON g.id = k.guru_id
WHERE k.dibatalkan_at IS NULL AND k.tanggal BETWEEN ? AND ?
ORDER BY k.tanggal, k.id`, dari, sampai)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LaporanKasbonRow{}
	for rows.Next() {
		var r LaporanKasbonRow
		if err := rows.Scan(&r.Tanggal, &r.GuruNama, &r.Nominal, &r.Keterangan); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type LaporanTagihanTerbitRow struct {
	Jumlah       int64
	Nominal      int64
	JumlahLunas  int64
	NominalLunas int64
	JumlahBatal  int64
}

// LaporanTagihanTerbit summarises tagihan issued (tanggal_tagih) in [dari, sampai].
func (q *Querier) LaporanTagihanTerbit(ctx context.Context, dari, sampai string) (LaporanTagihanTerbitRow, error) {
	var r LaporanTagihanTerbitRow
	err := q.Queries.db.QueryRowContext(ctx, `
SELECT
	COUNT(CASE WHEN status != 'batal' THEN 1 END),
	CAST(COALESCE(SUM(CASE WHEN status != 'batal' THEN nominal END), 0) AS INTEGER),
	COUNT(CASE WHEN status = 'lunas' THEN 1 END),
	CAST(COALESCE(SUM(CASE WHEN status = 'lunas' THEN nominal END), 0) AS INTEGER),
	COUNT(CASE WHEN status = 'batal' THEN 1 END)
FROM tagihan
WHERE substr(tanggal_tagih, 1, 10) BETWEEN ? AND ?`, dari, sampai).
		Scan(&r.Jumlah, &r.Nominal, &r.JumlahLunas, &r.NominalLunas, &r.JumlahBatal)
	return r, err
}

type LaporanSantriRow struct {
	SantriID              int64
	IDMahasantri          string
	Nama                  string
	Angkatan              string
	AngkatanKelas         string
	KelasNama             string
	Status                string
	NominalSpp            int64
	KeteranganTidakLanjut string
	InfaqTerakhirManual   string
	TerbitPeriode         int64
	DibayarPeriode        int64
	BelumBayar            int64
	JumlahBelumBayar      int64
	Terlambat             int64
	JumlahTerlambat       int64
	// Pembayaran terakhir (any time), from the latest lunas tagihan.
	TerakhirTanggal sql.NullString
	TerakhirBulanKe sql.NullInt64
	TerakhirNominal sql.NullInt64
	TerakhirMetode  sql.NullString
}

// ListLaporanSantri returns one row per santri (filtered by status and
// angkatan kelas; empty = all) with billing totals for [dari, sampai],
// outstanding amounts as of hariIni, and their latest payment.
func (q *Querier) ListLaporanSantri(ctx context.Context, dari, sampai, hariIni, status, angkatanKelas string) ([]LaporanSantriRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
WITH terakhir AS (
	SELECT santri_id, substr(tanggal_bayar, 1, 10) AS tanggal, bulan_ke, nominal, metode,
		ROW_NUMBER() OVER (PARTITION BY santri_id ORDER BY substr(tanggal_bayar, 1, 10) DESC, bulan_ke DESC, id DESC) AS rn
	FROM tagihan
	WHERE status = 'lunas' AND tanggal_bayar IS NOT NULL
),
agg AS (
	SELECT santri_id,
		COALESCE(SUM(CASE WHEN status != 'batal' AND substr(tanggal_tagih, 1, 10) BETWEEN ?1 AND ?2 THEN nominal END), 0) AS terbit,
		COALESCE(SUM(CASE WHEN status = 'lunas' AND substr(tanggal_bayar, 1, 10) BETWEEN ?1 AND ?2 THEN nominal END), 0) AS dibayar,
		COALESCE(SUM(CASE WHEN status = 'belum_bayar' THEN nominal END), 0) AS belum,
		COUNT(CASE WHEN status = 'belum_bayar' THEN 1 END) AS jumlah_belum,
		COALESCE(SUM(CASE WHEN status = 'belum_bayar' AND jatuh_tempo IS NOT NULL AND substr(jatuh_tempo, 1, 10) < ?3 THEN nominal END), 0) AS terlambat,
		COUNT(CASE WHEN status = 'belum_bayar' AND jatuh_tempo IS NOT NULL AND substr(jatuh_tempo, 1, 10) < ?3 THEN 1 END) AS jumlah_terlambat
	FROM tagihan
	GROUP BY santri_id
)
SELECT s.id, s.id_mahasantri, s.nama, s.angkatan, s.angkatan_kelas, COALESCE(k.nama_kelas, ''), s.status, s.nominal,
	s.keterangan_tidak_lanjut, s.infaq_terakhir,
	COALESCE(a.terbit, 0), COALESCE(a.dibayar, 0), COALESCE(a.belum, 0), COALESCE(a.jumlah_belum, 0),
	COALESCE(a.terlambat, 0), COALESCE(a.jumlah_terlambat, 0),
	tr.tanggal, tr.bulan_ke, tr.nominal, tr.metode
FROM santri s
LEFT JOIN kelas k ON k.id = s.kelas_id
LEFT JOIN agg a ON a.santri_id = s.id
LEFT JOIN terakhir tr ON tr.santri_id = s.id AND tr.rn = 1
WHERE (?4 = '' OR s.status = ?4) AND (?5 = '' OR s.angkatan_kelas = ?5)
ORDER BY s.nama, s.id`, dari, sampai, hariIni, status, angkatanKelas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LaporanSantriRow{}
	for rows.Next() {
		var r LaporanSantriRow
		if err := rows.Scan(&r.SantriID, &r.IDMahasantri, &r.Nama, &r.Angkatan, &r.AngkatanKelas, &r.KelasNama, &r.Status, &r.NominalSpp,
			&r.KeteranganTidakLanjut, &r.InfaqTerakhirManual,
			&r.TerbitPeriode, &r.DibayarPeriode, &r.BelumBayar, &r.JumlahBelumBayar, &r.Terlambat, &r.JumlahTerlambat,
			&r.TerakhirTanggal, &r.TerakhirBulanKe, &r.TerakhirNominal, &r.TerakhirMetode); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
