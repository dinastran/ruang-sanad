package queries

import (
	"context"
	"errors"
	"time"
)

// PlanningReadinessCounts reports source-system health without modifying any
// operational class, roster, teacher assignment, or schedule.
type PlanningReadinessCounts struct {
	AwaitingOnboarding          int64 `json:"awaiting_onboarding"`
	LegacyPreassigned           int64 `json:"legacy_preassigned"`
	CandidateDataComplete       int64 `json:"candidate_data_complete"`
	CRMPaymentVerified          int64 `json:"crm_payment_verified"`
	CRMVerificationUnknown      int64 `json:"crm_verification_unknown"`
	ActiveTeachers              int64 `json:"active_teachers"`
	ConfirmedAvailabilityGuru   int64 `json:"confirmed_availability_guru"`
	ApprovedTeachingLevelGuru   int64 `json:"approved_teaching_level_guru"`
	UsableTeacherCount          int64 `json:"usable_teacher_count"`
	ActiveClassesNoRoutineSlots int64 `json:"active_classes_no_routine_slots"`
	RegularClassesAboveMax      int64 `json:"regular_classes_above_max"`
}

const planningReadinessSQL = `
WITH awaiting AS (
	SELECT s.id, s.kelas_id, s.jenis_kelamin, s.level, s.jadwal,
		s.frekuensi, s.is_lengkap, oc.stage
	FROM onboarding_cases oc
	JOIN santri s ON s.id = oc.santri_id
	WHERE oc.stage IN ('onboarding_baru', 'matching_kelas', 'waiting_kuota')
	  AND s.status = 'aktif'
),
candidate AS (
	SELECT id, kelas_id FROM awaiting
	WHERE stage IN ('matching_kelas', 'waiting_kuota')
	AND is_lengkap = 1 AND jenis_kelamin IN ('L','P')
	AND TRIM(level) <> '' AND TRIM(jadwal) <> ''
	AND TRIM(frekuensi) <> ''
),
confirmed AS (
	SELECT c.id FROM candidate c
	WHERE EXISTS (
		SELECT 1 FROM leads l WHERE l.santri_id = c.id
		AND l.verified_at IS NOT NULL AND l.needs_review = 0
		AND l.status <> 'lost'
	)
),
availability AS (
	SELECT DISTINCT g.id
	FROM guru g
	JOIN guru_availability ga ON ga.guru_id = g.id
	WHERE g.is_aktif = 1 AND ga.confirmed_at IS NOT NULL
	AND ga.berlaku_mulai <= ?1
	AND (ga.berlaku_sampai IS NULL OR ga.berlaku_sampai >= ?1)
),
competent AS (
	SELECT DISTINCT g.id
	FROM guru g JOIN guru_teaching_levels tl ON tl.guru_id = g.id
	WHERE g.is_aktif = 1
)
SELECT
	(SELECT COUNT(*) FROM awaiting),
	(SELECT COUNT(*) FROM awaiting WHERE kelas_id IS NOT NULL),
	(SELECT COUNT(*) FROM candidate),
	(SELECT COUNT(*) FROM confirmed),
	(SELECT COUNT(*) FROM candidate) - (SELECT COUNT(*) FROM confirmed),
	(SELECT COUNT(*) FROM guru WHERE is_aktif = 1),
	(SELECT COUNT(*) FROM availability),
	(SELECT COUNT(*) FROM competent),
	(SELECT COUNT(*) FROM availability a JOIN competent c ON a.id = c.id),
	(SELECT COUNT(*) FROM kelas k
	 WHERE k.is_aktif = 1 AND k.guru_id IS NOT NULL
	 AND NOT EXISTS (SELECT 1 FROM kelas_jadwal_rutin r
		WHERE r.kelas_id = k.id AND r.is_aktif = 1
		AND r.berlaku_mulai <= ?1
		AND (r.berlaku_sampai IS NULL OR r.berlaku_sampai >= ?1))),
	(SELECT COUNT(*) FROM kelas k
	 WHERE k.is_aktif = 1 AND LOWER(k.tipe) IN ('reguler','regular')
	 AND k.kapasitas > 10)
`

// ReadPlanningReadiness uses a Jakarta-local as-of date passed by the service,
// so tests can be deterministic and UTC dates cannot shift reporting.
func (q *Querier) ReadPlanningReadiness(ctx context.Context, asOf string) (PlanningReadinessCounts, error) {
	var out PlanningReadinessCounts
	if d, err := time.Parse(time.DateOnly, asOf); err != nil || d.Format(time.DateOnly) != asOf {
		return out, errors.New("tanggal laporan planning tidak valid")
	}
	err := q.Queries.db.QueryRowContext(ctx, planningReadinessSQL, asOf).Scan(
		&out.AwaitingOnboarding, &out.LegacyPreassigned,
		&out.CandidateDataComplete, &out.CRMPaymentVerified, &out.CRMVerificationUnknown,
		&out.ActiveTeachers, &out.ConfirmedAvailabilityGuru,
		&out.ApprovedTeachingLevelGuru, &out.UsableTeacherCount,
		&out.ActiveClassesNoRoutineSlots, &out.RegularClassesAboveMax,
	)
	return out, err
}
