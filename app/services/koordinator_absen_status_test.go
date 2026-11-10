package services

import (
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

// Koordinator often fills attendance but forgets to switch the session from
// Dijadwalkan to Terlaksana. Saving attendance now does that automatically,
// while Libur/Batal and future sessions are refused without saving anything.
func TestSimpanAbsenOtomatisMenandaiTerlaksana(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	svc := NewKoordinatorService(f.querier)
	svc.today = func() time.Time { return time.Date(2026, 11, 10, 9, 0, 0, 0, ZonaWaktuRiayah) }
	var guruLainID int64
	require.NoError(t, f.db.QueryRow(`SELECT id FROM guru WHERE nama = 'Guru Lain'`).Scan(&guruLainID))
	absen := absenTerisi(map[int64]string{f.guruUtamaID: "hadir", f.guruBadalID: "alfa", guruLainID: "izin"})

	statusPembinaan := func(id int64) string {
		t.Helper()
		var s string
		require.NoError(t, f.db.QueryRow(`SELECT status FROM pembinaan WHERE id = ?`, id).Scan(&s))
		return s
	}
	jumlahAbsen := func(tabel, kolom string, id int64) int {
		t.Helper()
		var n int
		require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM `+tabel+` WHERE `+kolom+` = ?`, id).Scan(&n))
		return n
	}

	// Dijadwalkan (hari ini): tersimpan dan otomatis terlaksana.
	id, err := svc.CreatePembinaan(models.PembinaanRequest{Tanggal: "2026-11-10", Topik: "Tahsin", Status: "dijadwalkan"})
	require.NoError(t, err)
	berubah, err := svc.SavePembinaanAbsen(id, absen)
	require.NoError(t, err)
	require.True(t, berubah)
	require.Equal(t, "terlaksana", statusPembinaan(id))
	require.Equal(t, 3, jumlahAbsen("pembinaan_absen", "pembinaan_id", id))
	laporan, err := svc.GetLaporanPembinaanAbsensi(id)
	require.NoError(t, err)
	require.Equal(t, 1, laporan.Hadir)

	// Menyimpan ulang sesi yang sudah terlaksana: tidak ada perubahan status.
	berubah, err = svc.SavePembinaanAbsen(id, absen)
	require.NoError(t, err)
	require.False(t, berubah)

	// Hadir di sesi itu kini dihitung untuk kinerja guru.
	hadir, err := f.querier.CountPembinaanHadirByGuru(t.Context(), queries.CountPembinaanHadirByGuruParams{GuruID: f.guruUtamaID, Tanggal: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), Tanggal_2: time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	require.EqualValues(t, 1, hadir)

	// Libur dan tanggal yang belum tiba ditolak; tidak ada data tersimpan.
	libur, err := svc.CreatePembinaan(models.PembinaanRequest{Tanggal: "2026-11-03", Status: "libur"})
	require.NoError(t, err)
	_, err = svc.SavePembinaanAbsen(libur, absen)
	require.ErrorContains(t, err, "Libur")
	require.Equal(t, 0, jumlahAbsen("pembinaan_absen", "pembinaan_id", libur))

	depan, err := svc.CreatePembinaan(models.PembinaanRequest{Tanggal: "2026-11-17", Status: "dijadwalkan"})
	require.NoError(t, err)
	_, err = svc.SavePembinaanAbsen(depan, absen)
	require.ErrorContains(t, err, "2026-11-17")
	require.Equal(t, "dijadwalkan", statusPembinaan(depan))
	require.Equal(t, 0, jumlahAbsen("pembinaan_absen", "pembinaan_id", depan))

	// Absensi tidak lengkap ditolak dan status tidak berubah.
	kemarin, err := svc.CreatePembinaan(models.PembinaanRequest{Tanggal: "2026-11-09", Status: "dijadwalkan"})
	require.NoError(t, err)
	_, err = svc.SavePembinaanAbsen(kemarin, absen[:2])
	require.Error(t, err)
	require.Equal(t, "dijadwalkan", statusPembinaan(kemarin))

	// Rapat: perilaku sama, Batal ditolak.
	rapat, err := svc.CreateRapat(models.RapatRequest{Tanggal: "2026-11-08", Judul: "Rapat bulanan", Status: "dijadwalkan"})
	require.NoError(t, err)
	berubah, err = svc.SaveRapatAbsen(rapat, absen)
	require.NoError(t, err)
	require.True(t, berubah)
	r, err := svc.GetRapat(rapat)
	require.NoError(t, err)
	require.Equal(t, "terlaksana", r.Status)

	batal, err := svc.CreateRapat(models.RapatRequest{Tanggal: "2026-11-01", Status: "batal"})
	require.NoError(t, err)
	_, err = svc.SaveRapatAbsen(batal, absen)
	require.ErrorContains(t, err, "Batal")
	require.Equal(t, 0, jumlahAbsen("rapat_absen", "rapat_id", batal))

	// Riwayat absensi hanya memuat sesi terlaksana: bila sesi dikembalikan
	// ke dijadwalkan, absensinya tidak lagi dihitung.
	riwayat, err := svc.ListRiwayatAbsensiGuru()
	require.NoError(t, err)
	n := len(riwayat)
	require.NoError(t, svc.UpdatePembinaan(id, models.PembinaanRequest{Tanggal: "2026-11-10", Topik: "Tahsin", Status: "dijadwalkan"}))
	riwayat, err = svc.ListRiwayatAbsensiGuru()
	require.NoError(t, err)
	require.Equal(t, n-3, len(riwayat))
}
