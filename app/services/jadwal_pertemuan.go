package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type JadwalPertemuanService struct {
	querier *queries.Querier
}

func NewJadwalPertemuanService(querier *queries.Querier) *JadwalPertemuanService {
	return &JadwalPertemuanService{querier: querier}
}

func (s *JadwalPertemuanService) List(guruID *int64) ([]models.JadwalPertemuanResponse, error) {
	filter := sql.NullInt64{}
	if guruID != nil {
		filter = sql.NullInt64{Int64: *guruID, Valid: true}
	}
	rows, err := s.querier.ListJadwalPertemuanForGuru(context.Background(), filter)
	if err != nil {
		return nil, err
	}

	out := make([]models.JadwalPertemuanResponse, 0, len(rows))
	for _, row := range rows {
		canManage := row.Status == "dijadwalkan" && (guruID == nil || (row.GuruUtamaID.Valid && row.GuruUtamaID.Int64 == *guruID))
		item := models.JadwalPertemuanResponse{
			ID:                 row.ID,
			KelasID:            row.KelasID,
			KelasNama:          row.NamaKelas,
			GuruUtamaNama:      row.GuruUtamaNama,
			Tanggal:            row.Tanggal.Format("2006-01-02"),
			JamMulai:           row.JamMulai,
			Catatan:            row.Catatan,
			IsReschedule:       row.IsReschedule == 1,
			JadwalSemula:       row.JadwalSemula,
			AlasanReschedule:   row.AlasanReschedule,
			JadwalKelasBerubah: row.JadwalKelasBerubah == 1,
			GuruPenggantiNama:  row.GuruPenggantiNama,
			AlasanBadal:        row.AlasanBadal,
			Status:             row.Status,
			CanManage:          canManage,
			CanStart:           row.Status == "dijadwalkan",
		}
		if row.GuruPenggantiID.Valid {
			item.GuruPenggantiID = &row.GuruPenggantiID.Int64
		}
		if row.GuruUtamaID.Valid {
			item.GuruUtamaID = &row.GuruUtamaID.Int64
		}
		if row.PertemuanID.Valid {
			item.PertemuanID = &row.PertemuanID.Int64
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *JadwalPertemuanService) ListDue(kelasID int64) ([]models.JadwalPertemuanResponse, error) {
	rows, err := s.querier.ListDueJadwalPertemuanByKelas(context.Background(), queries.ListDueJadwalPertemuanByKelasParams{
		KelasID: kelasID,
		Tanggal: startOfToday(),
	})
	if err != nil {
		return nil, err
	}

	out := make([]models.JadwalPertemuanResponse, 0, len(rows))
	for _, row := range rows {
		item := models.JadwalPertemuanResponse{
			ID:                 row.ID,
			KelasID:            row.KelasID,
			Tanggal:            row.Tanggal.Format("2006-01-02"),
			JamMulai:           row.JamMulai,
			Catatan:            row.Catatan,
			IsReschedule:       row.IsReschedule == 1,
			JadwalSemula:       row.JadwalSemula,
			AlasanReschedule:   row.AlasanReschedule,
			JadwalKelasBerubah: row.JadwalKelasBerubah == 1,
			GuruPenggantiNama:  row.GuruPenggantiNama,
			AlasanBadal:        row.AlasanBadal,
			Status:             row.Status,
			CanManage:          true,
			CanStart:           true,
		}
		if row.GuruPenggantiID.Valid {
			item.GuruPenggantiID = &row.GuruPenggantiID.Int64
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *JadwalPertemuanService) Create(userID int64, req models.BuatJadwalPertemuanRequest) (int64, error) {
	tanggal, err := parseScheduleDate(req.Tanggal)
	if err != nil {
		return 0, err
	}
	if err := validateScheduleTime(req.JamMulai); err != nil {
		return 0, err
	}
	if tanggal.Before(startOfToday()) {
		return 0, fmt.Errorf("tanggal jadwal tidak boleh di masa lalu")
	}
	return s.querier.CreateJadwalPertemuan(context.Background(), queries.CreateJadwalPertemuanParams{
		KelasID:    req.KelasID,
		Tanggal:    tanggal,
		JamMulai:   req.JamMulai,
		Catatan:    req.Catatan,
		DibuatOleh: sql.NullInt64{Int64: userID, Valid: true},
	})
}

func (s *JadwalPertemuanService) Reschedule(id, kelasID int64, req models.RescheduleRequest) error {
	return s.reschedule(context.Background(), s.querier, id, kelasID, req)
}

func (s *JadwalPertemuanService) reschedule(ctx context.Context, querier *queries.Querier, id, kelasID int64, req models.RescheduleRequest) error {
	tanggal, err := parseScheduleDate(req.TanggalBaru)
	if err != nil {
		return err
	}
	if err := validateScheduleTime(req.JamBaru); err != nil {
		return err
	}
	if tanggal.Before(startOfToday()) {
		return fmt.Errorf("tanggal jadwal tidak boleh di masa lalu")
	}

	jadwal, err := querier.GetJadwalPertemuanByID(ctx, queries.GetJadwalPertemuanByIDParams{ID: id, KelasID: kelasID})
	if err != nil {
		return fmt.Errorf("jadwal tidak ditemukan: %w", err)
	}
	if jadwal.Status != "dijadwalkan" {
		return fmt.Errorf("jadwal yang sudah dimulai atau dibatalkan tidak dapat diubah")
	}

	jadwalSemula := jadwal.JadwalSemula
	if jadwalSemula == "" {
		jadwalSemula = jadwal.Tanggal.Format("2006-01-02") + " " + jadwal.JamMulai
	}
	rows, err := querier.UpdateJadwalPertemuanReschedule(ctx, queries.UpdateJadwalPertemuanRescheduleParams{
		Tanggal:          tanggal,
		JamMulai:         req.JamBaru,
		JadwalSemula:     jadwalSemula,
		AlasanReschedule: req.Alasan,
		ID:               id,
		KelasID:          kelasID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("jadwal tidak lagi dapat diubah")
	}
	return nil
}

func (s *JadwalPertemuanService) Badal(id, kelasID int64, req models.BadalRequest) error {
	return s.badal(context.Background(), s.querier, id, kelasID, req)
}

func (s *JadwalPertemuanService) badal(ctx context.Context, querier *queries.Querier, id, kelasID int64, req models.BadalRequest) error {
	jadwal, err := querier.GetJadwalPertemuanByID(ctx, queries.GetJadwalPertemuanByIDParams{ID: id, KelasID: kelasID})
	if err != nil {
		return fmt.Errorf("jadwal tidak ditemukan: %w", err)
	}
	if jadwal.Status != "dijadwalkan" {
		return fmt.Errorf("jadwal yang sudah dimulai atau dibatalkan tidak dapat diubah")
	}

	guru, err := querier.GuruGetByID(ctx, req.GuruPenggantiID)
	if err != nil || guru.IsAktif != 1 || !guru.UserID.Valid {
		return fmt.Errorf("guru badal harus aktif dan terhubung ke akun pengguna")
	}
	kelas, err := querier.GetKelasByID(ctx, kelasID)
	if err != nil {
		return fmt.Errorf("kelas tidak ditemukan: %w", err)
	}
	if kelas.GuruID.Valid && kelas.GuruID.Int64 == guru.ID {
		return fmt.Errorf("guru utama tidak dapat ditetapkan sebagai guru badal")
	}

	rows, err := querier.UpdateJadwalPertemuanBadal(ctx, queries.UpdateJadwalPertemuanBadalParams{
		GuruPenggantiID: sql.NullInt64{Int64: guru.ID, Valid: true},
		AlasanBadal:     req.Alasan,
		ID:              id,
		KelasID:         kelasID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("jadwal tidak lagi dapat diubah")
	}
	return nil
}

func (s *JadwalPertemuanService) Cancel(id, kelasID int64) error {
	return s.cancel(context.Background(), s.querier, id, kelasID)
}

func (s *JadwalPertemuanService) cancel(ctx context.Context, querier *queries.Querier, id, kelasID int64) error {
	rows, err := querier.CancelJadwalPertemuan(ctx, queries.CancelJadwalPertemuanParams{ID: id, KelasID: kelasID})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("jadwal tidak ditemukan atau tidak lagi dapat dibatalkan")
	}
	return nil
}

func (s *JadwalPertemuanService) Start(id, kelasID, userID int64, privileged bool) (*models.PertemuanResponse, error) {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	querier := s.querier.WithTx(tx)

	jadwal, err := querier.GetJadwalPertemuanByID(ctx, queries.GetJadwalPertemuanByIDParams{ID: id, KelasID: kelasID})
	if err != nil {
		return nil, fmt.Errorf("jadwal tidak ditemukan: %w", err)
	}
	if jadwal.Status != "dijadwalkan" {
		return nil, fmt.Errorf("jadwal sudah dimulai atau dibatalkan")
	}
	if _, err := querier.GetActivePertemuanByKelas(ctx, kelasID); err == nil {
		return nil, ErrPertemuanBerlangsung
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if !privileged {
		if err := ensureScheduledMeetingAccess(ctx, querier, jadwal, userID); err != nil {
			return nil, err
		}
	}
	if jadwal.Tanggal.After(startOfToday()) {
		return nil, fmt.Errorf("pertemuan baru dapat dimulai pada tanggal yang dijadwalkan")
	}

	claimed, err := querier.ClaimJadwalPertemuan(ctx, queries.ClaimJadwalPertemuanParams{ID: id, KelasID: kelasID})
	if err != nil {
		return nil, err
	}
	if claimed == 0 {
		return nil, fmt.Errorf("jadwal sudah diproses oleh pengguna lain")
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
	result, err := querier.CreatePertemuan(ctx, queries.CreatePertemuanParams{
		KelasID:          kelasID,
		PertemuanKe:      nextKe,
		Tanggal:          now,
		JamMulai:         now.Format("15:04"),
		JamSelesai:       "",
		Materi:           "",
		Catatan:          jadwal.Catatan,
		IsReschedule:     jadwal.IsReschedule,
		JadwalSemula:     jadwal.JadwalSemula,
		AlasanReschedule: jadwal.AlasanReschedule,
		IsBadal:          scheduleBoolToInt64(jadwal.GuruPenggantiID.Valid),
		GuruPenggantiID:  jadwal.GuruPenggantiID,
		AlasanBadal:      jadwal.AlasanBadal,
		Status:           "berlangsung",
		DibuatOleh:       sql.NullInt64{Int64: userID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	pertemuanID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	if err := querier.CompleteJadwalPertemuanStart(ctx, queries.CompleteJadwalPertemuanStartParams{
		PertemuanID: sql.NullInt64{Int64: pertemuanID, Valid: true},
		ID:          id,
		KelasID:     kelasID,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &models.PertemuanResponse{
		ID:               pertemuanID,
		KelasID:          kelasID,
		PertemuanKe:      nextKe,
		PertemuanLevelKe: nextLevelKe,
		Tanggal:          now.Format("2006-01-02"),
		JamMulai:         now.Format("15:04"),
		IsReschedule:     jadwal.IsReschedule == 1,
		JadwalSemula:     jadwal.JadwalSemula,
		AlasanReschedule: jadwal.AlasanReschedule,
		IsBadal:          jadwal.GuruPenggantiID.Valid,
		GuruPenggantiID:  nullInt64Ptr(jadwal.GuruPenggantiID),
		AlasanBadal:      jadwal.AlasanBadal,
		Status:           "berlangsung",
	}, nil
}

func ensureScheduledMeetingAccess(ctx context.Context, querier *queries.Querier, jadwal queries.JadwalPertemuan, userID int64) error {
	guru, err := querier.GuruGetByUserID(ctx, sql.NullInt64{Int64: userID, Valid: true})
	if err != nil {
		return fmt.Errorf("data guru tidak ditemukan")
	}
	kelas, err := querier.GetKelasByID(ctx, jadwal.KelasID)
	if err != nil {
		return fmt.Errorf("kelas tidak ditemukan")
	}
	isGuruUtama := kelas.GuruID.Valid && kelas.GuruID.Int64 == guru.ID
	isGuruBadal := jadwal.GuruPenggantiID.Valid && jadwal.GuruPenggantiID.Int64 == guru.ID
	if !isGuruUtama && !isGuruBadal {
		return fmt.Errorf("anda tidak memiliki akses untuk memulai jadwal ini")
	}
	return nil
}

func parseScheduleDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("tanggal jadwal wajib diisi")
	}
	tanggal, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("format tanggal tidak valid")
	}
	return tanggal, nil
}

func validateScheduleTime(value string) error {
	if value == "" {
		return fmt.Errorf("jam mulai wajib diisi")
	}
	if _, err := time.Parse("15:04", value); err != nil {
		return fmt.Errorf("format jam mulai tidak valid")
	}
	return nil
}

func startOfToday() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func scheduleBoolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func nullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}
