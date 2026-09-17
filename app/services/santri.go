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

func isValidIssuedIDMahasantri(id string) bool {
	if id != strings.TrimSpace(id) || !strings.HasPrefix(id, "MHS.") {
		return false
	}
	payload := strings.TrimPrefix(id, "MHS.")
	monthYearSeparator := strings.LastIndexByte(payload, '.')
	if monthYearSeparator < 0 {
		return false
	}
	monthYear := payload[monthYearSeparator+1:]
	payload = payload[:monthYearSeparator]
	sequenceSeparator := strings.LastIndexByte(payload, '.')
	if sequenceSeparator < 0 {
		return false
	}
	code := payload[:sequenceSeparator]
	sequence := payload[sequenceSeparator+1:]
	if code == "" || code != strings.TrimSpace(code) || len(sequence) < 4 || len(monthYear) != 6 {
		return false
	}
	for _, segment := range []string{sequence, monthYear} {
		for _, r := range segment {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	month := int(monthYear[0]-'0')*10 + int(monthYear[1]-'0')
	return month >= 1 && month <= 12
}

func isValidIssuedIDMahasantriForRow(id string, rowID int64) bool {
	if !isValidIssuedIDMahasantri(id) {
		return false
	}
	payload := strings.TrimPrefix(id, "MHS.")
	monthYearSeparator := strings.LastIndexByte(payload, '.')
	payload = payload[:monthYearSeparator]
	sequenceSeparator := strings.LastIndexByte(payload, '.')
	return payload[sequenceSeparator+1:] == fmt.Sprintf("%04d", rowID)
}

func validateMutableSantriInput(nama, jenisKelamin string) (string, string, error) {
	nama = strings.TrimSpace(nama)
	if nama == "" {
		return "", "", errors.New("nama wajib diisi")
	}
	jenisKelamin = strings.ToUpper(strings.TrimSpace(jenisKelamin))
	if jenisKelamin != "L" && jenisKelamin != "P" {
		return "", "", errors.New("jenis kelamin wajib L atau P")
	}
	return nama, jenisKelamin, nil
}

func (s *SantriService) validateSantriInput(ctx context.Context, querier *queries.Querier, nama, jenisKelamin, angkatan, tanggalDaftar string) (string, string, string, time.Time, error) {
	nama, jenisKelamin, err := validateMutableSantriInput(nama, jenisKelamin)
	if err != nil {
		return "", "", "", time.Time{}, err
	}

	angkatan = strings.TrimSpace(angkatan)
	if angkatan == "" {
		return "", "", "", time.Time{}, errors.New("angkatan wajib diisi")
	}
	if _, err := querier.GetAngkatanByKode(ctx, angkatan); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", "", time.Time{}, fmt.Errorf("angkatan %q tidak terdaftar di master angkatan", angkatan)
		}
		return "", "", "", time.Time{}, fmt.Errorf("gagal memvalidasi angkatan: %w", err)
	}

	tanggalDaftar = strings.TrimSpace(tanggalDaftar)
	if tanggalDaftar == "" {
		return "", "", "", time.Time{}, errors.New("tanggal daftar wajib diisi")
	}
	parsedDate, err := time.Parse(time.DateOnly, tanggalDaftar)
	if err != nil || parsedDate.Format(time.DateOnly) != tanggalDaftar {
		return "", "", "", time.Time{}, errors.New("tanggal daftar harus berformat YYYY-MM-DD")
	}

	return nama, jenisKelamin, angkatan, parsedDate, nil
}

func loadDuplicateIDMahasantri(ctx context.Context, querier *queries.Querier) (map[string]struct{}, error) {
	ids, err := querier.ListDuplicateIDMahasantri(ctx)
	if err != nil {
		return nil, err
	}
	duplicates := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		duplicates[id] = struct{}{}
	}
	return duplicates, nil
}

func idMahasantriNeedsReview(santri queries.Santri, duplicates map[string]struct{}) bool {
	_, duplicate := duplicates[santri.IDMahasantri]
	return !isValidIssuedIDMahasantriForRow(santri.IDMahasantri, santri.ID) || duplicate
}

func (s *SantriService) santriResponse(santri queries.Santri, duplicates map[string]struct{}) models.SantriResponse {
	response := santri.ToResponse()
	response.IDMahasantriTerbit = isValidIssuedIDMahasantriForRow(santri.IDMahasantri, santri.ID)
	response.IDMahasantriBermasalah = idMahasantriNeedsReview(santri, duplicates)
	return response
}

func (s *SantriService) Create(req models.CreateSantriRequest, createdBy int64) (*models.SantriResponse, error) {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txQuerier := s.querier.WithTx(tx)

	nama, jenisKelamin, angkatan, parsedDate, err := s.validateSantriInput(ctx, txQuerier, req.Nama, req.JenisKelamin, req.Angkatan, req.TanggalDaftar)
	if err != nil {
		return nil, err
	}
	req.Nama = nama
	req.JenisKelamin = jenisKelamin
	req.Angkatan = angkatan
	tanggalDaftar := sql.NullTime{Time: parsedDate, Valid: true}
	now := time.Now()
	tipe := s.engine.HitungTipe(req.KelasKode)
	frekuensi := s.engine.HitungFrekuensi(req.KelasKode)

	result, err := txQuerier.CreateSantri(ctx, queries.CreateSantriParams{
		IDMahasantri:  "",
		KelasKode:     req.KelasKode,
		Nama:          req.Nama,
		JenisKelamin:  req.JenisKelamin,
		Nominal:       req.Nominal,
		TanggalDaftar: tanggalDaftar,
		Angkatan:      req.Angkatan,
		AngkatanKelas: "",
		Usia:          sql.NullInt64{Int64: req.Usia, Valid: req.Usia > 0},
		Domisili:      req.Domisili,
		NoWa:          req.NoWA,
		Email:         req.Email,
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

	idMahasantri := s.GenerateIDMahasantri(req.Angkatan, id, tanggalDaftar.Time)
	if err := txQuerier.UpdateSantriIdMahasantri(ctx, queries.UpdateSantriIdMahasantriParams{
		IDMahasantri: idMahasantri,
		ID:           id,
	}); err != nil {
		return nil, err
	}
	santri, err := txQuerier.GetSantriByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.engine.WithQuerier(txQuerier).ProcessSantri(ctx, &santri); err != nil {
		return nil, err
	}
	santri, err = txQuerier.GetSantriByID(ctx, id)
	if err != nil {
		return nil, err
	}
	duplicates, err := loadDuplicateIDMahasantri(ctx, txQuerier)
	if err != nil {
		return nil, err
	}
	resp := s.santriResponse(santri, duplicates)
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *SantriService) GetByID(id int64) (*models.SantriResponse, error) {
	ctx := context.Background()
	santri, err := s.querier.GetSantriByID(ctx, id)
	if err != nil {
		return nil, err
	}
	duplicates, err := loadDuplicateIDMahasantri(ctx, s.querier)
	if err != nil {
		return nil, err
	}
	resp := s.santriResponse(santri, duplicates)
	return &resp, nil
}

func (s *SantriService) Delete(id int64) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	txQuerier := s.querier.WithTx(tx)
	santri, err := txQuerier.GetSantriByID(ctx, id)
	if err != nil {
		return err
	}
	if err := txQuerier.DeleteRiayahByTarget(ctx, queries.DeleteRiayahByTargetParams{
		TargetType: "santri",
		TargetID:   id,
	}); err != nil {
		return err
	}
	if err := txQuerier.DeleteSantri(ctx, id); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	if santri.VoiceNoteUrl != "" {
		path, pathErr := safeVoiceNotePath(santri.VoiceNoteUrl)
		if pathErr != nil {
			slog.Warn("failed to resolve deleted santri voice note", "santri_id", id, "path", santri.VoiceNoteUrl, "error", pathErr)
		} else if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			slog.Warn("failed to remove deleted santri voice note", "santri_id", id, "path", santri.VoiceNoteUrl, "error", err)
		}
	}
	return nil
}

func (s *SantriService) List(params models.SantriListParams) (*models.SantriListResponse, error) {
	ctx := context.Background()

	var kelasID interface{}
	if params.KelasID > 0 {
		kelasID = params.KelasID
	}

	total, err := s.querier.CountSantri(ctx, queries.CountSantriParams{
		AngkatanPendaftaran: params.AngkatanPendaftaran,
		AngkatanKelas:       params.AngkatanKelas,
		Level:               params.Level,
		Tipe:                params.Tipe,
		Jadwal:              params.Jadwal,
		Gender:              params.Gender,
		Status:              params.Status,
		KelasID:             kelasID,
		Lengkap:             params.Lengkap,
		IDBermasalah:        params.IDBermasalah,
		Search:              params.Search,
	})
	if err != nil {
		return nil, err
	}

	list, err := s.querier.ListSantri(ctx, queries.ListSantriParams{
		AngkatanPendaftaran: params.AngkatanPendaftaran,
		AngkatanKelas:       params.AngkatanKelas,
		Level:               params.Level,
		Tipe:                params.Tipe,
		Jadwal:              params.Jadwal,
		Gender:              params.Gender,
		Status:              params.Status,
		KelasID:             kelasID,
		Lengkap:             params.Lengkap,
		IDBermasalah:        params.IDBermasalah,
		Search:              params.Search,
		Off:                 params.Offset,
		Lim:                 params.Limit,
	})
	if err != nil {
		return nil, err
	}
	duplicates, err := loadDuplicateIDMahasantri(ctx, s.querier)
	if err != nil {
		return nil, err
	}

	santriList := make([]models.SantriResponse, len(list))
	for i := range list {
		santriList[i] = s.santriResponse(list[i], duplicates)
	}

	return &models.SantriListResponse{
		Data:  santriList,
		Total: total,
	}, nil
}

func (s *SantriService) UpdateByCS(id int64, req models.UpdateSantriCSRequest) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txQuerier := s.querier.WithTx(tx)

	existing, err := txQuerier.GetSantriByID(ctx, id)
	if err != nil {
		return err
	}
	nama, jenisKelamin, err := validateMutableSantriInput(req.Nama, req.JenisKelamin)
	if err != nil {
		return err
	}
	req.Nama = nama
	req.JenisKelamin = jenisKelamin
	now := time.Now()

	if err := txQuerier.UpdateSantriCSMutable(ctx, queries.UpdateSantriCSMutableParams{
		KelasKode:    req.KelasKode,
		Nama:         req.Nama,
		JenisKelamin: req.JenisKelamin,
		Nominal:      req.Nominal,
		Usia:         sql.NullInt64{Int64: req.Usia, Valid: req.Usia > 0},
		Domisili:     req.Domisili,
		NoWa:         req.NoWA,
		Email:        req.Email,
		UpdatedAt:    now,
		ID:           id,
	}); err != nil {
		return err
	}
	if !isValidIssuedIDMahasantriForRow(existing.IDMahasantri, id) {
		_, _, angkatan, parsedDate, err := s.validateSantriInput(ctx, txQuerier, req.Nama, req.JenisKelamin, req.Angkatan, req.TanggalDaftar)
		if err != nil {
			return err
		}
		if err := txQuerier.IssueSantriRegistrationIdentity(ctx, queries.IssueSantriRegistrationIdentityParams{
			IDMahasantri:  s.GenerateIDMahasantri(angkatan, id, parsedDate),
			Angkatan:      angkatan,
			TanggalDaftar: sql.NullTime{Time: parsedDate, Valid: true},
			UpdatedAt:     now,
			ID:            id,
		}); err != nil {
			return err
		}
	}
	santri, err := txQuerier.GetSantriByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.engine.WithQuerier(txQuerier).ProcessSantri(ctx, &santri); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SantriService) CorrectRegistrationIdentity(id int64, req models.CorrectSantriRegistrationRequest) (*models.SantriResponse, error) {
	if !req.Konfirmasi {
		return nil, errors.New("konfirmasi koreksi identitas wajib diberikan")
	}
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txQuerier := s.querier.WithTx(tx)
	existing, err := txQuerier.GetSantriByID(ctx, id)
	if err != nil {
		return nil, err
	}
	_, _, angkatan, parsedDate, err := s.validateSantriInput(ctx, txQuerier, existing.Nama, existing.JenisKelamin, req.Angkatan, req.TanggalDaftar)
	if err != nil {
		return nil, err
	}
	newID := s.GenerateIDMahasantri(angkatan, id, parsedDate)
	if err := txQuerier.CorrectSantriRegistrationIdentity(ctx, queries.CorrectSantriRegistrationIdentityParams{
		IDMahasantri: newID, Angkatan: angkatan, TanggalDaftar: sql.NullTime{Time: parsedDate, Valid: true}, UpdatedAt: time.Now(), ID: id,
	}); err != nil {
		return nil, err
	}
	updated, err := txQuerier.GetSantriByID(ctx, id)
	if err != nil {
		return nil, err
	}
	duplicates, err := loadDuplicateIDMahasantri(ctx, txQuerier)
	if err != nil {
		return nil, err
	}
	response := s.santriResponse(updated, duplicates)
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	slog.Info("superadmin corrected santri registration identity", "santri_id", id, "old_id_mahasantri", existing.IDMahasantri, "new_id_mahasantri", newID)
	return &response, nil
}

func (s *SantriService) CountIDMahasantriBermasalah() (int64, error) {
	return s.querier.CountSantri(context.Background(), queries.CountSantriParams{
		AngkatanPendaftaran: "", AngkatanKelas: "", Level: "", Tipe: "", Jadwal: "", Gender: "", Status: "",
		KelasID: nil, Lengkap: int64(-1), IDBermasalah: int64(1), Search: "",
	})
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
	fullPath, err := safeVoiceNotePath(santri.VoiceNoteUrl)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(fullPath); err != nil {
		return "", err
	}
	return fullPath, nil
}

func safeVoiceNotePath(voiceNoteURL string) (string, error) {
	path := filepath.Clean(filepath.FromSlash(voiceNoteURL))
	if path == "." || filepath.IsAbs(path) || filepath.Dir(path) != "voice-notes" || filepath.Base(path) == "." {
		return "", errors.New("lokasi voice note tidak valid")
	}
	return filepath.Join("data", path), nil
}

func (s *SantriService) UpdateByAdminKelas(id int64, req models.UpdateSantriAdminKelasRequest) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txQuerier := s.querier.WithTx(tx)
	existing, err := txQuerier.GetSantriByID(ctx, id)
	if err != nil {
		return err
	}
	req.AngkatanKelas = strings.TrimSpace(req.AngkatanKelas)
	if req.AngkatanKelas == "" {
		return errors.New("angkatan kelas wajib diisi")
	}
	if req.AngkatanKelas != existing.AngkatanKelas {
		if _, err := txQuerier.GetAngkatanByKode(ctx, req.AngkatanKelas); errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("angkatan kelas %q tidak terdaftar di master angkatan", req.AngkatanKelas)
		} else if err != nil {
			return fmt.Errorf("gagal memvalidasi angkatan kelas: %w", err)
		}
	}
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

	if err := txQuerier.UpdateSantriAdminKelas(ctx, queries.UpdateSantriAdminKelasParams{
		Fu:            req.Fu,
		TanggalVn:     tanggalVn,
		HasilVn:       req.HasilVn,
		MasukGrup:     req.MasukGrup,
		MulaiBelajar:  mulaiBelajar,
		Jumlah:        sql.NullInt64{Int64: req.Jumlah, Valid: req.Jumlah > 0},
		AngkatanKelas: req.AngkatanKelas,
		Level:         req.Level,
		Jadwal:        req.Jadwal,
		Guru:          req.Guru,
		UpdatedAt:     now,
		ID:            id,
	}); err != nil {
		return err
	}

	santri, err := txQuerier.GetSantriByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.engine.WithQuerier(txQuerier).ProcessSantri(ctx, &santri); err != nil {
		return err
	}
	if req.GuruID <= 0 {
		return tx.Commit()
	}
	updated, err := txQuerier.GetSantriByID(ctx, id)
	if err != nil {
		return err
	}
	if !updated.KelasID.Valid {
		return tx.Commit()
	}
	if err := txQuerier.AssignGuru(ctx, queries.AssignGuruParams{
		GuruID: sql.NullInt64{Int64: req.GuruID, Valid: true},
		ID:     updated.KelasID.Int64,
	}); err != nil {
		return err
	}
	return tx.Commit()
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
	ctx := context.Background()
	list, err := s.querier.GetPerluDilengkapi(ctx)
	if err != nil {
		return nil, err
	}
	duplicates, err := loadDuplicateIDMahasantri(ctx, s.querier)
	if err != nil {
		return nil, err
	}
	result := make([]models.SantriResponse, len(list))
	for i, santri := range list {
		result[i] = s.santriResponse(santri, duplicates)
	}
	return result, nil
}

func (s *SantriService) PindahkanKelas(santriID, kelasTujuanID int64) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txQuerier := s.querier.WithTx(tx)
	if err := s.engine.WithQuerier(txQuerier).PindahkanSantri(ctx, santriID, kelasTujuanID); err != nil {
		return err
	}
	return tx.Commit()
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
		AngkatanPendaftaran: "", AngkatanKelas: "", Level: "", Tipe: "", Jadwal: "",
		Gender: "", Status: "aktif", KelasID: nil, Lengkap: int64(-1), IDBermasalah: int64(0), Search: "",
	})

	lengkapCount, _ := s.querier.CountSantri(ctx, queries.CountSantriParams{
		AngkatanPendaftaran: "", AngkatanKelas: "", Level: "", Tipe: "", Jadwal: "",
		Gender: "", Status: "aktif", KelasID: nil, Lengkap: int64(1), IDBermasalah: int64(0), Search: "",
	})
	stats.SantriLengkap = lengkapCount

	stats.SantriPerluLengkap = stats.TotalSantri - lengkapCount

	tidakLanjutCount, _ := s.querier.CountSantri(ctx, queries.CountSantriParams{
		AngkatanPendaftaran: "", AngkatanKelas: "", Level: "", Tipe: "", Jadwal: "",
		Gender: "", Status: "tidak_lanjut", KelasID: nil, Lengkap: int64(-1), IDBermasalah: int64(0), Search: "",
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
