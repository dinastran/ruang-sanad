package queries

import (
	"context"
	"database/sql"
	"time"
)

// Hand-written like riwayat_mengajar.go: `sqlc generate` currently fails on an
// unrelated query, so these statements live here instead of queries/*.sql.

// ujrohPengajarExpr resolves who taught a meeting, matching Riwayat Mengajar:
// the badal teacher when one is set, otherwise the guru linked to the user who
// started the meeting, otherwise the class guru.
const ujrohPengajarExpr = `CASE
	WHEN p.is_badal = 1 AND p.guru_pengganti_id IS NOT NULL THEN p.guru_pengganti_id
	ELSE COALESCE(actor_g.id, k.guru_id)
END`

const listUjrohPertemuanLive = `
SELECT
	p.id,
	p.kelas_id,
	k.nama_kelas,
	substr(p.tanggal, 1, 10) AS tanggal,
	p.pertemuan_ke,
	CASE WHEN p.is_badal = 1 AND p.guru_pengganti_id IS NOT NULL THEN 1 ELSE 0 END AS is_badal,
	g.id AS guru_id,
	COALESCE(g.nama, '') AS guru_nama,
	COALESCE(g.status, '') AS guru_status,
	COALESCE(tg.nominal, ut.nominal, 0) AS tarif,
	CASE WHEN tg.guru_id IS NULL THEN 0 ELSE 1 END AS tarif_khusus
FROM pertemuan p
JOIN kelas k ON k.id = p.kelas_id
LEFT JOIN guru actor_g ON actor_g.user_id = p.dibuat_oleh
LEFT JOIN guru g ON g.id = ` + ujrohPengajarExpr + `
LEFT JOIN ujroh_tarif_guru tg ON tg.guru_id = g.id
LEFT JOIN ujroh_tarif ut ON ut.status = g.status
WHERE p.status = 'selesai' AND substr(p.tanggal, 1, 7) = ?
ORDER BY guru_nama, tanggal, p.id
`

type UjrohPertemuanLiveRow struct {
	PertemuanID int64
	KelasID     int64
	KelasNama   string
	Tanggal     string
	PertemuanKe int64
	IsBadal     int64
	GuruID      sql.NullInt64
	GuruNama    string
	GuruStatus  string
	Tarif       int64
	TarifKhusus int64
}

// ListUjrohPertemuanLive returns every completed meeting in bulan (YYYY-MM)
// with the teacher who earns it and that teacher's current rate.
func (q *Querier) ListUjrohPertemuanLive(ctx context.Context, bulan string) ([]UjrohPertemuanLiveRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, listUjrohPertemuanLive, bulan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UjrohPertemuanLiveRow{}
	for rows.Next() {
		var r UjrohPertemuanLiveRow
		if err := rows.Scan(&r.PertemuanID, &r.KelasID, &r.KelasNama, &r.Tanggal, &r.PertemuanKe, &r.IsBadal,
			&r.GuruID, &r.GuruNama, &r.GuruStatus, &r.Tarif, &r.TarifKhusus); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Querier) CountPertemuanBerlangsungBulan(ctx context.Context, bulan string) (int64, error) {
	var n int64
	err := q.Queries.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pertemuan WHERE status = 'berlangsung' AND substr(tanggal, 1, 7) = ?`, bulan).Scan(&n)
	return n, err
}

type UjrohBulanRow struct {
	Bulan       string
	DikunciAt   sql.NullTime
	DikunciOleh string
	DibukaAt    sql.NullTime
	DibukaOleh  string
}

func (q *Querier) GetUjrohBulan(ctx context.Context, bulan string) (UjrohBulanRow, error) {
	var r UjrohBulanRow
	err := q.Queries.db.QueryRowContext(ctx, `
SELECT b.bulan, b.dikunci_at, COALESCE(uk.name, ''), b.dibuka_at, COALESCE(ub.name, '')
FROM ujroh_bulan b
LEFT JOIN users uk ON uk.id = b.dikunci_oleh
LEFT JOIN users ub ON ub.id = b.dibuka_oleh
WHERE b.bulan = ?`, bulan).Scan(&r.Bulan, &r.DikunciAt, &r.DikunciOleh, &r.DibukaAt, &r.DibukaOleh)
	return r, err
}

// KunciUjrohBulan marks bulan locked. It returns false when it was already locked.
func (q *Querier) KunciUjrohBulan(ctx context.Context, bulan string, userID int64, at time.Time) (bool, error) {
	if _, err := q.Queries.db.ExecContext(ctx,
		`INSERT INTO ujroh_bulan (bulan) VALUES (?) ON CONFLICT(bulan) DO NOTHING`, bulan); err != nil {
		return false, err
	}
	res, err := q.Queries.db.ExecContext(ctx,
		`UPDATE ujroh_bulan SET dikunci_at = ?, dikunci_oleh = ? WHERE bulan = ? AND dikunci_at IS NULL`, at, userID, bulan)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// BukaUjrohBulan unlocks bulan and drops its snapshot. It returns false when
// bulan was not locked.
func (q *Querier) BukaUjrohBulan(ctx context.Context, bulan string, userID int64, at time.Time) (bool, error) {
	res, err := q.Queries.db.ExecContext(ctx,
		`UPDATE ujroh_bulan SET dikunci_at = NULL, dikunci_oleh = NULL, dibuka_at = ?, dibuka_oleh = ? WHERE bulan = ? AND dikunci_at IS NOT NULL`,
		at, userID, bulan)
	if err != nil {
		return false, err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return false, err
	}
	if _, err := q.Queries.db.ExecContext(ctx, `DELETE FROM ujroh_pertemuan WHERE bulan = ?`, bulan); err != nil {
		return false, err
	}
	if _, err := q.Queries.db.ExecContext(ctx, `DELETE FROM ujroh_guru WHERE bulan = ?`, bulan); err != nil {
		return false, err
	}
	return true, nil
}

type UjrohGuruRow struct {
	GuruID          int64
	GuruNama        string
	GuruStatus      string
	Tarif           int64
	JumlahPertemuan int64
	JumlahBadal     int64
	Total           int64
	DibayarAt       sql.NullTime
	DibayarOleh     string
	CatatanBayar    string
}

func (q *Querier) InsertUjrohGuru(ctx context.Context, bulan string, r UjrohGuruRow) error {
	_, err := q.Queries.db.ExecContext(ctx, `
INSERT INTO ujroh_guru (bulan, guru_id, guru_nama, guru_status, tarif, jumlah_pertemuan, jumlah_badal, total)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		bulan, r.GuruID, r.GuruNama, r.GuruStatus, r.Tarif, r.JumlahPertemuan, r.JumlahBadal, r.Total)
	return err
}

func (q *Querier) ListUjrohGuru(ctx context.Context, bulan string) ([]UjrohGuruRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT ug.guru_id, ug.guru_nama, ug.guru_status, ug.tarif, ug.jumlah_pertemuan, ug.jumlah_badal, ug.total,
	ug.dibayar_at, COALESCE(u.name, ''), ug.catatan_bayar
FROM ujroh_guru ug
LEFT JOIN users u ON u.id = ug.dibayar_oleh
WHERE ug.bulan = ?
ORDER BY ug.guru_nama`, bulan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UjrohGuruRow{}
	for rows.Next() {
		var r UjrohGuruRow
		if err := rows.Scan(&r.GuruID, &r.GuruNama, &r.GuruStatus, &r.Tarif, &r.JumlahPertemuan, &r.JumlahBadal, &r.Total,
			&r.DibayarAt, &r.DibayarOleh, &r.CatatanBayar); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Querier) CountUjrohGuruDibayar(ctx context.Context, bulan string) (int64, error) {
	var n int64
	err := q.Queries.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ujroh_guru WHERE bulan = ? AND dibayar_at IS NOT NULL`, bulan).Scan(&n)
	return n, err
}

// SetUjrohGuruDibayar records or clears a payment. It returns false when the
// guru has no locked row for bulan.
func (q *Querier) SetUjrohGuruDibayar(ctx context.Context, bulan string, guruID int64, at sql.NullTime, userID sql.NullInt64, catatan string) (bool, error) {
	res, err := q.Queries.db.ExecContext(ctx,
		`UPDATE ujroh_guru SET dibayar_at = ?, dibayar_oleh = ?, catatan_bayar = ? WHERE bulan = ? AND guru_id = ?`,
		at, userID, catatan, bulan, guruID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

type UjrohPertemuanRow struct {
	GuruID      int64
	PertemuanID int64
	KelasID     int64
	KelasNama   string
	Tanggal     string
	PertemuanKe int64
	IsBadal     int64
	Tarif       int64
}

func (q *Querier) InsertUjrohPertemuan(ctx context.Context, bulan string, r UjrohPertemuanRow) error {
	_, err := q.Queries.db.ExecContext(ctx, `
INSERT INTO ujroh_pertemuan (bulan, guru_id, pertemuan_id, kelas_id, kelas_nama, tanggal, pertemuan_ke, is_badal, tarif)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		bulan, r.GuruID, r.PertemuanID, r.KelasID, r.KelasNama, r.Tanggal, r.PertemuanKe, r.IsBadal, r.Tarif)
	return err
}

func (q *Querier) ListUjrohPertemuan(ctx context.Context, bulan string) ([]UjrohPertemuanRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT guru_id, pertemuan_id, kelas_id, kelas_nama, tanggal, pertemuan_ke, is_badal, tarif
FROM ujroh_pertemuan
WHERE bulan = ?
ORDER BY tanggal, pertemuan_id`, bulan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UjrohPertemuanRow{}
	for rows.Next() {
		var r UjrohPertemuanRow
		if err := rows.Scan(&r.GuruID, &r.PertemuanID, &r.KelasID, &r.KelasNama, &r.Tanggal, &r.PertemuanKe, &r.IsBadal, &r.Tarif); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type UjrohTarifRow struct {
	Status  string
	Nominal int64
}

func (q *Querier) ListUjrohTarif(ctx context.Context) ([]UjrohTarifRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `SELECT status, nominal FROM ujroh_tarif ORDER BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UjrohTarifRow{}
	for rows.Next() {
		var r UjrohTarifRow
		if err := rows.Scan(&r.Status, &r.Nominal); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Querier) UpsertUjrohTarif(ctx context.Context, status string, nominal, userID int64) error {
	_, err := q.Queries.db.ExecContext(ctx, `
INSERT INTO ujroh_tarif (status, nominal, diubah_oleh, updated_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(status) DO UPDATE SET nominal = excluded.nominal, diubah_oleh = excluded.diubah_oleh, updated_at = CURRENT_TIMESTAMP`,
		status, nominal, userID)
	return err
}

type UjrohTarifGuruRow struct {
	GuruID      int64
	Nama        string
	Status      string
	IsAktif     int64
	TarifKhusus sql.NullInt64
}

func (q *Querier) ListUjrohTarifGuru(ctx context.Context) ([]UjrohTarifGuruRow, error) {
	rows, err := q.Queries.db.QueryContext(ctx, `
SELECT g.id, g.nama, g.status, g.is_aktif, tg.nominal
FROM guru g
LEFT JOIN ujroh_tarif_guru tg ON tg.guru_id = g.id
WHERE g.is_aktif = 1 OR tg.guru_id IS NOT NULL
ORDER BY g.nama`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UjrohTarifGuruRow{}
	for rows.Next() {
		var r UjrohTarifGuruRow
		if err := rows.Scan(&r.GuruID, &r.Nama, &r.Status, &r.IsAktif, &r.TarifKhusus); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *Querier) UpsertUjrohTarifGuru(ctx context.Context, guruID, nominal, userID int64) error {
	_, err := q.Queries.db.ExecContext(ctx, `
INSERT INTO ujroh_tarif_guru (guru_id, nominal, diubah_oleh, updated_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(guru_id) DO UPDATE SET nominal = excluded.nominal, diubah_oleh = excluded.diubah_oleh, updated_at = CURRENT_TIMESTAMP`,
		guruID, nominal, userID)
	return err
}

func (q *Querier) DeleteUjrohTarifGuru(ctx context.Context, guruID int64) error {
	_, err := q.Queries.db.ExecContext(ctx, `DELETE FROM ujroh_tarif_guru WHERE guru_id = ?`, guruID)
	return err
}
