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

func setupTagihanService(t *testing.T, frekuensi string, pertemuanAwal int64) (*sql.DB, *TagihanService, int64) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	_, err = db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri, created_at) VALUES ('T', '2026', 'Reguler', 'L', 'Dasar', ?, 'Senin', 1, 'Kelas Uji', 20, 1, CURRENT_TIMESTAMP)`, frekuensi)
	require.NoError(t, err)
	kelasID, err := lastID(db)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO santri (nama, jenis_kelamin, nominal, frekuensi, kelas_id, status, pertemuan_awal, angkatan_kelas) VALUES ('Ahmad', 'L', 100000, ?, ?, 'aktif', ?, '2026')`, frekuensi, kelasID, pertemuanAwal)
	require.NoError(t, err)
	return db, NewTagihanService(queries.NewQuerier(db)), kelasID
}

func createSelesaiPertemuan(t *testing.T, db *sql.DB, kelasID, pertemuanKe int64) int64 {
	t.Helper()
	_, err := db.Exec(`INSERT INTO pertemuan (kelas_id, pertemuan_ke, tanggal, status) VALUES (?, ?, ?, 'selesai')`, kelasID, pertemuanKe, time.Date(2026, 7, int(pertemuanKe%28+1), 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	id, err := lastID(db)
	require.NoError(t, err)
	return id
}

func firstSantriID(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var id int64
	require.NoError(t, db.QueryRow(`SELECT id FROM santri ORDER BY id LIMIT 1`).Scan(&id))
	return id
}

func addAbsensi(t *testing.T, db *sql.DB, pertemuanID, santriID int64, status string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO absensi (pertemuan_id, santri_id, status) VALUES (?, ?, ?)`, pertemuanID, santriID, status)
	require.NoError(t, err)
}

func createMeetingWithAbsensi(t *testing.T, db *sql.DB, kelasID, pertemuanKe, santriID int64, status string) int64 {
	t.Helper()
	id := createSelesaiPertemuan(t, db, kelasID, pertemuanKe)
	addAbsensi(t, db, id, santriID, status)
	return id
}

func generateMeetings(t *testing.T, db *sql.DB, service *TagihanService, kelasID, santriID, start, count int64, status string) []int64 {
	t.Helper()
	ids := make([]int64, 0, count)
	for i := int64(0); i < count; i++ {
		id := createMeetingWithAbsensi(t, db, kelasID, start+i, santriID, status)
		require.NoError(t, service.GenerateForPertemuan(id))
		ids = append(ids, id)
	}
	return ids
}

func lastID(db *sql.DB) (int64, error) {
	var id int64
	err := db.QueryRow(`SELECT last_insert_rowid()`).Scan(&id)
	return id, err
}

func tagihanCount(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM tagihan`).Scan(&count))
	return count
}

func TestTagihan1xTerbitSetelahEmpatPertemuanSantri(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)

	ids := generateMeetings(t, db, service, kelasID, santriID, 1, 3, "hadir")
	require.Len(t, ids, 3)
	require.Zero(t, tagihanCount(t, db))

	p4 := createMeetingWithAbsensi(t, db, kelasID, 4, santriID, "hadir")
	require.NoError(t, service.GenerateForPertemuan(p4))
	require.EqualValues(t, 1, tagihanCount(t, db))
	var bulanKe, pertemuanKe int64
	require.NoError(t, db.QueryRow(`SELECT bulan_ke, pertemuan_ke FROM tagihan`).Scan(&bulanKe, &pertemuanKe))
	require.EqualValues(t, 2, bulanKe)
	require.EqualValues(t, 4, pertemuanKe)

	generateMeetings(t, db, service, kelasID, santriID, 5, 4, "hadir")
	require.EqualValues(t, 2, tagihanCount(t, db))
	require.NoError(t, db.QueryRow(`SELECT pertemuan_ke FROM tagihan WHERE bulan_ke = 3`).Scan(&pertemuanKe))
	require.EqualValues(t, 8, pertemuanKe)

	require.NoError(t, service.GenerateForPertemuan(p4))
	require.NoError(t, service.Sync())
	require.EqualValues(t, 2, tagihanCount(t, db), "retry dan Sync tidak boleh menggandakan tagihan")
}

func TestTagihan2xTerbitSetelahDelapanPertemuanSantri(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "2x/pekan", 0)
	santriID := firstSantriID(t, db)

	generateMeetings(t, db, service, kelasID, santriID, 1, 7, "hadir")
	require.Zero(t, tagihanCount(t, db))
	generateMeetings(t, db, service, kelasID, santriID, 8, 1, "hadir")
	require.EqualValues(t, 1, tagihanCount(t, db))

	generateMeetings(t, db, service, kelasID, santriID, 9, 8, "hadir")
	require.EqualValues(t, 2, tagihanCount(t, db))
	var bulan2, bulan3 int64
	require.NoError(t, db.QueryRow(`SELECT pertemuan_ke FROM tagihan WHERE bulan_ke = 2`).Scan(&bulan2))
	require.NoError(t, db.QueryRow(`SELECT pertemuan_ke FROM tagihan WHERE bulan_ke = 3`).Scan(&bulan3))
	require.EqualValues(t, 8, bulan2)
	require.EqualValues(t, 16, bulan3)
}

func TestSemuaStatusAbsensiMenghitungSatuPertemuan(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	statuses := []string{"hadir", "izin", "sakit", "alpa"}
	for i, status := range statuses {
		p := createMeetingWithAbsensi(t, db, kelasID, int64(i+1), santriID, status)
		require.NoError(t, service.GenerateForPertemuan(p))
	}
	require.EqualValues(t, 1, tagihanCount(t, db))

	p5 := createMeetingWithAbsensi(t, db, kelasID, 5, santriID, "telat")
	require.NoError(t, service.GenerateForPertemuan(p5))
	var counter int64
	require.NoError(t, db.QueryRow(`SELECT meeting_count FROM santri_billing_progress WHERE santri_id = ?`, santriID).Scan(&counter))
	require.EqualValues(t, 1, counter, "status telat juga bernilai satu pertemuan")
}

func TestSantriMasukTengahKelasMengikutiPertemuanPribadi(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "1x/pekan", 7)
	santriID := firstSantriID(t, db)

	// Nomor kelas sudah P7, tetapi ini adalah pertemuan pertama santri.
	generateMeetings(t, db, service, kelasID, santriID, 7, 3, "hadir")
	require.Zero(t, tagihanCount(t, db))
	p10 := createMeetingWithAbsensi(t, db, kelasID, 10, santriID, "hadir")
	require.NoError(t, service.GenerateForPertemuan(p10))

	var bulanKe, pertemuanKe int64
	require.NoError(t, db.QueryRow(`SELECT bulan_ke, pertemuan_ke FROM tagihan`).Scan(&bulanKe, &pertemuanKe))
	require.EqualValues(t, 2, bulanKe)
	require.EqualValues(t, 10, pertemuanKe)
}

func TestCutiTidakMenambahCounterDanAktifKembaliMelanjutkan(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	generateMeetings(t, db, service, kelasID, santriID, 1, 3, "hadir")

	// Saat cuti santri tidak ada di roster, sehingga tidak ada row absensi dan
	// dua pertemuan kelas berikutnya tidak boleh menggerakkan billing.
	for _, ke := range []int64{4, 5} {
		p := createSelesaiPertemuan(t, db, kelasID, ke)
		require.NoError(t, service.GenerateForPertemuan(p))
	}
	require.Zero(t, tagihanCount(t, db))

	p6 := createMeetingWithAbsensi(t, db, kelasID, 6, santriID, "izin")
	require.NoError(t, service.GenerateForPertemuan(p6))
	var pertemuanKe int64
	require.NoError(t, db.QueryRow(`SELECT pertemuan_ke FROM tagihan WHERE bulan_ke = 2`).Scan(&pertemuanKe))
	require.EqualValues(t, 6, pertemuanKe)
}

func TestPindahKelasTidakMeresetCounter(t *testing.T) {
	db, service, asalID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	generateMeetings(t, db, service, asalID, santriID, 1, 2, "hadir")

	_, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri) VALUES ('B', '2027', 'Reguler', 'L', 'Dasar', '1x/pekan', 'Selasa', 1, 'Kelas Tujuan', 20, 0)`)
	require.NoError(t, err)
	tujuanID, err := lastID(db)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE santri SET kelas_id = ?, angkatan_kelas = '2027' WHERE id = ?`, tujuanID, santriID)
	require.NoError(t, err)

	generateMeetings(t, db, service, tujuanID, santriID, 1, 2, "hadir")
	var bulanKe, kelasTagihan int64
	require.NoError(t, db.QueryRow(`SELECT bulan_ke, kelas_id FROM tagihan`).Scan(&bulanKe, &kelasTagihan))
	require.EqualValues(t, 2, bulanKe)
	require.EqualValues(t, tujuanID, kelasTagihan)
}

func TestFrekuensiNaikMempertahankanCounter(t *testing.T) {
	db, service, asalID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	generateMeetings(t, db, service, asalID, santriID, 1, 2, "hadir")

	_, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri) VALUES ('B2', '2026', 'Reguler', 'L', 'Dasar', '2x/pekan', 'Selasa', 1, 'Kelas 2x', 20, 0)`)
	require.NoError(t, err)
	tujuanID, err := lastID(db)
	require.NoError(t, err)
	engine := NewKelasEngineService(queries.NewQuerier(db)).WithBilling(service)
	require.NoError(t, engine.PindahkanSantri(context.Background(), santriID, tujuanID))

	var counter int64
	require.NoError(t, db.QueryRow(`SELECT meeting_count FROM santri_billing_progress WHERE santri_id = ?`, santriID).Scan(&counter))
	require.EqualValues(t, 2, counter)
	require.Zero(t, tagihanCount(t, db))

	generateMeetings(t, db, service, tujuanID, santriID, 1, 6, "hadir")
	require.EqualValues(t, 1, tagihanCount(t, db), "2 pertemuan lama + 6 pertemuan baru memenuhi threshold 8")
}

func TestFrekuensiTurunLangsungMenagihDanMembawaSisaCounter(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "2x/pekan", 0)
	santriID := firstSantriID(t, db)
	generateMeetings(t, db, service, kelasID, santriID, 1, 6, "hadir")
	require.Zero(t, tagihanCount(t, db))

	changedAt := time.Date(2026, 8, 5, 11, 30, 0, 0, time.Local)
	_, err := db.Exec(`UPDATE santri SET frekuensi = '1x/pekan' WHERE id = ?`, santriID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE kelas SET frekuensi = '1x/pekan' WHERE id = ?`, kelasID)
	require.NoError(t, err)
	require.NoError(t, service.ReconcileFrequencyChangeWithQuerier(queries.NewQuerier(db), santriID, "2x/pekan", "1x/pekan", changedAt))

	var bulanKe, counter int64
	var tanggal time.Time
	require.NoError(t, db.QueryRow(`SELECT bulan_ke, tanggal_tagih FROM tagihan`).Scan(&bulanKe, &tanggal))
	require.EqualValues(t, 2, bulanKe)
	require.Equal(t, changedAt.Format("2006-01-02"), tanggal.Format("2006-01-02"))
	require.NoError(t, db.QueryRow(`SELECT meeting_count FROM santri_billing_progress WHERE santri_id = ?`, santriID).Scan(&counter))
	require.EqualValues(t, 2, counter)

	generateMeetings(t, db, service, kelasID, santriID, 7, 2, "hadir")
	require.EqualValues(t, 2, tagihanCount(t, db))
	require.NoError(t, db.QueryRow(`SELECT bulan_ke FROM tagihan WHERE bulan_ke = 3`).Scan(&bulanKe))
	require.EqualValues(t, 3, bulanKe)
}

func TestReplayHistoriLamaMembuatHanyaPeriodeYangBelumAda(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	var p8 int64
	for ke := int64(1); ke <= 8; ke++ {
		p := createMeetingWithAbsensi(t, db, kelasID, ke, santriID, "hadir")
		if ke == 8 {
			p8 = p
		}
	}
	// Bentuk histori lama: bulan 2 baru tercatat di P8.
	_, err := db.Exec(`INSERT INTO tagihan (santri_id, kelas_id, pertemuan_id, bulan_ke, pertemuan_ke, nominal, tanggal_tagih, jatuh_tempo, angkatan_kelas) VALUES (?, ?, ?, 2, 8, 100000, '2026-07-09', '2026-07-16', '2026')`, santriID, kelasID, p8)
	require.NoError(t, err)

	require.NoError(t, service.Sync())
	require.EqualValues(t, 2, tagihanCount(t, db))
	var month3Pertemuan int64
	require.NoError(t, db.QueryRow(`SELECT pertemuan_ke FROM tagihan WHERE bulan_ke = 3`).Scan(&month3Pertemuan))
	require.EqualValues(t, 8, month3Pertemuan)

	require.NoError(t, service.Sync())
	require.EqualValues(t, 2, tagihanCount(t, db))
}

func TestReplayMenjagaNomorPeriodeTagihanExisting(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	var p236 int64
	for ke := int64(233); ke <= 236; ke++ {
		p := createMeetingWithAbsensi(t, db, kelasID, ke, santriID, "hadir")
		if ke == 236 {
			p236 = p
		}
	}

	// Simulasikan santri lama yang sudah berada di bulan 59, sementara aplikasi
	// hanya memiliki empat absensi digital terbaru. Counter baru harus meneruskan
	// nomor periode, bukan membuat ulang bulan 2.
	_, err := db.Exec(`INSERT INTO tagihan (santri_id, kelas_id, pertemuan_id, bulan_ke, pertemuan_ke, nominal, tanggal_tagih, jatuh_tempo, angkatan_kelas) VALUES (?, ?, ?, 59, 236, 100000, '2026-07-13', '2026-07-20', '2026')`, santriID, kelasID, p236)
	require.NoError(t, err)

	require.NoError(t, service.Sync())
	var month60 int64
	require.NoError(t, db.QueryRow(`SELECT pertemuan_ke FROM tagihan WHERE bulan_ke = 60`).Scan(&month60))
	require.EqualValues(t, 236, month60)
	var lowMonths int64
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM tagihan WHERE bulan_ke BETWEEN 2 AND 58`).Scan(&lowMonths))
	require.Zero(t, lowMonths)
}

func TestBaselineNomorKelasTidakMenentukanBulanTagihan(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	_, err := db.Exec(`UPDATE kelas SET pertemuan_terakhir = 232 WHERE id = ?`, kelasID)
	require.NoError(t, err)

	next, err := queries.NewQuerier(db).GetNextPertemuanKe(context.Background(), kelasID)
	require.NoError(t, err)
	require.EqualValues(t, 233, next)
	require.Error(t, NewKelasService(queries.NewQuerier(db)).SetPertemuanTerakhir(kelasID, 300))

	generateMeetings(t, db, service, kelasID, santriID, 233, 4, "hadir")
	var bulanKe, pertemuanKe int64
	require.NoError(t, db.QueryRow(`SELECT bulan_ke, pertemuan_ke FROM tagihan`).Scan(&bulanKe, &pertemuanKe))
	require.EqualValues(t, 2, bulanKe)
	require.EqualValues(t, 236, pertemuanKe)
}

func TestTagihanKeepsBilledClassCohortAfterSantriMoves(t *testing.T) {
	db, service, oldKelasID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	generateMeetings(t, db, service, oldKelasID, santriID, 1, 4, "hadir")
	var tagihanID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM tagihan`).Scan(&tagihanID))

	_, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas) VALUES ('NEW', '2027', 'Reguler', 'L', 'Dasar', '1x/pekan', 'Selasa', 1, 'Kelas Baru', 20)`)
	require.NoError(t, err)
	newKelasID, err := lastID(db)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE santri SET kelas_id = ?, angkatan_kelas = '2027' WHERE id = ?`, newKelasID, santriID)
	require.NoError(t, err)

	item, err := service.Get(tagihanID)
	require.NoError(t, err)
	require.Equal(t, "2026", item.AngkatanKelas)
	oldCohort, err := service.List(models.TagihanFilter{AngkatanKelas: "2026"})
	require.NoError(t, err)
	require.Len(t, oldCohort, 1)
	newCohort, err := service.List(models.TagihanFilter{AngkatanKelas: "2027"})
	require.NoError(t, err)
	require.Empty(t, newCohort)

	_, err = db.Exec(`DELETE FROM kelas WHERE id = ?`, oldKelasID)
	require.NoError(t, err)
	item, err = service.Get(tagihanID)
	require.NoError(t, err)
	require.Equal(t, "2026", item.AngkatanKelas)
	require.Nil(t, item.KelasID)
}

func TestMigration0027SnapshotsExistingTagihanClassCohort(t *testing.T) {
	migrationsPath, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	require.NoError(t, err)
	t.Chdir(t.TempDir())
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.UpTo(db, migrationsPath, 26))

	result, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, nama_kelas) VALUES ('OLD', '2026', 'Reguler', 'L', 'Dasar', '1x/pekan', 'Kelas Lama')`)
	require.NoError(t, err)
	kelasID, err := result.LastInsertId()
	require.NoError(t, err)
	result, err = db.Exec(`INSERT INTO santri (nama, angkatan_kelas, kelas_id) VALUES ('Ahmad', '2027', ?)`, kelasID)
	require.NoError(t, err)
	santriID, err := result.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO tagihan (santri_id, kelas_id, bulan_ke, pertemuan_ke, tanggal_tagih) VALUES (?, ?, 2, 8, '2026-07-01')`, santriID, kelasID)
	require.NoError(t, err)

	require.NoError(t, goose.UpTo(db, migrationsPath, 27))
	var cohort string
	require.NoError(t, db.QueryRow(`SELECT angkatan_kelas FROM tagihan`).Scan(&cohort))
	require.Equal(t, "2026", cohort)
}

func TestTagihanWhatsAppHelpers(t *testing.T) {
	require.Equal(t, "628123456789", normalisasiNomorWA("+62 812-3456-789"))
	require.Equal(t, "628123456789", normalisasiNomorWA("0812 3456 789"))
	require.Empty(t, normalisasiNomorWA("abc"))
	require.Equal(t, "100.000", formatRibuan(100000))
	_, err := frekuensiKePertemuan("1x/pekan")
	require.NoError(t, err)
	_, err = frekuensiKePertemuan("2x/pekan")
	require.NoError(t, err)
	_, err = frekuensiKePertemuan("4x pertemuan")
	require.Error(t, err)
}

func TestTagihanNominalSyncOverrideResetDanLunas(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	generateMeetings(t, db, service, kelasID, santriID, 1, 4, "hadir")

	var tagihanID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM tagihan`).Scan(&tagihanID))
	q := queries.NewQuerier(db)

	_, err := db.Exec(`UPDATE santri SET nominal = 200000 WHERE id = ?`, santriID)
	require.NoError(t, err)
	require.NoError(t, q.SyncOpenTagihanNominalForSantri(context.Background(), santriID, 200000))
	item, err := service.Get(tagihanID)
	require.NoError(t, err)
	require.EqualValues(t, 200000, item.Nominal)
	require.False(t, item.NominalOverride)

	require.NoError(t, service.SetNominalOverride(tagihanID, 0, 150000))
	item, err = service.Get(tagihanID)
	require.NoError(t, err)
	require.EqualValues(t, 150000, item.Nominal)
	require.True(t, item.NominalOverride)

	_, err = db.Exec(`UPDATE santri SET nominal = 250000 WHERE id = ?`, santriID)
	require.NoError(t, err)
	require.NoError(t, q.SyncOpenTagihanNominalForSantri(context.Background(), santriID, 250000))
	item, err = service.Get(tagihanID)
	require.NoError(t, err)
	require.EqualValues(t, 150000, item.Nominal)

	require.NoError(t, service.ResetNominalOverride(tagihanID))
	item, err = service.Get(tagihanID)
	require.NoError(t, err)
	require.EqualValues(t, 250000, item.Nominal)
	require.False(t, item.NominalOverride)

	userResult, err := db.Exec(`INSERT INTO users (email, name, role) VALUES ('finance-test@example.com', 'Finance Test', 'keuangan')`)
	require.NoError(t, err)
	userID, err := userResult.LastInsertId()
	require.NoError(t, err)
	require.NoError(t, service.MarkLunas(tagihanID, userID, models.MarkTagihanLunasRequest{}))
	_, err = db.Exec(`UPDATE santri SET nominal = 300000 WHERE id = ?`, santriID)
	require.NoError(t, err)
	require.NoError(t, q.SyncOpenTagihanNominalForSantri(context.Background(), santriID, 300000))
	item, err = service.Get(tagihanID)
	require.NoError(t, err)
	require.Equal(t, "lunas", item.Status)
	require.EqualValues(t, 250000, item.Nominal)
}

func TestTagihanFollowUpMultiTemplateCustomDanHistory(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "1x/pekan", 0)
	santriID := firstSantriID(t, db)
	_, err := db.Exec(`UPDATE santri SET no_wa = '081234567890', id_mahasantri = 'RS-TEST-001' WHERE id = ?`, santriID)
	require.NoError(t, err)
	generateMeetings(t, db, service, kelasID, santriID, 1, 4, "hadir")

	var tagihanID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM tagihan`).Scan(&tagihanID))
	userResult, err := db.Exec(`INSERT INTO users (email, name, role) VALUES ('fu-test@example.com', 'Petugas FU', 'keuangan')`)
	require.NoError(t, err)
	userID, err := userResult.LastInsertId()
	require.NoError(t, err)
	templates, err := service.ListTagihanTemplates()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(templates), 3)
	require.NotEqual(t, templates[0].Nama, templates[1].Nama)

	link, err := service.FollowUpURL(tagihanID, userID, models.FollowUpTagihanRequest{TemplateID: templates[0].ID})
	require.NoError(t, err)
	require.Contains(t, link, "https://wa.me/6281234567890?text=")
	logs, err := service.ListFollowUpLogs(tagihanID)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, templates[0].Nama, logs[0].TemplateNama)
	require.Equal(t, "Petugas FU", logs[0].PetugasNama)
	require.Contains(t, logs[0].MessageBody, "Ahmad")
	require.Contains(t, logs[0].MessageBody, "100.000")
	require.NotContains(t, logs[0].MessageBody, "{nama}")

	custom := "Pesan khusus hasil edit petugas"
	_, err = service.FollowUpURL(tagihanID, userID, models.FollowUpTagihanRequest{TemplateID: templates[1].ID, Message: custom})
	require.NoError(t, err)
	logs, err = service.ListFollowUpLogs(tagihanID)
	require.NoError(t, err)
	require.Len(t, logs, 2)
	require.Equal(t, templates[1].Nama, logs[0].TemplateNama)
	require.Equal(t, custom, logs[0].MessageBody)

	item, err := service.Get(tagihanID)
	require.NoError(t, err)
	require.EqualValues(t, 2, item.FuCount)
	require.NotEmpty(t, item.FuTerakhir)
}
