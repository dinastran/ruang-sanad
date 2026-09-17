package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type KoordinatorService struct {
	querier *queries.Querier
}

func NewKoordinatorService(querier *queries.Querier) *KoordinatorService {
	return &KoordinatorService{querier: querier}
}

func parseDateStrict(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func parseNullableDate(s string) sql.NullTime {
	if s == "" {
		return sql.NullTime{Valid: false}
	}
	for _, format := range []string{"2006-01-02", "02/01/2006", "1/2/2006", "2006/01/02"} {
		if parsed, err := time.Parse(format, s); err == nil {
			return sql.NullTime{Time: parsed, Valid: true}
		}
	}
	return sql.NullTime{Valid: false}
}

func nullDateStr(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02")
}

// ============ Dashboard ============

func (s *KoordinatorService) GetDashboard() (*models.KoordinatorDashboardResponse, error) {
	ctx := context.Background()
	total, _ := s.querier.CountGuruAktifTotal(ctx)
	tetap, _ := s.querier.CountGuruByStatusAktif(ctx, "tetap")
	partTime, _ := s.querier.CountGuruByStatusAktif(ctx, "part_time")
	totalKelas, _ := s.querier.CountKelasAktifTotal(ctx)
	totalSantri, _ := s.querier.CountSantriAktifTotal(ctx)

	belumRows, err := s.querier.ListGuruBelumAbsenPekanIni(ctx)
	if err != nil {
		return nil, err
	}
	belum := make([]models.GuruRingkas, 0, len(belumRows))
	for _, b := range belumRows {
		belum = append(belum, models.GuruRingkas{ID: b.ID, Nama: b.Nama, NoWa: b.NoWa})
	}

	return &models.KoordinatorDashboardResponse{
		TotalGuru:      total,
		GuruTetap:      tetap,
		GuruPartTime:   partTime,
		TotalKelas:     totalKelas,
		TotalSantri:    totalSantri,
		GuruBelumAbsen: belum,
	}, nil
}

// ListGuruSimple returns active guru for dropdowns (id, nama).
func (s *KoordinatorService) ListGuruSimple() ([]models.GuruRingkas, error) {
	rows, err := s.querier.ListGuruAktifSimple(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.GuruRingkas, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.GuruRingkas{ID: r.ID, Nama: r.Nama, NoWa: r.NoWa})
	}
	return out, nil
}

// KelasSimple is a minimal class option for dropdowns.
type KelasSimple struct {
	ID        int64  `json:"id"`
	NamaKelas string `json:"nama_kelas"`
}

func (s *KoordinatorService) ListKelasSimple() ([]KelasSimple, error) {
	rows, err := s.querier.ListKelasAktifSimple(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]KelasSimple, 0, len(rows))
	for _, r := range rows {
		out = append(out, KelasSimple{ID: r.ID, NamaKelas: r.NamaKelas})
	}
	return out, nil
}

// ============ Kompetensi ============

func (s *KoordinatorService) GetKompetensi(guruID int64) (*models.KompetensiResponse, error) {
	k, err := s.querier.GetGuruKompetensi(context.Background(), guruID)
	if err != nil {
		// No record yet → return zeroed competencies.
		return &models.KompetensiResponse{}, nil
	}
	return &models.KompetensiResponse{
		HafalanQuran:     k.HafalanQuran,
		HafalanTuhfah:    k.HafalanTuhfah,
		HafalanJazariy:   k.HafalanJazariy,
		HafalanKhaqaniy:  k.HafalanKhaqaniy,
		HafalanSyakhawiy: k.HafalanSyakhawiy,
		SanadQiroah:      k.SanadQiroah,
		BahasaArabPasif:  k.BahasaArabPasif,
		BahasaArabAktif:  k.BahasaArabAktif,
	}, nil
}

func (s *KoordinatorService) SaveKompetensi(guruID int64, req models.KompetensiRequest) error {
	return s.querier.UpsertGuruKompetensi(context.Background(), queries.UpsertGuruKompetensiParams{
		GuruID:           guruID,
		HafalanQuran:     clampLevel(req.HafalanQuran),
		HafalanTuhfah:    clampLevel(req.HafalanTuhfah),
		HafalanJazariy:   clampLevel(req.HafalanJazariy),
		HafalanKhaqaniy:  clampLevel(req.HafalanKhaqaniy),
		HafalanSyakhawiy: clampLevel(req.HafalanSyakhawiy),
		SanadQiroah:      clampLevel(req.SanadQiroah),
		BahasaArabPasif:  clampLevel(req.BahasaArabPasif),
		BahasaArabAktif:  clampLevel(req.BahasaArabAktif),
	})
}

func clampLevel(v int64) int64 {
	if v < 0 {
		return 0
	}
	if v > 3 {
		return 3
	}
	return v
}

// ============ Pembinaan ============

func (s *KoordinatorService) ListPembinaan() ([]models.PembinaanResponse, error) {
	rows, err := s.querier.ListPembinaan(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.PembinaanResponse, 0, len(rows))
	for _, p := range rows {
		out = append(out, mapPembinaan(p))
	}
	return out, nil
}

func (s *KoordinatorService) GetPembinaan(id int64) (*models.PembinaanResponse, error) {
	p, err := s.querier.GetPembinaan(context.Background(), id)
	if err != nil {
		return nil, err
	}
	r := mapPembinaan(p)
	return &r, nil
}

func (s *KoordinatorService) CreatePembinaan(req models.PembinaanRequest) (int64, error) {
	t, err := parseDateStrict(req.Tanggal)
	if err != nil {
		return 0, fmt.Errorf("tanggal tidak valid")
	}
	return s.querier.CreatePembinaan(context.Background(), queries.CreatePembinaanParams{
		Tanggal:    t,
		Bulan:      req.Bulan,
		PekanKe:    req.PekanKe,
		Topik:      req.Topik,
		Keterangan: req.Keterangan,
		Status:     normalizePembinaanStatus(req.Status),
	})
}

func (s *KoordinatorService) UpdatePembinaan(id int64, req models.PembinaanRequest) error {
	t, err := parseDateStrict(req.Tanggal)
	if err != nil {
		return fmt.Errorf("tanggal tidak valid")
	}
	return s.querier.UpdatePembinaan(context.Background(), queries.UpdatePembinaanParams{
		Tanggal:    t,
		Bulan:      req.Bulan,
		PekanKe:    req.PekanKe,
		Topik:      req.Topik,
		Keterangan: req.Keterangan,
		Status:     normalizePembinaanStatus(req.Status),
		ID:         id,
	})
}

func (s *KoordinatorService) DeletePembinaan(id int64) error {
	return s.querier.DeletePembinaan(context.Background(), id)
}

func (s *KoordinatorService) GetPembinaanAbsen(pembinaanID int64) ([]models.AbsenRow, error) {
	rows, err := s.querier.ListGuruWithPembinaanAbsen(context.Background(), pembinaanID)
	if err != nil {
		return nil, err
	}
	out := make([]models.AbsenRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.AbsenRow{GuruID: r.GuruID, Nama: r.Nama, Status: r.Status, Hadir: r.Hadir == 1, JamMasuk: r.JamMasuk, Keterangan: r.Keterangan, Alasan: r.Alasan})
	}
	return out, nil
}

func (s *KoordinatorService) SavePembinaanAbsen(pembinaanID int64, inputs []models.AbsenInput) error {
	ctx := context.Background()
	pembinaan, err := s.querier.GetPembinaan(ctx, pembinaanID)
	if err != nil {
		return fmt.Errorf("sesi pembinaan tidak ditemukan")
	}
	if pembinaan.Status != "terlaksana" {
		return fmt.Errorf("absensi hanya dapat disimpan untuk pembinaan terlaksana")
	}
	expected, err := s.GetPembinaanAbsen(pembinaanID)
	if err != nil {
		return err
	}
	if err := validateAbsenBatch(expected, inputs); err != nil {
		return err
	}
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	querier := s.querier.WithTx(tx)
	for _, in := range inputs {
		hadir := int64(0)
		if in.Hadir {
			hadir = 1
		}
		if err := querier.UpsertPembinaanAbsen(ctx, queries.UpsertPembinaanAbsenParams{
			PembinaanID: pembinaanID,
			GuruID:      in.GuruID,
			Hadir:       hadir,
			JamMasuk:    in.JamMasuk,
			Keterangan:  in.Keterangan,
			Alasan:      in.Alasan,
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func mapPembinaan(p queries.Pembinaan) models.PembinaanResponse {
	return models.PembinaanResponse{
		ID:         p.ID,
		Tanggal:    p.Tanggal.Format("2006-01-02"),
		Bulan:      p.Bulan,
		PekanKe:    p.PekanKe,
		Topik:      p.Topik,
		Keterangan: p.Keterangan,
		Status:     p.Status,
	}
}

func normalizePembinaanStatus(s string) string {
	switch s {
	case "terlaksana", "libur", "dijadwalkan":
		return s
	default:
		return "dijadwalkan"
	}
}

// ============ Rapat ============

func (s *KoordinatorService) ListRapat() ([]models.RapatResponse, error) {
	rows, err := s.querier.ListRapat(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.RapatResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, mapRapat(r))
	}
	return out, nil
}

func (s *KoordinatorService) GetRapat(id int64) (*models.RapatResponse, error) {
	r, err := s.querier.GetRapat(context.Background(), id)
	if err != nil {
		return nil, err
	}
	m := mapRapat(r)
	return &m, nil
}

func (s *KoordinatorService) CreateRapat(req models.RapatRequest) (int64, error) {
	t, err := parseDateStrict(req.Tanggal)
	if err != nil {
		return 0, fmt.Errorf("tanggal tidak valid")
	}
	return s.querier.CreateRapat(context.Background(), queries.CreateRapatParams{
		Tanggal: t,
		Judul:   req.Judul,
		Catatan: req.Catatan,
		Status:  normalizeRapatStatus(req.Status),
	})
}

func (s *KoordinatorService) UpdateRapat(id int64, req models.RapatRequest) error {
	t, err := parseDateStrict(req.Tanggal)
	if err != nil {
		return fmt.Errorf("tanggal tidak valid")
	}
	return s.querier.UpdateRapat(context.Background(), queries.UpdateRapatParams{
		Tanggal: t,
		Judul:   req.Judul,
		Catatan: req.Catatan,
		Status:  normalizeRapatStatus(req.Status),
		ID:      id,
	})
}

func (s *KoordinatorService) DeleteRapat(id int64) error {
	return s.querier.DeleteRapat(context.Background(), id)
}

func (s *KoordinatorService) GetRapatAbsen(rapatID int64) ([]models.AbsenRow, error) {
	rows, err := s.querier.ListGuruWithRapatAbsen(context.Background(), rapatID)
	if err != nil {
		return nil, err
	}
	out := make([]models.AbsenRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.AbsenRow{GuruID: r.GuruID, Nama: r.Nama, Status: r.Status, Hadir: r.Hadir == 1, JamMasuk: r.JamMasuk, Keterangan: r.Keterangan, Alasan: r.Alasan})
	}
	return out, nil
}

func (s *KoordinatorService) SaveRapatAbsen(rapatID int64, inputs []models.AbsenInput) error {
	ctx := context.Background()
	rapat, err := s.querier.GetRapat(ctx, rapatID)
	if err != nil {
		return fmt.Errorf("rapat tidak ditemukan")
	}
	if rapat.Status != "terlaksana" {
		return fmt.Errorf("absensi hanya dapat disimpan untuk rapat terlaksana")
	}
	expected, err := s.GetRapatAbsen(rapatID)
	if err != nil {
		return err
	}
	if err := validateAbsenBatch(expected, inputs); err != nil {
		return err
	}
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	querier := s.querier.WithTx(tx)
	for _, in := range inputs {
		hadir := int64(0)
		if in.Hadir {
			hadir = 1
		}
		if err := querier.UpsertRapatAbsen(ctx, queries.UpsertRapatAbsenParams{
			RapatID:    rapatID,
			GuruID:     in.GuruID,
			Hadir:      hadir,
			JamMasuk:   in.JamMasuk,
			Keterangan: in.Keterangan,
			Alasan:     in.Alasan,
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func validateAbsenInput(in models.AbsenInput) error {
	if in.GuruID <= 0 {
		return fmt.Errorf("guru tidak valid")
	}
	if strings.TrimSpace(in.Keterangan) == "" {
		return fmt.Errorf("keterangan kehadiran wajib diisi")
	}
	if in.Hadir && strings.TrimSpace(in.JamMasuk) == "" {
		return fmt.Errorf("jam masuk Zoom wajib diisi untuk guru hadir")
	}
	if !in.Hadir && strings.TrimSpace(in.Alasan) == "" {
		return fmt.Errorf("alasan ketidakhadiran wajib diisi")
	}
	return nil
}

func validateAbsenBatch(expected []models.AbsenRow, inputs []models.AbsenInput) error {
	if len(inputs) != len(expected) {
		return fmt.Errorf("absensi harus diisi untuk seluruh guru aktif")
	}
	expectedIDs := make(map[int64]struct{}, len(expected))
	for _, row := range expected {
		expectedIDs[row.GuruID] = struct{}{}
	}
	seen := make(map[int64]struct{}, len(inputs))
	for _, in := range inputs {
		if _, ok := expectedIDs[in.GuruID]; !ok {
			return fmt.Errorf("guru pada absensi tidak valid")
		}
		if _, duplicate := seen[in.GuruID]; duplicate {
			return fmt.Errorf("guru tidak boleh dicatat lebih dari sekali")
		}
		seen[in.GuruID] = struct{}{}
		if err := validateAbsenInput(in); err != nil {
			return err
		}
	}
	return nil
}

func (s *KoordinatorService) ListRiwayatAbsensiGuru() ([]models.RiwayatAbsensiGuru, error) {
	rows, err := s.querier.ListRiwayatAbsensiGuru(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.RiwayatAbsensiGuru, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.RiwayatAbsensiGuru{GuruID: r.GuruID, GuruNama: r.GuruNama, Kegiatan: r.Kegiatan, KegiatanID: r.KegiatanID, Tanggal: r.Tanggal.Format("2006-01-02"), Judul: r.Judul, Hadir: r.Hadir == 1, JamMasuk: r.JamMasuk, Keterangan: r.Keterangan, Alasan: r.Alasan})
	}
	return out, nil
}

func mapRapat(r queries.RapatGuru) models.RapatResponse {
	return models.RapatResponse{
		ID:      r.ID,
		Tanggal: r.Tanggal.Format("2006-01-02"),
		Judul:   r.Judul,
		Catatan: r.Catatan,
		Status:  r.Status,
	}
}

func normalizeRapatStatus(s string) string {
	switch s {
	case "terlaksana", "batal", "dijadwalkan":
		return s
	default:
		return "dijadwalkan"
	}
}

// ============ Kunjungan Kelas ============

func (s *KoordinatorService) ListKunjungan() ([]models.KunjunganResponse, error) {
	rows, err := s.querier.ListKunjungan(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.KunjunganResponse, 0, len(rows))
	for _, r := range rows {
		var kelasID *int64
		if r.KelasID.Valid {
			kelasID = &r.KelasID.Int64
		}
		out = append(out, models.KunjunganResponse{
			ID:            r.ID,
			GuruID:        r.GuruID,
			GuruNama:      r.GuruNama,
			KelasID:       kelasID,
			KelasNama:     r.KelasNama,
			TargetMulai:   nullDateStr(r.TargetMulai),
			TargetSelesai: nullDateStr(r.TargetSelesai),
			Tanggal:       nullDateStr(r.Tanggal),
			Jam:           r.Jam,
			Status:        r.Status,
			Catatan:       r.Catatan,
		})
	}
	return out, nil
}

func (s *KoordinatorService) CreateKunjungan(req models.KunjunganRequest) (int64, error) {
	if err := validateKunjunganAbsensi(req); err != nil {
		return 0, err
	}
	return s.querier.CreateKunjungan(context.Background(), queries.CreateKunjunganParams{
		GuruID:        req.GuruID,
		KelasID:       nullInt(req.KelasID),
		TargetMulai:   parseNullableDate(req.TargetMulai),
		TargetSelesai: parseNullableDate(req.TargetSelesai),
		Tanggal:       parseNullableDate(req.Tanggal),
		Jam:           req.Jam,
		Status:        normalizeKunjunganStatus(req.Status),
		Catatan:       req.Catatan,
	})
}

func (s *KoordinatorService) UpdateKunjungan(id int64, req models.KunjunganRequest) error {
	if err := validateKunjunganAbsensi(req); err != nil {
		return err
	}
	return s.querier.UpdateKunjungan(context.Background(), queries.UpdateKunjunganParams{
		GuruID:        req.GuruID,
		KelasID:       nullInt(req.KelasID),
		TargetMulai:   parseNullableDate(req.TargetMulai),
		TargetSelesai: parseNullableDate(req.TargetSelesai),
		Tanggal:       parseNullableDate(req.Tanggal),
		Jam:           req.Jam,
		Status:        normalizeKunjunganStatus(req.Status),
		Catatan:       req.Catatan,
		ID:            id,
	})
}

func validateKunjunganAbsensi(req models.KunjunganRequest) error {
	if normalizeKunjunganStatus(req.Status) != "terlaksana" {
		return nil
	}
	if _, err := parseDateStrict(req.Tanggal); err != nil {
		return fmt.Errorf("tanggal pelaksanaan wajib dan harus valid")
	}
	if strings.TrimSpace(req.Jam) == "" {
		return fmt.Errorf("jam masuk wajib diisi untuk kunjungan terlaksana")
	}
	if strings.TrimSpace(req.Catatan) == "" {
		return fmt.Errorf("keterangan kunjungan wajib diisi")
	}
	return nil
}

func (s *KoordinatorService) DeleteKunjungan(id int64) error {
	return s.querier.DeleteKunjungan(context.Background(), id)
}

func nullInt(v int64) sql.NullInt64 {
	if v <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: v, Valid: true}
}

func normalizeKunjunganStatus(s string) string {
	switch s {
	case "terlaksana", "ditunda", "batal", "dijadwalkan":
		return s
	default:
		return "dijadwalkan"
	}
}

// ============ Kalam Bersanad ============

func (s *KoordinatorService) ListKalam() ([]models.KalamResponse, error) {
	rows, err := s.querier.ListKalam(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.KalamResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.KalamResponse{
			ID:         r.ID,
			Tanggal:    r.Tanggal.Format("2006-01-02"),
			Topik:      r.Topik,
			Kitab:      r.Kitab,
			Keterangan: r.Keterangan,
			TotalShare: r.TotalShare,
		})
	}
	return out, nil
}

func (s *KoordinatorService) CreateKalam(req models.KalamRequest) (int64, error) {
	t, err := parseDateStrict(req.Tanggal)
	if err != nil {
		return 0, fmt.Errorf("tanggal tidak valid")
	}
	return s.querier.CreateKalam(context.Background(), queries.CreateKalamParams{
		Tanggal:    t,
		Topik:      req.Topik,
		Kitab:      req.Kitab,
		Keterangan: req.Keterangan,
	})
}

func (s *KoordinatorService) UpdateKalam(id int64, req models.KalamRequest) error {
	t, err := parseDateStrict(req.Tanggal)
	if err != nil {
		return fmt.Errorf("tanggal tidak valid")
	}
	return s.querier.UpdateKalam(context.Background(), queries.UpdateKalamParams{
		Tanggal:    t,
		Topik:      req.Topik,
		Kitab:      req.Kitab,
		Keterangan: req.Keterangan,
		ID:         id,
	})
}

func (s *KoordinatorService) DeleteKalam(id int64) error {
	return s.querier.DeleteKalam(context.Background(), id)
}

func (s *KoordinatorService) GetKalamShare(kalamID int64) ([]models.KalamShareRow, error) {
	rows, err := s.querier.ListGuruWithKalamShare(context.Background(), kalamID)
	if err != nil {
		return nil, err
	}
	out := make([]models.KalamShareRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.KalamShareRow{GuruID: r.GuruID, Nama: r.Nama, SudahShare: r.SudahShare == 1})
	}
	return out, nil
}

func (s *KoordinatorService) ToggleKalamShare(kalamID, guruID int64, on bool) error {
	ctx := context.Background()
	if on {
		return s.querier.ToggleKalamShareOn(ctx, queries.ToggleKalamShareOnParams{KalamID: kalamID, GuruID: guruID})
	}
	return s.querier.ToggleKalamShareOff(ctx, queries.ToggleKalamShareOffParams{KalamID: kalamID, GuruID: guruID})
}

// ============ WA Template ============

func (s *KoordinatorService) ListWaTemplate() ([]models.WaTemplateResponse, error) {
	rows, err := s.querier.ListWaTemplate(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.WaTemplateResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.WaTemplateResponse{ID: r.ID, Nama: r.Nama, TargetType: r.TargetType, Body: r.Body, IsAktif: r.IsAktif == 1})
	}
	return out, nil
}

func (s *KoordinatorService) CreateWaTemplate(req models.WaTemplateRequest) (int64, error) {
	return s.querier.CreateWaTemplate(context.Background(), queries.CreateWaTemplateParams{
		Nama:       req.Nama,
		TargetType: normalizeTargetType(req.TargetType),
		Body:       req.Body,
		IsAktif:    boolToInt(req.IsAktif),
	})
}

func (s *KoordinatorService) UpdateWaTemplate(id int64, req models.WaTemplateRequest) error {
	return s.querier.UpdateWaTemplate(context.Background(), queries.UpdateWaTemplateParams{
		Nama:       req.Nama,
		TargetType: normalizeTargetType(req.TargetType),
		Body:       req.Body,
		IsAktif:    boolToInt(req.IsAktif),
		ID:         id,
	})
}

func (s *KoordinatorService) DeleteWaTemplate(id int64) error {
	return s.querier.DeleteWaTemplate(context.Background(), id)
}

func normalizeTargetType(s string) string {
	if s == "guru" {
		return "guru"
	}
	return "santri"
}

// ============ Todo Koordinator ============

func (s *KoordinatorService) ListTodo() ([]models.TodoResponse, error) {
	rows, err := s.querier.ListTodo(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.TodoResponse, 0, len(rows))
	for _, t := range rows {
		out = append(out, mapTodo(t))
	}
	return out, nil
}

func (s *KoordinatorService) CreateTodo(req models.TodoRequest) (int64, error) {
	return s.querier.CreateTodo(context.Background(), queries.CreateTodoParams{
		Judul:         req.Judul,
		Teknis:        req.Teknis,
		Kebutuhan:     req.Kebutuhan,
		Deadline:      parseNullableDate(req.Deadline),
		Pic:           req.Pic,
		Status:        normalizeTodoStatus(req.Status),
		Recurring:     normalizeRecurring(req.Recurring),
		LinkPendukung: req.LinkPendukung,
		Catatan:       req.Catatan,
	})
}

func (s *KoordinatorService) UpdateTodo(id int64, req models.TodoRequest) error {
	return s.querier.UpdateTodo(context.Background(), queries.UpdateTodoParams{
		Judul:         req.Judul,
		Teknis:        req.Teknis,
		Kebutuhan:     req.Kebutuhan,
		Deadline:      parseNullableDate(req.Deadline),
		Pic:           req.Pic,
		Status:        normalizeTodoStatus(req.Status),
		Recurring:     normalizeRecurring(req.Recurring),
		LinkPendukung: req.LinkPendukung,
		Catatan:       req.Catatan,
		ID:            id,
	})
}

func (s *KoordinatorService) SetTodoStatus(id int64, status string) error {
	return s.querier.SetTodoStatus(context.Background(), queries.SetTodoStatusParams{Status: normalizeTodoStatus(status), ID: id})
}

func (s *KoordinatorService) DeleteTodo(id int64) error {
	return s.querier.DeleteTodo(context.Background(), id)
}

func mapTodo(t queries.TodoKoordinator) models.TodoResponse {
	return models.TodoResponse{
		ID:            t.ID,
		Judul:         t.Judul,
		Teknis:        t.Teknis,
		Kebutuhan:     t.Kebutuhan,
		Deadline:      nullDateStr(t.Deadline),
		Pic:           t.Pic,
		Status:        t.Status,
		Recurring:     t.Recurring,
		LinkPendukung: t.LinkPendukung,
		Catatan:       t.Catatan,
	}
}

func normalizeTodoStatus(s string) string {
	switch s {
	case "belum", "proses", "selesai", "batal":
		return s
	default:
		return "belum"
	}
}

func normalizeRecurring(s string) string {
	switch s {
	case "harian", "mingguan", "bulanan", "4bulanan", "none":
		return s
	default:
		return "none"
	}
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
