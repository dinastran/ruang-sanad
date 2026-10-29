package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func onboardingFixture(t *testing.T) (*sql.DB, *OnboardingService, int64, int64, int64) {
	t.Helper()
	db, q, santriService, actorID := setupSantriIntegrityService(t)
	santri, err := santriService.Create(validCreateSantriRequest(), actorID)
	require.NoError(t, err)
	var caseID int64
	require.NoError(t, db.QueryRow("SELECT id FROM onboarding_cases WHERE santri_id=?", santri.ID).Scan(&caseID))
	r, err := db.Exec("INSERT INTO users(email,name,role) VALUES('pic@example.com','PIC Kelas','admin_kelas')")
	require.NoError(t, err)
	picID, err := r.LastInsertId()
	require.NoError(t, err)
	return db, NewOnboardingService(q), caseID, picID, actorID
}

func TestOnboardingKanbanPICValidationAndAudit(t *testing.T) {
	db, s, id, picID, actorID := onboardingFixture(t)
	ctx := context.Background()
	require.ErrorIs(t, s.AssignPIC(ctx, id, actorID, actorID), ErrOnboardingInvalid, "CS cannot be assigned as PIC")
	require.NoError(t, s.AssignPIC(ctx, id, picID, actorID))
	var current int64
	require.NoError(t, db.QueryRow("SELECT pic_user_id FROM onboarding_cases WHERE id=?", id).Scan(&current))
	require.Equal(t, picID, current)
	require.ErrorIs(t, s.AssignPIC(ctx, id, picID, actorID), ErrOnboardingInvalid)
	activity, err := s.Activity(ctx, id)
	require.NoError(t, err)
	require.Len(t, activity, 2)
	require.Equal(t, "onboarding.pic_changed", activity[0].EventType)
	board, err := s.Board(ctx, 1)
	require.NoError(t, err)
	require.EqualValues(t, 1, board.Counts["onboarding_baru"])
	require.EqualValues(t, 1, board.Total)
	require.Equal(t, picID, board.Cases[0].PICUserID)
}

func TestOnboardingKanbanMoveRequiresPICAndPreservesSLA(t *testing.T) {
	db, s, id, picID, actorID := onboardingFixture(t)
	ctx := context.Background()
	var beforeRegistered, beforeDeadline time.Time
	require.NoError(t, db.QueryRow("SELECT registered_at,deadline_at FROM onboarding_cases WHERE id=?", id).Scan(&beforeRegistered, &beforeDeadline))
	require.ErrorIs(t, s.Move(ctx, id, actorID, "matching_kelas", "", ""), ErrOnboardingInvalid)
	require.NoError(t, s.AssignPIC(ctx, id, picID, actorID))
	require.ErrorIs(t, s.Move(ctx, id, actorID, "waiting_kuota", "", ""), ErrOnboardingInvalid, "skip not allowed")
	require.NoError(t, s.Move(ctx, id, actorID, "matching_kelas", "", ""))
	require.ErrorIs(t, s.Move(ctx, id, actorID, "onboarding_baru", "", ""), ErrOnboardingInvalid, "backward requires reason")
	require.NoError(t, s.Move(ctx, id, actorID, "onboarding_baru", "Perlu koreksi profil", ""))
	var stage string
	var registered, deadline time.Time
	require.NoError(t, db.QueryRow("SELECT stage,registered_at,deadline_at FROM onboarding_cases WHERE id=?", id).Scan(&stage, &registered, &deadline))
	require.Equal(t, "onboarding_baru", stage)
	require.True(t, beforeRegistered.Equal(registered))
	require.True(t, beforeDeadline.Equal(deadline))
	history, err := s.Activity(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "Perlu koreksi profil", history[0].Note)
	require.Equal(t, "onboarding_baru", history[0].ToStage)
}

func TestOnboardingKanbanWaitingRequiresExistingClassAndReadyRequiresQuota(t *testing.T) {
	db, s, id, picID, actorID := onboardingFixture(t)
	ctx := context.Background()
	require.NoError(t, s.AssignPIC(ctx, id, picID, actorID))
	require.NoError(t, s.Move(ctx, id, actorID, "matching_kelas", "", ""))
	require.ErrorIs(t, s.Move(ctx, id, actorID, "waiting_kuota", "", ""), ErrOnboardingInvalid)
	// The santri's original kelas_id is the only authoritative assignment.
	// No class-control record is created by PR 2.
	var controls int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM onboarding_class_controls").Scan(&controls))
	require.Zero(t, controls)
}

func TestOnboardingKanbanCancelTerminalAndAuditRollback(t *testing.T) {
	db, s, id, picID, actorID := onboardingFixture(t)
	ctx := context.Background()
	require.ErrorIs(t, s.Move(ctx, id, actorID, "batal", "", ""), ErrOnboardingInvalid)
	require.NoError(t, s.Move(ctx, id, actorID, "batal", "Permintaan santri", ""))
	require.ErrorIs(t, s.Move(ctx, id, actorID, "matching_kelas", "", ""), ErrOnboardingInvalid)
	require.ErrorIs(t, s.AssignPIC(ctx, id, picID, actorID), ErrOnboardingInvalid)
	var reason string
	require.NoError(t, db.QueryRow("SELECT cancellation_reason FROM onboarding_cases WHERE id=?", id).Scan(&reason))
	require.Equal(t, "Permintaan santri", reason)
}

func TestOnboardingKanbanAuditFailureRollsBackPIC(t *testing.T) {
	db, s, id, picID, actorID := onboardingFixture(t)
	_, err := db.Exec(`CREATE TRIGGER fail_onboarding_audit BEFORE INSERT ON onboarding_activity_logs
    BEGIN SELECT RAISE(ABORT, 'audit unavailable'); END`)
	require.NoError(t, err)
	err = s.AssignPIC(context.Background(), id, picID, actorID)
	require.ErrorContains(t, err, "audit unavailable")
	var pic sql.NullInt64
	require.NoError(t, db.QueryRow("SELECT pic_user_id FROM onboarding_cases WHERE id=?", id).Scan(&pic))
	require.False(t, pic.Valid)
}

func TestOnboardingKanbanDateValidation(t *testing.T) {
	stage := onboardingStageIndex("siap_mulai")
	require.Equal(t, 3, stage)
	require.Equal(t, -1, onboardingStageIndex("unknown"))
	_, err := onboardingReason(string(make([]byte, 301)))
	require.True(t, errors.Is(err, ErrOnboardingInvalid))
}


func TestOnboardingKanbanFullStageSequenceRequiresQuotaAndActualStart(t *testing.T) {
    db, service, caseID, picID, actorID := onboardingFixture(t)
    ctx := context.Background()

    _, err := db.Exec("INSERT INTO guru (nama, jenis_kelamin, is_aktif) VALUES ('Guru Onboarding', 'L', 1)")
    require.NoError(t,err)
    var guruID int64
    require.NoError(t,db.QueryRow("SELECT id FROM guru WHERE nama='Guru Onboarding'").Scan(&guruID))
    result, err := db.Exec(`INSERT INTO kelas (
        kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi,
        jadwal, sub_index, nama_kelas, guru_id, kapasitas, jumlah_santri, is_aktif
    ) VALUES ('PR2-TEST','AKA38','Reguler','L','Dasar','1x','Senin',1,'Kelas Onboarding',?,10,0,1)`,guruID)
    require.NoError(t,err)
    kelasID, err := result.LastInsertId()
    require.NoError(t,err)

    _, err = db.Exec(`UPDATE santri SET kelas_id=?, is_lengkap=1
        WHERE id=(SELECT santri_id FROM onboarding_cases WHERE id=?)`,kelasID,caseID)
    require.NoError(t,err)

    require.NoError(t,service.AssignPIC(ctx,caseID,picID,actorID))
    require.NoError(t,service.Move(ctx,caseID,actorID,"matching_kelas","",""))
    require.NoError(t,service.Move(ctx,caseID,actorID,"waiting_kuota","",""))
    require.ErrorIs(t,service.Move(ctx,caseID,actorID,"siap_mulai","",""),ErrOnboardingInvalid,
        "PR 3 quota must be explicitly configured")
    _, err = db.Exec(`INSERT INTO onboarding_class_controls (
        kelas_id,target_kuota,first_candidate_at,created_at,updated_at
    ) VALUES (?,2,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,kelasID)
    require.NoError(t,err)
    require.ErrorIs(t,service.Move(ctx,caseID,actorID,"siap_mulai","",""),ErrOnboardingInvalid,
        "below-target class must not become ready")
    _, err = db.Exec("UPDATE onboarding_class_controls SET target_kuota=1 WHERE kelas_id=?",kelasID)
    require.NoError(t,err)
    require.NoError(t,service.Move(ctx,caseID,actorID,"siap_mulai","",""))
    require.ErrorIs(t,service.Move(ctx,caseID,actorID,"masuk_kelas","",""),ErrOnboardingInvalid,
        "actual start date is mandatory")
    require.ErrorIs(t,service.Move(ctx,caseID,actorID,"masuk_kelas","","2099-01-01"),ErrOnboardingInvalid)
    today := time.Now().In(onboardingWIB).Format(time.DateOnly)
    require.NoError(t,service.Move(ctx,caseID,actorID,"masuk_kelas","",""+today))
    var stage string
    var actual sql.NullTime
    require.NoError(t,db.QueryRow("SELECT stage,actual_started_at FROM onboarding_cases WHERE id=?",caseID).Scan(&stage,&actual))
    require.Equal(t,"masuk_kelas",stage)
    require.True(t,actual.Valid)
    require.ErrorIs(t,service.Move(ctx,caseID,actorID,"siap_mulai","Koreksi",""),ErrOnboardingInvalid)
    history,err:=service.Activity(ctx,caseID)
    require.NoError(t,err)
    require.Equal(t,"masuk_kelas",history[0].ToStage)
}
