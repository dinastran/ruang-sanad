package queries

// Handwritten queries for CRM programs (Tahsin Reguler vs short programs such
// as Sepekan Bersama Al-Fatihah), their sessions and public attendance.
import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
)

// LeadProgramRegulerID is the built-in Tahsin Reguler program (migration 0116).
const LeadProgramRegulerID int64 = 1

// LeadProgramExpr is a lead's effective program: chosen by hand, else its
// source's program, else Tahsin Reguler. Use with the leads alias l.
const LeadProgramExpr = `COALESCE(l.program_id, (SELECT s.program_id FROM lead_sources s WHERE s.id = l.lead_source_id), 1)`

var ErrLeadProgramNotFound = errors.New("program tidak ditemukan")

const leadProgramSelect = `SELECT p.id, p.nama, p.jenis, COALESCE(p.produk_id, 0), COALESCE(pr.nama, ''), COALESCE(pr.harga, 0),
	COALESCE(p.produk_batch_id, 0), COALESCE(b.nama, ''), p.keterangan, p.is_aktif, p.kapasitas_kelas
	FROM lead_programs p
	LEFT JOIN produk pr ON pr.id = p.produk_id
	LEFT JOIN produk_batch b ON b.id = p.produk_batch_id`

func scanLeadProgram(row interface{ Scan(...any) error }) (models.LeadProgram, error) {
	var p models.LeadProgram
	err := row.Scan(&p.ID, &p.Nama, &p.Jenis, &p.ProdukID, &p.ProdukNama, &p.ProdukHarga, &p.ProdukBatchID, &p.BatchNama, &p.Keterangan, &p.IsAktif, &p.KapasitasKelas)
	return p, err
}

func (q *Queries) ListLeadPrograms(ctx context.Context) ([]models.LeadProgram, error) {
	rows, err := q.db.QueryContext(ctx, leadProgramSelect+` ORDER BY p.id = 1 DESC, p.is_aktif DESC, p.nama`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.LeadProgram, 0)
	for rows.Next() {
		p, err := scanLeadProgram(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (q *Queries) GetLeadProgram(ctx context.Context, id int64) (*models.LeadProgram, error) {
	p, err := scanLeadProgram(q.db.QueryRowContext(ctx, leadProgramSelect+` WHERE p.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrLeadProgramNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func nullID(id int64) any {
	if id <= 0 {
		return nil
	}
	return id
}

func (q *Queries) CreateLeadProgram(ctx context.Context, r models.LeadProgramRequest, now time.Time) (int64, error) {
	var id int64
	err := q.db.QueryRowContext(ctx, `INSERT INTO lead_programs (nama, jenis, produk_id, produk_batch_id, keterangan, is_aktif, kapasitas_kelas, created_at, updated_at)
	 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`, r.Nama, r.Jenis, nullID(r.ProdukID), nullID(r.ProdukBatchID), r.Keterangan, r.IsAktif, r.KapasitasKelas, now.UTC(), now.UTC()).Scan(&id)
	return id, err
}

func (q *Queries) UpdateLeadProgram(ctx context.Context, id int64, r models.LeadProgramRequest, now time.Time) error {
	_, err := q.db.ExecContext(ctx, `UPDATE lead_programs SET nama = ?, jenis = ?, produk_id = ?, produk_batch_id = ?, keterangan = ?, is_aktif = ?, kapasitas_kelas = ?, updated_at = ? WHERE id = ?`,
		r.Nama, r.Jenis, nullID(r.ProdukID), nullID(r.ProdukBatchID), r.Keterangan, r.IsAktif, r.KapasitasKelas, now.UTC(), id)
	return err
}

// ListLeadSourcePrograms maps lead source id → program id (0 = reguler).
func (q *Queries) ListLeadSourcePrograms(ctx context.Context) (map[int64]int64, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, COALESCE(program_id, 0) FROM lead_sources`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int64{}
	for rows.Next() {
		var id, pid int64
		if err := rows.Scan(&id, &pid); err != nil {
			return nil, err
		}
		out[id] = pid
	}
	return out, rows.Err()
}

func (q *Queries) ClearLeadSourcesProgram(ctx context.Context, programID int64) error {
	_, err := q.db.ExecContext(ctx, `UPDATE lead_sources SET program_id = NULL WHERE program_id = ?`, programID)
	return err
}

func (q *Queries) SetLeadSourceProgram(ctx context.Context, sourceID, programID int64) (int64, error) {
	res, err := q.db.ExecContext(ctx, `UPDATE lead_sources SET program_id = ? WHERE id = ?`, nullID(programID), sourceID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// LeadProgramCounts counts leads per effective program and status.
func (q *Queries) LeadProgramCounts(ctx context.Context) (map[int64]map[string]int64, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+LeadProgramExpr+` AS pid, l.status, COUNT(*) FROM leads l GROUP BY pid, l.status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]map[string]int64{}
	for rows.Next() {
		var pid, n int64
		var status string
		if err := rows.Scan(&pid, &status, &n); err != nil {
			return nil, err
		}
		if out[pid] == nil {
			out[pid] = map[string]int64{}
		}
		out[pid][status] = n
	}
	return out, rows.Err()
}

func (q *Queries) LeadEffectiveProgramID(ctx context.Context, leadID int64) (int64, error) {
	var id int64
	err := q.db.QueryRowContext(ctx, `SELECT `+LeadProgramExpr+` FROM leads l WHERE l.id = ?`, leadID).Scan(&id)
	return id, err
}

// SetLeadProgram sets the program chosen by hand (0 = follow the source).
func (q *Queries) SetLeadProgram(ctx context.Context, leadID, programID int64, now time.Time) error {
	_, err := q.db.ExecContext(ctx, `UPDATE leads SET program_id = ?, updated_at = ? WHERE id = ?`, nullID(programID), now.UTC(), leadID)
	return err
}

// MarkLeadWonProgram closes a verified lead into a short program (no santri).
func (q *Queries) MarkLeadWonProgram(ctx context.Context, leadID, actorID int64, now time.Time) (int64, error) {
	res, err := q.db.ExecContext(ctx, `UPDATE leads SET status = 'won', converted_at = ?, converted_by = ?, updated_at = ?
	 WHERE id = ? AND santri_id IS NULL AND status = 'open' AND stage = 'terverifikasi'`, now.UTC(), actorID, now.UTC(), leadID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ReopenLeadForReguler reopens a participant's lead in Tahsin Reguler so CS
// can offer the regular class; it may later become a santri.
func (q *Queries) ReopenLeadForReguler(ctx context.Context, leadID int64, now time.Time) (int64, error) {
	res, err := q.db.ExecContext(ctx, `UPDATE leads SET status = 'open', stage = 'rekomendasi', program_id = 1, converted_at = NULL, converted_by = NULL, updated_at = ?
	 WHERE id = ? AND status = 'won' AND santri_id IS NULL`, now.UTC(), leadID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (q *Queries) SetProductOrderLead(ctx context.Context, orderID, leadID int64) error {
	_, err := q.db.ExecContext(ctx, `UPDATE product_orders SET lead_id = ? WHERE id = ?`, leadID, orderID)
	return err
}

func (q *Queries) GetLeadProgramOrder(ctx context.Context, leadID int64) (*models.LeadProgramOrder, error) {
	var o models.LeadProgramOrder
	err := q.db.QueryRowContext(ctx, `SELECT id, COALESCE(order_no, ''), status FROM product_orders WHERE lead_id = ?`, leadID).Scan(&o.OrderID, &o.OrderNo, &o.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// ListProgramPeserta returns active orders for the program's product (and
// batch when set), oldest first.
func (q *Queries) ListProgramPeserta(ctx context.Context, produkID, batchID int64) ([]models.ProgramPeserta, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT o.id, COALESCE(o.order_no, ''), o.nama_pembeli, o.no_wa, CAST(o.tanggal AS TEXT), COALESCE(o.lead_id, 0), COALESCE(l.status, ''), COALESCE(l.jenis_kelamin, '')
	 FROM product_orders o LEFT JOIN leads l ON l.id = o.lead_id
	 WHERE o.status = 'aktif' AND EXISTS (SELECT 1 FROM product_order_items i WHERE i.order_id = o.id AND i.produk_id = ? AND (? = 0 OR i.produk_batch_id = ?))
	 ORDER BY o.id`, produkID, batchID, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.ProgramPeserta, 0)
	for rows.Next() {
		var p models.ProgramPeserta
		if err := rows.Scan(&p.OrderID, &p.OrderNo, &p.Nama, &p.NoWA, &p.Tanggal, &p.LeadID, &p.LeadStatus, &p.LeadJenisKelamin); err != nil {
			return nil, err
		}
		if len(p.Tanggal) > 10 {
			p.Tanggal = p.Tanggal[:10]
		}
		p.Hadir = []int64{}
		out = append(out, p)
	}
	return out, rows.Err()
}

const programSesiSelect = `SELECT s.id, s.program_id, s.ke, s.judul, s.tanggal, s.jam_mulai, s.token,
	(SELECT COUNT(*) FROM program_kehadiran k WHERE k.sesi_id = s.id) FROM program_sesi s`

func scanProgramSesi(row interface{ Scan(...any) error }) (models.ProgramSesi, error) {
	var s models.ProgramSesi
	err := row.Scan(&s.ID, &s.ProgramID, &s.Ke, &s.Judul, &s.Tanggal, &s.JamMulai, &s.Token, &s.Hadir)
	return s, err
}

func (q *Queries) ListProgramSesi(ctx context.Context, programID int64) ([]models.ProgramSesi, error) {
	rows, err := q.db.QueryContext(ctx, programSesiSelect+` WHERE s.program_id = ? ORDER BY s.ke`, programID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.ProgramSesi, 0)
	for rows.Next() {
		s, err := scanProgramSesi(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (q *Queries) GetProgramSesi(ctx context.Context, id int64) (*models.ProgramSesi, error) {
	s, err := scanProgramSesi(q.db.QueryRowContext(ctx, programSesiSelect+` WHERE s.id = ?`, id))
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (q *Queries) GetProgramSesiByToken(ctx context.Context, token string) (*models.ProgramSesi, error) {
	s, err := scanProgramSesi(q.db.QueryRowContext(ctx, programSesiSelect+` WHERE s.token = ?`, token))
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (q *Queries) CreateProgramSesi(ctx context.Context, programID int64, r models.ProgramSesiRequest, token string) (int64, error) {
	var id int64
	err := q.db.QueryRowContext(ctx, `INSERT INTO program_sesi (program_id, ke, judul, tanggal, jam_mulai, token) VALUES (?, ?, ?, ?, ?, ?) RETURNING id`,
		programID, r.Ke, r.Judul, r.Tanggal, r.JamMulai, token).Scan(&id)
	return id, err
}

func (q *Queries) UpdateProgramSesi(ctx context.Context, id int64, r models.ProgramSesiRequest) error {
	_, err := q.db.ExecContext(ctx, `UPDATE program_sesi SET ke = ?, judul = ?, tanggal = ?, jam_mulai = ? WHERE id = ?`, r.Ke, r.Judul, r.Tanggal, r.JamMulai, id)
	return err
}

func (q *Queries) DeleteProgramSesi(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM program_sesi WHERE id = ?`, id)
	return err
}

func (q *Queries) ListProgramKehadiran(ctx context.Context, programID int64) ([]models.ProgramKehadiran, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT k.sesi_id, k.order_id, k.cara FROM program_kehadiran k JOIN program_sesi s ON s.id = k.sesi_id WHERE s.program_id = ?`, programID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.ProgramKehadiran, 0)
	for rows.Next() {
		var k models.ProgramKehadiran
		if err := rows.Scan(&k.SesiID, &k.OrderID, &k.Cara); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// InsertProgramKehadiran records attendance once; it reports whether a new
// row was written (false = already present).
func (q *Queries) InsertProgramKehadiran(ctx context.Context, sesiID, orderID int64, cara string, now time.Time) (bool, error) {
	res, err := q.db.ExecContext(ctx, `INSERT OR IGNORE INTO program_kehadiran (sesi_id, order_id, cara, created_at) VALUES (?, ?, ?, ?)`, sesiID, orderID, cara, now.UTC())
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (q *Queries) DeleteProgramKehadiran(ctx context.Context, sesiID, orderID int64) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM program_kehadiran WHERE sesi_id = ? AND order_id = ?`, sesiID, orderID)
	return err
}

// leadProgramFilter is the list condition for one program (LeadListFilter).
func leadProgramFilter(programID int64) (string, []any) {
	return strings.TrimSpace(LeadProgramExpr) + ` = ?`, []any{programID}
}
