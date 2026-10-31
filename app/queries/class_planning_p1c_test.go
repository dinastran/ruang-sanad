package queries

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func p1cDB(t *testing.T) *sql.DB {
	t.Helper()
	db,err:=sql.Open("sqlite",":memory:")
	if err!=nil{t.Fatal(err)}
	db.SetMaxOpenConns(1)
	t.Cleanup(func(){_ =db.Close()})
	statements:=[]string{
		"CREATE TABLE guru_teaching_levels(guru_id INTEGER,level_kode TEXT,approved_by INTEGER,approved_at DATETIME,PRIMARY KEY(guru_id,level_kode))",
		"CREATE TABLE level(kode TEXT PRIMARY KEY)",
		"INSERT INTO level VALUES('1')",
		"CREATE TABLE santri(id INTEGER PRIMARY KEY,jenis_kelamin TEXT,level TEXT,status TEXT)",
		"CREATE TABLE onboarding_cases(santri_id INTEGER,stage TEXT)",
		"CREATE TABLE leads(santri_id INTEGER,jadwal_kode TEXT)",
		"CREATE TABLE jadwal_penawaran(kode TEXT PRIMARY KEY,hari TEXT,jam_mulai TEXT,jam_selesai TEXT,gender TEXT,level TEXT,is_buka INTEGER,urutan INTEGER)",
		`CREATE TABLE santri_schedule_alternatives(
			id INTEGER PRIMARY KEY AUTOINCREMENT,santri_id INTEGER,jadwal_kode TEXT,
			status TEXT CHECK(status IN ('proposed','approved','revoked')),
			consent_channel TEXT NOT NULL DEFAULT '',consent_note TEXT NOT NULL DEFAULT '',
			consent_confirmed_by INTEGER,consent_confirmed_at DATETIME,updated_by INTEGER,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,UNIQUE(santri_id,jadwal_kode),
			CHECK(status!='approved' OR (consent_channel!='' AND length(trim(consent_note))>0
				AND consent_confirmed_by IS NOT NULL AND consent_confirmed_at IS NOT NULL)))`,
		"CREATE TABLE santri_schedule_alternative_events(alternative_id INTEGER,santri_id INTEGER,actor_user_id INTEGER,old_status TEXT,new_status TEXT,consent_channel TEXT,reason TEXT)",
		"INSERT INTO santri VALUES(1,'P','1','aktif'),(2,'L','2','aktif')",
		"INSERT INTO onboarding_cases VALUES(1,'matching_kelas'),(2,'matching_kelas')",
		"INSERT INTO leads VALUES(1,'J1')",
		"INSERT INTO jadwal_penawaran VALUES('J1','2','19:30','21:00','P','1',1,1),('J2','6','19:30','21:00','P','1',1,2),('J3','7','19:30','21:00','L','2',1,3),('J4','7','19:30','21:00','P','1',0,4)",
	}
	for _,s:=range statements{if _,err:=db.Exec(s);err!=nil{t.Fatal(err)}}
	return db
}

func TestP1CTeacherLevelsQueries(t *testing.T) {
	ctx:=context.Background()
	q:=NewQuerier(p1cDB(t))
	if err:=q.ReplaceTeachingLevelsTx(ctx,10,5,[]string{"1","2"});err!=nil{t.Fatal(err)}
	approved,err:=q.ListPlanningTeacherLevels(ctx)
	if err!=nil||len(approved)!=1||approved[0].Level!="1"{
		t.Fatalf("orphaned master level should not qualify for recommendation: %+v %v",approved,err)
	}

	rows,err:=q.ListGuruTeachingLevels(ctx,10)
	if err!=nil||len(rows)!=2||rows[0].ApprovedBy!=5{t.Fatalf("level insert/read: %+v %v",rows,err)}
	if err:=q.ReplaceTeachingLevelsTx(ctx,10,5,[]string{});err!=nil{t.Fatal(err)}
	rows,err=q.ListGuruTeachingLevels(ctx,10)
	if err!=nil||len(rows)!=0{t.Fatalf("revocation did not empty list: %+v %v",rows,err)}
}

func TestP1CAlternativeEligibilityAndAudit(t *testing.T) {
	ctx:=context.Background()
	q:=NewQuerier(p1cDB(t))
	if ok,_:=q.CanRecordSantriAlternative(ctx,1,"J1");ok{t.Fatal("primary schedule cannot be duplicated as alternative")}
	if ok,_:=q.CanRecordSantriAlternative(ctx,1,"J3");ok{t.Fatal("opposite-gender schedule is forbidden")}
	if ok,_:=q.CanRecordSantriAlternative(ctx,1,"J4");ok{t.Fatal("closed schedule is forbidden")}
	ok,err:=q.CanRecordSantriAlternative(ctx,1,"J2")
	if err!=nil||!ok{t.Fatalf("confirmed valid offered alternate rejected: %v",err)}
	id,err:=q.SaveSantriAlternativeTx(ctx,1,"J2","approved","whatsapp","Santri mengonfirmasi",7)
	if err!=nil{t.Fatal(err)}
	if err:=q.AppendSantriAlternativeEventTx(ctx,id,1,7,"","approved","whatsapp","Santri mengonfirmasi");err!=nil{t.Fatal(err)}
	rows,err:=q.ListSantriScheduleAlternatives(ctx,1)
	if err!=nil||len(rows)!=1||rows[0].Status!="approved"||rows[0].ConsentConfirmedAt==""{
		t.Fatalf("approval not persisted: %+v %v",rows,err)
	}
	id,err=q.SaveSantriAlternativeTx(ctx,1,"J2","revoked","","Dibatalkan santri",7)
	if err!=nil{t.Fatal(err)}
	if err:=q.AppendSantriAlternativeEventTx(ctx,id,1,7,"approved","revoked","","Dibatalkan santri");err!=nil{t.Fatal(err)}
	rows,err=q.ListSantriScheduleAlternatives(ctx,1)
	if err!=nil||len(rows)!=1||rows[0].Status!="revoked"{t.Fatalf("revocation not persisted: %+v %v",rows,err)}
	var events int
	if err:=q.Queries.db.QueryRowContext(ctx,"SELECT COUNT(*) FROM santri_schedule_alternative_events").Scan(&events);err!=nil||events!=2{
		t.Fatalf("missing audit events: %d %v",events,err)
	}
}
