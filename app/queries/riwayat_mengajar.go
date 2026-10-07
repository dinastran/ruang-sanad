package queries

import (
	"context"
	"database/sql"
)

const listRiwayatMengajar = `
SELECT
	p.id,
	p.kelas_id,
	k.nama_kelas,
	COALESCE(g.nama, '') AS guru_nama,
	p.pertemuan_ke,
	p.pertemuan_level_ke,
	p.level_nama,
	p.tanggal,
	p.jam_mulai,
	p.jam_selesai,
	p.materi,
	p.catatan,
	p.is_reschedule,
	p.jadwal_semula,
	p.alasan_reschedule,
	p.is_badal,
	p.guru_pengganti_id,
	p.alasan_badal,
	p.status
FROM pertemuan p
JOIN kelas k ON k.id = p.kelas_id
LEFT JOIN guru g ON g.id = CASE
	WHEN p.is_badal = 1 THEN p.guru_pengganti_id
	ELSE k.guru_id
END
WHERE date(p.tanggal) >= date(?) AND date(p.tanggal) <= date(?)
  AND (
	? IS NULL
	OR (p.is_badal = 1 AND p.guru_pengganti_id = ?)
	OR (p.is_badal = 0 AND k.guru_id = ?)
  )
ORDER BY p.tanggal DESC, p.id DESC
`

type ListRiwayatMengajarParams struct {
	GuruID         sql.NullInt64
	TanggalMulai   string
	TanggalSelesai string
}

type ListRiwayatMengajarRow struct {
	ID               int64
	KelasID          int64
	KelasNama        string
	GuruNama         string
	PertemuanKe      int64
	PertemuanLevelKe int64
	LevelNama        string
	Tanggal          time.Time
	JamMulai         string
	JamSelesai       string
	Materi           string
	Catatan          string
	IsReschedule     int64
	JadwalSemula     string
	AlasanReschedule string
	IsBadal          int64
	GuruPenggantiID  sql.NullInt64
	AlasanBadal      string
	Status           string
}

func (q *Queries) ListRiwayatMengajar(ctx context.Context, arg ListRiwayatMengajarParams) ([]ListRiwayatMengajarRow, error) {
	rows, err := q.db.QueryContext(
		ctx,
		listRiwayatMengajar,
		arg.TanggalMulai,
		arg.TanggalSelesai,
		arg.GuruID,
		arg.GuruID,
		arg.GuruID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ListRiwayatMengajarRow, 0)
	for rows.Next() {
		var item ListRiwayatMengajarRow
		if err := rows.Scan(
			&item.ID,
			&item.KelasID,
			&item.KelasNama,
			&item.GuruNama,
			&item.PertemuanKe,
			&item.PertemuanLevelKe,
			&item.LevelNama,
			&item.Tanggal,
			&item.JamMulai,
			&item.JamSelesai,
			&item.Materi,
			&item.Catatan,
			&item.IsReschedule,
			&item.JadwalSemula,
			&item.AlasanReschedule,
			&item.IsBadal,
			&item.GuruPenggantiID,
			&item.AlasanBadal,
			&item.Status,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
