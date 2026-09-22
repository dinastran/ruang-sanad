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
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	_, err = db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri, created_at) VALUES ('T', '2026', 'Reguler', 'L', 'Dasar', ?, 'Senin', 1, 'Kelas Uji', 20, 1, CURRENT_TIMESTAMP)`, frekuensi)
	require.NoError(t, err)
	kelasID, err := lastID(db)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO santri (nama, nominal, frekuensi, kelas_id, status, pertemuan_awal) VALUES ('Ahmad', 100000, ?, ?, 'aktif', ?)`, frekuensi, kelasID, pertemuanAwal)
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

func TestGenerateTagihanRegulerDanIdempoten(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "1x/pekan", 0)
	pertemuan4 := createSelesaiPertemuan(t, db, kelasID, 4)
	require.NoError(t, service.GenerateForPertemuan(pertemuan4))
	require.Zero(t, tagihanCount(t, db))
	pertemuan8 := createSelesaiPertemuan(t, db, kelasID, 8)
	require.NoError(t, service.GenerateForPertemuan(pertemuan8))
	require.NoError(t, service.GenerateForPertemuan(pertemuan8))
	require.EqualValues(t, 1, tagihanCount(t, db))
	var bulanKe, nominal int64
	require.NoError(t, db.QueryRow(`SELECT bulan_ke, nominal FROM tagihan`).Scan(&bulanKe, &nominal))
	require.EqualValues(t, 2, bulanKe)
	require.EqualValues(t, 100000, nominal)
	_, err := db.Exec(`UPDATE santri SET nominal = 200000`)
	require.NoError(t, err)
	pertemuan12 := createSelesaiPertemuan(t, db, kelasID, 12)
	require.NoError(t, service.GenerateForPertemuan(pertemuan12))
	var snapshot int64
	require.NoError(t, db.QueryRow(`SELECT nominal FROM tagihan WHERE bulan_ke = 2`).Scan(&snapshot))
	require.EqualValues(t, 100000, snapshot)
}

func TestGenerateTagihanIntensifDanAnchorSantri(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "2x/pekan", 0)
	pertemuan8 := createSelesaiPertemuan(t, db, kelasID, 8)
	require.NoError(t, service.GenerateForPertemuan(pertemuan8))
	require.Zero(t, tagihanCount(t, db))
	pertemuan16 := createSelesaiPertemuan(t, db, kelasID, 16)
	require.NoError(t, service.GenerateForPertemuan(pertemuan16))
	require.EqualValues(t, 1, tagihanCount(t, db))

	dbAnchor, anchorService, anchorKelasID := setupTagihanService(t, "1x/pekan", 9)
	anchored8 := createSelesaiPertemuan(t, dbAnchor, anchorKelasID, 8)
	require.NoError(t, anchorService.GenerateForPertemuan(anchored8))
	require.Zero(t, tagihanCount(t, dbAnchor))
	anchored12 := createSelesaiPertemuan(t, dbAnchor, anchorKelasID, 12)
	require.NoError(t, anchorService.GenerateForPertemuan(anchored12))
	require.EqualValues(t, 1, tagihanCount(t, dbAnchor))
}

func TestBaselinePertemuanMempertahankanNomorTagihanRiil(t *testing.T) {
	db, service, kelasID := setupTagihanService(t, "1x/pekan", 0)
	_, err := db.Exec(`UPDATE kelas SET pertemuan_terakhir = 232 WHERE id = ?`, kelasID)
	require.NoError(t, err)

	next, err := queries.NewQuerier(db).GetNextPertemuanKe(context.Background(), kelasID)
	require.NoError(t, err)
	require.EqualValues(t, 233, next)

	pertemuan233 := createSelesaiPertemuan(t, db, kelasID, 233)
	require.NoError(t, service.GenerateForPertemuan(pertemuan233))
	require.Zero(t, tagihanCount(t, db))
	require.Error(t, NewKelasService(queries.NewQuerier(db)).SetPertemuanTerakhir(kelasID, 300))

	pertemuan236 := createSelesaiPertemuan(t, db, kelasID, 236)
	require.NoError(t, service.GenerateForPertemuan(pertemuan236))
	require.EqualValues(t, 1, tagihanCount(t, db))

	var bulanKe, pertemuanKe int64
	require.NoError(t, db.QueryRow(`SELECT bulan_ke, pertemuan_ke FROM tagihan`).Scan(&bulanKe, &pertemuanKe))
	require.EqualValues(t, 59, bulanKe)
	require.EqualValues(t, 236, pertemuanKe)
}

func TestTagihanKeepsBilledClassCohortAfterSantriMoves(t *testing.T) {
	db, service, oldKelasID := setupTagihanService(t, "1x/pekan", 0)
	_, err := db.Exec(`UPDATE santri SET angkatan_kelas = '2026'`)
	require.NoError(t, err)
	pertemuan8 := createSelesaiPertemuan(t, db, oldKelasID, 8)
	require.NoError(t, service.GenerateForPertemuan(pertemuan8))
	var tagihanID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM tagihan`).Scan(&tagihanID))

	_, err = db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas) VALUES ('NEW', '2027', 'Reguler', 'L', 'Dasar', '1x/pekan', 'Selasa', 1, 'Kelas Baru', 20)`)
	require.NoError(t, err)
	newKelasID, err := lastID(db)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE santri SET kelas_id = ?, angkatan_kelas = '2027'`, newKelasID)
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
	oldCohort, err = service.List(models.TagihanFilter{AngkatanKelas: "2026"})
	require.NoError(t, err)
	require.Len(t, oldCohort, 1)
	newCohort, err = service.List(models.TagihanFilter{AngkatanKelas: "2027"})
	require.NoError(t, err)
	require.Empty(t, newCohort)
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

func TestTagihanSantriPindahKeKelasTertinggalMelanjutkanBulan(t *testing.T) {
	db, service, asalID := setupTagihanService(t, "1x/pekan", 0)
	require.NoError(t, service.GenerateForPertemuan(createSelesaiPertemuan(t, db, asalID, 8)))
	require.NoError(t, service.GenerateForPertemuan(createSelesaiPertemuan(t, db, asalID, 12)))
	require.EqualValues(t, 2, tagihanCount(t, db))

	// Pindah ke kelas yang baru di pertemuan 1 (anchor = pertemuan berikutnya).
	_, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas) VALUES ('B', '2026', 'Reguler', 'L', 'Dasar', '1x/pekan', 'Selasa', 1, 'Kelas Tujuan', 20)`)
	require.NoError(t, err)
	tujuanID, err := lastID(db)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE santri SET kelas_id = ?, pertemuan_awal = 2`, tujuanID)
	require.NoError(t, err)

	var santriID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM santri`).Scan(&santriID))
	hadir := func(pertemuanID int64) {
		_, err := db.Exec(`INSERT INTO absensi (pertemuan_id, santri_id, status) VALUES (?, ?, 'hadir')`, pertemuanID, santriID)
		require.NoError(t, err)
	}

	p8 := createSelesaiPertemuan(t, db, tujuanID, 8)
	hadir(p8)
	require.NoError(t, service.GenerateForPertemuan(p8))
	require.NoError(t, service.GenerateForPertemuan(p8), "idempoten per pertemuan")
	p12 := createSelesaiPertemuan(t, db, tujuanID, 12)
	hadir(p12)
	require.NoError(t, service.GenerateForPertemuan(p12))
	// Pertemuan 16 tanpa absensi santri (mis. sedang cuti): Sync tidak boleh
	// menagihnya sebagai bulan lanjutan.
	createSelesaiPertemuan(t, db, tujuanID, 16)
	require.NoError(t, service.Sync())

	rows, err := db.Query(`SELECT bulan_ke FROM tagihan WHERE kelas_id = ? ORDER BY bulan_ke`, tujuanID)
	require.NoError(t, err)
	defer rows.Close()
	var bulan []int64
	for rows.Next() {
		var b int64
		require.NoError(t, rows.Scan(&b))
		bulan = append(bulan, b)
	}
	require.Equal(t, []int64{4, 5}, bulan, "tagihan di kelas tujuan melanjutkan bulan, tidak hilang")
}
