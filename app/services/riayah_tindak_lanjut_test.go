package services

import (
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestTindakLanjutAlpaDitanganiAdminKelas(t *testing.T) {
	f := setupRiayahSantri(t)
	adminUser := insertTestUser(t, f.db, "admin@example.com", "Admin Kelas")
	admin := models.RiayahViewer{UserID: adminUser, CanWrite: true}
	guruA := models.RiayahViewer{UserID: f.guruAUser, GuruID: &f.guruA, CanWrite: true}
	tgl := func(hariLalu int) string { return f.today.AddDate(0, 0, -hariLalu).Format("2006-01-02") }

	alpa := f.insertSantri(t, "Alpa Dua", f.kelasA)
	f.absen(t, f.kelasA, alpa, 20, "hadir", "")
	f.absen(t, f.kelasA, alpa, 10, "alpa", "")
	f.absen(t, f.kelasA, alpa, 3, "alpa", "")

	terputus := f.insertSantri(t, "Izin Di Tengah", f.kelasA)
	f.absen(t, f.kelasA, terputus, 10, "alpa", "")
	f.absen(t, f.kelasA, terputus, 6, "izin", "")
	f.absen(t, f.kelasA, terputus, 3, "alpa", "")

	rapor := f.insertSantri(t, "Hanya Dirapor", f.kelasB)
	f.absen(t, f.kelasB, rapor, 10, "alpa", "")
	f.absen(t, f.kelasB, rapor, 3, "alpa", "")
	// Rapor bulanan bukan kontak tindak lanjut.
	_, err := f.db.Exec(`INSERT INTO riayah_kontak (santri_id, guru_id, tanggal, media, jenis, periode) VALUES (?, ?, ?, 'wa', 'rapor', '2026-08')`, rapor, f.guruB, tgl(1))
	require.NoError(t, err)

	perlu, sudah, err := f.service.ListTindakLanjutAlpa(f.today)
	require.NoError(t, err)
	require.Len(t, perlu, 2)
	require.Empty(t, sudah)
	byID := map[int64]*models.RiayahTindakLanjutAlpa{}
	for _, it := range perlu {
		byID[it.ID] = it.TindakLanjutAlpa
	}
	require.NotContains(t, byID, terputus, "izin memutus rangkaian alpa")
	require.Contains(t, byID, rapor, "rapor tidak menutup penanda")
	require.Equal(t, 2, byID[alpa].AlpaBeruntun)
	require.Equal(t, tgl(3), byID[alpa].AlpaTerakhir)
	require.Equal(t, []string{"alpa", "alpa", "hadir"}, byID[alpa].AbsenTerakhir)

	n, err := f.service.JumlahTindakLanjutAlpa(f.today)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	// Guru melihat penanda sebagai info Admin Kelas, bukan tugasnya.
	items, ringkasan, err := f.service.ListPerhatian(guruA, f.today)
	require.NoError(t, err)
	require.False(t, findItem(items, alpa).PerluPerhatianGuru())
	require.Equal(t, 1, ringkasan.AlpaPerluDihubungi)

	// Admin Kelas menghubungi: penanda hilang dan badge langsung berkurang.
	require.NoError(t, f.service.CatatKontak(admin, alpa, models.RiayahKontakInput{Media: "telepon", Tanggal: tgl(1), Catatan: "Sedang sakit"}, f.today))
	n, err = f.service.JumlahTindakLanjutAlpa(f.today)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	perlu, sudah, err = f.service.ListTindakLanjutAlpa(f.today)
	require.NoError(t, err)
	require.Len(t, perlu, 1)
	require.Equal(t, rapor, perlu[0].ID)
	require.Len(t, sudah, 1)
	require.Equal(t, alpa, sudah[0].ID)
	require.Equal(t, models.TindakLanjutSelesai, sudah[0].TindakLanjutAlpa.Status)
	require.Equal(t, "Admin Kelas", sudah[0].TindakLanjutAlpa.KontakOleh)
	require.Equal(t, "telepon", sudah[0].TindakLanjutAlpa.KontakMedia)
	require.Empty(t, sudah[0].Penanda)

	items, _, err = f.service.ListPerhatian(guruA, f.today)
	require.NoError(t, err)
	require.Empty(t, findItem(items, alpa).Penanda)
	require.Equal(t, tgl(1), findItem(items, alpa).TindakLanjutAlpa.KontakTanggal)

	// Kontak oleh guru juga menutup penanda.
	guruB := models.RiayahViewer{UserID: f.guruBUser, GuruID: &f.guruB, CanWrite: true}
	require.NoError(t, f.service.CatatKontak(guruB, rapor, models.RiayahKontakInput{Media: "wa"}, f.today))
	perlu, _, err = f.service.ListTindakLanjutAlpa(f.today)
	require.NoError(t, err)
	require.Empty(t, perlu)

	// Alpa baru sesudah kontak membuka penanda lagi.
	f.absen(t, f.kelasA, alpa, 0, "alpa", "")
	perlu, _, err = f.service.ListTindakLanjutAlpa(f.today)
	require.NoError(t, err)
	require.Len(t, perlu, 1)
	require.Equal(t, alpa, perlu[0].ID)
	require.Equal(t, 3, perlu[0].TindakLanjutAlpa.AlpaBeruntun)
	require.Equal(t, models.PenanggungAdminKelas, perlu[0].Penanda[0].Penanggung)
}

func TestTindakLanjutAlpaKontakDiHariAlpaMenutupPenanda(t *testing.T) {
	f := setupRiayahSantri(t)
	admin := models.RiayahViewer{UserID: 1, CanWrite: true}
	s := f.insertSantri(t, "Santri", f.kelasA)
	f.absen(t, f.kelasA, s, 7, "alpa", "")
	f.absen(t, f.kelasA, s, 0, "alpa", "")
	require.NoError(t, f.service.CatatKontak(admin, s, models.RiayahKontakInput{Media: "wa"}, f.today))

	perlu, sudah, err := f.service.ListTindakLanjutAlpa(f.today)
	require.NoError(t, err)
	require.Empty(t, perlu)
	require.Len(t, sudah, 1)
}

func TestTindakLanjutAlpaBertahanSetelahHadirDanLebihDari12Pertemuan(t *testing.T) {
	f := setupRiayahSantri(t)
	santri := f.insertSantri(t, "Rizki", f.kelasA)
	f.absen(t, f.kelasA, santri, 50, "alpa", "")
	f.absen(t, f.kelasA, santri, 49, "alpa", "")
	for hari := 45; hari >= 32; hari-- { f.absen(t, f.kelasA, santri, hari, "hadir", "") }
	perlu, sudah, err := f.service.ListTindakLanjutAlpa(f.today)
	require.NoError(t, err)
	require.Empty(t, sudah)
	require.Len(t, perlu, 1)
	require.Equal(t, santri, perlu[0].ID)
	require.Equal(t, 2, perlu[0].TindakLanjutAlpa.AlpaBeruntun)
	require.Equal(t, models.PenanggungAdminKelas, perlu[0].Penanda[len(perlu[0].Penanda)-1].Penanggung)
	admin := models.RiayahViewer{UserID: insertTestUser(t, f.db, "alpa-new@example.com", "Admin Kelas"), CanWrite: true}
	require.NoError(t, f.service.CatatKontak(admin, santri, models.RiayahKontakInput{
		Media: "telepon", Jenis: "tindak_lanjut_alpa", Catatan: "Sudah dihubungi dan lanjut belajar",
	}, f.today))
	perlu, sudah, err = f.service.ListTindakLanjutAlpa(f.today)
	require.NoError(t, err)
	require.Empty(t, perlu)
	require.Len(t, sudah, 1)
	require.Equal(t, "tindak_lanjut_alpa", sudah[0].TindakLanjutAlpa.KontakJenis)
}

func TestTindakLanjutAlpaPenutupanBeradabDanBukaUlang(t *testing.T) {
	f := setupRiayahSantri(t)
	santri := f.insertSantri(t, "Santri ditindaklanjuti", f.kelasA)
	f.absen(t, f.kelasA, santri, 14, "alpa", "")
	f.absen(t, f.kelasA, santri, 7, "alpa", "")
	adminID := insertTestUser(t, f.db, "alpa-closure@example.com", "Admin Kelas")
	require.ErrorContains(t, f.service.SelesaikanTindakLanjutAlpa(adminID, santri, "pendek", f.today), "minimal 10")
	require.ErrorContains(t, f.service.CatatKontak(models.RiayahViewer{UserID: adminID, CanWrite: true},
		santri, models.RiayahKontakInput{Media: "wa", Jenis: "tindak_lanjut_alpa"}, f.today), "minimal 10")
	require.NoError(t, f.service.SelesaikanTindakLanjutAlpa(adminID, santri, "Sudah didampingi dan kembali hadir", f.today))
	perlu, sudah, err := f.service.ListTindakLanjutAlpa(f.today)
	require.NoError(t, err)
	require.Empty(t, perlu)
	require.Len(t, sudah, 1)
	require.Equal(t, "tutup_alpa", sudah[0].TindakLanjutAlpa.KontakJenis)
	require.Equal(t, "Sudah didampingi dan kembali hadir", sudah[0].TindakLanjutAlpa.KontakCatatan)
	require.ErrorContains(t, f.service.SelesaikanTindakLanjutAlpa(adminID, santri, "Coba tutup ulang", f.today), "sudah selesai")
	items, _, err := f.service.ListPerhatian(models.RiayahViewer{}, f.today)
	require.NoError(t, err)
	require.Empty(t, findItem(items, santri).KontakTerakhir)
	// Pertemuan berikutnya harus benar-benar terjadi sesudah tanggal penutupan.
	f.absen(t, f.kelasA, santri, -1, "hadir", "")
	f.absen(t, f.kelasA, santri, -2, "alpa", "")
	f.absen(t, f.kelasA, santri, -3, "alpa", "")
	perlu, _, err = f.service.ListTindakLanjutAlpa(f.today.AddDate(0, 0, 3))
	require.NoError(t, err)
	require.Len(t, perlu, 1)
}

func TestTindakLanjutAlpaKontakSebelumAlpaTakMenutupKasus(t *testing.T) {
	f := setupRiayahSantri(t)
	santri := f.insertSantri(t, "Santri", f.kelasA)
	admin := models.RiayahViewer{UserID: insertTestUser(t, f.db, "alpa-before@example.com", "Admin"), CanWrite: true}
	require.NoError(t, f.service.CatatKontak(admin, santri, models.RiayahKontakInput{
		Media: "wa", Tanggal: f.today.AddDate(0, 0, -20).Format("2006-01-02"),
	}, f.today))
	f.absen(t, f.kelasA, santri, 14, "alpa", "")
	f.absen(t, f.kelasA, santri, 7, "alpa", "")
	f.absen(t, f.kelasA, santri, 0, "hadir", "")
	perlu, _, err := f.service.ListTindakLanjutAlpa(f.today)
	require.NoError(t, err)
	require.Len(t, perlu, 1)
}
