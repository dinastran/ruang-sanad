package queries

import (
	"context"
	"database/sql"
	"time"
)

type RoutineScheduleRow struct {
	ID             int64
	KelasID        int64
	Hari           int64
	JamMulai       string
	BerlakuMulai   time.Time
	BerlakuSampai  sql.NullTime
	IsAktif        int64
	DibuatOleh     sql.NullInt64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (q *Querier) ListRoutineSchedulesByClass(ctx context.Context, kelasID int64) ([]RoutineScheduleRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT id, kelas_id, hari, jam_mulai, berlaku_mulai, berlaku_sampai,
       is_aktif, dibuat_oleh, created_at, updated_at
FROM kelas_jadwal_rutin
WHERE kelas_id = ? AND is_aktif = 1 AND berlaku_sampai IS NULL
ORDER BY hari, jam_mulai, id
`, kelasID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []RoutineScheduleRow{}
	for rows.Next() {
		var row RoutineScheduleRow
		if err := rows.Scan(
			&row.ID, &row.KelasID, &row.Hari, &row.JamMulai,
			&row.BerlakuMulai, &row.BerlakuSampai, &row.IsAktif,
			&row.DibuatOleh, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (q *Querier) ListRoutineSchedulesForRange(ctx context.Context, start, end time.Time, kelasID sql.NullInt64) ([]RoutineScheduleRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT r.id, r.kelas_id, r.hari, r.jam_mulai, r.berlaku_mulai, r.berlaku_sampai,
       r.is_aktif, r.dibuat_oleh, r.created_at, r.updated_at
FROM kelas_jadwal_rutin r
JOIN kelas k ON k.id = r.kelas_id
WHERE k.is_aktif = 1
  AND r.berlaku_mulai <= ?
  AND (r.berlaku_sampai IS NULL OR r.berlaku_sampai >= ?)
  AND (? IS NULL OR r.kelas_id = ?)
ORDER BY r.kelas_id, r.berlaku_mulai, r.hari, r.jam_mulai, r.id
`, end, start, kelasID, kelasID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []RoutineScheduleRow{}
	for rows.Next() {
		var row RoutineScheduleRow
		if err := rows.Scan(
			&row.ID, &row.KelasID, &row.Hari, &row.JamMulai,
			&row.BerlakuMulai, &row.BerlakuSampai, &row.IsAktif,
			&row.DibuatOleh, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (q *Querier) CreateRoutineSchedule(ctx context.Context, kelasID, hari int64, jamMulai string, berlakuMulai time.Time, userID sql.NullInt64) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, `
INSERT INTO kelas_jadwal_rutin
(kelas_id, hari, jam_mulai, berlaku_mulai, dibuat_oleh)
VALUES (?, ?, ?, ?, ?)
`, kelasID, hari, jamMulai, berlakuMulai, userID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (q *Querier) CloseActiveRoutineSchedules(ctx context.Context, kelasID int64, berlakuSampai time.Time) error {
	_, err := q.Queries.db.ExecContext(ctx, `
UPDATE kelas_jadwal_rutin
SET is_aktif = 0, berlaku_sampai = ?, updated_at = CURRENT_TIMESTAMP
WHERE kelas_id = ? AND is_aktif = 1 AND berlaku_sampai IS NULL
`, berlakuSampai, kelasID)
	return err
}

func (q *Querier) DeleteFutureRoutineSchedules(ctx context.Context, kelasID int64, afterDate time.Time) error {
	_, err := q.Queries.db.ExecContext(ctx, `
DELETE FROM kelas_jadwal_rutin
WHERE kelas_id = ?
  AND is_aktif = 1
  AND berlaku_mulai > ?
`, kelasID, afterDate)
	return err
}

func (q *Querier) DeleteFutureRoutineOccurrences(ctx context.Context, kelasID int64, afterDate time.Time) error {
	_, err := q.Queries.db.ExecContext(ctx, `
DELETE FROM jadwal_pertemuan
WHERE kelas_id = ?
  AND is_otomatis = 1
  AND status = 'dijadwalkan'
  AND is_reschedule = 0
  AND guru_pengganti_id IS NULL
  AND tanggal > ?
`, kelasID, afterDate)
	return err
}

func (q *Querier) CreateExtraSchedule(ctx context.Context, kelasID int64, tanggal time.Time, jamMulai, catatan string, userID sql.NullInt64) (int64, error) {
	row := q.Queries.db.QueryRowContext(ctx, `
INSERT INTO jadwal_pertemuan
(kelas_id, tanggal, jam_mulai, catatan, dibuat_oleh, is_tambahan)
SELECT ?, ?, ?, ?, ?, 1
WHERE NOT EXISTS (
    SELECT 1
    FROM jadwal_pertemuan
    WHERE kelas_id = ?
      AND tanggal = ?
      AND jam_mulai = ?
      AND status IN ('dijadwalkan', 'diproses', 'dimulai')
)
RETURNING id
`, kelasID, tanggal, jamMulai, catatan, userID, kelasID, tanggal, jamMulai)
	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (q *Querier) ListExtraScheduleIDsForRange(ctx context.Context, start, end time.Time) (map[int64]bool, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT id
FROM jadwal_pertemuan
WHERE is_tambahan = 1
  AND substr(tanggal, 1, 10) >= substr(?, 1, 10)
  AND substr(tanggal, 1, 10) <= substr(?, 1, 10)
`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

func (q *Querier) CreateRoutineOccurrence(ctx context.Context, routineID, kelasID int64, tanggal time.Time, jamMulai string) (bool, error) {
	adopted, err := q.Queries.db.ExecContext(ctx, `
UPDATE jadwal_pertemuan
SET jadwal_rutin_id = ?, tanggal_rutin = ?, is_otomatis = 1, is_tambahan = 0,
    jadwal_kelas_berubah = 0, updated_at = CURRENT_TIMESTAMP
WHERE id = (
    SELECT id
    FROM jadwal_pertemuan
    WHERE kelas_id = ?
      AND tanggal = ?
      AND jam_mulai = ?
      AND jadwal_rutin_id IS NULL
      AND status = 'dijadwalkan'
      AND is_reschedule = 0
      AND guru_pengganti_id IS NULL
    ORDER BY id
    LIMIT 1
)
`, routineID, tanggal, kelasID, tanggal, jamMulai)
	if err != nil {
		return false, err
	}
	if rows, err := adopted.RowsAffected(); err != nil {
		return false, err
	} else if rows > 0 {
		return true, nil
	}

	result, err := q.Queries.db.ExecContext(ctx, `
INSERT OR IGNORE INTO jadwal_pertemuan
(kelas_id, tanggal, jam_mulai, catatan, dibuat_oleh, jadwal_rutin_id, tanggal_rutin, is_otomatis)
VALUES (?, ?, ?, '', NULL, ?, ?, 1)
`, kelasID, tanggal, jamMulai, routineID, tanggal)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}
