package queries

import (
	"context"
	"database/sql"
	"errors"
)

type ClassPlanWorkflowRow struct {
	ID int64 `json:"id"`
	Source string `json:"source"`
	Status string `json:"status"`
	Program string `json:"program"`
	Angkatan string `json:"angkatan"`
	Gender string `json:"gender"`
	GuruID int64 `json:"guru_id"`
	Frequency int64 `json:"frequency"`
	MinQuota int64 `json:"min_quota"`
	Capacity int64 `json:"capacity"`
	TargetKelasID int64 `json:"target_kelas_id"`
	Version int64 `json:"version"`
	ConfirmedBy int64 `json:"coordinator_confirmed_by"`
	ConfirmedVersion int64 `json:"coordinator_confirmed_version"`
	ApprovedBy int64 `json:"admin_approved_by"`
	StudentCount int64 `json:"student_count"`
}

type ClassPlanWorkflowSlot struct {
	Day int `json:"day"`
	Start string `json:"start"`
	End string `json:"end"`
}

type ClassPlanWorkflowCandidate struct {
	ID int64
	Gender string
	Level string
	Angkatan string
	Status string
	ClassID sql.NullInt64
	Stage string
}

type ClassPlanOfferEvidence struct {
	Code string
	Days string
	Start string
	End string
}

func (q *Querier) ListClassPlanWorkflow(ctx context.Context) ([]ClassPlanWorkflowRow,error) {
	rows,err:=q.Queries.db.QueryContext(ctx,`SELECT p.id,p.source,p.status,p.tipe,p.angkatan,p.gender,
		COALESCE(p.guru_id,0),p.frequency,p.min_quota,p.capacity,COALESCE(p.target_kelas_id,0),p.version,
		COALESCE(p.coordinator_confirmed_by,0),COALESCE(p.coordinator_confirmed_version,0),
		COALESCE(p.admin_approved_by,0),(SELECT COUNT(*) FROM class_plan_students s
		    WHERE s.plan_id=p.id AND s.status IN ('candidate','reserved'))
		FROM class_plans p
		WHERE p.status IN ('draft','pending_coordinator','pending_admin','pending_exception','blocked')
		ORDER BY p.updated_at DESC,p.id DESC LIMIT 100`)
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make([]ClassPlanWorkflowRow,0)
	for rows.Next(){
		var r ClassPlanWorkflowRow
		if err:=rows.Scan(&r.ID,&r.Source,&r.Status,&r.Program,&r.Angkatan,&r.Gender,
			&r.GuruID,&r.Frequency,&r.MinQuota,&r.Capacity,&r.TargetKelasID,&r.Version,
			&r.ConfirmedBy,&r.ConfirmedVersion,&r.ApprovedBy,&r.StudentCount);err!=nil{return nil,err}
		out=append(out,r)
	}
	return out,rows.Err()
}

func (q *Querier) GetClassPlanWorkflow(ctx context.Context,id int64) (ClassPlanWorkflowRow,error) {
	var r ClassPlanWorkflowRow
	err:=q.Queries.db.QueryRowContext(ctx,`SELECT p.id,p.source,p.status,p.tipe,p.angkatan,p.gender,
		COALESCE(p.guru_id,0),p.frequency,p.min_quota,p.capacity,COALESCE(p.target_kelas_id,0),p.version,
		COALESCE(p.coordinator_confirmed_by,0),COALESCE(p.coordinator_confirmed_version,0),
		COALESCE(p.admin_approved_by,0),(SELECT COUNT(*) FROM class_plan_students s
		    WHERE s.plan_id=p.id AND s.status IN ('candidate','reserved'))
		FROM class_plans p WHERE p.id=?`,id).Scan(
			&r.ID,&r.Source,&r.Status,&r.Program,&r.Angkatan,&r.Gender,
			&r.GuruID,&r.Frequency,&r.MinQuota,&r.Capacity,&r.TargetKelasID,&r.Version,
			&r.ConfirmedBy,&r.ConfirmedVersion,&r.ApprovedBy,&r.StudentCount)
	return r,err
}

func (q *Querier) CreateClassPlanWorkflow(ctx context.Context,actorID int64,p ClassPlanWorkflowRow,slots []ClassPlanWorkflowSlot,candidates []int64,level string) (int64,error) {
	result,err:=q.Queries.db.ExecContext(ctx,`INSERT INTO class_plans
		(source,status,tipe,angkatan,gender,guru_id,frequency,min_quota,capacity,target_kelas_id,ruleset_version,created_by)
		VALUES ('manual','draft',?,?,?,?,?,?,?,?,'p2a-approval-v1',?)`,
		p.Program,p.Angkatan,p.Gender,p.GuruID,p.Frequency,p.MinQuota,p.Capacity,
		nullableClassPlanID(p.TargetKelasID),actorID)
	if err!=nil{return 0,err}
	id,err:=result.LastInsertId()
	if err!=nil{return 0,err}
	for _,slot:=range slots{
		if _,err:=q.Queries.db.ExecContext(ctx,`INSERT INTO class_plan_slots(plan_id,hari,jam_mulai,jam_selesai) VALUES (?,?,?,?)`,id,slot.Day,slot.Start,slot.End);err!=nil{return 0,err}
	}
	if _,err:=q.Queries.db.ExecContext(ctx,`INSERT INTO class_plan_levels(plan_id,level_kode) VALUES (?,?)`,id,level);err!=nil{return 0,err}
	for _,santriID:=range candidates{
		if _,err:=q.Queries.db.ExecContext(ctx,`INSERT INTO class_plan_students(plan_id,santri_id,level_kode,status,schedule_approved)
			VALUES (?,?,?,'candidate',0)`,id,santriID,level);err!=nil{return 0,err}
	}
	return id,nil
}

func nullableClassPlanID(id int64) interface{} {if id>0{return id};return nil}

func (q *Querier) ClassPlanWorkflowSlots(ctx context.Context,id int64) ([]ClassPlanWorkflowSlot,error) {
	rows,err:=q.Queries.db.QueryContext(ctx,`SELECT hari,jam_mulai,jam_selesai FROM class_plan_slots WHERE plan_id=? ORDER BY hari`,id)
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make([]ClassPlanWorkflowSlot,0)
	for rows.Next(){var r ClassPlanWorkflowSlot;if err:=rows.Scan(&r.Day,&r.Start,&r.End);err!=nil{return nil,err};out=append(out,r)}
	return out,rows.Err()
}

func (q *Querier) ClassPlanWorkflowCandidates(ctx context.Context,id int64) ([]ClassPlanWorkflowCandidate,error) {
	rows,err:=q.Queries.db.QueryContext(ctx,`SELECT s.id,s.jenis_kelamin,s.level,s.angkatan_kelas,s.status,s.kelas_id,
		COALESCE(oc.stage,'')
		FROM class_plan_students ps JOIN santri s ON s.id=ps.santri_id
		LEFT JOIN onboarding_cases oc ON oc.santri_id=s.id
		WHERE ps.plan_id=? AND ps.status='candidate' ORDER BY s.id`,id)
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make([]ClassPlanWorkflowCandidate,0)
	for rows.Next(){var r ClassPlanWorkflowCandidate
		if err:=rows.Scan(&r.ID,&r.Gender,&r.Level,&r.Angkatan,&r.Status,&r.ClassID,&r.Stage);err!=nil{return nil,err}
		out=append(out,r)}
	return out,rows.Err()
}

func (q *Querier) ClassPlanCandidateOffers(ctx context.Context,santriID int64) ([]ClassPlanOfferEvidence,error) {
	const statement = `SELECT jp.kode,jp.hari,jp.jam_mulai,jp.jam_selesai
	FROM leads l JOIN jadwal_penawaran jp ON jp.kode=l.jadwal_kode
	JOIN santri s ON s.id=l.santri_id
	WHERE l.santri_id=? AND l.verified_at IS NOT NULL AND l.needs_review=0
	AND l.status IN ('open','won') AND jp.is_buka=1
	AND (jp.gender='' OR jp.gender=s.jenis_kelamin)
	AND (jp.level='' OR jp.level=s.level)
	UNION
	SELECT jp.kode,jp.hari,jp.jam_mulai,jp.jam_selesai
	FROM santri_schedule_alternatives a
	JOIN jadwal_penawaran jp ON jp.kode=a.jadwal_kode
	JOIN santri s ON s.id=a.santri_id
	WHERE a.santri_id=? AND a.status='approved'
	AND a.consent_confirmed_at IS NOT NULL
	AND a.consent_channel IN ('whatsapp','phone','other') AND TRIM(a.consent_note)<>''
	AND jp.is_buka=1 AND (jp.gender='' OR jp.gender=s.jenis_kelamin)
	AND (jp.level='' OR jp.level=s.level)
	AND EXISTS (SELECT 1 FROM leads l WHERE l.santri_id=a.santri_id
		AND l.verified_at IS NOT NULL AND l.needs_review=0 AND l.status IN ('open','won'))`
	rows,err:=q.Queries.db.QueryContext(ctx,statement,santriID,santriID)
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make([]ClassPlanOfferEvidence,0)
	for rows.Next(){var r ClassPlanOfferEvidence
		if err:=rows.Scan(&r.Code,&r.Days,&r.Start,&r.End);err!=nil{return nil,err}
		out=append(out,r)}
	return out,rows.Err()
}

func (q *Querier) ClassPlanWorkflowTeacherValid(ctx context.Context,guruID int64,gender,level string) (bool,error) {
	var ok bool
	err:=q.Queries.db.QueryRowContext(ctx,`SELECT EXISTS(
		SELECT 1 FROM guru g
		JOIN guru_teaching_levels gl ON gl.guru_id=g.id AND gl.level_kode=?
		JOIN level l ON l.kode=gl.level_kode
		WHERE g.id=? AND g.is_aktif=1 AND g.jenis_kelamin IN ('L','P')
		AND NOT (g.jenis_kelamin='P' AND ?='L')
	)`,level,guruID,gender).Scan(&ok)
	return ok,err
}

func (q *Querier) ClassPlanWorkflowTargetValid(ctx context.Context,id int64,p ClassPlanWorkflowRow,level string) (bool,error) {
	if id==0{return true,nil}
	var ok bool
	err:=q.Queries.db.QueryRowContext(ctx,`SELECT EXISTS(
		SELECT 1 FROM kelas k
		WHERE k.id=? AND k.is_aktif=1 AND LOWER(k.tipe) IN ('reguler','regular')
		AND k.jenis_kelamin=? AND k.angkatan=? AND k.level=? AND k.guru_id=?
		AND CAST(k.frekuensi AS TEXT)=?
		AND NOT EXISTS(SELECT 1 FROM pertemuan m WHERE m.kelas_id=k.id)
	)`,id,p.Gender,p.Angkatan,level,p.GuruID,fmtClassPlanFrequency(p.Frequency)).Scan(&ok)
	return ok,err
}
func fmtClassPlanFrequency(f int64) string {if f==1{return "1x/pekan"};return "2x/pekan"}

func (q *Querier) ClassPlanWorkflowLevel(ctx context.Context,id int64) (string,error) {
	var level string
	err:=q.Queries.db.QueryRowContext(ctx,`SELECT level_kode FROM class_plan_levels WHERE plan_id=?`,id).Scan(&level)
	return level,err
}

func (q *Querier) ClassPlanTransition(ctx context.Context,id,version int64,from,to string,actorID int64) (bool,error) {
	var result sql.Result
	var err error
	switch to {
	case "pending_coordinator","pending_exception":
		result,err=q.Queries.db.ExecContext(ctx,`UPDATE class_plans SET status=?,updated_at=CURRENT_TIMESTAMP
			WHERE id=? AND version=? AND status=? AND coordinator_confirmed_by IS NULL AND admin_approved_by IS NULL`,to,id,version,from)
	case "pending_admin":
		result,err=q.Queries.db.ExecContext(ctx,`UPDATE class_plans SET status='pending_admin',
			coordinator_confirmed_by=?,coordinator_confirmed_at=CURRENT_TIMESTAMP,coordinator_confirmed_version=version,
			updated_at=CURRENT_TIMESTAMP WHERE id=? AND version=? AND status='pending_coordinator'
			AND coordinator_confirmed_by IS NULL`,actorID,id,version)
	case "admin_approved":
		result,err=q.Queries.db.ExecContext(ctx,`UPDATE class_plans SET
			admin_approved_by=?,admin_approved_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP
			WHERE id=? AND version=? AND status='pending_admin'
			AND coordinator_confirmed_by IS NOT NULL AND coordinator_confirmed_version=version
			AND admin_approved_by IS NULL`,actorID,id,version)
	default:return false,errors.New("unsupported class plan transition")
	}
	if err!=nil{return false,err}
	n,err:=result.RowsAffected()
	return n==1,err
}

func (q *Querier) AppendClassPlanWorkflowEvent(ctx context.Context,id,actorID int64,eventType,from,to,reason string) error {
	_,err:=q.Queries.db.ExecContext(ctx,`INSERT INTO class_plan_events
		(plan_id,actor_user_id,event_type,from_status,to_status,reason)
		VALUES (?,?,?,?,?,?)`,id,actorID,eventType,from,to,reason)
	return err
}
