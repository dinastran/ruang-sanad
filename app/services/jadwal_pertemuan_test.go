package services

import (
	"context"
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

type jadwalTestFixture struct {
	db            *sql.DB
	querier       *queries.Querier
	service       *JadwalPertemuanService
	pertemuan     *PertemuanService
	kelasID       int64
	guruUtamaID   int64
	guruUtamaUser int64
	guruBadalID   int64
	guruBadalUser int64
	guruLainUser  int64
}

func setupJadwalPertemuanService(t *testing.T) jadwalTestFixture {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))

	guruUtamaUser := insertTestUser(t, db, "utama@example.com", "Guru Utama")
	guruBadalUser := insertTestUser(t, db, "badal@example.com", "Guru Badal")
	guruLainUser := insertTestUser(t, db, "lain@example.com", "Guru Lain")
	guruUtamaID := insertTestGuru(t, db, "Guru Utama", guruUtamaUser)
	guruBadalID := insertTestGuru(t, db, "Guru Badal", guruBadalUser)
	_ = insertTestGuru(t, db, "Guru Lain", guruLainUser)

	_, err = db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, guru_id, kapasitas, jumlah_santri) VALUES ('JADWAL', '2026', 'Reguler', 'L', 'Dasar', '1x/pekan', 'Senin', 1, 'Kelas Jadwal', ?, 20, 0)`, guruUtamaID)
	require.NoError(t, err)
	kelasID, err := lastID(db)
	require.NoError(t, err)

	querier := queries.NewQuerier(db)
	return jadwalTestFixture{
		db:            db,
		querier:       querier,
		service:       NewJadwalPertemuanService(querier),
		pertemuan:     NewPertemuanService(querier, nil),
		kelasID:       kelasID,
		guruUtamaID:   guruUtamaID,
		guruUtamaUser: guruUtamaUser,
		guruBadalID:   guruBadalID,
		guruBadalUser: guruBadalUser,
		guruLainUser:  guruLainUser,
	}
}

func insertTestUser(t *testing.T, db *sql.DB, email, name string) int64 {
	t.Helper()
	_, err := db.Exec(`INSERT INTO users (email, name, role) VALUES (?, ?, 'guru')`, email, name)
	require.NoError(t, err)
	id, err := lastID(db)
	require.NoError(t, err)
	return id
}

func insertTestGuru(t *testing.T, db *sql.DB, name string, userID int64) int64 {
	t.Helper()
	_, err := db.Exec(`INSERT INTO guru (nama, user_id, is_aktif) VALUES (?, ?, 1)`, name, userID)
	require.NoError(t, err)
	id, err := lastID(db)
	require.NoError(t, err)
	return id
}

func TestJadwalTidakMengambilNomorPertemuanSebelumDimulai(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	today := time.Now().Format("2006-01-02")

	jadwalID, err := f.service.Create(f.guruUtamaUser, models.BuatJadwalPertemuanRequest{
		KelasID: f.kelasID, Tanggal: today, JamMulai: "09:00",
	})
	require.NoError(t, err)

	next, err := f.querier.GetNextPertemuanKe(context.Background(), f.kelasID)
	require.NoError(t, err)
	require.EqualValues(t, 1, next)

	_, err = f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{JamMulai: "08:00"})
	require.ErrorIs(t, err, ErrJadwalPertemuanAktif)
	require.NoError(t, f.service.Cancel(jadwalID, f.kelasID))

	spontan, err := f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{JamMulai: "08:00"})
	require.NoError(t, err)
	require.EqualValues(t, 1, spontan.PertemuanKe)
	active, err := f.pertemuan.GetActivePertemuan(f.kelasID)
	require.NoError(t, err)
	require.NotNil(t, active)
	require.Equal(t, spontan.ID, active.ID)
	_, err = f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{JamMulai: "08:30"})
	require.ErrorIs(t, err, ErrPertemuanBerlangsung)
}

func TestJadwalMendatangTidakMenghalangiPertemuanSpontan(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	_, err := f.service.Create(f.guruUtamaUser, models.BuatJadwalPertemuanRequest{
		KelasID: f.kelasID, Tanggal: tomorrow, JamMulai: "09:00",
	})
	require.NoError(t, err)

	due, err := f.service.ListDue(f.kelasID)
	require.NoError(t, err)
	require.Empty(t, due)
	spontan, err := f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{JamMulai: "08:00"})
	require.NoError(t, err)
	require.EqualValues(t, 1, spontan.PertemuanKe)
}

func TestListDueMengembalikanJadwalHariIniDanTerlewat(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1)
	_, err := f.db.Exec(`INSERT INTO jadwal_pertemuan (kelas_id, tanggal, jam_mulai, status) VALUES (?, ?, '08:00', 'dijadwalkan')`, f.kelasID, yesterday)
	require.NoError(t, err)
	_, err = f.service.Create(f.guruUtamaUser, models.BuatJadwalPertemuanRequest{KelasID: f.kelasID, Tanggal: today, JamMulai: "09:00"})
	require.NoError(t, err)
	_, err = f.service.Create(f.guruUtamaUser, models.BuatJadwalPertemuanRequest{KelasID: f.kelasID, Tanggal: time.Now().AddDate(0, 0, 1).Format("2006-01-02"), JamMulai: "10:00"})
	require.NoError(t, err)

	due, err := f.service.ListDue(f.kelasID)
	require.NoError(t, err)
	require.Len(t, due, 2)
	require.Equal(t, yesterday.Format("2006-01-02"), due[0].Tanggal)
	require.Equal(t, today, due[1].Tanggal)
}

func TestGuruBadalDapatMemulaiJadwalDanMetadataTersalin(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	today := time.Now().Format("2006-01-02")
	jadwalID, err := f.service.Create(f.guruUtamaUser, models.BuatJadwalPertemuanRequest{
		KelasID: f.kelasID, Tanggal: today, JamMulai: "09:00", Catatan: "Persiapan materi",
	})
	require.NoError(t, err)
	require.NoError(t, f.service.Reschedule(jadwalID, f.kelasID, models.RescheduleRequest{
		TanggalBaru: today, JamBaru: "10:00", Alasan: "Penyesuaian waktu",
	}))
	require.NoError(t, f.service.Badal(jadwalID, f.kelasID, models.BadalRequest{
		GuruPenggantiID: f.guruBadalID, Alasan: "Guru utama berhalangan",
	}))

	badalID := f.guruBadalID
	badalList, err := f.service.List(&badalID)
	require.NoError(t, err)
	require.Len(t, badalList, 1)
	require.False(t, badalList[0].CanManage)
	require.True(t, badalList[0].CanStart)

	pertemuan, err := f.service.Start(jadwalID, f.kelasID, f.guruBadalUser, false)
	require.NoError(t, err)
	require.EqualValues(t, 1, pertemuan.PertemuanKe)
	require.True(t, pertemuan.IsReschedule)
	require.True(t, pertemuan.IsBadal)
	require.Equal(t, "Penyesuaian waktu", pertemuan.AlasanReschedule)
	require.Equal(t, "Guru utama berhalangan", pertemuan.AlasanBadal)
	require.NotNil(t, pertemuan.GuruPenggantiID)
	require.Equal(t, f.guruBadalID, *pertemuan.GuruPenggantiID)

	stored, _, err := f.pertemuan.GetPertemuanByID(pertemuan.ID, f.kelasID)
	require.NoError(t, err)
	require.Equal(t, "Persiapan materi", stored.Catatan)
	require.True(t, stored.IsReschedule)
	require.True(t, stored.IsBadal)

	badalList, err = f.service.List(&badalID)
	require.NoError(t, err)
	require.Len(t, badalList, 1)
	require.Equal(t, "dimulai", badalList[0].Status)
	require.NotNil(t, badalList[0].PertemuanID)
	require.Error(t, f.service.Reschedule(jadwalID, f.kelasID, models.RescheduleRequest{
		TanggalBaru: today, JamBaru: "11:00",
	}))
	require.NoError(t, f.pertemuan.SelesaiPertemuan(pertemuan.ID, f.kelasID, models.SelesaiPertemuanRequest{Materi: "Materi uji"}, f.guruBadalUser))
	badalList, err = f.service.List(&badalID)
	require.NoError(t, err)
	require.Empty(t, badalList)
}

func TestJadwalMenolakGuruYangTidakDitugaskanDanTanggalMendatang(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	jadwalID, err := f.service.Create(f.guruUtamaUser, models.BuatJadwalPertemuanRequest{
		KelasID: f.kelasID, Tanggal: tomorrow, JamMulai: "09:00",
	})
	require.NoError(t, err)

	_, err = f.service.Start(jadwalID, f.kelasID, f.guruUtamaUser, false)
	require.ErrorContains(t, err, "tanggal yang dijadwalkan")

	require.NoError(t, f.service.Reschedule(jadwalID, f.kelasID, models.RescheduleRequest{
		TanggalBaru: time.Now().Format("2006-01-02"), JamBaru: "09:00",
	}))
	_, err = f.service.Start(jadwalID, f.kelasID, f.guruLainUser, false)
	require.ErrorContains(t, err, "tidak memiliki akses")
}

func TestRiwayatHanyaMenampilkanPertemuanSelesai(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	berlangsung, err := f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{JamMulai: "08:00"})
	require.NoError(t, err)

	riwayat, err := f.pertemuan.ListRiwayat(f.kelasID)
	require.NoError(t, err)
	require.Empty(t, riwayat)

	require.NoError(t, f.querier.UpdatePertemuanStatus(context.Background(), queries.UpdatePertemuanStatusParams{Status: "selesai", ID: berlangsung.ID}))
	riwayat, err = f.pertemuan.ListRiwayat(f.kelasID)
	require.NoError(t, err)
	require.Len(t, riwayat, 1)
	require.Equal(t, "selesai", riwayat[0].Status)
}

func TestSelesaiPertemuanMemvalidasiDanRollbackAbsensi(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	_, err := f.db.Exec(`INSERT INTO santri (nama, kelas_id, status) VALUES ('Santri Uji', ?, 'aktif')`, f.kelasID)
	require.NoError(t, err)
	santriID, err := lastID(f.db)
	require.NoError(t, err)
	pertemuan, err := f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{JamMulai: "08:00"})
	require.NoError(t, err)

	err = f.pertemuan.SelesaiPertemuan(pertemuan.ID, f.kelasID, models.SelesaiPertemuanRequest{
		Materi: "Materi", Absensi: []models.AbsensiInput{{SantriID: santriID, Status: "invalid"}},
	}, f.guruUtamaUser)
	require.ErrorContains(t, err, "tidak valid")

	stored, _, err := f.pertemuan.GetPertemuanByID(pertemuan.ID, f.kelasID)
	require.NoError(t, err)
	require.Equal(t, "berlangsung", stored.Status)
	var absensiCount int64
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM absensi WHERE pertemuan_id = ?`, pertemuan.ID).Scan(&absensiCount))
	require.Zero(t, absensiCount)

	err = f.pertemuan.SelesaiPertemuan(pertemuan.ID, f.kelasID, models.SelesaiPertemuanRequest{Materi: "Materi"}, f.guruUtamaUser)
	require.ErrorContains(t, err, "seluruh santri aktif")
	stored, _, err = f.pertemuan.GetPertemuanByID(pertemuan.ID, f.kelasID)
	require.NoError(t, err)
	require.Equal(t, "berlangsung", stored.Status)

	require.NoError(t, f.pertemuan.SelesaiPertemuan(pertemuan.ID, f.kelasID, models.SelesaiPertemuanRequest{
		Materi: "Materi", Absensi: []models.AbsensiInput{{SantriID: santriID, Status: "hadir", Catatan: "Menyimak dengan baik"}},
	}, f.guruUtamaUser))
	_, absensi, err := f.pertemuan.GetPertemuanByID(pertemuan.ID, f.kelasID)
	require.NoError(t, err)
	require.Len(t, absensi, 1)
	require.Equal(t, "Menyimak dengan baik", absensi[0].Catatan)
}

func TestSelesaiPertemuanMateriIndividualMewajibkanDanMeneruskanBatasMateri(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	_, err := f.db.Exec(`UPDATE kelas SET materi_individual = 1 WHERE id = ?`, f.kelasID)
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO santri (nama, kelas_id, status) VALUES ('Santri Uji', ?, 'aktif')`, f.kelasID)
	require.NoError(t, err)
	santriID, err := lastID(f.db)
	require.NoError(t, err)

	pertama, err := f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{JamMulai: "08:00"})
	require.NoError(t, err)
	err = f.pertemuan.SelesaiPertemuan(pertama.ID, f.kelasID, models.SelesaiPertemuanRequest{
		Absensi: []models.AbsensiInput{{SantriID: santriID, Status: "hadir"}},
	}, f.guruUtamaUser)
	require.ErrorContains(t, err, "batas materi wajib")

	require.NoError(t, f.pertemuan.SelesaiPertemuan(pertama.ID, f.kelasID, models.SelesaiPertemuanRequest{
		Absensi: []models.AbsensiInput{{SantriID: santriID, Status: "hadir", BatasMateri: "Jilid 2 halaman 7"}},
	}, f.guruUtamaUser))

	kedua, err := f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{JamMulai: "09:00"})
	require.NoError(t, err)
	require.NoError(t, f.pertemuan.SelesaiPertemuan(kedua.ID, f.kelasID, models.SelesaiPertemuanRequest{
		Absensi: []models.AbsensiInput{{SantriID: santriID, Status: "izin", BatasMateri: "Harus diabaikan"}},
	}, f.guruUtamaUser))

	_, absensi, err := f.pertemuan.GetPertemuanByID(kedua.ID, f.kelasID)
	require.NoError(t, err)
	require.Len(t, absensi, 1)
	require.Empty(t, absensi[0].BatasMateri)

	last, err := f.pertemuan.GetBatasMateriTerakhir(f.kelasID)
	require.NoError(t, err)
	require.Equal(t, "Jilid 2 halaman 7", last[santriID])
	require.ErrorContains(t, f.pertemuan.EditAbsensi(f.kelasID, absensi[0].ID, "hadir", "", ""), "batas materi wajib")
}

func TestGetLastCompletedAbsensiHanyaMengambilPertemuanTerakhir(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	_, err := f.db.Exec(`INSERT INTO santri (nama, kelas_id, status) VALUES ('Santri Uji', ?, 'aktif')`, f.kelasID)
	require.NoError(t, err)
	santriID, err := lastID(f.db)
	require.NoError(t, err)

	pertama, err := f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{JamMulai: "08:00"})
	require.NoError(t, err)
	require.NoError(t, f.pertemuan.SelesaiPertemuan(pertama.ID, f.kelasID, models.SelesaiPertemuanRequest{
		Materi: "Materi pertama", Absensi: []models.AbsensiInput{{SantriID: santriID, Status: "izin", Catatan: "Catatan lama"}},
	}, f.guruUtamaUser))

	kedua, err := f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{JamMulai: "09:00"})
	require.NoError(t, err)
	require.NoError(t, f.pertemuan.SelesaiPertemuan(kedua.ID, f.kelasID, models.SelesaiPertemuanRequest{
		Materi: "Materi terbaru", Absensi: []models.AbsensiInput{{SantriID: santriID, Status: "hadir", Catatan: "Catatan terbaru"}},
	}, f.guruUtamaUser))

	pertemuan, absensi, err := f.pertemuan.GetLastCompletedAbsensi(f.kelasID)
	require.NoError(t, err)
	require.NotNil(t, pertemuan)
	require.Equal(t, kedua.ID, pertemuan.ID)
	require.Equal(t, "Materi terbaru", pertemuan.Materi)
	require.Len(t, absensi, 1)
	require.Equal(t, "hadir", absensi[0].Status)
	require.Equal(t, "Catatan terbaru", absensi[0].Catatan)
}

func TestGetLastCompletedAbsensiTanpaPertemuanSelesai(t *testing.T) {
	f := setupJadwalPertemuanService(t)

	pertemuan, absensi, err := f.pertemuan.GetLastCompletedAbsensi(f.kelasID)
	require.NoError(t, err)
	require.Nil(t, pertemuan)
	require.Empty(t, absensi)
}

func TestSelesaiPertemuanHanyaDapatDiprosesSekali(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	pertemuan, err := f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{JamMulai: "08:00"})
	require.NoError(t, err)

	results := make(chan error, 2)
	for range 2 {
		go func() {
			results <- f.pertemuan.SelesaiPertemuan(pertemuan.ID, f.kelasID, models.SelesaiPertemuanRequest{Materi: "Materi"}, f.guruUtamaUser)
		}()
	}
	first, second := <-results, <-results
	require.True(t, (first == nil) != (second == nil), "tepat satu request completion harus berhasil")
}

func TestStartTerjadwalDanSpontanBersamaanHanyaMembuatSatuPertemuan(t *testing.T) {
	f := setupJadwalPertemuanService(t)
	jadwalID, err := f.service.Create(f.guruUtamaUser, models.BuatJadwalPertemuanRequest{
		KelasID: f.kelasID, Tanggal: time.Now().Format("2006-01-02"), JamMulai: "09:00",
	})
	require.NoError(t, err)

	results := make(chan error, 2)
	go func() {
		_, startErr := f.service.Start(jadwalID, f.kelasID, f.guruUtamaUser, false)
		results <- startErr
	}()
	go func() {
		_, startErr := f.pertemuan.MulaiPertemuan(f.kelasID, f.guruUtamaUser, models.MulaiPertemuanRequest{JamMulai: "09:00"})
		results <- startErr
	}()
	first, second := <-results, <-results
	require.True(t, (first == nil) != (second == nil), "tepat satu jalur start harus berhasil")

	var total, active int64
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*), SUM(CASE WHEN status = 'berlangsung' THEN 1 ELSE 0 END) FROM pertemuan WHERE kelas_id = ?`, f.kelasID).Scan(&total, &active))
	require.EqualValues(t, 1, total)
	require.EqualValues(t, 1, active)
}
