package services

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func productCRMLastID(db *sql.DB) (int64, error) {
	var id int64
	err := db.QueryRow(`SELECT last_insert_rowid()`).Scan(&id)
	return id, err
}

type productCRMFixture struct {
	db      *sql.DB
	querier *queries.Querier
	service *ProductCRMService
	adminID int64
	santriID int64
}

func setupProductCRM(t *testing.T) productCRMFixture {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))

	_, err = db.Exec(`INSERT INTO users (email, name, role) VALUES ('produk@example.com', 'Admin Kelas', 'admin_kelas')`)
	require.NoError(t, err)
	adminID, err := productCRMLastID(db)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO santri (id_mahasantri, nama, jenis_kelamin, status) VALUES ('MHS.TEST.0001.092026', 'Ahmad', 'L', 'aktif')`)
	require.NoError(t, err)
	santriID, err := productCRMLastID(db)
	require.NoError(t, err)

	q := queries.NewQuerier(db)
	return productCRMFixture{db: db, querier: q, service: NewProductCRMService(q), adminID: adminID, santriID: santriID}
}

func (f productCRMFixture) createBook(t *testing.T, name string) int64 {
	t.Helper()
	require.NoError(t, f.service.CreateProduct(models.CreateProductRequest{Nama: name, Kategori: "buku", TrackStok: true}))
	products, err := f.service.Products()
	require.NoError(t, err)
	for _, product := range products {
		if product.Nama == name {
			return product.ID
		}
	}
	t.Fatal("book not created")
	return 0
}

func TestProductPurchaseReducesAndCancellationRestoresStock(t *testing.T) {
	f := setupProductCRM(t)
	productID := f.createBook(t, "Buku Tahsin 1")
	require.NoError(t, f.service.AddStock(models.StockEntryRequest{ProdukID: productID, Tipe: "stok_awal", Qty: 2}, f.adminID))

	require.NoError(t, f.service.AssignProduct(f.santriID, models.AssignProductRequest{
		ProdukID: productID, Tanggal: "2026-09-23",
	}, f.adminID))
	stock, err := f.querier.CurrentProdukStock(t.Context(), productID, sql.NullInt64{})
	require.NoError(t, err)
	require.Equal(t, int64(1), stock)

	owned, err := f.service.ListMahasantriProducts(f.santriID)
	require.NoError(t, err)
	require.Len(t, owned, 1)
	require.Equal(t, "Buku Tahsin 1", owned[0].ProdukNama)

	require.NoError(t, f.service.CancelAssignment(owned[0].ID, f.adminID))
	stock, err = f.querier.CurrentProdukStock(t.Context(), productID, sql.NullInt64{})
	require.NoError(t, err)
	require.Equal(t, int64(2), stock)
	owned, err = f.service.ListMahasantriProducts(f.santriID)
	require.NoError(t, err)
	require.Empty(t, owned)
}

func TestProductPurchaseRejectedWhenStockEmpty(t *testing.T) {
	f := setupProductCRM(t)
	productID := f.createBook(t, "Buku Kosong")
	err := f.service.AssignProduct(f.santriID, models.AssignProductRequest{ProdukID: productID, Tanggal: "2026-09-23"}, f.adminID)
	require.ErrorIs(t, err, ErrStokHabis)
}

func TestProgramWithBatchRequiresBatchAndDoesNotChangeStock(t *testing.T) {
	f := setupProductCRM(t)
	require.NoError(t, f.service.CreateProduct(models.CreateProductRequest{Nama: "Dauroh Siroh", Kategori: "program"}))
	products, err := f.service.Products()
	require.NoError(t, err)
	var productID int64
	for _, p := range products {
		if p.Nama == "Dauroh Siroh" {
			productID = p.ID
		}
	}
	require.NotZero(t, productID)
	require.NoError(t, f.service.CreateBatch(productID, models.CreateProductBatchRequest{Nama: "Batch 1", TanggalMulai: "2026-10-01"}))
	products, err = f.service.Products()
	require.NoError(t, err)
	var batchID int64
	for _, p := range products {
		if p.ID == productID {
			require.Len(t, p.Batches, 1)
			batchID = p.Batches[0].ID
		}
	}

	err = f.service.AssignProduct(f.santriID, models.AssignProductRequest{ProdukID: productID, Tanggal: "2026-09-23"}, f.adminID)
	require.ErrorContains(t, err, "batch/edisi wajib")

	require.NoError(t, f.service.AssignProduct(f.santriID, models.AssignProductRequest{
		ProdukID: productID, ProdukBatchID: batchID, Tanggal: "2026-09-23",
	}, f.adminID))
	owned, err := f.service.ListMahasantriProducts(f.santriID)
	require.NoError(t, err)
	require.Len(t, owned, 1)
	require.Equal(t, "Batch 1", owned[0].BatchNama)
	stock, err := f.querier.CurrentProdukStock(t.Context(), productID, sql.NullInt64{Int64: batchID, Valid: true})
	require.NoError(t, err)
	require.Zero(t, stock)
}

func TestProductHistoryLocksCategoryAndStockMode(t *testing.T) {
	f := setupProductCRM(t)
	productID := f.createBook(t, "Buku Terkunci")
	require.NoError(t, f.service.AddStock(models.StockEntryRequest{ProdukID: productID, Tipe: "stok_awal", Qty: 1}, f.adminID))
	require.NoError(t, f.service.AssignProduct(f.santriID, models.AssignProductRequest{ProdukID: productID, Tanggal: "2026-09-23"}, f.adminID))

	err := f.service.UpdateProduct(productID, models.UpdateProductRequest{
		Nama: "Buku Terkunci", Kategori: "program", IsAktif: true,
	})
	require.ErrorContains(t, err, "histori transaksi")
}

func TestProductOpnameCreatesAdjustment(t *testing.T) {
	f := setupProductCRM(t)
	productID := f.createBook(t, "Buku Opname")
	require.NoError(t, f.service.AddStock(models.StockEntryRequest{ProdukID: productID, Tipe: "stok_masuk", Qty: 5}, f.adminID))
	require.NoError(t, f.service.Opname(models.StockOpnameRequest{ProdukID: productID, StokFisik: 3, Catatan: "Hitung gudang"}, f.adminID))

	stock, err := f.querier.CurrentProdukStock(t.Context(), productID, sql.NullInt64{})
	require.NoError(t, err)
	require.Equal(t, int64(3), stock)
	mutations, err := f.service.Mutations(10)
	require.NoError(t, err)
	require.NotEmpty(t, mutations)
	require.Equal(t, "opname", mutations[0].Tipe)
	require.Equal(t, int64(-2), mutations[0].Qty)
}


func TestProductDashboardSummarizesCRMAndInventory(t *testing.T) {
	f := setupProductCRM(t)
	require.NoError(t, f.service.CreateProduct(models.CreateProductRequest{
		Nama: "Buku Intelligence", Kategori: "buku", TrackStok: true, MinimumStock: 2,
	}))
	products, err := f.service.Products()
	require.NoError(t, err)
	var productID int64
	for _, p := range products {
		if p.Nama == "Buku Intelligence" {
			productID = p.ID
			require.Equal(t, int64(2), p.MinimumStock)
		}
	}
	require.NotZero(t, productID)
	require.NoError(t, f.service.AddStock(models.StockEntryRequest{
		ProdukID: productID, Tipe: "stok_awal", Qty: 3,
	}, f.adminID))
	require.NoError(t, f.service.AssignProduct(f.santriID, models.AssignProductRequest{
		ProdukID: productID, Tanggal: "2026-09-23",
	}, f.adminID))

	dashboard, filters, err := f.service.Dashboard(models.ProductCRMDashboardFilters{
		DateFrom: "2026-09-01",
		DateTo: "2026-09-30",
		ProductID: productID,
	})
	require.NoError(t, err)
	require.Equal(t, "2026-09-01", filters.DateFrom)
	require.Equal(t, int64(1), dashboard.Summary.ActiveProducts)
	require.Equal(t, int64(1), dashboard.Summary.SantriTotal)
	require.Equal(t, int64(1), dashboard.Summary.SantriWithProduct)
	require.Equal(t, float64(100), dashboard.Summary.CoverageRate)
	require.Equal(t, int64(1), dashboard.Summary.ActiveOwnerships)
	require.Equal(t, int64(2), dashboard.Summary.TotalStock)
	require.Equal(t, int64(1), dashboard.Summary.LowStockProducts)
	require.Equal(t, int64(1), dashboard.Summary.PeriodAssignments)
	require.NotEmpty(t, dashboard.DistributionTrend)
	require.NotEmpty(t, dashboard.TopProducts)
	require.Equal(t, "Buku Intelligence", dashboard.TopProducts[0].Label)
	require.NotEmpty(t, dashboard.InventoryHealth)
	require.Equal(t, int64(2), dashboard.InventoryHealth[0].Stock)
	require.Equal(t, "critical", dashboard.InventoryHealth[0].Status)
}
