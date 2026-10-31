package queries

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestPlanningApprovedAlternativesReadOnlyEligibility(t *testing.T) {
	db,err:=sql.Open("sqlite",":memory:")
	if err!=nil{t.Fatal(err)}
	defer db.Close()
	db.SetMaxOpenConns(1)
	fixtures:=[]string{
		"CREATE TABLE santri(id INTEGER PRIMARY KEY,angkatan_kelas TEXT,jenis_kelamin TEXT,level TEXT,frekuensi TEXT,status TEXT,kelas_id INTEGER,is_lengkap INTEGER)",
		"CREATE TABLE onboarding_cases(santri_id INTEGER,stage TEXT)",
		"CREATE TABLE leads(santri_id INTEGER,jadwal_kode TEXT,verified_at TEXT,needs_review INTEGER,status TEXT)",
		"CREATE TABLE jadwal_penawaran(kode TEXT,hari TEXT,jam_mulai TEXT,jam_selesai TEXT,gender TEXT,level TEXT,is_buka INTEGER)",
		"CREATE TABLE santri_schedule_alternatives(santri_id INTEGER,jadwal_kode TEXT,status TEXT,consent_confirmed_at TEXT,consent_channel TEXT,consent_note TEXT)",
		"CREATE TABLE class_plan_students(santri_id INTEGER,status TEXT)",
		"INSERT INTO santri VALUES (1,'A','P','1','2x/pekan','aktif',NULL,1),(2,'A','P','1','2x/pekan','aktif',NULL,1),(3,'A','P','1','2x/pekan','aktif',7,1),(4,'A','P','1','2x/pekan','aktif',NULL,1)",
		"INSERT INTO onboarding_cases VALUES (1,'waiting_kuota'),(2,'waiting_kuota'),(3,'waiting_kuota'),(4,'waiting_kuota')",
		"INSERT INTO leads VALUES(1,'PRIMARY','2026-10-30',0,'won'),(2,'PRIMARY',NULL,0,'open'),(3,'PRIMARY','2026-10-30',0,'won'),(4,'PRIMARY','2026-10-30',0,'won')",
		"INSERT INTO jadwal_penawaran VALUES ('PRIMARY','2,5','19:30','21:00','P','1',1),('ALT','6,7','19:30','21:00','P','1',1),('MALE','6,7','19:30','21:00','L','1',1),('CLOSED','6,7','19:30','21:00','P','1',0)",
		"INSERT INTO santri_schedule_alternatives VALUES (1,'ALT','approved','2026-10-30','whatsapp','Santri menyetujui'),(1,'PRIMARY','approved','2026-10-30','whatsapp','duplicate primary'),(1,'MALE','approved','2026-10-30','whatsapp','invalid gender'),(1,'CLOSED','approved','2026-10-30','whatsapp','closed'),(2,'ALT','approved','2026-10-30','whatsapp','not paid'),(3,'ALT','approved','2026-10-30','whatsapp','assigned'),(4,'ALT','proposed',NULL,'','not approved')",
	}
	for _,stmt:=range fixtures{if _,err:=db.Exec(stmt);err!=nil{t.Fatal(err)}}
	q:=NewQuerier(db)
	got,err:=q.ListPlanningApprovedAlternatives(context.Background())
	if err!=nil{t.Fatal(err)}
	if len(got)!=1||got[0].SantriID!=1||got[0].OfferCode!="ALT"{
		t.Fatalf("consent/eligibility not enforced: %+v",got)
	}
	if _,err:=db.Exec("INSERT INTO class_plan_students VALUES(1,'reserved')");err!=nil{t.Fatal(err)}
	got,err=q.ListPlanningApprovedAlternatives(context.Background())
	if err!=nil||len(got)!=0{t.Fatalf("reserved student must not reappear: %+v, %v",got,err)}
}

func TestPlanningApprovedAlternativesRequiresConsentEvidence(t *testing.T) {
	db,err:=sql.Open("sqlite",":memory:")
	if err!=nil{t.Fatal(err)}
	defer db.Close()
	db.SetMaxOpenConns(1)
	fixtures:=[]string{
		"CREATE TABLE santri(id INTEGER PRIMARY KEY,angkatan_kelas TEXT,jenis_kelamin TEXT,level TEXT,frekuensi TEXT,status TEXT,kelas_id INTEGER,is_lengkap INTEGER)",
		"CREATE TABLE onboarding_cases(santri_id INTEGER,stage TEXT)",
		"CREATE TABLE leads(santri_id INTEGER,jadwal_kode TEXT,verified_at TEXT,needs_review INTEGER,status TEXT)",
		"CREATE TABLE jadwal_penawaran(kode TEXT,hari TEXT,jam_mulai TEXT,jam_selesai TEXT,gender TEXT,level TEXT,is_buka INTEGER)",
		"CREATE TABLE santri_schedule_alternatives(santri_id INTEGER,jadwal_kode TEXT,status TEXT,consent_confirmed_at TEXT,consent_channel TEXT,consent_note TEXT)",
		"CREATE TABLE class_plan_students(santri_id INTEGER,status TEXT)",
		"INSERT INTO santri VALUES(1,'A','P','1','1x/pekan','aktif',NULL,1)",
		"INSERT INTO onboarding_cases VALUES(1,'matching_kelas')",
		"INSERT INTO leads VALUES(1,'PRIMARY','2026-10-30',0,'won')",
		"INSERT INTO jadwal_penawaran VALUES('ALT','6','19:30','21:00','P','1',1)",
		"INSERT INTO santri_schedule_alternatives VALUES(1,'ALT','approved',NULL,'whatsapp','missing confirmation timestamp')",
	}
	for _,stmt:=range fixtures{if _,err:=db.Exec(stmt);err!=nil{t.Fatal(err)}}
	q:=NewQuerier(db)
	rows,err:=q.ListPlanningApprovedAlternatives(context.Background())
	if err!=nil||len(rows)!=0{t.Fatalf("missing timestamp must not qualify: %+v, %v",rows,err)}
	if _,err:=db.Exec("UPDATE santri_schedule_alternatives SET consent_confirmed_at='2026-10-30',consent_note=''");err!=nil{t.Fatal(err)}
	rows,err=q.ListPlanningApprovedAlternatives(context.Background())
	if err!=nil||len(rows)!=0{t.Fatalf("missing evidence must not qualify: %+v, %v",rows,err)}
}
