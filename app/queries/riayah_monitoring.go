package queries

import "context"

type RiayahCurrentAssignment struct {
	SantriID int64
	GuruID   int64
	GuruNama string
	Mulai    string
}

type RiayahTeacherContact struct {
	GuruID         int64
	SantriID       int64
	KontakTerakhir string
}

type RiayahTeacherReport struct {
	GuruID   int64
	SantriID int64
}

type RiayahTeacherActivity struct {
	GuruID    int64
	Terakhir  string
}

func (q *Queries) ListRiayahCurrentAssignments(ctx context.Context) ([]RiayahCurrentAssignment, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT s.id,
		       COALESCE(k.guru_id, 0),
		       COALESCE(g.nama, ''),
		       COALESCE(strftime('%Y-%m-%d', s.mulai_belajar), strftime('%Y-%m-%d', s.tanggal_daftar), '')
		FROM santri s
		LEFT JOIN kelas k ON k.id = s.kelas_id
		LEFT JOIN guru g ON g.id = k.guru_id
		WHERE s.status = 'aktif'
		ORDER BY COALESCE(g.nama, ''), s.nama`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RiayahCurrentAssignment, 0)
	for rows.Next() {
		var item RiayahCurrentAssignment
		if err := rows.Scan(&item.SantriID, &item.GuruID, &item.GuruNama, &item.Mulai); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Queries) ListRiayahTeacherContacts(ctx context.Context) ([]RiayahTeacherContact, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT rk.guru_id, rk.santri_id, CAST(MAX(rk.tanggal) AS TEXT)
		FROM riayah_kontak rk
		JOIN santri s ON s.id = rk.santri_id
		JOIN kelas k ON k.id = s.kelas_id
		WHERE s.status = 'aktif'
		  AND rk.guru_id IS NOT NULL
		  AND k.guru_id = rk.guru_id
		GROUP BY rk.guru_id, rk.santri_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RiayahTeacherContact, 0)
	for rows.Next() {
		var item RiayahTeacherContact
		if err := rows.Scan(&item.GuruID, &item.SantriID, &item.KontakTerakhir); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Queries) ListRiayahTeacherReports(ctx context.Context, periode string) ([]RiayahTeacherReport, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT rk.guru_id, rk.santri_id
		FROM riayah_kontak rk
		JOIN santri s ON s.id = rk.santri_id
		JOIN kelas k ON k.id = s.kelas_id
		WHERE s.status = 'aktif'
		  AND rk.guru_id IS NOT NULL
		  AND k.guru_id = rk.guru_id
		  AND rk.jenis = 'rapor'
		  AND rk.periode = ?
		GROUP BY rk.guru_id, rk.santri_id`, periode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RiayahTeacherReport, 0)
	for rows.Next() {
		var item RiayahTeacherReport
		if err := rows.Scan(&item.GuruID, &item.SantriID); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Queries) ListRiayahTeacherActivities(ctx context.Context) ([]RiayahTeacherActivity, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT guru_id, COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', MAX(activity_at)), '')
		FROM (
			SELECT rk.guru_id AS guru_id, rk.created_at AS activity_at
			FROM riayah_kontak rk
			JOIN santri s ON s.id = rk.santri_id
			JOIN kelas k ON k.id = s.kelas_id
			WHERE s.status = 'aktif'
			  AND rk.guru_id IS NOT NULL
			  AND k.guru_id = rk.guru_id
			UNION ALL
			SELECT cr.guru_id AS guru_id, cr.created_at AS activity_at
			FROM catatan_riayah cr
			JOIN santri s ON s.id = cr.target_id AND cr.target_type = 'santri'
			JOIN kelas k ON k.id = s.kelas_id
			WHERE s.status = 'aktif'
			  AND cr.guru_id IS NOT NULL
			  AND k.guru_id = cr.guru_id
		)
		GROUP BY guru_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RiayahTeacherActivity, 0)
	for rows.Next() {
		var item RiayahTeacherActivity
		if err := rows.Scan(&item.GuruID, &item.Terakhir); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
