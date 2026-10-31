package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

type activationFixture struct {
	db            *sql.DB
	q             *queries.Querier
	planID        int64
	adminID       int64
	coordinatorID int64
	studentIDs    []int64
}

func setupActivationFixture(t *testing.T) activationFixture {
	t.Helper()
	db, q := setupSantriDatabase(t)
	insert := func(statement string, args ...any) int64 {
		t.Helper()
		r, err := db.Exec(statement, args...)
		require.NoError(t, err)
		id, err := r.LastInsertId()
		require.NoError(t, err)
		return id
	}
	admin := insert("INSERT INTO users(email,name,role) VALUES ('planning-admin@example.com','Planning Admin','admin_kelas')")
	coordinator := insert("INSERT INTO users(email,name,role) VALUES ('planning-coordinator@example.com','Planning Coordinator','koordinator_guru')")
	insert("INSERT INTO level(kode,nama,urutan) VALUES ('1','Pra Tahsin',1)")
	guru := insert("INSERT INTO guru(nama,jenis_kelamin,is_aktif) VALUES ('Guru Planning Test','P',1)")
	insert(`INSERT INTO guru_teaching_levels(guru_id,level_kode,approved_by,approved_at)
		VALUES (?,?,?,CURRENT_TIMESTAMP)`, guru, "1", coordinator)
	insert(`INSERT INTO guru_availability(guru_id,hari,jam_mulai,jam_selesai,berlaku_mulai,confirmed_by,confirmed_at)
		VALUES (?,2,'19:00','22:00','2026-01-01',?,CURRENT_TIMESTAMP)`, guru, coordinator)
	insert(`INSERT INTO jadwal_penawaran
		(kode,hari,jam_mulai,jam_selesai,level,gender,is_buka,periode_mulai,created_at,updated_at)
		VALUES ('PLAN-ONE','2','19:30','21:00','1','P',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`)
	ids := make([]int64, 0, 8)
	for i := 1; i <= 8; i++ {
		id := insert(`INSERT INTO santri
			(nama,jenis_kelamin,angkatan,angkatan_kelas,kelas_kode,level,jadwal,tipe,frekuensi,
			 status,is_lengkap,kelas_id)
			VALUES (?,?,?,?,?,'1','Selasa malam','Reguler','1x/pekan','aktif',1,NULL)`,
			fmt.Sprintf("Santri Planning %d", i), "P", "2026-A", "2026-A", "R 1X")
		ids = append(ids, id)
		insert(`INSERT INTO onboarding_cases
			(santri_id,stage,registered_at,deadline_at,created_at,updated_at)
			VALUES (?,'matching_kelas','2026-10-01','2027-01-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, id)
		insert(`INSERT INTO leads(wa_chat_id,santri_id,jadwal_kode,verified_at,status,needs_review)
			VALUES (?,?, 'PLAN-ONE',CURRENT_TIMESTAMP,'won',0)`,
			fmt.Sprintf("planning-%d@wa.test", i), id)
	}
	id, err := q.CreateClassPlanWorkflow(context.Background(), admin, queries.ClassPlanWorkflowRow{
		Program: "reguler", Angkatan: "2026-A", Gender: "P", GuruID: guru, Frequency: 1, MinQuota: 8, Capacity: 10,
	}, []queries.ClassPlanWorkflowSlot{{Day: 2, Start: "19:30", End: "21:00"}}, ids, "1")
	require.NoError(t, err)
	for _, step := range []struct {
		from, to string
		actor    int64
	}{
		{"draft", "pending_coordinator", admin},
		{"pending_coordinator", "pending_admin", coordinator},
		{"pending_admin", "admin_approved", admin},
	} {
		ok, err := q.ClassPlanTransition(context.Background(), id, 1, step.from, step.to, step.actor)
		require.NoError(t, err)
		require.True(t, ok)
	}
	return activationFixture{db: db, q: q, planID: id, adminID: admin, coordinatorID: coordinator, studentIDs: ids}
}

func (f activationFixture) reserve(t *testing.T) {
	t.Helper()
	s := NewClassPlanWorkflowService(f.q)
	// Test through service, including live eligibility and availability checks.
	require.NoError(t, s.Action(context.Background(), f.planID, 1, f.adminID, "reserve", ""))
}

func (f activationFixture) count(t *testing.T, statement string, args ...any) int {
	t.Helper()
	var count int
	require.NoError(t, f.db.QueryRow(statement, args...).Scan(&count))
	return count
}

func TestPlanningActivationCreatesOneClassAndIsIdempotent(t *testing.T) {
	f := setupActivationFixture(t)
	f.reserve(t)
	s := NewClassPlanWorkflowService(f.q)
	classID, err := s.Activate(context.Background(), f.planID, 1, f.adminID)
	require.NoError(t, err)
	require.Positive(t, classID)
	require.Equal(t, 1, f.count(t, "SELECT COUNT(*) FROM kelas WHERE id=?", classID))
	require.Equal(t, 1, f.count(t, "SELECT COUNT(*) FROM kelas_jadwal_rutin WHERE kelas_id=?", classID))
	require.Equal(t, 8, f.count(t, "SELECT COUNT(*) FROM santri WHERE kelas_id=?", classID))
	require.Equal(t, 8, f.count(t, "SELECT COUNT(*) FROM class_plan_students WHERE plan_id=? AND status='committed'", f.planID))
	require.Equal(t, 1, f.count(t, "SELECT COUNT(*) FROM class_plans WHERE id=? AND status='activated' AND activated_kelas_id=?", f.planID, classID))
	require.Equal(t, 1, f.count(t, "SELECT COUNT(*) FROM kelas_allowed_levels WHERE kelas_id=? AND level_kode='1'", classID))
	require.Equal(t, 1, f.count(t, "SELECT COUNT(*) FROM class_plan_events WHERE plan_id=? AND event_type='planning.activated'", f.planID))
	var kelasJadwal, santriJadwal, level string
	require.NoError(t, f.db.QueryRow("SELECT jadwal FROM kelas WHERE id=?", classID).Scan(&kelasJadwal))
	require.NoError(t, f.db.QueryRow("SELECT jadwal,level FROM santri WHERE id=?", f.studentIDs[0]).Scan(&santriJadwal, &level))
	require.Equal(t, "Selasa 19:30-21:00", kelasJadwal)
	require.Equal(t, kelasJadwal, santriJadwal)
	require.Equal(t, "1", level, "individual level must not be overwritten")
	retryID, err := s.Activate(context.Background(), f.planID, 1, f.adminID)
	require.NoError(t, err)
	require.Equal(t, classID, retryID)
	require.Equal(t, 1, f.count(t, "SELECT COUNT(*) FROM kelas WHERE kunci_kelas=?", fmt.Sprintf("PLANNING:%d", f.planID)))
	require.Equal(t, 1, f.count(t, "SELECT COUNT(*) FROM class_plan_events WHERE plan_id=? AND event_type='planning.activated'", f.planID))
	// Onboarding only moves to masuk_kelas after the separately recorded actual start.
	require.Equal(t, 0, f.count(t, "SELECT COUNT(*) FROM onboarding_cases WHERE stage='masuk_kelas'"))
	// Unrelated profile updates must not reassign a planned-class student.
	santri, err := f.q.GetSantriByID(context.Background(), f.studentIDs[0])
	require.NoError(t, err)
	require.NoError(t, NewKelasEngineService(f.q).ProcessSantri(context.Background(), &santri))
	require.Equal(t, 1, f.count(t, "SELECT COUNT(*) FROM santri WHERE id=? AND kelas_id=?", f.studentIDs[0], classID))
	santri.Jadwal = "Rabu 19:30-21:00"
	err = NewKelasEngineService(f.q).ProcessSantri(context.Background(), &santri)
	require.ErrorContains(t, err, "eksplisit")
}

func TestPlanningActivationWithoutReservationOrCoordinatorRightsRejected(t *testing.T) {
	f := setupActivationFixture(t)
	s := NewClassPlanWorkflowService(f.q)
	_, err := s.Activate(context.Background(), f.planID, 1, f.adminID)
	require.ErrorIs(t, err, ErrPlanWorkflowConflict)
	f.reserve(t)
	_, err = s.Activate(context.Background(), f.planID, 1, f.coordinatorID)
	require.ErrorIs(t, err, ErrPlanWorkflowForbidden)
	_, err = s.Activate(context.Background(), f.planID, 2, f.adminID)
	require.ErrorIs(t, err, ErrPlanWorkflowConflict)
	require.Equal(t, 0, f.count(t, "SELECT COUNT(*) FROM kelas"))
	require.Equal(t, 8, f.count(t, "SELECT COUNT(*) FROM class_plan_students WHERE status='reserved'"))
}

func TestPlanningActivationRollsBackClassAndRosterOnRoutineFailure(t *testing.T) {
	f := setupActivationFixture(t)
	f.reserve(t)
	_, err := f.db.Exec(`CREATE TRIGGER stop_planning_routine BEFORE INSERT ON kelas_jadwal_rutin
		BEGIN SELECT RAISE(FAIL,'test routine failure'); END`)
	require.NoError(t, err)
	s := NewClassPlanWorkflowService(f.q)
	_, err = s.Activate(context.Background(), f.planID, 1, f.adminID)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "test routine failure"))
	require.Equal(t, 0, f.count(t, "SELECT COUNT(*) FROM kelas WHERE kunci_kelas=?", fmt.Sprintf("PLANNING:%d", f.planID)))
	require.Equal(t, 0, f.count(t, "SELECT COUNT(*) FROM santri WHERE kelas_id IS NOT NULL"))
	require.Equal(t, 8, f.count(t, "SELECT COUNT(*) FROM class_plan_students WHERE status='reserved'"))
	require.Equal(t, 0, f.count(t, "SELECT COUNT(*) FROM class_plan_events WHERE event_type='planning.activated'"))
	require.Equal(t, 1, f.count(t, "SELECT COUNT(*) FROM class_plans WHERE id=? AND status='pending_admin'", f.planID))
}

func TestPlanningActivationRejectsChangedEligibilityWithoutPartialWrite(t *testing.T) {
	f := setupActivationFixture(t)
	f.reserve(t)
	_, err := f.db.Exec("UPDATE santri SET frekuensi='2x/pekan' WHERE id=?", f.studentIDs[2])
	require.NoError(t, err)
	_, err = NewClassPlanWorkflowService(f.q).Activate(context.Background(), f.planID, 1, f.adminID)
	require.Error(t, err)
	require.Equal(t, 0, f.count(t, "SELECT COUNT(*) FROM kelas WHERE kunci_kelas=?", fmt.Sprintf("PLANNING:%d", f.planID)))
	require.Equal(t, 8, f.count(t, "SELECT COUNT(*) FROM class_plan_students WHERE status='reserved'"))
}
