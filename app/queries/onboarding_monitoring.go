package queries

import (
	"context"
	"database/sql"
	"time"
)

type OnboardingMonitoringCaseRow struct {
	ID              int64
	Stage           string
	PICUserID       sql.NullInt64
	RegisteredAt    time.Time
	DeadlineAt      time.Time
	ActualStartedAt sql.NullTime
	CancelledAt     sql.NullTime
}

const listOnboardingMonitoringCasesSQL = `SELECT id,stage,pic_user_id,registered_at,deadline_at,
 actual_started_at,cancelled_at FROM onboarding_cases ORDER BY id`

// All recorded onboarding cases are considered for KPI, including terminal cases.
func (q *Querier) ListOnboardingMonitoringCases(ctx context.Context) ([]OnboardingMonitoringCaseRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, listOnboardingMonitoringCasesSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OnboardingMonitoringCaseRow, 0)
	for rows.Next() {
		var r OnboardingMonitoringCaseRow
		if err := rows.Scan(&r.ID, &r.Stage, &r.PICUserID, &r.RegisteredAt,
			&r.DeadlineAt, &r.ActualStartedAt, &r.CancelledAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Class monitoring is not capped at the 200 entries shown in PR 3's UI.
func (q *Querier) ListAllOnboardingClassControls(ctx context.Context) ([]OnboardingClassControlRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, onboardingClassControlSelect+" ORDER BY k.id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OnboardingClassControlRow, 0)
	for rows.Next() {
		r, err := scanOnboardingClass(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Create milestone notifications for a specific user, or all users belonging
// to supplied roles. Both use the same partial unique index from migration 0092.
func (q *Querier) NotifyOnboardingRoles(ctx context.Context, roles []string, kind, title, message, url, referenceType string, referenceID int64, createdAt time.Time) (int64, error) {
	var inserted int64
	for _, role := range roles {
		result, err := q.Queries.db.ExecContext(ctx, `INSERT INTO notifications
   (user_id,type,title,message,action_url,reference_type,reference_id,created_at)
   SELECT id,?,?,?,?,?,?,? FROM users WHERE role=? AND id>0
   ON CONFLICT DO NOTHING`, kind, title, message, url, referenceType, referenceID, createdAt, role)
		if err != nil {
			return inserted, err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return inserted, err
		}
		inserted += n
	}
	return inserted, nil
}

func (q *Querier) NotifyOnboardingUser(ctx context.Context, userID int64, kind, title, message, url, referenceType string, referenceID int64, createdAt time.Time) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, `INSERT INTO notifications
  (user_id,type,title,message,action_url,reference_type,reference_id,created_at)
  SELECT id,?,?,?,?,?,?,? FROM users WHERE id=? AND role IN ('admin_kelas','super_admin')
  ON CONFLICT DO NOTHING`, kind, title, message, url, referenceType, referenceID, createdAt, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

type OnboardingUserNoticeRow struct {
	ID        int64
	Type      string
	Title     string
	Message   string
	ActionURL string
	ReadAt    sql.NullTime
	CreatedAt time.Time
}

func (q *Querier) ListOnboardingUserNotices(ctx context.Context, userID int64) ([]OnboardingUserNoticeRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `SELECT id,type,title,message,action_url,read_at,created_at
 FROM notifications WHERE user_id=? AND type LIKE 'onboarding_%'
 ORDER BY created_at DESC,id DESC LIMIT 30`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OnboardingUserNoticeRow, 0)
	for rows.Next() {
		var r OnboardingUserNoticeRow
		if err := rows.Scan(&r.ID, &r.Type, &r.Title, &r.Message, &r.ActionURL, &r.ReadAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Querier) MarkOnboardingUserNoticeRead(ctx context.Context, userID, noticeID int64, now time.Time) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, `UPDATE notifications SET read_at=?
 WHERE id=? AND user_id=? AND type LIKE 'onboarding_%' AND read_at IS NULL`, now, noticeID, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
