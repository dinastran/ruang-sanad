package queries

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func availabilityTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE guru_availability (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		guru_id INTEGER NOT NULL,
		hari INTEGER NOT NULL CHECK(hari BETWEEN 1 AND 7),
		jam_mulai TEXT NOT NULL, jam_selesai TEXT NOT NULL,
		berlaku_mulai DATE NOT NULL, berlaku_sampai DATE,
		confirmed_by INTEGER, confirmed_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(guru_id, hari, jam_mulai, jam_selesai, berlaku_mulai)
	)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestGuruAvailabilityQueriesCRUDAndOwnership(t *testing.T) {
	ctx := context.Background()
	q := NewQuerier(availabilityTestDB(t))
	slot := GuruAvailabilityRow{
		GuruID: 10, Hari: 6, JamMulai: "19:30", JamSelesai: "21:00",
		BerlakuMulai: "2026-11-01", BerlakuSampai: "2026-12-31",
	}
	id, err := q.CreateGuruAvailability(ctx, slot, 5)
	if err != nil {
		t.Fatal(err)
	}
	list, err := q.ListGuruAvailability(ctx, 10)
	if err != nil || len(list) != 1 || list[0].ConfirmedBy != 5 || list[0].BerlakuSampai != slot.BerlakuSampai {
		t.Fatalf("list mismatch: %+v, %v", list, err)
	}
	if _, err := q.GetGuruAvailability(ctx, 11, id); err != sql.ErrNoRows {
		t.Fatalf("slot must not be readable through other teacher: %v", err)
	}
	count, err := q.CountGuruAvailabilityOverlap(ctx, 10, 0, 6, "20:00", "21:30", "2026-11-01", "")
	if err != nil || count != 1 {
		t.Fatalf("overlap detection failed: %d, %v", count, err)
	}
	count, err = q.CountGuruAvailabilityOverlap(ctx, 10, id, 6, "20:00", "21:30", "2026-11-01", "")
	if err != nil || count != 0 {
		t.Fatalf("edit must exclude current slot: %d, %v", count, err)
	}
	count, err = q.CountGuruAvailabilityOverlap(ctx, 10, 0, 6, "21:00", "22:30", "2026-11-01", "")
	if err != nil || count != 0 {
		t.Fatalf("adjacent interval must be allowed: %d, %v", count, err)
	}
	count, err = q.CountGuruAvailabilityOverlap(ctx, 10, 0, 6, "20:00", "21:30", "2027-01-01", "")
	if err != nil || count != 0 {
		t.Fatalf("non-overlapping effective period must be allowed: %d, %v", count, err)
	}
	slot.ID = id
	slot.JamMulai = "18:00"
	slot.JamSelesai = "19:30"
	affected, err := q.UpdateGuruAvailability(ctx, slot, 6)
	if err != nil || affected != 1 {
		t.Fatalf("update failed: %d, %v", affected, err)
	}
	other, err := q.GetGuruAvailability(ctx, 10, id)
	if err != nil || other.ConfirmedBy != 6 || other.JamMulai != "18:00" {
		t.Fatalf("updated slot mismatch: %+v, %v", other, err)
	}
	affected, err = q.DeleteGuruAvailability(ctx, 11, id)
	if err != nil || affected != 0 {
		t.Fatalf("cross-teacher delete must not affect row: %d, %v", affected, err)
	}
	affected, err = q.DeleteGuruAvailability(ctx, 10, id)
	if err != nil || affected != 1 {
		t.Fatalf("delete failed: %d, %v", affected, err)
	}
}
