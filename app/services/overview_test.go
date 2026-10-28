package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupOverview(t *testing.T) (*OverviewService, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	q := queries.NewQuerier(db)
	now := func() time.Time { return time.Date(2026, 3, 15, 9, 0, 0, 0, ZonaWaktuRiayah) }
	ujroh := NewUjrohService(q)
	ujroh.now = now
	finance := NewKeuanganDashboardService(q, ujroh)
	finance.now = now
	service := NewOverviewService(q, finance, NewRiayahSantriService(q))
	service.now = now
	return service, db
}

func TestOverviewSnapshotsPeriodAndReadOnly(t *testing.T) {
	s, db := setupOverview(t)
	for _, statement := range []string{
		`INSERT INTO guru (id,nama,is_aktif) VALUES (1,'Aktif',1),(2,'Nonaktif',0)`,
		`INSERT INTO kelas (id,kunci_kelas,angkatan,tipe,jenis_kelamin,level,frekuensi,nama_kelas,guru_id,kapasitas,is_aktif) VALUES (1,'A','2026','Reguler','L','Dasar','1x/pekan','A',1,10,1),(2,'B','2026','Reguler','L','Dasar','1x/pekan','B',2,10,1),(3,'C','2026','Private','L','Dasar','1x/pekan','C',NULL,1,0)`,
		`INSERT INTO santri (nama,tipe,status,kelas_id,tanggal_daftar) VALUES ('A','Reguler','aktif',1,'2026-01-10'),('B','Reguler','cuti',1,'2026-02-10'),('C','Reguler','tidak_lanjut',1,'2026-03-02'),('D','Private','aktif',3,'2026-03-03'),('E','Private','aktif',NULL,NULL)`,
		`INSERT INTO pertemuan (kelas_id,pertemuan_ke,tanggal,status,is_badal,is_reschedule) VALUES (1,1,'2026-03-10','selesai',1,1),(1,2,'2026-03-14','berlangsung',0,0),(2,1,'2026-03-15','berlangsung',0,0),(2,2,'2026-02-14','selesai',0,0)`,
		`INSERT INTO todo_koordinator (judul,deadline,status) VALUES ('Hari ini','2026-03-15','belum'),('Besok','2026-03-16','belum'),('Sudah','2026-03-01','selesai')`,
	} {
		_, err := db.Exec(statement)
		require.NoError(t, err)
	}
	var before, after int64
	require.NoError(t, db.QueryRow(`SELECT total_changes()`).Scan(&before))
	d, err := s.Get(context.Background(), "", "super_admin")
	require.NoError(t, err)
	require.Empty(t, d.Errors)
	require.EqualValues(t, 5, d.Santri.Total)
	require.EqualValues(t, 3, d.Santri.Aktif)
	require.InDelta(t, 60, d.Santri.ActiveRate, .001)
	require.EqualValues(t, 3, d.Operations.Incomplete)
	require.EqualValues(t, 2, d.Operations.Classes)
	require.EqualValues(t, 1, d.Operations.Teachers)
	require.EqualValues(t, 1, d.Operations.WithoutTeacher)
	require.EqualValues(t, 2, d.Operations.WithoutClass)
	require.EqualValues(t, 1, d.Operations.OpenMeetings, "today's ongoing meeting is not overdue")
	require.EqualValues(t, 1, d.Operations.Todos)
	require.EqualValues(t, 1, d.Operations.Finished)
	require.EqualValues(t, 1, d.Operations.Badal)
	require.EqualValues(t, 1, d.Operations.Rescheduled)
	require.Len(t, d.Growth, 6)
	require.EqualValues(t, 2, d.Growth[5].Total)
	require.Len(t, d.Programs, 2)
	for _, p := range d.Programs {
		if p.Name == "Reguler" {
			require.EqualValues(t, 2, p.Occupied)
			require.EqualValues(t, 20, p.Seats)
			require.EqualValues(t, 1, p.Active)
		}
	}
	old, err := s.Get(context.Background(), "2026-02", "super_admin")
	require.NoError(t, err)
	require.Empty(t, old.Errors)
	require.Equal(t, d.Santri, old.Santri, "month filter must not pretend current statuses are historical")
	require.EqualValues(t, 1, old.Operations.Finished)
	require.Zero(t, old.Operations.Badal)
	require.EqualValues(t, 1, old.Growth[5].Total)
	require.NoError(t, db.QueryRow(`SELECT total_changes()`).Scan(&after))
	require.Equal(t, before, after, "opening the overview must not mutate the database")
}

func TestOverviewAttendanceAndLeads(t *testing.T) {
	s, db := setupOverview(t)
	for _, statement := range []string{
		`INSERT INTO kelas (id,kunci_kelas,angkatan,tipe,jenis_kelamin,level,frekuensi,nama_kelas,kapasitas,is_aktif) VALUES (1,'A','2026','Reguler','L','Dasar','1x/pekan','A',10,1)`,
		`INSERT INTO santri (id,nama,tipe,status,kelas_id) VALUES (1,'A','Reguler','aktif',1),(2,'B','Reguler','aktif',1),(3,'C','Reguler','aktif',1)`,
		`INSERT INTO pertemuan (id,kelas_id,pertemuan_ke,tanggal,status) VALUES (1,1,1,'2026-03-03','selesai'),(2,1,2,'2026-03-10','selesai'),(3,1,3,'2026-03-14','berlangsung'),(4,1,4,'2026-01-20','selesai'),(5,1,5,'2025-09-20','selesai')`,
		`INSERT INTO absensi (pertemuan_id,santri_id,status) VALUES (1,1,'hadir'),(1,2,'izin'),(1,3,'alpa'),(2,1,'hadir'),(2,2,'hadir'),(2,3,'sakit'),(3,1,'hadir'),(4,1,'hadir'),(5,1,'alpa')`,
	} {
		_, err := db.Exec(statement)
		require.NoError(t, err)
	}
	seedLead := func(chat string, created time.Time, converted *time.Time) {
		t.Helper()
		status := "open"
		var conv any
		if converted != nil {
			status, conv = "won", *converted
		}
		_, err := db.Exec(`INSERT INTO leads (wa_chat_id, created_at, updated_at, converted_at, stage, status) VALUES (?, ?, ?, ?, 'baru', ?)`, chat, created, created, conv, status)
		require.NoError(t, err)
	}
	wib := func(month time.Month, day, hour int) time.Time { return time.Date(2026, month, day, hour, 0, 0, 0, ZonaWaktuRiayah) }
	closed := wib(3, 2, 10)
	closedFeb := wib(2, 27, 10)
	// 1 March 01:00 WIB is still 28 February in UTC: it belongs to March.
	seedLead("march-early", wib(3, 1, 1), &closed)
	seedLead("march", wib(3, 12, 9), nil)
	seedLead("feb", wib(2, 10, 9), &closedFeb)

	d, err := s.Get(context.Background(), "", "super_admin")
	require.NoError(t, err)
	require.Empty(t, d.Errors)
	require.Len(t, d.Attendance, 6)
	require.Equal(t, "2025-10", d.Attendance[0].Month)
	require.Equal(t, "2026-03", d.Attendance[5].Month)
	require.Equal(t, models.OverviewAttendanceMonth{Month: "2026-03", Meetings: 2, Hadir: 3, Izin: 1, Sakit: 1, Alpa: 1}, d.Attendance[5], "ongoing meetings are excluded")
	require.Equal(t, models.OverviewAttendanceMonth{Month: "2026-01", Meetings: 1, Hadir: 1}, d.Attendance[3])
	require.Equal(t, models.OverviewAttendanceMonth{Month: "2026-02"}, d.Attendance[4], "empty months are filled with zero")
	require.Equal(t, &models.OverviewLeads{Masuk: 2, Closing: 1, MasukSebelumnya: 1, ClosingSebelumnya: 1}, d.Leads)

	fin, err := s.Get(context.Background(), "", "keuangan")
	require.NoError(t, err)
	require.Empty(t, fin.Attendance)
	require.Nil(t, fin.Leads)
}

func TestOverviewPermissionsAndValidation(t *testing.T) {
	s, _ := setupOverview(t)
	for _, role := range []string{"admin", "keuangan"} {
		d, err := s.Get(context.Background(), "2026-03", role)
		require.NoError(t, err)
		require.Empty(t, d.Errors)
		require.NotNil(t, d.Finance)
		require.Nil(t, d.Santri)
		require.Nil(t, d.Operations)
		require.Nil(t, d.Riayah)
		require.Nil(t, d.Products)
		require.Empty(t, d.Programs)
		payload, err := json.Marshal(d)
		require.NoError(t, err)
		require.NotContains(t, string(payload), `"santri":`)
		require.NotContains(t, string(payload), `"operations":`)
	}
	for _, role := range []string{"user", "cs", "admin_kelas", "guru", "koordinator_guru", "unknown", ""} {
		d, err := s.Get(context.Background(), "2026-03", role)
		require.Error(t, err)
		require.Nil(t, d)
	}
	for _, month := range []string{"2026-04", "2026-13", "2026-3", "1999-12", "bad"} {
		d, err := s.Get(context.Background(), month, "super_admin")
		require.Error(t, err)
		require.Nil(t, d)
	}
	// Near midnight UTC, the default month is already April in Jakarta.
	s.now = func() time.Time { return time.Date(2026, 3, 31, 18, 0, 0, 0, time.UTC) }
	d, err := s.Get(context.Background(), "", "super_admin")
	require.NoError(t, err)
	require.Equal(t, "2026-04", d.Bulan)
}

func TestOverviewPartialFailureDoesNotBecomeZero(t *testing.T) {
	s, db := setupOverview(t)
	_, err := db.Exec(`DROP TABLE todo_koordinator`)
	require.NoError(t, err)
	d, err := s.Get(context.Background(), "", "super_admin")
	require.NoError(t, err)
	require.Nil(t, d.Operations)
	require.NotNil(t, d.Santri)
	require.NotNil(t, d.Finance)
	require.Contains(t, d.Errors, "Operasional kelas dan guru")
	_, err = db.Exec(`DROP TABLE tagihan`)
	require.NoError(t, err)
	d, err = s.Get(context.Background(), "", "super_admin")
	require.NoError(t, err)
	require.Nil(t, d.Finance)
	require.Contains(t, d.Errors, "Keuangan")
}

func TestOverviewEmptyIsSuccessfulZero(t *testing.T) {
	s, _ := setupOverview(t)
	d, err := s.Get(context.Background(), "", "super_admin")
	require.NoError(t, err)
	require.Empty(t, d.Errors)
	require.NotNil(t, d.Santri)
	require.Zero(t, d.Santri.Total)
	require.NotNil(t, d.Operations)
	require.NotNil(t, d.Riayah)
	require.NotNil(t, d.Finance)
	require.Empty(t, d.Priorities)
	require.Empty(t, d.Programs)
	require.Len(t, d.Growth, 6)
}
