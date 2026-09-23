package queries

import (
	"context"
	"strings"

	"github.com/maulanashalihin/laju-go/app/models"
)

func santriAnalyticsWhere(filters models.SantriAnalyticsFilters, includeSegment bool) (string, []interface{}) {
	clauses := []string{"1 = 1"}
	args := make([]interface{}, 0, 12)
	add := func(clause string, value interface{}) {
		clauses = append(clauses, clause)
		args = append(args, value)
	}

	if filters.DateFrom != "" {
		add("date(tanggal_daftar) >= date(?)", filters.DateFrom)
	}
	if filters.DateTo != "" {
		add("date(tanggal_daftar) <= date(?)", filters.DateTo)
	}
	if filters.AngkatanPendaftaran != "" {
		add("angkatan = ?", filters.AngkatanPendaftaran)
	}
	if filters.AngkatanKelas != "" {
		add("angkatan_kelas = ?", filters.AngkatanKelas)
	}
	if filters.Level != "" {
		add("level = ?", filters.Level)
	}
	if filters.Tipe != "" {
		add("tipe = ?", filters.Tipe)
	}

	if includeSegment {
		switch filters.SegmentAgeBucket {
		case "le17":
			clauses = append(clauses, "usia IS NOT NULL AND usia > 0 AND usia <= 17")
		case "18-24":
			clauses = append(clauses, "usia BETWEEN 18 AND 24")
		case "25-34":
			clauses = append(clauses, "usia BETWEEN 25 AND 34")
		case "35-44":
			clauses = append(clauses, "usia BETWEEN 35 AND 44")
		case "45-54":
			clauses = append(clauses, "usia BETWEEN 45 AND 54")
		case "55plus":
			clauses = append(clauses, "usia >= 55")
		}
		if filters.SegmentGender != "" {
			if filters.SegmentGender == "unknown" {
				clauses = append(clauses, "jenis_kelamin NOT IN ('L', 'P')")
			} else {
				add("jenis_kelamin = ?", filters.SegmentGender)
			}
		}
		if filters.SegmentDomisili != "" {
			add("domisili = ?", filters.SegmentDomisili)
		}
		if filters.SegmentStatus != "" {
			add("status = ?", filters.SegmentStatus)
		}
		if filters.SegmentRegistrationMo != "" {
			add("strftime('%Y-%m', tanggal_daftar) = ?", filters.SegmentRegistrationMo)
		}
	}

	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (q *Queries) SantriAnalyticsSummary(ctx context.Context, filters models.SantriAnalyticsFilters) (models.SantriAnalyticsSummary, error) {
	where, args := santriAnalyticsWhere(filters, false)
	query := `SELECT
		COUNT(*) AS total,
		COALESCE(SUM(CASE WHEN status = 'aktif' THEN 1 ELSE 0 END), 0) AS aktif,
		COALESCE(SUM(CASE WHEN status = 'cuti' THEN 1 ELSE 0 END), 0) AS cuti,
		COALESCE(SUM(CASE WHEN status = 'nonaktif' THEN 1 ELSE 0 END), 0) AS nonaktif,
		COALESCE(SUM(CASE WHEN status = 'tidak_lanjut' THEN 1 ELSE 0 END), 0) AS tidak_lanjut,
		COALESCE(AVG(CASE WHEN usia IS NOT NULL AND usia > 0 THEN usia END), 0) AS rata_rata_usia
	FROM santri` + where

	var row models.SantriAnalyticsSummary
	err := q.db.QueryRowContext(ctx, query, args...).Scan(
		&row.Total, &row.Aktif, &row.Cuti, &row.Nonaktif, &row.TidakLanjut, &row.RataRataUsia,
	)
	return row, err
}

func (q *Queries) SantriAnalyticsGrowth(ctx context.Context, filters models.SantriAnalyticsFilters) ([]models.SantriAnalyticsGrowthPoint, error) {
	where, args := santriAnalyticsWhere(filters, false)
	query := `SELECT strftime('%Y-%m', tanggal_daftar) AS month, COUNT(*) AS total
	FROM santri` + where + ` AND tanggal_daftar IS NOT NULL
	GROUP BY strftime('%Y-%m', tanggal_daftar)
	HAVING month IS NOT NULL AND month != ''
	ORDER BY month`
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.SantriAnalyticsGrowthPoint, 0)
	for rows.Next() {
		var point models.SantriAnalyticsGrowthPoint
		if err := rows.Scan(&point.Month, &point.Total); err != nil {
			return nil, err
		}
		out = append(out, point)
	}
	return out, rows.Err()
}

func (q *Queries) SantriAnalyticsAge(ctx context.Context, filters models.SantriAnalyticsFilters) ([]models.SantriAnalyticsPoint, error) {
	where, args := santriAnalyticsWhere(filters, false)
	query := `SELECT
		CASE
			WHEN usia <= 17 THEN 'le17'
			WHEN usia BETWEEN 18 AND 24 THEN '18-24'
			WHEN usia BETWEEN 25 AND 34 THEN '25-34'
			WHEN usia BETWEEN 35 AND 44 THEN '35-44'
			WHEN usia BETWEEN 45 AND 54 THEN '45-54'
			ELSE '55plus'
		END AS bucket,
		COUNT(*) AS total
	FROM santri` + where + ` AND usia IS NOT NULL AND usia > 0
	GROUP BY bucket
	ORDER BY CASE bucket
		WHEN 'le17' THEN 1 WHEN '18-24' THEN 2 WHEN '25-34' THEN 3
		WHEN '35-44' THEN 4 WHEN '45-54' THEN 5 ELSE 6 END`
	return q.santriAnalyticsPoints(ctx, query, args...)
}

func (q *Queries) SantriAnalyticsGender(ctx context.Context, filters models.SantriAnalyticsFilters) ([]models.SantriAnalyticsPoint, error) {
	where, args := santriAnalyticsWhere(filters, false)
	query := `SELECT CASE WHEN jenis_kelamin = 'L' THEN 'L' WHEN jenis_kelamin = 'P' THEN 'P' ELSE 'unknown' END AS gender,
		COUNT(*) AS total
	FROM santri` + where + `
	GROUP BY gender
	ORDER BY CASE gender WHEN 'P' THEN 1 WHEN 'L' THEN 2 ELSE 3 END`
	return q.santriAnalyticsPoints(ctx, query, args...)
}

func (q *Queries) SantriAnalyticsDomisili(ctx context.Context, filters models.SantriAnalyticsFilters, limit int64) ([]models.SantriAnalyticsPoint, error) {
	where, args := santriAnalyticsWhere(filters, false)
	query := `SELECT domisili, COUNT(*) AS total
	FROM santri` + where + ` AND TRIM(domisili) != ''
	GROUP BY domisili
	ORDER BY total DESC, domisili ASC
	LIMIT ?`
	args = append(args, limit)
	return q.santriAnalyticsPoints(ctx, query, args...)
}

func (q *Queries) SantriAnalyticsStatus(ctx context.Context, filters models.SantriAnalyticsFilters) ([]models.SantriAnalyticsPoint, error) {
	where, args := santriAnalyticsWhere(filters, false)
	query := `SELECT CASE WHEN TRIM(status) = '' THEN 'unknown' ELSE status END AS status_value, COUNT(*) AS total
	FROM santri` + where + `
	GROUP BY status_value
	ORDER BY total DESC, status_value ASC`
	return q.santriAnalyticsPoints(ctx, query, args...)
}

func (q *Queries) SantriAnalyticsLevel(ctx context.Context, filters models.SantriAnalyticsFilters) ([]models.SantriAnalyticsPoint, error) {
	where, args := santriAnalyticsWhere(filters, false)
	query := `SELECT level, COUNT(*) AS total
	FROM santri` + where + ` AND TRIM(level) != ''
	GROUP BY level
	ORDER BY total DESC, level ASC`
	return q.santriAnalyticsPoints(ctx, query, args...)
}

func (q *Queries) SantriAnalyticsTipe(ctx context.Context, filters models.SantriAnalyticsFilters) ([]models.SantriAnalyticsPoint, error) {
	where, args := santriAnalyticsWhere(filters, false)
	query := `SELECT tipe, COUNT(*) AS total
	FROM santri` + where + ` AND TRIM(tipe) != ''
	GROUP BY tipe
	ORDER BY total DESC, tipe ASC`
	return q.santriAnalyticsPoints(ctx, query, args...)
}

func (q *Queries) santriAnalyticsPoints(ctx context.Context, query string, args ...interface{}) ([]models.SantriAnalyticsPoint, error) {
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.SantriAnalyticsPoint, 0)
	for rows.Next() {
		var point models.SantriAnalyticsPoint
		if err := rows.Scan(&point.Value, &point.Total); err != nil {
			return nil, err
		}
		point.Label = point.Value
		out = append(out, point)
	}
	return out, rows.Err()
}

func (q *Queries) SantriAnalyticsStatusByAngkatan(ctx context.Context, filters models.SantriAnalyticsFilters) ([]models.SantriAnalyticsCohortStatus, error) {
	where, args := santriAnalyticsWhere(filters, false)
	query := `SELECT
		CASE WHEN TRIM(angkatan) = '' THEN 'Tanpa Angkatan' ELSE angkatan END AS cohort,
		COALESCE(SUM(CASE WHEN status = 'aktif' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = 'cuti' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = 'nonaktif' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = 'tidak_lanjut' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status NOT IN ('aktif', 'cuti', 'nonaktif', 'tidak_lanjut') THEN 1 ELSE 0 END), 0),
		COUNT(*)
	FROM santri` + where + `
	GROUP BY cohort
	ORDER BY CASE WHEN cohort = 'Tanpa Angkatan' THEN 1 ELSE 0 END, cohort ASC`
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.SantriAnalyticsCohortStatus, 0)
	for rows.Next() {
		var row models.SantriAnalyticsCohortStatus
		if err := rows.Scan(&row.Angkatan, &row.Aktif, &row.Cuti, &row.Nonaktif, &row.TidakLanjut, &row.Lainnya, &row.Total); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (q *Queries) SantriAnalyticsDataQuality(ctx context.Context, filters models.SantriAnalyticsFilters) (models.SantriAnalyticsDataQuality, error) {
	where, args := santriAnalyticsWhere(filters, false)
	query := `SELECT
		COUNT(*),
		COALESCE(SUM(CASE WHEN usia IS NULL OR usia <= 0 THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN TRIM(domisili) = '' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN tanggal_daftar IS NULL THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN jenis_kelamin NOT IN ('L', 'P') THEN 1 ELSE 0 END), 0)
	FROM santri` + where

	var row models.SantriAnalyticsDataQuality
	err := q.db.QueryRowContext(ctx, query, args...).Scan(
		&row.Total, &row.UsiaKosong, &row.DomisiliKosong, &row.TanggalDaftarKosong, &row.JenisKelaminKosong,
	)
	return row, err
}

func (q *Queries) SantriAnalyticsDetailCount(ctx context.Context, filters models.SantriAnalyticsFilters) (int64, error) {
	where, args := santriAnalyticsWhere(filters, true)
	var total int64
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM santri`+where, args...).Scan(&total)
	return total, err
}

func (q *Queries) SantriAnalyticsDetail(ctx context.Context, filters models.SantriAnalyticsFilters) ([]models.SantriAnalyticsRow, error) {
	where, args := santriAnalyticsWhere(filters, true)
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	page := filters.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	query := `SELECT
		id, id_mahasantri, nama, COALESCE(usia, 0), jenis_kelamin, domisili,
		COALESCE(strftime('%Y-%m-%d', tanggal_daftar), ''),
		angkatan, angkatan_kelas, level, tipe, status
	FROM santri` + where + `
	ORDER BY CASE WHEN tanggal_daftar IS NULL THEN 1 ELSE 0 END, tanggal_daftar DESC, id DESC
	LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.SantriAnalyticsRow, 0, limit)
	for rows.Next() {
		var row models.SantriAnalyticsRow
		if err := rows.Scan(
			&row.ID, &row.IDMahasantri, &row.Nama, &row.Usia, &row.JenisKelamin, &row.Domisili,
			&row.TanggalDaftar, &row.Angkatan, &row.AngkatanKelas, &row.Level, &row.Tipe, &row.Status,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
