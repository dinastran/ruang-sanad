package services

import (
	"database/sql"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

func tgl(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func insertTagihanDashboard(t *testing.T, db *sql.DB, kelasID, santriID, bulanKe, nominal int64, tagih, tempo time.Time, status string, bayar *time.Time) int64 {
	t.Helper()
	var tanggalBayar sql.NullTime
	if bayar != nil {
		tanggalBayar = sql.NullTime{Time: *bayar, Valid: true}
	}
	_, err := db.Exec(`INSERT INTO tagihan (santri_id, kelas_id, bulan_ke, pertemuan_ke, nominal, tanggal_tagih, jatuh_tempo, status, tanggal_bayar, angkatan_kelas) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '2026')`,
		santriID, kelasID, bulanKe, bulanKe*4, nominal, tagih, tempo, status, tanggalBayar)
	require.NoError(t, err)
	id, err := lastID(db)
	require.NoError(t, err)
	return id
}

func TestKeuanganDashboard(t *testing.T) {
	db, _, kelasID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	now := func() time.Time { return time.Date(2026, 3, 15, 9, 0, 0, 0, ZonaWaktuRiayah) }
	querier := queries.NewQuerier(db)
	ujroh := NewUjrohService(querier)
	ujroh.now = now
	service := NewKeuanganDashboardService(querier, ujroh)
	service.now = now

	akhirMaret, februari := tgl(2026, 3, 31), tgl(2026, 2, 10)
	// Terbit Maret: lunas dibayar di hari terakhir bulan, belum bayar, dan batal.
	insertTagihanDashboard(t, db, kelasID, santriID, 1, 100000, tgl(2026, 3, 2), tgl(2026, 3, 9), "lunas", &akhirMaret)
	insertTagihanDashboard(t, db, kelasID, santriID, 2, 100000, tgl(2026, 3, 3), tgl(2026, 3, 20), "belum_bayar", nil)
	insertTagihanDashboard(t, db, kelasID, santriID, 3, 100000, tgl(2026, 3, 4), tgl(2026, 3, 11), "batal", nil)
	// Februari: dibayar Februari (kas masuk Feb, bukan Maret).
	insertTagihanDashboard(t, db, kelasID, santriID, 4, 50000, tgl(2026, 2, 1), tgl(2026, 2, 8), "lunas", &februari)
	// Piutang terlambat 14, 54, dan 104 hari.
	insertTagihanDashboard(t, db, kelasID, santriID, 5, 20000, tgl(2026, 2, 22), tgl(2026, 3, 1), "belum_bayar", nil)
	insertTagihanDashboard(t, db, kelasID, santriID, 6, 30000, tgl(2026, 1, 13), tgl(2026, 1, 20), "belum_bayar", nil)
	tertua := insertTagihanDashboard(t, db, kelasID, santriID, 7, 40000, tgl(2025, 11, 24), tgl(2025, 12, 1), "belum_bayar", nil)

	_, err := db.Exec(`INSERT INTO ujroh_bulan (bulan, dikunci_at) VALUES ('2026-02', ?)`, tgl(2026, 3, 1))
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO ujroh_guru (bulan, guru_id, guru_nama, total, diterima, dibayar_at) VALUES ('2026-02', 1, 'Ustadz A', 300000, 300000, ?), ('2026-02', 2, 'Ustadz B', 200000, 200000, NULL)`, tgl(2026, 3, 5))
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO produk (nama, kategori, harga) VALUES ('Program X', 'program', 75000)`)
	require.NoError(t, err)
	produkID, err := lastID(db)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO product_orders (tanggal, nama_pembeli, sumber, status) VALUES ('2026-03-10', 'A', 'offline', 'aktif'), ('2026-03-11', 'B', 'offline', 'dibatalkan')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO product_order_items (order_id, produk_id, qty, harga_satuan) SELECT id, ?, 2, 75000 FROM product_orders`, produkID)
	require.NoError(t, err)

	d, err := service.Get("")
	require.NoError(t, err)
	require.Equal(t, "2026-03", d.Bulan)
	require.Equal(t, "Maret 2026", d.BulanLabel)
	require.Equal(t, "", d.BulanBerikut, "bulan berjalan tidak punya bulan berikut")
	require.Len(t, d.Tren, 6)
	require.Equal(t, "2025-10", d.Tren[0].Bulan)

	require.EqualValues(t, 100000, d.Sekarang.KasMasukSpp, "pembayaran tanggal 31 masuk bulan Maret")
	require.EqualValues(t, 150000, d.Sekarang.KasMasukProduk, "order dibatalkan tidak dihitung")
	require.EqualValues(t, 250000, d.Sekarang.KasMasuk)
	require.EqualValues(t, 300000, d.Sekarang.KasKeluar)
	require.EqualValues(t, -50000, d.Sekarang.KasBersih)
	require.EqualValues(t, 1, d.Produk.JumlahOrder)
	require.EqualValues(t, 150000, d.Produk.Omzet)
	require.EqualValues(t, 50000, d.Sebelumnya.KasMasukSpp)
	require.EqualValues(t, 200000, d.Sekarang.TerbitNominal, "tagihan batal tidak dihitung")
	require.InDelta(t, 50.0, d.Sekarang.Kolektibilitas, 0.01)

	require.EqualValues(t, 100000, d.Aging[0].Nominal)
	require.EqualValues(t, 20000, d.Aging[1].Nominal)
	require.EqualValues(t, 30000, d.Aging[2].Nominal)
	require.EqualValues(t, 40000, d.Aging[3].Nominal)
	require.EqualValues(t, 190000, d.TotalPiutang)
	require.EqualValues(t, 3, d.JumlahTerlambat)

	require.Len(t, d.Prioritas, 3)
	require.Equal(t, tertua, d.Prioritas[0].ID)
	require.EqualValues(t, 104, d.Prioritas[0].HariTelat)

	require.Len(t, d.PiutangKelas, 1)
	require.EqualValues(t, 190000, d.PiutangKelas[0].Nominal)
	require.Contains(t, d.PiutangKelas[0].Href, "kelas_id=")

	require.EqualValues(t, 200000, d.Ujroh.KewajibanTerkunci)
	require.EqualValues(t, 1, d.Ujroh.KewajibanTerkunciGuru)

	feb, err := service.Get("2026-02")
	require.NoError(t, err)
	require.Equal(t, "2026-03", feb.BulanBerikut)
	require.True(t, feb.Ujroh.Terkunci)
	require.EqualValues(t, 500000, feb.Ujroh.TotalUjroh)
	require.EqualValues(t, 300000, feb.Ujroh.TotalDibayar)

	_, err = service.Get("2026-13")
	require.ErrorIs(t, err, ErrUjrohBulanTidakValid)
}
