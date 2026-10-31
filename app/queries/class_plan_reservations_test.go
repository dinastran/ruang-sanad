package queries

import (
	"context"
	"os"
	"strings"
	"testing"
)

func reservationDB(t *testing.T) *Querier {
	t.Helper()
	db := workflowQueryDB(t)
	sqls := []string{
		"CREATE TABLE santri(id INTEGER PRIMARY KEY,kelas_id INTEGER,status TEXT NOT NULL)",
		"CREATE UNIQUE INDEX uq_live_class_plan_reservation ON class_plan_students(santri_id) WHERE status='reserved'",
	}
	for _, s := range sqls {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	for i := int64(1); i <= 12; i++ {
		if _, err := db.Exec("INSERT INTO santri VALUES (?,NULL,'aktif')", i); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile("../../migrations/0099_guard_class_plan_reservations.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, _, ok := strings.Cut(string(raw), "-- +goose Down")
	if !ok {
		t.Fatal("missing Goose Down section")
	}
	if _, err := db.Exec(up); err != nil {
		t.Fatalf("apply actual 0099 migration: %v", err)
	}
	return NewQuerier(db)
}

func approvedPlan(t *testing.T, q *Querier, ids []int64) int64 {
	t.Helper()
	ctx := context.Background()
	id, err := q.CreateClassPlanWorkflow(ctx, 17, ClassPlanWorkflowRow{
		Program: "reguler", Angkatan: "A", Gender: "P", GuruID: 10,
		Frequency: 1, MinQuota: 8, Capacity: 10,
	}, []ClassPlanWorkflowSlot{{Day: 2, Start: "19:30", End: "21:00"}}, ids, "1")
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		from, to string
		actor    int64
	}{
		{"draft", "pending_coordinator", 17},
		{"pending_coordinator", "pending_admin", 23},
		{"pending_admin", "admin_approved", 17},
	} {
		ok, err := q.ClassPlanTransition(ctx, id, 1, step.from, step.to, step.actor)
		if err != nil || !ok {
			t.Fatalf("%s transition: %v %v", step.to, ok, err)
		}
	}
	return id
}

func TestClassPlanReservationGuardsLegacyClassChangesAndDelete(t *testing.T) {
	ctx := context.Background()
	q := reservationDB(t)
	ids := []int64{1, 2, 3, 4, 5, 6, 7, 8}
	id := approvedPlan(t, q, ids)
	tx, err := q.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	txQ := q.WithTx(tx)
	n, err := txQ.ReserveApprovedClassPlanCandidates(ctx, id, 1)
	if err != nil || n != 8 {
		t.Fatalf("reservation failed: %d %v", n, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	reserved, err := q.ClassPlanReservationCount(ctx, id)
	if err != nil || reserved != 8 {
		t.Fatalf("missing reservations: %d %v", reserved, err)
	}
	if _, err := q.ReserveApprovedClassPlanCandidates(ctx, id, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := q.Queries.db.ExecContext(ctx, "UPDATE santri SET kelas_id=99 WHERE id=1"); err == nil {
		t.Fatal("legacy change must be blocked for reserved santri")
	}
	if _, err := q.Queries.db.ExecContext(ctx, "DELETE FROM santri WHERE id=1"); err == nil {
		t.Fatal("legacy deletion must be blocked for reserved santri")
	}
	if _, err := q.Queries.db.ExecContext(ctx, "UPDATE santri SET kelas_id=99 WHERE id=9"); err != nil {
		t.Fatalf("unreserved legacy updates must continue: %v", err)
	}
	tx, err = q.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	txQ = q.WithTx(tx)
	ok, err := txQ.CancelApprovedPlan(ctx, id, 1)
	if err != nil || !ok {
		t.Fatalf("cancel failed: %v %v", ok, err)
	}
	if err := txQ.ReleaseClassPlanReservations(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := q.Queries.db.ExecContext(ctx, "UPDATE santri SET kelas_id=88 WHERE id=1"); err != nil {
		t.Fatalf("cancel must release legacy assignment: %v", err)
	}
}

func TestClassPlanReservationRequiresApprovalAndSingleOwner(t *testing.T) {
	ctx := context.Background()
	q := reservationDB(t)
	ids := []int64{1, 2, 3, 4, 5, 6, 7, 8}
	id := approvedPlan(t, q, ids)
	if n, err := q.ReserveApprovedClassPlanCandidates(ctx, id, 2); err != nil || n != 0 {
		t.Fatalf("stale version must not reserve: %d %v", n, err)
	}
	if n, err := q.ReserveApprovedClassPlanCandidates(ctx, id, 1); err != nil || n != 8 {
		t.Fatalf("first reservation failed: %d %v", n, err)
	}
	if n, err := q.ReserveApprovedClassPlanCandidates(ctx, id, 1); err != nil || n != 0 {
		t.Fatalf("duplicate must be idempotent at SQL boundary: %d %v", n, err)
	}
	other := approvedPlan(t, q, []int64{1, 9, 10, 11})
	// The partial unique index prevents the second plan from claiming santri #1.
	tx, err := q.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	txQ := q.WithTx(tx)
	if _, err := txQ.ReserveApprovedClassPlanCandidates(ctx, other, 1); err == nil {
		t.Fatal("overlapping reservation should fail")
	}
	_ = tx.Rollback()
	count, err := q.ClassPlanReservationCount(ctx, other)
	if err != nil || count != 0 {
		t.Fatalf("failed multi-student reservation must roll back: %d %v", count, err)
	}
}
