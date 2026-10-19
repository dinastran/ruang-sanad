package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

// Dates written through the driver as time.Time are stored as
// "YYYY-MM-DD 00:00:00 +0000 UTC", which SQLite date()/strftime() cannot
// parse. New santri registered from the form were therefore missing from the
// registration trend, and overview meeting counts read as zero.
func TestDriverWrittenDatesAreCounted(t *testing.T) {
	s, db := setupOverview(t)
	d := func(y int, m time.Month, day int) sql.NullTime {
		return sql.NullTime{Time: time.Date(y, m, day, 0, 0, 0, 0, time.UTC), Valid: true}
	}
	_, err := db.Exec(`INSERT INTO guru (id,nama,is_aktif) VALUES (1,'G',1)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO kelas (id,kunci_kelas,angkatan,tipe,jenis_kelamin,level,frekuensi,nama_kelas,guru_id,kapasitas,is_aktif) VALUES (1,'A','2026','Reguler','L','Dasar','1x/pekan','A',1,10,1)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO santri (nama,status,kelas_id,tanggal_daftar,mulai_belajar) VALUES ('Baru','aktif',1,?,?),('Lama','aktif',1,'2026-02-01',NULL)`, d(2026, 3, 10), d(2026, 3, 11))
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO pertemuan (kelas_id,pertemuan_ke,tanggal,status,is_badal) VALUES (1,1,?,'selesai',1),(1,2,?,'berlangsung',0)`, d(2026, 3, 5), d(2026, 3, 12))
	require.NoError(t, err)

	var raw string
	require.NoError(t, db.QueryRow(`SELECT CAST(tanggal_daftar AS TEXT) FROM santri WHERE nama='Baru'`).Scan(&raw))
	require.Contains(t, raw, "+0000 UTC", "precondition: driver format")

	analytics := NewSantriAnalyticsService(s.querier)
	data, err := analytics.Get(models.SantriAnalyticsFilters{DateFrom: "2026-03-01", DateTo: "2026-03-31"})
	require.NoError(t, err)
	require.EqualValues(t, 1, data.Summary.Total, "March registration filter must include driver-written date")
	require.NotEmpty(t, data.Growth)
	require.Equal(t, "2026-03", data.Growth[len(data.Growth)-1].Month)

	ov, err := s.Get(context.Background(), "2026-03", "super_admin")
	require.NoError(t, err)
	require.EqualValues(t, 1, ov.Operations.OpenMeetings)
	require.EqualValues(t, 1, ov.Operations.Finished)
	require.EqualValues(t, 1, ov.Operations.Badal)
	var march int64
	for _, g := range ov.Growth {
		if g.Month == "2026-03" {
			march = g.Total
		}
	}
	require.EqualValues(t, 1, march, "dashboard growth must count the new registration")
}
