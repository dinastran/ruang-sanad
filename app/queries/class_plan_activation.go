package queries

import (
	"context"
	"database/sql"
	"fmt"
)

// CreateActivatedPlanningClass is called only within the activation service's
// transaction after re-validating approvals, roster and teacher availability.
func (q *Querier) CreateActivatedPlanningClass(ctx context.Context, plan ClassPlanWorkflowRow, level, schedule string) (int64, error) {
	key := fmt.Sprintf("PLANNING:%d", plan.ID)
	name := fmt.Sprintf("%s | Reguler | %s | %s | Plan #%d", plan.Gender, level, plan.Angkatan, plan.ID)
	frequency := fmtClassPlanFrequency(plan.Frequency)
	result, err := q.Queries.db.ExecContext(ctx, `INSERT INTO kelas
		(kunci_kelas,angkatan,tipe,jenis_kelamin,level,frekuensi,jadwal,
		 sub_index,nama_kelas,guru_id,kapasitas,jumlah_santri,created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		key, plan.Angkatan, "Reguler", plan.Gender, level, frequency, schedule,
		1, name, plan.GuruID, plan.Capacity, 0)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (q *Querier) InsertPlanningClassRoutine(ctx context.Context, classID, actorID int64, day int, start, effectiveDate string) error {
	_, err := q.Queries.db.ExecContext(ctx, `INSERT INTO kelas_jadwal_rutin
		(kelas_id,hari,jam_mulai,berlaku_mulai,is_aktif,dibuat_oleh)
		VALUES (?,?,?,?,1,?)`, classID, day, start, effectiveDate, actorID)
	return err
}

func (q *Querier) InsertPlanningClassLevel(ctx context.Context, classID int64, level string) error {
	_, err := q.Queries.db.ExecContext(ctx,
		"INSERT INTO kelas_allowed_levels(kelas_id,level_kode) VALUES (?,?)", classID, level)
	return err
}

// Move the reservation to committed before changing kelas_id: trigger 0099
// intentionally blocks a kelas_id mutation while a candidate is reserved.
func (q *Querier) CommitPlanningReservations(ctx context.Context, planID int64) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, `UPDATE class_plan_students
		SET status='committed' WHERE plan_id=? AND status='reserved'`, planID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// A conditional update makes legacy or concurrent changes visible as a
// zero-row conflict. Academic level and original CRM offer are preserved.
func (q *Querier) AssignActivatedPlanningStudent(ctx context.Context, planID, studentID, classID int64, gender, cohort, level, frequency, schedule string) (bool, error) {
	result, err := q.Queries.db.ExecContext(ctx, `UPDATE santri SET kelas_id=?,jadwal=?,
		updated_at=CURRENT_TIMESTAMP
		WHERE id=? AND kelas_id IS NULL AND status='aktif' AND is_lengkap=1
			AND jenis_kelamin=? AND angkatan_kelas=? AND level=? AND frekuensi=?
			AND EXISTS(SELECT 1 FROM class_plan_students ps
				WHERE ps.plan_id=? AND ps.santri_id=santri.id AND ps.status='committed'
				AND ps.level_kode=?)`,
		classID, schedule, studentID, gender, cohort, level, frequency, planID, level)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (q *Querier) SetPlanningClassRosterCount(ctx context.Context, classID, total int64) error {
	result, err := q.Queries.db.ExecContext(ctx,
		"UPDATE kelas SET jumlah_santri=? WHERE id=?", total, classID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// Final CAS is idempotent only once the entire transaction is committed.
func (q *Querier) MarkClassPlanActivated(ctx context.Context, planID, version, classID, total int64) (bool, error) {
	result, err := q.Queries.db.ExecContext(ctx, `UPDATE class_plans SET
		status='activated',activated_kelas_id=?,updated_at=CURRENT_TIMESTAMP
		WHERE id=? AND version=? AND status='pending_admin'
			AND target_kelas_id IS NULL AND activated_kelas_id IS NULL
			AND coordinator_confirmed_by IS NOT NULL
			AND coordinator_confirmed_version=version
			AND admin_approved_by IS NOT NULL
			AND (SELECT COUNT(*) FROM class_plan_students WHERE plan_id=class_plans.id AND status='committed')=?
			AND NOT EXISTS(SELECT 1 FROM class_plan_students WHERE plan_id=class_plans.id AND status='reserved')`,
		classID, planID, version, total)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// A planned class already activated by this workflow must not be silently
// relocated by ordinary legacy ProcessSantri when unrelated fields are edited.
// Explicit PindahkanSantri remains a separate, user-directed action.
func (q *Querier) IsActivatedClassForSantri(ctx context.Context, studentID, classID int64) (bool, error) {
	var valid bool
	err := q.Queries.db.QueryRowContext(ctx, `SELECT EXISTS(
		SELECT 1 FROM class_plans p
		JOIN class_plan_students ps ON ps.plan_id=p.id
		WHERE p.status='activated' AND p.activated_kelas_id=?
		  AND ps.santri_id=? AND ps.status='committed'
	)`, classID, studentID).Scan(&valid)
	return valid, err
}
