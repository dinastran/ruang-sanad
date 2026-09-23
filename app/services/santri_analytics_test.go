package services

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupSantriAnalytics(t *testing.T) *SantriAnalyticsService {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))

	_, err = db.Exec(`INSERT INTO level (kode, nama, urutan) VALUES ('01', 'Dasar', 1), ('02', 'Lanjutan', 2)`)
	require.NoError(t, err)
	rows := []struct {
		name, gender, date, cohort, classCohort, level, tipe, status, domisili string
		age interface{}
	}{
		{"Aisyah", "P", "2026-01-10", "A", "A", "01", "Reguler", "aktif", "Bandung", 36},
		{"Fatimah", "P", "2026-01-15", "A", "A", "01", "Reguler", "aktif", "Bandung", 42},
		{"Ahmad", "L", "2026-03-01", "B", "B", "02", "Intensif", "aktif", "Jakarta", 28},
		{"Umar", "L", "2026-03-12", "B", "B", "02", "Intensif", "cuti", "Jakarta", 46},
		{"Maryam", "P", "2026-03-20", "B", "B", "01", "Reguler", "nonaktif", "Surabaya", 56},
		{"Data Kurang", "", "", "C", "", "", "", "tidak_lanjut", "", nil},
	}
	for _, row := range rows {
		_, err := db.Exec(`INSERT INTO santri (nama, jenis_kelamin, tanggal_daftar, angkatan, angkatan_kelas, level, tipe, status, domisili, usia)
			VALUES (?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?, ?)`,
			row.name, row.gender, row.date, row.cohort, row.classCohort, row.level, row.tipe, row.status, row.domisili, row.age)
		require.NoError(t, err)
	}
	return NewSantriAnalyticsService(queries.NewQuerier(db))
}

func TestSantriAnalyticsOverviewAndDataQuality(t *testing.T) {
	data, err := setupSantriAnalytics(t).Get(models.SantriAnalyticsFilters{Page: 1, Limit: 25})
	require.NoError(t, err)
	require.EqualValues(t, 6, data.Summary.Total)
	require.EqualValues(t, 3, data.Summary.Aktif)
	require.EqualValues(t, 1, data.Summary.Cuti)
	require.EqualValues(t, 1, data.Summary.Nonaktif)
	require.EqualValues(t, 1, data.Summary.TidakLanjut)
	require.InDelta(t, 50, data.Summary.ActiveRate, 0.001)
	require.EqualValues(t, 1, data.DataQuality.UsiaKosong)
	require.EqualValues(t, 1, data.DataQuality.DomisiliKosong)
	require.EqualValues(t, 1, data.DataQuality.TanggalDaftarKosong)
	require.EqualValues(t, 1, data.DataQuality.JenisKelaminKosong)
}

func TestSantriAnalyticsGrowthFillsMissingMonth(t *testing.T) {
	data, err := setupSantriAnalytics(t).Get(models.SantriAnalyticsFilters{DateFrom: "2026-01-01", DateTo: "2026-03-31", Page: 1, Limit: 25})
	require.NoError(t, err)
	require.Len(t, data.Growth, 3)
	require.Equal(t, "2026-01", data.Growth[0].Month)
	require.EqualValues(t, 2, data.Growth[0].Total)
	require.Equal(t, "2026-02", data.Growth[1].Month)
	require.Zero(t, data.Growth[1].Total)
	require.Equal(t, "2026-03", data.Growth[2].Month)
	require.EqualValues(t, 3, data.Growth[2].Total)
}

func TestSantriAnalyticsDrillDownCombinesSegments(t *testing.T) {
	data, err := setupSantriAnalytics(t).Get(models.SantriAnalyticsFilters{
		SegmentAgeBucket: "35-44", SegmentGender: "P", SegmentDomisili: "Bandung", SegmentStatus: "aktif", Page: 1, Limit: 25,
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, data.RowTotal)
	require.Len(t, data.Rows, 2)
	for _, row := range data.Rows {
		require.Equal(t, "P", row.JenisKelamin)
		require.Equal(t, "Bandung", row.Domisili)
		require.Equal(t, "aktif", row.Status)
		require.GreaterOrEqual(t, row.Usia, int64(35))
		require.LessOrEqual(t, row.Usia, int64(44))
	}
}

func TestNormalizeSantriAnalyticsFiltersRejectsInvalidInput(t *testing.T) {
	_, err := NormalizeSantriAnalyticsFilters(models.SantriAnalyticsFilters{DateFrom: "2026-13-01"})
	require.Error(t, err)
	_, err = NormalizeSantriAnalyticsFilters(models.SantriAnalyticsFilters{SegmentAgeBucket: "20-99"})
	require.Error(t, err)
	_, err = NormalizeSantriAnalyticsFilters(models.SantriAnalyticsFilters{SegmentRegistrationMo: "2026-99"})
	require.Error(t, err)
}
