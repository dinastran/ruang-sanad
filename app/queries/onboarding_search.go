package queries

import (
	"context"
	"strings"
	"time"
)

// OnboardingBoardFilter is applied before pagination, so a search never
// silently excludes cases on later pages.
type OnboardingBoardFilter struct {
	Query string
	PICID int64
	UnassignedPIC bool
	SLA string
	ClassID int64
	Angkatan string
	// Sort and Dir are validated by the handler; unknown values fall back
	// to the default registration order.
	Sort string
	Dir string
	Now time.Time
}

// OnboardingBoardListRow extends the sqlc board row with santri profile
// columns used by the list view.
type OnboardingBoardListRow struct {
	ListOnboardingBoardRow
	Angkatan     string
	Level        string
	Tipe         string
	Frekuensi    string
	JenisKelamin string
	Domisili     string
}

var onboardingSortColumns = map[string]string{
	"nama":       "LOWER(s.nama)",
	"angkatan":   "COALESCE(s.angkatan,'')",
	"registered": "oc.registered_at",
	"deadline":   "julianday(substr(oc.deadline_at,1,19))",
	"stage":      "CASE oc.stage WHEN 'onboarding_baru' THEN 0 WHEN 'matching_kelas' THEN 1 WHEN 'waiting_kuota' THEN 2 WHEN 'siap_mulai' THEN 3 WHEN 'masuk_kelas' THEN 4 ELSE 5 END",
}

// ValidOnboardingSort reports whether key is a supported list sort column.
func ValidOnboardingSort(key string) bool {
	_, ok := onboardingSortColumns[key]
	return ok
}

func onboardingBoardOrder(f OnboardingBoardFilter) string {
	column, ok := onboardingSortColumns[f.Sort]
	if !ok {
		return " ORDER BY CASE WHEN oc.stage='batal' THEN 1 ELSE 0 END, oc.registered_at DESC, oc.id DESC"
	}
	dir := "ASC"
	if f.Dir == "desc" {
		dir = "DESC"
	}
	// Cancelled cases stay at the end regardless of sort column.
	return " ORDER BY CASE WHEN oc.stage='batal' THEN 1 ELSE 0 END, " + column + " " + dir + ", oc.registered_at DESC, oc.id DESC"
}

type OnboardingClassOption struct {
	ID int64 `json:"id"`
	Name string `json:"name"`
}

func onboardingLike(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, `\`, `\\`)
	raw = strings.ReplaceAll(raw, "%", `\%`)
	raw = strings.ReplaceAll(raw, "_", `\_`)
	return "%" + raw + "%"
}

func onboardingBoardPredicate(f OnboardingBoardFilter) (string, []any) {
	clauses := []string{}
	args := []any{}
	if strings.TrimSpace(f.Query) != "" {
		clauses = append(clauses, `(LOWER(s.nama) LIKE ? ESCAPE char(92) OR LOWER(COALESCE(s.id_mahasantri,'')) LIKE ? ESCAPE char(92) OR LOWER(COALESCE(k.nama_kelas,'')) LIKE ? ESCAPE char(92))`)
		pattern := onboardingLike(f.Query)
		args = append(args, pattern, pattern, pattern)
	}
	if f.UnassignedPIC {
		clauses = append(clauses, "oc.pic_user_id IS NULL")
	} else if f.PICID > 0 {
		clauses = append(clauses, "oc.pic_user_id = ?")
		args = append(args, f.PICID)
	}
	if f.ClassID > 0 {
		clauses = append(clauses, "s.kelas_id = ?")
		args = append(args, f.ClassID)
	}
	if f.Angkatan != "" {
		clauses = append(clauses, "s.angkatan = ?")
		args = append(args, f.Angkatan)
	}
	if f.SLA == "overdue" || f.SLA == "at_risk" {
		now := f.Now
		if now.IsZero() {
			now = time.Now().UTC()
		}
		clauses = append(clauses, "oc.stage NOT IN ('masuk_kelas','batal')")
		clauses = append(clauses, "julianday(substr(oc.deadline_at,1,19)) IS NOT NULL")
		if f.SLA == "overdue" {
			clauses = append(clauses, "julianday(substr(oc.deadline_at,1,19)) <= julianday(?)")
			args = append(args, now.UTC().Format(time.RFC3339))
		} else {
			// The alert period starts on D11 (deadline is start of D15 WIB).
			// SQLite TIME values include a Go timezone suffix which julianday cannot parse;
			// the first 19 characters are a UTC calendar timestamp.
			wib := time.FixedZone("WIB", 7*60*60)
			local := now.In(wib)
			dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, wib)
			clauses = append(clauses, "julianday(substr(oc.deadline_at,1,19)) > julianday(?) AND julianday(substr(oc.deadline_at,1,19)) <= julianday(?)")
			args = append(args, now.UTC().Format(time.RFC3339), dayStart.AddDate(0, 0, 4).UTC().Format(time.RFC3339))
		}
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (q *Querier) ListOnboardingBoardFiltered(ctx context.Context, f OnboardingBoardFilter, limit, offset int64) ([]OnboardingBoardListRow, int64, []CountOnboardingStagesRow, error) {
	const source = ` FROM onboarding_cases oc JOIN santri s ON s.id=oc.santri_id LEFT JOIN users u ON u.id=oc.pic_user_id LEFT JOIN kelas k ON k.id=s.kelas_id`
	where, args := onboardingBoardPredicate(f)
	countQuery := "SELECT oc.stage, COUNT(*)" + source + where + " GROUP BY oc.stage"
	countRows, err := q.Queries.db.QueryContext(ctx, countQuery, args...)
	if err != nil { return nil, 0, nil, err }
	counts := []CountOnboardingStagesRow{}
	var total int64
	for countRows.Next() {
		var row CountOnboardingStagesRow
		if err := countRows.Scan(&row.Stage, &row.Total); err != nil {
			countRows.Close()
			return nil, 0, nil, err
		}
		total += row.Total
		counts = append(counts, row)
	}
	if err := countRows.Err(); err != nil {
		countRows.Close()
		return nil, 0, nil, err
	}
	countRows.Close()

	listQuery := `SELECT oc.id,oc.santri_id,s.nama,s.id_mahasantri,oc.stage,oc.pic_user_id,COALESCE(u.name,''),oc.registered_at,oc.deadline_at,oc.actual_started_at,s.kelas_id,COALESCE(k.nama_kelas,''),` +
		`COALESCE(s.angkatan,''),COALESCE(s.level,''),COALESCE(s.tipe,''),COALESCE(s.frekuensi,''),COALESCE(s.jenis_kelamin,''),COALESCE(s.domisili,'')` + source + where +
		onboardingBoardOrder(f) + " LIMIT ? OFFSET ?"
	listArgs := append(append([]any{}, args...), limit, offset)
	rows, err := q.Queries.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil { return nil, 0, nil, err }
	defer rows.Close()
	items := []OnboardingBoardListRow{}
	for rows.Next() {
		var r OnboardingBoardListRow
		if err := rows.Scan(&r.ID,&r.SantriID,&r.Nama,&r.IDMahasantri,&r.Stage,&r.PicUserID,&r.PicName,&r.RegisteredAt,&r.DeadlineAt,&r.ActualStartedAt,&r.KelasID,&r.KelasNama,
			&r.Angkatan,&r.Level,&r.Tipe,&r.Frekuensi,&r.JenisKelamin,&r.Domisili); err != nil {
			return nil, 0, nil, err
		}
		items = append(items, r)
	}
	return items, total, counts, rows.Err()
}

func (q *Querier) ListOnboardingClassOptions(ctx context.Context) ([]OnboardingClassOption, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `SELECT k.id,k.nama_kelas FROM kelas k WHERE EXISTS (SELECT 1 FROM santri s JOIN onboarding_cases oc ON oc.santri_id=s.id WHERE s.kelas_id=k.id) ORDER BY k.nama_kelas,k.id LIMIT 2000`)
	if err != nil { return nil, err }
	defer rows.Close()
	out := []OnboardingClassOption{}
	for rows.Next() {
		var v OnboardingClassOption
		if err := rows.Scan(&v.ID,&v.Name); err != nil { return nil, err }
		out = append(out, v)
	}
	return out, rows.Err()
}

// OnboardingQuotaFilter always scopes to classes with onboarding cases,
// preserving the existing rule that historical classes are never backfilled.
type OnboardingQuotaFilter struct {
	Query string
	Status string
}

func onboardingQuotaPredicate(f OnboardingQuotaFilter) (string, []any) {
	where := []string{}
	args := []any{}
	if strings.TrimSpace(f.Query) != "" {
		where = append(where, `(LOWER(x.nama_kelas) LIKE ? ESCAPE char(92) OR CAST(x.id AS TEXT)=?)`)
		args = append(args, onboardingLike(f.Query), strings.TrimSpace(f.Query))
	}
	switch f.Status {
	case "needs_target":
		where = append(where, "COALESCE(x.target_kuota,0)=0")
	case "shortage":
		where = append(where, "x.target_kuota>0 AND x.jumlah_santri<x.target_kuota")
	case "ready":
		where = append(where, "x.is_aktif=1 AND x.guru_id IS NOT NULL AND TRIM(COALESCE(x.jadwal,''))<>'' AND x.target_kuota>0 AND x.target_kuota<=x.kapasitas AND (x.jumlah_santri>=x.target_kuota OR x.exception_approved_at IS NOT NULL)")
	}
	if len(where)==0 { return "", args }
	return " WHERE "+strings.Join(where," AND "), args
}

func (q *Querier) ListOnboardingClassControlsFiltered(ctx context.Context, f OnboardingQuotaFilter, limit, offset int64) ([]OnboardingClassControlRow, int64, error) {
	source := " FROM (" + onboardingClassControlSelect + ") AS x"
	where,args := onboardingQuotaPredicate(f)
	var total int64
	if err := q.Queries.db.QueryRowContext(ctx,"SELECT COUNT(*)"+source+where,args...).Scan(&total); err != nil { return nil,0,err }
	listArgs := append(append([]any{},args...),limit,offset)
	rows,err := q.Queries.db.QueryContext(ctx,"SELECT x.*"+source+where+" ORDER BY x.registered_at DESC,x.id DESC LIMIT ? OFFSET ?",listArgs...)
	if err != nil { return nil,0,err }
	defer rows.Close()
	out := []OnboardingClassControlRow{}
	for rows.Next() {
		v,err := scanOnboardingClass(rows.Scan)
		if err != nil { return nil,0,err }
		out=append(out,v)
	}
	return out,total,rows.Err()
}

type OnboardingClassMemberRow struct {
	ID int64 `json:"id"`
	Nama string `json:"nama"`
	IDMahasantri string `json:"id_mahasantri"`
	Status string `json:"status"`
	Stage string `json:"stage"`
}

func (q *Querier) ListOnboardingClassMembers(ctx context.Context, classID, limit, offset int64) ([]OnboardingClassMemberRow, int64, error) {
	const base = ` FROM santri s LEFT JOIN onboarding_cases oc ON oc.santri_id=s.id WHERE s.kelas_id=? AND s.status IN ('aktif','cuti')`
	var total int64
	if err:=q.Queries.db.QueryRowContext(ctx,"SELECT COUNT(*)"+base,classID).Scan(&total); err!=nil { return nil,0,err }
	rows,err:=q.Queries.db.QueryContext(ctx,`SELECT s.id,s.nama,COALESCE(s.id_mahasantri,''),s.status,COALESCE(oc.stage,'')`+base+" ORDER BY s.nama COLLATE NOCASE,s.id LIMIT ? OFFSET ?",classID,limit,offset)
	if err!=nil { return nil,0,err }
	defer rows.Close()
	items:=[]OnboardingClassMemberRow{}
	for rows.Next() {
		var r OnboardingClassMemberRow
		if err:=rows.Scan(&r.ID,&r.Nama,&r.IDMahasantri,&r.Status,&r.Stage); err!=nil { return nil,0,err }
		items=append(items,r)
	}
	return items,total,rows.Err()
}

