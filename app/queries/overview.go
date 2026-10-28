package queries

// Handwritten aggregate queries, following santri_analytics.go. No writes or
// schedule generation occur while loading the management overview.
import (
	"context"

	"github.com/maulanashalihin/laju-go/app/models"
)

func (q *Queries) OverviewOperations(ctx context.Context, today, from, until string) (*models.OverviewOperations, error) {
	var out models.OverviewOperations
	err := q.db.QueryRowContext(ctx, `SELECT
 (SELECT COUNT(*) FROM santri WHERE status = 'aktif' AND is_lengkap = 0),
	 (SELECT COUNT(*) FROM guru WHERE is_aktif = 1),
	 (SELECT COUNT(*) FROM kelas WHERE is_aktif = 1),
	 (SELECT COUNT(*) FROM kelas k LEFT JOIN guru g ON g.id = k.guru_id WHERE k.is_aktif = 1 AND (g.id IS NULL OR g.is_aktif != 1)),
	 (SELECT COUNT(*) FROM santri s LEFT JOIN kelas k ON k.id = s.kelas_id WHERE s.status = 'aktif' AND (k.id IS NULL OR k.is_aktif != 1)),
	 (SELECT COUNT(*) FROM pertemuan WHERE status = 'berlangsung' AND date(substr(tanggal, 1, 10)) < date(?)),
	 (SELECT COUNT(*) FROM kunjungan_tindak_lanjut WHERE status = 'terbuka' AND date(substr(target_tanggal, 1, 10)) <= date(?)),
	 (SELECT COUNT(*) FROM todo_koordinator WHERE status IN ('belum', 'proses') AND date(substr(deadline, 1, 10)) <= date(?)),
	 (SELECT COUNT(*) FROM pertemuan WHERE status = 'selesai' AND date(substr(tanggal, 1, 10)) BETWEEN date(?) AND date(?)),
	 (SELECT COUNT(*) FROM pertemuan WHERE status = 'selesai' AND is_badal = 1 AND date(substr(tanggal, 1, 10)) BETWEEN date(?) AND date(?)),
	 (SELECT COUNT(*) FROM pertemuan WHERE status = 'selesai' AND is_reschedule = 1 AND date(substr(tanggal, 1, 10)) BETWEEN date(?) AND date(?))`,
		today, today, today, from, until, from, until, from, until).Scan(
		&out.Incomplete, &out.Teachers, &out.Classes, &out.WithoutTeacher, &out.WithoutClass,
		&out.OpenMeetings, &out.FollowUps, &out.Todos, &out.Finished, &out.Badal, &out.Rescheduled)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (q *Queries) OverviewPrograms(ctx context.Context) ([]models.OverviewProgram, error) {
	rows, err := q.db.QueryContext(ctx, `WITH students AS (
	 SELECT COALESCE(NULLIF(TRIM(tipe), ''), 'Belum diisi') AS name,
	 SUM(CASE WHEN status = 'aktif' THEN 1 ELSE 0 END) AS active,
	 SUM(CASE WHEN status = 'cuti' THEN 1 ELSE 0 END) AS cuti,
	 SUM(CASE WHEN status IN ('nonaktif', 'tidak_lanjut') THEN 1 ELSE 0 END) AS stopped
	 FROM santri GROUP BY 1
	), classes AS (
	 SELECT COALESCE(NULLIF(TRIM(k.tipe), ''), 'Belum diisi') AS name,
	 COUNT(*) AS total, SUM(k.kapasitas) AS seats,
	 SUM((SELECT COUNT(*) FROM santri s WHERE s.kelas_id = k.id AND s.status IN ('aktif', 'cuti'))) AS occupied
	 FROM kelas k WHERE k.is_aktif = 1 GROUP BY 1
	), names AS (SELECT name FROM students UNION SELECT name FROM classes)
	 SELECT n.name, COALESCE(s.active,0), COALESCE(s.cuti,0), COALESCE(s.stopped,0),
	 COALESCE(c.total,0), COALESCE(c.seats,0), COALESCE(c.occupied,0)
	 FROM names n LEFT JOIN students s ON s.name = n.name LEFT JOIN classes c ON c.name = n.name
	 ORDER BY COALESCE(s.active,0) DESC, n.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.OverviewProgram, 0)
	for rows.Next() {
		var r models.OverviewProgram
		if err := rows.Scan(&r.Name, &r.Active, &r.Cuti, &r.Stopped, &r.Classes, &r.Seats, &r.Occupied); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// OverviewAttendance groups attendance of finished meetings between from and
// until (inclusive, YYYY-MM-DD) by meeting month. Months without meetings are
// not returned.
func (q *Queries) OverviewAttendance(ctx context.Context, from, until string) ([]models.OverviewAttendanceMonth, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT substr(p.tanggal, 1, 7) AS bulan, COUNT(DISTINCT p.id),
	 COALESCE(SUM(CASE WHEN a.status = 'hadir' THEN 1 ELSE 0 END), 0),
	 COALESCE(SUM(CASE WHEN a.status = 'izin' THEN 1 ELSE 0 END), 0),
	 COALESCE(SUM(CASE WHEN a.status = 'sakit' THEN 1 ELSE 0 END), 0),
	 COALESCE(SUM(CASE WHEN a.status = 'alpa' THEN 1 ELSE 0 END), 0)
	 FROM pertemuan p LEFT JOIN absensi a ON a.pertemuan_id = p.id
	 WHERE p.status = 'selesai' AND date(substr(p.tanggal, 1, 10)) BETWEEN date(?) AND date(?)
	 GROUP BY bulan ORDER BY bulan`, from, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.OverviewAttendanceMonth, 0)
	for rows.Next() {
		var r models.OverviewAttendanceMonth
		if err := rows.Scan(&r.Month, &r.Meetings, &r.Hadir, &r.Izin, &r.Sakit, &r.Alpa); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
