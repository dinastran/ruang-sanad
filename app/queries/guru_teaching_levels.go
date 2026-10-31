package queries

import "context"

type GuruTeachingLevelRow struct {
	GuruID int64  `json:"guru_id"`
	LevelKode string `json:"level_kode"`
	ApprovedBy int64 `json:"approved_by"`
	ApprovedAt string `json:"approved_at"`
}

func (q *Querier) ListGuruTeachingLevels(ctx context.Context, guruID int64) ([]GuruTeachingLevelRow,error) {
	rows,err:=q.Queries.db.QueryContext(ctx,
		"SELECT guru_id,level_kode,approved_by,CAST(approved_at AS TEXT) FROM guru_teaching_levels WHERE guru_id=? ORDER BY level_kode",guruID)
	if err!=nil {return nil,err}
	defer rows.Close()
	out:=make([]GuruTeachingLevelRow,0)
	for rows.Next(){
		var r GuruTeachingLevelRow
		if err:=rows.Scan(&r.GuruID,&r.LevelKode,&r.ApprovedBy,&r.ApprovedAt);err!=nil{return nil,err}
		out=append(out,r)
	}
	return out,rows.Err()
}

// ReplaceTeachingLevelsTx must be called inside a transaction by the service,
// after validating the exact master level codes and authorized actor.
func (q *Querier) ReplaceTeachingLevelsTx(ctx context.Context, guruID,actorID int64,levels []string) error {
	if _,err:=q.Queries.db.ExecContext(ctx,"DELETE FROM guru_teaching_levels WHERE guru_id=?",guruID);err!=nil{return err}
	for _,level:=range levels {
		if _,err:=q.Queries.db.ExecContext(ctx,
			"INSERT INTO guru_teaching_levels(guru_id,level_kode,approved_by,approved_at) VALUES (?,?,?,CURRENT_TIMESTAMP)",
			guruID,level,actorID);err!=nil{return err}
	}
	return nil
}
