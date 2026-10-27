package services

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestHitungAlfaBeruntun(t *testing.T) {
	cases := []struct {
		name   string
		status []string
		want   int
	}{
		{"kosong", nil, 0},
		{"tiga alfa", []string{"alfa", "alfa", "alfa", "hadir"}, 3},
		{"terputus izin", []string{"alfa", "izin", "alfa", "alfa"}, 1},
		{"terputus sakit", []string{"alfa", "alfa", "sakit", "alfa"}, 2},
		{"terbaru hadir", []string{"hadir", "alfa", "alfa", "alfa"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, hitungAlfaBeruntun(tc.status))
		})
	}
	require.Equal(t, "merah", levelAlfaBeruntun(3))
	require.Equal(t, "merah", levelAlfaBeruntun(5))
	require.Equal(t, "oranye", levelAlfaBeruntun(2))
	require.Equal(t, "", levelAlfaBeruntun(1))
}

func absenTerisi(guru map[int64]string) []models.AbsenInput {
	out := make([]models.AbsenInput, 0, len(guru))
	for id, status := range guru {
		in := models.AbsenInput{GuruID: id, Kehadiran: status}
		switch status {
		case models.KehadiranHadir:
			in.JamMasuk = "19:30"
		case models.KehadiranIzin, models.KehadiranSakit:
			in.Alasan = "keperluan keluarga"
		}
		out = append(out, in)
	}
	return out
}

func TestAnalisisAlfaPembinaanMenandaiAlfaBerturutTurut(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	svc := NewKoordinatorService(f.querier)
	var guruLainID int64
	require.NoError(t, f.db.QueryRow(`SELECT id FROM guru WHERE nama = 'Guru Lain'`).Scan(&guruLainID))

	buatSesi := func(tanggal, status string) int64 {
		t.Helper()
		id, err := svc.CreatePembinaan(models.PembinaanRequest{Tanggal: tanggal, Topik: "Sesi " + tanggal, Status: status})
		require.NoError(t, err)
		return id
	}

	p1 := buatSesi("2026-09-01", "terlaksana")
	require.NoError(t, svc.SavePembinaanAbsen(p1, absenTerisi(map[int64]string{f.guruUtamaID: "alfa", f.guruBadalID: "hadir", guruLainID: "hadir"})))
	p2 := buatSesi("2026-09-08", "terlaksana")
	require.NoError(t, svc.SavePembinaanAbsen(p2, absenTerisi(map[int64]string{f.guruUtamaID: "alfa", f.guruBadalID: "izin", guruLainID: "alfa"})))
	buatSesi("2026-09-15", "libur")
	p4 := buatSesi("2026-09-22", "terlaksana")
	require.NoError(t, svc.SavePembinaanAbsen(p4, absenTerisi(map[int64]string{f.guruUtamaID: "alfa", f.guruBadalID: "alfa", guruLainID: "alfa"})))
	// Sesi terlaksana yang absensinya belum diisi tidak memutus hitungan.
	buatSesi("2026-09-29", "terlaksana")
	// Guru baru tanpa catatan absen tidak boleh langsung ditandai.
	baruID := insertTestGuru(t, f.db, "Guru Baru", insertTestUser(t, f.db, "baru@example.com", "Guru Baru"))

	a, err := svc.AnalisisAlfaPembinaan()
	require.NoError(t, err)

	require.Equal(t, 3, a.AmbangAlfa)
	require.Len(t, a.Sesi, 4)
	require.Equal(t, "2026-09-29", a.Sesi[0].Tanggal)
	require.Equal(t, 1, a.Merah)
	require.Equal(t, 1, a.Waspada)

	byID := map[int64]models.AnalisisKehadiranGuru{}
	for _, g := range a.Guru {
		byID[g.GuruID] = g
	}
	require.Equal(t, f.guruUtamaID, a.Guru[0].GuruID, "guru dengan alfa terbanyak di urutan teratas")

	utama := byID[f.guruUtamaID]
	require.Equal(t, 3, utama.AlfaBeruntun)
	require.Equal(t, "merah", utama.Level)
	require.Equal(t, []string{"", "alfa", "alfa", "alfa"}, utama.Riwayat)
	require.Equal(t, "", utama.TerakhirHadir)

	lain := byID[guruLainID]
	require.Equal(t, 2, lain.AlfaBeruntun)
	require.Equal(t, "oranye", lain.Level)
	require.Equal(t, "2026-09-01", lain.TerakhirHadir)
	require.Equal(t, 1, lain.HadirTerakhir)
	require.Equal(t, 3, lain.DicatatTerakhir)

	badal := byID[f.guruBadalID]
	require.Equal(t, 1, badal.AlfaBeruntun, "izin memutus hitungan alfa")
	require.Equal(t, "", badal.Level)

	baru := byID[baruID]
	require.Equal(t, 0, baru.AlfaBeruntun)
	require.Equal(t, []string{"", "", "", ""}, baru.Riwayat)

	require.NotNil(t, a.PersenHadir)
	require.InDelta(t, 200.0/9.0, *a.PersenHadir, 0.01)

	ringkasan, err := svc.RingkasanAlfaBeruntun()
	require.NoError(t, err)
	require.Equal(t, 1, ringkasan.PembinaanMerah)
	require.Equal(t, 1, ringkasan.PembinaanWaspada)
	require.Equal(t, 0, ringkasan.RapatMerah)
}

func TestAnalisisAlfaRapatTerpisahDariPembinaan(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	svc := NewKoordinatorService(f.querier)
	var guruLainID int64
	require.NoError(t, f.db.QueryRow(`SELECT id FROM guru WHERE nama = 'Guru Lain'`).Scan(&guruLainID))

	for _, tanggal := range []string{"2026-07-01", "2026-08-01", "2026-09-01"} {
		id, err := svc.CreateRapat(models.RapatRequest{Tanggal: tanggal, Judul: "Rapat " + tanggal, Status: "terlaksana"})
		require.NoError(t, err)
		require.NoError(t, svc.SaveRapatAbsen(id, absenTerisi(map[int64]string{f.guruUtamaID: "hadir", f.guruBadalID: "alfa", guruLainID: "sakit"})))
	}
	_, err := svc.CreateRapat(models.RapatRequest{Tanggal: "2026-10-01", Judul: "Batal", Status: "batal"})
	require.NoError(t, err)

	a, err := svc.AnalisisAlfaRapat()
	require.NoError(t, err)
	require.Len(t, a.Sesi, 3)
	require.Equal(t, 1, a.Merah)
	require.Equal(t, f.guruBadalID, a.Guru[0].GuruID)
	require.Equal(t, 3, a.Guru[0].AlfaBeruntun)

	p, err := svc.AnalisisAlfaPembinaan()
	require.NoError(t, err)
	require.Equal(t, 0, p.Merah)
	require.Empty(t, p.Sesi)
}

func TestSimpanAbsenMemvalidasiStatusKehadiran(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	svc := NewKoordinatorService(f.querier)
	var guruLainID int64
	require.NoError(t, f.db.QueryRow(`SELECT id FROM guru WHERE nama = 'Guru Lain'`).Scan(&guruLainID))
	id, err := svc.CreatePembinaan(models.PembinaanRequest{Tanggal: "2026-09-01", Status: "terlaksana"})
	require.NoError(t, err)

	// Izin wajib beralasan.
	err = svc.SavePembinaanAbsen(id, []models.AbsenInput{
		{GuruID: f.guruUtamaID, Kehadiran: "izin"},
		{GuruID: f.guruBadalID, Kehadiran: "hadir", JamMasuk: "19:30"},
		{GuruID: guruLainID, Kehadiran: "alfa"},
	})
	require.ErrorContains(t, err, "alasan izin/sakit wajib diisi")

	// Status tidak dikenal ditolak.
	err = svc.SavePembinaanAbsen(id, []models.AbsenInput{
		{GuruID: f.guruUtamaID, Kehadiran: "bolos"},
		{GuruID: f.guruBadalID, Kehadiran: "hadir", JamMasuk: "19:30"},
		{GuruID: guruLainID, Kehadiran: "alfa"},
	})
	require.ErrorContains(t, err, "status kehadiran tidak valid")

	// Alfa boleh tanpa alasan; klien lama (hanya Hadir) tetap didukung.
	require.NoError(t, svc.SavePembinaanAbsen(id, []models.AbsenInput{
		{GuruID: f.guruUtamaID, Kehadiran: "sakit", Alasan: "demam"},
		{GuruID: f.guruBadalID, Hadir: true, JamMasuk: "19:30"},
		{GuruID: guruLainID, Kehadiran: "alfa"},
	}))

	laporan, err := svc.GetLaporanPembinaanAbsensi(id)
	require.NoError(t, err)
	require.Equal(t, 1, laporan.Hadir)
	require.Equal(t, 2, laporan.TidakHadir)
	require.Equal(t, 1, laporan.Sakit)
	require.Equal(t, 1, laporan.Alfa)
	require.Equal(t, 0, laporan.Izin)

	var hadir int64
	var kehadiran string
	require.NoError(t, f.db.QueryRow(`SELECT hadir, kehadiran FROM pembinaan_absen WHERE pembinaan_id = ? AND guru_id = ?`, id, f.guruBadalID).Scan(&hadir, &kehadiran))
	require.EqualValues(t, 1, hadir)
	require.Equal(t, "hadir", kehadiran)
	require.NoError(t, f.db.QueryRow(`SELECT hadir, kehadiran FROM pembinaan_absen WHERE pembinaan_id = ? AND guru_id = ?`, id, f.guruUtamaID).Scan(&hadir, &kehadiran))
	require.EqualValues(t, 0, hadir)
	require.Equal(t, "sakit", kehadiran)
}

func TestMigrasiKehadiranMengisiDataLama(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	dir := filepath.Join("..", "..", "migrations")
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.UpTo(db, dir, 84))

	exec := func(query string, args ...any) {
		t.Helper()
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	exec(`INSERT INTO guru (id, nama, is_aktif) VALUES (1, 'A', 1), (2, 'B', 1)`)
	exec(`INSERT INTO pembinaan (id, tanggal, status) VALUES (1, '2026-09-01', 'terlaksana')`)
	exec(`INSERT INTO rapat_guru (id, tanggal, status) VALUES (1, '2026-09-01', 'terlaksana')`)
	exec(`INSERT INTO pembinaan_absen (pembinaan_id, guru_id, hadir, alasan) VALUES (1, 1, 1, ''), (1, 2, 0, 'sakit')`)
	exec(`INSERT INTO rapat_absen (rapat_id, guru_id, hadir, alasan) VALUES (1, 1, 0, 'izin'), (1, 2, 1, '')`)

	require.NoError(t, goose.Up(db, dir))

	read := func(query string) []string {
		t.Helper()
		rows, err := db.Query(query)
		require.NoError(t, err)
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			require.NoError(t, rows.Scan(&s))
			out = append(out, s)
		}
		return out
	}
	require.Equal(t, []string{"hadir", "alfa"}, read(`SELECT kehadiran FROM pembinaan_absen ORDER BY guru_id`))
	require.Equal(t, []string{"alfa", "hadir"}, read(`SELECT kehadiran FROM rapat_absen ORDER BY guru_id`))

	_, err = db.Exec(`UPDATE rapat_absen SET kehadiran = 'bolos' WHERE guru_id = 1`)
	require.Error(t, err, "CHECK constraint menolak status tak dikenal")
}
