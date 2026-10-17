package queries

import (
	"context"
)

// Hand-written like ujroh.go: `sqlc generate` currently fails on an unrelated
// query, so these statements live here instead of queries/*.sql.

type UjrohKomponenRow struct {
	ID             int64
	Nama           string
	Jenis          string
	NominalDefault int64
	IsAktif        int64
}

func (q *Querier) ListUjrohKomponen(ctx context.Context) ([]UjrohKomponenRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx,
		`SELECT id, nama, jenis, nominal_default, is_aktif FROM ujroh_komponen ORDER BY is_aktif DESC, jenis DESC, nama`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UjrohKomponenRow{}
	for rows.Next() {
		var r UjrohKomponenRow
		if err := rows.Scan(&r.ID, &r.Nama, &r.Jenis, &r.NominalDefault, &r.IsAktif); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Querier) GetUjrohKomponen(ctx context.Context, id int64) (UjrohKomponenRow, error) {
	var r UjrohKomponenRow
	err := q.Queries.db.QueryRowContext(ctx,
		`SELECT id, nama, jenis, nominal_default, is_aktif FROM ujroh_komponen WHERE id = ?`, id).
		Scan(&r.ID, &r.Nama, &r.Jenis, &r.NominalDefault, &r.IsAktif)
	return r, err
}

func (q *Querier) InsertUjrohKomponen(ctx context.Context, nama, jenis string, nominalDefault, userID int64) (int64, error) {
	res, err := q.Queries.db.ExecContext(ctx,
		`INSERT INTO ujroh_komponen (nama, jenis, nominal_default, diubah_oleh) VALUES (?, ?, ?, ?)`,
		nama, jenis, nominalDefault, userID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (q *Querier) UpdateUjrohKomponen(ctx context.Context, id int64, nama string, nominalDefault, isAktif, userID int64) (bool, error) {
	res, err := q.Queries.db.ExecContext(ctx, `
UPDATE ujroh_komponen SET nama = ?, nominal_default = ?, is_aktif = ?, diubah_oleh = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?`, nama, nominalDefault, isAktif, userID, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

type UjrohKomponenGuruRow struct {
	KomponenID int64
	GuruID     int64
	Nominal    int64
}

func (q *Querier) ListUjrohKomponenGuru(ctx context.Context) ([]UjrohKomponenGuruRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `SELECT komponen_id, guru_id, nominal FROM ujroh_komponen_guru`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UjrohKomponenGuruRow{}
	for rows.Next() {
		var r UjrohKomponenGuruRow
		if err := rows.Scan(&r.KomponenID, &r.GuruID, &r.Nominal); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Querier) UpsertUjrohKomponenGuru(ctx context.Context, komponenID, guruID, nominal, userID int64) error {
	_, err := q.Queries.db.ExecContext(ctx, `
INSERT INTO ujroh_komponen_guru (komponen_id, guru_id, nominal, diubah_oleh, updated_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(komponen_id, guru_id) DO UPDATE SET nominal = excluded.nominal, diubah_oleh = excluded.diubah_oleh, updated_at = CURRENT_TIMESTAMP`,
		komponenID, guruID, nominal, userID)
	return err
}

func (q *Querier) DeleteUjrohKomponenGuru(ctx context.Context, komponenID, guruID int64) error {
	_, err := q.Queries.db.ExecContext(ctx, `DELETE FROM ujroh_komponen_guru WHERE komponen_id = ? AND guru_id = ?`, komponenID, guruID)
	return err
}

// ujrohKomponenTetapAktif is the set of recurring components currently paid:
// active tetap component assigned to an active guru.
const ujrohKomponenTetapAktif = `
FROM ujroh_komponen_guru kg
JOIN ujroh_komponen k ON k.id = kg.komponen_id AND k.is_aktif = 1 AND k.jenis = 'tetap'
JOIN guru g ON g.id = kg.guru_id AND g.is_aktif = 1`

const listUjrohKomponenLive = `
SELECT g.id, g.nama, g.status, k.id, k.nama, k.jenis,
	COALESCE(kb.jumlah, 1), COALESCE(kb.tarif, kg.nominal), COALESCE(kb.catatan, ''),
	CASE WHEN kb.id IS NULL THEN 0 ELSE 1 END, kg.nominal
` + ujrohKomponenTetapAktif + `
LEFT JOIN ujroh_komponen_bulan kb ON kb.bulan = ? AND kb.guru_id = kg.guru_id AND kb.komponen_id = kg.komponen_id
UNION ALL
SELECT g.id, g.nama, g.status, k.id, k.nama, k.jenis, kb.jumlah, kb.tarif, kb.catatan, 1, 0
FROM ujroh_komponen_bulan kb
JOIN ujroh_komponen k ON k.id = kb.komponen_id
JOIN guru g ON g.id = kb.guru_id
WHERE kb.bulan = ? AND NOT EXISTS (
	SELECT 1 ` + ujrohKomponenTetapAktif + `
	WHERE kg.guru_id = kb.guru_id AND kg.komponen_id = kb.komponen_id
)
ORDER BY 2, 6 DESC, 5`

type UjrohKomponenLiveRow struct {
	GuruID       int64
	GuruNama     string
	GuruStatus   string
	KomponenID   int64
	KomponenNama string
	Jenis        string
	Jumlah       int64
	Tarif        int64
	Catatan      string
	DiubahBulan  int64
	// NominalTetap is the recurring amount for a tetap assignment (0 otherwise).
	NominalTetap int64
}

// ListUjrohKomponenLive returns every component line for bulan: recurring
// tetap amounts (with any adjustment for this month applied) plus the month's
// own entries.
func (q *Querier) ListUjrohKomponenLive(ctx context.Context, bulan string) ([]UjrohKomponenLiveRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, listUjrohKomponenLive, bulan, bulan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UjrohKomponenLiveRow{}
	for rows.Next() {
		var r UjrohKomponenLiveRow
		if err := rows.Scan(&r.GuruID, &r.GuruNama, &r.GuruStatus, &r.KomponenID, &r.KomponenNama, &r.Jenis,
			&r.Jumlah, &r.Tarif, &r.Catatan, &r.DiubahBulan, &r.NominalTetap); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Querier) UpsertUjrohKomponenBulan(ctx context.Context, bulan string, guruID, komponenID, jumlah, tarif int64, catatan string, userID int64) error {
	_, err := q.Queries.db.ExecContext(ctx, `
INSERT INTO ujroh_komponen_bulan (bulan, guru_id, komponen_id, jumlah, tarif, catatan, diubah_oleh, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(bulan, guru_id, komponen_id) DO UPDATE SET jumlah = excluded.jumlah, tarif = excluded.tarif,
	catatan = excluded.catatan, diubah_oleh = excluded.diubah_oleh, updated_at = CURRENT_TIMESTAMP`,
		bulan, guruID, komponenID, jumlah, tarif, catatan, userID)
	return err
}

func (q *Querier) DeleteUjrohKomponenBulan(ctx context.Context, bulan string, guruID, komponenID int64) error {
	_, err := q.Queries.db.ExecContext(ctx,
		`DELETE FROM ujroh_komponen_bulan WHERE bulan = ? AND guru_id = ? AND komponen_id = ?`, bulan, guruID, komponenID)
	return err
}

type UjrohGuruKomponenRow struct {
	GuruID       int64
	KomponenID   int64
	KomponenNama string
	Jenis        string
	Jumlah       int64
	Tarif        int64
	Total        int64
	Catatan      string
}

func (q *Querier) InsertUjrohGuruKomponen(ctx context.Context, bulan string, r UjrohGuruKomponenRow) error {
	_, err := q.Queries.db.ExecContext(ctx, `
INSERT INTO ujroh_guru_komponen (bulan, guru_id, komponen_id, komponen_nama, jenis, jumlah, tarif, total, catatan)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		bulan, r.GuruID, r.KomponenID, r.KomponenNama, r.Jenis, r.Jumlah, r.Tarif, r.Total, r.Catatan)
	return err
}

func (q *Querier) ListUjrohGuruKomponen(ctx context.Context, bulan string) ([]UjrohGuruKomponenRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT guru_id, komponen_id, komponen_nama, jenis, jumlah, tarif, total, catatan
FROM ujroh_guru_komponen WHERE bulan = ? ORDER BY jenis DESC, komponen_nama`, bulan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UjrohGuruKomponenRow{}
	for rows.Next() {
		var r UjrohGuruKomponenRow
		if err := rows.Scan(&r.GuruID, &r.KomponenID, &r.KomponenNama, &r.Jenis, &r.Jumlah, &r.Tarif, &r.Total, &r.Catatan); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
