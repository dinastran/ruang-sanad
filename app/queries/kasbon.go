package queries

import (
	"context"
	"database/sql"
)

// Hand-written like ujroh.go: `sqlc generate` currently fails on an unrelated
// query, so these statements live here instead of queries/*.sql.

type KasbonRow struct {
	ID             int64
	GuruID         int64
	GuruNama       string
	Tanggal        string
	Nominal        int64
	Cicilan        int64
	MulaiBulan     string
	Keterangan     string
	DibuatOlehNama string
	CreatedAt      sql.NullTime
	DibatalkanAt   sql.NullTime
	// Terpotong is the sum of deductions frozen in locked months.
	Terpotong int64
}

const kasbonSelect = `
SELECT k.id, k.guru_id, COALESCE(g.nama, ''), k.tanggal, k.nominal, k.cicilan, k.mulai_bulan, k.keterangan,
	COALESCE(u.name, ''), k.created_at, k.dibatalkan_at,
	COALESCE((SELECT SUM(p.nominal) FROM kasbon_potongan p WHERE p.kasbon_id = k.id), 0)
FROM kasbon k
LEFT JOIN guru g ON g.id = k.guru_id
LEFT JOIN users u ON u.id = k.dibuat_oleh`

func scanKasbon(scan func(dest ...any) error) (KasbonRow, error) {
	var r KasbonRow
	err := scan(&r.ID, &r.GuruID, &r.GuruNama, &r.Tanggal, &r.Nominal, &r.Cicilan, &r.MulaiBulan, &r.Keterangan,
		&r.DibuatOlehNama, &r.CreatedAt, &r.DibatalkanAt, &r.Terpotong)
	return r, err
}

func (q *Querier) ListKasbon(ctx context.Context) ([]KasbonRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, kasbonSelect+` ORDER BY k.dibatalkan_at IS NOT NULL, k.tanggal DESC, k.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []KasbonRow{}
	for rows.Next() {
		r, err := scanKasbon(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Querier) GetKasbon(ctx context.Context, id int64) (KasbonRow, error) {
	return scanKasbon(q.Queries.db.QueryRowContext(ctx, kasbonSelect+` WHERE k.id = ?`, id).Scan)
}

type KasbonUntukBulanRow struct {
	ID         int64
	GuruID     int64
	Tanggal    string
	Nominal    int64
	Cicilan    int64
	Keterangan string
	// Terpotong excludes bulan itself, so it is the balance before bulan.
	Terpotong int64
}

// ListKasbonUntukBulan returns active kasbon that may be deducted in bulan,
// oldest first, with what other (locked) months already deducted.
func (q *Querier) ListKasbonUntukBulan(ctx context.Context, bulan string) ([]KasbonUntukBulanRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT k.id, k.guru_id, k.tanggal, k.nominal, k.cicilan, k.keterangan,
	COALESCE((SELECT SUM(p.nominal) FROM kasbon_potongan p WHERE p.kasbon_id = k.id AND p.bulan != ?), 0)
FROM kasbon k
WHERE k.dibatalkan_at IS NULL AND k.mulai_bulan <= ?
ORDER BY k.guru_id, k.tanggal, k.id`, bulan, bulan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []KasbonUntukBulanRow{}
	for rows.Next() {
		var r KasbonUntukBulanRow
		if err := rows.Scan(&r.ID, &r.GuruID, &r.Tanggal, &r.Nominal, &r.Cicilan, &r.Keterangan, &r.Terpotong); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Querier) InsertKasbon(ctx context.Context, guruID int64, tanggal string, nominal, cicilan int64, mulaiBulan, keterangan string, userID int64) (int64, error) {
	res, err := q.Queries.db.ExecContext(ctx, `
INSERT INTO kasbon (guru_id, tanggal, nominal, cicilan, mulai_bulan, keterangan, dibuat_oleh) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		guruID, tanggal, nominal, cicilan, mulaiBulan, keterangan, userID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (q *Querier) UpdateKasbonCicilan(ctx context.Context, id, cicilan int64, keterangan string) error {
	_, err := q.Queries.db.ExecContext(ctx,
		`UPDATE kasbon SET cicilan = ?, keterangan = ? WHERE id = ? AND dibatalkan_at IS NULL`, cicilan, keterangan, id)
	return err
}

func (q *Querier) BatalkanKasbon(ctx context.Context, id, userID int64) error {
	_, err := q.Queries.db.ExecContext(ctx,
		`UPDATE kasbon SET dibatalkan_at = CURRENT_TIMESTAMP, dibatalkan_oleh = ? WHERE id = ? AND dibatalkan_at IS NULL`, userID, id)
	return err
}

type KasbonPotonganRow struct {
	KasbonID   int64
	GuruID     int64
	Nominal    int64
	Tanggal    string
	Total      int64
	Keterangan string
	// SisaSetelah is the kasbon balance after every locked deduction up to and including bulan.
	SisaSetelah int64
}

func (q *Querier) InsertKasbonPotongan(ctx context.Context, bulan string, kasbonID, guruID, nominal int64) error {
	_, err := q.Queries.db.ExecContext(ctx,
		`INSERT INTO kasbon_potongan (bulan, kasbon_id, guru_id, nominal) VALUES (?, ?, ?, ?)`, bulan, kasbonID, guruID, nominal)
	return err
}

func (q *Querier) ListKasbonPotongan(ctx context.Context, bulan string) ([]KasbonPotonganRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT p.kasbon_id, p.guru_id, p.nominal, k.tanggal, k.nominal, k.keterangan,
	k.nominal - COALESCE((SELECT SUM(p2.nominal) FROM kasbon_potongan p2 WHERE p2.kasbon_id = p.kasbon_id AND p2.bulan <= p.bulan), 0)
FROM kasbon_potongan p
JOIN kasbon k ON k.id = p.kasbon_id
WHERE p.bulan = ?
ORDER BY p.guru_id, k.tanggal, k.id`, bulan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []KasbonPotonganRow{}
	for rows.Next() {
		var r KasbonPotonganRow
		if err := rows.Scan(&r.KasbonID, &r.GuruID, &r.Nominal, &r.Tanggal, &r.Total, &r.Keterangan, &r.SisaSetelah); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type UjrohKasKeluarRow struct {
	Bulan   string
	Jumlah  int64
	Nominal int64
}

// KeuanganUjrohKasKeluarPerBulan is cash paid to guru per month: net ujroh
// (diterima) by payment date plus kasbon by the date it was handed out. Using
// diterima avoids counting a kasbon twice (once given, once deducted).
func (q *Querier) KeuanganUjrohKasKeluarPerBulan(ctx context.Context, dariBulan, sampaiBulan string) ([]UjrohKasKeluarRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT bulan, CAST(SUM(jumlah) AS INTEGER), CAST(SUM(nominal) AS INTEGER) FROM (
	SELECT substr(dibayar_at, 1, 7) AS bulan, COUNT(*) AS jumlah, COALESCE(SUM(diterima), 0) AS nominal
	FROM ujroh_guru
	WHERE dibayar_at IS NOT NULL
	GROUP BY substr(dibayar_at, 1, 7)
	UNION ALL
	SELECT substr(tanggal, 1, 7), 0, COALESCE(SUM(nominal), 0)
	FROM kasbon
	WHERE dibatalkan_at IS NULL
	GROUP BY substr(tanggal, 1, 7)
)
WHERE bulan >= ? AND bulan <= ?
GROUP BY bulan`, dariBulan, sampaiBulan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UjrohKasKeluarRow{}
	for rows.Next() {
		var r UjrohKasKeluarRow
		if err := rows.Scan(&r.Bulan, &r.Jumlah, &r.Nominal); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// KeuanganUjrohTerkunciBelumDibayarBersih is the net amount still owed for
// locked months.
func (q *Querier) KeuanganUjrohTerkunciBelumDibayarBersih(ctx context.Context) (jumlahGuru, nominal int64, err error) {
	err = q.Queries.db.QueryRowContext(ctx, `
SELECT COUNT(*), CAST(COALESCE(SUM(g.diterima), 0) AS INTEGER)
FROM ujroh_guru g
JOIN ujroh_bulan b ON b.bulan = g.bulan
WHERE b.dikunci_at IS NOT NULL AND g.dibayar_at IS NULL`).Scan(&jumlahGuru, &nominal)
	return jumlahGuru, nominal, err
}
