package services

import (
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func TestProdukVarianValidation(t *testing.T) {
	db, q := setupSantriDatabase(t)
	s := NewProductCRMService(q)
	ctx := t.Context()
	res, err := db.Exec(`INSERT INTO produk (nama, kategori, harga) VALUES ('Tahsin Reguler', 'program', 150000)`)
	require.NoError(t, err)
	produkID, _ := res.LastInsertId()

	require.Error(t, s.CreateVarian(ctx, produkID, models.ProdukVarianRequest{Nama: "  ", Harga: 1}), "nama kosong")
	require.Error(t, s.CreateVarian(ctx, produkID, models.ProdukVarianRequest{Nama: "1x", Harga: -1}), "harga negatif")
	require.Error(t, s.CreateVarian(ctx, produkID, models.ProdukVarianRequest{Nama: "3x", Harga: 1, PaketKode: "3x"}), "paket di luar mapping")
	require.Error(t, s.CreateVarian(ctx, 9999, models.ProdukVarianRequest{Nama: "x", Harga: 1}), "produk tidak ada")

	require.NoError(t, s.CreateVarian(ctx, produkID, models.ProdukVarianRequest{Nama: "2x / pekan", Harga: 350000, PaketKode: "2x", Urutan: 2, IsAktif: true}))
	require.NoError(t, s.CreateVarian(ctx, produkID, models.ProdukVarianRequest{Nama: "1x / pekan", Harga: 200000, PaketKode: "1x", Urutan: 1, IsAktif: true}))

	list, err := s.ListVarian(ctx, produkID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, "1x / pekan", list[0].Nama, "urut berdasarkan urutan")

	require.NoError(t, s.UpdateVarian(ctx, list[0].ID, models.ProdukVarianRequest{Nama: "1x / pekan", Harga: 210000, PaketKode: "1x", Urutan: 1, IsAktif: false}))
	list, err = s.ListVarian(ctx, produkID)
	require.NoError(t, err)
	require.EqualValues(t, 210000, list[0].Harga)
	require.False(t, list[0].IsAktif)
	require.Error(t, s.UpdateVarian(ctx, 9999, models.ProdukVarianRequest{Nama: "x", Harga: 1}))
}
