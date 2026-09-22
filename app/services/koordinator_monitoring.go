package services

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type KoordinatorMonitoringService struct {
	querier       *queries.Querier
	jadwalService *JadwalPertemuanService
}

func NewKoordinatorMonitoringService(querier *queries.Querier, jadwalService *JadwalPertemuanService) *KoordinatorMonitoringService {
	return &KoordinatorMonitoringService{querier: querier, jadwalService: jadwalService}
}

func (s *KoordinatorMonitoringService) List(filter models.ClassMonitoringFilter) (*models.ClassMonitoringResponse, error) {
	start, end, err := monitoringDateRange(filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, err
	}
	rows, err := s.querier.ListClassMonitoringSchedules(context.Background(), queries.ListClassMonitoringSchedulesParams{
		StartDate: start,
		EndDate:   end,
	})
	if err != nil {
		return nil, err
	}
	unscheduled, err := s.querier.ListUnscheduledMonitoringMeetings(context.Background(), queries.ListUnscheduledMonitoringMeetingsParams{
		StartDate: start,
		EndDate:   end,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	items := make([]models.ClassMonitoringItem, 0, len(rows)+len(unscheduled))
	for _, row := range rows {
		items = append(items, mapClassMonitoringItem(row, now))
	}
	for _, row := range unscheduled {
		items = append(items, mapUnscheduledMonitoringItem(row))
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Tanggal != items[j].Tanggal {
			return items[i].Tanggal < items[j].Tanggal
		}
		if items[i].JamMulai != items[j].JamMulai {
			return items[i].JamMulai < items[j].JamMulai
		}
		return items[i].NamaKelas < items[j].NamaKelas
	})

	response := &models.ClassMonitoringResponse{Items: make([]models.ClassMonitoringItem, 0, len(items))}
	for _, item := range items {
		if filter.GuruID > 0 && !monitoringMatchesGuru(item, filter.GuruID) {
			continue
		}
		if filter.KelasID > 0 && item.KelasID != filter.KelasID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if err := s.enrichMonitoringItem(&item); err != nil {
			return nil, err
		}
		response.Items = append(response.Items, item)
		updateMonitoringSummary(&response.Summary, item)
	}
	return response, nil
}

func (s *KoordinatorMonitoringService) SendReminder(actorID, scheduleID, kelasID int64, req models.MonitoringReminderRequest) error {
	kind := strings.TrimSpace(req.Kind)
	if kind != "presensi_guru" && kind != "absensi_peserta" {
		return fmt.Errorf("jenis pengingat tidak valid")
	}
	target, err := s.querier.GetMonitoringReminderTarget(context.Background(), queries.GetMonitoringReminderTargetParams{
		ID:      scheduleID,
		KelasID: kelasID,
	})
	if err != nil {
		return fmt.Errorf("jadwal tidak ditemukan")
	}
	if target.Status == "dibatalkan" {
		return fmt.Errorf("jadwal sudah dibatalkan")
	}
	if kind == "presensi_guru" && target.Status != "dijadwalkan" {
		return fmt.Errorf("pertemuan sudah dimulai, pengingat presensi tidak diperlukan")
	}
	if !target.AssignedUserID.Valid {
		return fmt.Errorf("guru yang bertugas belum terhubung ke akun pengguna")
	}

	title := "Pengingat presensi kelas"
	defaultMessage := fmt.Sprintf("Mohon segera melakukan presensi untuk kelas %s pada %s pukul %s.", target.NamaKelas, target.Tanggal.Format("02/01/2006"), target.JamMulai)
	if kind == "absensi_peserta" {
		title = "Pengingat absensi peserta"
		defaultMessage = fmt.Sprintf("Mohon lengkapi absensi peserta kelas %s pada %s.", target.NamaKelas, target.Tanggal.Format("02/01/2006"))
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = defaultMessage
	}

	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)
	recent, err := q.CountRecentScheduleReminders(ctx, queries.CountRecentScheduleRemindersParams{
		ScheduleID: sql.NullInt64{Int64: scheduleID, Valid: true},
		Kind:       kind,
	})
	if err != nil {
		return err
	}
	if recent > 0 {
		return fmt.Errorf("pengingat yang sama baru saja dikirim, coba lagi beberapa menit lagi")
	}
	if _, err := q.CreateNotification(ctx, queries.CreateNotificationParams{
		UserID:        target.AssignedUserID.Int64,
		Type:          kind,
		Title:         title,
		Message:       message,
		ActionUrl:     "/app/guru/jadwal-pertemuan",
		ReferenceType: "jadwal_pertemuan",
		ReferenceID:   sql.NullInt64{Int64: scheduleID, Valid: true},
		CreatedBy:     sql.NullInt64{Int64: actorID, Valid: true},
	}); err != nil {
		return err
	}
	if err := q.CreateScheduleActivityLog(ctx, queries.CreateScheduleActivityLogParams{
		JadwalPertemuanID: scheduleID,
		Action:            "pengingat_dikirim",
		Details:           title + " kepada " + target.AssignedTeacherName,
		CreatedBy:         sql.NullInt64{Int64: actorID, Valid: true},
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *KoordinatorMonitoringService) AddNote(actorID, scheduleID, kelasID int64, req models.MonitoringNoteRequest) error {
	note := strings.TrimSpace(req.Note)
	if note == "" {
		return fmt.Errorf("catatan tindak lanjut wajib diisi")
	}
	ctx := context.Background()
	if _, err := s.querier.GetJadwalPertemuanByID(ctx, queries.GetJadwalPertemuanByIDParams{ID: scheduleID, KelasID: kelasID}); err != nil {
		return fmt.Errorf("jadwal tidak ditemukan")
	}
	_, err := s.querier.CreateClassMonitoringNote(ctx, queries.CreateClassMonitoringNoteParams{
		JadwalPertemuanID: scheduleID,
		Note:              note,
		CreatedBy:         sql.NullInt64{Int64: actorID, Valid: true},
	})
	return err
}

// Reschedule, AssignSubstitute and Cancel change the schedule and write the
// activity log in one transaction, so a failed log never leaves a changed
// schedule behind an error message.
func (s *KoordinatorMonitoringService) Reschedule(actorID, scheduleID, kelasID int64, req models.RescheduleRequest) error {
	return s.withScheduleTx(func(ctx context.Context, q *queries.Querier) error {
		if err := s.jadwalService.reschedule(ctx, q, scheduleID, kelasID, req); err != nil {
			return err
		}
		return logScheduleActivity(ctx, q, actorID, scheduleID, "jadwal_diubah", fmt.Sprintf("Jadwal diubah ke %s pukul %s. %s", req.TanggalBaru, req.JamBaru, strings.TrimSpace(req.Alasan)))
	})
}

func (s *KoordinatorMonitoringService) AssignSubstitute(actorID, scheduleID, kelasID int64, req models.BadalRequest) error {
	return s.withScheduleTx(func(ctx context.Context, q *queries.Querier) error {
		if err := s.jadwalService.badal(ctx, q, scheduleID, kelasID, req); err != nil {
			return err
		}
		return logScheduleActivity(ctx, q, actorID, scheduleID, "guru_pengganti_ditetapkan", strings.TrimSpace(req.Alasan))
	})
}

func (s *KoordinatorMonitoringService) Cancel(actorID, scheduleID, kelasID int64, req models.MonitoringCancelRequest) error {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return fmt.Errorf("alasan pembatalan wajib diisi")
	}
	return s.withScheduleTx(func(ctx context.Context, q *queries.Querier) error {
		if err := s.jadwalService.cancel(ctx, q, scheduleID, kelasID); err != nil {
			return err
		}
		return logScheduleActivity(ctx, q, actorID, scheduleID, "jadwal_dibatalkan", reason)
	})
}

func (s *KoordinatorMonitoringService) withScheduleTx(fn func(ctx context.Context, q *queries.Querier) error) error {
	ctx := context.Background()
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(ctx, s.querier.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

func logScheduleActivity(ctx context.Context, q *queries.Querier, actorID, scheduleID int64, action, details string) error {
	return q.CreateScheduleActivityLog(ctx, queries.CreateScheduleActivityLogParams{
		JadwalPertemuanID: scheduleID,
		Action:            action,
		Details:           strings.TrimSpace(details),
		CreatedBy:         sql.NullInt64{Int64: actorID, Valid: true},
	})
}

func (s *KoordinatorMonitoringService) enrichMonitoringItem(item *models.ClassMonitoringItem) error {
	ctx := context.Background()
	if item.PertemuanID != nil {
		rows, err := s.querier.ListMonitoringAttendanceByMeeting(ctx, queries.ListMonitoringAttendanceByMeetingParams{
			PertemuanID: *item.PertemuanID,
			KelasID:     sql.NullInt64{Int64: item.KelasID, Valid: true},
		})
		if err != nil {
			return err
		}
		item.Attendance = make([]models.MonitoringAttendance, 0, len(rows))
		for _, row := range rows {
			item.Attendance = append(item.Attendance, models.MonitoringAttendance{
				SantriID:         row.SantriID,
				SantriNama:       row.SantriNama,
				IDMahasantri:     row.IDMahasantri,
				AttendanceStatus: row.AttendanceStatus,
				AttendanceNote:   row.AttendanceNote,
				BatasMateri:      row.BatasMateri,
			})
		}
	}

	// Catatan dan riwayat terikat ke jadwal_pertemuan; pertemuan tanpa jadwal tidak memilikinya.
	if item.TanpaJadwal {
		return nil
	}

	notes, err := s.querier.ListClassMonitoringNotes(ctx, item.ScheduleID)
	if err != nil {
		return err
	}
	item.Notes = make([]models.MonitoringNote, 0, len(notes))
	for _, row := range notes {
		item.Notes = append(item.Notes, models.MonitoringNote{ID: row.ID, Note: row.Note, AuthorName: row.AuthorName, CreatedAt: row.CreatedAt.In(wib).Format("2006-01-02 15:04")})
	}

	activities, err := s.querier.ListScheduleActivityLogs(ctx, item.ScheduleID)
	if err != nil {
		return err
	}
	item.Activities = make([]models.ScheduleActivity, 0, len(activities))
	for _, row := range activities {
		item.Activities = append(item.Activities, models.ScheduleActivity{ID: row.ID, Action: row.Action, Details: row.Details, ActorName: row.ActorName, CreatedAt: row.CreatedAt.In(wib).Format("2006-01-02 15:04")})
	}
	return nil
}

func mapClassMonitoringItem(row queries.ListClassMonitoringSchedulesRow, now time.Time) models.ClassMonitoringItem {
	scheduledAt, _ := time.ParseInLocation("2006-01-02 15:04", row.Tanggal.Format("2006-01-02")+" "+row.JamMulai, wib)
	status := "belum_mulai"
	switch {
	case row.ScheduleStatus == "dibatalkan":
		status = "dibatalkan"
	case row.ScheduleStatus == "selesai" || row.MeetingStatus == "selesai":
		status = "selesai"
	case row.ScheduleStatus == "dimulai" || row.MeetingStatus == "berlangsung" || row.MeetingStatus == "menyelesaikan":
		status = "berlangsung"
	case !scheduledAt.After(now):
		status = "terlambat"
	}

	teacherAttendance := "belum_hadir"
	checkedInAt := ""
	if row.TeacherCheckedInAt.Valid {
		checkedInAt = row.TeacherCheckedInAt.Time.In(wib).Format("2006-01-02 15:04")
		teacherAttendance = "hadir"
		if row.TeacherCheckedInAt.Time.After(scheduledAt) {
			teacherAttendance = "terlambat"
		}
	}
	studentAttendance := "belum_tersedia"
	if row.PertemuanID.Valid {
		studentAttendance = "belum_lengkap"
		if row.AttendanceCount >= row.ActiveStudentCount {
			studentAttendance = "lengkap"
		}
	}
	if status == "dibatalkan" {
		teacherAttendance = "tidak_berlaku"
		studentAttendance = "tidak_berlaku"
	}

	item := models.ClassMonitoringItem{
		ScheduleID:         row.ScheduleID,
		KelasID:            row.KelasID,
		NamaKelas:          row.NamaKelas,
		Angkatan:           row.Angkatan,
		Level:              row.Level,
		Frekuensi:          row.Frekuensi,
		JadwalKelas:        row.JadwalKelas,
		MateriIndividual:   row.MateriIndividual == 1,
		Tanggal:            row.Tanggal.Format("2006-01-02"),
		JamMulai:           row.JamMulai,
		ScheduleNote:       row.ScheduleNote,
		Status:             status,
		ScheduleStatus:     row.ScheduleStatus,
		IsReschedule:       row.IsReschedule == 1,
		JadwalSemula:       row.JadwalSemula,
		AlasanReschedule:   row.AlasanReschedule,
		GuruUtamaNama:      row.GuruUtamaNama,
		GuruPenggantiNama:  row.GuruPenggantiNama,
		AlasanBadal:        row.AlasanBadal,
		ActualStartTime:    row.ActualStartTime,
		ActualEndTime:      row.ActualEndTime,
		TeacherCheckedInAt: checkedInAt,
		TeacherAttendance:  teacherAttendance,
		StudentAttendance:  studentAttendance,
		ActiveStudentCount: row.ActiveStudentCount,
		AttendanceCount:    row.AttendanceCount,
		Materi:             row.Materi,
		MeetingNote:        row.MeetingNote,
		NeedsAction:        status == "terlambat" || (status == "selesai" && studentAttendance != "lengkap"),
		Attendance:         []models.MonitoringAttendance{},
		Notes:              []models.MonitoringNote{},
		Activities:         []models.ScheduleActivity{},
	}
	item.GuruUtamaID = nullInt64Ptr(row.GuruUtamaID)
	item.GuruPenggantiID = nullInt64Ptr(row.GuruPenggantiID)
	item.AssignedUserID = nullInt64Ptr(row.AssignedUserID)
	item.PertemuanID = nullInt64Ptr(row.PertemuanID)
	return item
}

func mapUnscheduledMonitoringItem(row queries.ListUnscheduledMonitoringMeetingsRow) models.ClassMonitoringItem {
	status := "berlangsung"
	if row.MeetingStatus == "selesai" {
		status = "selesai"
	}
	studentAttendance := "belum_lengkap"
	if row.AttendanceCount >= row.ActiveStudentCount {
		studentAttendance = "lengkap"
	}
	pertemuanID := row.PertemuanID
	item := models.ClassMonitoringItem{
		KelasID:            row.KelasID,
		TanpaJadwal:        true,
		NamaKelas:          row.NamaKelas,
		Angkatan:           row.Angkatan,
		Level:              row.Level,
		Frekuensi:          row.Frekuensi,
		JadwalKelas:        row.JadwalKelas,
		MateriIndividual:   row.MateriIndividual == 1,
		Tanggal:            row.Tanggal.Format("2006-01-02"),
		JamMulai:           row.JamMulai,
		Status:             status,
		ScheduleStatus:     "tanpa_jadwal",
		GuruUtamaNama:      row.GuruUtamaNama,
		GuruPenggantiNama:  row.GuruPenggantiNama,
		AlasanBadal:        row.AlasanBadal,
		PertemuanID:        &pertemuanID,
		ActualStartTime:    row.Tanggal.Format("2006-01-02") + " " + row.JamMulai,
		TeacherCheckedInAt: row.TeacherCheckedInAt.In(wib).Format("2006-01-02 15:04"),
		TeacherAttendance:  "hadir",
		StudentAttendance:  studentAttendance,
		ActiveStudentCount: row.ActiveStudentCount,
		AttendanceCount:    row.AttendanceCount,
		Materi:             row.Materi,
		MeetingNote:        row.MeetingNote,
		NeedsAction:        status == "selesai" && studentAttendance != "lengkap",
		Attendance:         []models.MonitoringAttendance{},
		Notes:              []models.MonitoringNote{},
		Activities:         []models.ScheduleActivity{},
	}
	if row.JamSelesai != "" {
		item.ActualEndTime = row.Tanggal.Format("2006-01-02") + " " + row.JamSelesai
	}
	item.GuruUtamaID = nullInt64Ptr(row.GuruUtamaID)
	item.GuruPenggantiID = nullInt64Ptr(row.GuruPenggantiID)
	return item
}

func monitoringDateRange(startValue, endValue string) (time.Time, time.Time, error) {
	if startValue == "" {
		startValue = time.Now().In(wib).Format("2006-01-02")
	}
	if endValue == "" {
		endValue = startValue
	}
	start, err := time.ParseInLocation("2006-01-02", startValue, wib)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("tanggal awal tidak valid")
	}
	end, err := time.ParseInLocation("2006-01-02", endValue, wib)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("tanggal akhir tidak valid")
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("tanggal akhir tidak boleh sebelum tanggal awal")
	}
	if end.Sub(start) > 31*24*time.Hour {
		return time.Time{}, time.Time{}, fmt.Errorf("rentang monitoring maksimal 31 hari")
	}
	return start, end, nil
}

func monitoringMatchesGuru(item models.ClassMonitoringItem, guruID int64) bool {
	return (item.GuruUtamaID != nil && *item.GuruUtamaID == guruID) || (item.GuruPenggantiID != nil && *item.GuruPenggantiID == guruID)
}

func updateMonitoringSummary(summary *models.ClassMonitoringSummary, item models.ClassMonitoringItem) {
	summary.Total++
	switch item.Status {
	case "belum_mulai":
		summary.BelumMulai++
	case "terlambat":
		summary.BelumMulai++
	case "berlangsung":
		summary.Berlangsung++
	case "selesai":
		summary.Selesai++
	case "dibatalkan":
		summary.Dibatalkan++
	}
	if item.NeedsAction {
		summary.PerluTindakan++
	}
}
