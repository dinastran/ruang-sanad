package queries

import (
	"context"
	"database/sql"
)

type ScheduleAlternativeOffer struct {
	Kode string `json:"kode"`
	Hari string `json:"hari"`
	JamMulai string `json:"jam_mulai"`
	JamSelesai string `json:"jam_selesai"`
	Gender string `json:"gender"`
	Level string `json:"level"`
}

type SantriScheduleAlternative struct {
	ID int64 `json:"id"`
	JadwalKode string `json:"jadwal_kode"`
	Status string `json:"status"`
	ConsentChannel string `json:"consent_channel"`
	ConsentNote string `json:"consent_note"`
	ConsentConfirmedBy int64 `json:"consent_confirmed_by"`
	ConsentConfirmedAt string `json:"consent_confirmed_at"`
	UpdatedAt string `json:"updated_at"`
	Label string `json:"label"`
}

func (q *Querier) ListAlternateOffersForSantri(ctx context.Context, id int64) ([]ScheduleAlternativeOffer,error) {
	const statement = `SELECT j.kode,j.hari,j.jam_mulai,j.jam_selesai,j.gender,j.level
		FROM jadwal_penawaran j JOIN santri s ON s.id=?
		WHERE j.is_buka=1 AND (j.gender='' OR j.gender=s.jenis_kelamin)
		AND (j.level='' OR j.level=s.level)
		ORDER BY j.urutan,j.kode`
	rows,err:=q.Queries.db.QueryContext(ctx,statement,id)
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make([]ScheduleAlternativeOffer,0)
	for rows.Next(){var r ScheduleAlternativeOffer
		if err:=rows.Scan(&r.Kode,&r.Hari,&r.JamMulai,&r.JamSelesai,&r.Gender,&r.Level);err!=nil{return nil,err}
		out=append(out,r)}
	return out,rows.Err()
}

func (q *Querier) ListSantriScheduleAlternatives(ctx context.Context, id int64) ([]SantriScheduleAlternative,error) {
	const statement = `SELECT a.id,a.jadwal_kode,a.status,a.consent_channel,a.consent_note,
		COALESCE(a.consent_confirmed_by,0),COALESCE(CAST(a.consent_confirmed_at AS TEXT),''),
		CAST(a.updated_at AS TEXT),
		COALESCE(j.hari||' '||j.jam_mulai||'-'||j.jam_selesai,'Jadwal tidak tersedia')
		FROM santri_schedule_alternatives a
		LEFT JOIN jadwal_penawaran j ON j.kode=a.jadwal_kode
		WHERE a.santri_id=? ORDER BY a.updated_at DESC,a.id DESC`
	rows,err:=q.Queries.db.QueryContext(ctx,statement,id)
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make([]SantriScheduleAlternative,0)
	for rows.Next(){var r SantriScheduleAlternative
		if err:=rows.Scan(&r.ID,&r.JadwalKode,&r.Status,&r.ConsentChannel,&r.ConsentNote,
			&r.ConsentConfirmedBy,&r.ConsentConfirmedAt,&r.UpdatedAt,&r.Label);err!=nil{return nil,err}
		out=append(out,r)}
	return out,rows.Err()
}

func (q *Querier) GetSantriAlternativeState(ctx context.Context,id int64,kode string) (int64,string,string,string,error) {
	var rowID int64
	var status,channel,note string
	err:=q.Queries.db.QueryRowContext(ctx,`SELECT id,status,consent_channel,consent_note
		FROM santri_schedule_alternatives WHERE santri_id=? AND jadwal_kode=?`,id,kode).Scan(&rowID,&status,&channel,&note)
	return rowID,status,channel,note,err
}

func (q *Querier) CanRecordSantriAlternative(ctx context.Context,id int64,kode string) (bool,error) {
	var valid bool
	err:=q.Queries.db.QueryRowContext(ctx,`SELECT EXISTS(
		SELECT 1 FROM santri s
		JOIN onboarding_cases oc ON oc.santri_id=s.id
		JOIN jadwal_penawaran j ON j.kode=?
		WHERE s.id=? AND s.status='aktif'
		AND oc.stage NOT IN ('masuk_kelas','batal')
		AND j.is_buka=1
		AND (j.gender='' OR j.gender=s.jenis_kelamin)
		AND (j.level='' OR j.level=s.level)
		AND NOT EXISTS (SELECT 1 FROM leads l WHERE l.santri_id=s.id AND l.jadwal_kode=j.kode)
	)`,kode,id).Scan(&valid)
	return valid,err
}

func (q *Querier) SaveSantriAlternativeTx(ctx context.Context,id int64,kode,status,channel,note string,actor int64) (int64,error) {
	var confirmer sql.NullInt64
	if status=="approved" {
		confirmer=sql.NullInt64{Int64:actor,Valid:true}
	}
	// Timestamp expressions must stay SQL expressions, not bound strings.
	const statement = `INSERT INTO santri_schedule_alternatives
		(santri_id,jadwal_kode,status,consent_channel,consent_note,
		 consent_confirmed_by,consent_confirmed_at,updated_by)
		VALUES (?,?,?,?,?,?,
		 CASE WHEN ?='approved' THEN CURRENT_TIMESTAMP ELSE NULL END,?)
		ON CONFLICT(santri_id,jadwal_kode) DO UPDATE SET
		 status=excluded.status,consent_channel=excluded.consent_channel,
		 consent_note=excluded.consent_note,consent_confirmed_by=excluded.consent_confirmed_by,
		 consent_confirmed_at=excluded.consent_confirmed_at,
		 updated_by=excluded.updated_by,updated_at=CURRENT_TIMESTAMP
		RETURNING id`
	var newID int64
	err:=q.Queries.db.QueryRowContext(ctx,statement,id,kode,status,channel,note,confirmer,status,actor).Scan(&newID)
	return newID,err
}

func (q *Querier) AppendSantriAlternativeEventTx(ctx context.Context,alternativeID,santriID,actorID int64,oldStatus,newStatus,channel,note string) error {
	_,err:=q.Queries.db.ExecContext(ctx,`INSERT INTO santri_schedule_alternative_events
		(alternative_id,santri_id,actor_user_id,old_status,new_status,consent_channel,reason)
		VALUES (?,?,?,?,?,?,?)`,alternativeID,santriID,actorID,oldStatus,newStatus,channel,note)
	return err
}
