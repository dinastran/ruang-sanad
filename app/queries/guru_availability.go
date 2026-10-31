package queries

import (
	"context"
	"database/sql"
)

type GuruAvailabilityRow struct {
	ID            int64  `json:"id"`
	GuruID        int64  `json:"guru_id"`
	Hari          int64  `json:"hari"`
	JamMulai      string `json:"jam_mulai"`
	JamSelesai    string `json:"jam_selesai"`
	BerlakuMulai  string `json:"berlaku_mulai"`
	BerlakuSampai string `json:"berlaku_sampai"`
	ConfirmedBy   int64  `json:"confirmed_by"`
	ConfirmedAt   string `json:"confirmed_at"`
}

const guruAvailabilitySelect = `SELECT id, guru_id, hari, jam_mulai, jam_selesai,
	CAST(berlaku_mulai AS TEXT), COALESCE(CAST(berlaku_sampai AS TEXT), ''),
	COALESCE(confirmed_by, 0), COALESCE(CAST(confirmed_at AS TEXT), '')
	FROM guru_availability`

func scanGuruAvailability(scan func(...any) error) (GuruAvailabilityRow, error) {
	var r GuruAvailabilityRow
	err := scan(&r.ID, &r.GuruID, &r.Hari, &r.JamMulai, &r.JamSelesai,
		&r.BerlakuMulai, &r.BerlakuSampai, &r.ConfirmedBy, &r.ConfirmedAt)
	return r, err
}

func (q *Querier) ListGuruAvailability(ctx context.Context, guruID int64) ([]GuruAvailabilityRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, guruAvailabilitySelect+
		" WHERE guru_id = ? ORDER BY hari, jam_mulai, berlaku_mulai", guruID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]GuruAvailabilityRow, 0)
	for rows.Next() {
		item, err := scanGuruAvailability(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (q *Querier) GetGuruAvailability(ctx context.Context, guruID, id int64) (GuruAvailabilityRow, error) {
	return scanGuruAvailability(q.Queries.db.QueryRowContext(ctx,
		guruAvailabilitySelect+" WHERE guru_id = ? AND id = ?", guruID, id).Scan)
}

// CountGuruAvailabilityOverlap checks both time intervals and date-effective ranges.
// A zero end date means open-ended. The excluded ID is used during updates.
func (q *Querier) CountGuruAvailabilityOverlap(ctx context.Context, guruID, excludeID, day int64, start, end, validFrom, validUntil string) (int64, error) {
	const statement = `SELECT COUNT(*) FROM guru_availability
		WHERE guru_id = ? AND id != ? AND hari = ?
		AND jam_mulai < ? AND ? < jam_selesai
		AND (berlaku_sampai IS NULL OR berlaku_sampai >= ?)
		AND (? = '' OR berlaku_mulai <= ?)`
	var count int64
	err := q.Queries.db.QueryRowContext(ctx, statement,
		guruID, excludeID, day, end, start, validFrom, validUntil, validUntil).Scan(&count)
	return count, err
}

func (q *Querier) CreateGuruAvailability(ctx context.Context, r GuruAvailabilityRow, actorID int64) (int64, error) {
	var until sql.NullString
	if r.BerlakuSampai != "" {
		until = sql.NullString{String: r.BerlakuSampai, Valid: true}
	}
	result, err := q.Queries.db.ExecContext(ctx, `INSERT INTO guru_availability
		(guru_id, hari, jam_mulai, jam_selesai, berlaku_mulai, berlaku_sampai,
		 confirmed_by, confirmed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		r.GuruID, r.Hari, r.JamMulai, r.JamSelesai, r.BerlakuMulai, until, actorID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (q *Querier) UpdateGuruAvailability(ctx context.Context, r GuruAvailabilityRow, actorID int64) (int64, error) {
	var until sql.NullString
	if r.BerlakuSampai != "" {
		until = sql.NullString{String: r.BerlakuSampai, Valid: true}
	}
	result, err := q.Queries.db.ExecContext(ctx, `UPDATE guru_availability SET
		hari = ?, jam_mulai = ?, jam_selesai = ?, berlaku_mulai = ?, berlaku_sampai = ?,
		confirmed_by = ?, confirmed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND guru_id = ?`,
		r.Hari, r.JamMulai, r.JamSelesai, r.BerlakuMulai, until, actorID, r.ID, r.GuruID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (q *Querier) DeleteGuruAvailability(ctx context.Context, guruID, id int64) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx,
		"DELETE FROM guru_availability WHERE id = ? AND guru_id = ?", id, guruID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
