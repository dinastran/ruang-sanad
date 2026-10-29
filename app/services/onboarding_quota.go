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

// Class phase is computed from the immutable first-candidate date in WIB.
// These are class phases, not the individual santri SLA or manual Kanban stages.
func OnboardingClassPhase(firstCandidate, now time.Time) (int, string) {
	a := firstCandidate.In(onboardingWIB)
	b := now.In(onboardingWIB)
	start := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, onboardingWIB)
	today := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, onboardingWIB)
	day := int(today.Sub(start).Hours() / 24)
	if day < 0 {
		day = 0
	}
	switch {
	case day <= 3:
		return day, "Matching"
	case day <= 7:
		return day, "Fill"
	case day <= 10:
		return day, "Push"
	case day <= 13:
		return day, "Resolve"
	case day == 14:
		return day, "Start"
	default:
		return day, "Overdue"
	}
}

func onboardingIsPrivate(tipe string) bool {
	return strings.EqualFold(strings.TrimSpace(tipe), "Private")
}
func suggestedQuota(tipe string, capacity int64) int64 {
	suggestion := int64(10)
	if onboardingIsPrivate(tipe) {
		suggestion = 1
	}
	if capacity < suggestion {
		suggestion = capacity
	}
	return suggestion
}
func quotaRowToModel(r queries.OnboardingClassControlRow, now time.Time) models.OnboardingQuotaClass {
	anchor := r.FirstCaseAt
	if r.FirstCandidateAt.Valid {
		anchor = r.FirstCandidateAt.Time
	}
	day, phase := OnboardingClassPhase(anchor, now)
	private := onboardingIsPrivate(r.Tipe)
	target := r.TargetKuota.Int64
	gap := target - r.JumlahSantri
	if gap < 0 {
		gap = 0
	}
	need := !private && target > 0 && r.JumlahSantri < target
	approved := r.ExceptionApprovedAt.Valid
	ready := r.IsAktif == 1 && r.GuruID.Valid && strings.TrimSpace(r.Jadwal) != "" &&
		target > 0 && target <= r.Kapasitas && (r.JumlahSantri >= target || approved)
	out := models.OnboardingQuotaClass{
		KelasID: r.KelasID, Nama: r.KelasNama, Tipe: r.Tipe, Private: private,
		Kapasitas: r.Kapasitas, JumlahSantri: r.JumlahSantri, KandidatAktif: r.KandidatAktif,
		TargetKuota: target, SuggestedTarget: suggestedQuota(r.Tipe, r.Kapasitas), Kekurangan: gap,
		FirstCandidateAt: anchor.In(onboardingWIB).Format(time.RFC3339),
		Day:              day, Phase: phase, VacancyRequired: need, CanSendAlert: need && day >= 4 && !r.VacancyAlertSentAt.Valid,
		ExceptionApproved: approved, ExceptionReason: r.ExceptionReason,
		ExceptionApprovedBy: r.ExceptionApprovedBy.Int64,
		GuruConfirmed:       r.GuruID.Valid, JadwalConfirmed: strings.TrimSpace(r.Jadwal) != "",
		IsActive: r.IsAktif == 1, CanStart: ready,
	}
	if r.VacancyAlertSentAt.Valid {
		out.VacancySentAt = r.VacancyAlertSentAt.Time.In(onboardingWIB).Format(time.RFC3339)
	}
	if approved {
		out.ExceptionApprovedAt = r.ExceptionApprovedAt.Time.In(onboardingWIB).Format(time.RFC3339)
	}
	if need {
		out.VacancyText = fmt.Sprintf("VACANCY ALERT — RUANG SANAD\nKelas: %s\nProgram: %s\nKuota operasional: %d/%d (maksimum %d)\nKekurangan: %d santri\nFase: %s (D%d)\nMohon koordinasi pengisian kuota melalui CS dan Marketing.", r.KelasNama, r.Tipe, r.JumlahSantri, target, r.Kapasitas, gap, phase, day)
	}
	return out
}

func (s *OnboardingService) ClassControls(ctx context.Context) (models.OnboardingQuotaBoard, error) {
	rows, err := s.querier.ListOnboardingClassControls(ctx)
	if err != nil {
		return models.OnboardingQuotaBoard{}, err
	}
	items := make([]models.OnboardingQuotaClass, 0, len(rows))
	now := time.Now().UTC()
	for _, r := range rows {
		items = append(items, quotaRowToModel(r, now))
	}
	return models.OnboardingQuotaBoard{Classes: items}, nil
}
func (s *OnboardingService) ClassActions(ctx context.Context, id int64) ([]models.OnboardingQuotaAction, error) {
	if id <= 0 {
		return nil, ErrOnboardingNotFound
	}
	if _, err := s.querier.GetOnboardingClassControl(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOnboardingNotFound
		}
		return nil, err
	}
	rows, err := s.querier.ListOnboardingClassActions(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]models.OnboardingQuotaAction, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.OnboardingQuotaAction{
			ID: r.ID, EventType: r.EventType, Note: r.Note, ActorName: r.ActorName,
			CreatedAt: r.CreatedAt.In(onboardingWIB).Format(time.RFC3339),
		})
	}
	return out, nil
}

func (s *OnboardingService) verifyClassActor(ctx context.Context, q *queries.Querier, actorID int64, roles ...string) error {
	if actorID <= 0 {
		return onboardingInputError("aktor tidak valid")
	}
	role, err := q.GetOnboardingPICRole(ctx, actorID)
	if errors.Is(err, sql.ErrNoRows) {
		return onboardingInputError("aktor tidak ditemukan")
	}
	if err != nil {
		return err
	}
	for _, allowed := range roles {
		if role == allowed {
			return nil
		}
	}
	return onboardingInputError("akses tidak diizinkan untuk peran ini")
}

func (s *OnboardingService) SetClassQuota(ctx context.Context, kelasID, actorID, target int64) error {
	if kelasID <= 0 || target <= 0 {
		return onboardingInputError("kelas dan target kuota wajib valid")
	}
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)
	if err := s.verifyClassActor(ctx, q, actorID, "super_admin"); err != nil {
		return err
	}
	r, err := q.GetOnboardingClassControl(ctx, kelasID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOnboardingNotFound
	}
	if err != nil {
		return err
	}
	if r.IsAktif != 1 || target > r.Kapasitas {
		return onboardingInputError("target harus positif, tidak melebihi kapasitas, dan kelas aktif")
	}
	if r.TargetKuota.Valid && r.TargetKuota.Int64 == target {
		return onboardingInputError("target kuota tidak berubah")
	}
	now := time.Now().UTC()
	if !r.TargetKuota.Valid {
		if err := q.CreateOnboardingQuota(ctx, kelasID, target, r.FirstCaseAt, now); err != nil {
			return err
		}
	} else {
		affected, err := q.ChangeOnboardingQuota(ctx, kelasID, target, now)
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrOnboardingConflict
		}
	}
	note := fmt.Sprintf("Target kuota %d → %d dari kapasitas %d; exception dan status alert lama dibatalkan jika ada", r.TargetKuota.Int64, target, r.Kapasitas)
	if err := q.CreateOnboardingClassAction(ctx, kelasID, actorID, "quota_set", note, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *OnboardingService) ClassAction(ctx context.Context, kelasID, actorID int64, action, noteRaw string) error {
	note, err := onboardingReason(noteRaw)
	if err != nil {
		return err
	}
	if kelasID <= 0 {
		return ErrOnboardingNotFound
	}
	if note == "" {
		return onboardingInputError("catatan atau alasan wajib diisi")
	}
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)
	roles := []string{"admin_kelas", "super_admin"}
	switch action {
	case "cs_followup":
		roles = append(roles, "cs")
	case "marketing_followup", "vacancy_sent":
	case "exception_approved", "exception_revoked":
		roles = []string{"super_admin"}
	default:
		return onboardingInputError("jenis tindakan tidak dikenal")
	}
	if err := s.verifyClassActor(ctx, q, actorID, roles...); err != nil {
		return err
	}
	r, err := q.GetOnboardingClassControl(ctx, kelasID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOnboardingNotFound
	}
	if err != nil {
		return err
	}
	if !r.TargetKuota.Valid || r.TargetKuota.Int64 <= 0 || r.TargetKuota.Int64 > r.Kapasitas {
		return onboardingInputError("konfigurasi target kuota belum valid")
	}
	now := time.Now().UTC()
	anchor := r.FirstCaseAt
	if r.FirstCandidateAt.Valid {
		anchor = r.FirstCandidateAt.Time
	}
	day, _ := OnboardingClassPhase(anchor, now)
	affected := int64(1)
	switch action {
	case "vacancy_sent":
		if onboardingIsPrivate(r.Tipe) || day < 4 || r.JumlahSantri >= r.TargetKuota.Int64 {
			return onboardingInputError("Vacancy Alert hanya untuk kelas non-private di bawah kuota sejak D4")
		}
		affected, err = q.MarkOnboardingVacancySent(ctx, kelasID, now)
	case "cs_followup", "marketing_followup":
		if onboardingIsPrivate(r.Tipe) || day < 8 || r.JumlahSantri >= r.TargetKuota.Int64 {
			return onboardingInputError("tindak lanjut Push hanya untuk kelas non-private di bawah kuota sejak D8")
		}
	case "exception_approved":
		if day < 11 || r.JumlahSantri >= r.TargetKuota.Int64 {
			return onboardingInputError("exception hanya untuk kelas di bawah target pada D11 atau setelahnya")
		}
		affected, err = q.SetOnboardingException(ctx, kelasID, actorID, note, now)
	case "exception_revoked":
		affected, err = q.RevokeOnboardingException(ctx, kelasID, now)
	}
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrOnboardingConflict
	}
	if err := q.CreateOnboardingClassAction(ctx, kelasID, actorID, action, note, now); err != nil {
		return err
	}
	return tx.Commit()
}
