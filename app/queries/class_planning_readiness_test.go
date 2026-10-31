package queries

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestReadPlanningReadinessDistinguishesPreassignedAndVerified(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	db.SetMaxOpenConns(1)
	schema := []string{
		"CREATE TABLE santri (id INTEGER PRIMARY KEY, kelas_id INTEGER, jenis_kelamin TEXT, level TEXT, jadwal TEXT, frekuensi TEXT, is_lengkap INTEGER, status TEXT)",
		"CREATE TABLE onboarding_cases (santri_id INTEGER, stage TEXT)",
		"CREATE TABLE leads (santri_id INTEGER, verified_at TEXT, needs_review INTEGER, status TEXT)",
		"CREATE TABLE guru (id INTEGER PRIMARY KEY, is_aktif INTEGER)",
		"CREATE TABLE guru_availability (guru_id INTEGER, confirmed_at TEXT, berlaku_mulai TEXT, berlaku_sampai TEXT)",
		"CREATE TABLE guru_teaching_levels (guru_id INTEGER, level_kode TEXT)",
		"CREATE TABLE level (kode TEXT PRIMARY KEY)",
		"CREATE TABLE kelas (id INTEGER, guru_id INTEGER, is_aktif INTEGER, tipe TEXT, kapasitas INTEGER)",
		"CREATE TABLE kelas_jadwal_rutin (kelas_id INTEGER, is_aktif INTEGER, berlaku_mulai TEXT, berlaku_sampai TEXT)",
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil { t.Fatal(err) }
	}
	fixtures := []string{
		"INSERT INTO santri VALUES (1, 10, 'P', 'PRA', 'Selasa 19:30', '2x', 1, 'aktif')",
		"INSERT INTO santri VALUES (2, NULL, 'P', 'PRA', 'Selasa 19:30', '2x', 1, 'aktif')",
		"INSERT INTO santri VALUES (3, NULL, 'L', '', '', '', 0, 'aktif')",
		"INSERT INTO santri VALUES (4, 10, 'P', 'PRA', 'Selasa 19:30', '2x', 1, 'aktif')",
		"INSERT INTO onboarding_cases VALUES (1,'waiting_kuota'), (2,'matching_kelas'), (3,'onboarding_baru'), (4,'masuk_kelas')",
		"INSERT INTO leads VALUES (1, '2026-10-30 10:00:00', 0, 'open')",
		"INSERT INTO leads VALUES (2, '2026-10-30 10:00:00', 0, 'closed')",
		"INSERT INTO guru VALUES (100,1),(200,1),(300,0)",
		"INSERT INTO guru_availability VALUES (100,'2026-10-30','2026-10-01',NULL),(200,'2026-10-30','2026-10-01',NULL),(300,'2026-10-30','2026-10-01',NULL)",
		"INSERT INTO level VALUES ('PRA')",
		"INSERT INTO guru_teaching_levels VALUES (100,'PRA'),(300,'PRA')",
		"INSERT INTO kelas VALUES (10,100,1,'reguler',15)",
	}
	for _, stmt := range fixtures { if _, err := db.Exec(stmt); err != nil { t.Fatal(err) } }
	q := NewQuerier(db)
	r, err := q.ReadPlanningReadiness(context.Background(), "2026-10-31")
	if err != nil { t.Fatal(err) }
	if r.AwaitingOnboarding != 3 || r.LegacyPreassigned != 1 || r.CandidateDataComplete != 2 ||
		r.CRMPaymentVerified != 1 || r.CRMVerificationUnknown != 1 {
		t.Fatalf("onboarding counts wrong: %+v", r)
	}
	if r.ActiveTeachers != 2 || r.ConfirmedAvailabilityGuru != 2 ||
		r.ApprovedTeachingLevelGuru != 1 || r.UsableTeacherCount != 1 {
		t.Fatalf("teacher counts wrong: %+v", r)
	}
	if r.ActiveClassesNoRoutineSlots != 1 || r.RegularClassesAboveMax != 1 {
		t.Fatalf("class health counts wrong: %+v", r)
	}
	if _, err := db.Exec("INSERT INTO kelas_jadwal_rutin VALUES (10,1,'2026-10-01',NULL)"); err != nil { t.Fatal(err) }
	r, err = q.ReadPlanningReadiness(context.Background(), "2026-10-31")
	if err != nil || r.ActiveClassesNoRoutineSlots != 0 {
		t.Fatalf("routine schedule not reflected: %+v, %v", r, err)
	}
}

func TestReadPlanningReadinessInvalidDate(t *testing.T) {
	q := NewQuerier(nil)
	if _, err := q.ReadPlanningReadiness(context.Background(), "2026-02-30"); err == nil {
		t.Fatal("invalid date accepted")
	}
}
