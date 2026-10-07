package services

import (
	"context"
	"database/sql"
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func insertSantriPerluDilengkapi(t *testing.T, db *sql.DB, nama string) int64 {
	t.Helper()
	result, err := db.Exec(`INSERT INTO santri (kelas_kode, nama, jenis_kelamin, angkatan_kelas, status) VALUES ('R 1X', ?, 'L', 'AKA38', 'aktif')`, nama)
	require.NoError(t, err)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

func adminKelasRequest(guruID int64) models.UpdateSantriAdminKelasRequest {
	return models.UpdateSantriAdminKelasRequest{AngkatanKelas: "AKA38", Level: "01", Jadwal: "Senin", GuruID: guruID}
}

func TestUpdateByAdminKelasGuruBedaMembuatKelasTerpisah(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	ctx := context.Background()
	guruA := insertTestGuru(t, db, "Guru A", insertTestUser(t, db, "a@example.com", "Guru A"))
	guruB := insertTestGuru(t, db, "Guru B", insertTestUser(t, db, "b@example.com", "Guru B"))

	santriA := insertSantriPerluDilengkapi(t, db, "Santri A")
	require.NoError(t, service.UpdateByAdminKelas(santriA, adminKelasRequest(guruA)))
	santriB := insertSantriPerluDilengkapi(t, db, "Santri B")
	require.NoError(t, service.UpdateByAdminKelas(santriB, adminKelasRequest(guruB)))

	storedA, err := querier.GetSantriByID(ctx, santriA)
	require.NoError(t, err)
	storedB, err := querier.GetSantriByID(ctx, santriB)
	require.NoError(t, err)
	require.NotEqual(t, storedA.KelasID.Int64, storedB.KelasID.Int64)

	kelasA, err := querier.GetKelasByID(ctx, storedA.KelasID.Int64)
	require.NoError(t, err)
	kelasB, err := querier.GetKelasByID(ctx, storedB.KelasID.Int64)
	require.NoError(t, err)
	require.Equal(t, guruA, kelasA.GuruID.Int64, "guru kelas A tidak boleh tertimpa")
	require.Equal(t, guruB, kelasB.GuruID.Int64)
	require.Equal(t, kelasA.KunciKelas, kelasB.KunciKelas)
	require.EqualValues(t, 2, kelasB.SubIndex)

	// A third santri for guru A joins guru A's class, not guru B's.
	santriC := insertSantriPerluDilengkapi(t, db, "Santri C")
	require.NoError(t, service.UpdateByAdminKelas(santriC, adminKelasRequest(guruA)))
	storedC, err := querier.GetSantriByID(ctx, santriC)
	require.NoError(t, err)
	require.Equal(t, storedA.KelasID.Int64, storedC.KelasID.Int64)
}

func TestUpdateByAdminKelasMengisiKelasTanpaGuru(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	ctx := context.Background()
	guruB := insertTestGuru(t, db, "Guru B", insertTestUser(t, db, "b@example.com", "Guru B"))

	santriA := insertSantriPerluDilengkapi(t, db, "Santri A")
	require.NoError(t, service.UpdateByAdminKelas(santriA, adminKelasRequest(0)))
	storedA, err := querier.GetSantriByID(ctx, santriA)
	require.NoError(t, err)

	santriB := insertSantriPerluDilengkapi(t, db, "Santri B")
	require.NoError(t, service.UpdateByAdminKelas(santriB, adminKelasRequest(guruB)))
	storedB, err := querier.GetSantriByID(ctx, santriB)
	require.NoError(t, err)
	require.Equal(t, storedA.KelasID.Int64, storedB.KelasID.Int64)

	kelas, err := querier.GetKelasByID(ctx, storedB.KelasID.Int64)
	require.NoError(t, err)
	require.Equal(t, guruB, kelas.GuruID.Int64)
}

func TestUpdateByAdminKelasGantiGuruMemindahkanSantri(t *testing.T) {
	db, querier, service, _ := setupSantriIntegrityService(t)
	ctx := context.Background()
	guruA := insertTestGuru(t, db, "Guru A", insertTestUser(t, db, "a@example.com", "Guru A"))
	guruB := insertTestGuru(t, db, "Guru B", insertTestUser(t, db, "b@example.com", "Guru B"))

	santriA := insertSantriPerluDilengkapi(t, db, "Santri A")
	require.NoError(t, service.UpdateByAdminKelas(santriA, adminKelasRequest(guruA)))
	santriB := insertSantriPerluDilengkapi(t, db, "Santri B")
	require.NoError(t, service.UpdateByAdminKelas(santriB, adminKelasRequest(guruA)))
	before, err := querier.GetSantriByID(ctx, santriB)
	require.NoError(t, err)

	require.NoError(t, service.UpdateByAdminKelas(santriB, adminKelasRequest(guruB)))
	after, err := querier.GetSantriByID(ctx, santriB)
	require.NoError(t, err)
	require.NotEqual(t, before.KelasID.Int64, after.KelasID.Int64)

	kelasLama, err := querier.GetKelasByID(ctx, before.KelasID.Int64)
	require.NoError(t, err)
	require.Equal(t, guruA, kelasLama.GuruID.Int64)
	require.EqualValues(t, 1, kelasLama.JumlahSantri)
}
