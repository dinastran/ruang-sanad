package queries

import (
	"context"
	"strings"
)

// SantriStatusInfo is the status snapshot shown next to a santri in lists
// owned by other modules (e.g. Daftar Tagihan).
type SantriStatusInfo struct {
	Status       string
	StatusAlasan string
	CutiMulai    string
	CutiSelesai  string
}

func (q *Querier) ListSantriStatusByIDs(ctx context.Context, santriIDs []int64) (map[int64]SantriStatusInfo, error) {
	out := map[int64]SantriStatusInfo{}
	if len(santriIDs) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(santriIDs))
	args := make([]interface{}, len(santriIDs))
	for i, id := range santriIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := "SELECT id, status, status_alasan, cuti_mulai, cuti_selesai FROM santri WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := q.Queries.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var info SantriStatusInfo
		if err := rows.Scan(&id, &info.Status, &info.StatusAlasan, &info.CutiMulai, &info.CutiSelesai); err != nil {
			return nil, err
		}
		out[id] = info
	}
	return out, rows.Err()
}
