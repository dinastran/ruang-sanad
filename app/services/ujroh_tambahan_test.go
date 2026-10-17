package services

import (
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func (f ujrohFixture) komponen(t *testing.T, nama, jenis string, nominal int64) int64 {
	t.Helper()
	require.NoError(t, f.svc.CreateKomponen(models.SimpanUjrohKomponenRequest{Nama: nama, Jenis: jenis, NominalDefault: nominal}, f.guruUtamaUser))
	list, err := f.svc.ListKomponen()
	require.NoError(t, err)
	for _, k := range list {
		if k.Nama == nama {
			return k.ID
		}
	}
	t.Fatalf("komponen %s tidak ada", nama)
	return 0
}

func (f ujrohFixture) kasbon(t *testing.T, nominal, cicilan int64, mulai string) int64 {
	t.Helper()
	require.NoError(t, f.svc.CreateKasbon(models.CreateKasbonRequest{GuruID: f.guruUtamaID, Tanggal: mulai + "-01", Nominal: nominal, Cicilan: cicilan, MulaiBulan: mulai, Keterangan: "keperluan keluarga"}, f.guruUtamaUser))
	list, err := f.svc.ListKasbon()
	require.NoError(t, err)
	return list[0].ID
}

func nominal(n int64) *int64 { return &n }

func TestUjrohKomponenTetapInsidentalDanGuruTanpaPertemuan(t *testing.T) {
	f := setupUjroh(t)
	f.pertemuan(t, 1, "2026-09-02", "selesai", f.guruUtamaUser, 0)
	f.pertemuan(t, 2, "2026-09-09", "selesai", f.guruUtamaUser, 0)

	kuota := f.komponen(t, "Kuota", UjrohKomponenTetap, 100000)
	mc := f.komponen(t, "MC", UjrohKomponenInsidental, 50000)
	require.ErrorIs(t, f.svc.CreateKomponen(models.SimpanUjrohKomponenRequest{Nama: "kuota", Jenis: UjrohKomponenTetap}, f.guruUtamaUser), ErrUjrohKomponenNamaDipakai)
	require.ErrorIs(t, f.svc.SetKomponenGuru(mc, f.guruUtamaID, nominal(1), f.guruUtamaUser), ErrUjrohKomponenBukanTetap)

	require.NoError(t, f.svc.SetKomponenGuru(kuota, f.guruUtamaID, nominal(100000), f.guruUtamaUser))
	require.NoError(t, f.svc.SetKomponenGuru(kuota, f.guruBadalID, nominal(80000), f.guruUtamaUser))
	require.NoError(t, f.svc.SetKomponenBulan("2026-09", f.guruUtamaID, mc, models.SetUjrohKomponenBulanRequest{Jumlah: 2, Tarif: 50000, Catatan: "Walimah"}, f.guruUtamaUser))
	require.ErrorIs(t, f.svc.SetKomponenBulan("2026-09", f.guruUtamaID, mc, models.SetUjrohKomponenBulanRequest{Jumlah: 0, Tarif: 50000}, f.guruUtamaUser), ErrUjrohKomponenJumlah)

	rekap, err := f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	utama := findGuru(t, rekap, f.guruUtamaID)
	require.EqualValues(t, 150000, utama.Total)
	require.EqualValues(t, 200000, utama.TotalKomponen, "kuota 100rb + MC 2x50rb")
	require.EqualValues(t, 350000, utama.Diterima)
	require.Len(t, utama.Komponen, 2)

	// Guru tanpa pertemuan tetap menerima komponen tetap.
	badal := findGuru(t, rekap, f.guruBadalID)
	require.EqualValues(t, 0, badal.JumlahPertemuan)
	require.EqualValues(t, 80000, badal.Diterima)
	require.EqualValues(t, 430000, rekap.Ringkasan.TotalUjroh)
	require.EqualValues(t, 150000, rekap.Ringkasan.TotalPokok)
	require.EqualValues(t, 280000, rekap.Ringkasan.TotalKomponen)

	// Komponen tetap bisa ditiadakan untuk bulan ini saja.
	require.NoError(t, f.svc.SetKomponenBulan("2026-09", f.guruBadalID, kuota, models.SetUjrohKomponenBulanRequest{Jumlah: 0, Tarif: 80000}, f.guruUtamaUser))
	require.NoError(t, f.svc.Kunci("2026-09", f.guruUtamaUser))

	rekap, err = f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	require.Len(t, rekap.Guru, 1, "guru tanpa pokok dan komponen tidak masuk rekap terkunci")
	utama = findGuru(t, rekap, f.guruUtamaID)
	require.EqualValues(t, 350000, utama.Diterima)
	require.Len(t, utama.Komponen, 2)

	// Setelah dikunci: input bulan ditolak dan perubahan master tidak mengubah angka.
	require.ErrorIs(t, f.svc.SetKomponenBulan("2026-09", f.guruUtamaID, mc, models.SetUjrohKomponenBulanRequest{Jumlah: 1, Tarif: 50000}, f.guruUtamaUser), ErrUjrohSudahDikunci)
	require.NoError(t, f.svc.SetKomponenGuru(kuota, f.guruUtamaID, nominal(500000), f.guruUtamaUser))
	rekap, err = f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	require.EqualValues(t, 350000, findGuru(t, rekap, f.guruUtamaID).Diterima)

	// Bulan berikutnya ikut nominal tetap terbaru, tanpa MC.
	oktober, err := f.svc.GetRekap("2026-10")
	require.NoError(t, err)
	require.EqualValues(t, 500000, findGuru(t, oktober, f.guruUtamaID).TotalKomponen)
}

func TestKasbonMengurangiUjrohSesuaiContoh(t *testing.T) {
	f := setupUjroh(t)
	f.pertemuan(t, 1, "2026-09-02", "selesai", f.guruUtamaUser, 0) // pokok 75rb
	kuota := f.komponen(t, "Tunjangan", UjrohKomponenTetap, 0)
	require.NoError(t, f.svc.SetKomponenGuru(kuota, f.guruUtamaID, nominal(1925000), f.guruUtamaUser))
	f.kasbon(t, 500000, 0, "2026-09")

	rekap, err := f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	utama := findGuru(t, rekap, f.guruUtamaID)
	require.EqualValues(t, 2000000, utama.Total+utama.TotalKomponen)
	require.EqualValues(t, 500000, utama.TotalKasbon)
	require.EqualValues(t, 1500000, utama.Diterima)
	require.EqualValues(t, 1500000, rekap.Ringkasan.TotalUjroh)
}

func TestKasbonCicilanDibatasiUjrohDanIkutKunci(t *testing.T) {
	f := setupUjroh(t)
	f.pertemuan(t, 1, "2026-08-05", "selesai", f.guruUtamaUser, 0) // Agustus: 75rb
	f.pertemuan(t, 2, "2026-09-02", "selesai", f.guruUtamaUser, 0)
	f.pertemuan(t, 3, "2026-09-09", "selesai", f.guruUtamaUser, 0) // September: 150rb
	id := f.kasbon(t, 300000, 100000, "2026-08")

	agustus, err := f.svc.GetRekap("2026-08")
	require.NoError(t, err)
	utama := findGuru(t, agustus, f.guruUtamaID)
	require.EqualValues(t, 75000, utama.TotalKasbon, "potongan tidak boleh melebihi ujroh")
	require.EqualValues(t, 0, utama.Diterima)
	require.NoError(t, f.svc.Kunci("2026-08", f.guruUtamaUser))

	require.ErrorIs(t, f.svc.BatalkanKasbon(id, f.guruUtamaUser), ErrKasbonSudahDipotong)
	require.ErrorIs(t, f.svc.CreateKasbon(models.CreateKasbonRequest{GuruID: f.guruUtamaID, Tanggal: "2026-08-20", Nominal: 1000, MulaiBulan: "2026-08"}, f.guruUtamaUser), ErrKasbonMulaiBulanDikunci)

	september, err := f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	utama = findGuru(t, september, f.guruUtamaID)
	require.Len(t, utama.Kasbon, 1)
	require.EqualValues(t, 225000, utama.Kasbon[0].SisaSebelum)
	require.EqualValues(t, 100000, utama.Kasbon[0].Potongan)
	require.EqualValues(t, 50000, utama.Diterima)
	require.NoError(t, f.svc.Kunci("2026-09", f.guruUtamaUser))

	list, err := f.svc.ListKasbon()
	require.NoError(t, err)
	require.EqualValues(t, 175000, list[0].Terpotong)
	require.EqualValues(t, 125000, list[0].Sisa)
	require.Equal(t, "berjalan", list[0].Status)

	// Rekap terkunci menampilkan potongan beku; membuka kunci mengembalikan sisa.
	september, err = f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	utama = findGuru(t, september, f.guruUtamaID)
	require.EqualValues(t, 100000, utama.TotalKasbon)
	require.EqualValues(t, 125000, utama.Kasbon[0].SisaSetelah)
	require.NoError(t, f.svc.Buka("2026-09", f.guruUtamaUser))
	list, err = f.svc.ListKasbon()
	require.NoError(t, err)
	require.EqualValues(t, 225000, list[0].Sisa)

	// Kasbon yang belum terpotong bisa dibatalkan dan tidak lagi memotong.
	baru := f.kasbon(t, 20000, 0, "2026-09")
	require.NoError(t, f.svc.BatalkanKasbon(baru, f.guruUtamaUser))
	september, err = f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	require.Len(t, findGuru(t, september, f.guruUtamaID).Kasbon, 1)
}

func TestKasKeluarDashboardMemakaiUjrohBersihDanKasbon(t *testing.T) {
	f := setupUjroh(t)
	f.pertemuan(t, 1, "2026-09-02", "selesai", f.guruUtamaUser, 0)
	f.kasbon(t, 50000, 0, "2026-09") // diberikan 1 Sep
	require.NoError(t, f.svc.Kunci("2026-09", f.guruUtamaUser))
	require.NoError(t, f.svc.TandaiDibayar("2026-09", f.guruUtamaID, f.guruUtamaUser, models.TandaiUjrohDibayarRequest{Tanggal: "2026-10-01"}))

	rows, err := f.querier.KeuanganUjrohKasKeluarPerBulan(t.Context(), "2026-09", "2026-10")
	require.NoError(t, err)
	got := map[string]int64{}
	for _, r := range rows {
		got[r.Bulan] = r.Nominal
	}
	require.EqualValues(t, 50000, got["2026-09"], "kasbon keluar saat diberikan")
	require.EqualValues(t, 25000, got["2026-10"], "ujroh 75rb dikurangi kasbon 50rb")
}
