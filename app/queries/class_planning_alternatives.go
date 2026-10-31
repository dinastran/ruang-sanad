package queries

import "context"

// PlanningApprovedAlternative is an explicitly approved, currently offered
// alternative. It does not represent additional demand or a reserved seat.
type PlanningApprovedAlternative struct {
	SantriID   int64
	OfferCode  string
	OfferDays  string
	OfferStart string
	OfferEnd   string
}

const planningApprovedAlternativesSQL = `SELECT DISTINCT a.santri_id,a.jadwal_kode,j.hari,j.jam_mulai,j.jam_selesai
FROM santri_schedule_alternatives a
JOIN santri s ON s.id=a.santri_id
JOIN onboarding_cases oc ON oc.santri_id=s.id
JOIN jadwal_penawaran j ON j.kode=a.jadwal_kode
WHERE a.status='approved'
  AND a.consent_confirmed_at IS NOT NULL
  AND a.consent_channel IN ('whatsapp','phone','other')
  AND TRIM(a.consent_note)<>''
  AND s.status='aktif' AND s.kelas_id IS NULL AND s.is_lengkap=1
  AND s.jenis_kelamin IN ('L','P')
  AND TRIM(s.level)<>'' AND TRIM(s.angkatan_kelas)<>''
  AND oc.stage IN ('matching_kelas','waiting_kuota')
  AND j.is_buka=1
  AND (j.gender='' OR j.gender=s.jenis_kelamin)
  AND (j.level='' OR j.level=s.level)
  AND EXISTS (
     SELECT 1 FROM leads l WHERE l.santri_id=s.id
       AND l.verified_at IS NOT NULL AND l.needs_review=0
       AND l.status IN ('open','won')
  )
  AND NOT EXISTS (
     SELECT 1 FROM leads l WHERE l.santri_id=s.id AND l.jadwal_kode=a.jadwal_kode
  )
  AND NOT EXISTS (
     SELECT 1 FROM class_plan_students ps WHERE ps.santri_id=s.id
       AND ps.status IN ('reserved','committed')
  )
ORDER BY a.santri_id,a.jadwal_kode
LIMIT 1001`

// ListPlanningApprovedAlternatives never writes or changes payment status.
// Its result must be intersected with authoritative primary demand before use.
func (q *Querier) ListPlanningApprovedAlternatives(ctx context.Context) ([]PlanningApprovedAlternative, error) {
	rows, err := q.Queries.db.QueryContext(ctx, planningApprovedAlternativesSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PlanningApprovedAlternative, 0)
	for rows.Next() {
		var r PlanningApprovedAlternative
		if err := rows.Scan(&r.SantriID, &r.OfferCode, &r.OfferDays, &r.OfferStart, &r.OfferEnd); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
