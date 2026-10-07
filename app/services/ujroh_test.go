package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

type ujrohFixture struct {
	jadwalTestFixture
	svc *UjrohService
}

func setupUjroh(t *testing.T) ujrohFixture {
	t.Helper()
	f := setupJadwalPertemuanService(t)
	_, err := f.db.Exec(`UPDATE guru SET status = 'part_time' WHERE id = ?`, f.guruBadalID)
	require.NoError(t, err)
	svc := NewUjrohService(f.querier)
	svc.now = func() time.Time { return time.Date(2026, 10, 7, 9, 0, 0, 0, ZonaWaktuRiayah) }
	return ujrohFixture{jadwalTestFixture: f, svc: svc}
}

func (f ujrohFixture) pertemuan(t *testing.T, ke int64, tanggal any, status string, dibuatOleh int64, badalGuru int64) {
	t.Helper()
	var badal any
	isBadal := 0
	if badalGuru > 0 {
		badal, isBadal = badalGuru, 1
	}
	_, err := f.db.Exec(`INSERT INTO pertemuan (kelas_id, pertemuan_ke, tanggal, status, dibuat_oleh, is_badal, guru_pengganti_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		f.kelasID, ke, tanggal, status, dibuatOleh, isBadal, badal)
	require.NoError(t, err)
}

func findGuru(t *testing.T, rekap *models.UjrohRekap, guruID int64) models.UjrohGuru {
	t.Helper()
	for _, g := range rekap.Guru {
		if g.GuruID == guruID {
			return g
		}
	}
	t.Fatalf("guru %d tidak ada di rekap", guruID)
	return models.UjrohGuru{}
}

func TestUjrohRekapMenghitungTarifStatusDanBadal(t *testing.T) {
	f := setupUjroh(t)
	// Stored the way the app stores it (driver timestamp) and as a plain date.
	f.pertemuan(t, 1, time.Date(2026, 9, 2, 8, 0, 0, 0, ZonaWaktuRiayah), "selesai", f.guruUtamaUser, 0)
	f.pertemuan(t, 2, "2026-09-09", "selesai", f.guruUtamaUser, 0)
	f.pertemuan(t, 3, "2026-09-16", "selesai", f.guruUtamaUser, f.guruBadalID)
	f.pertemuan(t, 4, "2026-09-23", "berlangsung", f.guruUtamaUser, 0)
	f.pertemuan(t, 5, "2026-10-01", "selesai", f.guruUtamaUser, 0)

	rekap, err := f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	require.False(t, rekap.Terkunci)
	require.EqualValues(t, 1, rekap.Berlangsung)

	utama := findGuru(t, rekap, f.guruUtamaID)
	require.EqualValues(t, 2, utama.JumlahPertemuan)
	require.EqualValues(t, 75000, utama.Tarif)
	require.EqualValues(t, 150000, utama.Total)

	badal := findGuru(t, rekap, f.guruBadalID)
	require.EqualValues(t, 1, badal.JumlahPertemuan)
	require.EqualValues(t, 1, badal.JumlahBadal)
	require.EqualValues(t, 50000, badal.Total)

	require.EqualValues(t, 200000, rekap.Ringkasan.TotalUjroh)
	require.EqualValues(t, 3, rekap.Ringkasan.JumlahPertemuan)
}

func TestUjrohTarifKhususDanTarifDefaultBisaDiubah(t *testing.T) {
	f := setupUjroh(t)
	f.pertemuan(t, 1, "2026-09-02", "selesai", f.guruUtamaUser, 0)
	f.pertemuan(t, 2, "2026-09-09", "selesai", f.guruUtamaUser, f.guruBadalID)

	require.NoError(t, f.svc.UpdateTarif(models.UpdateUjrohTarifRequest{Tetap: 80000, PartTime: 55000}, f.guruUtamaUser))
	khusus := int64(90000)
	require.NoError(t, f.svc.SetTarifGuru(f.guruUtamaID, &khusus, f.guruUtamaUser))

	rekap, err := f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	utama := findGuru(t, rekap, f.guruUtamaID)
	require.True(t, utama.TarifKhusus)
	require.EqualValues(t, 90000, utama.Total)
	require.EqualValues(t, 55000, findGuru(t, rekap, f.guruBadalID).Total)

	require.NoError(t, f.svc.SetTarifGuru(f.guruUtamaID, nil, f.guruUtamaUser))
	rekap, err = f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	require.EqualValues(t, 80000, findGuru(t, rekap, f.guruUtamaID).Total)

	negatif := int64(-1)
	require.ErrorIs(t, f.svc.SetTarifGuru(f.guruUtamaID, &negatif, f.guruUtamaUser), ErrUjrohNominalTidakSah)
}

func TestUjrohKunciMembekukanAngkaDanMenandaiSelisih(t *testing.T) {
	f := setupUjroh(t)
	f.pertemuan(t, 1, "2026-09-02", "selesai", f.guruUtamaUser, 0)
	f.pertemuan(t, 2, "2026-09-09", "selesai", f.guruUtamaUser, 0)

	require.NoError(t, f.svc.Kunci("2026-09", f.guruUtamaUser))
	require.ErrorIs(t, f.svc.Kunci("2026-09", f.guruUtamaUser), ErrUjrohSudahDikunci)

	// Rate change and a late-completed meeting after locking.
	require.NoError(t, f.svc.UpdateTarif(models.UpdateUjrohTarifRequest{Tetap: 100000, PartTime: 50000}, f.guruUtamaUser))
	f.pertemuan(t, 3, "2026-09-30", "selesai", f.guruUtamaUser, f.guruBadalID)

	rekap, err := f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	require.True(t, rekap.Terkunci)
	utama := findGuru(t, rekap, f.guruUtamaID)
	require.EqualValues(t, 150000, utama.Total, "angka terkunci tidak boleh berubah")
	require.Len(t, utama.Pertemuan, 2)
	require.True(t, utama.Selisih)
	require.EqualValues(t, 200000, utama.LiveTotal)
	require.Len(t, rekap.GuruBaruSelisih, 1)
	require.Equal(t, f.guruBadalID, rekap.GuruBaruSelisih[0].GuruID)
	require.EqualValues(t, 150000, rekap.Ringkasan.TotalUjroh)
}

func TestUjrohPembayaranHanyaSetelahKunciDanMenahanBukaKunci(t *testing.T) {
	f := setupUjroh(t)
	f.pertemuan(t, 1, "2026-09-02", "selesai", f.guruUtamaUser, 0)
	req := models.TandaiUjrohDibayarRequest{Tanggal: "2026-10-05", Catatan: "transfer"}

	require.ErrorIs(t, f.svc.TandaiDibayar("2026-09", f.guruUtamaID, f.guruUtamaUser, req), ErrUjrohBelumDikunci)
	require.NoError(t, f.svc.Kunci("2026-09", f.guruUtamaUser))
	require.ErrorIs(t, f.svc.TandaiDibayar("2026-09", f.guruBadalID, f.guruUtamaUser, req), ErrUjrohGuruTidakAda)
	require.NoError(t, f.svc.TandaiDibayar("2026-09", f.guruUtamaID, f.guruUtamaUser, req))

	rekap, err := f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	utama := findGuru(t, rekap, f.guruUtamaID)
	require.True(t, utama.Dibayar)
	require.Equal(t, "2026-10-05", utama.DibayarAt)
	require.Equal(t, "transfer", utama.CatatanBayar)
	require.EqualValues(t, 75000, rekap.Ringkasan.TotalDibayar)
	require.EqualValues(t, 0, rekap.Ringkasan.TotalBelum)

	require.ErrorIs(t, f.svc.Buka("2026-09", f.guruUtamaUser), ErrUjrohSudahAdaDibayar)
	require.NoError(t, f.svc.BatalDibayar("2026-09", f.guruUtamaID))
	require.NoError(t, f.svc.Buka("2026-09", f.guruUtamaUser))
	require.ErrorIs(t, f.svc.Buka("2026-09", f.guruUtamaUser), ErrUjrohBelumDikunci)

	rekap, err = f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	require.False(t, rekap.Terkunci)
	require.NotEmpty(t, rekap.DibukaAt)
	require.EqualValues(t, 75000, rekap.Ringkasan.TotalUjroh)
}

func TestUjrohPertemuanTanpaGuruDanValidasiBulan(t *testing.T) {
	f := setupUjroh(t)
	_, err := f.db.Exec(`UPDATE kelas SET guru_id = NULL WHERE id = ?`, f.kelasID)
	require.NoError(t, err)
	adminUser := insertTestUser(t, f.db, "admin@example.com", "Admin")
	f.pertemuan(t, 1, "2026-09-02", "selesai", adminUser, 0)

	rekap, err := f.svc.GetRekap("2026-09")
	require.NoError(t, err)
	require.Empty(t, rekap.Guru)
	require.Len(t, rekap.TanpaGuru, 1)

	_, err = f.svc.GetRekap("2026-9")
	require.ErrorIs(t, err, ErrUjrohBulanTidakValid)
	require.ErrorIs(t, f.svc.Kunci("2026-11", f.guruUtamaUser), ErrUjrohBulanMendatang)

	bulan, err := f.svc.NormalizeBulan("")
	require.NoError(t, err)
	require.Equal(t, "2026-10", bulan)
}
