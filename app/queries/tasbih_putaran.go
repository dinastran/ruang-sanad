package queries

import "context"

// Ditulis manual: parser sqlc (sqlite) belum mendukung CTE dengan alias ini.
// Tetap satu-satunya tempat SQL dieksekusi (lapisan Query).

type ListTasbihPutaranPaketParams struct {
	UserID int64
	Dari   string
}

type ListTasbihPutaranPaketRow struct {
	ID      int64
	Nama    string
	Selesai int64
}

// Putaran selesai = setiap item paket punya total jumlah >= target item dalam
// run yang sama (satu item boleh terbagi ke beberapa sesi karena jeda). Run
// dihitung dalam rentang bila item terakhirnya dimulai pada/sesudah Dari.
const listTasbihPutaranPaket = `
WITH item_tuntas AS (
    SELECT s.paket_id, s.paket_run_id, s.dzikir_id, MIN(s.tanggal_wib) AS mulai,
           (SELECT COUNT(DISTINCT i2.dzikir_id) FROM tasbih_paket_item i2 WHERE i2.paket_id = s.paket_id) AS n_item
    FROM tasbih_sesi s
    WHERE s.user_id = ? AND s.paket_run_id IS NOT NULL
    GROUP BY s.paket_id, s.paket_run_id, s.dzikir_id
    HAVING SUM(s.jumlah) >= (SELECT MAX(i.target) FROM tasbih_paket_item i WHERE i.paket_id = s.paket_id AND i.dzikir_id = s.dzikir_id)
),
run_selesai AS (
    SELECT paket_id, paket_run_id, MAX(mulai) AS mulai_terakhir
    FROM item_tuntas
    GROUP BY paket_id, paket_run_id
    HAVING COUNT(*) = MAX(n_item)
)
SELECT p.id, p.nama, COUNT(r.paket_run_id) AS selesai
FROM tasbih_paket p
LEFT JOIN run_selesai r ON r.paket_id = p.id AND r.mulai_terakhir >= ?
GROUP BY p.id, p.nama, p.urutan
ORDER BY p.urutan, p.id`

func (q *Queries) ListTasbihPutaranPaket(ctx context.Context, arg ListTasbihPutaranPaketParams) ([]ListTasbihPutaranPaketRow, error) {
	rows, err := q.db.QueryContext(ctx, listTasbihPutaranPaket, arg.UserID, arg.Dari)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ListTasbihPutaranPaketRow{}
	for rows.Next() {
		var r ListTasbihPutaranPaketRow
		if err := rows.Scan(&r.ID, &r.Nama, &r.Selesai); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
