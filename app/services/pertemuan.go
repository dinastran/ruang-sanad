package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var ErrJadwalPertemuanAktif = errors.New("kelas masih memiliki jadwal pertemuan aktif")
var ErrPertemuanBerlangsung = errors.New("kelas masih memiliki pertemuan yang sedang berlangsung")

type PertemuanService struct {
	querier *queries.Querier
	tagihan interface {
		GenerateOnCompletionWithQuerier(*queries.Querier, int64) error
	}
}

func NewPertemuanService(querier *queries.Querier, tagihan interface {
	GenerateOnCompletionWithQuerier(*queries.Querier, int64) error
}) *PertemuanService {
	return &PertemuanService{querier: querier, tagihan: tagihan}
}

func (s *PertemuanService) MulaiPertemuan(kelasID, userID int64, req models.MulaiPertemuanRequest) (*models.PertemuanResponse, error) {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	querier := s.querier.WithTx(tx)
	if _, err := querier.GetActivePertemuanByKelas(ctx, kelasID); err == nil {
		return nil, ErrPertemuanBerlangsung
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	dueSchedules, err := querier.CountDueJadwalPertemuanByKelas(ctx, queries.CountDueJadwalPertemuanByKelasParams{
		KelasID: kelasID,
		Tanggal: startOfToday(),
	})
	if err != nil {
		return nil, err
	}
	if dueSchedules > 0 {
		return nil, ErrJadwalPertemuanAktif
	}

	nextKe, err := querier.GetNextPertemuanKe(ctx, kelasID)
	if err != nil {
		return nil, err
	}
	nextLevelKe, err := querier.GetNextPertemuanLevelKe(ctx, kelasID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	jamMulai := req.JamMulai
	if jamMulai == "" {
		jamMulai = now.Format("15:04")
	}
	result, err := querier.CreatePertemuan(ctx, queries.CreatePertemuanParams{
		KelasID:          kelasID,
		PertemuanKe:      nextKe,
		Tanggal:          now,
		JamMulai:         jamMulai,
		JamSelesai:       "",
		Materi:           "",
		Catatan:          req.Catatan,
		IsReschedule:     0,
		JadwalSemula:     "",
		AlasanReschedule: "",
		IsBadal:          0,
		GuruPenggantiID:  sql.NullInt64{},
		AlasanBadal:      "",
		Status:           "berlangsung",
		DibuatOleh:       sql.NullInt64{Int64: userID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.PertemuanResponse{
		ID:               id,
		KelasID:          kelasID,
		PertemuanKe:      nextKe,
		PertemuanLevelKe: nextLevelKe,
		Tanggal:          now.Format("2006-01-02"),
		JamMulai:         jamMulai,
		Status:           "berlangsung",
	}, nil
}

func (s *PertemuanService) GetActivePertemuan(kelasID int64) (*models.PertemuanResponse, error) {
	pertemuan, err := s.querier.GetActivePertemuanByKelas(context.Background(), kelasID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mapPertemuanToResponse(pertemuan), nil
}

func (s *PertemuanService) GetLastCompletedAbsensi(kelasID int64) (*models.PertemuanResponse, []models.AbsensiResponse, error) {
	pertemuan, err := s.querier.GetLastPertemuanByKelas(context.Background(), kelasID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, []models.AbsensiResponse{}, nil
	}
	if err != nil {
		return nil, nil, err
	}

	rows, err := s.querier.GetAbsensiByPertemuan(context.Background(), pertemuan.ID)
	if err != nil {
		return nil, nil, err
	}
	absensi := make([]models.AbsensiResponse, 0, len(rows))
	for _, row := range rows {
		absensi = append(absensi, models.AbsensiResponse{
			ID:          row.ID,
			PertemuanID: row.PertemuanID,
			SantriID:    row.SantriID,
			SantriNama:  row.SantriNama,
			Status:      row.Status,
			Catatan:     row.Catatan,
			BatasMateri: row.BatasMateri,
		})
	}

	return mapPertemuanToResponse(pertemuan), absensi, nil
}

func (s *PertemuanService) GetBatasMateriTerakhir(kelasID int64) (map[int64]string, error) {
	rows, err := s.querier.GetBatasMateriTerakhirByKelas(context.Background(), kelasID)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]string, len(rows))
	for _, row := range rows {
		result[row.SantriID] = row.BatasMateri
	}
	return result, nil
}

func (s *PertemuanService) SelesaiPertemuan(pertemuanID, kelasID int64, req models.SelesaiPertemuanRequest, userID int64) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	querier := s.querier.WithTx(tx)
	claimed, err := querier.ClaimPertemuanCompletion(ctx, queries.ClaimPertemuanCompletionParams{ID: pertemuanID, KelasID: kelasID})
	if err != nil {
		return err
	}
	if claimed == 0 {
		return fmt.Errorf("pertemuan tidak ditemukan atau sudah selesai")
	}

	pertemuan, err := querier.GetPertemuanByID(ctx, pertemuanID)
	if err != nil {
		return err
	}
	kelas, err := querier.GetKelasByID(ctx, kelasID)
	if err != nil {
		return err
	}
	if kelas.MateriIndividual != 1 && strings.TrimSpace(req.Materi) == "" {
		return fmt.Errorf("materi umum wajib diisi")
	}
	santri, err := querier.GetSantriByKelasID(ctx, sql.NullInt64{Int64: kelasID, Valid: true})
	if err != nil {
		return err
	}
	if len(req.Absensi) != len(santri) {
		return fmt.Errorf("absensi wajib diisi untuk seluruh santri aktif")
	}

	allowedSantri := make(map[int64]struct{}, len(santri))
	for _, item := range santri {
		allowedSantri[item.ID] = struct{}{}
	}
	allowedStatus := map[string]struct{}{"hadir": {}, "izin": {}, "sakit": {}, "alpa": {}, "telat": {}}
	seen := make(map[int64]struct{}, len(req.Absensi))
	for _, item := range req.Absensi {
		if _, ok := allowedSantri[item.SantriID]; !ok {
			return fmt.Errorf("santri %d tidak ditemukan di kelas ini", item.SantriID)
		}
		if _, ok := seen[item.SantriID]; ok {
			return fmt.Errorf("absensi santri %d tercatat lebih dari sekali", item.SantriID)
		}
		if _, ok := allowedStatus[item.Status]; !ok {
			return fmt.Errorf("status absensi %q tidak valid", item.Status)
		}
		batasMateri := strings.TrimSpace(item.BatasMateri)
		if kelas.MateriIndividual == 1 && (item.Status == "hadir" || item.Status == "telat") && batasMateri == "" {
			return fmt.Errorf("batas materi wajib diisi untuk santri yang %s", item.Status)
		}
		if item.Status != "hadir" && item.Status != "telat" {
			batasMateri = ""
		}
		seen[item.SantriID] = struct{}{}
		if err := querier.CreateOrUpdateAbsensi(ctx, queries.CreateOrUpdateAbsensiParams{
			PertemuanID: pertemuanID,
			SantriID:    item.SantriID,
			Status:      item.Status,
			Catatan:     item.Catatan,
			BatasMateri: batasMateri,
			DibuatOleh:  sql.NullInt64{Int64: userID, Valid: true},
		}); err != nil {
			return fmt.Errorf("gagal menyimpan absensi santri %d: %w", item.SantriID, err)
		}
	}

	if s.tagihan != nil {
		if err := s.tagihan.GenerateOnCompletionWithQuerier(querier, pertemuanID); err != nil {
			return err
		}
	}
	updated, err := querier.UpdatePertemuanSelesai(ctx, queries.UpdatePertemuanSelesaiParams{
		JamSelesai: time.Now().Format("15:04"),
		Materi:     req.Materi,
		Catatan:    req.Catatan,
		ID:         pertemuanID,
		KelasID:    kelasID,
	})
	if err != nil {
		return err
	}
	if updated == 0 {
		return fmt.Errorf("pertemuan gagal diselesaikan karena status telah berubah")
	}
	if err := querier.MarkJadwalPertemuanSelesai(ctx, sql.NullInt64{Int64: pertemuan.ID, Valid: true}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PertemuanService) GetPertemuanByID(pertemuanID, kelasID int64) (*models.PertemuanResponse, []models.AbsensiResponse, error) {
	p, err := s.querier.GetPertemuanByID(context.Background(), pertemuanID)
	if err != nil {
		return nil, nil, err
	}
	if p.KelasID != kelasID {
		return nil, nil, fmt.Errorf("pertemuan tidak ditemukan di kelas ini")
	}

	absensi, err := s.querier.GetAbsensiByPertemuan(context.Background(), pertemuanID)
	if err != nil {
		return nil, nil, err
	}

	absensiResp := make([]models.AbsensiResponse, 0, len(absensi))
	for _, a := range absensi {
		absensiResp = append(absensiResp, models.AbsensiResponse{
			ID:          a.ID,
			PertemuanID: a.PertemuanID,
			SantriID:    a.SantriID,
			SantriNama:  a.SantriNama,
			Status:      a.Status,
			Catatan:     a.Catatan,
			BatasMateri: a.BatasMateri,
		})
	}

	return mapPertemuanToResponse(p), absensiResp, nil
}

func (s *PertemuanService) ListRiwayat(kelasID int64) ([]models.PertemuanResponse, error) {
	list, err := s.querier.ListRiwayatPertemuanByKelas(context.Background(), kelasID)
	if err != nil {
		return nil, err
	}
	out := make([]models.PertemuanResponse, 0, len(list))
	for _, pe := range list {
		out = append(out, *mapPertemuanToResponse(pe))
	}
	return out, nil
}

func (s *PertemuanService) EnsureCanAccessPertemuan(pertemuanID, kelasID, userID int64, privileged bool) error {
	ctx := context.Background()
	pertemuan, err := s.querier.GetPertemuanByID(ctx, pertemuanID)
	if err != nil || pertemuan.KelasID != kelasID {
		return fmt.Errorf("pertemuan tidak ditemukan")
	}
	if privileged {
		return nil
	}
	guru, err := s.querier.GuruGetByUserID(ctx, sql.NullInt64{Int64: userID, Valid: true})
	if err != nil {
		return fmt.Errorf("data guru tidak ditemukan")
	}
	kelas, err := s.querier.GetKelasByID(ctx, kelasID)
	if err != nil {
		return fmt.Errorf("kelas tidak ditemukan")
	}
	isGuruUtama := kelas.GuruID.Valid && kelas.GuruID.Int64 == guru.ID
	isGuruBadal := pertemuan.GuruPenggantiID.Valid && pertemuan.GuruPenggantiID.Int64 == guru.ID
	if !isGuruUtama && !isGuruBadal {
		return fmt.Errorf("anda tidak memiliki akses ke pertemuan ini")
	}
	return nil
}

func (s *PertemuanService) GetRekapAbsensi(kelasID int64) ([]models.SantriGuruResponse, []models.PertemuanResponse, map[string]models.AbsensiResponse, error) {
	santriRows, err := s.querier.GetRekapAbsensiKelas(context.Background(), sql.NullInt64{Int64: kelasID, Valid: true})
	if err != nil {
		return nil, nil, nil, err
	}

	santri := make([]models.SantriGuruResponse, 0, len(santriRows))
	for _, sr := range santriRows {
		santri = append(santri, models.SantriGuruResponse{
			ID:           sr.SantriID,
			IDMahasantri: sr.IDMahasantri,
			Nama:         sr.SantriNama,
		})
	}

	pertemuanRows, err := s.querier.GetPertemuanByKelasForRekap(context.Background(), kelasID)
	if err != nil {
		return nil, nil, nil, err
	}

	pertemuan := make([]models.PertemuanResponse, 0, len(pertemuanRows))
	for _, pr := range pertemuanRows {
		pertemuan = append(pertemuan, models.PertemuanResponse{
			ID:               pr.ID,
			PertemuanKe:      pr.PertemuanKe,
			PertemuanLevelKe: pr.PertemuanLevelKe,
			LevelNama:        pr.LevelNama,
			Tanggal:          pr.Tanggal.Format("2006-01-02"),
		})
	}

	absensi := make(map[string]models.AbsensiResponse)
	for _, p := range pertemuan {
		rows, err := s.querier.GetAbsensiByPertemuan(context.Background(), p.ID)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, a := range rows {
			absensi[fmt.Sprintf("%d:%d", a.SantriID, p.ID)] = models.AbsensiResponse{ID: a.ID, PertemuanID: p.ID, SantriID: a.SantriID, Status: a.Status, Catatan: a.Catatan, BatasMateri: a.BatasMateri}
		}
	}

	for i := range santri {
		total := int64(0)
		for _, p := range pertemuan {
			absen, ok := absensi[fmt.Sprintf("%d:%d", santri[i].ID, p.ID)]
			if !ok {
				continue
			}
			total++
			switch absen.Status {
			case "hadir":
				santri[i].TotalHadir++
			case "izin":
				santri[i].TotalIzin++
			case "sakit":
				santri[i].TotalSakit++
			case "alpa":
				santri[i].TotalAlpa++
			case "telat":
				santri[i].TotalTelat++
			}
		}
		if total > 0 {
			santri[i].PersenHadir = float64(santri[i].TotalHadir) / float64(total) * 100
		}
	}

	return santri, pertemuan, absensi, nil
}

func (s *PertemuanService) GetNextPertemuanKe(kelasID int64) (int64, error) {
	return s.querier.GetNextPertemuanKe(context.Background(), kelasID)
}

// GetNextPertemuanLevelKe returns the next meeting number within the class's
// current level, which is what guru see after a level change.
func (s *PertemuanService) GetNextPertemuanLevelKe(kelasID int64) (int64, error) {
	return s.querier.GetNextPertemuanLevelKe(context.Background(), kelasID)
}

func (s *PertemuanService) EditAbsensi(kelasID, absensiID int64, status, catatan, batasMateri string) error {
	absensi, err := s.querier.GetAbsensiByID(context.Background(), absensiID)
	if err != nil {
		return err
	}
	pertemuan, err := s.querier.GetPertemuanByID(context.Background(), absensi.PertemuanID)
	if err != nil {
		return err
	}
	if pertemuan.KelasID != kelasID {
		return fmt.Errorf("absensi tidak ditemukan di kelas ini")
	}
	kelas, err := s.querier.GetKelasByID(context.Background(), kelasID)
	if err != nil {
		return err
	}
	batasMateri = strings.TrimSpace(batasMateri)
	if kelas.MateriIndividual == 1 && (status == "hadir" || status == "telat") && batasMateri == "" {
		return fmt.Errorf("batas materi wajib diisi untuk santri yang %s", status)
	}
	if status != "hadir" && status != "telat" {
		batasMateri = ""
	}
	return s.querier.UpdateAbsensi(context.Background(), queries.UpdateAbsensiParams{
		Status:      status,
		Catatan:     catatan,
		BatasMateri: batasMateri,
		ID:          absensiID,
	})
}

func mapPertemuanToResponse(p queries.Pertemuan) *models.PertemuanResponse {
	r := &models.PertemuanResponse{
		ID:               p.ID,
		KelasID:          p.KelasID,
		PertemuanKe:      p.PertemuanKe,
		PertemuanLevelKe: p.PertemuanLevelKe,
		LevelNama:        p.LevelNama,
		Tanggal:          p.Tanggal.Format("2006-01-02"),
		JamMulai:         p.JamMulai,
		JamSelesai:       p.JamSelesai,
		Materi:           p.Materi,
		Catatan:          p.Catatan,
		IsReschedule:     p.IsReschedule == 1,
		JadwalSemula:     p.JadwalSemula,
		AlasanReschedule: p.AlasanReschedule,
		IsBadal:          p.IsBadal == 1,
		AlasanBadal:      p.AlasanBadal,
		Status:           p.Status,
	}
	if p.GuruPenggantiID.Valid {
		r.GuruPenggantiID = &p.GuruPenggantiID.Int64
	}
	return r
}
