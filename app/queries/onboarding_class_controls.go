package queries

import (
	"context"
	"database/sql"
	"time"
)

// OnboardingClassControlRow joins only classes that have new onboarding cases.
// Historical classes without onboarding cases are never backfilled.
type OnboardingClassControlRow struct {
	KelasID             int64
	KelasNama           string
	Tipe                string
	Kapasitas           int64
	JumlahSantri        int64
	KandidatAktif       int64
	IsAktif             int64
	GuruID              sql.NullInt64
	Jadwal              string
	FirstCaseAt         time.Time
	TargetKuota         sql.NullInt64
	FirstCandidateAt    sql.NullTime
	VacancyAlertSentAt  sql.NullTime
	ExceptionApprovedBy sql.NullInt64
	ExceptionApprovedAt sql.NullTime
	ExceptionReason     string
}

const onboardingClassControlSelect = `SELECT
 k.id, k.nama_kelas, k.tipe, k.kapasitas,
 (SELECT COUNT(*) FROM santri ss WHERE ss.kelas_id=k.id AND ss.status IN ('aktif','cuti')) AS jumlah_santri,
 (SELECT COUNT(*) FROM onboarding_cases oc
  JOIN santri sc ON sc.id=oc.santri_id
  WHERE sc.kelas_id=k.id AND sc.status='aktif' AND oc.stage IN
  ('onboarding_baru','matching_kelas','waiting_kuota','siap_mulai')) AS kandidat_aktif,
 k.is_aktif, k.guru_id, k.jadwal, firstoc.registered_at,
 cc.target_kuota, cc.first_candidate_at, cc.vacancy_alert_sent_at,
 cc.exception_approved_by, cc.exception_approved_at, COALESCE(cc.exception_reason,'')
FROM kelas k
JOIN onboarding_cases firstoc ON firstoc.id=(
 SELECT oc2.id FROM onboarding_cases oc2
 JOIN santri sc2 ON sc2.id=oc2.santri_id
 WHERE sc2.kelas_id=k.id
 ORDER BY oc2.registered_at ASC, oc2.id ASC LIMIT 1
)
LEFT JOIN onboarding_class_controls cc ON cc.kelas_id=k.id`

func scanOnboardingClass(scan func(...any) error) (OnboardingClassControlRow, error) {
	var r OnboardingClassControlRow
	err := scan(&r.KelasID, &r.KelasNama, &r.Tipe, &r.Kapasitas,
		&r.JumlahSantri, &r.KandidatAktif, &r.IsAktif, &r.GuruID, &r.Jadwal, &r.FirstCaseAt,
		&r.TargetKuota, &r.FirstCandidateAt, &r.VacancyAlertSentAt,
		&r.ExceptionApprovedBy, &r.ExceptionApprovedAt, &r.ExceptionReason)
	return r, err
}

func (q *Querier) ListOnboardingClassControls(ctx context.Context) ([]OnboardingClassControlRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, onboardingClassControlSelect+" ORDER BY firstoc.registered_at DESC, k.id DESC LIMIT 200")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OnboardingClassControlRow, 0)
	for rows.Next() {
		item, err := scanOnboardingClass(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Querier) GetOnboardingClassControl(ctx context.Context, id int64) (OnboardingClassControlRow, error) {
	return scanOnboardingClass(q.Queries.db.QueryRowContext(ctx, onboardingClassControlSelect+" WHERE k.id=?", id).Scan)
}

// Target changes invalidate existing exceptions so approval is never reused for a new threshold.
func (q *Querier) CreateOnboardingQuota(ctx context.Context, id, target int64, anchor, now time.Time) error {
	_, err := q.Queries.db.ExecContext(ctx, `INSERT INTO onboarding_class_controls
 (kelas_id,target_kuota,first_candidate_at,created_at,updated_at) VALUES (?,?,?,?,?)`,
		id, target, anchor, now, now)
	return err
}
func (q *Querier) ChangeOnboardingQuota(ctx context.Context, id, target int64, now time.Time) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, `UPDATE onboarding_class_controls
 SET target_kuota=?,updated_at=?,exception_approved_by=NULL,
 exception_approved_at=NULL,exception_reason=''
 WHERE kelas_id=?`, target, now, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
func (q *Querier) MarkOnboardingVacancySent(ctx context.Context, id int64, now time.Time) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, `UPDATE onboarding_class_controls
 SET vacancy_alert_sent_at=?,updated_at=? WHERE kelas_id=? AND vacancy_alert_sent_at IS NULL`, now, now, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
func (q *Querier) SetOnboardingException(ctx context.Context, id, actor int64, reason string, now time.Time) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, `UPDATE onboarding_class_controls
 SET exception_approved_by=?,exception_approved_at=?,exception_reason=?,updated_at=?
 WHERE kelas_id=? AND exception_approved_at IS NULL`, actor, now, reason, now, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
func (q *Querier) RevokeOnboardingException(ctx context.Context, id int64, now time.Time) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, `UPDATE onboarding_class_controls
 SET exception_approved_by=NULL,exception_approved_at=NULL,exception_reason='',updated_at=?
 WHERE kelas_id=? AND exception_approved_at IS NOT NULL`, now, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
func (q *Querier) CreateOnboardingClassAction(ctx context.Context, id, actor int64, event, note string, now time.Time) error {
	_, err := q.Queries.db.ExecContext(ctx, `INSERT INTO onboarding_class_action_logs
 (kelas_id,actor_user_id,event_type,note,created_at) VALUES (?,?,?,?,?)`, id, actor, event, note, now)
	return err
}

type OnboardingClassActionRow struct {
	ID        int64
	EventType string
	Note      string
	CreatedAt time.Time
	ActorName string
}

func (q *Querier) ListOnboardingClassActions(ctx context.Context, id int64) ([]OnboardingClassActionRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `SELECT a.id,a.event_type,a.note,a.created_at,
 COALESCE(u.name,'Sistem') FROM onboarding_class_action_logs a
 LEFT JOIN users u ON u.id=a.actor_user_id
 WHERE a.kelas_id=? ORDER BY a.id DESC LIMIT 100`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OnboardingClassActionRow, 0)
	for rows.Next() {
		var r OnboardingClassActionRow
		if err := rows.Scan(&r.ID, &r.EventType, &r.Note, &r.CreatedAt, &r.ActorName); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
