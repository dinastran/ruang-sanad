package queries

import (
	"context"
	"database/sql"
)

// ClassPlanExceptionReview is immutable request evidence plus a mutable
// decision. All callers use a transaction-scoped Querier.
type ClassPlanExceptionReview struct {
	ID           int64
	PlanID       int64
	Status       string
	Reason       string
	RequestedBy  int64
	DecidedBy    sql.NullInt64
	DecisionNote string
}

func (q *Querier) ClassPlanLatestException(ctx context.Context, planID int64) (ClassPlanExceptionReview, error) {
	var r ClassPlanExceptionReview
	err := q.Queries.db.QueryRowContext(ctx, `SELECT id,plan_id,status,reason,requested_by,decided_by,decision_note
		FROM class_plan_exceptions WHERE plan_id=? ORDER BY id DESC LIMIT 1`, planID).Scan(
		&r.ID, &r.PlanID, &r.Status, &r.Reason, &r.RequestedBy, &r.DecidedBy, &r.DecisionNote)
	return r, err
}

// An exception is not a waiver of gender, eligibility, schedule, teacher
// availability, or capacity rules. The only waived check is MIN_QUOTA_NOT_MET.
func (q *Querier) ClassPlanHasApprovedQuotaException(ctx context.Context, planID int64) (bool, error) {
	var approved bool
	err := q.Queries.db.QueryRowContext(ctx, `SELECT EXISTS(
		SELECT 1 FROM class_plan_exceptions e
		WHERE e.plan_id=? AND e.status='approved'
		AND e.decided_by IS NOT NULL AND e.decided_by<>e.requested_by
		AND length(trim(e.decision_note))>0
		AND e.id=(SELECT MAX(id) FROM class_plan_exceptions WHERE plan_id=?)
	)`, planID, planID).Scan(&approved)
	return approved, err
}

func (q *Querier) ClassPlanRequestQuotaException(ctx context.Context, planID, actorID int64, reason string) error {
	_, err := q.Queries.db.ExecContext(ctx, `INSERT INTO class_plan_exceptions
		(plan_id,status,reason,requested_by)
		VALUES (?,'requested',?,?)`, planID, reason, actorID)
	return err
}

// ClassPlanDecideQuotaException CAS ensures an exception cannot be decided
// twice, and cannot be decided by its requester.
func (q *Querier) ClassPlanDecideQuotaException(ctx context.Context, planID, exceptionID, actorID, version int64, decision, note string) (bool, error) {
	result, err := q.Queries.db.ExecContext(ctx, `UPDATE class_plan_exceptions
		SET status=?,decided_by=?,decided_at=CURRENT_TIMESTAMP,decision_note=?
		WHERE id=? AND plan_id=? AND status='requested'
		AND requested_by<>?
		AND id=(SELECT MAX(id) FROM class_plan_exceptions WHERE plan_id=?)
		AND EXISTS(SELECT 1 FROM class_plans p WHERE p.id=? AND p.status='pending_exception'
			AND p.version=? AND p.coordinator_confirmed_by IS NULL AND p.admin_approved_by IS NULL)`,
		decision, actorID, note, exceptionID, planID, actorID, planID, planID, version)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

func (q *Querier) ClassPlanTransitionQuotaDecision(ctx context.Context, planID, version, exceptionID int64, approved bool) (bool, error) {
	want := "approved"
	target := "pending_coordinator"
	if !approved {
		want = "rejected"
		target = "blocked"
	}
	result, err := q.Queries.db.ExecContext(ctx, `UPDATE class_plans
		SET status=?,updated_at=CURRENT_TIMESTAMP
		WHERE id=? AND version=? AND status='pending_exception'
		AND coordinator_confirmed_by IS NULL AND admin_approved_by IS NULL
		AND EXISTS(SELECT 1 FROM class_plan_exceptions e
			WHERE e.id=? AND e.plan_id=class_plans.id AND e.status=?
			AND e.decided_by IS NOT NULL AND e.decided_by<>e.requested_by
			AND length(trim(e.decision_note))>0)`,
		target, planID, version, exceptionID, want)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}
