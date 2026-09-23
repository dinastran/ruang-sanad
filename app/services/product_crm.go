package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var (
	ErrProdukTidakAktif = errors.New("produk tidak aktif")
	ErrStokHabis         = errors.New("stok produk habis")
	ErrProdukSudahAda    = errors.New("produk sudah tercatat pada mahasantri")
)

type ProductCRMService struct {
	querier *queries.Querier
}

func NewProductCRMService(querier *queries.Querier) *ProductCRMService {
	return &ProductCRMService{querier: querier}
}

func validProductCategory(category string) bool {
	return category == "buku" || category == "program"
}

func validDate(value string) bool {
	if value == "" {
		return false
	}
	parsed, err := time.Parse(time.DateOnly, value)
	return err == nil && parsed.Format(time.DateOnly) == value
}

func nullBatch(id int64) sql.NullInt64 {
	return sql.NullInt64{Int64: id, Valid: id > 0}
}

func nullRef(id int64) sql.NullInt64 {
	return sql.NullInt64{Int64: id, Valid: id > 0}
}

func nullDate(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}

func (s *ProductCRMService) Products() ([]models.Product, error) {
	ctx := context.Background()
	items, err := s.querier.ListProduk(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		batches, err := s.querier.ListProdukBatch(ctx, items[i].ID)
		if err != nil {
			return nil, err
		}
		items[i].Batches = batches
	}
	return items, nil
}

func (s *ProductCRMService) CreateProduct(req models.CreateProductRequest) error {
	req.Nama = strings.TrimSpace(req.Nama)
	req.Kategori = strings.ToLower(strings.TrimSpace(req.Kategori))
	if req.Nama == "" {
		return errors.New("nama produk wajib diisi")
	}
	if !validProductCategory(req.Kategori) {
		return errors.New("kategori produk tidak valid")
	}
	if req.MinimumStock < 0 {
		return errors.New("minimum stok tidak boleh negatif")
	}
	if req.Kategori == "program" || !req.TrackStok {
		req.TrackStok = false
		req.MinimumStock = 0
	}
	_, err := s.querier.CreateProduk(context.Background(), req.Nama, req.Kategori, req.TrackStok, req.MinimumStock)
	return err
}

func (s *ProductCRMService) UpdateProduct(id int64, req models.UpdateProductRequest) error {
	ctx := context.Background()
	req.Nama = strings.TrimSpace(req.Nama)
	req.Kategori = strings.ToLower(strings.TrimSpace(req.Kategori))
	if req.Nama == "" || !validProductCategory(req.Kategori) {
		return errors.New("data produk tidak valid")
	}
	current, err := s.querier.GetProduk(ctx, id)
	if err != nil {
		return err
	}
	if req.MinimumStock < 0 {
		return errors.New("minimum stok tidak boleh negatif")
	}
	if req.Kategori == "program" || !req.TrackStok {
		req.TrackStok = false
		req.MinimumStock = 0
	}
	if current.Kategori != req.Kategori || current.TrackStok != req.TrackStok {
		hasHistory, err := s.querier.ProdukHasHistory(ctx, id)
		if err != nil {
			return err
		}
		if hasHistory {
			return errors.New("kategori dan mode stok tidak dapat diubah setelah produk memiliki histori transaksi")
		}
	}
	return s.querier.UpdateProduk(ctx, id, req.Nama, req.Kategori, req.TrackStok, req.IsAktif, req.MinimumStock)
}

func (s *ProductCRMService) CreateBatch(productID int64, req models.CreateProductBatchRequest) error {
	ctx := context.Background()
	if _, err := s.querier.GetProduk(ctx, productID); err != nil {
		return err
	}
	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" {
		return errors.New("nama batch/edisi wajib diisi")
	}
	if req.TanggalMulai != "" && !validDate(req.TanggalMulai) {
		return errors.New("tanggal mulai tidak valid")
	}
	if req.TanggalSelesai != "" && !validDate(req.TanggalSelesai) {
		return errors.New("tanggal selesai tidak valid")
	}
	if req.TanggalMulai != "" && req.TanggalSelesai != "" && req.TanggalSelesai < req.TanggalMulai {
		return errors.New("tanggal selesai tidak boleh sebelum tanggal mulai")
	}
	_, err := s.querier.CreateProdukBatch(ctx, productID, req.Nama, nullDate(req.TanggalMulai), nullDate(req.TanggalSelesai))
	return err
}

func (s *ProductCRMService) UpdateBatch(id int64, req models.UpdateProductBatchRequest) error {
	ctx := context.Background()
	if _, err := s.querier.GetProdukBatch(ctx, id); err != nil {
		return err
	}
	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" {
		return errors.New("nama batch/edisi wajib diisi")
	}
	if req.TanggalMulai != "" && !validDate(req.TanggalMulai) {
		return errors.New("tanggal mulai tidak valid")
	}
	if req.TanggalSelesai != "" && !validDate(req.TanggalSelesai) {
		return errors.New("tanggal selesai tidak valid")
	}
	if req.TanggalMulai != "" && req.TanggalSelesai != "" && req.TanggalSelesai < req.TanggalMulai {
		return errors.New("tanggal selesai tidak boleh sebelum tanggal mulai")
	}
	return s.querier.UpdateProdukBatch(ctx, id, req.Nama, nullDate(req.TanggalMulai), nullDate(req.TanggalSelesai), req.IsAktif)
}

func (s *ProductCRMService) validateBatch(ctx context.Context, q *queries.Querier, product models.Product, batchID int64) (sql.NullInt64, error) {
	activeBatches, err := q.CountActiveProdukBatches(ctx, product.ID)
	if err != nil {
		return sql.NullInt64{}, err
	}
	if activeBatches > 0 && batchID <= 0 {
		return sql.NullInt64{}, errors.New("batch/edisi wajib dipilih untuk produk ini")
	}
	if batchID <= 0 {
		return sql.NullInt64{}, nil
	}
	batch, err := q.GetProdukBatch(ctx, batchID)
	if err != nil {
		return sql.NullInt64{}, errors.New("batch/edisi tidak ditemukan")
	}
	if batch.ProdukID != product.ID {
		return sql.NullInt64{}, errors.New("batch/edisi tidak sesuai dengan produk")
	}
	if !batch.IsAktif {
		return sql.NullInt64{}, errors.New("batch/edisi sudah tidak aktif")
	}
	return nullBatch(batchID), nil
}

func (s *ProductCRMService) AddStock(req models.StockEntryRequest, userID int64) error {
	if req.Qty <= 0 {
		return errors.New("jumlah stok masuk harus lebih dari nol")
	}
	if req.Tipe != "stok_awal" && req.Tipe != "stok_masuk" {
		return errors.New("tipe stok tidak valid")
	}
	ctx := context.Background()
	product, err := s.querier.GetProduk(ctx, req.ProdukID)
	if err != nil {
		return err
	}
	if !product.TrackStok {
		return errors.New("produk ini tidak menggunakan stok")
	}
	batch, err := s.validateBatch(ctx, s.querier, product, req.ProdukBatchID)
	if err != nil {
		return err
	}
	_, err = s.querier.InsertStockMutation(ctx, product.ID, batch, req.Tipe, req.Qty, "", sql.NullInt64{}, strings.TrimSpace(req.Catatan), userID)
	return err
}

func (s *ProductCRMService) Opname(req models.StockOpnameRequest, userID int64) error {
	if req.StokFisik < 0 {
		return errors.New("stok fisik tidak boleh negatif")
	}
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	product, err := q.GetProduk(ctx, req.ProdukID)
	if err != nil {
		return err
	}
	if !product.TrackStok {
		return errors.New("produk ini tidak menggunakan stok")
	}
	batch, err := s.validateBatch(ctx, q, product, req.ProdukBatchID)
	if err != nil {
		return err
	}
	current, err := q.CurrentProdukStock(ctx, product.ID, batch)
	if err != nil {
		return err
	}
	delta := req.StokFisik - current
	opnameID, err := q.InsertStockOpname(ctx, product.ID, batch, current, req.StokFisik, delta, strings.TrimSpace(req.Catatan), userID)
	if err != nil {
		return err
	}
	if delta != 0 {
		if _, err := q.InsertStockMutation(ctx, product.ID, batch, "opname", delta, "stok_opname", nullRef(opnameID), strings.TrimSpace(req.Catatan), userID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *ProductCRMService) AssignProduct(santriID int64, req models.AssignProductRequest, userID int64) error {
	if !validDate(req.Tanggal) {
		return errors.New("tanggal transaksi wajib berformat YYYY-MM-DD")
	}
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	exists, err := q.MahasantriExists(ctx, santriID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("mahasantri tidak ditemukan")
	}
	product, err := q.GetProduk(ctx, req.ProdukID)
	if err != nil {
		return err
	}
	if !product.IsAktif {
		return ErrProdukTidakAktif
	}
	batch, err := s.validateBatch(ctx, q, product, req.ProdukBatchID)
	if err != nil {
		return err
	}
	already, err := q.HasActiveMahasantriProduk(ctx, santriID, product.ID, batch)
	if err != nil {
		return err
	}
	if already {
		return ErrProdukSudahAda
	}
	if product.TrackStok {
		stock, err := q.CurrentProdukStock(ctx, product.ID, batch)
		if err != nil {
			return err
		}
		if stock <= 0 {
			return ErrStokHabis
		}
	}
	assignmentID, err := q.CreateMahasantriProduk(ctx, santriID, product.ID, batch, req.Tanggal, strings.TrimSpace(req.Catatan), userID)
	if err != nil {
		return err
	}
	if product.TrackStok {
		if _, err := q.InsertStockMutation(ctx, product.ID, batch, "pembelian", -1, "mahasantri_produk", nullRef(assignmentID), fmt.Sprintf("Pembelian oleh mahasantri #%d", santriID), userID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *ProductCRMService) CancelAssignment(id, userID int64) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.querier.WithTx(tx)

	item, err := q.GetMahasantriProduk(ctx, id)
	if err != nil {
		return err
	}
	if item.Status != "aktif" {
		return errors.New("transaksi sudah dibatalkan")
	}
	if err := q.CancelMahasantriProduk(ctx, id, userID); err != nil {
		return err
	}
	if item.TrackStok {
		batch := nullBatch(item.ProdukBatchID)
		if _, err := q.InsertStockMutation(ctx, item.ProdukID, batch, "pembatalan", 1, "mahasantri_produk", nullRef(item.ID), "Pembatalan transaksi produk mahasantri", userID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func normalizeProductDashboardFilters(filters models.ProductCRMDashboardFilters) (models.ProductCRMDashboardFilters, time.Time, time.Time, error) {
	now := time.Now()
	if strings.TrimSpace(filters.DateTo) == "" {
		filters.DateTo = now.Format(time.DateOnly)
	}
	if strings.TrimSpace(filters.DateFrom) == "" {
		filters.DateFrom = now.AddDate(0, 0, -29).Format(time.DateOnly)
	}
	from, err := time.Parse(time.DateOnly, filters.DateFrom)
	if err != nil {
		return filters, time.Time{}, time.Time{}, errors.New("tanggal awal dashboard tidak valid")
	}
	to, err := time.Parse(time.DateOnly, filters.DateTo)
	if err != nil {
		return filters, time.Time{}, time.Time{}, errors.New("tanggal akhir dashboard tidak valid")
	}
	if to.Before(from) {
		return filters, time.Time{}, time.Time{}, errors.New("tanggal akhir tidak boleh sebelum tanggal awal")
	}
	filters.Category = strings.ToLower(strings.TrimSpace(filters.Category))
	if filters.Category != "" && !validProductCategory(filters.Category) {
		return filters, time.Time{}, time.Time{}, errors.New("kategori dashboard tidak valid")
	}
	filters.Angkatan = strings.TrimSpace(filters.Angkatan)
	filters.Status = strings.TrimSpace(filters.Status)
	return filters, from, to, nil
}

func productDashboardBucket(from, to time.Time) string {
	days := int(to.Sub(from).Hours()/24) + 1
	switch {
	case days <= 45:
		return "day"
	case days <= 180:
		return "week"
	default:
		return "month"
	}
}

func (s *ProductCRMService) Dashboard(filters models.ProductCRMDashboardFilters) (*models.ProductCRMDashboard, models.ProductCRMDashboardFilters, error) {
	filters, from, to, err := normalizeProductDashboardFilters(filters)
	if err != nil {
		return nil, filters, err
	}
	ctx := context.Background()
	bucket := productDashboardBucket(from, to)

	summary, err := s.querier.ProductDashboardSummary(ctx, filters)
	if err != nil {
		return nil, filters, err
	}
	summary.PeriodAssignments, err = s.querier.ProductDashboardPeriodAssignments(ctx, filters)
	if err != nil {
		return nil, filters, err
	}

	periodDays := int(to.Sub(from).Hours()/24) + 1
	previous := filters
	previousTo := from.AddDate(0, 0, -1)
	previousFrom := previousTo.AddDate(0, 0, -(periodDays - 1))
	previous.DateFrom = previousFrom.Format(time.DateOnly)
	previous.DateTo = previousTo.Format(time.DateOnly)
	summary.PreviousAssignments, err = s.querier.ProductDashboardPeriodAssignments(ctx, previous)
	if err != nil {
		return nil, filters, err
	}
	if summary.PreviousAssignments > 0 {
		summary.AssignmentChangeRate = float64(summary.PeriodAssignments-summary.PreviousAssignments) * 100 / float64(summary.PreviousAssignments)
	} else if summary.PeriodAssignments > 0 {
		summary.AssignmentChangeRate = 100
	}

	trend, err := s.querier.ProductDashboardDistributionTrend(ctx, filters, bucket)
	if err != nil { return nil, filters, err }
	topProducts, err := s.querier.ProductDashboardTopProducts(ctx, filters, 8)
	if err != nil { return nil, filters, err }
	composition, err := s.querier.ProductDashboardCategoryComposition(ctx, filters)
	if err != nil { return nil, filters, err }
	coverage, err := s.querier.ProductDashboardCoverageByAngkatan(ctx, filters, 10)
	if err != nil { return nil, filters, err }
	movement, err := s.querier.ProductDashboardStockMovement(ctx, filters, bucket)
	if err != nil { return nil, filters, err }
	health, err := s.querier.ProductDashboardInventoryHealth(ctx, filters, 10)
	if err != nil { return nil, filters, err }

	return &models.ProductCRMDashboard{
		Summary: summary,
		DistributionTrend: trend,
		TopProducts: topProducts,
		CategoryComposition: composition,
		CoverageByAngkatan: coverage,
		StockMovement: movement,
		InventoryHealth: health,
	}, filters, nil
}

func (s *ProductCRMService) ListCRM(filters models.ProductCRMFilters) (*models.ProductCRMList, error) {
	ctx := context.Background()
	total, err := s.querier.CountProductCRM(ctx, filters)
	if err != nil {
		return nil, err
	}
	rows, err := s.querier.ListProductCRMRows(ctx, filters)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		products, err := s.querier.ListMahasantriProduk(ctx, rows[i].ID)
		if err != nil {
			return nil, err
		}
		rows[i].Produk = products
	}
	return &models.ProductCRMList{Data: rows, Total: total}, nil
}

func (s *ProductCRMService) ListMahasantriProducts(santriID int64) ([]models.MahasantriProduct, error) {
	return s.querier.ListMahasantriProduk(context.Background(), santriID)
}

func (s *ProductCRMService) Mutations(limit int64) ([]models.StockMutation, error) {
	return s.querier.ListStockMutations(context.Background(), limit)
}
