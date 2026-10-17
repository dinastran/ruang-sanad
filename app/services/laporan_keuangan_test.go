package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

func TestLaporanKeuanganPeriode(t *testing.T) {
	db, _, kelasID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	_, err := db.Exec(`INSERT INTO santri (nama, status, infaq_terakhir, angkatan_kelas) VALUES ('Budi', 'aktif', 'Juni 2026 (catatan lama)', '2026')`)
	require.NoError(t, err)
	budiID, err := lastID(db)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO santri (nama, status, keterangan_tidak_lanjut) VALUES ('Umar', 'tidak_lanjut', 'Berhenti')`)
	require.NoError(t, err)

	svc := NewLaporanService(queries.NewQuerier(db), nil, nil)
	svc.now = func() time.Time { return time.Date(2026, 3, 15, 9, 0, 0, 0, ZonaWaktuRiayah) }

	feb, akhirMaret := tgl(2026, 2, 10), tgl(2026, 3, 31)
	insertTagihanDashboard(t, db, kelasID, santriID, 1, 100000, tgl(2026, 2, 1), tgl(2026, 2, 8), "lunas", &feb)
	lunas := insertTagihanDashboard(t, db, kelasID, santriID, 2, 100000, tgl(2026, 3, 1), tgl(2026, 3, 8), "lunas", &akhirMaret)
	_, err = db.Exec(`UPDATE tagihan SET metode = 'Transfer BSI' WHERE id = ?`, lunas)
	require.NoError(t, err)
	insertTagihanDashboard(t, db, kelasID, santriID, 3, 100000, tgl(2026, 3, 2), tgl(2026, 3, 9), "belum_bayar", nil)   // lewat tempo
	insertTagihanDashboard(t, db, kelasID, santriID, 4, 100000, tgl(2026, 3, 10), tgl(2026, 3, 20), "belum_bayar", nil) // belum tempo
	insertTagihanDashboard(t, db, kelasID, santriID, 5, 100000, tgl(2026, 3, 11), tgl(2026, 3, 18), "batal", nil)

	_, err = db.Exec(`INSERT INTO ujroh_bulan (bulan, dikunci_at) VALUES ('2026-02', ?)`, tgl(2026, 3, 1))
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO ujroh_guru (bulan, guru_id, guru_nama, total, total_komponen, total_kasbon, diterima, dibayar_at) VALUES ('2026-02', 1, 'Ustadz A', 300000, 50000, 100000, 250000, ?)`, tgl(2026, 3, 5))
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO guru (nama) VALUES ('Ustadz Kasbon')`)
	require.NoError(t, err)
	guruID, err := lastID(db)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO kasbon (guru_id, tanggal, nominal, mulai_bulan) VALUES (?, '2026-03-03', 40000, '2026-03'), (?, '2026-04-01', 99000, '2026-04')`, guruID, guruID)
	require.NoError(t, err)

	l, err := svc.GetLaporanKeuangan(models.LaporanKeuanganFilter{})
	require.NoError(t, err)
	require.Equal(t, "2026-03-01", l.Filter.Dari)
	require.Equal(t, "2026-03-31", l.Filter.Sampai)

	require.EqualValues(t, 100000, l.Ringkasan.PemasukanSpp, "hanya pembayaran bertanggal di periode")
	require.EqualValues(t, 1, l.Ringkasan.JumlahBayarSpp)
	require.EqualValues(t, 250000, l.Ringkasan.PengeluaranUjroh, "ujroh bersih")
	require.EqualValues(t, 40000, l.Ringkasan.PengeluaranKasbon)
	require.EqualValues(t, 100000-290000, l.Ringkasan.Saldo)
	require.Equal(t, "Transfer BSI", l.SppPerMetode[0].Nama)

	require.EqualValues(t, 3, l.Tagihan.Jumlah, "tagihan batal tidak dihitung")
	require.EqualValues(t, 1, l.Tagihan.JumlahBatal)
	require.EqualValues(t, 200000, l.Piutang.BelumBayar)
	require.EqualValues(t, 100000, l.Piutang.Terlambat)
	require.EqualValues(t, 1, l.Piutang.JumlahSantri)

	var utama, budi *models.LaporanSantri
	for i := range l.Santri {
		switch l.Santri[i].SantriID {
		case santriID:
			utama = &l.Santri[i]
		case budiID:
			budi = &l.Santri[i]
		}
	}
	require.NotNil(t, utama)
	require.NotNil(t, utama.PembayaranTerakhir, "pembayaran terakhir diambil dari tagihan lunas")
	require.Equal(t, "2026-03-31", utama.PembayaranTerakhir.Tanggal)
	require.EqualValues(t, 2, utama.PembayaranTerakhir.BulanKe)
	require.Equal(t, "Transfer BSI", utama.PembayaranTerakhir.Metode)
	require.EqualValues(t, 100000, utama.DibayarPeriode)
	require.EqualValues(t, 300000, utama.TerbitPeriode)
	require.NotNil(t, budi)
	require.Nil(t, budi.PembayaranTerakhir)
	require.Equal(t, "Juni 2026 (catatan lama)", budi.CatatanInfaqLama)
	require.Len(t, l.TidakLanjut, 1)
	require.Contains(t, l.OpsiAngkatan, "2026")

	// Filter tabel per santri tidak mengubah ringkasan dan piutang.
	l, err = svc.GetLaporanKeuangan(models.LaporanKeuanganFilter{Dari: "2026-02-01", Sampai: "2026-02-28", Status: "tidak_lanjut"})
	require.NoError(t, err)
	require.Len(t, l.Santri, 1)
	require.EqualValues(t, 100000, l.Ringkasan.PemasukanSpp)
	require.EqualValues(t, 0, l.Ringkasan.PengeluaranUjroh)
	require.EqualValues(t, 200000, l.Piutang.BelumBayar)

	_, err = svc.GetLaporanKeuangan(models.LaporanKeuanganFilter{Dari: "2026-03-10", Sampai: "2026-03-01"})
	require.ErrorIs(t, err, ErrLaporanPeriodeTidakValid)
}
