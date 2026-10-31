package queries

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestPlanningDemandOnlyVerifiedUnassignedWithKnownSchedule(t *testing.T) {
	db,err:=sql.Open("sqlite",":memory:")
	if err!=nil{t.Fatal(err)}
	defer db.Close()
	db.SetMaxOpenConns(1)
	schema:=[]string{
		"CREATE TABLE santri(id INTEGER PRIMARY KEY,angkatan_kelas TEXT,jenis_kelamin TEXT,level TEXT,frekuensi TEXT,status TEXT,kelas_id INTEGER,is_lengkap INTEGER)",
		"CREATE TABLE onboarding_cases(santri_id INTEGER,stage TEXT,deadline_at DATETIME)",
		"CREATE TABLE leads(santri_id INTEGER, jadwal_kode TEXT,verified_at DATETIME,needs_review INTEGER,status TEXT)",
		"CREATE TABLE jadwal_penawaran(kode TEXT,hari TEXT,jam_mulai TEXT,jam_selesai TEXT,gender TEXT)",
		"CREATE TABLE class_plan_students(santri_id INTEGER,status TEXT)",
	}
	for _,s:=range schema{if _,err:=db.Exec(s);err!=nil{t.Fatal(err)}}
	fixtures:=[]string{
		"INSERT INTO jadwal_penawaran VALUES('SJ2','2,5','19:30','21:00','P')",
		"INSERT INTO santri VALUES (1,'2026-A','P','PRA','2x/pekan','aktif',NULL,1),(2,'2026-A','P','PRA','2x/pekan','aktif',NULL,1),(3,'2026-A','P','PRA','2x/pekan','aktif',NULL,1),(4,'2026-A','P','PRA','2x/pekan','aktif',20,1),(5,'2026-A','L','PRA','2x/pekan','aktif',NULL,1)",
		"INSERT INTO onboarding_cases VALUES (1,'matching_kelas','2026-11-01'),(2,'matching_kelas','2026-11-01'),(3,'matching_kelas','2026-11-01'),(4,'waiting_kuota','2026-11-01'),(5,'matching_kelas','2026-11-01')",
		"INSERT INTO leads VALUES (1,'SJ2','2026-10-30',0,'won'),(2,'SJ2',NULL,0,'open'),(3,'SJ2','2026-10-30',0,'open'),(4,'SJ2','2026-10-30',0,'won'),(5,'SJ2','2026-10-30',0,'won')",
		"INSERT INTO class_plan_students VALUES (3,'reserved')",
	}
	for _,s:=range fixtures{if _,err:=db.Exec(s);err!=nil{t.Fatal(err)}}
	rows,err:=NewQuerier(db).ListPlanningDemand(context.Background())
	if err!=nil{t.Fatal(err)}
	if len(rows)!=1||rows[0].SantriID!=1||rows[0].OfferCode!="SJ2"{
		t.Fatalf("only verified, unassigned, unreserved matching gender is eligible: %+v",rows)
	}
}

func TestPlanningTeacherBusyCombinesRoutinesAndApprovedPlans(t *testing.T) {
	db,err:=sql.Open("sqlite",":memory:")
	if err!=nil{t.Fatal(err)}
	defer db.Close()
	db.SetMaxOpenConns(1)
	schema:=[]string{
		"CREATE TABLE kelas(id INTEGER PRIMARY KEY,guru_id INTEGER,is_aktif INTEGER)",
		"CREATE TABLE kelas_jadwal_rutin(kelas_id INTEGER,hari INTEGER,jam_mulai TEXT,is_aktif INTEGER,berlaku_mulai DATE,berlaku_sampai DATE)",
		"CREATE TABLE class_plans(id INTEGER PRIMARY KEY,guru_id INTEGER,status TEXT)",
		"CREATE TABLE class_plan_slots(plan_id INTEGER,hari INTEGER,jam_mulai TEXT,jam_selesai TEXT)",
		"INSERT INTO kelas VALUES(1,10,1)",
		"INSERT INTO kelas_jadwal_rutin VALUES(1,2,'19:30',1,'2026-10-01',NULL)",
		"INSERT INTO class_plans VALUES(20,10,'pending_admin'),(21,10,'draft')",
		"INSERT INTO class_plan_slots VALUES(20,5,'20:00','21:30'),(21,6,'20:00','21:30')",
	}
	for _,s:=range schema{if _,err:=db.Exec(s);err!=nil{t.Fatal(err)}}
	rows,err:=NewQuerier(db).ListPlanningTeacherBusy(context.Background(),"2026-10-31")
	if err!=nil{t.Fatal(err)}
	if len(rows)!=2||rows[0].End!="21:00"||rows[1].End!="21:30"{
		t.Fatalf("expected routine and pending_admin with no draft: %+v",rows)
	}
}
