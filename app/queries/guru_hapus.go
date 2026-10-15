package queries

import (
	"context"
)

// GuruRiwayat counts every record that ties a guru to past activity. A guru
// with any of these can only be deactivated, because deleting the guru row
// cascades into (or orphans) this history.
type GuruRiwayat struct {
	KelasAktif      int64
	KelasNonaktif   int64
	PertemuanBadal  int64
	JadwalBadal     int64
	Ujroh           int64
	Riayah          int64
	AbsensiKegiatan int64
	Tilawah         int64
	Kunjungan       int64
	Tsi             int64
	KalamShare      int64
}

const getGuruRiwayat = `
SELECT
	(SELECT COUNT(*) FROM kelas WHERE guru_id = ?1 AND is_aktif = 1),
	(SELECT COUNT(*) FROM kelas WHERE guru_id = ?1 AND is_aktif = 0),
	(SELECT COUNT(*) FROM pertemuan WHERE guru_pengganti_id = ?1),
	(SELECT COUNT(*) FROM jadwal_pertemuan WHERE guru_pengganti_id = ?1),
	(SELECT COUNT(*) FROM ujroh_guru WHERE guru_id = ?1)
		+ (SELECT COUNT(*) FROM ujroh_pertemuan WHERE guru_id = ?1),
	(SELECT COUNT(*) FROM catatan_riayah WHERE guru_id = ?1)
		+ (SELECT COUNT(*) FROM catatan_riayah WHERE target_type = 'guru' AND target_id = ?1)
		+ (SELECT COUNT(*) FROM riayah_kontak WHERE guru_id = ?1),
	(SELECT COUNT(*) FROM pembinaan_absen WHERE guru_id = ?1)
		+ (SELECT COUNT(*) FROM rapat_absen WHERE guru_id = ?1),
	(SELECT COUNT(*) FROM tilawah_harian WHERE guru_id = ?1),
	(SELECT COUNT(*) FROM kunjungan_kelas WHERE guru_id = ?1),
	(SELECT COUNT(*) FROM tsi_periode WHERE guru_id = ?1),
	(SELECT COUNT(*) FROM kalam_share_log WHERE guru_id = ?1)
`

func (q *Querier) GetGuruRiwayat(ctx context.Context, guruID int64) (GuruRiwayat, error) {
	var r GuruRiwayat
	err := q.Queries.db.QueryRowContext(ctx, getGuruRiwayat, guruID).Scan(
		&r.KelasAktif,
		&r.KelasNonaktif,
		&r.PertemuanBadal,
		&r.JadwalBadal,
		&r.Ujroh,
		&r.Riayah,
		&r.AbsensiKegiatan,
		&r.Tilawah,
		&r.Kunjungan,
		&r.Tsi,
		&r.KalamShare,
	)
	return r, err
}

func (q *Querier) SetGuruNonaktif(ctx context.Context, guruID int64) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, `UPDATE guru SET is_aktif = 0 WHERE id = ?`, guruID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (q *Querier) DeleteGuruByID(ctx context.Context, guruID int64) (int64, error) {
	result, err := q.Queries.db.ExecContext(ctx, `DELETE FROM guru WHERE id = ?`, guruID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
