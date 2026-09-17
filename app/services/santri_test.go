package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupSantriDeleteService(t *testing.T) (*sql.DB, *SantriService) {
	db, querier := setupSantriDatabase(t)
	return db, NewSantriService(querier, nil)
}

func setupSantriDatabase(t *testing.T) (*sql.DB, *queries.Querier) {
	t.Helper()
	migrationsPath, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	require.NoError(t, err)
	t.Chdir(t.TempDir())

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, migrationsPath))

	var foreignKeys int
	require.NoError(t, db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys))
	require.Equal(t, 1, foreignKeys)
	return db, queries.NewQuerier(db)
}

func setupSantriIntegrityService(t *testing.T) (*sql.DB, *queries.Querier, *SantriService, int64) {
	t.Helper()
	db, querier := setupSantriDatabase(t)
	_, err := db.Exec(`INSERT INTO angkatan (kode, keterangan) VALUES ('AKA38', 'Angkatan 38'), ('AKA39', 'Angkatan 39')`)
	require.NoError(t, err)
	result, err := db.Exec(`INSERT INTO users (email, name, role) VALUES ('cs-test@example.com', 'CS Test', 'cs')`)
	require.NoError(t, err)
	userID, err := result.LastInsertId()
	require.NoError(t, err)
	service := NewSantriService(querier, NewKelasEngineService(querier))
	return db, querier, service, userID
}

func validCreateSantriRequest() models.CreateSantriRequest {
	return models.CreateSantriRequest{
		KelasKode:     "R 1X",
		Nama:          "Santri Uji",
		JenisKelamin:  "L",
		Nominal:       100000,
		TanggalDaftar: "2026-02-03",
		Angkatan:      "AKA38",
		Usia:          21,
		Domisili:      "Bandung",
	}
}

func validUpdateSantriRequest() models.UpdateSantriCSRequest {
	request := validCreateSantriRequest()
	return models.UpdateSantriCSRequest{
		KelasKode:     request.KelasKode,
		Nama:          request.Nama,
		JenisKelamin:  request.JenisKelamin,
		Nominal:       request.Nominal,
		TanggalDaftar: request.TanggalDaftar,
		Angkatan:      request.Angkatan,
		Usia:          request.Usia,
		Domisili:      request.Domisili,
	}
}

func TestSantriCreateValidatesRequiredIdentityMetadata(t *testing.T) {
	db, _, service, userID := setupSantriIntegrityService(t)
	tests := []struct {
		name      string
		mutate    func(*models.CreateSantriRequest)
		wantError string
	}{
		{name: "blank date", mutate: func(request *models.CreateSantriRequest) { request.TanggalDaftar = " " }, wantError: "tanggal daftar wajib diisi"},
		{name: "invalid date", mutate: func(request *models.CreateSantriRequest) { request.TanggalDaftar = "03/02/2026" }, wantError: "YYYY-MM-DD"},
		{name: "blank angkatan", mutate: func(request *models.CreateSantriRequest) { request.Angkatan = "" }, wantError: "angkatan wajib diisi"},
		{name: "unknown master code", mutate: func(request *models.CreateSantriRequest) { request.Angkatan = "AKA99" }, wantError: "tidak terdaftar"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := validCreateSantriRequest()
			test.mutate(&request)
			_, err := service.Create(request, userID)
			require.ErrorContains(t, err, test.wantError)
		})
	}
	requireTableCount(t, db, "santri", 0)
}

func TestSantriCreateIssuesDeterministicIDAndRollsBackIDFailure(t *testing.T) {
	t.Run("successful create", func(t *testing.T) {
		db, _, service, userID := setupSantriIntegrityService(t)
		response, err := service.Create(validCreateSantriRequest(), userID)
		require.NoError(t, err)
		require.Equal(t, "MHS.AKA38.0001.022026", response.IDMahasantri)
		require.False(t, response.IDMahasantriBermasalah)

		var stored string
		require.NoError(t, db.QueryRow(`SELECT id_mahasantri FROM santri WHERE id = ?`, response.ID).Scan(&stored))
		require.Equal(t, response.IDMahasantri, stored)
		problematic, err := service.CountIDMahasantriBermasalah()
		require.NoError(t, err)
		require.Zero(t, problematic)
	})

	t.Run("ID update failure rolls back insert", func(t *testing.T) {
		db, _, service, userID := setupSantriIntegrityService(t)
		_, err := db.Exec(`CREATE TRIGGER fail_santri_id BEFORE UPDATE OF id_mahasantri ON santri BEGIN SELECT RAISE(ABORT, 'ID generation failed'); END`)
		require.NoError(t, err)

		_, err = service.Create(validCreateSantriRequest(), userID)
		require.ErrorContains(t, err, "ID generation failed")
		requireTableCount(t, db, "santri", 0)
	})

	t.Run("engine failure rolls back insert and issued ID", func(t *testing.T) {
		db, _, service, userID := setupSantriIntegrityService(t)
		_, err := db.Exec(`CREATE TRIGGER fail_santri_engine BEFORE UPDATE OF tipe ON santri BEGIN SELECT RAISE(ABORT, 'engine failed'); END`)
		require.NoError(t, err)

		_, err = service.Create(validCreateSantriRequest(), userID)
		require.ErrorContains(t, err, "engine failed")
		requireTableCount(t, db, "santri", 0)
	})
}

func TestSantriUpdateByCSSynchronizesMissingOrMalformedIssuedIDs(t *testing.T) {
	tests := []struct {
		name       string
		existingID string
	}{
		{name: "empty", existingID: ""},
		{name: "malformed", existingID: "MHS..0001.022026"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, querier, service, _ := setupSantriIntegrityService(t)
			result, err := db.Exec(`INSERT INTO santri (id_mahasantri, nama, jenis_kelamin, angkatan, tanggal_daftar, status) VALUES (?, 'Legacy', 'L', 'AKA38', '2026-02-03', 'aktif')`, test.existingID)
			require.NoError(t, err)
			id, err := result.LastInsertId()
			require.NoError(t, err)

			require.NoError(t, service.UpdateByCS(id, validUpdateSantriRequest()))
			stored, err := querier.GetSantriByID(context.Background(), id)
			require.NoError(t, err)
			require.Equal(t, service.GenerateIDMahasantri("AKA38", id, stored.TanggalDaftar.Time), stored.IDMahasantri)
		})
	}
}

func TestNurulIssuedIdentityStaysImmutableWhileClassCohortBecomes40(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	_, err := db.Exec(`INSERT INTO angkatan (kode, keterangan) VALUES ('40', 'Angkatan 40')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO santri (id, id_mahasantri, kelas_kode, nama, jenis_kelamin, angkatan, tanggal_daftar, status) VALUES (317, 'MHS.39.0317.072026', 'R 1X', 'Nurul Aisyah', 'P', '39', NULL, 'aktif')`)
	require.NoError(t, err)

	request := validUpdateSantriRequest()
	request.Nama = "Nurul Aisyah"
	request.JenisKelamin = "P"
	request.Angkatan = "AKA38"
	request.TanggalDaftar = "2026-02-03"
	require.NoError(t, service.UpdateByCS(317, request))
	require.NoError(t, service.UpdateByAdminKelas(317, models.UpdateSantriAdminKelasRequest{AngkatanKelas: "40", Level: "01", Jadwal: "Senin"}))

	stored, err := querier.GetSantriByID(context.Background(), 317)
	require.NoError(t, err)
	require.Equal(t, "MHS.39.0317.072026", stored.IDMahasantri)
	require.Equal(t, "39", stored.Angkatan)
	require.False(t, stored.TanggalDaftar.Valid)
	require.Equal(t, "40", stored.AngkatanKelas)
	require.True(t, stored.KelasID.Valid)
	kelas, err := querier.GetKelasByID(context.Background(), stored.KelasID.Int64)
	require.NoError(t, err)
	require.Equal(t, "40", kelas.Angkatan)
	response, err := service.GetByID(317)
	require.NoError(t, err)
	require.False(t, response.IDMahasantriBermasalah)
}

func TestSantriUpdateByCSDoesNotRewriteAlreadyCorrectIssuedID(t *testing.T) {
	db, _, service, _ := setupSantriIntegrityService(t)
	result, err := db.Exec(`INSERT INTO santri (id_mahasantri, nama, jenis_kelamin, angkatan, tanggal_daftar, status) VALUES ('', 'Sudah Benar', 'L', 'AKA38', '2026-02-03', 'aktif')`)
	require.NoError(t, err)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	expected := service.GenerateIDMahasantri("AKA38", id, mustDate(t, "2026-02-03"))
	_, err = db.Exec(`UPDATE santri SET id_mahasantri = ? WHERE id = ?`, expected, id)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TRIGGER reject_redundant_id_update BEFORE UPDATE OF id_mahasantri ON santri BEGIN SELECT RAISE(ABORT, 'ID should not be rewritten'); END`)
	require.NoError(t, err)

	request := validUpdateSantriRequest()
	request.Nama = "Sudah Benar"
	require.NoError(t, service.UpdateByCS(id, request))
	response, err := service.GetByID(id)
	require.NoError(t, err)
	require.Equal(t, expected, response.IDMahasantri)
	require.False(t, response.IDMahasantriBermasalah)
}

func TestSantriUpdateByCSRetainsUnchangedClassAtCapacity(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	kunciKelas := service.engine.BuatKunciKelas("R 1X", "L", "01", "1x/pekan", "Senin", "AKA38")
	result, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri) VALUES (?, 'AKA38', 'Reguler', 'L', '01', '1x/pekan', 'Senin', 1, 'Kelas Penuh', 10, 10)`, kunciKelas)
	require.NoError(t, err)
	kelasID, err := result.LastInsertId()
	require.NoError(t, err)
	result, err = db.Exec(`INSERT INTO santri (id_mahasantri, kelas_kode, nama, jenis_kelamin, nominal, angkatan, angkatan_kelas, tanggal_daftar, level, jadwal, kelas_id, status) VALUES ('', 'R 1X', 'Santri Tetap', 'L', 100000, 'AKA38', 'AKA38', '2026-02-03', '01', 'Senin', ?, 'aktif')`, kelasID)
	require.NoError(t, err)
	santriID, err := result.LastInsertId()
	require.NoError(t, err)
	for i := 0; i < 9; i++ {
		_, err = db.Exec(`INSERT INTO santri (nama, jenis_kelamin, kelas_id, status) VALUES (?, 'L', ?, 'aktif')`, fmt.Sprintf("Pengisi %d", i+1), kelasID)
		require.NoError(t, err)
	}

	request := validUpdateSantriRequest()
	request.Nama = "Santri Tetap"
	require.NoError(t, service.UpdateByCS(santriID, request))

	stored, err := querier.GetSantriByID(context.Background(), santriID)
	require.NoError(t, err)
	require.True(t, stored.KelasID.Valid)
	require.Equal(t, kelasID, stored.KelasID.Int64)
	requireTableCount(t, db, "kelas", 1)
	kelas, err := querier.GetKelasByID(context.Background(), kelasID)
	require.NoError(t, err)
	require.EqualValues(t, 10, kelas.JumlahSantri)
	var storedCount int64
	require.NoError(t, db.QueryRow(`SELECT jumlah_santri FROM kelas WHERE id = ?`, kelasID).Scan(&storedCount))
	require.EqualValues(t, 10, storedCount)
}

func TestSantriUpdateByAdminKelasDecrementFailureRollsBackMove(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	kunciKelas := service.engine.BuatKunciKelas("R 1X", "L", "01", "1x/pekan", "Senin", "AKA38")
	result, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri) VALUES (?, 'AKA38', 'Reguler', 'L', '01', '1x/pekan', 'Senin', 1, 'Kelas Lama', 10, 1)`, kunciKelas)
	require.NoError(t, err)
	oldKelasID, err := result.LastInsertId()
	require.NoError(t, err)
	result, err = db.Exec(`INSERT INTO santri (id_mahasantri, kelas_kode, nama, jenis_kelamin, nominal, angkatan, angkatan_kelas, tanggal_daftar, level, jadwal, kelas_id, status) VALUES ('', 'R 1X', 'Nama Lama', 'L', 50000, 'AKA38', 'AKA38', '2026-01-02', '01', 'Senin', ?, 'aktif')`, oldKelasID)
	require.NoError(t, err)
	santriID, err := result.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TRIGGER fail_decrement BEFORE UPDATE OF jumlah_santri ON kelas WHEN NEW.jumlah_santri < OLD.jumlah_santri BEGIN SELECT RAISE(ABORT, 'decrement failed'); END`)
	require.NoError(t, err)

	require.ErrorContains(t, service.UpdateByAdminKelas(santriID, models.UpdateSantriAdminKelasRequest{AngkatanKelas: "AKA39", Level: "01", Jadwal: "Senin"}), "decrement kelas lama")

	stored, err := querier.GetSantriByID(context.Background(), santriID)
	require.NoError(t, err)
	require.Equal(t, "AKA38", stored.AngkatanKelas)
	require.Equal(t, "AKA38", stored.Angkatan)
	require.Equal(t, "", stored.IDMahasantri)
	require.True(t, stored.KelasID.Valid)
	require.Equal(t, oldKelasID, stored.KelasID.Int64)
	requireTableCount(t, db, "kelas", 1)
	oldKelas, err := querier.GetKelasByID(context.Background(), oldKelasID)
	require.NoError(t, err)
	require.EqualValues(t, 1, oldKelas.JumlahSantri)
}

func TestSantriUpdateByAdminKelasMoveRetainsOldClassAndHistory(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	kunciKelas := service.engine.BuatKunciKelas("R 1X", "L", "01", "1x/pekan", "Senin", "AKA38")
	result, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri) VALUES (?, 'AKA38', 'Reguler', 'L', '01', '1x/pekan', 'Senin', 1, 'Kelas Bersejarah', 10, 1)`, kunciKelas)
	require.NoError(t, err)
	oldKelasID, err := result.LastInsertId()
	require.NoError(t, err)
	result, err = db.Exec(`INSERT INTO santri (id_mahasantri, kelas_kode, nama, jenis_kelamin, nominal, angkatan, angkatan_kelas, tanggal_daftar, level, jadwal, kelas_id, status) VALUES ('', 'R 1X', 'Santri Pindah', 'L', 100000, 'AKA38', 'AKA38', '2026-02-03', '01', 'Senin', ?, 'aktif')`, oldKelasID)
	require.NoError(t, err)
	santriID, err := result.LastInsertId()
	require.NoError(t, err)
	result, err = db.Exec(`INSERT INTO pertemuan (kelas_id, pertemuan_ke, tanggal, status) VALUES (?, 1, '2026-02-10', 'selesai')`, oldKelasID)
	require.NoError(t, err)
	pertemuanID, err := result.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO absensi (pertemuan_id, santri_id, status) VALUES (?, ?, 'hadir')`, pertemuanID, santriID)
	require.NoError(t, err)

	require.NoError(t, service.UpdateByAdminKelas(santriID, models.UpdateSantriAdminKelasRequest{AngkatanKelas: "AKA39", Level: "01", Jadwal: "Senin"}))

	stored, err := querier.GetSantriByID(context.Background(), santriID)
	require.NoError(t, err)
	require.True(t, stored.KelasID.Valid)
	require.Equal(t, "AKA39", stored.AngkatanKelas)
	require.NotEqual(t, oldKelasID, stored.KelasID.Int64)
	requireTableCount(t, db, "kelas", 2)
	oldKelas, err := querier.GetKelasByID(context.Background(), oldKelasID)
	require.NoError(t, err)
	require.Zero(t, oldKelas.JumlahSantri)
	requireTableCount(t, db, "pertemuan", 1)
	requireTableCount(t, db, "absensi", 1)
	var historyKelasID int64
	require.NoError(t, db.QueryRow(`SELECT kelas_id FROM pertemuan WHERE id = ?`, pertemuanID).Scan(&historyKelasID))
	require.Equal(t, oldKelasID, historyKelasID)
}

func TestSantriPindahkanKelasSameFullClassIsNoOp(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	result, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri) VALUES ('MANUAL-SAME', 'AKA38', 'Reguler', 'L', '01', '1x/pekan', 'Senin', 1, 'Kelas Penuh', 10, 10)`)
	require.NoError(t, err)
	kelasID, err := result.LastInsertId()
	require.NoError(t, err)
	result, err = db.Exec(`INSERT INTO santri (nama, jenis_kelamin, kelas_id, status) VALUES ('Santri Sama', 'L', ?, 'aktif')`, kelasID)
	require.NoError(t, err)
	santriID, err := result.LastInsertId()
	require.NoError(t, err)
	for i := 0; i < 9; i++ {
		_, err = db.Exec(`INSERT INTO santri (nama, jenis_kelamin, kelas_id, status) VALUES (?, 'L', ?, 'aktif')`, fmt.Sprintf("Penuh %d", i+1), kelasID)
		require.NoError(t, err)
	}

	require.NoError(t, service.PindahkanKelas(santriID, kelasID))
	stored, err := querier.GetSantriByID(context.Background(), santriID)
	require.NoError(t, err)
	require.Equal(t, kelasID, stored.KelasID.Int64)
	var storedCount int64
	require.NoError(t, db.QueryRow(`SELECT jumlah_santri FROM kelas WHERE id = ?`, kelasID).Scan(&storedCount))
	require.EqualValues(t, 10, storedCount)
}

func TestSantriPindahkanKelasSyncsClassCohortAndPreservesIdentity(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	sourceKey := service.engine.BuatKunciKelas("R 1X", "P", "01", "1x/pekan", "Senin", "39")
	targetKey := service.engine.BuatKunciKelas("P 2X", "P", "02", "2x/pekan", "Selasa", "40")
	result, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri) VALUES (?, '39', 'Reguler', 'P', '01', '1x/pekan', 'Senin', 1, 'Kelas 39', 10, 1)`, sourceKey)
	require.NoError(t, err)
	sourceID, err := result.LastInsertId()
	require.NoError(t, err)
	result, err = db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri) VALUES (?, '40', 'Private', 'P', '02', '2x/pekan', 'Selasa', 1, 'Kelas 40', 10, 0)`, targetKey)
	require.NoError(t, err)
	targetID, err := result.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO santri (id, id_mahasantri, kelas_kode, nama, jenis_kelamin, angkatan, angkatan_kelas, tanggal_daftar, level, jadwal, tipe, frekuensi, kelas_id, status) VALUES (317, 'MHS.39.0317.072026', 'R 1X', 'Nurul Aisyah', 'P', '39', '39', NULL, '01', 'Senin', 'Reguler', '1x/pekan', ?, 'aktif')`, sourceID)
	require.NoError(t, err)

	require.NoError(t, service.PindahkanKelas(317, targetID))
	stored, err := querier.GetSantriByID(context.Background(), 317)
	require.NoError(t, err)
	require.Equal(t, "40", stored.AngkatanKelas)
	require.Equal(t, "P 2X", stored.KelasKode)
	require.Equal(t, "02", stored.Level)
	require.Equal(t, "Selasa", stored.Jadwal)
	require.Equal(t, "Private", stored.Tipe)
	require.Equal(t, "2x/pekan", stored.Frekuensi)
	require.Equal(t, "39", stored.Angkatan)
	require.Equal(t, "MHS.39.0317.072026", stored.IDMahasantri)
	require.False(t, stored.TanggalDaftar.Valid)
	require.NoError(t, service.engine.ProcessSantri(context.Background(), &stored))
	restored, err := querier.GetSantriByID(context.Background(), 317)
	require.NoError(t, err)
	require.Equal(t, targetID, restored.KelasID.Int64)
	requireTableCount(t, db, "kelas", 2)
}

func TestSantriPindahkanKelasRejectsInvalidDestinations(t *testing.T) {
	tests := []struct {
		name        string
		active      int64
		gender      string
		capacity    int64
		occupants   int
		wantMessage string
	}{
		{name: "inactive", active: 0, gender: "L", capacity: 10, wantMessage: "tidak aktif"},
		{name: "gender mismatch", active: 1, gender: "P", capacity: 10, wantMessage: "jenis kelamin"},
		{name: "configured capacity", active: 1, gender: "L", capacity: 2, occupants: 2, wantMessage: "kapasitas 2"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, querier, service, _ := setupSantriIntegrityService(t)
			result, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri) VALUES ('SOURCE', 'AKA38', 'Reguler', 'L', '01', '1x/pekan', 'Senin', 1, 'Sumber', 10, 1)`)
			require.NoError(t, err)
			sourceID, err := result.LastInsertId()
			require.NoError(t, err)
			result, err = db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri, is_aktif) VALUES ('TARGET', 'AKA39', 'Reguler', ?, '02', '2x/pekan', 'Selasa', 1, 'Tujuan', ?, ?, ?)`, test.gender, test.capacity, test.occupants, test.active)
			require.NoError(t, err)
			targetID, err := result.LastInsertId()
			require.NoError(t, err)
			for i := 0; i < test.occupants; i++ {
				_, err = db.Exec(`INSERT INTO santri (nama, jenis_kelamin, kelas_id, status) VALUES (?, ?, ?, 'aktif')`, fmt.Sprintf("Pengisi %d", i+1), test.gender, targetID)
				require.NoError(t, err)
			}
			result, err = db.Exec(`INSERT INTO santri (nama, jenis_kelamin, angkatan_kelas, kelas_id, status) VALUES ('Santri', 'L', 'AKA38', ?, 'aktif')`, sourceID)
			require.NoError(t, err)
			santriID, err := result.LastInsertId()
			require.NoError(t, err)

			require.ErrorContains(t, service.PindahkanKelas(santriID, targetID), test.wantMessage)
			stored, err := querier.GetSantriByID(context.Background(), santriID)
			require.NoError(t, err)
			require.Equal(t, sourceID, stored.KelasID.Int64)
			require.Equal(t, "AKA38", stored.AngkatanKelas)
		})
	}
}

func TestKelasEngineUsesConfiguredCapacityAndSkipsInactiveClasses(t *testing.T) {
	tests := []struct {
		name         string
		active       int64
		capacity     int64
		wantSubIndex int64
	}{
		{name: "configured capacity", active: 1, capacity: 1, wantSubIndex: 2},
		{name: "inactive class", active: 0, capacity: 10, wantSubIndex: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, querier, service, _ := setupSantriIntegrityService(t)
			key := service.engine.BuatKunciKelas("R 1X", "L", "01", "1x/pekan", "Senin", "AKA38")
			_, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri, is_aktif) VALUES (?, 'AKA38', 'Reguler', 'L', '01', '1x/pekan', 'Senin', 1, 'Lama', ?, 1, ?)`, key, test.capacity, test.active)
			require.NoError(t, err)
			var existingKelasID int64
			require.NoError(t, db.QueryRow(`SELECT id FROM kelas WHERE kunci_kelas = ? AND sub_index = 1`, key).Scan(&existingKelasID))
			_, err = db.Exec(`INSERT INTO santri (nama, jenis_kelamin, kelas_id, status) VALUES ('Pengisi', 'L', ?, 'aktif')`, existingKelasID)
			require.NoError(t, err)
			result, err := db.Exec(`INSERT INTO santri (kelas_kode, nama, jenis_kelamin, angkatan_kelas, level, jadwal, status) VALUES ('R 1X', 'Baru', 'L', 'AKA38', '01', 'Senin', 'aktif')`)
			require.NoError(t, err)
			id, err := result.LastInsertId()
			require.NoError(t, err)
			st, err := querier.GetSantriByID(context.Background(), id)
			require.NoError(t, err)
			require.NoError(t, service.engine.ProcessSantri(context.Background(), &st))
			stored, err := querier.GetSantriByID(context.Background(), id)
			require.NoError(t, err)
			assigned, err := querier.GetKelasByID(context.Background(), stored.KelasID.Int64)
			require.NoError(t, err)
			require.Equal(t, test.wantSubIndex, assigned.SubIndex)
			require.EqualValues(t, 1, assigned.IsAktif)
		})
	}
}

func TestKelasEngineCreatesNewClassAfterFifteenSantri(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	result, err := db.Exec(`INSERT INTO santri (kelas_kode, nama, jenis_kelamin, angkatan_kelas, level, jadwal, status) VALUES ('R 1X', 'Santri Pertama', 'L', 'AKA38', '01', 'Senin', 'aktif')`)
	require.NoError(t, err)
	firstID, err := result.LastInsertId()
	require.NoError(t, err)

	first, err := querier.GetSantriByID(context.Background(), firstID)
	require.NoError(t, err)
	require.NoError(t, service.engine.ProcessSantri(context.Background(), &first))
	first, err = querier.GetSantriByID(context.Background(), firstID)
	require.NoError(t, err)
	require.True(t, first.KelasID.Valid)

	for i := 0; i < 14; i++ {
		_, err = db.Exec(`INSERT INTO santri (nama, jenis_kelamin, kelas_id, status) VALUES (?, 'L', ?, 'aktif')`, fmt.Sprintf("Pengisi %d", i+1), first.KelasID.Int64)
		require.NoError(t, err)
	}

	result, err = db.Exec(`INSERT INTO santri (kelas_kode, nama, jenis_kelamin, angkatan_kelas, level, jadwal, status) VALUES ('R 1X', 'Santri Keenam Belas', 'L', 'AKA38', '01', 'Senin', 'aktif')`)
	require.NoError(t, err)
	secondID, err := result.LastInsertId()
	require.NoError(t, err)
	second, err := querier.GetSantriByID(context.Background(), secondID)
	require.NoError(t, err)
	require.NoError(t, service.engine.ProcessSantri(context.Background(), &second))

	firstKelas, err := querier.GetKelasByID(context.Background(), first.KelasID.Int64)
	require.NoError(t, err)
	require.EqualValues(t, 15, firstKelas.Kapasitas)
	require.EqualValues(t, 15, firstKelas.JumlahSantri)
	second, err = querier.GetSantriByID(context.Background(), secondID)
	require.NoError(t, err)
	secondKelas, err := querier.GetKelasByID(context.Background(), second.KelasID.Int64)
	require.NoError(t, err)
	require.EqualValues(t, 2, secondKelas.SubIndex)
	require.EqualValues(t, 15, secondKelas.Kapasitas)
	require.EqualValues(t, 1, secondKelas.JumlahSantri)
}

func TestSantriUpdateByAdminKelasRetainsUnchangedLegacyCohort(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	result, err := db.Exec(`INSERT INTO santri (kelas_kode, nama, jenis_kelamin, angkatan_kelas, status) VALUES ('R 1X', 'Legacy', 'L', 'LEGACY-37', 'aktif')`)
	require.NoError(t, err)
	id, err := result.LastInsertId()
	require.NoError(t, err)

	require.NoError(t, service.UpdateByAdminKelas(id, models.UpdateSantriAdminKelasRequest{AngkatanKelas: "LEGACY-37", Level: "01", Jadwal: "Senin"}))
	stored, err := querier.GetSantriByID(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "LEGACY-37", stored.AngkatanKelas)
	require.ErrorContains(t, service.UpdateByAdminKelas(id, models.UpdateSantriAdminKelasRequest{AngkatanKelas: "LEGACY-36", Level: "01", Jadwal: "Senin"}), "tidak terdaftar")
}

func TestSantriPindahkanKelasIncrementFailureRollsBackEverything(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	result, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri) VALUES ('SOURCE', 'AKA38', 'Reguler', 'L', '01', '1x/pekan', 'Senin', 1, 'Kelas Sumber', 10, 1)`)
	require.NoError(t, err)
	sourceID, err := result.LastInsertId()
	require.NoError(t, err)
	result, err = db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri) VALUES ('TARGET', 'AKA38', 'Reguler', 'L', '01', '1x/pekan', 'Selasa', 1, 'Kelas Tujuan', 10, 0)`)
	require.NoError(t, err)
	targetID, err := result.LastInsertId()
	require.NoError(t, err)
	result, err = db.Exec(`INSERT INTO santri (nama, jenis_kelamin, kelas_id, pertemuan_awal, status) VALUES ('Santri Manual', 'L', ?, 7, 'aktif')`, sourceID)
	require.NoError(t, err)
	santriID, err := result.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO pertemuan (kelas_id, pertemuan_ke, tanggal, status) VALUES (?, 1, '2026-02-10', 'selesai')`, targetID)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TRIGGER fail_increment BEFORE UPDATE OF jumlah_santri ON kelas WHEN NEW.jumlah_santri > OLD.jumlah_santri BEGIN SELECT RAISE(ABORT, 'increment failed'); END`)
	require.NoError(t, err)

	require.ErrorContains(t, service.PindahkanKelas(santriID, targetID), "increment failed")
	stored, err := querier.GetSantriByID(context.Background(), santriID)
	require.NoError(t, err)
	require.True(t, stored.KelasID.Valid)
	require.Equal(t, sourceID, stored.KelasID.Int64)
	require.EqualValues(t, 7, stored.PertemuanAwal)
	var sourceCount, targetCount int64
	require.NoError(t, db.QueryRow(`SELECT jumlah_santri FROM kelas WHERE id = ?`, sourceID).Scan(&sourceCount))
	require.NoError(t, db.QueryRow(`SELECT jumlah_santri FROM kelas WHERE id = ?`, targetID).Scan(&targetCount))
	require.EqualValues(t, 1, sourceCount)
	require.Zero(t, targetCount)
}

func TestSantriUpdateByCSEngineFailureRollsBackMetadataAndIDRepair(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	result, err := db.Exec(`INSERT INTO santri (id_mahasantri, kelas_kode, nama, jenis_kelamin, nominal, angkatan, tanggal_daftar, level, jadwal, status) VALUES ('', 'OLD', 'Nama Lama', 'L', 50000, 'AKA38', '2026-01-02', '01', 'Senin', 'aktif')`)
	require.NoError(t, err)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TRIGGER fail_santri_engine BEFORE UPDATE OF tipe ON santri BEGIN SELECT RAISE(ABORT, 'engine failed'); END`)
	require.NoError(t, err)

	request := validUpdateSantriRequest()
	request.Nama = "Nama Baru"
	request.Angkatan = "AKA39"
	request.TanggalDaftar = "2026-04-20"
	require.ErrorContains(t, service.UpdateByCS(id, request), "engine failed")

	stored, err := querier.GetSantriByID(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "Nama Lama", stored.Nama)
	require.Equal(t, "AKA38", stored.Angkatan)
	require.Equal(t, "", stored.IDMahasantri)
	require.Equal(t, "2026-01-02", stored.TanggalDaftar.Time.Format(time.DateOnly))
	requireTableCount(t, db, "kelas", 0)
}

func TestValidIssuedIDMahasantri(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{name: "standard", id: "MHS.AKA38.0001.012026", valid: true},
		{name: "dotted master code", id: "MHS.AKA.38.0001.122026", valid: true},
		{name: "invalid zero month", id: "MHS.AKA38.0001.002026", valid: false},
		{name: "invalid month thirteen", id: "MHS.AKA38.0001.132026", valid: false},
		{name: "outer whitespace", id: " MHS.AKA38.0001.012026", valid: false},
		{name: "code whitespace", id: "MHS. AKA38.0001.012026", valid: false},
		{name: "short sequence", id: "MHS.AKA38.001.012026", valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.valid, isValidIssuedIDMahasantri(test.id))
		})
	}
}

func TestSantriIDReviewDetectionAndFilter(t *testing.T) {
	db, _, service, _ := setupSantriIntegrityService(t)
	insert := func(idMahasantri, angkatan, tanggal string) int64 {
		t.Helper()
		result, err := db.Exec(`INSERT INTO santri (id_mahasantri, nama, jenis_kelamin, angkatan, tanggal_daftar, status) VALUES (?, 'Inventaris', 'P', ?, NULLIF(?, ''), 'aktif')`, idMahasantri, angkatan, tanggal)
		require.NoError(t, err)
		id, err := result.LastInsertId()
		require.NoError(t, err)
		return id
	}

	goodID := insert("", "AKA38", "2026-02-03")
	_, err := db.Exec(`UPDATE santri SET id_mahasantri = ? WHERE id = ?`, service.GenerateIDMahasantri("AKA38", goodID, mustDate(t, "2026-02-03")), goodID)
	require.NoError(t, err)
	insert("", "", "")
	insert("MHS..0003.022026", "AKA38", "2026-02-03")
	mismatchID := insert("MHS.AKA38.9999.022026", "AKA38", "2026-02-03")
	_, err = db.Exec(`UPDATE santri SET status = 'tidak_lanjut' WHERE id = ?`, mismatchID)
	require.NoError(t, err)
	unknownMasterID := insert("", "AKA404", "2026-02-03")
	_, err = db.Exec(`UPDATE santri SET id_mahasantri = ? WHERE id = ?`, service.GenerateIDMahasantri("AKA404", unknownMasterID, mustDate(t, "2026-02-03")), unknownMasterID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE santri SET tanggal_daftar = NULL WHERE id = ?`, unknownMasterID)
	require.NoError(t, err)
	asciiWhitespaceID := insert("", "AKA38", "2026-02-03")
	_, err = db.Exec(`UPDATE santri SET id_mahasantri = ? WHERE id = ?`, strings.Replace(service.GenerateIDMahasantri("AKA38", asciiWhitespaceID, mustDate(t, "2026-02-03")), "MHS.", "MHS.\t", 1), asciiWhitespaceID)
	require.NoError(t, err)

	result, err := service.List(models.SantriListParams{Status: "", Lengkap: -1, IDBermasalah: 1, Limit: 100})
	require.NoError(t, err)
	require.EqualValues(t, 4, result.Total)
	require.Len(t, result.Data, 4)
	for _, santri := range result.Data {
		require.True(t, santri.IDMahasantriBermasalah)
	}

	good, err := service.GetByID(goodID)
	require.NoError(t, err)
	require.False(t, good.IDMahasantriBermasalah)
	mismatch, err := service.GetByID(mismatchID)
	require.NoError(t, err)
	require.True(t, mismatch.IDMahasantriBermasalah)
	require.Equal(t, "tidak_lanjut", mismatch.Status)
	unknownMaster, err := service.GetByID(unknownMasterID)
	require.NoError(t, err)
	require.False(t, unknownMaster.IDMahasantriBermasalah)
	asciiWhitespace, err := service.GetByID(asciiWhitespaceID)
	require.NoError(t, err)
	require.True(t, asciiWhitespace.IDMahasantriBermasalah)
	count, err := service.CountIDMahasantriBermasalah()
	require.NoError(t, err)
	require.EqualValues(t, 4, count)
	goodResponse, err := service.GetByID(goodID)
	require.NoError(t, err)
	insert(goodResponse.IDMahasantri, "AKA38", "")
	count, err = service.CountIDMahasantriBermasalah()
	require.NoError(t, err)
	require.EqualValues(t, 6, count)
	good, err = service.GetByID(goodID)
	require.NoError(t, err)
	require.True(t, good.IDMahasantriBermasalah)
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.DateOnly, value)
	require.NoError(t, err)
	return parsed
}

func TestImportCSVUsesSantriValidationWithoutPartialRows(t *testing.T) {
	db, querier, service, userID := setupSantriIntegrityService(t)
	importService := NewImportService(querier, service)
	csvData := strings.Join([]string{
		"kelas_kode,nama,no_whatsapp,email,jenis_kelamin,nominal,tanggal_daftar,angkatan,usia,domisili",
		"R 1X,,0812,a@example.com,L,100000,2026-02-03,AKA38,20,Bandung",
		"R 1X,Tanpa Tanggal,0812,b@example.com,L,100000,,AKA38,20,Bandung",
		"R 1X,Tanpa Angkatan,0812,c@example.com,L,100000,2026-02-03,,20,Bandung",
		"R 1X,Kode Salah,0812,d@example.com,L,100000,2026-02-03,AKA99,20,Bandung",
		"R 1X,Tanggal Salah,0812,e@example.com,L,100000,03/02/2026,AKA38,20,Bandung",
	}, "\n")

	result, err := importService.ProcessCSV(strings.NewReader(csvData), userID, "invalid.csv")
	require.NoError(t, err)
	require.Equal(t, 5, result.Gagal)
	require.Zero(t, result.Berhasil)
	require.Contains(t, result.Catatan, "Baris 2: nama wajib diisi")
	require.Contains(t, result.Catatan, "Baris 3: tanggal daftar wajib diisi")
	require.Contains(t, result.Catatan, "Baris 4: angkatan wajib diisi")
	require.Contains(t, result.Catatan, "Baris 5: angkatan \"AKA99\" tidak terdaftar")
	require.Contains(t, result.Catatan, "Baris 6: tanggal daftar harus berformat YYYY-MM-DD")
	requireTableCount(t, db, "santri", 0)
}

func TestImportCSVSucceedsForEightAndTenColumnLayouts(t *testing.T) {
	db, querier, service, userID := setupSantriIntegrityService(t)
	importService := NewImportService(querier, service)

	eightColumns := "kelas_kode,nama,jenis_kelamin,nominal,tanggal_daftar,angkatan,usia,domisili\nR 1X,Delapan Kolom,L,100000,2026-02-03,AKA38,20,Bandung\n"
	result, err := importService.ProcessCSV(strings.NewReader(eightColumns), userID, "eight.csv")
	require.NoError(t, err)
	require.Equal(t, 1, result.Berhasil)
	require.Zero(t, result.Gagal)

	tenColumns := "\ufeffkelas_kode,nama,no_whatsapp,email,jenis_kelamin,nominal,tanggal_daftar,angkatan,usia,domisili\nR 1X,Sepuluh Kolom,0812,ten@example.com,P,120000,2026-03-04,AKA39,22,Jakarta\n"
	result, err = importService.ProcessCSV(strings.NewReader(tenColumns), userID, "ten.csv")
	require.NoError(t, err)
	require.Equal(t, 1, result.Berhasil)
	require.Zero(t, result.Gagal)
	requireTableCount(t, db, "santri", 2)

	rows, err := db.Query(`SELECT id_mahasantri FROM santri ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"MHS.AKA38.0001.022026", "MHS.AKA39.0002.032026"}, ids)
	var assignedClassCohorts int64
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM santri WHERE angkatan_kelas != ''`).Scan(&assignedClassCohorts))
	require.Zero(t, assignedClassCohorts)
}

func TestSuperadminCorrectionReissuesRegistrationIdentity(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	_, err := db.Exec(`INSERT INTO santri (id, id_mahasantri, nama, jenis_kelamin, angkatan, tanggal_daftar, status) VALUES (317, 'MHS.39.0317.072026', 'Nurul Aisyah', 'P', '39', NULL, 'aktif')`)
	require.NoError(t, err)
	_, err = service.CorrectRegistrationIdentity(317, models.CorrectSantriRegistrationRequest{Angkatan: "AKA38", TanggalDaftar: "2026-02-03"})
	require.ErrorContains(t, err, "konfirmasi")
	corrected, err := service.CorrectRegistrationIdentity(317, models.CorrectSantriRegistrationRequest{Angkatan: "AKA38", TanggalDaftar: "2026-02-03", Konfirmasi: true})
	require.NoError(t, err)
	require.Equal(t, "MHS.AKA38.0317.022026", corrected.IDMahasantri)
	stored, err := querier.GetSantriByID(context.Background(), 317)
	require.NoError(t, err)
	require.Equal(t, "AKA38", stored.Angkatan)
	require.Equal(t, "2026-02-03", stored.TanggalDaftar.Time.Format(time.DateOnly))
}

func TestMigration0026BackfillsSeparateCohortsAndKeepsUnknownDateNull(t *testing.T) {
	migrationsPath, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	require.NoError(t, err)
	t.Chdir(t.TempDir())
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.UpTo(db, migrationsPath, 25))
	_, err = db.Exec(`INSERT INTO angkatan (kode, keterangan) VALUES ('MASTER', 'Master Placement')`)
	require.NoError(t, err)
	result, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, nama_kelas) VALUES ('K40', '40', 'Reguler', 'P', '01', '1x/pekan', 'Kelas 40')`)
	require.NoError(t, err)
	kelasID, err := result.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO santri (id, id_mahasantri, nama, angkatan, tanggal_daftar, kelas_id) VALUES
		(317, 'MHS.39.0317.072026', 'Nurul Assigned', '40', '2026-08-10', ?),
		(318, 'MHS.AKA.39.0318.072026', 'Dotted Unassigned', 'legacy', NULL, NULL),
		(319, 'MHS.39.0319.072026', 'Matching Exact Date', 'legacy', '2026-07-15', NULL),
		(320, 'MHS.REG.0320.072026', 'Master Valid Placement', 'MASTER', '2026-07-15 23:59:59', NULL),
		(321, 'MHS.' || char(9) || 'BAD.0321.072026', 'Whitespace Cohort', 'legacy', '2026-07-15', NULL)`, kelasID)
	require.NoError(t, err)
	require.NoError(t, goose.UpTo(db, migrationsPath, 26))
	var registration, class string
	var date sql.NullString
	require.NoError(t, db.QueryRow(`SELECT angkatan, angkatan_kelas, CAST(tanggal_daftar AS TEXT) FROM santri WHERE id = 317`).Scan(&registration, &class, &date))
	require.Equal(t, "39", registration)
	require.Equal(t, "40", class)
	require.False(t, date.Valid)
	require.NoError(t, db.QueryRow(`SELECT angkatan, angkatan_kelas, CAST(tanggal_daftar AS TEXT) FROM santri WHERE id = 318`).Scan(&registration, &class, &date))
	require.Equal(t, "AKA.39", registration)
	require.Empty(t, class)
	require.False(t, date.Valid)
	require.NoError(t, db.QueryRow(`SELECT angkatan, angkatan_kelas, CAST(tanggal_daftar AS TEXT) FROM santri WHERE id = 319`).Scan(&registration, &class, &date))
	require.Equal(t, "39", registration)
	require.Empty(t, class)
	require.Equal(t, sql.NullString{String: "2026-07-15", Valid: true}, date)
	require.NoError(t, db.QueryRow(`SELECT angkatan, angkatan_kelas, CAST(tanggal_daftar AS TEXT) FROM santri WHERE id = 320`).Scan(&registration, &class, &date))
	require.Equal(t, "REG", registration)
	require.Equal(t, "MASTER", class)
	require.Equal(t, sql.NullString{String: "2026-07-15 23:59:59", Valid: true}, date)
	require.NoError(t, db.QueryRow(`SELECT angkatan, angkatan_kelas, CAST(tanggal_daftar AS TEXT) FROM santri WHERE id = 321`).Scan(&registration, &class, &date))
	require.Equal(t, "legacy", registration)
	require.Empty(t, class)
	require.Equal(t, sql.NullString{String: "2026-07-15", Valid: true}, date)
	require.NoError(t, goose.DownTo(db, migrationsPath, 25))
	_, err = db.Exec(`SELECT angkatan_kelas FROM santri LIMIT 1`)
	require.Error(t, err)
}

func TestMigration0028IncreasesDefaultCapacityWithoutChangingCustomCapacity(t *testing.T) {
	migrationsPath, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	require.NoError(t, err)
	t.Chdir(t.TempDir())
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.UpTo(db, migrationsPath, 27))
	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)

	for _, kelas := range []struct {
		key      string
		capacity int
	}{
		{key: "DEFAULT-10", capacity: 10},
		{key: "CUSTOM-2", capacity: 2},
		{key: "CUSTOM-20", capacity: 20},
	} {
		_, err = db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, nama_kelas, kapasitas) VALUES (?, 'AKA38', 'Reguler', 'L', '01', '1x/pekan', ?, ?)`, kelas.key, kelas.key, kelas.capacity)
		require.NoError(t, err)
	}
	var kelasID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM kelas WHERE kunci_kelas = 'DEFAULT-10'`).Scan(&kelasID))
	_, err = db.Exec(`INSERT INTO santri (nama, kelas_id) VALUES ('Tetap Terhubung', ?)`, kelasID)
	require.NoError(t, err)
	result, err := db.Exec(`INSERT INTO guru (nama, jenis_kelamin) VALUES ('Guru Uji', 'L')`)
	require.NoError(t, err)
	guruID, err := result.LastInsertId()
	require.NoError(t, err)
	result, err = db.Exec(`INSERT INTO pertemuan (kelas_id, pertemuan_ke, tanggal) VALUES (?, 1, '2026-08-24')`, kelasID)
	require.NoError(t, err)
	pertemuanID, err := result.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO jadwal_pertemuan (kelas_id, tanggal, jam_mulai) VALUES (?, '2026-08-24', '19:00')`, kelasID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO tagihan (santri_id, kelas_id, pertemuan_id, bulan_ke, pertemuan_ke, tanggal_tagih) VALUES (1, ?, ?, 1, 1, '2026-08-24')`, kelasID, pertemuanID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO kunjungan_kelas (guru_id, kelas_id) VALUES (?, ?)`, guruID, kelasID)
	require.NoError(t, err)
	result, err = db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, nama_kelas) VALUES ('DELETED-HIGH', 'AKA38', 'Reguler', 'L', '01', '1x/pekan', 'Kelas Terhapus')`)
	require.NoError(t, err)
	deletedHighID, err := result.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM kelas WHERE id = ?`, deletedHighID)
	require.NoError(t, err)

	require.NoError(t, goose.UpTo(db, migrationsPath, 28))
	capacities := map[string]int{}
	rows, err := db.Query(`SELECT kunci_kelas, kapasitas FROM kelas`)
	require.NoError(t, err)
	for rows.Next() {
		var key string
		var capacity int
		require.NoError(t, rows.Scan(&key, &capacity))
		capacities[key] = capacity
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
	require.Equal(t, map[string]int{"DEFAULT-10": 15, "CUSTOM-2": 2, "CUSTOM-20": 20}, capacities)
	for _, table := range []string{"santri", "pertemuan", "jadwal_pertemuan", "tagihan", "kunjungan_kelas"} {
		requireTableCount(t, db, table, 1)
	}

	result, err = db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, nama_kelas) VALUES ('NEW-DEFAULT', 'AKA38', 'Reguler', 'L', '01', '1x/pekan', 'Kelas Baru')`)
	require.NoError(t, err)
	newID, err := result.LastInsertId()
	require.NoError(t, err)
	require.Greater(t, newID, deletedHighID)
	var defaultCapacity int
	require.NoError(t, db.QueryRow(`SELECT kapasitas FROM kelas WHERE kunci_kelas = 'NEW-DEFAULT'`).Scan(&defaultCapacity))
	require.Equal(t, 15, defaultCapacity)

	rows, err = db.Query(`PRAGMA foreign_key_check`)
	require.NoError(t, err)
	require.False(t, rows.Next())
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())

	require.NoError(t, goose.DownTo(db, migrationsPath, 27))
	_, err = db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, nama_kelas) VALUES ('DOWN-DEFAULT', 'AKA38', 'Reguler', 'L', '01', '1x/pekan', 'Kelas Lama')`)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT kapasitas FROM kelas WHERE kunci_kelas = 'DOWN-DEFAULT'`).Scan(&defaultCapacity))
	require.Equal(t, 10, defaultCapacity)
}

func TestImportCSVRejectsUnsupportedHeaderAndElevenColumnRow(t *testing.T) {
	db, querier, service, userID := setupSantriIntegrityService(t)
	importService := NewImportService(querier, service)

	_, err := importService.ProcessCSV(strings.NewReader("nama,angkatan\nSantri,AKA38\n"), userID, "bad-header.csv")
	require.ErrorContains(t, err, "header CSV tidak didukung")

	elevenColumns := "kelas_kode,nama,no_whatsapp,email,jenis_kelamin,nominal,tanggal_daftar,angkatan,usia,domisili\nR 1X,Sebelas Kolom,0812,test@example.com,L,100000,2026-02-03,AKA38,20,Bandung,EKSTRA\n"
	result, err := importService.ProcessCSV(strings.NewReader(elevenColumns), userID, "eleven.csv")
	require.NoError(t, err)
	require.Zero(t, result.Berhasil)
	require.Equal(t, 1, result.Gagal)
	require.Contains(t, result.Catatan, "jumlah kolom 11, seharusnya 10")
	requireTableCount(t, db, "santri", 0)
}

func TestMigration0025NormalizesSantriTanggalDaftar(t *testing.T) {
	migrationsPath, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	require.NoError(t, err)
	t.Chdir(t.TempDir())
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.UpTo(db, migrationsPath, 24))

	_, err = db.Exec(`INSERT INTO santri (nama, tanggal_daftar) VALUES
		('Timestamp', '2026-02-03 00:00:00 +0000 UTC'),
		('SQLTimestamp', '2026-02-03 12:34:56'),
		('TimestampT', '2026-02-03T12:34:56Z'),
		('OffsetPlus', '2026-02-03T12:34:56+07:00'),
		('OffsetMinus', '2026-02-03T12:34:56-05:30'),
		('Impossible', '2026-02-30'),
		('Malformed', 'bukan-tanggal'),
		('GarbageSuffix', '2026-02-03garbage'),
		('TimestampGarbage', '2026-02-03 00:00:00 +0000 UTCgarbage')`)
	require.NoError(t, err)
	require.NoError(t, goose.UpTo(db, migrationsPath, 25))

	rows, err := db.Query(`SELECT nama, CAST(tanggal_daftar AS TEXT) FROM santri ORDER BY id`)
	require.NoError(t, err)
	values := map[string]sql.NullString{}
	for rows.Next() {
		var name string
		var value sql.NullString
		require.NoError(t, rows.Scan(&name, &value))
		values[name] = value
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
	require.Equal(t, sql.NullString{String: "2026-02-03", Valid: true}, values["Timestamp"])
	require.Equal(t, sql.NullString{String: "2026-02-03", Valid: true}, values["SQLTimestamp"])
	require.Equal(t, sql.NullString{String: "2026-02-03", Valid: true}, values["TimestampT"])
	require.Equal(t, sql.NullString{String: "2026-02-03", Valid: true}, values["OffsetPlus"])
	require.Equal(t, sql.NullString{String: "2026-02-03", Valid: true}, values["OffsetMinus"])
	require.False(t, values["Impossible"].Valid)
	require.False(t, values["Malformed"].Valid)
	require.False(t, values["GarbageSuffix"].Valid)
	require.False(t, values["TimestampGarbage"].Valid)

	require.NoError(t, goose.DownTo(db, migrationsPath, 24))
	var normalized string
	require.NoError(t, db.QueryRow(`SELECT CAST(tanggal_daftar AS TEXT) FROM santri WHERE nama = 'Timestamp'`).Scan(&normalized))
	require.Equal(t, "2026-02-03", normalized)
}

func insertDeleteTestKelas(t *testing.T, db *sql.DB, key string) int64 {
	t.Helper()
	result, err := db.Exec(`INSERT INTO kelas (kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal, sub_index, nama_kelas, kapasitas, jumlah_santri) VALUES (?, '2026', 'Reguler', 'L', 'Dasar', '1x/pekan', 'Senin', 1, ?, 20, 1)`, key, "Kelas "+key)
	require.NoError(t, err)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

func requireTableCount(t *testing.T, db *sql.DB, table string, want int64) {
	t.Helper()
	var count int64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&count))
	require.Equal(t, want, count, table)
}

func TestSantriDeleteRemovesRelatedDataAndVoiceNoteButKeepsKelas(t *testing.T) {
	db, service := setupSantriDeleteService(t)
	kelasID := insertDeleteTestKelas(t, db, "DELETE")

	voiceNoteURL := filepath.ToSlash(filepath.Join("voice-notes", "santri-delete.mp3"))
	require.NoError(t, os.MkdirAll(filepath.Join("data", "voice-notes"), 0o755))
	voiceNotePath := filepath.Join("data", filepath.FromSlash(voiceNoteURL))
	require.NoError(t, os.WriteFile(voiceNotePath, []byte("voice note"), 0o600))

	result, err := db.Exec(`INSERT INTO santri (nama, kelas_id, status, voice_note_url) VALUES ('Santri Hapus', ?, 'aktif', ?)`, kelasID, voiceNoteURL)
	require.NoError(t, err)
	santriID, err := result.LastInsertId()
	require.NoError(t, err)
	result, err = db.Exec(`INSERT INTO pertemuan (kelas_id, pertemuan_ke, tanggal, status) VALUES (?, 1, '2026-08-18', 'selesai')`, kelasID)
	require.NoError(t, err)
	pertemuanID, err := result.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO absensi (pertemuan_id, santri_id, status) VALUES (?, ?, 'hadir')`, pertemuanID, santriID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO tagihan (santri_id, kelas_id, bulan_ke, pertemuan_ke, nominal, tanggal_tagih) VALUES (?, ?, 1, 1, 100000, '2026-08-18')`, santriID, kelasID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO catatan_riayah (target_type, target_id, catatan) VALUES ('santri', ?, 'Catatan santri')`, santriID)
	require.NoError(t, err)

	require.NoError(t, service.Delete(santriID))
	requireTableCount(t, db, "santri", 0)
	requireTableCount(t, db, "absensi", 0)
	requireTableCount(t, db, "tagihan", 0)
	requireTableCount(t, db, "catatan_riayah", 0)
	requireTableCount(t, db, "kelas", 1)
	_, err = os.Stat(voiceNotePath)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestSantriDeleteMissingIDPreservesUnrelatedData(t *testing.T) {
	db, service := setupSantriDeleteService(t)
	kelasID := insertDeleteTestKelas(t, db, "KEEP")
	voiceNoteURL := filepath.ToSlash(filepath.Join("voice-notes", "santri-keep.mp3"))
	require.NoError(t, os.MkdirAll(filepath.Join("data", "voice-notes"), 0o755))
	voiceNotePath := filepath.Join("data", filepath.FromSlash(voiceNoteURL))
	require.NoError(t, os.WriteFile(voiceNotePath, []byte("keep"), 0o600))

	result, err := db.Exec(`INSERT INTO santri (nama, kelas_id, status, voice_note_url) VALUES ('Santri Tetap', ?, 'aktif', ?)`, kelasID, voiceNoteURL)
	require.NoError(t, err)
	santriID, err := result.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO catatan_riayah (target_type, target_id, catatan) VALUES ('santri', ?, 'Tetap ada')`, santriID)
	require.NoError(t, err)

	require.Error(t, service.Delete(santriID+999))
	_, err = queries.NewQuerier(db).GetSantriByID(context.Background(), santriID)
	require.NoError(t, err)
	requireTableCount(t, db, "santri", 1)
	requireTableCount(t, db, "catatan_riayah", 1)
	requireTableCount(t, db, "kelas", 1)
	_, err = os.Stat(voiceNotePath)
	require.NoError(t, err)
}

func TestSafeVoiceNotePath(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{name: "valid flat file", url: "voice-notes/santri-1.mp3", want: filepath.Join("data", "voice-notes", "santri-1.mp3")},
		{name: "absolute", url: "/voice-notes/santri-1.mp3", wantErr: true},
		{name: "traversal", url: "voice-notes/../../secret.mp3", wantErr: true},
		{name: "nested directory", url: "voice-notes/link/secret.mp3", wantErr: true},
		{name: "wrong directory", url: "uploads/santri-1.mp3", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := safeVoiceNotePath(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
