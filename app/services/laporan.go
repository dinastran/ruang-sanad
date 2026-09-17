package services

import (
	"context"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type LaporanService struct {
	querier       *queries.Querier
	santriService *SantriService
	kelasService  *KelasService
}

func NewLaporanService(querier *queries.Querier, santriService *SantriService, kelasService *KelasService) *LaporanService {
	return &LaporanService{
		querier:       querier,
		santriService: santriService,
		kelasService:  kelasService,
	}
}

func (s *LaporanService) GetDashboardStats() (*models.DashboardStats, error) {
	totalSantri, err := s.querier.CountSantri(context.Background(), queries.CountSantriParams{
		AngkatanPendaftaran: "",
		AngkatanKelas:       "",
		Level:               "",
		Tipe:                "",
		Jadwal:              "",
		Gender:              "",
		Status:              "aktif",
		KelasID:             nil,
		Lengkap:             int64(-1),
		IDBermasalah:        int64(0),
		Search:              "",
	})
	if err != nil {
		return nil, err
	}

	// NOTE: all string filters must be set to "" explicitly. CountSantriParams
	// fields are interface{}, so an unset field is nil → SQL NULL, and the
	// "@angkatan = '' OR ..." guards become NULL (false) and exclude every row.
	santriLengkap, err := s.querier.CountSantri(context.Background(), queries.CountSantriParams{
		AngkatanPendaftaran: "", AngkatanKelas: "", Level: "", Tipe: "", Jadwal: "",
		Gender: "", Status: "aktif", KelasID: nil, Lengkap: int64(1), IDBermasalah: int64(0), Search: "",
	})
	if err != nil {
		return nil, err
	}

	santriPerluLengkap, err := s.querier.CountSantri(context.Background(), queries.CountSantriParams{
		AngkatanPendaftaran: "", AngkatanKelas: "", Level: "", Tipe: "", Jadwal: "",
		Gender: "", Status: "aktif", KelasID: nil, Lengkap: int64(0), IDBermasalah: int64(0), Search: "",
	})
	if err != nil {
		return nil, err
	}

	totalKelas, err := s.kelasService.CountTotal()
	if err != nil {
		return nil, err
	}

	santriTidakLanjut := int64(0)
	statusRows, err := s.santriService.CountByStatus()
	if err == nil {
		for _, r := range statusRows {
			if r.Status == "tidak_lanjut" {
				santriTidakLanjut = r.Total
			}
		}
	}

	santriLaki := int64(0)
	santriPerempuan := int64(0)
	genderRows, err := s.santriService.CountByGender()
	if err == nil {
		for _, g := range genderRows {
			if g.JenisKelamin == "L" {
				santriLaki = g.Total
			} else if g.JenisKelamin == "P" {
				santriPerempuan = g.Total
			}
		}
	}

	perLevel, err := s.santriService.CountByLevel()
	if err != nil {
		return nil, err
	}
	// Map level kode (e.g. "01") -> nama (e.g. "PT") so the distribution card
	// shows human-readable names instead of codes.
	levelNamaByKode := map[string]string{}
	if levels, err := s.querier.ListLevel(context.Background()); err == nil {
		for _, lv := range levels {
			levelNamaByKode[lv.Kode] = lv.Nama
		}
	}
	levelStats := make([]models.LevelStat, len(perLevel))
	for i, l := range perLevel {
		nama := levelNamaByKode[l.Level]
		if nama == "" {
			nama = l.Level
		}
		levelStats[i] = models.LevelStat{Level: nama, Total: l.Total}
	}

	perTipe, err := s.santriService.CountByTipe()
	if err != nil {
		return nil, err
	}
	tipeStats := make([]models.TipeStat, len(perTipe))
	for i, t := range perTipe {
		tipeStats[i] = models.TipeStat{Tipe: t.Tipe, Total: t.Total}
	}

	nominalRows, err := s.santriService.SumNominalByAngkatan()
	if err != nil {
		return nil, err
	}
	nominalStats := make([]models.NominalAngkatan, len(nominalRows))
	totalNominal := float64(0)
	for i, n := range nominalRows {
		if n.TotalNominal.Valid {
			nominalStats[i] = models.NominalAngkatan{
				Angkatan:     n.Angkatan,
				TotalNominal: n.TotalNominal.Float64,
			}
			totalNominal += n.TotalNominal.Float64
		}
	}

	return &models.DashboardStats{
		TotalSantri:        totalSantri,
		SantriLengkap:      santriLengkap,
		SantriPerluLengkap: santriPerluLengkap,
		SantriTidakLanjut:  santriTidakLanjut,
		TotalKelas:         totalKelas,
		TotalNominal:       totalNominal,
		SantriLaki:         santriLaki,
		SantriPerempuan:    santriPerempuan,
		PerLevel:           levelStats,
		PerTipe:            tipeStats,
		NominalPerAngkatan: nominalStats,
	}, nil
}

func (s *LaporanService) GetLaporanKeuangan() (*models.LaporanKeuangan, error) {
	nominalRows, err := s.santriService.SumNominalByAngkatan()
	if err != nil {
		return nil, err
	}
	nominalStats := make([]models.NominalAngkatan, len(nominalRows))
	for i, n := range nominalRows {
		val := float64(0)
		if n.TotalNominal.Valid {
			val = n.TotalNominal.Float64
		}
		nominalStats[i] = models.NominalAngkatan{
			Angkatan:     n.Angkatan,
			TotalNominal: val,
		}
	}

	statusRows, err := s.santriService.CountByStatus()
	if err != nil {
		return nil, err
	}
	tidakLanjut := int64(0)
	for _, r := range statusRows {
		if r.Status == "tidak_lanjut" {
			tidakLanjut = r.Total
		}
	}

	return &models.LaporanKeuangan{
		PerAngkatan: nominalStats,
		TidakLanjut: tidakLanjut,
	}, nil
}

// ListTidakLanjut returns the santri whose payment follow-up has stopped
// (status = tidak_lanjut), including their keterangan for the report table.
func (s *LaporanService) ListTidakLanjut() ([]models.SantriResponse, error) {
	res, err := s.santriService.List(models.SantriListParams{
		Status: "tidak_lanjut", Lengkap: -1, Limit: 1000, Offset: 0,
	})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// ListInfaqRekap returns active santri with their nominal & infaq_terakhir for
// the infaq recap table.
func (s *LaporanService) ListInfaqRekap() ([]models.SantriResponse, error) {
	res, err := s.santriService.List(models.SantriListParams{
		Status: "aktif", Lengkap: -1, Limit: 1000, Offset: 0,
	})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}
