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

// BatasAbsenSusulanHari is how many days back a guru may still fill in the
// attendance of a session they missed. Older sessions go through admin.
const BatasAbsenSusulanHari = 14

type JadwalPertemuanService struct {
	querier *queries.Querier
}

func NewJadwalPertemuanService(querier *queries.Querier) *JadwalPertemuanService {
	return &JadwalPertemuanService{querier: querier}
}

func (s *JadwalPertemuanService) EnsureRoutineOccurrences(start, end time.Time, kelasID *int64) error {
	if end.Before(start) {
		return fmt.Errorf("rentang jadwal rutin tidak valid")
	}
	loc := start.Location()
	start = routineDate(start, loc)
	end = routineDate(end, loc)

	filter := sql.NullInt64{}
	if kelasID != nil {
		filter = sql.NullInt64{Int64: *kelasID, Valid: true}
	}
	rows, err := s.querier.ListRoutineSchedulesForRange(context.Background(), start, end, filter)
	if err != nil {
		return err
	}

	for _, row := range rows {
		from := start
		activeFrom := routineDate(row.BerlakuMulai, loc)
		if activeFrom.After(from) {
			from = activeFrom
		}
		to := end
		if row.BerlakuSampai.Valid {
			activeUntil := routineDate(row.BerlakuSampai.Time, loc)
			if activeUntil.Before(to) {
				to = activeUntil
			}
		}
		if to.Before(from) {
			continue
		}

		delta := (int(row.Hari) - isoWeekday(from) + 7) % 7
		date := from.AddDate(0, 0, delta)
		for !date.After(to) {
			if _, err := s.querier.CreateRoutineOccurrence(context.Background(), row.ID, row.KelasID, date, row.JamMulai); err != nil {
				return err
			}
			date = date.AddDate(0, 0, 7)
		}
	}
	return nil
}

func (s *JadwalPertemuanService) EnsureRoutineOccurrencesForClass(kelasID int64, start, end time.Time) error {
	return s.EnsureRoutineOccurrences(start, end, &kelasID)
}

func routineDate(value time.Time, loc *time.Location) time.Time {
	parsed, err := time.ParseInLocation("2006-01-02", value.Format("2006-01-02"), loc)
	if err != nil {
		return value
	}
	return parsed
}

func isoWeekday(value time.Time) int {
	if value.Weekday() == time.Sunday {
		return 7
	}
	return int(value.Weekday())
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

	today := startOfToday()
	out := make([]models.JadwalPertemuanResponse, 0, len(rows))
	for _, row := range rows {
		canManage := row.Status == "dijadwalkan" && (guruID == nil || (row.GuruUtamaID.Valid && row.GuruUtamaID.Int64 == *guruID))
		lampau := row.Tanggal.Before(today)
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
			IsSusulan:          row.IsSusulan == 1,
			AlasanSusulan:      row.AlasanSusulan,
			Status:             row.Status,
			CanManage:          canManage,
			CanStart:           row.Status == "dijadwalkan" && !lampau,
			// guruID nil = admin kelas/super admin, yang tidak dibatasi jendela susulan.
			CanSusulan: row.Status == "dijadwalkan" && lampau && (guruID == nil || withinSusulanWindow(row.Tanggal, today)),
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

	today := startOfToday()
	out := make([]models.JadwalPertemuanResponse, 0, len(rows))
	for _, row := range rows {
		// Sesi yang tanggalnya sudah lewat diisi lewat absen susulan agar
		// pertemuannya tercatat di tanggal jadwal, bukan hari ini.
		lampau := row.Tanggal.Before(today)
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
			CanStart:           !lampau,
			CanSusulan:         lampau && withinSusulanWindow(row.Tanggal, today),
		}
		if row.GuruPenggantiID.Valid {
			item.GuruPenggantiID = &row.GuruPenggantiID.Int64
		}
		out = append(out, item)
	}
	return out, nil
}

// ListDibatalkan returns the class's cancelled sessions, newest first, for the
// meeting history.
func (s *JadwalPertemuanService) ListDibatalkan(kelasID int64) ([]models.JadwalDibatalkanItem, error) {
	rows, err := s.querier.ListJadwalDibatalkanByKelas(context.Background(), kelasID)
	if err != nil {
		return nil, err
	}
	out := make([]models.JadwalDibatalkanItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.JadwalDibatalkanItem{
			ID:                 row.ID,
			Tanggal:            row.Tanggal.Format("2006-01-02"),
			JamMulai:           row.JamMulai,
			AlasanBatal:        row.AlasanBatal,
			DibatalkanOleh:     row.DibatalkanOleh,
			DibatalkanOlehRole: row.DibatalkanOlehRole,
			DibatalkanPada:     row.UpdatedAt.In(wib).Format("2006-01-02 15:04"),
		})
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
	id, err := s.querier.CreateExtraSchedule(
		context.Background(),
		req.KelasID,
		tanggal,
		req.JamMulai,
		req.Catatan,
		sql.NullInt64{Int64: userID, Valid: true},
	)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("sudah ada jadwal aktif untuk kelas ini pada tanggal dan jam yang sama")
	}
	return id, err
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

// Cancel is the guru's cancellation: the reason is required, kept on the
// schedule and in its activity log, and every koordinator is notified.
func (s *JadwalPertemuanService) Cancel(id, kelasID, actorID int64, alasan string) error {
	alasan = strings.TrimSpace(alasan)
	if alasan == "" {
		return fmt.Errorf("alasan pembatalan wajib diisi")
	}
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)

	if err := s.cancel(ctx, q, id, kelasID, alasan); err != nil {
		return err
	}
	jadwal, err := q.GetJadwalPertemuanByID(ctx, queries.GetJadwalPertemuanByIDParams{ID: id, KelasID: kelasID})
	if err != nil {
		return err
	}
	if err := logScheduleActivity(ctx, q, actorID, id, "jadwal_dibatalkan", alasan); err != nil {
		return err
	}
	kelas, err := q.GetKelasByID(ctx, kelasID)
	if err != nil {
		return err
	}
	message := fmt.Sprintf("Kelas %s pada %s pukul %s dibatalkan. Alasan: %s", kelas.NamaKelas, jadwal.Tanggal.Format("02/01/2006"), jadwal.JamMulai, alasan)
	if err := notifyKoordinator(ctx, q, actorID, id, "jadwal_dibatalkan", "Kelas dibatalkan guru", message); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *JadwalPertemuanService) cancel(ctx context.Context, querier *queries.Querier, id, kelasID int64, alasan string) error {
	rows, err := querier.CancelJadwalPertemuan(ctx, queries.CancelJadwalPertemuanParams{AlasanBatal: alasan, ID: id, KelasID: kelasID})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("jadwal tidak ditemukan atau tidak lagi dapat dibatalkan")
	}
	return nil
}

func (s *JadwalPertemuanService) Start(id, kelasID, userID int64, privileged bool) (*models.PertemuanResponse, error) {
	return s.start(id, kelasID, userID, privileged, nil)
}

// StartSusulan opens the meeting of a session whose date has passed without
// it being started, so the guru can still fill in attendance. The meeting
// keeps the scheduled date and time; the schedule is labelled susulan with
// the guru's reason, logged, and the koordinator is notified.
func (s *JadwalPertemuanService) StartSusulan(id, kelasID, userID int64, privileged bool, alasan string) (*models.PertemuanResponse, error) {
	alasan = strings.TrimSpace(alasan)
	if alasan == "" {
		return nil, fmt.Errorf("alasan absen susulan wajib diisi")
	}
	return s.start(id, kelasID, userID, privileged, &alasan)
}

func (s *JadwalPertemuanService) start(id, kelasID, userID int64, privileged bool, alasanSusulan *string) (*models.PertemuanResponse, error) {
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
	today := startOfToday()
	if jadwal.Tanggal.After(today) {
		return nil, fmt.Errorf("pertemuan baru dapat dimulai pada tanggal yang dijadwalkan")
	}
	susulan := alasanSusulan != nil
	lampau := jadwal.Tanggal.Before(today)
	if lampau && !susulan {
		return nil, fmt.Errorf("tanggal jadwal sudah lewat, gunakan \"Isi absen susulan\" di Jadwal Pertemuan")
	}
	if susulan {
		if !lampau {
			return nil, fmt.Errorf("absen susulan hanya untuk jadwal yang tanggalnya sudah lewat")
		}
		if !privileged && !withinSusulanWindow(jadwal.Tanggal, today) {
			return nil, fmt.Errorf("absen susulan hanya dapat diisi maksimal %d hari setelah jadwal, hubungi koordinator atau admin kelas", BatasAbsenSusulanHari)
		}
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
	tanggal, jamMulai := now, now.Format("15:04")
	if susulan {
		tanggal, jamMulai = jadwal.Tanggal, jadwal.JamMulai
	}
	result, err := querier.CreatePertemuan(ctx, queries.CreatePertemuanParams{
		KelasID:          kelasID,
		PertemuanKe:      nextKe,
		Tanggal:          tanggal,
		JamMulai:         jamMulai,
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
	if susulan {
		if err := s.markSusulan(ctx, querier, jadwal, userID, *alasanSusulan); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &models.PertemuanResponse{
		ID:               pertemuanID,
		KelasID:          kelasID,
		PertemuanKe:      nextKe,
		PertemuanLevelKe: nextLevelKe,
		Tanggal:          tanggal.Format("2006-01-02"),
		JamMulai:         jamMulai,
		IsReschedule:     jadwal.IsReschedule == 1,
		JadwalSemula:     jadwal.JadwalSemula,
		AlasanReschedule: jadwal.AlasanReschedule,
		IsBadal:          jadwal.GuruPenggantiID.Valid,
		GuruPenggantiID:  nullInt64Ptr(jadwal.GuruPenggantiID),
		AlasanBadal:      jadwal.AlasanBadal,
		Status:           "berlangsung",
	}, nil
}

func (s *JadwalPertemuanService) markSusulan(ctx context.Context, q *queries.Querier, jadwal queries.JadwalPertemuan, userID int64, alasan string) error {
	if err := q.MarkJadwalPertemuanSusulan(ctx, queries.MarkJadwalPertemuanSusulanParams{AlasanSusulan: alasan, ID: jadwal.ID, KelasID: jadwal.KelasID}); err != nil {
		return err
	}
	if err := logScheduleActivity(ctx, q, userID, jadwal.ID, "absen_susulan", alasan); err != nil {
		return err
	}
	kelas, err := q.GetKelasByID(ctx, jadwal.KelasID)
	if err != nil {
		return err
	}
	message := fmt.Sprintf("Absen kelas %s untuk jadwal %s pukul %s diisi susulan. Alasan: %s", kelas.NamaKelas, jadwal.Tanggal.Format("02/01/2006"), jadwal.JamMulai, alasan)
	return notifyKoordinator(ctx, q, userID, jadwal.ID, "absen_susulan", "Absen susulan diisi guru", message)
}

// withinSusulanWindow reports whether a past session date is still inside the
// window in which the guru may fill in attendance late.
func withinSusulanWindow(tanggal, today time.Time) bool {
	batas := today.AddDate(0, 0, -BatasAbsenSusulanHari)
	return !routineDate(tanggal, today.Location()).Before(batas)
}

// notifyKoordinator sends an in-app notification about a schedule to every
// koordinator guru, linking to the monitoring page.
func notifyKoordinator(ctx context.Context, q *queries.Querier, actorID, scheduleID int64, kind, title, message string) error {
	ids, err := q.ListKoordinatorUserIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == actorID {
			continue
		}
		if _, err := q.CreateNotification(ctx, queries.CreateNotificationParams{
			UserID:        id,
			Type:          kind,
			Title:         title,
			Message:       message,
			ActionUrl:     "/app/koordinator-guru/monitoring-kelas",
			ReferenceType: "jadwal_pertemuan",
			ReferenceID:   sql.NullInt64{Int64: scheduleID, Valid: true},
			CreatedBy:     sql.NullInt64{Int64: actorID, Valid: true},
		}); err != nil {
			return err
		}
	}
	return nil
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
