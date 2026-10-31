package queries

import (
	"context"
	"database/sql"
)

type PlanningDemand struct {
	SantriID    int64
	Angkatan   string
	Gender     string
	Level      string
	Frequency  string
	OfferCode  string
	OfferDays  string
	OfferStart string
	OfferEnd   string
}

type PlanningExistingClass struct {
	ID        int64
	Angkatan  string
	Gender    string
	Level     string
	Frequency string
	Capacity  int64
	Occupancy int64
	GuruID    int64
	TeacherGender string
}

type PlanningClassSlot struct {
	KelasID int64
	Day     int64
	Start   string
}

type PlanningTeacherWindow struct {
	GuruID    int64
	Gender    string
	Status    string
	Day       int64
	Start     string
	End       string
}

type PlanningTeacherLevel struct {
	GuruID int64
	Level  string
}

type PlanningTeacherBusy struct {
	GuruID int64
	Day    int64
	Start  string
	End    string
}

type PlanningWorkloadLimit struct {
	GuruID    int64
	MaxMinutes sql.NullInt64
}

const planningDemandSQL = `SELECT s.id, s.angkatan_kelas, s.jenis_kelamin, s.level,
	s.frekuensi, l.jadwal_kode, jp.hari, jp.jam_mulai, jp.jam_selesai
FROM onboarding_cases oc
JOIN santri s ON s.id = oc.santri_id
JOIN leads l ON l.santri_id = s.id
JOIN jadwal_penawaran jp ON jp.kode = l.jadwal_kode
WHERE oc.stage IN ('matching_kelas','waiting_kuota')
	AND s.status = 'aktif' AND s.kelas_id IS NULL
	AND s.is_lengkap = 1 AND s.jenis_kelamin IN ('L','P')
	AND TRIM(s.level) != '' AND TRIM(s.angkatan_kelas) != ''
	AND l.verified_at IS NOT NULL AND l.needs_review = 0
	AND l.status IN ('open','won')
	AND (jp.gender = '' OR jp.gender = s.jenis_kelamin)
	AND NOT EXISTS (SELECT 1 FROM class_plan_students ps WHERE ps.santri_id = s.id AND ps.status IN ('reserved','committed'))
ORDER BY oc.deadline_at ASC, s.id ASC
LIMIT 501`

// ListPlanningDemand is deliberately conservative. Unverified or unknown manual
// payments, ambiguous free-text schedules and already-assigned students are
// excluded from recommendation counts rather than assumed eligible.
func (q *Querier) ListPlanningDemand(ctx context.Context) ([]PlanningDemand, error) {
	rows, err := q.Queries.db.QueryContext(ctx, planningDemandSQL)
	if err != nil { return nil, err }
	defer rows.Close()
	out := make([]PlanningDemand, 0)
	for rows.Next() {
		var r PlanningDemand
		if err := rows.Scan(&r.SantriID, &r.Angkatan, &r.Gender, &r.Level,
			&r.Frequency, &r.OfferCode, &r.OfferDays, &r.OfferStart, &r.OfferEnd); err != nil { return nil, err }
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Querier) ListPlanningExistingClasses(ctx context.Context) ([]PlanningExistingClass, error) {
	const statement = `SELECT k.id,k.angkatan,k.jenis_kelamin,k.level,k.frekuensi,k.kapasitas,
	(SELECT COUNT(*) FROM santri s WHERE s.kelas_id = k.id AND s.status IN ('aktif','cuti')),
	k.guru_id,g.jenis_kelamin
	FROM kelas k JOIN guru g ON g.id=k.guru_id AND g.is_aktif=1
	WHERE k.is_aktif=1 AND LOWER(k.tipe) IN ('reguler','regular')
	AND k.guru_id IS NOT NULL
	AND NOT EXISTS (SELECT 1 FROM pertemuan p WHERE p.kelas_id=k.id)
	ORDER BY k.id`
	rows, err := q.Queries.db.QueryContext(ctx, statement)
	if err != nil { return nil, err }
	defer rows.Close()
	out := make([]PlanningExistingClass, 0)
	for rows.Next() {
		var r PlanningExistingClass
		if err := rows.Scan(&r.ID,&r.Angkatan,&r.Gender,&r.Level,&r.Frequency,
			&r.Capacity,&r.Occupancy,&r.GuruID,&r.TeacherGender); err != nil { return nil, err }
		out = append(out,r)
	}
	return out, rows.Err()
}

func (q *Querier) ListPlanningCurrentClassSlots(ctx context.Context, asOf string) ([]PlanningClassSlot,error) {
	const statement = `SELECT r.kelas_id,r.hari,r.jam_mulai
	FROM kelas_jadwal_rutin r
	JOIN kelas k ON k.id=r.kelas_id
	WHERE k.is_aktif=1 AND r.is_aktif=1 AND r.berlaku_mulai<=?
	AND (r.berlaku_sampai IS NULL OR r.berlaku_sampai>=?)
	ORDER BY r.kelas_id,r.hari,r.jam_mulai`
	rows,err:=q.Queries.db.QueryContext(ctx,statement,asOf,asOf)
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make([]PlanningClassSlot,0)
	for rows.Next(){
		var r PlanningClassSlot
		if err:=rows.Scan(&r.KelasID,&r.Day,&r.Start);err!=nil{return nil,err}
		out=append(out,r)
	}
	return out,rows.Err()
}

func (q *Querier) ListPlanningTeacherWindows(ctx context.Context, asOf string) ([]PlanningTeacherWindow,error) {
	const statement = `SELECT g.id,g.jenis_kelamin,g.status,a.hari,a.jam_mulai,a.jam_selesai
	FROM guru_availability a
	JOIN guru g ON g.id=a.guru_id
	WHERE g.is_aktif=1 AND a.confirmed_at IS NOT NULL
	AND a.berlaku_mulai<=? AND (a.berlaku_sampai IS NULL OR a.berlaku_sampai>=?)
	ORDER BY g.id,a.hari,a.jam_mulai`
	rows,err:=q.Queries.db.QueryContext(ctx,statement,asOf,asOf)
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make([]PlanningTeacherWindow,0)
	for rows.Next(){
		var r PlanningTeacherWindow
		if err:=rows.Scan(&r.GuruID,&r.Gender,&r.Status,&r.Day,&r.Start,&r.End);err!=nil{return nil,err}
		out=append(out,r)
	}
	return out,rows.Err()
}

func (q *Querier) ListPlanningTeacherLevels(ctx context.Context) ([]PlanningTeacherLevel,error) {
	rows,err:=q.Queries.db.QueryContext(ctx,"SELECT guru_id,level_kode FROM guru_teaching_levels ORDER BY guru_id,level_kode")
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make([]PlanningTeacherLevel,0)
	for rows.Next(){
		var r PlanningTeacherLevel
		if err:=rows.Scan(&r.GuruID,&r.Level);err!=nil{return nil,err}
		out=append(out,r)
	}
	return out,rows.Err()
}

func (q *Querier) ListPlanningTeacherBusy(ctx context.Context, asOf string) ([]PlanningTeacherBusy,error) {
	// Active routine meetings are 90 minutes. Confirmed plans with no
	// operational class also reserve the indicated intervals.
	const statement = `SELECT k.guru_id,r.hari,r.jam_mulai,
	substr(time(r.jam_mulai,'+90 minutes'),1,5) AS jam_selesai
	FROM kelas_jadwal_rutin r JOIN kelas k ON k.id=r.kelas_id
	WHERE k.is_aktif=1 AND k.guru_id IS NOT NULL AND r.is_aktif=1
	AND r.berlaku_mulai<=? AND (r.berlaku_sampai IS NULL OR r.berlaku_sampai>=?)
	UNION ALL
	SELECT p.guru_id,s.hari,s.jam_mulai,s.jam_selesai
	FROM class_plans p JOIN class_plan_slots s ON s.plan_id=p.id
	WHERE p.status='pending_admin' AND p.guru_id IS NOT NULL
	ORDER BY 1,2,3`
	rows,err:=q.Queries.db.QueryContext(ctx,statement,asOf,asOf)
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make([]PlanningTeacherBusy,0)
	for rows.Next(){
		var r PlanningTeacherBusy
		if err:=rows.Scan(&r.GuruID,&r.Day,&r.Start,&r.End);err!=nil{return nil,err}
		out=append(out,r)
	}
	return out,rows.Err()
}

func (q *Querier) ListPlanningUnsafeTeacherIDs(ctx context.Context, asOf string) (map[int64]bool,error) {
	const statement = `SELECT DISTINCT k.guru_id FROM kelas k
	WHERE k.is_aktif=1 AND k.guru_id IS NOT NULL
	AND NOT EXISTS (
		SELECT 1 FROM kelas_jadwal_rutin r
		WHERE r.kelas_id=k.id AND r.is_aktif=1
		AND r.berlaku_mulai<=? AND (r.berlaku_sampai IS NULL OR r.berlaku_sampai>=?)
	)`
	rows,err:=q.Queries.db.QueryContext(ctx,statement,asOf,asOf)
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make(map[int64]bool)
	for rows.Next(){var id int64;if err:=rows.Scan(&id);err!=nil{return nil,err};out[id]=true}
	return out,rows.Err()
}

func (q *Querier) ListPlanningWorkloadLimits(ctx context.Context) (map[int64]sql.NullInt64,error) {
	rows,err:=q.Queries.db.QueryContext(ctx,"SELECT guru_id,max_weekly_minutes FROM guru_workload_policy")
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make(map[int64]sql.NullInt64)
	for rows.Next(){var id int64;var max sql.NullInt64;if err:=rows.Scan(&id,&max);err!=nil{return nil,err};out[id]=max}
	return out,rows.Err()
}
