package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type SantriService struct {
	querier *queries.Querier
	engine  *KelasEngineService
}

func NewSantriService(querier *queries.Querier, engine *KelasEngineService) *SantriService {
	return &SantriService{querier: querier, engine: engine}
}

func (s *SantriService) GenerateIDMahasantri(angkatan string, santriID int64, tanggalDaftar time.Time) string {
	noUrut := fmt.Sprintf("%04d", santriID)
	bulanTahun := tanggalDaftar.Format("012006")
	return fmt.Sprintf("MHS.%s.%s.%s", angkatan, noUrut, bulanTahun)
}

func (s *SantriService) Create(req models.CreateSantriRequest, createdBy int64) (*models.SantriResponse, error) {
	now := time.Now()

	var tanggalDaftar sql.NullTime
	if req.TanggalDaftar != "" {
		t, err := time.Parse("2006-01-02", req.TanggalDaftar)
		if err == nil {
			tanggalDaftar = sql.NullTime{Time: t, Valid: true}
		}
	}

	tipe := s.engine.HitungTipe(req.KelasKode)
	frekuensi := s.engine.HitungFrekuensi(req.KelasKode)

	result, err := s.querier.CreateSantri(context.Background(), queries.CreateSantriParams{
		IDMahasantri:  "",
		KelasKode:     req.KelasKode,
		Nama:          req.Nama,
		JenisKelamin:  req.JenisKelamin,
		Nominal:       req.Nominal,
		TanggalDaftar: tanggalDaftar,
		Angkatan:      req.Angkatan,
		Usia:          sql.NullInt64{Int64: req.Usia, Valid: req.Usia > 0},
		Domisili:      req.Domisili,
		NoWa:          req.NoWA,
		Tipe:          tipe,
		Frekuensi:     frekuensi,
		IsLengkap:     0,
		KelasID:       sql.NullInt64{Valid: false},
		Status:        "aktif",
		CreatedBy:     sql.NullInt64{Int64: createdBy, Valid: true},
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	if tanggalDaftar.Valid {
		idMahasantri := s.GenerateIDMahasantri(req.Angkatan, id, tanggalDaftar.Time)
		_ = s.querier.UpdateSantriIdMahasantri(context.Background(), queries.UpdateSantriIdMahasantriParams{
			IDMahasantri: idMahasantri,
			ID:           id,
		})
	}

	santri, err := s.querier.GetSantriByID(context.Background(), id)
	if err != nil {
		return nil, err
	}

	if err := s.engine.ProcessSantri(context.Background(), &santri); err != nil {
		return nil, err
	}

	santri, _ = s.querier.GetSantriByID(context.Background(), id)
	resp := santri.ToResponse()
	return &resp, nil
}

func (s *SantriService) GetByID(id int64) (*models.SantriResponse, error) {
	santri, err := s.querier.GetSantriByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	resp := santri.ToResponse()
	return &resp, nil
}

func (s *SantriService) List(params models.SantriListParams) (*models.SantriListResponse, error) {
	ctx := context.Background()

	var kelasID interface{}
	if params.KelasID > 0 {
		kelasID = params.KelasID
	}

	total, err := s.querier.CountSantri(ctx, queries.CountSantriParams{
		Angkatan: params.Angkatan,
		Level:    params.Level,
		Tipe:     params.Tipe,
		Jadwal:   params.Jadwal,
		Gender:   params.Gender,
		Status:   params.Status,
		KelasID:  kelasID,
		Lengkap:  params.Lengkap,
		Search:   params.Search,
	})
	if err != nil {
		return nil, err
	}

	list, err := s.querier.ListSantri(ctx, queries.ListSantriParams{
		Angkatan: params.Angkatan,
		Level:    params.Level,
		Tipe:     params.Tipe,
		Jadwal:   params.Jadwal,
		Gender:   params.Gender,
		Status:   params.Status,
		KelasID:  kelasID,
		Lengkap:  params.Lengkap,
		Search:   params.Search,
		Off:      params.Offset,
		Lim:      params.Limit,
	})
	if err != nil {
		return nil, err
	}

	santriList := make([]models.SantriResponse, len(list))
	for i, s := range list {
		santriList[i] = s.ToResponse()
	}

	return &models.SantriListResponse{
		Data:  santriList,
		Total: total,
	}, nil
}

func (s *SantriService) UpdateByCS(id int64, req models.UpdateSantriCSRequest) error {
	now := time.Now()
	var tanggalDaftar sql.NullTime
	if req.TanggalDaftar != "" {
		t, err := time.Parse("2006-01-02", req.TanggalDaftar)
		if err == nil {
			tanggalDaftar = sql.NullTime{Time: t, Valid: true}
		}
	}

	if err := s.querier.UpdateSantriCS(context.Background(), queries.UpdateSantriCSParams{
		KelasKode:     req.KelasKode,
		Nama:          req.Nama,
		JenisKelamin:  req.JenisKelamin,
		Nominal:       req.Nominal,
		TanggalDaftar: tanggalDaftar,
		Angkatan:      req.Angkatan,
		Usia:          sql.NullInt64{Int64: req.Usia, Valid: req.Usia > 0},
		Domisili:      req.Domisili,
		NoWa:          req.NoWA,
		UpdatedAt:     now,
		ID:            id,
	}); err != nil {
		return err
	}

	// Re-run the kelas engine: changing kelas_kode/gender/angkatan can move the
	// santri to a different class (the class key includes gender & angkatan).
	santri, err := s.querier.GetSantriByID(context.Background(), id)
	if err != nil {
		return err
	}
	return s.engine.ProcessSantri(context.Background(), &santri)
}

const maxVoiceNoteSize = 20 * 1024 * 1024

var voiceNoteExtensions = map[string]string{
	"audio/mpeg":      ".mp3",
	"audio/ogg":       ".ogg",
	"application/ogg": ".ogg",
	"audio/wav":       ".wav",
	"audio/wave":      ".wav",
	"audio/x-wav":     ".wav",
	"audio/mp4":       ".m4a",
	"audio/webm":      ".webm",
}

func (s *SantriService) UpdateVoiceNote(id int64, description string, file io.ReadSeeker, fileSize int64) (*models.SantriResponse, error) {
	if len(description) > 2000 {
		return nil, errors.New("keterangan voice note maksimal 2000 karakter")
	}
	santri, err := s.querier.GetSantriByID(context.Background(), id)
	if err != nil {
		return nil, err
	}

	voiceNoteURL := santri.VoiceNoteUrl
	newVoiceNotePath := ""
	if file != nil {
		if fileSize > maxVoiceNoteSize {
			return nil, errors.New("ukuran voice note maksimal 20 MB")
		}

		buffer := make([]byte, 512)
		read, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, errors.New("gagal membaca voice note")
		}
		if read == 0 {
			return nil, errors.New("voice note kosong")
		}
		extension, allowed := voiceNoteExtensions[http.DetectContentType(buffer[:read])]
		if !allowed {
			return nil, errors.New("format voice note harus MP3, OGG, WAV, M4A, atau WebM")
		}

		if err := os.MkdirAll("data/voice-notes", 0755); err != nil {
			return nil, err
		}
		filename := fmt.Sprintf("santri-%d-%d%s", id, time.Now().UnixNano(), extension)
		path := filepath.Join("data", "voice-notes", filename)
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		destination, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		_, copyErr := io.Copy(destination, file)
		closeErr := destination.Close()
		if copyErr != nil || closeErr != nil {
			_ = os.Remove(path)
			if copyErr != nil {
				return nil, copyErr
			}
			return nil, closeErr
		}
		voiceNoteURL = filepath.ToSlash(filepath.Join("voice-notes", filename))
		newVoiceNotePath = path
	}

	if err := s.querier.UpdateSantriVoiceNote(context.Background(), queries.UpdateSantriVoiceNoteParams{
		VoiceNoteUrl: voiceNoteURL,
		KeteranganVn: description,
		UpdatedAt:    time.Now(),
		ID:           id,
	}); err != nil {
		if newVoiceNotePath != "" {
			_ = os.Remove(newVoiceNotePath)
		}
		return nil, err
	}
	if santri.VoiceNoteUrl != "" && santri.VoiceNoteUrl != voiceNoteURL {
		if err := os.Remove(filepath.Join("data", filepath.FromSlash(santri.VoiceNoteUrl))); err != nil && !os.IsNotExist(err) {
			slog.Warn("failed to remove replaced voice note", "santri_id", id, "path", santri.VoiceNoteUrl, "error", err)
		}
	}

	updated, err := s.querier.GetSantriByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	response := updated.ToResponse()
	return &response, nil
}

func (s *SantriService) VoiceNotePath(id int64) (string, error) {
	santri, err := s.querier.GetSantriByID(context.Background(), id)
	if err != nil {
		return "", err
	}
	if santri.VoiceNoteUrl == "" {
		return "", errors.New("voice note belum tersedia")
	}
	path := filepath.Clean(filepath.FromSlash(santri.VoiceNoteUrl))
	if path == "." || strings.HasPrefix(path, "..") || !strings.HasPrefix(filepath.ToSlash(path), "voice-notes/") {
		return "", errors.New("lokasi voice note tidak valid")
	}
	fullPath := filepath.Join("data", path)
	if _, err := os.Stat(fullPath); err != nil {
		return "", err
	}
	return fullPath, nil
}

func (s *SantriService) UpdateByAdminKelas(id int64, req models.UpdateSantriAdminKelasRequest) error {
	now := time.Now()

	var tanggalVn sql.NullTime
	if req.TanggalVn != "" {
		t, err := time.Parse("2006-01-02", req.TanggalVn)
		if err == nil {
			tanggalVn = sql.NullTime{Time: t, Valid: true}
		}
	}

	var mulaiBelajar sql.NullTime
	if req.MulaiBelajar != "" {
		t, err := time.Parse("2006-01-02", req.MulaiBelajar)
		if err == nil {
			mulaiBelajar = sql.NullTime{Time: t, Valid: true}
		}
	}

	if err := s.querier.UpdateSantriAdminKelas(context.Background(), queries.UpdateSantriAdminKelasParams{
		Fu:           req.Fu,
		TanggalVn:    tanggalVn,
		HasilVn:      req.HasilVn,
		MasukGrup:    req.MasukGrup,
		MulaiBelajar: mulaiBelajar,
		Jumlah:       sql.NullInt64{Int64: req.Jumlah, Valid: req.Jumlah > 0},
		Level:        req.Level,
		Jadwal:       req.Jadwal,
		Guru:         req.Guru,
		Tipe:         "",
		Frekuensi:    "",
		IsLengkap:    0,
		KelasID:      sql.NullInt64{Valid: false},
		UpdatedAt:    now,
		ID:           id,
	}); err != nil {
		return err
	}

	santri, err := s.querier.GetSantriByID(context.Background(), id)
	if err != nil {
		return err
	}

	return s.engine.ProcessSantri(context.Background(), &santri)
}

func (s *SantriService) UpdateByKeuangan(id int64, req models.UpdateSantriKeuanganRequest) error {
	status := "aktif"
	if req.KeteranganTidakLanjut != "" {
		status = "tidak_lanjut"
	}

	// The santri stays linked to their class (still listed under "Tidak Lanjut"
	// in the class detail); the class occupancy count excludes them because the
	// kelas queries count only active santri.
	return s.querier.UpdateSantriKeuangan(context.Background(), queries.UpdateSantriKeuanganParams{
		InfaqTerakhir:         req.InfaqTerakhir,
		KeteranganTidakLanjut: req.KeteranganTidakLanjut,
		Status:                status,
		UpdatedAt:             time.Now(),
		ID:                    id,
	})
}

func (s *SantriService) ListPerluDilengkapi() ([]models.SantriResponse, error) {
	list, err := s.querier.GetPerluDilengkapi(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]models.SantriResponse, len(list))
	for i, s := range list {
		result[i] = s.ToResponse()
	}
	return result, nil
}

func (s *SantriService) PindahkanKelas(santriID, kelasTujuanID int64) error {
	return s.engine.PindahkanSantri(context.Background(), santriID, kelasTujuanID)
}

func (s *SantriService) CountByStatus() ([]queries.CountSantriByStatusRow, error) {
	return s.querier.CountSantriByStatus(context.Background())
}

func (s *SantriService) CountByGender() ([]queries.CountSantriByGenderRow, error) {
	return s.querier.CountSantriByGender(context.Background())
}

func (s *SantriService) CountByLevel() ([]queries.CountSantriByLevelRow, error) {
	return s.querier.CountSantriByLevel(context.Background())
}

func (s *SantriService) CountByTipe() ([]queries.CountSantriByTipeRow, error) {
	return s.querier.CountSantriByTipe(context.Background())
}

func (s *SantriService) SumNominalByAngkatan() ([]queries.SumNominalByAngkatanRow, error) {
	return s.querier.SumNominalByAngkatan(context.Background())
}

func (s *SantriService) GetDashboardStats() (*models.DashboardStats, error) {
	ctx := context.Background()
	stats := &models.DashboardStats{}

	kelasCount, err := s.querier.CountKelasTotal(ctx)
	if err == nil {
		stats.TotalKelas = kelasCount
	}

	stats.TotalSantri, _ = s.querier.CountSantri(ctx, queries.CountSantriParams{
		Angkatan: "", Level: "", Tipe: "", Jadwal: "",
		Gender: "", Status: "aktif", KelasID: nil, Lengkap: int64(-1), Search: "",
	})

	lengkapCount, _ := s.querier.CountSantri(ctx, queries.CountSantriParams{
		Angkatan: "", Level: "", Tipe: "", Jadwal: "",
		Gender: "", Status: "aktif", KelasID: nil, Lengkap: int64(1), Search: "",
	})
	stats.SantriLengkap = lengkapCount

	stats.SantriPerluLengkap = stats.TotalSantri - lengkapCount

	tidakLanjutCount, _ := s.querier.CountSantri(ctx, queries.CountSantriParams{
		Angkatan: "", Level: "", Tipe: "", Jadwal: "",
		Gender: "", Status: "tidak_lanjut", KelasID: nil, Lengkap: int64(-1), Search: "",
	})
	stats.SantriTidakLanjut = tidakLanjutCount

	genderRows, err := s.querier.CountSantriByGender(ctx)
	if err == nil {
		for _, g := range genderRows {
			if g.JenisKelamin == "L" {
				stats.SantriLaki = g.Total
			} else if g.JenisKelamin == "P" {
				stats.SantriPerempuan = g.Total
			}
		}
	}

	levelData, _ := s.querier.CountSantriByLevel(ctx)
	for _, l := range levelData {
		stats.PerLevel = append(stats.PerLevel, models.LevelStat{Level: l.Level, Total: l.Total})
	}

	tipeData, _ := s.querier.CountSantriByTipe(ctx)
	for _, t := range tipeData {
		stats.PerTipe = append(stats.PerTipe, models.TipeStat{Tipe: t.Tipe, Total: t.Total})
	}

	nominalData, _ := s.querier.SumNominalByAngkatan(ctx)
	for _, n := range nominalData {
		totalNom := float64(0)
		if n.TotalNominal.Valid {
			totalNom = n.TotalNominal.Float64
		}
		stats.NominalPerAngkatan = append(stats.NominalPerAngkatan, models.NominalAngkatan{Angkatan: n.Angkatan, TotalNominal: totalNom})
	}

	return stats, nil
}
