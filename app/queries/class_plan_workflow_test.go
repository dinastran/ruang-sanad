package queries

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func workflowQueryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	stmts := []string{
		`CREATE TABLE class_plans(
			id INTEGER PRIMARY KEY AUTOINCREMENT,source TEXT NOT NULL,status TEXT NOT NULL,
			tipe TEXT NOT NULL,angkatan TEXT NOT NULL,gender TEXT NOT NULL,guru_id INTEGER,
			frequency INTEGER NOT NULL,min_quota INTEGER NOT NULL,capacity INTEGER NOT NULL,
			target_kelas_id INTEGER,version INTEGER NOT NULL DEFAULT 1,
			ruleset_version TEXT NOT NULL,created_by INTEGER NOT NULL,
			coordinator_confirmed_by INTEGER,coordinator_confirmed_at DATETIME,
			coordinator_confirmed_version INTEGER,
			admin_approved_by INTEGER,admin_approved_at DATETIME,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		"CREATE TABLE class_plan_slots(plan_id INTEGER,hari INTEGER,jam_mulai TEXT,jam_selesai TEXT)",
		"CREATE TABLE class_plan_levels(plan_id INTEGER,level_kode TEXT)",
		"CREATE TABLE class_plan_students(plan_id INTEGER,santri_id INTEGER,level_kode TEXT,status TEXT,schedule_approved INTEGER)",
		"CREATE TABLE class_plan_events(plan_id INTEGER,actor_user_id INTEGER,event_type TEXT,from_status TEXT,to_status TEXT,reason TEXT)",
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestClassPlanWorkflowVersionAndApprovalCAS(t *testing.T) {
	ctx := context.Background()
	q := NewQuerier(workflowQueryDB(t))
	p := ClassPlanWorkflowRow{Program: "reguler", Angkatan: "2026-A", Gender: "P", GuruID: 10, Frequency: 2, MinQuota: 8, Capacity: 10}
	id, err := q.CreateClassPlanWorkflow(ctx, 17, p, []ClassPlanWorkflowSlot{
		{Day: 2, Start: "19:30", End: "21:00"}, {Day: 5, Start: "19:30", End: "21:00"},
	}, []int64{1, 2, 3, 4, 5, 6, 7, 8}, "1")
	if err != nil {
		t.Fatal(err)
	}
	items, err := q.ListClassPlanWorkflow(ctx)
	if err != nil || len(items) != 1 || items[0].StudentCount != 8 || items[0].Version != 1 || items[0].Status != "draft" {
		t.Fatalf("unexpected draft: %+v %v", items, err)
	}
	if ok, err := q.ClassPlanTransition(ctx, id, 99, "draft", "pending_coordinator", 17); err != nil || ok {
		t.Fatalf("stale version should be rejected: %v %v", ok, err)
	}
	if ok, err := q.ClassPlanTransition(ctx, id, 1, "draft", "pending_coordinator", 17); err != nil || !ok {
		t.Fatalf("submit failed: %v %v", ok, err)
	}
	if ok, err := q.ClassPlanTransition(ctx, id, 1, "draft", "pending_coordinator", 17); err != nil || ok {
		t.Fatalf("duplicate submit should be rejected: %v %v", ok, err)
	}
	if ok, err := q.ClassPlanTransition(ctx, id, 1, "pending_coordinator", "pending_admin", 23); err != nil || !ok {
		t.Fatalf("coordinator confirm failed: %v %v", ok, err)
	}
	if ok, err := q.ClassPlanTransition(ctx, id, 1, "pending_coordinator", "pending_admin", 23); err != nil || ok {
		t.Fatalf("duplicate coordinator confirm should be rejected: %v %v", ok, err)
	}
	if ok, err := q.ClassPlanTransition(ctx, id, 1, "pending_admin", "admin_approved", 17); err != nil || !ok {
		t.Fatalf("admin approval failed: %v %v", ok, err)
	}
	if ok, err := q.ClassPlanTransition(ctx, id, 1, "pending_admin", "admin_approved", 17); err != nil || ok {
		t.Fatalf("duplicate admin approval should be rejected: %v %v", ok, err)
	}
	got, err := q.GetClassPlanWorkflow(ctx, id)
	if err != nil || got.Status != "pending_admin" || got.ConfirmedBy != 23 ||
		got.ConfirmedVersion != 1 || got.ApprovedBy != 17 {
		t.Fatalf("approval record unexpected: %+v %v", got, err)
	}
	if err := q.AppendClassPlanWorkflowEvent(ctx, id, 17, "planning.admin_approved", "pending_admin", "pending_admin", "approved"); err != nil {
		t.Fatal(err)
	}
	var events int
	if err := q.Queries.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM class_plan_events WHERE plan_id=?", id).Scan(&events); err != nil || events != 1 {
		t.Fatalf("audit event missing: %d %v", events, err)
	}
}

func TestClassPlanWorkflowUnderQuotaCannotJumpToCoordinator(t *testing.T) {
	q := NewQuerier(workflowQueryDB(t))
	id, err := q.CreateClassPlanWorkflow(context.Background(), 17, ClassPlanWorkflowRow{
		Program: "reguler", Angkatan: "2026-A", Gender: "P", GuruID: 10, Frequency: 1,
		MinQuota: 8, Capacity: 10,
	}, []ClassPlanWorkflowSlot{{Day: 6, Start: "19:30", End: "21:00"}}, []int64{1, 2, 3}, "1")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := q.ClassPlanTransition(context.Background(), id, 1, "draft", "pending_exception", 17)
	if err != nil || !ok {
		t.Fatalf("exception request failed: %v %v", ok, err)
	}
	ok, err = q.ClassPlanTransition(context.Background(), id, 1, "pending_coordinator", "pending_admin", 23)
	if err != nil || ok {
		t.Fatalf("exception cannot skip coordinator: %v %v", ok, err)
	}
}
