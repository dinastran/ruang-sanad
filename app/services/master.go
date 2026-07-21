package services

import (
	"context"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type MasterService struct {
	querier *queries.Querier
}

func NewMasterService(querier *queries.Querier) *MasterService {
	return &MasterService{querier: querier}
}

// ---- Angkatan ----

func (s *MasterService) ListAngkatan() ([]models.AngkatanResponse, error) {
	list, err := s.querier.ListAngkatan(context.Background())
	out := make([]models.AngkatanResponse, 0, len(list))
	for _, a := range list {
		out = append(out, models.AngkatanResponse{
			ID:         a.ID,
			Kode:       a.Kode,
			Keterangan: a.Keterangan,
			IsAktif:    a.IsAktif == 1,
		})
	}
	return out, err
}

func (s *MasterService) CreateAngkatan(req models.CreateAngkatanRequest) error {
	return s.querier.CreateAngkatan(context.Background(), queries.CreateAngkatanParams{
		Kode:       req.Kode,
		Keterangan: req.Keterangan,
		IsAktif:    1,
	})
}

func (s *MasterService) UpdateAngkatan(id int64, req models.UpdateAngkatanRequest) error {
	return s.querier.UpdateAngkatan(context.Background(), queries.UpdateAngkatanParams{
		Keterangan: req.Keterangan,
		IsAktif:    1,
		ID:         id,
	})
}

func (s *MasterService) DeleteAngkatan(id int64) error {
	return s.querier.DeleteAngkatan(context.Background(), id)
}

// ---- Level ----

func (s *MasterService) ListLevel() ([]models.LevelResponse, error) {
	list, err := s.querier.ListLevel(context.Background())
	out := make([]models.LevelResponse, 0, len(list))
	for _, l := range list {
		out = append(out, models.LevelResponse{
			ID:     l.ID,
			Kode:   l.Kode,
			Nama:   l.Nama,
			Urutan: l.Urutan,
		})
	}
	return out, err
}

func (s *MasterService) CreateLevel(req models.CreateLevelRequest) error {
	return s.querier.CreateLevel(context.Background(), queries.CreateLevelParams{
		Kode:   req.Kode,
		Nama:   req.Nama,
		Urutan: req.Urutan,
	})
}

func (s *MasterService) UpdateLevel(id int64, req models.UpdateLevelRequest) error {
	return s.querier.UpdateLevel(context.Background(), queries.UpdateLevelParams{
		Nama:   req.Nama,
		Urutan: req.Urutan,
		ID:     id,
	})
}

func (s *MasterService) DeleteLevel(id int64) error {
	return s.querier.DeleteLevel(context.Background(), id)
}

// ---- Jadwal ----

func (s *MasterService) ListJadwal() ([]models.JadwalResponse, error) {
	list, err := s.querier.ListJadwal(context.Background())
	out := make([]models.JadwalResponse, 0, len(list))
	for _, j := range list {
		out = append(out, models.JadwalResponse{
			ID:   j.ID,
			Nama: j.Nama,
		})
	}
	return out, err
}

func (s *MasterService) CreateJadwal(req models.CreateJadwalRequest) error {
	return s.querier.CreateJadwal(context.Background(), req.Nama)
}

func (s *MasterService) UpdateJadwal(id int64, req models.UpdateJadwalRequest) error {
	return s.querier.UpdateJadwal(context.Background(), queries.UpdateJadwalParams{
		Nama: req.Nama,
		ID:   id,
	})
}

func (s *MasterService) DeleteJadwal(id int64) error {
	return s.querier.DeleteJadwal(context.Background(), id)
}

// ---- Guru ----

// ListGuru returns only active guru (used by santri/kelas dropdowns).
func (s *MasterService) ListGuru() ([]models.GuruResponse, error) {
	return s.mapGuru(s.querier.ListGuru(context.Background()))
}

// ListGuruAll returns every guru (used by the master management page).
func (s *MasterService) ListGuruAll() ([]models.GuruResponse, error) {
	return s.mapGuru(s.querier.ListGuruAll(context.Background()))
}

func (s *MasterService) mapGuru(list []queries.Guru, err error) ([]models.GuruResponse, error) {
	out := make([]models.GuruResponse, 0, len(list))
	for _, g := range list {
		out = append(out, models.GuruResponse{
			ID:           g.ID,
			Nama:         g.Nama,
			JenisKelamin: g.JenisKelamin,
			IsAktif:      g.IsAktif == 1,
		})
	}
	return out, err
}

func (s *MasterService) CreateGuru(req models.CreateGuruRequest) error {
	return s.querier.CreateGuru(context.Background(), queries.CreateGuruParams{
		Nama:         req.Nama,
		JenisKelamin: req.JenisKelamin,
		IsAktif:      1,
	})
}

func (s *MasterService) UpdateGuru(id int64, req models.UpdateGuruRequest) error {
	return s.querier.UpdateGuru(context.Background(), queries.UpdateGuruParams{
		Nama:         req.Nama,
		JenisKelamin: req.JenisKelamin,
		IsAktif:      1,
		ID:           id,
	})
}

func (s *MasterService) DeleteGuru(id int64) error {
	return s.querier.DeleteGuru(context.Background(), id)
}

// ---- Kode Kelas ----

func (s *MasterService) ListKodeKelas() ([]models.KodeKelasResponse, error) {
	list, err := s.querier.ListKodeKelas(context.Background())
	out := make([]models.KodeKelasResponse, 0, len(list))
	for _, k := range list {
		out = append(out, models.KodeKelasResponse{
			ID:        k.ID,
			Kode:      k.Kode,
			Tipe:      k.Tipe,
			Frekuensi: k.Frekuensi,
			Urutan:    k.Urutan,
		})
	}
	return out, err
}

func (s *MasterService) CreateKodeKelas(req models.CreateKodeKelasRequest) error {
	return s.querier.CreateKodeKelas(context.Background(), queries.CreateKodeKelasParams{
		Kode:      req.Kode,
		Tipe:      req.Tipe,
		Frekuensi: req.Frekuensi,
		Urutan:    req.Urutan,
	})
}

func (s *MasterService) UpdateKodeKelas(id int64, req models.UpdateKodeKelasRequest) error {
	return s.querier.UpdateKodeKelas(context.Background(), queries.UpdateKodeKelasParams{
		Tipe:      req.Tipe,
		Frekuensi: req.Frekuensi,
		Urutan:    req.Urutan,
		ID:        id,
	})
}

func (s *MasterService) DeleteKodeKelas(id int64) error {
	return s.querier.DeleteKodeKelas(context.Background(), id)
}
