package services

import (
	"context"
	"database/sql"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type KelasService struct {
	querier *queries.Querier
}

func NewKelasService(querier *queries.Querier) *KelasService {
	return &KelasService{querier: querier}
}

func (s *KelasService) ListByAngkatan(angkatan string) ([]models.KelasResponse, error) {
	list, err := s.querier.ListKelasByAngkatan(context.Background(), angkatan)
	if err != nil {
		return nil, err
	}
	result := make([]models.KelasResponse, len(list))
	for i, k := range list {
		result[i] = k.ToResponse()
	}
	return result, nil
}

func (s *KelasService) GetByID(id int64) (*models.KelasResponse, error) {
	k, err := s.querier.GetKelasByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	resp := k.ToResponse()
	return &resp, nil
}

func (s *KelasService) GetSantriByKelasID(kelasID int64) ([]models.SantriResponse, error) {
	list, err := s.querier.GetSantriByKelasID(context.Background(), sql.NullInt64{Int64: kelasID, Valid: true})
	if err != nil {
		return nil, err
	}
	result := make([]models.SantriResponse, len(list))
	for i, s := range list {
		result[i] = s.ToResponse()
	}
	return result, nil
}

func (s *KelasService) AssignGuru(kelasID, guruID int64) error {
	return s.querier.AssignGuru(context.Background(), queries.AssignGuruParams{
		GuruID: sql.NullInt64{Int64: guruID, Valid: true},
		ID:     kelasID,
	})
}

func (s *KelasService) SetAktif(kelasID int64, aktif bool) error {
	var v int64
	if aktif {
		v = 1
	}
	return s.querier.SetKelasAktif(context.Background(), queries.SetKelasAktifParams{
		IsAktif: v,
		ID:      kelasID,
	})
}

// Delete removes a class. Santri linked to it have their kelas_id set to NULL
// via the ON DELETE SET NULL foreign key, so they become unassigned.
func (s *KelasService) Delete(kelasID int64) error {
	return s.querier.DeleteKelas(context.Background(), kelasID)
}

func (s *KelasService) CountTotal() (int64, error) {
	return s.querier.CountKelasTotal(context.Background())
}

func (s *KelasService) CountByAngkatan() ([]queries.CountKelasByAngkatanRow, error) {
	return s.querier.CountKelasByAngkatan(context.Background())
}
