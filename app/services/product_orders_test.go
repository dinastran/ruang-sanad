package services

import (
	"database/sql"
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestManualProductOrderReducesAndCancellationRestoresStock(t *testing.T) {
	f := setupProductCRM(t)
	orderService := NewProductOrderService(f.querier)
	productID := f.createBook(t, "Buku Order Manual")
	require.NoError(t, f.service.AddStock(models.StockEntryRequest{
		ProdukID: productID, Tipe: "stok_awal", Qty: 5,
	}, f.adminID))

	orderID, err := orderService.Create(models.CreateProductOrderRequest{
		Tanggal:     "2026-09-28",
		NamaPembeli: "Fulan",
		NoWA:        "081234567890",
		Sumber:      "whatsapp",
		Items: []models.CreateProductOrderItemRequest{
			{ProdukID: productID, Qty: 3},
		},
	}, f.adminID)
	require.NoError(t, err)
	require.NotZero(t, orderID)

	stock, err := f.querier.CurrentProdukStock(t.Context(), productID, sql.NullInt64{})
	require.NoError(t, err)
	require.Equal(t, int64(2), stock)

	orders, err := orderService.List(10)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, "Fulan", orders[0].NamaPembeli)
	require.Equal(t, "ORD-000001", orders[0].OrderNo)
	require.Len(t, orders[0].Items, 1)
	require.Equal(t, int64(3), orders[0].Items[0].Qty)

	owned, err := f.service.ListMahasantriProducts(f.santriID)
	require.NoError(t, err)
	require.Empty(t, owned)

	require.NoError(t, orderService.Cancel(orderID, f.adminID))
	stock, err = f.querier.CurrentProdukStock(t.Context(), productID, sql.NullInt64{})
	require.NoError(t, err)
	require.Equal(t, int64(5), stock)

	orders, err = orderService.List(10)
	require.NoError(t, err)
	require.Equal(t, "dibatalkan", orders[0].Status)
	require.Error(t, orderService.Cancel(orderID, f.adminID))
}

func TestManualProductOrderRejectsInsufficientStockAtomically(t *testing.T) {
	f := setupProductCRM(t)
	orderService := NewProductOrderService(f.querier)
	productID := f.createBook(t, "Buku Stok Terbatas")
	require.NoError(t, f.service.AddStock(models.StockEntryRequest{
		ProdukID: productID, Tipe: "stok_awal", Qty: 2,
	}, f.adminID))

	_, err := orderService.Create(models.CreateProductOrderRequest{
		Tanggal:     "2026-09-28",
		NamaPembeli: "Pembeli",
		Sumber:      "instagram",
		Items: []models.CreateProductOrderItemRequest{
			{ProdukID: productID, Qty: 3},
		},
	}, f.adminID)
	require.ErrorIs(t, err, ErrStokOrderTidakCukup)

	stock, stockErr := f.querier.CurrentProdukStock(t.Context(), productID, sql.NullInt64{})
	require.NoError(t, stockErr)
	require.Equal(t, int64(2), stock)

	orders, listErr := orderService.List(10)
	require.NoError(t, listErr)
	require.Empty(t, orders)
}

func TestManualProgramOrderDoesNotChangeStock(t *testing.T) {
	f := setupProductCRM(t)
	orderService := NewProductOrderService(f.querier)
	require.NoError(t, f.service.CreateProduct(models.CreateProductRequest{
		Nama: "Program Klinik Talaqqi", Kategori: "program",
	}))
	products, err := f.service.Products()
	require.NoError(t, err)
	var productID int64
	for _, product := range products {
		if product.Nama == "Program Klinik Talaqqi" {
			productID = product.ID
		}
	}
	require.NotZero(t, productID)

	_, err = orderService.Create(models.CreateProductOrderRequest{
		Tanggal:     "2026-09-28",
		NamaPembeli: "Peserta Umum",
		Sumber:      "offline",
		Items: []models.CreateProductOrderItemRequest{
			{ProdukID: productID, Qty: 2},
		},
	}, f.adminID)
	require.NoError(t, err)

	stock, err := f.querier.CurrentProdukStock(t.Context(), productID, sql.NullInt64{})
	require.NoError(t, err)
	require.Zero(t, stock)

	orders, err := orderService.List(10)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, "program", orders[0].Items[0].Kategori)
}

func TestManualProductOrderSnapshotsHarga(t *testing.T) {
	f := setupProductCRM(t)
	orderService := NewProductOrderService(f.querier)
	require.NoError(t, f.service.CreateProduct(models.CreateProductRequest{Nama: "Program Tahsin", Kategori: "program", Harga: 150000}))
	require.Error(t, f.service.CreateProduct(models.CreateProductRequest{Nama: "Minus", Kategori: "program", Harga: -1}))
	products, err := f.service.Products()
	require.NoError(t, err)
	var productID int64
	for _, p := range products {
		if p.Nama == "Program Tahsin" {
			productID = p.ID
			require.EqualValues(t, 150000, p.Harga)
		}
	}
	require.NotZero(t, productID)

	diskon := int64(100000)
	_, err = orderService.Create(models.CreateProductOrderRequest{
		Tanggal: "2026-09-28", NamaPembeli: "Fulan", Sumber: "whatsapp",
		Items: []models.CreateProductOrderItemRequest{
			{ProdukID: productID, Qty: 2},
			{ProdukID: productID, Qty: 1, HargaSatuan: &diskon},
		},
	}, f.adminID)
	require.NoError(t, err)

	// Changing the product price later must not change the order's value.
	require.NoError(t, f.service.UpdateProduct(productID, models.UpdateProductRequest{Nama: "Program Tahsin", Kategori: "program", IsAktif: true, Harga: 200000}))

	orders, err := orderService.List(10)
	require.NoError(t, err)
	require.Len(t, orders[0].Items, 2)
	require.EqualValues(t, 150000, orders[0].Items[0].HargaSatuan)
	require.EqualValues(t, 300000, orders[0].Items[0].Subtotal)
	require.EqualValues(t, 100000, orders[0].Items[1].HargaSatuan)
	require.EqualValues(t, 400000, orders[0].Total)

	negatif := int64(-5)
	_, err = orderService.Create(models.CreateProductOrderRequest{
		Tanggal: "2026-09-28", NamaPembeli: "Fulan", Sumber: "whatsapp",
		Items: []models.CreateProductOrderItemRequest{{ProdukID: productID, Qty: 1, HargaSatuan: &negatif}},
	}, f.adminID)
	require.ErrorIs(t, err, ErrHargaProdukNegatif)
}
