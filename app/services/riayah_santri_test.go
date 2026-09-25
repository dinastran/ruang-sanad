package services

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type riayahTestFixture struct {
	db        *sql.DB
	service   *RiayahSantriService
	guruA     int64
	guruAUser int64
	guruB     int64
	guruBUser int64
	kelasA    int64
	kelasB    int64
	today     time.Time
	pertemuan map[int64]int64
}

func setupRiayahSantri(t *testing.T) *riayahTestFixture {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))

	f := &riayahTestFixture{db: db, service: NewRiayahSantriService(queries.NewQuerier(db)), pertemuan: map[int64]int64{}}
	f.today = time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	f.guruAUser = insertTestUser(t, db, "a@example.com", "Guru A")
	f.guruBUser = insertTestUser(t, db, "b@example.com", "Guru B")
	f.guruA = insertTestGuru(t, db, "Guru A", f.guruAUser)
	f.guruB = insertTestGuru(t, db, "Guru B", f.guruBUser)
	f.kelasA = f.insertKelas(t, "RA", "Kelas A", f.guruA)
	f.kelasB = f.insertKelas(t, "RB", "Kelas B", f.guruB)
	return f
}

func (f *riayahTestFixture) insertKelas(t *testing.T, kunci, nama string, guruID int64) int64 {
	t.Helper()
	_, err := f.db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, guru_id, kapasitas, jumlah_santri) VALUES (?, '2026', 'Reguler', 'L', 'Dasar', '1x/pekan', 'Senin', 1, ?, ?, 20, 0)`, kunci, nama, guruID)
	require.NoError(t, err)
	id, err := lastID(f.db)
	require.NoError(t, err)
	return id
}

func (f *riayahTestFixture) insertSantri(t *testing.T, nama string, kelasID int64) int64 {
	t.Helper()
	// Mulai belajar 5 hari lalu agar penanda "belum disapa" tidak ikut muncul.
	mulai := f.today.AddDate(0, 0, -5).Format("2006-01-02")
	_, err := f.db.Exec(`INSERT INTO santri (nama, kelas_id, status, mulai_belajar) VALUES (?, ?, 'aktif', ?)`, nama, kelasID, mulai)
	require.NoError(t, err)
	id, err := lastID(f.db)
	require.NoError(t, err)
	return id
}

// absen mencatat satu pertemuan selesai `hariLalu` hari sebelum today.
func (f *riayahTestFixture) absen(t *testing.T, kelasID, santriID int64, hariLalu int, status, batas string) {
	t.Helper()
	f.pertemuan[kelasID]++
	tanggal := f.today.AddDate(0, 0, -hariLalu).Format("2006-01-02")
	_, err := f.db.Exec(`INSERT INTO pertemuan (kelas_id, pertemuan_ke, pertemuan_level_ke, tanggal, status) VALUES (?, ?, ?, ?, 'selesai')`, kelasID, f.pertemuan[kelasID], f.pertemuan[kelasID], tanggal)
	require.NoError(t, err)
	pid, err := lastID(f.db)
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO absensi (pertemuan_id, santri_id, status, batas_materi) VALUES (?, ?, ?, ?)`, pid, santriID, status, batas)
	require.NoError(t, err)
}

func findItem(items []models.RiayahSantriItem, id int64) *models.RiayahSantriItem {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

func kodePenanda(item *models.RiayahSantriItem) []string {
	out := []string{}
	for _, p := range item.Penanda {
		out = append(out, p.Kode)
	}
	return out
}

func TestRiayahPenandaKehadiranDanProgres(t *testing.T) {
	f := setupRiayahSantri(t)
	alpa := f.insertSantri(t, "Alpa Berturut", f.kelasA)
	rendah := f.insertSantri(t, "Hadir Rendah", f.kelasA)
	sedikit := f.insertSantri(t, "Data Sedikit", f.kelasA)
	macet := f.insertSantri(t, "Progres Macet", f.kelasA)
	lancar := f.insertSantri(t, "Lancar", f.kelasA)

	// Alpa 2x terakhir berturut-turut.
	f.absen(t, f.kelasA, alpa, 20, "hadir", "")
	f.absen(t, f.kelasA, alpa, 10, "alpa", "")
	f.absen(t, f.kelasA, alpa, 3, "alpa", "")

	// 1 dari 4 hadir dalam 30 hari (25%), tanpa alpa berturut.
	f.absen(t, f.kelasA, rendah, 25, "izin", "")
	f.absen(t, f.kelasA, rendah, 18, "sakit", "")
	f.absen(t, f.kelasA, rendah, 11, "alpa", "")
	f.absen(t, f.kelasA, rendah, 4, "hadir", "")

	// Hanya 2 pertemuan: belum cukup data untuk persen kehadiran.
	f.absen(t, f.kelasA, sedikit, 10, "izin", "")
	f.absen(t, f.kelasA, sedikit, 3, "hadir", "")

	// Batas materi sama di 4 pertemuan hadir terakhir; alpa di sela tidak dihitung.
	f.absen(t, f.kelasA, macet, 40, "hadir", "Al-Baqarah 1")
	f.absen(t, f.kelasA, macet, 28, "hadir", "Al-Baqarah 5")
	f.absen(t, f.kelasA, macet, 21, "hadir", "al-baqarah  5")
	f.absen(t, f.kelasA, macet, 14, "telat", "Al-Baqarah 5")
	f.absen(t, f.kelasA, macet, 10, "alpa", "")
	f.absen(t, f.kelasA, macet, 7, "hadir", "Al-Baqarah 5")

	f.absen(t, f.kelasA, lancar, 21, "hadir", "Juz 1")
	f.absen(t, f.kelasA, lancar, 14, "hadir", "Juz 1")
	f.absen(t, f.kelasA, lancar, 7, "hadir", "Juz 1")
	f.absen(t, f.kelasA, lancar, 1, "hadir", "Juz 2")

	items, ringkasan, err := f.service.ListPerhatian(models.RiayahViewer{GuruID: &f.guruA}, f.today)
	require.NoError(t, err)
	require.Len(t, items, 5)

	require.Equal(t, []string{models.PenandaKehadiran}, kodePenanda(findItem(items, alpa)))
	require.Contains(t, findItem(items, alpa).Penanda[0].Alasan, "Alpa 2")
	require.Equal(t, []string{models.PenandaKehadiran}, kodePenanda(findItem(items, rendah)))
	require.Contains(t, findItem(items, rendah).Penanda[0].Alasan, "Hadir 1 dari 4")
	require.Empty(t, findItem(items, sedikit).Penanda)
	require.Equal(t, []string{models.PenandaProgres}, kodePenanda(findItem(items, macet)))
	require.Empty(t, findItem(items, lancar).Penanda)
	require.Equal(t, "Juz 2", findItem(items, lancar).BatasMateriTerakhir)

	// Merah di atas kuning, kuning di atas tanpa penanda.
	require.Equal(t, models.LevelMerah, items[0].Penanda[0].Level)
	require.Equal(t, models.LevelMerah, items[1].Penanda[0].Level)
	require.Equal(t, macet, items[2].ID)

	require.Equal(t, 5, ringkasan.TotalSantri)
	require.Equal(t, 3, ringkasan.PerluPerhatian)
	require.Equal(t, 2, ringkasan.Kehadiran)
	require.Equal(t, 1, ringkasan.Progres)
}

func TestRiayahCakupanGuruDanAkses(t *testing.T) {
	f := setupRiayahSantri(t)
	santriA := f.insertSantri(t, "Santri A", f.kelasA)
	santriB := f.insertSantri(t, "Santri B", f.kelasB)

	guruA := models.RiayahViewer{UserID: f.guruAUser, GuruID: &f.guruA, CanWrite: true}
	items, _, err := f.service.ListPerhatian(guruA, f.today)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, santriA, items[0].ID)

	_, err = f.service.GetProfil(guruA, santriB, f.today)
	require.ErrorIs(t, err, ErrRiayahAksesDitolak)
	require.ErrorIs(t, f.service.TambahCatatan(guruA, santriB, "tidak boleh"), ErrRiayahAksesDitolak)

	semua, _, err := f.service.ListPerhatian(models.RiayahViewer{}, f.today)
	require.NoError(t, err)
	require.Len(t, semua, 2)

	koordinator := models.RiayahViewer{UserID: 99}
	_, err = f.service.GetProfil(koordinator, santriB, f.today)
	require.NoError(t, err)
	require.ErrorIs(t, f.service.TambahCatatan(koordinator, santriB, "baca saja"), ErrRiayahAksesDitolak)
}

func TestRiayahProfilTimelineDanHakHapusCatatan(t *testing.T) {
	f := setupRiayahSantri(t)
	santri := f.insertSantri(t, "Santri Profil", f.kelasA)
	f.absen(t, f.kelasA, santri, 14, "hadir", "Iqra 2 hal 10")
	f.absen(t, f.kelasA, santri, 7, "sakit", "")

	guruA := models.RiayahViewer{UserID: f.guruAUser, GuruID: &f.guruA, CanWrite: true}
	require.NoError(t, f.service.TambahCatatan(guruA, santri, "Semangat, tapi sering kelelahan"))
	require.Error(t, f.service.TambahCatatan(guruA, santri, "   "))

	// Catatan dari guru lain (mis. guru sebelumnya) tidak boleh dihapus guru A.
	_, err := f.db.Exec(`INSERT INTO catatan_riayah (guru_id, author_user_id, target_type, target_id, catatan) VALUES (?, ?, 'santri', ?, 'Catatan guru lama')`, f.guruB, f.guruBUser, santri)
	require.NoError(t, err)
	catatanLain, err := lastID(f.db)
	require.NoError(t, err)

	profil, err := f.service.GetProfil(guruA, santri, f.today)
	require.NoError(t, err)
	require.Equal(t, int64(2), profil.Rekap.Total)
	require.Equal(t, int64(1), profil.Rekap.Hadir)
	require.Equal(t, int64(1), profil.Rekap.Sakit)
	require.Equal(t, "Iqra 2 hal 10", profil.Santri.BatasMateriTerakhir)
	require.Equal(t, "Guru A", profil.Santri.GuruNama)
	require.Len(t, profil.Timeline, 4)
	for i := 1; i < len(profil.Timeline); i++ {
		require.GreaterOrEqual(t, profil.Timeline[i-1].Tanggal, profil.Timeline[i].Tanggal)
	}
	for _, item := range profil.Timeline {
		if item.Jenis == "catatan" {
			require.Equal(t, item.ID != catatanLain, item.BisaHapus, item.Isi)
		}
	}

	require.ErrorIs(t, f.service.HapusCatatan(guruA, santri, catatanLain), ErrRiayahAksesDitolak)
	require.NoError(t, f.service.HapusCatatan(models.RiayahViewer{UserID: 1, CanWrite: true}, santri, catatanLain))
}

func TestRiayahPenandaBelumDisapaDanCatatKontak(t *testing.T) {
	f := setupRiayahSantri(t)
	baru := f.insertSantri(t, "Santri Baru", f.kelasA)
	lama := f.insertSantri(t, "Santri Lama", f.kelasA)
	tanpaTanggal := f.insertSantri(t, "Tanpa Tanggal", f.kelasA)
	_, err := f.db.Exec(`UPDATE santri SET mulai_belajar = ? WHERE id = ?`, f.today.AddDate(0, 0, -60).Format("2006-01-02"), lama)
	require.NoError(t, err)
	_, err = f.db.Exec(`UPDATE santri SET mulai_belajar = NULL WHERE id = ?`, tanpaTanggal)
	require.NoError(t, err)

	guruA := models.RiayahViewer{UserID: f.guruAUser, GuruID: &f.guruA, CanWrite: true}
	items, ringkasan, err := f.service.ListPerhatian(guruA, f.today)
	require.NoError(t, err)
	require.Empty(t, findItem(items, baru).Penanda)
	require.Equal(t, []string{models.PenandaKontak}, kodePenanda(findItem(items, lama)))
	require.Contains(t, findItem(items, lama).Penanda[0].Alasan, "sejak mulai belajar (60 hari)")
	require.Equal(t, "Belum pernah disapa", findItem(items, tanpaTanggal).Penanda[0].Alasan)
	require.Equal(t, 2, ringkasan.Kontak)

	// Kontak 20 hari lalu masih ditandai; kontak 3 hari lalu menghapus penanda.
	require.NoError(t, f.service.CatatKontak(guruA, lama, models.RiayahKontakInput{Media: "wa", Tanggal: f.today.AddDate(0, 0, -20).Format("2006-01-02")}, f.today))
	items, _, err = f.service.ListPerhatian(guruA, f.today)
	require.NoError(t, err)
	require.Contains(t, findItem(items, lama).Penanda[0].Alasan, "Terakhir disapa 20 hari lalu")

	require.NoError(t, f.service.CatatKontak(guruA, lama, models.RiayahKontakInput{Media: "telepon", Catatan: "Kabar baik"}, f.today))
	items, _, err = f.service.ListPerhatian(guruA, f.today)
	require.NoError(t, err)
	require.Empty(t, findItem(items, lama).Penanda)
	require.Equal(t, f.today.Format("2006-01-02"), findItem(items, lama).KontakTerakhir)

	// Validasi input.
	require.Error(t, f.service.CatatKontak(guruA, lama, models.RiayahKontakInput{Media: "surat"}, f.today))
	require.Error(t, f.service.CatatKontak(guruA, lama, models.RiayahKontakInput{Media: "wa", Tanggal: f.today.AddDate(0, 0, 2).Format("2006-01-02")}, f.today))
	require.Error(t, f.service.CatatKontak(guruA, lama, models.RiayahKontakInput{Media: "wa", Jenis: "rapor", Periode: "2026-09"}, f.today), "rapor tanpa pesan pribadi")
	require.ErrorIs(t, f.service.CatatKontak(models.RiayahViewer{UserID: 99}, lama, models.RiayahKontakInput{Media: "wa"}, f.today), ErrRiayahAksesDitolak)

	// Kontak muncul di timeline dan hanya penulisnya (guru) yang bisa menghapus.
	profil, err := f.service.GetProfil(guruA, lama, f.today)
	require.NoError(t, err)
	var kontakID int64
	jumlahKontak := 0
	for _, item := range profil.Timeline {
		if item.Jenis == "kontak" {
			jumlahKontak++
			require.True(t, item.BisaHapus)
			kontakID = item.ID
		}
	}
	require.Equal(t, 2, jumlahKontak)
	guruB := models.RiayahViewer{UserID: f.guruBUser, GuruID: &f.guruB, CanWrite: true}
	require.ErrorIs(t, f.service.HapusKontak(guruB, lama, kontakID), ErrRiayahAksesDitolak)
	require.NoError(t, f.service.HapusKontak(guruA, lama, kontakID))
}

func TestRiayahRaporBulananDanPelacakanPengiriman(t *testing.T) {
	f := setupRiayahSantri(t)
	santri := f.insertSantri(t, "Santri Rapor", f.kelasA)
	lain := f.insertSantri(t, "Santri Lain", f.kelasA)
	// today = 2026-09-25, jadi bulan lalu = 2026-08.
	f.absen(t, f.kelasA, santri, 57, "hadir", "Juz 1") // 2026-07-30, di luar periode
	f.absen(t, f.kelasA, santri, 52, "hadir", "Juz 2") // 2026-08-04
	f.absen(t, f.kelasA, santri, 45, "izin", "")       // 2026-08-11
	f.absen(t, f.kelasA, santri, 38, "hadir", "Juz 3") // 2026-08-18
	f.absen(t, f.kelasA, santri, 25, "hadir", "Juz 4") // 2026-08-31
	f.absen(t, f.kelasA, santri, 24, "alpa", "")       // 2026-09-01, di luar periode

	guruA := models.RiayahViewer{UserID: f.guruAUser, GuruID: &f.guruA, CanWrite: true}
	rapor, err := f.service.GetRapor(guruA, santri, "", f.today)
	require.NoError(t, err)
	require.Equal(t, "2026-08", rapor.Periode)
	require.Equal(t, int64(4), rapor.Rekap.Total)
	require.Equal(t, int64(3), rapor.Rekap.Hadir)
	require.Equal(t, int64(1), rapor.Rekap.Izin)
	require.Equal(t, "Juz 2", rapor.BatasAwal)
	require.Equal(t, "Juz 4", rapor.BatasAkhir)
	require.Len(t, rapor.Pertemuan, 4)
	require.Empty(t, rapor.TerkirimPada)
	require.Equal(t, []string{"2026-09", "2026-08", "2026-07", "2026-06", "2026-05", "2026-04"}, rapor.PeriodeOpsi)

	_, err = f.service.GetRapor(guruA, santri, "2026-10", f.today)
	require.Error(t, err)
	_, err = f.service.GetRapor(guruA, santri, "Agustus", f.today)
	require.Error(t, err)

	rapor08 := models.RiayahKontakInput{Media: "wa", Jenis: "rapor", Periode: "2026-08"}
	rapor08.Catatan = "Mantap"
	require.Error(t, f.service.CatatKontak(guruA, santri, rapor08, f.today), "pesan pribadi terlalu pendek")
	rapor08.Catatan = "Masya Allah, bacaan ananda makin tartil bulan ini."
	require.NoError(t, f.service.CatatKontak(guruA, santri, rapor08, f.today))
	require.Error(t, f.service.CatatKontak(guruA, santri, models.RiayahKontakInput{Media: "wa", Jenis: "rapor", Periode: "2026-10", Catatan: rapor08.Catatan}, f.today))

	rapor, err = f.service.GetRapor(guruA, santri, "2026-08", f.today)
	require.NoError(t, err)
	require.Equal(t, []string{"2026-09-25"}, rapor.TerkirimPada)

	items, ringkasan, err := f.service.ListPerhatian(guruA, f.today)
	require.NoError(t, err)
	require.Equal(t, "2026-08", ringkasan.RaporPeriode)
	require.Equal(t, 1, ringkasan.RaporTerkirim)
	require.True(t, findItem(items, santri).RaporTerkirim)
	require.False(t, findItem(items, lain).RaporTerkirim)
	// Mengirim rapor juga dihitung sebagai menyapa santri.
	require.Equal(t, "2026-09-25", findItem(items, santri).KontakTerakhir)

	// Koordinator boleh melihat rapor tetapi tidak mencatat pengiriman.
	_, err = f.service.GetRapor(models.RiayahViewer{UserID: 99}, santri, "2026-08", f.today)
	require.NoError(t, err)
	_, err = f.service.GetRapor(models.RiayahViewer{UserID: f.guruBUser, GuruID: &f.guruB, CanWrite: true}, santri, "2026-08", f.today)
	require.ErrorIs(t, err, ErrRiayahAksesDitolak)
}

func TestRiayahTemplateWA(t *testing.T) {
	f := setupRiayahSantri(t)
	_, err := f.db.Exec(`INSERT INTO wa_template (nama, target_type, body, is_aktif) VALUES
		('Sapaan Koordinator', 'santri', 'Halo {nama}', 1),
		('Nonaktif', 'santri', 'x', 0),
		('Untuk Guru', 'guru', 'y', 1)`)
	require.NoError(t, err)

	templates := f.service.ListTemplateWA()
	require.Equal(t, "Sapaan Koordinator", templates[0].Nama)
	require.Len(t, templates, 1+len(riayahTemplateBawaan))
	for _, tpl := range templates {
		require.NotEqual(t, "Nonaktif", tpl.Nama)
		require.NotEqual(t, "Untuk Guru", tpl.Nama)
	}
}

func TestRiayahPerbaikanReview(t *testing.T) {
	f := setupRiayahSantri(t)
	guruA := models.RiayahViewer{UserID: f.guruAUser, GuruID: &f.guruA, CanWrite: true}

	// Hadir tanpa batas materi memutus deret "progres macet".
	putus := f.insertSantri(t, "Deret Putus", f.kelasA)
	f.absen(t, f.kelasA, putus, 28, "hadir", "B")
	f.absen(t, f.kelasA, putus, 21, "hadir", "B")
	f.absen(t, f.kelasA, putus, 14, "hadir", "B")
	f.absen(t, f.kelasA, putus, 7, "hadir", "")
	f.absen(t, f.kelasA, putus, 1, "hadir", "B")

	// Santri aktif tanpa kelas: terlihat oleh admin, tidak oleh guru.
	tanpaKelas, err := f.db.Exec(`INSERT INTO santri (nama, status) VALUES ('Tanpa Kelas', 'aktif')`)
	require.NoError(t, err)
	tanpaKelasID, err := tanpaKelas.LastInsertId()
	require.NoError(t, err)

	items, _, err := f.service.ListPerhatian(guruA, f.today)
	require.NoError(t, err)
	require.Empty(t, findItem(items, putus).Penanda)
	require.Nil(t, findItem(items, tanpaKelasID))

	admin := models.RiayahViewer{UserID: 1, CanWrite: true}
	semua, _, err := f.service.ListPerhatian(admin, f.today)
	require.NoError(t, err)
	require.NotNil(t, findItem(semua, tanpaKelasID))
	// Profil santri tanpa kelas tetap menampilkan penandanya.
	profil, err := f.service.GetProfil(admin, tanpaKelasID, f.today)
	require.NoError(t, err)
	require.Equal(t, findItem(semua, tanpaKelasID).Penanda, profil.Santri.Penanda)

	// Rapor tidak tercatat ganda di hari yang sama.
	rapor := models.RiayahKontakInput{Media: "wa", Jenis: "rapor", Periode: "2026-08", Catatan: "Bacaan ananda makin lancar, terus semangat."}
	require.NoError(t, f.service.CatatKontak(guruA, putus, rapor, f.today))
	require.Error(t, f.service.CatatKontak(guruA, putus, rapor, f.today))
}

func TestRiayahHariDihitungPerKalenderWIB(t *testing.T) {
	// 03:00 WIB tanggal 25, kontak terakhir tanggal 10 = 15 hari kalender.
	today := time.Date(2026, 9, 25, 3, 0, 0, 0, ZonaWaktuRiayah)
	p := hitungPenandaKontak("2026-09-10", "", today)
	require.NotNil(t, p)
	require.Contains(t, p.Alasan, "15 hari")
	require.Nil(t, hitungPenandaKontak("2026-09-11", "", today))
}
