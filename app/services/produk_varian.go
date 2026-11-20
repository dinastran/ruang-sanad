package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var ErrVarianTidakAda = errors.New("varian tidak ditemukan")

func toProdukVarian(v queries.ProdukVarian) models.ProdukVarian {
	return models.ProdukVarian{
		ID: v.ID, ProdukID: v.ProdukID, Nama: v.Nama, Harga: v.Harga,
		PaketKode: v.PaketKode, Urutan: v.Urutan, IsAktif: v.IsAktif == 1,
	}
}

func (s *ProductCRMService) validateVarian(ctx context.Context, req *models.ProdukVarianRequest) error {
	req.Nama = truncateRunes(strings.TrimSpace(req.Nama), 80)
	req.PaketKode = strings.ToLower(strings.TrimSpace(req.PaketKode))
	if req.Nama == "" {
		return errors.New("nama varian wajib diisi")
	}
	if req.Harga < 0 {
		return errors.New("harga varian tidak boleh negatif")
	}
	if req.PaketKode != "" {
		settings, err := s.querier.GetCRMSettings(ctx)
		if err != nil {
			return err
		}
		if paketToKelasKode(settings, req.PaketKode) == "" {
			return errors.New("kode paket tidak dikenal di pengaturan CRM")
		}
	}
	return nil
}

func (s *ProductCRMService) ListVarian(ctx context.Context, produkID int64) ([]models.ProdukVarian, error) {
	rows, err := s.querier.ListProdukVarianByProduk(ctx, produkID)
	if err != nil {
		return nil, err
	}
	out := make([]models.ProdukVarian, 0, len(rows))
	for _, r := range rows {
		out = append(out, toProdukVarian(r))
	}
	return out, nil
}

// VarianByProduk groups every varian by produk id (Produk CRM page).
func (s *ProductCRMService) VarianByProduk(ctx context.Context) (map[int64][]models.ProdukVarian, error) {
	rows, err := s.querier.ListAllProdukVarian(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64][]models.ProdukVarian{}
	for _, r := range rows {
		out[r.ProdukID] = append(out[r.ProdukID], toProdukVarian(r))
	}
	return out, nil
}

func (s *ProductCRMService) CreateVarian(ctx context.Context, produkID int64, req models.ProdukVarianRequest) error {
	if err := s.validateVarian(ctx, &req); err != nil {
		return err
	}
	if _, err := s.querier.GetProduk(ctx, produkID); err != nil {
		return errors.New("produk tidak ditemukan")
	}
	now := time.Now().UTC()
	_, err := s.querier.CreateProdukVarian(ctx, queries.CreateProdukVarianParams{
		ProdukID: produkID, Nama: req.Nama, Harga: req.Harga, PaketKode: req.PaketKode,
		Urutan: req.Urutan, IsAktif: boolInt(req.IsAktif), CreatedAt: now, UpdatedAt: now,
	})
	return err
}

func (s *ProductCRMService) UpdateVarian(ctx context.Context, id int64, req models.ProdukVarianRequest) error {
	if err := s.validateVarian(ctx, &req); err != nil {
		return err
	}
	if _, err := s.querier.GetProdukVarian(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrVarianTidakAda
		}
		return err
	}
	return s.querier.UpdateProdukVarian(ctx, queries.UpdateProdukVarianParams{
		Nama: req.Nama, Harga: req.Harga, PaketKode: req.PaketKode, Urutan: req.Urutan,
		IsAktif: boolInt(req.IsAktif), UpdatedAt: time.Now().UTC(), ID: id,
	})
}
