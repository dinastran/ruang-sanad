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

type perubahanFixture struct {
	db       *sql.DB
	querier  *queries.Querier
	engine   *KelasEngineService
	service  *KelasPerubahanService
	kelasID  int64
	guruUser int64
	adminID  int64
	santri   []int64
}

const (
	testJadwalLama = "Senin, jam 20.30 WIB"
	testJadwalBaru = "Rabu, jam 19.00 WIB"
)

func setupKelasPerubahan(t *testing.T) perubahanFixture {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))

	_, err = db.Exec(`INSERT INTO level (kode, nama, urutan) VALUES ('01', 'PT', 1), ('02', 'TQ', 2), ('03', 'TALAQQI', 3)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO jadwal (nama) VALUES (?), (?)`, testJadwalLama, testJadwalBaru)
	require.NoError(t, err)

	guruUser := insertTestUser(t, db, "guru@example.com", "Guru Kelas")
	guruID := insertTestGuru(t, db, "Guru Kelas", guruUser)
	adminID := insertTestUser(t, db, "admin@example.com", "Admin Kelas")

	querier := queries.NewQuerier(db)
	engine := NewKelasEngineService(querier)
	kunci := engine.BuatKunciKelas("P1X", "L", "01", "1x/pekan", testJadwalLama, "2026")
	nama := namaKelas("L", "P1X", "PT", "1x/pekan", testJadwalLama, "2026")
	_, err = db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, guru_id, kapasitas, jumlah_santri) VALUES (?, '2026', 'Private', 'L', '01', '1x/pekan', ?, 1, ?, ?, 15, 2)`, kunci, testJadwalLama, nama, guruID)
	require.NoError(t, err)
	kelasID, err := lastID(db)
	require.NoError(t, err)

	santri := make([]int64, 0, 2)
	for _, n := range []string{"Ahmad", "Bilal"} {
		_, err = db.Exec(`INSERT INTO santri (nama, jenis_kelamin, kelas_kode, angkatan_kelas, level, jadwal, tipe, frekuensi, kelas_id, status, is_lengkap) VALUES (?, 'L', 'P1X', '2026', '01', ?, 'Private', '1x/pekan', ?, 'aktif', 1)`, n, testJadwalLama, kelasID)
		require.NoError(t, err)
		id, err := lastID(db)
		require.NoError(t, err)
		santri = append(santri, id)
	}

	return perubahanFixture{
		db:       db,
		querier:  querier,
		engine:   engine,
		service:  NewKelasPerubahanService(querier, engine),
		kelasID:  kelasID,
		guruUser: guruUser,
		adminID:  adminID,
		santri:   santri,
	}
}

func (f perubahanFixture) buatPertemuan(t *testing.T, status string) queries.Pertemuan {
	t.Helper()
	ctx := context.Background()
	next, err := f.querier.GetNextPertemuanKe(ctx, f.kelasID)
	require.NoError(t, err)
	result, err := f.querier.CreatePertemuan(ctx, queries.CreatePertemuanParams{
		KelasID:     f.kelasID,
		PertemuanKe: next,
		Tanggal:     time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		Status:      status,
	})
	require.NoError(t, err)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	p, err := f.querier.GetPertemuanByID(ctx, id)
	require.NoError(t, err)
	return p
}

func TestGantiLevelKelasRestartsLevelNumberingButKeepsBillingNumber(t *testing.T) {
	f := setupKelasPerubahan(t)
	for range 3 {
		p := f.buatPertemuan(t, "selesai")
		require.Equal(t, "PT", p.LevelNama)
		require.Equal(t, p.PertemuanKe, p.PertemuanLevelKe)
	}

	require.NoError(t, f.service.GantiLevelKelas(f.kelasID, "02", f.adminID))

	kelas, err := f.querier.GetKelasByID(context.Background(), f.kelasID)
	require.NoError(t, err)
	require.Equal(t, "02", kelas.Level)
	require.Equal(t, f.engine.BuatKunciKelas("P1X", "L", "02", "1x/pekan", testJadwalLama, "2026"), kelas.KunciKelas)
	require.Equal(t, namaKelas("L", "P1X", "TQ", "1x/pekan", testJadwalLama, "2026"), kelas.NamaKelas)
	require.EqualValues(t, 1, kelas.SubIndex)

	var santriLevel string
	require.NoError(t, f.db.QueryRow(`SELECT level FROM santri WHERE id = ?`, f.santri[0]).Scan(&santriLevel))
	require.Equal(t, "02", santriLevel)

	p := f.buatPertemuan(t, "selesai")
	require.EqualValues(t, 4, p.PertemuanKe, "internal numbering must continue for billing")
	require.EqualValues(t, 1, p.PertemuanLevelKe)
	require.Equal(t, "TQ", p.LevelNama)
	nextLevelKe, err := f.querier.GetNextPertemuanLevelKe(context.Background(), f.kelasID)
	require.NoError(t, err)
	require.EqualValues(t, 2, nextLevelKe)

	var jenis, lama, baru string
	var pertemuanKe int64
	require.NoError(t, f.db.QueryRow(`SELECT jenis, nilai_lama, nilai_baru, pertemuan_ke FROM kelas_perubahan WHERE kelas_id = ?`, f.kelasID).Scan(&jenis, &lama, &baru, &pertemuanKe))
	require.Equal(t, PerubahanLevelKelas, jenis)
	require.Equal(t, "PT", lama)
	require.Equal(t, "TQ", baru)
	require.EqualValues(t, 3, pertemuanKe)

	var notif int64
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND type = 'perubahan_kelas'`, f.guruUser).Scan(&notif))
	require.EqualValues(t, 1, notif)
}

func TestGantiLevelKelasKeepsSantriInSameClassForEngine(t *testing.T) {
	f := setupKelasPerubahan(t)
	require.NoError(t, f.service.GantiLevelKelas(f.kelasID, "02", f.adminID))

	santri, err := f.querier.GetSantriByID(context.Background(), f.santri[0])
	require.NoError(t, err)
	require.NoError(t, f.engine.ProcessSantri(context.Background(), &santri))
	santri, err = f.querier.GetSantriByID(context.Background(), f.santri[0])
	require.NoError(t, err)
	require.Equal(t, f.kelasID, santri.KelasID.Int64)
}

func TestGantiLevelKelasRejectsInvalidRequests(t *testing.T) {
	f := setupKelasPerubahan(t)
	require.ErrorContains(t, f.service.GantiLevelKelas(f.kelasID, "01", f.adminID), "sama")
	require.ErrorContains(t, f.service.GantiLevelKelas(f.kelasID, "99", f.adminID), "tidak ditemukan")

	f.buatPertemuan(t, "berlangsung")
	require.ErrorContains(t, f.service.GantiLevelKelas(f.kelasID, "02", f.adminID), "berlangsung")
	kelas, err := f.querier.GetKelasByID(context.Background(), f.kelasID)
	require.NoError(t, err)
	require.Equal(t, "01", kelas.Level)
}

func TestGantiLevelKelasStaysSeparateWhenKunciAlreadyUsed(t *testing.T) {
	f := setupKelasPerubahan(t)
	kunciTQ := f.engine.BuatKunciKelas("P1X", "L", "02", "1x/pekan", testJadwalLama, "2026")
	_, err := f.db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas) VALUES (?, '2026', 'Private', 'L', '02', '1x/pekan', ?, 1, 'Kelas TQ lain', 15)`, kunciTQ, testJadwalLama)
	require.NoError(t, err)

	require.NoError(t, f.service.GantiLevelKelas(f.kelasID, "02", f.adminID))
	kelas, err := f.querier.GetKelasByID(context.Background(), f.kelasID)
	require.NoError(t, err)
	require.Equal(t, kunciTQ, kelas.KunciKelas)
	require.EqualValues(t, 2, kelas.SubIndex)
	require.Contains(t, kelas.NamaKelas, "— Kelas 2")
}

func TestGantiJadwalKelasUpdatesClassAndFlagsScheduledSessions(t *testing.T) {
	f := setupKelasPerubahan(t)
	_, err := f.db.Exec(`INSERT INTO jadwal_pertemuan (kelas_id, tanggal, jam_mulai, status) VALUES (?, '2026-10-05', '20:30', 'dijadwalkan'), (?, '2026-09-01', '20:30', 'selesai')`, f.kelasID, f.kelasID)
	require.NoError(t, err)

	require.ErrorContains(t, f.service.GantiJadwalKelas(f.kelasID, "Jumat, jam 10.00 WIB", f.adminID), "tidak ditemukan")
	require.ErrorContains(t, f.service.GantiJadwalKelas(f.kelasID, testJadwalLama, f.adminID), "sama")
	require.NoError(t, f.service.GantiJadwalKelas(f.kelasID, testJadwalBaru, f.adminID))

	kelas, err := f.querier.GetKelasByID(context.Background(), f.kelasID)
	require.NoError(t, err)
	require.Equal(t, testJadwalBaru, kelas.Jadwal)
	require.Equal(t, f.engine.BuatKunciKelas("P1X", "L", "01", "1x/pekan", testJadwalBaru, "2026"), kelas.KunciKelas)
	require.Contains(t, kelas.NamaKelas, testJadwalBaru)

	var santriJadwal string
	require.NoError(t, f.db.QueryRow(`SELECT jadwal FROM santri WHERE id = ?`, f.santri[1]).Scan(&santriJadwal))
	require.Equal(t, testJadwalBaru, santriJadwal)

	var flagged, untouched int64
	require.NoError(t, f.db.QueryRow(`SELECT jadwal_kelas_berubah FROM jadwal_pertemuan WHERE status = 'dijadwalkan'`).Scan(&flagged))
	require.NoError(t, f.db.QueryRow(`SELECT jadwal_kelas_berubah FROM jadwal_pertemuan WHERE status = 'selesai'`).Scan(&untouched))
	require.EqualValues(t, 1, flagged)
	require.EqualValues(t, 0, untouched)

	riwayat, err := f.service.ListRiwayat(f.kelasID)
	require.NoError(t, err)
	require.Len(t, riwayat, 1)
	require.Equal(t, PerubahanJadwalKelas, riwayat[0].Jenis)
	require.Equal(t, "Admin Kelas", riwayat[0].DibuatOlehNama)
}

func TestGantiLevelSantriToNewClass(t *testing.T) {
	f := setupKelasPerubahan(t)
	f.buatPertemuan(t, "selesai")

	tujuanID, err := f.service.GantiLevelSantri(f.kelasID, models.GantiLevelSantriRequest{
		SantriIDs:     []int64{f.santri[0]},
		BuatKelasBaru: true,
		Level:         "03",
	}, f.adminID)
	require.NoError(t, err)
	require.NotEqual(t, f.kelasID, tujuanID)

	tujuan, err := f.querier.GetKelasByID(context.Background(), tujuanID)
	require.NoError(t, err)
	require.Equal(t, "03", tujuan.Level)
	require.Equal(t, testJadwalLama, tujuan.Jadwal)
	require.Equal(t, "Private", tujuan.Tipe)
	require.EqualValues(t, 1, tujuan.JumlahSantri)

	santri, err := f.querier.GetSantriByID(context.Background(), f.santri[0])
	require.NoError(t, err)
	require.Equal(t, tujuanID, santri.KelasID.Int64)
	require.Equal(t, "03", santri.Level)
	// The new class continues the origin's internal numbering (1 meeting so far),
	// so the moved santri's billing periods keep advancing.
	require.EqualValues(t, 2, santri.PertemuanAwal)
	nextKe, err := f.querier.GetNextPertemuanKe(context.Background(), tujuanID)
	require.NoError(t, err)
	require.EqualValues(t, 2, nextKe)
	nextLevelKe, err := f.querier.GetNextPertemuanLevelKe(context.Background(), tujuanID)
	require.NoError(t, err)
	require.EqualValues(t, 1, nextLevelKe)

	other, err := f.querier.GetSantriByID(context.Background(), f.santri[1])
	require.NoError(t, err)
	require.Equal(t, f.kelasID, other.KelasID.Int64)

	riwayat, err := f.service.ListRiwayat(tujuanID)
	require.NoError(t, err)
	require.Len(t, riwayat, 1)
	require.Equal(t, "Ahmad", riwayat[0].SantriNama)
	require.Equal(t, "PT", riwayat[0].NilaiLama)
	require.Equal(t, "TALAQQI", riwayat[0].NilaiBaru)
}

func TestGantiLevelSantriRejectsInvalidRequests(t *testing.T) {
	f := setupKelasPerubahan(t)
	_, err := f.service.GantiLevelSantri(f.kelasID, models.GantiLevelSantriRequest{}, f.adminID)
	require.ErrorContains(t, err, "minimal satu")

	_, err = f.db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas) VALUES ('LAIN-PT', '2026', 'Private', 'L', '01', '1x/pekan', ?, 1, 'Kelas PT lain', 15)`, testJadwalLama)
	require.NoError(t, err)
	samaLevelID, err := lastID(f.db)
	require.NoError(t, err)
	_, err = f.service.GantiLevelSantri(f.kelasID, models.GantiLevelSantriRequest{SantriIDs: f.santri, KelasTujuanID: samaLevelID}, f.adminID)
	require.ErrorContains(t, err, "level kelas tujuan sama")

	_, err = f.db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas) VALUES ('PENUH-TQ', '2026', 'Private', 'L', '02', '1x/pekan', ?, 1, 'Kelas TQ kecil', 1)`, testJadwalLama)
	require.NoError(t, err)
	kecilID, err := lastID(f.db)
	require.NoError(t, err)
	_, err = f.service.GantiLevelSantri(f.kelasID, models.GantiLevelSantriRequest{SantriIDs: f.santri, KelasTujuanID: kecilID}, f.adminID)
	require.ErrorContains(t, err, "tidak cukup")

	_, err = f.service.GantiLevelSantri(kecilID, models.GantiLevelSantriRequest{SantriIDs: f.santri, BuatKelasBaru: true, Level: "03"}, f.adminID)
	require.ErrorContains(t, err, "tidak terdaftar")

	var moved int64
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM santri WHERE kelas_id != ?`, f.kelasID).Scan(&moved))
	require.Zero(t, moved)
}

func TestGantiLevelSantriNewClassDoesNotSkipBilledPeriods(t *testing.T) {
	f := setupKelasPerubahan(t)
	_, err := f.db.Exec(`UPDATE santri SET nominal = 100000`)
	require.NoError(t, err)
	tagihan := NewTagihanService(f.querier)
	for range 12 {
		p := f.buatPertemuan(t, "selesai")
		require.NoError(t, tagihan.GenerateForPertemuan(p.ID))
	}

	tujuanID, err := f.service.GantiLevelSantri(f.kelasID, models.GantiLevelSantriRequest{
		SantriIDs: []int64{f.santri[0], f.santri[0]}, BuatKelasBaru: true, Level: "02",
	}, f.adminID)
	require.NoError(t, err)

	var riwayat int64
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM kelas_perubahan WHERE santri_id = ?`, f.santri[0]).Scan(&riwayat))
	require.EqualValues(t, 1, riwayat, "duplicate santri ids are ignored")

	baru := perubahanFixture{db: f.db, querier: f.querier, kelasID: tujuanID}
	var last queries.Pertemuan
	for range 4 {
		last = baru.buatPertemuan(t, "selesai")
		require.NoError(t, tagihan.GenerateForPertemuan(last.ID))
	}
	require.EqualValues(t, 16, last.PertemuanKe)
	require.EqualValues(t, 4, last.PertemuanLevelKe)

	var bulanKe int64
	require.NoError(t, f.db.QueryRow(`SELECT MAX(bulan_ke) FROM tagihan WHERE santri_id = ?`, f.santri[0]).Scan(&bulanKe))
	require.EqualValues(t, 4, bulanKe, "month 4 must be billed after moving to the new-level class")
}

func TestGantiLevelSantriNewClassFitsLargeSelection(t *testing.T) {
	f := setupKelasPerubahan(t)
	_, err := f.db.Exec(`UPDATE kelas SET kapasitas = 30 WHERE id = ?`, f.kelasID)
	require.NoError(t, err)
	ids := append([]int64{}, f.santri...)
	for range 15 {
		_, err := f.db.Exec(`INSERT INTO santri (nama, jenis_kelamin, kelas_kode, angkatan_kelas, level, jadwal, kelas_id, status, is_lengkap) VALUES ('Santri', 'L', 'P1X', '2026', '01', ?, ?, 'aktif', 1)`, testJadwalLama, f.kelasID)
		require.NoError(t, err)
		id, err := lastID(f.db)
		require.NoError(t, err)
		ids = append(ids, id)
	}
	tujuanID, err := f.service.GantiLevelSantri(f.kelasID, models.GantiLevelSantriRequest{SantriIDs: ids, BuatKelasBaru: true, Level: "02"}, f.adminID)
	require.NoError(t, err)
	tujuan, err := f.querier.GetKelasByID(context.Background(), tujuanID)
	require.NoError(t, err)
	require.EqualValues(t, 17, tujuan.JumlahSantri)
	require.EqualValues(t, 17, tujuan.Kapasitas)
}

func TestAnchorEditAfterLevelChangeKeepsLevelNumbering(t *testing.T) {
	f := setupKelasPerubahan(t)
	kelasService := NewKelasService(f.querier)
	require.NoError(t, kelasService.SetPertemuanTerakhir(f.kelasID, 20))
	require.NoError(t, f.service.GantiLevelKelas(f.kelasID, "02", f.adminID))

	for _, anchor := range []int64{5, 12} {
		require.NoError(t, kelasService.SetPertemuanTerakhir(f.kelasID, anchor))
		next, err := f.querier.GetNextPertemuanLevelKe(context.Background(), f.kelasID)
		require.NoError(t, err)
		require.EqualValues(t, 1, next)
	}
	p := f.buatPertemuan(t, "selesai")
	require.EqualValues(t, 13, p.PertemuanKe)
	require.EqualValues(t, 1, p.PertemuanLevelKe)
}

func TestGantiLevelKelasLegacyKunciUsesSantriKelasKode(t *testing.T) {
	f := setupKelasPerubahan(t)
	_, err := f.db.Exec(`UPDATE kelas SET kunci_kelas = 'LEGACY' WHERE id = ?`, f.kelasID)
	require.NoError(t, err)
	require.NoError(t, f.service.GantiLevelKelas(f.kelasID, "02", f.adminID))

	santri, err := f.querier.GetSantriByID(context.Background(), f.santri[0])
	require.NoError(t, err)
	require.NoError(t, f.engine.ProcessSantri(context.Background(), &santri))
	santri, err = f.querier.GetSantriByID(context.Background(), f.santri[0])
	require.NoError(t, err)
	require.Equal(t, f.kelasID, santri.KelasID.Int64)
}
