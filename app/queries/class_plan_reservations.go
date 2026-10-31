package queries

import "context"

// ReserveApprovedClassPlanCandidates runs inside the caller's transaction.
// The partial unique index on class_plan_students(santri_id) enforces
// at most one live reservation even when plans are submitted concurrently.
func (q *Querier) ReserveApprovedClassPlanCandidates(ctx context.Context, planID, version int64) (int64,error) {
	const statement = `UPDATE class_plan_students SET status='reserved',schedule_approved=1
	WHERE plan_id=? AND status='candidate'
	AND EXISTS(
		SELECT 1 FROM class_plans p WHERE p.id=class_plan_students.plan_id
		AND p.version=? AND p.status='pending_admin'
		AND p.target_kelas_id IS NULL
		AND p.coordinator_confirmed_by IS NOT NULL
		AND p.coordinator_confirmed_version=p.version
		AND p.admin_approved_by IS NOT NULL
	)
	AND EXISTS(
		SELECT 1 FROM santri s WHERE s.id=class_plan_students.santri_id
		AND s.kelas_id IS NULL AND s.status='aktif'
	)`
	result,err:=q.Queries.db.ExecContext(ctx,statement,planID,version)
	if err!=nil{return 0,err}
	return result.RowsAffected()
}

func (q *Querier) ClassPlanReservationCount(ctx context.Context,planID int64) (int64,error) {
	var count int64
	err:=q.Queries.db.QueryRowContext(ctx,`SELECT COUNT(*) FROM class_plan_students WHERE plan_id=? AND status='reserved'`,planID).Scan(&count)
	return count,err
}

func (q *Querier) CancelApprovedPlan(ctx context.Context,planID,version int64) (bool,error) {
	result,err:=q.Queries.db.ExecContext(ctx,`UPDATE class_plans
		SET status='cancelled',updated_at=CURRENT_TIMESTAMP
		WHERE id=? AND version=? AND status='pending_admin'
		AND admin_approved_by IS NOT NULL AND coordinator_confirmed_by IS NOT NULL
		AND coordinator_confirmed_version=version
		AND activated_kelas_id IS NULL`,planID,version)
	if err!=nil{return false,err}
	n,err:=result.RowsAffected()
	return n==1,err
}

func (q *Querier) ReleaseClassPlanReservations(ctx context.Context,planID int64) error {
	_,err:=q.Queries.db.ExecContext(ctx,`UPDATE class_plan_students SET status='released'
		WHERE plan_id=? AND status IN ('candidate','reserved')`,planID)
	return err
}

// HasLiveClassPlanReservation is useful for error reporting in legacy entry
// points. The actual integrity guard is the SQLite trigger migration 0095.
func (q *Querier) HasLiveClassPlanReservation(ctx context.Context,santriID int64) (bool,error) {
	var found bool
	err:=q.Queries.db.QueryRowContext(ctx,`SELECT EXISTS(
		SELECT 1 FROM class_plan_students ps JOIN class_plans p ON p.id=ps.plan_id
		WHERE ps.santri_id=? AND ps.status='reserved' AND p.status='pending_admin'
	)`,santriID).Scan(&found)
	return found,err
}
