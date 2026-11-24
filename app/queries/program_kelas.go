package queries

// Handwritten queries for the ikhwan/akhwat classes of short CRM programs.
import (
	"context"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
)

func (q *Queries) ListProgramKelas(ctx context.Context, programID int64) ([]models.ProgramKelas, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, program_id, jenis_kelamin, urutan, pengajar, zoom_url
	 FROM program_kelas WHERE program_id = ? ORDER BY jenis_kelamin, urutan`, programID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.ProgramKelas, 0)
	for rows.Next() {
		var k models.ProgramKelas
		if err := rows.Scan(&k.ID, &k.ProgramID, &k.JenisKelamin, &k.Urutan, &k.Pengajar, &k.ZoomURL); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (q *Queries) GetProgramKelas(ctx context.Context, id int64) (models.ProgramKelas, error) {
	var k models.ProgramKelas
	err := q.db.QueryRowContext(ctx, `SELECT id, program_id, jenis_kelamin, urutan, pengajar, zoom_url FROM program_kelas WHERE id = ?`, id).
		Scan(&k.ID, &k.ProgramID, &k.JenisKelamin, &k.Urutan, &k.Pengajar, &k.ZoomURL)
	return k, err
}

func (q *Queries) CreateProgramKelas(ctx context.Context, programID int64, jk string, urutan int64, now time.Time) (int64, error) {
	var id int64
	err := q.db.QueryRowContext(ctx, `INSERT INTO program_kelas (program_id, jenis_kelamin, urutan, created_at) VALUES (?, ?, ?, ?) RETURNING id`,
		programID, jk, urutan, now.UTC()).Scan(&id)
	return id, err
}

func (q *Queries) UpdateProgramKelas(ctx context.Context, id int64, pengajar, zoomURL string) error {
	_, err := q.db.ExecContext(ctx, `UPDATE program_kelas SET pengajar = ?, zoom_url = ? WHERE id = ?`, pengajar, zoomURL, id)
	return err
}

func (q *Queries) DeleteProgramKelas(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM program_kelas WHERE id = ?`, id)
	return err
}

// ListProgramPesertaKelas maps order id → stored class assignment.
func (q *Queries) ListProgramPesertaKelas(ctx context.Context, programID int64) (map[int64]models.ProgramPesertaKelas, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT order_id, jenis_kelamin, COALESCE(kelas_id, 0), cara FROM program_peserta_kelas WHERE program_id = ?`, programID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]models.ProgramPesertaKelas{}
	for rows.Next() {
		var r models.ProgramPesertaKelas
		if err := rows.Scan(&r.OrderID, &r.JenisKelamin, &r.KelasID, &r.Cara); err != nil {
			return nil, err
		}
		out[r.OrderID] = r
	}
	return out, rows.Err()
}

// UpsertProgramPesertaKelas stores a participant's gender and class
// (kelasID 0 = no class yet).
func (q *Queries) UpsertProgramPesertaKelas(ctx context.Context, programID, orderID int64, jk string, kelasID int64, cara string, now time.Time) error {
	_, err := q.db.ExecContext(ctx, `INSERT INTO program_peserta_kelas (program_id, order_id, jenis_kelamin, kelas_id, cara, updated_at)
	 VALUES (?, ?, ?, ?, ?, ?)
	 ON CONFLICT(program_id, order_id) DO UPDATE SET jenis_kelamin = excluded.jenis_kelamin, kelas_id = excluded.kelas_id,
	   cara = excluded.cara, updated_at = excluded.updated_at`,
		programID, orderID, jk, nullID(kelasID), cara, now.UTC())
	return err
}
