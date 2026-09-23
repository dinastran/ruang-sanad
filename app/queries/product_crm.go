package queries

import (
	"context"
	"database/sql"
	"strings"

	"github.com/maulanashalihin/laju-go/app/models"
)

type productScanner interface {
	Scan(dest ...interface{}) error
}

func scanProduct(s productScanner) (models.Product, error) {
	var out models.Product
	var track, aktif int64
	err := s.Scan(&out.ID, &out.Nama, &out.Kategori, &track, &aktif, &out.Stok, &out.MinimumStock)
	out.TrackStok = track == 1
	out.IsAktif = aktif == 1
	return out, err
}

func (q *Queries) ListProduk(ctx context.Context) ([]models.Product, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT p.id, p.nama, p.kategori, p.track_stok, p.is_aktif, COALESCE(SUM(sm.qty), 0), p.minimum_stock
		FROM produk p
		LEFT JOIN stok_mutasi sm ON sm.produk_id = p.id
		GROUP BY p.id, p.nama, p.kategori, p.track_stok, p.is_aktif, p.minimum_stock
		ORDER BY p.is_aktif DESC, p.nama ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Product, 0)
	for rows.Next() {
		item, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Queries) GetProduk(ctx context.Context, id int64) (models.Product, error) {
	return scanProduct(q.db.QueryRowContext(ctx, `
		SELECT p.id, p.nama, p.kategori, p.track_stok, p.is_aktif, COALESCE(SUM(sm.qty), 0), p.minimum_stock
		FROM produk p
		LEFT JOIN stok_mutasi sm ON sm.produk_id = p.id
		WHERE p.id = ?
		GROUP BY p.id, p.nama, p.kategori, p.track_stok, p.is_aktif, p.minimum_stock`, id))
}

func (q *Queries) CreateProduk(ctx context.Context, nama, kategori string, trackStok bool, minimumStock int64) (int64, error) {
	res, err := q.db.ExecContext(ctx, `
		INSERT INTO produk (nama, kategori, track_stok, is_aktif, minimum_stock)
		VALUES (?, ?, ?, 1, ?)`, nama, kategori, trackStok, minimumStock)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (q *Queries) UpdateProduk(ctx context.Context, id int64, nama, kategori string, trackStok, isAktif bool, minimumStock int64) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE produk
		SET nama = ?, kategori = ?, track_stok = ?, is_aktif = ?, minimum_stock = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, nama, kategori, trackStok, isAktif, minimumStock, id)
	return err
}

func (q *Queries) ProdukHasHistory(ctx context.Context, productID int64) (bool, error) {
	var total int64
	err := q.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM mahasantri_produk WHERE produk_id = ?) +
			(SELECT COUNT(*) FROM stok_mutasi WHERE produk_id = ?)`,
		productID, productID,
	).Scan(&total)
	return total > 0, err
}

func (q *Queries) ListProdukBatch(ctx context.Context, productID int64) ([]models.ProductBatch, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT id, produk_id, nama,
		       COALESCE(strftime('%Y-%m-%d', tanggal_mulai), ''),
		       COALESCE(strftime('%Y-%m-%d', tanggal_selesai), ''),
		       is_aktif
		FROM produk_batch
		WHERE produk_id = ?
		ORDER BY is_aktif DESC, id DESC`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.ProductBatch, 0)
	for rows.Next() {
		var item models.ProductBatch
		var aktif int64
		if err := rows.Scan(&item.ID, &item.ProdukID, &item.Nama, &item.TanggalMulai, &item.TanggalSelesai, &aktif); err != nil {
			return nil, err
		}
		item.IsAktif = aktif == 1
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Queries) GetProdukBatch(ctx context.Context, id int64) (models.ProductBatch, error) {
	var item models.ProductBatch
	var aktif int64
	err := q.db.QueryRowContext(ctx, `
		SELECT id, produk_id, nama,
		       COALESCE(strftime('%Y-%m-%d', tanggal_mulai), ''),
		       COALESCE(strftime('%Y-%m-%d', tanggal_selesai), ''),
		       is_aktif
		FROM produk_batch WHERE id = ?`, id).Scan(
		&item.ID, &item.ProdukID, &item.Nama, &item.TanggalMulai, &item.TanggalSelesai, &aktif,
	)
	item.IsAktif = aktif == 1
	return item, err
}

func (q *Queries) CountActiveProdukBatches(ctx context.Context, productID int64) (int64, error) {
	var total int64
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM produk_batch WHERE produk_id = ? AND is_aktif = 1`, productID).Scan(&total)
	return total, err
}

func (q *Queries) CreateProdukBatch(ctx context.Context, productID int64, nama string, mulai, selesai sql.NullString) (int64, error) {
	res, err := q.db.ExecContext(ctx, `
		INSERT INTO produk_batch (produk_id, nama, tanggal_mulai, tanggal_selesai, is_aktif)
		VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), 1)`,
		productID, nama, mulai.String, selesai.String,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (q *Queries) UpdateProdukBatch(ctx context.Context, id int64, nama string, mulai, selesai sql.NullString, isAktif bool) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE produk_batch
		SET nama = ?, tanggal_mulai = NULLIF(?, ''), tanggal_selesai = NULLIF(?, ''),
		    is_aktif = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		nama, mulai.String, selesai.String, isAktif, id,
	)
	return err
}

func (q *Queries) CurrentProdukStock(ctx context.Context, productID int64, batchID sql.NullInt64) (int64, error) {
	var total int64
	var err error
	if batchID.Valid {
		err = q.db.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(qty), 0) FROM stok_mutasi
			WHERE produk_id = ? AND produk_batch_id = ?`, productID, batchID.Int64).Scan(&total)
	} else {
		err = q.db.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(qty), 0) FROM stok_mutasi
			WHERE produk_id = ? AND produk_batch_id IS NULL`, productID).Scan(&total)
	}
	return total, err
}

func (q *Queries) InsertStockMutation(ctx context.Context, productID int64, batchID sql.NullInt64, tipe string, qty int64, refType string, refID sql.NullInt64, catatan string, userID int64) (int64, error) {
	res, err := q.db.ExecContext(ctx, `
		INSERT INTO stok_mutasi
			(produk_id, produk_batch_id, tipe, qty, referensi_type, referensi_id, catatan, dicatat_oleh)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		productID, nullableInt64(batchID), tipe, qty, refType, nullableInt64(refID), catatan, userID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (q *Queries) InsertStockOpname(ctx context.Context, productID int64, batchID sql.NullInt64, sistem, fisik, selisih int64, catatan string, userID int64) (int64, error) {
	res, err := q.db.ExecContext(ctx, `
		INSERT INTO stok_opname
			(produk_id, produk_batch_id, stok_sistem, stok_fisik, selisih, catatan, dicatat_oleh)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		productID, nullableInt64(batchID), sistem, fisik, selisih, catatan, userID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func nullableInt64(v sql.NullInt64) interface{} {
	if !v.Valid {
		return nil
	}
	return v.Int64
}

func (q *Queries) MahasantriExists(ctx context.Context, santriID int64) (bool, error) {
	var total int64
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM santri WHERE id = ?`, santriID).Scan(&total)
	return total > 0, err
}

func (q *Queries) HasActiveMahasantriProduk(ctx context.Context, santriID, productID int64, batchID sql.NullInt64) (bool, error) {
	var total int64
	var err error
	if batchID.Valid {
		err = q.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM mahasantri_produk
			WHERE santri_id = ? AND produk_id = ? AND produk_batch_id = ? AND status = 'aktif'`,
			santriID, productID, batchID.Int64,
		).Scan(&total)
	} else {
		err = q.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM mahasantri_produk
			WHERE santri_id = ? AND produk_id = ? AND produk_batch_id IS NULL AND status = 'aktif'`,
			santriID, productID,
		).Scan(&total)
	}
	return total > 0, err
}

func (q *Queries) CreateMahasantriProduk(ctx context.Context, santriID, productID int64, batchID sql.NullInt64, tanggal, catatan string, userID int64) (int64, error) {
	res, err := q.db.ExecContext(ctx, `
		INSERT INTO mahasantri_produk
			(santri_id, produk_id, produk_batch_id, tanggal, status, catatan, dicatat_oleh)
		VALUES (?, ?, ?, ?, 'aktif', ?, ?)`,
		santriID, productID, nullableInt64(batchID), tanggal, catatan, userID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (q *Queries) GetMahasantriProduk(ctx context.Context, id int64) (models.MahasantriProduct, error) {
	var out models.MahasantriProduct
	var batchID sql.NullInt64
	var track int64
	err := q.db.QueryRowContext(ctx, `
		SELECT mp.id, mp.santri_id, mp.produk_id, mp.produk_batch_id,
		       p.nama, COALESCE(pb.nama, ''), p.kategori, p.track_stok,
		       strftime('%Y-%m-%d', mp.tanggal), mp.status, mp.catatan,
		       COALESCE(u.name, '')
		FROM mahasantri_produk mp
		JOIN produk p ON p.id = mp.produk_id
		LEFT JOIN produk_batch pb ON pb.id = mp.produk_batch_id
		LEFT JOIN users u ON u.id = mp.dicatat_oleh
		WHERE mp.id = ?`, id).Scan(
		&out.ID, &out.SantriID, &out.ProdukID, &batchID, &out.ProdukNama, &out.BatchNama,
		&out.Kategori, &track, &out.Tanggal, &out.Status, &out.Catatan, &out.DicatatOleh,
	)
	if batchID.Valid {
		out.ProdukBatchID = batchID.Int64
	}
	out.TrackStok = track == 1
	return out, err
}

func (q *Queries) CancelMahasantriProduk(ctx context.Context, id, userID int64) error {
	res, err := q.db.ExecContext(ctx, `
		UPDATE mahasantri_produk
		SET status = 'dibatalkan', dibatalkan_at = CURRENT_TIMESTAMP, dibatalkan_oleh = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'aktif'`, userID, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (q *Queries) ListMahasantriProduk(ctx context.Context, santriID int64) ([]models.MahasantriProduct, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT mp.id, mp.santri_id, mp.produk_id, mp.produk_batch_id,
		       p.nama, COALESCE(pb.nama, ''), p.kategori, p.track_stok,
		       strftime('%Y-%m-%d', mp.tanggal), mp.status, mp.catatan,
		       COALESCE(u.name, '')
		FROM mahasantri_produk mp
		JOIN produk p ON p.id = mp.produk_id
		LEFT JOIN produk_batch pb ON pb.id = mp.produk_batch_id
		LEFT JOIN users u ON u.id = mp.dicatat_oleh
		WHERE mp.santri_id = ? AND mp.status = 'aktif'
		ORDER BY mp.tanggal DESC, mp.id DESC`, santriID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.MahasantriProduct, 0)
	for rows.Next() {
		var item models.MahasantriProduct
		var batchID sql.NullInt64
		var track int64
		if err := rows.Scan(
			&item.ID, &item.SantriID, &item.ProdukID, &batchID,
			&item.ProdukNama, &item.BatchNama, &item.Kategori, &track,
			&item.Tanggal, &item.Status, &item.Catatan, &item.DicatatOleh,
		); err != nil {
			return nil, err
		}
		if batchID.Valid {
			item.ProdukBatchID = batchID.Int64
		}
		item.TrackStok = track == 1
		out = append(out, item)
	}
	return out, rows.Err()
}

func productCRMWhere(filters models.ProductCRMFilters) (string, []interface{}) {
	clauses := []string{"1 = 1"}
	args := make([]interface{}, 0, 12)
	add := func(clause string, value interface{}) {
		clauses = append(clauses, clause)
		args = append(args, value)
	}

	if strings.TrimSpace(filters.Search) != "" {
		clauses = append(clauses, "(LOWER(s.nama) LIKE LOWER(?) OR LOWER(s.id_mahasantri) LIKE LOWER(?))")
		term := "%" + strings.TrimSpace(filters.Search) + "%"
		args = append(args, term, term)
	}
	if filters.Angkatan != "" {
		add("s.angkatan = ?", filters.Angkatan)
	}
	if filters.Status != "" {
		add("s.status = ?", filters.Status)
	}
	if filters.HasProductID > 0 {
		clause := "EXISTS (SELECT 1 FROM mahasantri_produk mp WHERE mp.santri_id = s.id AND mp.produk_id = ? AND mp.status = 'aktif'"
		args = append(args, filters.HasProductID)
		if filters.BatchID > 0 {
			clause += " AND mp.produk_batch_id = ?"
			args = append(args, filters.BatchID)
		}
		clauses = append(clauses, clause+")")
	}
	if filters.MissingProductID > 0 {
		clauses = append(clauses, "NOT EXISTS (SELECT 1 FROM mahasantri_produk mp WHERE mp.santri_id = s.id AND mp.produk_id = ? AND mp.status = 'aktif')")
		args = append(args, filters.MissingProductID)
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (q *Queries) CountProductCRM(ctx context.Context, filters models.ProductCRMFilters) (int64, error) {
	where, args := productCRMWhere(filters)
	var total int64
	err := q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM santri s"+where, args...).Scan(&total)
	return total, err
}

func (q *Queries) ListProductCRMRows(ctx context.Context, filters models.ProductCRMFilters) ([]models.ProductCRMRow, error) {
	where, args := productCRMWhere(filters)
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	page := filters.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	query := `SELECT s.id, s.id_mahasantri, s.nama, s.angkatan, s.angkatan_kelas, s.level, s.status
		FROM santri s` + where + `
		ORDER BY s.nama ASC, s.id ASC
		LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.ProductCRMRow, 0, limit)
	for rows.Next() {
		var row models.ProductCRMRow
		if err := rows.Scan(&row.ID, &row.IDMahasantri, &row.Nama, &row.Angkatan, &row.AngkatanKelas, &row.Level, &row.Status); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (q *Queries) ListStockMutations(ctx context.Context, limit int64) ([]models.StockMutation, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := q.db.QueryContext(ctx, `
		SELECT sm.id, sm.produk_id, sm.produk_batch_id, p.nama, COALESCE(pb.nama, ''),
		       sm.tipe, sm.qty, sm.catatan, COALESCE(u.name, ''),
		       strftime('%Y-%m-%d %H:%M', sm.created_at)
		FROM stok_mutasi sm
		JOIN produk p ON p.id = sm.produk_id
		LEFT JOIN produk_batch pb ON pb.id = sm.produk_batch_id
		LEFT JOIN users u ON u.id = sm.dicatat_oleh
		ORDER BY sm.id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.StockMutation, 0)
	for rows.Next() {
		var item models.StockMutation
		var batchID sql.NullInt64
		if err := rows.Scan(
			&item.ID, &item.ProdukID, &batchID, &item.ProdukNama, &item.BatchNama,
			&item.Tipe, &item.Qty, &item.Catatan, &item.DicatatOleh, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		if batchID.Valid {
			item.ProdukBatchID = batchID.Int64
		}
		out = append(out, item)
	}
	return out, rows.Err()
}



func dashboardPopulationWhere(filters models.ProductCRMDashboardFilters) (string, []interface{}) {
	clauses := []string{"1 = 1"}
	args := make([]interface{}, 0, 8)
	if strings.TrimSpace(filters.Angkatan) != "" {
		clauses = append(clauses, "s.angkatan = ?")
		args = append(args, strings.TrimSpace(filters.Angkatan))
	}
	if strings.TrimSpace(filters.Status) != "" {
		clauses = append(clauses, "s.status = ?")
		args = append(args, strings.TrimSpace(filters.Status))
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func dashboardAssignmentWhere(filters models.ProductCRMDashboardFilters, includeDate bool) (string, []interface{}) {
	clauses := []string{"mp.status = 'aktif'"}
	args := make([]interface{}, 0, 12)
	if filters.ProductID > 0 {
		clauses = append(clauses, "mp.produk_id = ?")
		args = append(args, filters.ProductID)
	}
	if filters.BatchID > 0 {
		clauses = append(clauses, "mp.produk_batch_id = ?")
		args = append(args, filters.BatchID)
	}
	if strings.TrimSpace(filters.Category) != "" {
		clauses = append(clauses, "p.kategori = ?")
		args = append(args, strings.TrimSpace(filters.Category))
	}
	if strings.TrimSpace(filters.Angkatan) != "" {
		clauses = append(clauses, "s.angkatan = ?")
		args = append(args, strings.TrimSpace(filters.Angkatan))
	}
	if strings.TrimSpace(filters.Status) != "" {
		clauses = append(clauses, "s.status = ?")
		args = append(args, strings.TrimSpace(filters.Status))
	}
	if includeDate {
		if filters.DateFrom != "" {
			clauses = append(clauses, "date(mp.tanggal) >= date(?)")
			args = append(args, filters.DateFrom)
		}
		if filters.DateTo != "" {
			clauses = append(clauses, "date(mp.tanggal) <= date(?)")
			args = append(args, filters.DateTo)
		}
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func dashboardProductWhere(filters models.ProductCRMDashboardFilters) (string, []interface{}) {
	clauses := []string{"p.is_aktif = 1"}
	args := make([]interface{}, 0, 4)
	if filters.ProductID > 0 {
		clauses = append(clauses, "p.id = ?")
		args = append(args, filters.ProductID)
	}
	if strings.TrimSpace(filters.Category) != "" {
		clauses = append(clauses, "p.kategori = ?")
		args = append(args, strings.TrimSpace(filters.Category))
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func dashboardBucketExpr(bucket string, column string) string {
	switch bucket {
	case "day":
		return "strftime('%Y-%m-%d', " + column + ")"
	case "week":
		return "strftime('%Y-W%W', " + column + ")"
	default:
		return "strftime('%Y-%m', " + column + ")"
	}
}

func (q *Queries) ProductDashboardSummary(ctx context.Context, filters models.ProductCRMDashboardFilters) (models.ProductCRMDashboardSummary, error) {
	var out models.ProductCRMDashboardSummary
	productWhere, productArgs := dashboardProductWhere(filters)
	if err := q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM produk p"+productWhere, productArgs...).Scan(&out.ActiveProducts); err != nil {
		return out, err
	}

	populationWhere, populationArgs := dashboardPopulationWhere(filters)
	if err := q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM santri s"+populationWhere, populationArgs...).Scan(&out.SantriTotal); err != nil {
		return out, err
	}

	assignmentWhere, assignmentArgs := dashboardAssignmentWhere(filters, false)
	assignmentBase := " FROM mahasantri_produk mp JOIN produk p ON p.id = mp.produk_id JOIN santri s ON s.id = mp.santri_id" + assignmentWhere
	if err := q.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT mp.santri_id)"+assignmentBase, assignmentArgs...).Scan(&out.SantriWithProduct); err != nil {
		return out, err
	}
	if err := q.db.QueryRowContext(ctx, "SELECT COUNT(*)"+assignmentBase, assignmentArgs...).Scan(&out.ActiveOwnerships); err != nil {
		return out, err
	}
	if out.SantriTotal > 0 {
		out.CoverageRate = float64(out.SantriWithProduct) * 100 / float64(out.SantriTotal)
	}

	stockClauses := []string{"p.is_aktif = 1", "p.track_stok = 1"}
	stockArgs := make([]interface{}, 0, 4)
	if filters.ProductID > 0 {
		stockClauses = append(stockClauses, "p.id = ?")
		stockArgs = append(stockArgs, filters.ProductID)
	}
	if strings.TrimSpace(filters.Category) != "" {
		stockClauses = append(stockClauses, "p.kategori = ?")
		stockArgs = append(stockArgs, strings.TrimSpace(filters.Category))
	}
	joinStock := " LEFT JOIN stok_mutasi sm ON sm.produk_id = p.id"
	if filters.BatchID > 0 {
		joinStock += " AND sm.produk_batch_id = ?"
		stockArgs = append([]interface{}{filters.BatchID}, stockArgs...)
	}
	stockQuery := "SELECT COALESCE(SUM(sm.qty), 0) FROM produk p" + joinStock + " WHERE " + strings.Join(stockClauses, " AND ")
	if err := q.db.QueryRowContext(ctx, stockQuery, stockArgs...).Scan(&out.TotalStock); err != nil {
		return out, err
	}

	health, err := q.ProductDashboardInventoryHealth(ctx, filters, 100000)
	if err != nil {
		return out, err
	}
	for _, item := range health {
		if item.Status == "critical" || item.Status == "empty" {
			out.LowStockProducts++
		}
	}
	return out, nil
}

func (q *Queries) ProductDashboardPeriodAssignments(ctx context.Context, filters models.ProductCRMDashboardFilters) (int64, error) {
	where, args := dashboardAssignmentWhere(filters, true)
	var total int64
	err := q.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM mahasantri_produk mp
		JOIN produk p ON p.id = mp.produk_id
		JOIN santri s ON s.id = mp.santri_id`+where, args...).Scan(&total)
	return total, err
}

func (q *Queries) ProductDashboardDistributionTrend(ctx context.Context, filters models.ProductCRMDashboardFilters, bucket string) ([]models.ProductCRMTrendPoint, error) {
	where, args := dashboardAssignmentWhere(filters, true)
	expr := dashboardBucketExpr(bucket, "mp.tanggal")
	rows, err := q.db.QueryContext(ctx, `
		SELECT `+expr+` AS period, COUNT(*)
		FROM mahasantri_produk mp
		JOIN produk p ON p.id = mp.produk_id
		JOIN santri s ON s.id = mp.santri_id`+where+`
		GROUP BY period
		ORDER BY period ASC`, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	out := make([]models.ProductCRMTrendPoint, 0)
	for rows.Next() {
		var item models.ProductCRMTrendPoint
		if err := rows.Scan(&item.Period, &item.Total); err != nil { return nil, err }
		item.Label = item.Period
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Queries) ProductDashboardTopProducts(ctx context.Context, filters models.ProductCRMDashboardFilters, limit int64) ([]models.ProductCRMDashboardPoint, error) {
	where, args := dashboardAssignmentWhere(filters, true)
	args = append(args, limit)
	rows, err := q.db.QueryContext(ctx, `
		SELECT p.nama, CAST(p.id AS TEXT), COUNT(*)
		FROM mahasantri_produk mp
		JOIN produk p ON p.id = mp.produk_id
		JOIN santri s ON s.id = mp.santri_id`+where+`
		GROUP BY p.id, p.nama
		ORDER BY COUNT(*) DESC, p.nama ASC
		LIMIT ?`, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	out := make([]models.ProductCRMDashboardPoint, 0)
	for rows.Next() {
		var item models.ProductCRMDashboardPoint
		if err := rows.Scan(&item.Label, &item.Value, &item.Total); err != nil { return nil, err }
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Queries) ProductDashboardCategoryComposition(ctx context.Context, filters models.ProductCRMDashboardFilters) ([]models.ProductCRMDashboardPoint, error) {
	where, args := dashboardAssignmentWhere(filters, true)
	rows, err := q.db.QueryContext(ctx, `
		SELECT CASE p.kategori WHEN 'buku' THEN 'Buku' WHEN 'program' THEN 'Program' ELSE p.kategori END,
		       p.kategori, COUNT(*)
		FROM mahasantri_produk mp
		JOIN produk p ON p.id = mp.produk_id
		JOIN santri s ON s.id = mp.santri_id`+where+`
		GROUP BY p.kategori
		ORDER BY COUNT(*) DESC`, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	out := make([]models.ProductCRMDashboardPoint, 0)
	for rows.Next() {
		var item models.ProductCRMDashboardPoint
		if err := rows.Scan(&item.Label, &item.Value, &item.Total); err != nil { return nil, err }
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Queries) ProductDashboardCoverageByAngkatan(ctx context.Context, filters models.ProductCRMDashboardFilters, limit int64) ([]models.ProductCRMDashboardPoint, error) {
	populationWhere, populationArgs := dashboardPopulationWhere(filters)
	subClauses := []string{"mp.status = 'aktif'", "mp.santri_id = s.id"}
	subArgs := make([]interface{}, 0, 4)
	if filters.ProductID > 0 {
		subClauses = append(subClauses, "mp.produk_id = ?")
		subArgs = append(subArgs, filters.ProductID)
	}
	if filters.BatchID > 0 {
		subClauses = append(subClauses, "mp.produk_batch_id = ?")
		subArgs = append(subArgs, filters.BatchID)
	}
	if strings.TrimSpace(filters.Category) != "" {
		subClauses = append(subClauses, "p.kategori = ?")
		subArgs = append(subArgs, strings.TrimSpace(filters.Category))
	}
	query := `
		SELECT COALESCE(NULLIF(TRIM(s.angkatan), ''), 'Tanpa Angkatan') AS label,
		       COALESCE(NULLIF(TRIM(s.angkatan), ''), '') AS value,
		       CAST(ROUND(100.0 * SUM(CASE WHEN EXISTS (
		           SELECT 1 FROM mahasantri_produk mp
		           JOIN produk p ON p.id = mp.produk_id
		           WHERE `+strings.Join(subClauses, " AND ")+`
		       ) THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0)) AS INTEGER) AS coverage
		FROM santri s`+populationWhere+`
		GROUP BY COALESCE(NULLIF(TRIM(s.angkatan), ''), 'Tanpa Angkatan')
		ORDER BY coverage ASC, label ASC
		LIMIT ?`
	args := append(subArgs, populationArgs...)
	args = append(args, limit)
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	out := make([]models.ProductCRMDashboardPoint, 0)
	for rows.Next() {
		var item models.ProductCRMDashboardPoint
		if err := rows.Scan(&item.Label, &item.Value, &item.Total); err != nil { return nil, err }
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Queries) ProductDashboardInventoryHealth(ctx context.Context, filters models.ProductCRMDashboardFilters, limit int64) ([]models.ProductCRMInventoryHealth, error) {
	clauses := []string{"p.is_aktif = 1", "p.track_stok = 1"}
	args := make([]interface{}, 0, 5)
	if filters.ProductID > 0 {
		clauses = append(clauses, "p.id = ?")
		args = append(args, filters.ProductID)
	}
	if strings.TrimSpace(filters.Category) != "" {
		clauses = append(clauses, "p.kategori = ?")
		args = append(args, strings.TrimSpace(filters.Category))
	}
	join := " LEFT JOIN stok_mutasi sm ON sm.produk_id = p.id"
	if filters.BatchID > 0 {
		join += " AND sm.produk_batch_id = ?"
		args = append([]interface{}{filters.BatchID}, args...)
	}
	args = append(args, limit)
	rows, err := q.db.QueryContext(ctx, `
		SELECT p.id, p.nama, COALESCE(SUM(sm.qty), 0) AS stock, p.minimum_stock
		FROM produk p`+join+`
		WHERE `+strings.Join(clauses, " AND ")+`
		GROUP BY p.id, p.nama, p.minimum_stock
		ORDER BY CASE WHEN COALESCE(SUM(sm.qty), 0) <= p.minimum_stock THEN 0 ELSE 1 END,
		         COALESCE(SUM(sm.qty), 0) ASC, p.nama ASC
		LIMIT ?`, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	out := make([]models.ProductCRMInventoryHealth, 0)
	for rows.Next() {
		var item models.ProductCRMInventoryHealth
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.Stock, &item.MinimumStock); err != nil { return nil, err }
		switch {
		case item.Stock <= 0:
			item.Status = "empty"
		case item.Stock <= item.MinimumStock:
			item.Status = "critical"
		default:
			item.Status = "safe"
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Queries) ProductDashboardStockMovement(ctx context.Context, filters models.ProductCRMDashboardFilters, bucket string) ([]models.ProductCRMStockMovement, error) {
	clauses := []string{"p.track_stok = 1"}
	args := make([]interface{}, 0, 10)
	if filters.ProductID > 0 {
		clauses = append(clauses, "sm.produk_id = ?")
		args = append(args, filters.ProductID)
	}
	if filters.BatchID > 0 {
		clauses = append(clauses, "sm.produk_batch_id = ?")
		args = append(args, filters.BatchID)
	}
	if strings.TrimSpace(filters.Category) != "" {
		clauses = append(clauses, "p.kategori = ?")
		args = append(args, strings.TrimSpace(filters.Category))
	}
	if filters.DateFrom != "" {
		clauses = append(clauses, "date(sm.created_at) >= date(?)")
		args = append(args, filters.DateFrom)
	}
	if filters.DateTo != "" {
		clauses = append(clauses, "date(sm.created_at) <= date(?)")
		args = append(args, filters.DateTo)
	}
	expr := dashboardBucketExpr(bucket, "sm.created_at")
	rows, err := q.db.QueryContext(ctx, `
		SELECT `+expr+` AS period,
		       COALESCE(SUM(CASE WHEN sm.qty > 0 AND sm.tipe != 'pembatalan' THEN sm.qty ELSE 0 END), 0) AS masuk,
		       COALESCE(SUM(CASE WHEN sm.qty < 0 THEN ABS(sm.qty) ELSE 0 END), 0) AS keluar
		FROM stok_mutasi sm
		JOIN produk p ON p.id = sm.produk_id
		WHERE `+strings.Join(clauses, " AND ")+`
		GROUP BY period
		ORDER BY period ASC`, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	out := make([]models.ProductCRMStockMovement, 0)
	for rows.Next() {
		var item models.ProductCRMStockMovement
		if err := rows.Scan(&item.Period, &item.Masuk, &item.Keluar); err != nil { return nil, err }
		item.Label = item.Period
		out = append(out, item)
	}
	return out, rows.Err()
}


func dashboardDrilldownSubquery(filters models.ProductCRMDashboardFilters, includeDate bool, period string) (string, []interface{}) {
	clauses := []string{"mp2.santri_id = s.id", "mp2.status = 'aktif'"}
	args := make([]interface{}, 0, 12)
	if filters.ProductID > 0 {
		clauses = append(clauses, "mp2.produk_id = ?")
		args = append(args, filters.ProductID)
	}
	if filters.BatchID > 0 {
		clauses = append(clauses, "mp2.produk_batch_id = ?")
		args = append(args, filters.BatchID)
	}
	if strings.TrimSpace(filters.Category) != "" {
		clauses = append(clauses, "p2.kategori = ?")
		args = append(args, strings.TrimSpace(filters.Category))
	}
	if includeDate {
		if filters.DateFrom != "" {
			clauses = append(clauses, "date(mp2.tanggal) >= date(?)")
			args = append(args, filters.DateFrom)
		}
		if filters.DateTo != "" {
			clauses = append(clauses, "date(mp2.tanggal) <= date(?)")
			args = append(args, filters.DateTo)
		}
	}
	period = strings.TrimSpace(period)
	if period != "" {
		switch {
		case len(period) == 10:
			clauses = append(clauses, "strftime('%Y-%m-%d', mp2.tanggal) = ?")
			args = append(args, period)
		case strings.Contains(period, "-W"):
			clauses = append(clauses, "strftime('%Y-W%W', mp2.tanggal) = ?")
			args = append(args, period)
		case len(period) == 7:
			clauses = append(clauses, "strftime('%Y-%m', mp2.tanggal) = ?")
			args = append(args, period)
		}
	}
	return "SELECT 1 FROM mahasantri_produk mp2 JOIN produk p2 ON p2.id = mp2.produk_id WHERE " + strings.Join(clauses, " AND "), args
}

func dashboardDrilldownWhere(filters models.ProductCRMDashboardFilters, mode, period string) (string, []interface{}) {
	where, args := dashboardPopulationWhere(filters)
	clauses := []string{strings.TrimPrefix(where, " WHERE ")}
	allArgs := append([]interface{}{}, args...)
	switch mode {
	case "period":
		subquery, subArgs := dashboardDrilldownSubquery(filters, true, period)
		clauses = append(clauses, "EXISTS ("+subquery+")")
		allArgs = append(allArgs, subArgs...)
	case "has":
		subquery, subArgs := dashboardDrilldownSubquery(filters, false, "")
		clauses = append(clauses, "EXISTS ("+subquery+")")
		allArgs = append(allArgs, subArgs...)
	case "missing":
		subquery, subArgs := dashboardDrilldownSubquery(filters, false, "")
		clauses = append(clauses, "NOT EXISTS ("+subquery+")")
		allArgs = append(allArgs, subArgs...)
	}
	return " WHERE " + strings.Join(clauses, " AND "), allArgs
}

func (q *Queries) CountProductDashboardDrilldown(ctx context.Context, filters models.ProductCRMDashboardFilters, mode, period string) (int64, error) {
	where, args := dashboardDrilldownWhere(filters, mode, period)
	var total int64
	err := q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM santri s"+where, args...).Scan(&total)
	return total, err
}

func (q *Queries) ListProductDashboardDrilldown(ctx context.Context, filters models.ProductCRMDashboardFilters, mode, period string, page, limit int64) ([]models.ProductCRMDrilldownRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	where, args := dashboardDrilldownWhere(filters, mode, period)
	args = append(args, limit, (page-1)*limit)
	rows, err := q.db.QueryContext(ctx, `
		SELECT s.id, COALESCE(s.id_mahasantri, ''), s.nama, COALESCE(s.no_wa, ''),
		       COALESCE(s.angkatan, ''), COALESCE(s.angkatan_kelas, ''), COALESCE(s.level, ''), COALESCE(s.status, '')
		FROM santri s`+where+`
		ORDER BY s.nama ASC, s.id ASC
		LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.ProductCRMDrilldownRow, 0, limit)
	for rows.Next() {
		var item models.ProductCRMDrilldownRow
		if err := rows.Scan(
			&item.ID, &item.IDMahasantri, &item.Nama, &item.NoWA,
			&item.Angkatan, &item.AngkatanKelas, &item.Level, &item.Status,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Queries) ListMahasantriOwnedPairs(ctx context.Context, santriID int64) ([]models.ProductCRMDrilldownOwned, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT produk_id, COALESCE(produk_batch_id, 0)
		FROM mahasantri_produk
		WHERE santri_id = ? AND status = 'aktif'
		ORDER BY produk_id, COALESCE(produk_batch_id, 0)`, santriID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.ProductCRMDrilldownOwned, 0)
	for rows.Next() {
		var item models.ProductCRMDrilldownOwned
		if err := rows.Scan(&item.ProductID, &item.BatchID); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
