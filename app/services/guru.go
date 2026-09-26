package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type GuruService struct {
	querier *queries.Querier
}

func NewGuruService(querier *queries.Querier) *GuruService {
	return &GuruService{querier: querier}
}

func (s *GuruService) GetGuruByUserID(userID int64) (*models.GuruDetailResponse, error) {
	g, err := s.querier.GuruGetByUserID(context.Background(), sql.NullInt64{Int64: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("guru not found for user: %w", err)
	}
	if g.IsAktif != 1 {
		return nil, fmt.Errorf("akun guru tidak aktif")
	}
	totalKelas, _ := s.querier.CountKelasByGuruID(context.Background(), sql.NullInt64{Int64: g.ID, Valid: true})
	totalSantri, _ := s.querier.CountSantriAktifByGuruID(context.Background(), sql.NullInt64{Int64: g.ID, Valid: true})

	tanggalGabung := ""
	if g.TanggalGabung.Valid {
		tanggalGabung = g.TanggalGabung.Time.Format("2006-01-02")
	}

	var userIDPtr *int64
	if g.UserID.Valid {
		userIDPtr = &g.UserID.Int64
	}

	return &models.GuruDetailResponse{
		ID:            g.ID,
		Nama:          g.Nama,
		Gelar:         g.Gelar,
		JenisKelamin:  g.JenisKelamin,
		Status:        g.Status,
		NoWa:          g.NoWa,
		Email:         g.Email,
		TanggalGabung: tanggalGabung,
		Foto:          g.Foto,
		IsAktif:       g.IsAktif == 1,
		UserID:        userIDPtr,
		TotalKelas:    totalKelas,
		TotalSantri:   totalSantri,
	}, nil
}

func (s *GuruService) ListDirectory() ([]models.GuruDirectoryResponse, error) {
	list, err := s.querier.GuruListAll(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.GuruDirectoryResponse, 0, len(list))
	for _, guru := range list {
		out = append(out, s.mapDirectoryGuru(guru))
	}
	return out, nil
}

func (s *GuruService) GetDirectoryByID(id int64) (*models.GuruDirectoryResponse, error) {
	guru, err := s.querier.GuruGetByID(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("guru not found: %w", err)
	}
	result := s.mapDirectoryGuru(guru)
	return &result, nil
}

func (s *GuruService) UpdateDirectory(id int64, req models.UpdateGuruDirectoryRequest) error {
	var tanggalGabung sql.NullTime
	if req.TanggalGabung != "" {
		parsed, err := time.Parse("2006-01-02", req.TanggalGabung)
		if err != nil {
			return fmt.Errorf("tanggal gabung tidak valid")
		}
		tanggalGabung = sql.NullTime{Time: parsed, Valid: true}
	}

	isAktif := int64(0)
	if req.IsAktif {
		isAktif = 1
	}
	return s.querier.UpdateGuruDirectory(context.Background(), queries.UpdateGuruDirectoryParams{
		Nama:          req.Nama,
		Gelar:         req.Gelar,
		JenisKelamin:  req.JenisKelamin,
		Status:        req.Status,
		NoWa:          req.NoWa,
		Email:         req.Email,
		TanggalGabung: tanggalGabung,
		Foto:          req.Foto,
		IsAktif:       isAktif,
		ID:            id,
	})
}

// CreateDirectory registers a new master guru record and returns its id.
func (s *GuruService) CreateDirectory(req models.CreateGuruDirectoryRequest) (int64, error) {
	var tanggalGabung sql.NullTime
	if req.TanggalGabung != "" {
		parsed, err := time.Parse("2006-01-02", req.TanggalGabung)
		if err != nil {
			return 0, fmt.Errorf("tanggal gabung tidak valid")
		}
		tanggalGabung = sql.NullTime{Time: parsed, Valid: true}
	}

	isAktif := int64(0)
	if req.IsAktif {
		isAktif = 1
	}
	return s.querier.CreateGuruFull(context.Background(), queries.CreateGuruFullParams{
		Nama:          req.Nama,
		Gelar:         req.Gelar,
		JenisKelamin:  req.JenisKelamin,
		Status:        req.Status,
		NoWa:          req.NoWa,
		Email:         req.Email,
		TanggalGabung: tanggalGabung,
		Foto:          req.Foto,
		IsAktif:       isAktif,
	})
}

// ListLinkableUsers returns accounts not yet linked to any guru record.
func (s *GuruService) ListLinkableUsers() ([]models.LinkableUserResponse, error) {
	rows, err := s.querier.ListLinkableUsers(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.LinkableUserResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.LinkableUserResponse{
			ID:    r.ID,
			Name:  r.Name,
			Email: r.Email,
			Role:  r.Role,
		})
	}
	return out, nil
}

// LinkUser connects a login account to a guru record. The guru.user_id UNIQUE
// index guarantees one account maps to one guru; we surface a friendly error
// instead of letting the constraint bubble up.
func (s *GuruService) LinkUser(guruID, userID int64) error {
	if _, err := s.querier.GuruGetByID(context.Background(), guruID); err != nil {
		return fmt.Errorf("guru tidak ditemukan")
	}
	existing, err := s.querier.CountGuruByUserID(context.Background(), sql.NullInt64{Int64: userID, Valid: true})
	if err != nil {
		return err
	}
	if existing > 0 {
		return fmt.Errorf("akun ini sudah terhubung dengan guru lain")
	}
	return s.querier.UpdateGuruUserID(context.Background(), queries.UpdateGuruUserIDParams{
		UserID: sql.NullInt64{Int64: userID, Valid: true},
		ID:     guruID,
	})
}

// UnlinkUser detaches the login account from a guru record.
func (s *GuruService) UnlinkUser(guruID int64) error {
	return s.querier.UpdateGuruUserID(context.Background(), queries.UpdateGuruUserIDParams{
		UserID: sql.NullInt64{Valid: false},
		ID:     guruID,
	})
}

func (s *GuruService) mapDirectoryGuru(guru queries.Guru) models.GuruDirectoryResponse {
	tanggalGabung := ""
	if guru.TanggalGabung.Valid {
		tanggalGabung = guru.TanggalGabung.Time.Format("2006-01-02")
	}
	var userID *int64
	if guru.UserID.Valid {
		userID = &guru.UserID.Int64
	}
	totalKelas, _ := s.querier.CountKelasByGuruID(context.Background(), sql.NullInt64{Int64: guru.ID, Valid: true})
	totalSantri, _ := s.querier.CountSantriAktifByGuruID(context.Background(), sql.NullInt64{Int64: guru.ID, Valid: true})
	return models.GuruDirectoryResponse{
		ID:            guru.ID,
		Nama:          guru.Nama,
		Gelar:         guru.Gelar,
		JenisKelamin:  guru.JenisKelamin,
		Status:        guru.Status,
		NoWa:          guru.NoWa,
		Email:         guru.Email,
		TanggalGabung: tanggalGabung,
		Foto:          guru.Foto,
		IsAktif:       guru.IsAktif == 1,
		UserID:        userID,
		TotalKelas:    totalKelas,
		TotalSantri:   totalSantri,
	}
}

func (s *GuruService) EnsureOwnsClass(guruID, kelasID int64) error {
	k, err := s.querier.GetKelasByID(context.Background(), kelasID)
	if err != nil {
		return fmt.Errorf("kelas not found: %w", err)
	}
	if !k.GuruID.Valid || k.GuruID.Int64 != guruID {
		return fmt.Errorf("anda tidak memiliki akses ke kelas ini")
	}
	return nil
}

// GetViewerGuruID returns nil for a super admin, which denotes unrestricted read access.
func (s *GuruService) GetViewerGuruID(userID int64, isSuperAdmin bool) (*int64, error) {
	if isSuperAdmin {
		return nil, nil
	}
	guru, err := s.GetGuruByUserID(userID)
	if err != nil {
		return nil, err
	}
	return &guru.ID, nil
}

func (s *GuruService) EnsureCanAccessClass(guruID *int64, kelasID int64) error {
	if guruID == nil {
		_, err := s.querier.GetKelasByID(context.Background(), kelasID)
		if err != nil {
			return fmt.Errorf("kelas not found: %w", err)
		}
		return nil
	}
	return s.EnsureOwnsClass(*guruID, kelasID)
}

func (s *GuruService) EnsureSantriInClass(kelasID, santriID int64) error {
	santri, err := s.querier.GetSantriByKelasID(context.Background(), sql.NullInt64{Int64: kelasID, Valid: true})
	if err != nil {
		return err
	}
	for _, s := range santri {
		if s.ID == santriID {
			return nil
		}
	}
	return fmt.Errorf("santri tidak ditemukan di kelas ini")
}

func (s *GuruService) GetDashboard(guruID int64) (*models.GuruDashboardResponse, error) {
	return s.getDashboard(sql.NullInt64{Int64: guruID, Valid: true})
}

func (s *GuruService) GetDashboardForViewer(guruID *int64) (*models.GuruDashboardResponse, error) {
	filter := sql.NullInt64{}
	if guruID != nil {
		filter = sql.NullInt64{Int64: *guruID, Valid: true}
	}
	return s.getDashboard(filter)
}

func (s *GuruService) getDashboard(guruID sql.NullInt64) (*models.GuruDashboardResponse, error) {
	stats, err := s.querier.GetGuruStats(context.Background(), guruID)
	if err != nil {
		return nil, err
	}

	jadwalRows, err := s.querier.GetJadwalMengajarHariIni(context.Background(), guruID)
	if err != nil {
		return nil, err
	}
	jadwal := make([]models.KelasGuruResponse, 0, len(jadwalRows))
	for _, j := range jadwalRows {
		jumlahSantri := int64(0)
		switch v := j.JumlahSantriAktif.(type) {
		case int64:
			jumlahSantri = v
		}
		jadwal = append(jadwal, models.KelasGuruResponse{
			ID:           j.ID,
			NamaKelas:    j.NamaKelas,
			Jadwal:       j.Jadwal,
			Level:        j.Level,
			Tipe:         j.Tipe,
			JenisKelamin: j.JenisKelamin,
			Frekuensi:    j.Frekuensi,
			Kapasitas:    j.Kapasitas,
			JumlahSantri: jumlahSantri,
		})
	}

	belumRows, err := s.querier.GetKelasBelumAbsen(context.Background(), guruID)
	if err != nil {
		return nil, err
	}
	belum := make([]models.KelasGuruResponse, 0, len(belumRows))
	for _, b := range belumRows {
		pertemuanTerakhir := ""
		if b.PertemuanTerakhir > 0 {
			pertemuanTerakhir = fmt.Sprintf("%d", b.PertemuanTerakhir)
		}
		tanggalTerakhir := ""
		switch v := b.TanggalTerakhir.(type) {
		case string:
			tanggalTerakhir = v
		}
		belum = append(belum, models.KelasGuruResponse{
			ID:                b.ID,
			NamaKelas:         b.NamaKelas,
			Jadwal:            b.Jadwal,
			Level:             b.Level,
			Tipe:              b.Tipe,
			PertemuanTerakhir: pertemuanTerakhir,
			TanggalTerakhir:   tanggalTerakhir,
		})
	}

	return &models.GuruDashboardResponse{
		TotalKelas:      stats.TotalKelas,
		TotalSantri:     stats.TotalSantri,
		SantriAktif:     stats.SantriAktif,
		TotalPertemuan:  stats.TotalPertemuan,
		JadwalHariIni:   jadwal,
		KelasBelumAbsen: belum,
	}, nil
}

func (s *GuruService) ListKelasSaya(guruID int64) ([]models.KelasGuruResponse, error) {
	return s.listKelas(sql.NullInt64{Int64: guruID, Valid: true})
}

func (s *GuruService) ListKelasForViewer(guruID *int64) ([]models.KelasGuruResponse, error) {
	filter := sql.NullInt64{}
	if guruID != nil {
		filter = sql.NullInt64{Int64: *guruID, Valid: true}
	}
	return s.listKelas(filter)
}

func (s *GuruService) listKelas(guruID sql.NullInt64) ([]models.KelasGuruResponse, error) {
	rows, err := s.querier.ListKelasByGuruID(context.Background(), guruID)
	if err != nil {
		return nil, err
	}
	out := make([]models.KelasGuruResponse, 0, len(rows))
	for _, r := range rows {
		var guruID *int64
		if r.GuruID.Valid {
			guruID = &r.GuruID.Int64
		}
		out = append(out, models.KelasGuruResponse{
			ID:           r.ID,
			GuruID:       guruID,
			GuruNama:     r.GuruNama.String,
			NamaKelas:    r.NamaKelas,
			Level:        r.Level,
			Tipe:         r.Tipe,
			Frekuensi:    r.Frekuensi,
			Jadwal:       r.Jadwal,
			JenisKelamin: r.JenisKelamin,
			Kapasitas:    r.Kapasitas,
			JumlahSantri: r.JumlahSantri,
		})
	}
	return out, nil
}

func (s *GuruService) GetDetailKelas(guruID, kelasID int64) (*models.KelasGuruResponse, []models.SantriGuruResponse, error) {
	return s.GetDetailKelasForViewer(&guruID, kelasID)
}

func (s *GuruService) GetDetailKelasForViewer(guruID *int64, kelasID int64) (*models.KelasGuruResponse, []models.SantriGuruResponse, error) {
	if err := s.EnsureCanAccessClass(guruID, kelasID); err != nil {
		return nil, nil, err
	}

	k, err := s.querier.GetKelasByID(context.Background(), kelasID)
	if err != nil {
		return nil, nil, fmt.Errorf("kelas not found: %w", err)
	}

	kelas := &models.KelasGuruResponse{
		ID:               k.ID,
		NamaKelas:        k.NamaKelas,
		Level:            k.Level,
		Tipe:             k.Tipe,
		Frekuensi:        k.Frekuensi,
		Jadwal:           k.Jadwal,
		JenisKelamin:     k.JenisKelamin,
		Kapasitas:        k.Kapasitas,
		JumlahSantri:     k.JumlahSantri,
		MateriIndividual: k.MateriIndividual == 1,
	}
	if k.PertemuanTerakhir > 0 {
		// Before any recorded meeting the anchor is shown relative to the
		// current level, which restarts at 1 after a level change.
		levelAwal := int64(0)
		if state, err := s.querier.GetKelasPerubahanState(context.Background(), kelasID); err == nil {
			levelAwal = state.LevelPertemuanAwal
		}
		if n := k.PertemuanTerakhir - levelAwal; n > 0 {
			kelas.PertemuanTerakhir = fmt.Sprintf("%d", n)
		}
	}
	if pertemuanTerakhir, err := s.querier.GetLastPertemuanByKelas(context.Background(), kelasID); err == nil {
		kelas.PertemuanTerakhir = fmt.Sprintf("%d", pertemuanTerakhir.PertemuanLevelKe)
		kelas.TanggalTerakhir = pertemuanTerakhir.Tanggal.Format("2006-01-02")
		kelas.MateriTerakhir = pertemuanTerakhir.Materi
	}

	santriRows, err := s.querier.GetSantriByKelasID(context.Background(), sql.NullInt64{Int64: kelasID, Valid: true})
	if err != nil {
		return nil, nil, err
	}

	absensiStats, _ := s.querier.CountAbsensiSantriByKelas(context.Background(), kelasID)
	batasMateriRows, err := s.querier.GetBatasMateriTerakhirByKelas(context.Background(), kelasID)
	if err != nil {
		return nil, nil, err
	}
	batasMateriBySantri := make(map[int64]string, len(batasMateriRows))
	for _, row := range batasMateriRows {
		batasMateriBySantri[row.SantriID] = row.BatasMateri
	}
	statsMap := make(map[int64]struct {
		totalHadir, totalIzin, totalSakit, totalAlpa, totalTelat int64
	})
	for _, as := range absensiStats {
		hadir := int64(0)
		if as.TotalHadir.Valid {
			hadir = int64(as.TotalHadir.Float64)
		}
		izin := int64(0)
		if as.TotalIzin.Valid {
			izin = int64(as.TotalIzin.Float64)
		}
		sakit := int64(0)
		if as.TotalSakit.Valid {
			sakit = int64(as.TotalSakit.Float64)
		}
		alpa := int64(0)
		if as.TotalAlpa.Valid {
			alpa = int64(as.TotalAlpa.Float64)
		}
		telat := int64(0)
		if as.TotalTelat.Valid {
			telat = int64(as.TotalTelat.Float64)
		}
		statsMap[as.SantriID] = struct {
			totalHadir, totalIzin, totalSakit, totalAlpa, totalTelat int64
		}{hadir, izin, sakit, alpa, telat}
	}

	out := make([]models.SantriGuruResponse, 0, len(santriRows))
	for _, sant := range santriRows {
		stats := statsMap[sant.ID]
		total := stats.totalHadir + stats.totalIzin + stats.totalSakit + stats.totalAlpa + stats.totalTelat
		persenHadir := 0.0
		if total > 0 {
			persenHadir = float64(stats.totalHadir) / float64(total) * 100
		}

		tanggalMulai := ""
		if sant.TanggalDaftar.Valid {
			tanggalMulai = sant.TanggalDaftar.Time.Format("2006-01-02")
		}

		var usia *int64
		if sant.Usia.Valid {
			usia = &sant.Usia.Int64
		}

		tanggalHadirTerakhir := ""
		lastAbsen, err := s.querier.GetAbsensiTerakhirBySantri(context.Background(), sant.ID)
		if err == nil && lastAbsen.Status == "hadir" {
			tanggalHadirTerakhir = lastAbsen.Tanggal.Format("2006-01-02")
		}

		out = append(out, models.SantriGuruResponse{
			ID:                   sant.ID,
			IDMahasantri:         sant.IDMahasantri,
			Nama:                 sant.Nama,
			Usia:                 usia,
			Domisili:             sant.Domisili,
			NoWa:                 sant.NoWa,
			TanggalMulai:         tanggalMulai,
			PersenHadir:          persenHadir,
			TotalHadir:           stats.totalHadir,
			TotalIzin:            stats.totalIzin,
			TotalSakit:           stats.totalSakit,
			TotalAlpa:            stats.totalAlpa,
			TotalTelat:           stats.totalTelat,
			TanggalHadirTerakhir: tanggalHadirTerakhir,
			BatasMateriTerakhir:  batasMateriBySantri[sant.ID],
		})
	}

	return kelas, out, nil
}

func (s *GuruService) GetRiwayatSantri(guruID, santriID int64) ([]models.AbsensiResponse, error) {
	absensi, err := s.querier.GetAbsensiBySantri(context.Background(), santriID)
	if err != nil {
		return nil, err
	}
	out := make([]models.AbsensiResponse, 0, len(absensi))
	for _, a := range absensi {
		out = append(out, models.AbsensiResponse{
			ID:          a.ID,
			PertemuanID: a.PertemuanID,
			SantriID:    a.SantriID,
			SantriNama:  "",
			Status:      a.Status,
			Catatan:     a.Catatan,
			BatasMateri: a.BatasMateri,
		})
	}
	return out, nil
}

// ============ Tilawah Harian ============

func monthRange(now time.Time) (string, string) {
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, -1)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

func (s *GuruService) CheckinTilawah(guruID int64) error {
	today := HariIniRiayah().Format("2006-01-02")
	t, _ := time.Parse("2006-01-02", today)
	return s.querier.CreateTilawah(context.Background(), queries.CreateTilawahParams{GuruID: guruID, Tanggal: t})
}

func (s *GuruService) UncheckTilawah(guruID int64) error {
	today := HariIniRiayah().Format("2006-01-02")
	t, _ := time.Parse("2006-01-02", today)
	return s.querier.DeleteTilawah(context.Background(), queries.DeleteTilawahParams{GuruID: guruID, Tanggal: t})
}

func (s *GuruService) GetTilawahStatus(guruID int64) (*models.TilawahStatusResponse, error) {
	ctx := context.Background()
	// Tanggal tilawah mengikuti WIB, bukan jam server (UTC).
	now := HariIniRiayah()
	today, _ := time.Parse("2006-01-02", now.Format("2006-01-02"))
	startStr, endStr := monthRange(now)
	start, _ := time.Parse("2006-01-02", startStr)
	end, _ := time.Parse("2006-01-02", endStr)

	existsToday, _ := s.querier.ExistsTilawah(ctx, queries.ExistsTilawahParams{GuruID: guruID, Tanggal: today})
	count, _ := s.querier.CountTilawahByGuruRange(ctx, queries.CountTilawahByGuruRangeParams{GuruID: guruID, Tanggal: start, Tanggal_2: end})
	dates, _ := s.querier.ListTilawahByGuruRange(ctx, queries.ListTilawahByGuruRangeParams{GuruID: guruID, Tanggal: start, Tanggal_2: end})

	list := make([]string, 0, len(dates))
	for _, d := range dates {
		list = append(list, d.Format("2006-01-02"))
	}
	return &models.TilawahStatusResponse{
		SudahHariIni: existsToday > 0,
		BulanIni:     count,
		Tanggal:      list,
	}, nil
}

func formatTime(t time.Time) string {
	return t.Format("15:04")
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}
