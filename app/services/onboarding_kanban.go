package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var (
	ErrOnboardingInvalid  = errors.New("perpindahan onboarding tidak valid")
	ErrOnboardingConflict = errors.New("data onboarding telah berubah; muat ulang halaman")
	ErrOnboardingNotFound = errors.New("kasus onboarding tidak ditemukan")
)

const onboardingBoardPageSize int64 = 100

var onboardingStages = []string{
	"onboarding_baru", "matching_kelas", "waiting_kuota", "siap_mulai", "masuk_kelas",
}

func onboardingStageIndex(s string) int {
	for i, v := range onboardingStages {
		if v == s {
			return i
		}
	}
	return -1
}

func onboardingInputError(message string) error {
	return fmt.Errorf("%w: %s", ErrOnboardingInvalid, message)
}

func onboardingReason(raw string) (string, error) {
	reason := strings.TrimSpace(raw)
	if utf8.RuneCountInString(reason) > 300 {
		return "", onboardingInputError("alasan maksimal 300 karakter")
	}
	return reason, nil
}

func (s *OnboardingService) Board(ctx context.Context, page int64) (models.OnboardingBoard, error) {
	if page < 1 {
		page = 1
	}
	if page > 10000 {
		return models.OnboardingBoard{}, onboardingInputError("halaman tidak valid")
	}
	rows, err := s.querier.ListOnboardingBoard(ctx, queries.ListOnboardingBoardParams{
		Limit: onboardingBoardPageSize, Offset: (page - 1) * onboardingBoardPageSize,
	})
	if err != nil {
		return models.OnboardingBoard{}, err
	}
	total, err := s.querier.CountOnboardingBoard(ctx)
	if err != nil {
		return models.OnboardingBoard{}, err
	}
	counts, err := s.querier.CountOnboardingStages(ctx)
	if err != nil {
		return models.OnboardingBoard{}, err
	}
	users, err := s.querier.ListOnboardingPICs(ctx)
	if err != nil {
		return models.OnboardingBoard{}, err
	}
	result := models.OnboardingBoard{
		Cases:  make([]models.OnboardingBoardCard, 0, len(rows)),
		Counts: map[string]int64{},
		PICs:   make([]models.OnboardingPICOption, 0, len(users)),
		Total:  total, Page: page, PageSize: onboardingBoardPageSize,
	}
	for _, row := range counts {
		result.Counts[row.Stage] = row.Total
	}
	for _, user := range users {
		result.PICs = append(result.PICs, models.OnboardingPICOption{ID: user.ID, Name: user.Name, Email: user.Email})
	}
	for _, r := range rows {
		card := models.OnboardingBoardCard{
			ID: r.ID, SantriID: r.SantriID, Nama: r.Nama, IDMahasantri: r.IDMahasantri,
			Stage: r.Stage, PICUserID: r.PicUserID.Int64, PICName: r.PicName,
			RegisteredAt: r.RegisteredAt.In(onboardingWIB).Format(time.RFC3339),
			DeadlineAt:   r.DeadlineAt.In(onboardingWIB).Format(time.RFC3339),
			KelasID:      r.KelasID.Int64, KelasNama: r.KelasNama,
		}
		if r.ActualStartedAt.Valid {
			card.ActualStartedAt = r.ActualStartedAt.Time.In(onboardingWIB).Format(time.RFC3339)
		}
		result.Cases = append(result.Cases, card)
	}
	return result, nil
}

func (s *OnboardingService) Activity(ctx context.Context, id int64) ([]models.OnboardingActivityItem, error) {
	if id <= 0 {
		return nil, ErrOnboardingNotFound
	}
	if _, err := s.querier.GetOnboardingCaseState(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOnboardingNotFound
		}
		return nil, err
	}
	rows, err := s.querier.ListOnboardingActivity(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]models.OnboardingActivityItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.OnboardingActivityItem{
			ID: r.ID, EventType: r.EventType, FromStage: r.FromStage, ToStage: r.ToStage,
			Note: r.Note, ActorName: r.ActorName, CreatedAt: r.CreatedAt.In(onboardingWIB).Format(time.RFC3339),
		})
	}
	return out, nil
}

// AssignPIC validates the referenced user role and changes PIC atomically
// with an audit event. It never changes the onboarding SLA anchor.
func (s *OnboardingService) AssignPIC(ctx context.Context, caseID, picID, actorID int64) error {
	if caseID <= 0 || actorID <= 0 || picID < 0 {
		return onboardingInputError("ID tidak valid")
	}
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)

	state, err := q.GetOnboardingCaseState(ctx, caseID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOnboardingNotFound
	}
	if err != nil {
		return err
	}
	if state.Stage == "masuk_kelas" || state.Stage == "batal" {
		return onboardingInputError("kasus selesai tidak dapat diubah")
	}
	if picID > 0 {
		role, err := q.GetOnboardingPICRole(ctx, picID)
		if errors.Is(err, sql.ErrNoRows) {
			return onboardingInputError("PIC tidak ditemukan")
		}
		if err != nil {
			return err
		}
		if role != "admin_kelas" && role != "super_admin" {
			return onboardingInputError("PIC harus Admin Kelas atau Super Admin")
		}
	}
	if state.PicUserID.Int64 == picID {
		return onboardingInputError("PIC tidak berubah")
	}
	now := time.Now().UTC()
	affected, err := q.SetOnboardingPIC(ctx, queries.SetOnboardingPICParams{
		PicUserID:     sql.NullInt64{Int64: picID, Valid: picID > 0},
		ExpectedPicID: state.PicUserID, UpdatedAt: now, ID: caseID,
	})
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrOnboardingConflict
	}
	note := fmt.Sprintf("PIC %d → %d", state.PicUserID.Int64, picID)
	if err := q.LogOnboardingActivity(ctx, queries.LogOnboardingActivityParams{
		CaseID: caseID, ActorUserID: sql.NullInt64{Int64: actorID, Valid: true},
		EventType: "onboarding.pic_changed", FromStage: state.Stage, ToStage: state.Stage,
		Note: note, CreatedAt: now,
	}); err != nil {
		return err
	}
	return tx.Commit()
}

// validateOnboardingClassReady is shared by the readiness and final-start
// transitions. A previously approved exception can be revoked, or the target
// quota, guru, jadwal, and class status can change before actual start.
func validateOnboardingClassReady(ctx context.Context, q *queries.Querier, kelasID int64) error {
	kelas, err := q.GetKelasByID(ctx, kelasID)
	if errors.Is(err, sql.ErrNoRows) {
		return onboardingInputError("kelas kandidat tidak ditemukan")
	}
	if err != nil {
		return err
	}
	if kelas.IsAktif != 1 || !kelas.GuruID.Valid || strings.TrimSpace(kelas.Jadwal) == "" {
		return onboardingInputError("kelas aktif, guru, dan jadwal harus dikonfirmasi")
	}
	control, err := q.GetOnboardingClassControl(ctx, kelasID)
	if errors.Is(err, sql.ErrNoRows) {
		return onboardingInputError("target kuota belum dikonfigurasi oleh Super Admin")
	}
	if err != nil {
		return err
	}
	if !control.TargetKuota.Valid {
		return onboardingInputError("target kuota belum dikonfigurasi oleh Super Admin")
	}
	targetKuota := control.TargetKuota.Int64
	if targetKuota < 1 || targetKuota > kelas.Kapasitas {
		return onboardingInputError("target kuota kelas tidak valid")
	}
	if kelas.JumlahSantri < targetKuota && !control.ExceptionApprovedAt.Valid {
		return onboardingInputError("kuota belum terpenuhi; diperlukan exception Super Admin")
	}
	return nil
}

// Move enforces validated, adjacent stage transitions. Backward movement
// requires a recorded reason. 'Batal' is terminal, not a Kanban column.
func (s *OnboardingService) Move(ctx context.Context, caseID, actorID int64, target, rawReason, actualDate string) error {
	if caseID <= 0 || actorID <= 0 {
		return onboardingInputError("ID tidak valid")
	}
	reason, err := onboardingReason(rawReason)
	if err != nil {
		return err
	}
	if target != "batal" && onboardingStageIndex(target) < 0 {
		return onboardingInputError("status tujuan tidak dikenal")
	}
	tx, err := s.querier.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.querier.WithTx(tx)
	state, err := q.GetOnboardingCaseState(ctx, caseID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOnboardingNotFound
	}
	if err != nil {
		return err
	}
	if state.Stage == "batal" || state.Stage == "masuk_kelas" {
		return onboardingInputError("kasus terminal tidak dapat dipindahkan")
	}
	if target == state.Stage {
		return onboardingInputError("status tidak berubah")
	}
	now := time.Now().UTC()
	var affected int64
	event := "onboarding.stage_changed"

	if target == "batal" {
		if reason == "" {
			return onboardingInputError("alasan batal wajib diisi")
		}
		affected, err = q.CancelOnboardingCase(ctx, queries.CancelOnboardingCaseParams{
			CancelledAt: sql.NullTime{Time: now, Valid: true}, CancellationReason: reason, UpdatedAt: now,
			ID: caseID, ExpectedStage: state.Stage,
		})
		event = "onboarding.cancelled"
	} else {
		fromIndex, toIndex := onboardingStageIndex(state.Stage), onboardingStageIndex(target)
		if fromIndex < 0 || (toIndex != fromIndex+1 && toIndex != fromIndex-1) {
			return onboardingInputError("perpindahan harus ke tahap yang bersebelahan")
		}
		if toIndex < fromIndex && reason == "" {
			return onboardingInputError("alasan perpindahan mundur wajib diisi")
		}
		santri, err := q.GetSantriByID(ctx, state.SantriID)
		if err != nil {
			return err
		}
		if toIndex > fromIndex {
			if santri.Status != "aktif" {
				return onboardingInputError("status santri harus aktif")
			}
			if !state.PicUserID.Valid {
				return onboardingInputError("tetapkan PIC Admin Kelas terlebih dahulu")
			}
			switch target {
			case "matching_kelas":
				if strings.TrimSpace(santri.Nama) == "" || strings.TrimSpace(santri.Angkatan) == "" ||
					(santri.JenisKelamin != "L" && santri.JenisKelamin != "P") {
					return onboardingInputError("data dasar santri belum lengkap")
				}
			case "waiting_kuota":
				if !santri.KelasID.Valid || santri.IsLengkap == 0 {
					return onboardingInputError("lengkapi data kelas dan tentukan kandidat melalui Admin Kelas")
				}
				if _, err := q.GetKelasByID(ctx, santri.KelasID.Int64); err != nil {
					return onboardingInputError("kelas kandidat tidak ditemukan")
				}
			case "siap_mulai":
				if !santri.KelasID.Valid {
					return onboardingInputError("kelas kandidat belum ditetapkan")
				}
				if err := validateOnboardingClassReady(ctx, q, santri.KelasID.Int64); err != nil {
					return err
				}
			case "masuk_kelas":
				if !santri.KelasID.Valid {
					return onboardingInputError("kelas belum ditetapkan")
				}
				// Class conditions can change after a case reaches Siap Mulai.
				// Recheck live quota, exception, guru, and jadwal before finalizing.
				if err := validateOnboardingClassReady(ctx, q, santri.KelasID.Int64); err != nil {
					return err
				}
				date, err := time.ParseInLocation(time.DateOnly, strings.TrimSpace(actualDate), onboardingWIB)
				if err != nil {
					return onboardingInputError("tanggal mulai aktual wajib berformat YYYY-MM-DD")
				}
				regDate := state.RegisteredAt.In(onboardingWIB)
				earliest := time.Date(regDate.Year(), regDate.Month(), regDate.Day(), 0, 0, 0, 0, onboardingWIB)
				today := time.Now().In(onboardingWIB)
				todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, onboardingWIB)
				if date.Before(earliest) || date.After(todayStart) {
					return onboardingInputError("tanggal mulai harus antara tanggal pendaftaran dan hari ini")
				}
				affected, err = q.FinishOnboardingCase(ctx, queries.FinishOnboardingCaseParams{
					ActualStartedAt: sql.NullTime{Time: date.UTC(), Valid: true}, UpdatedAt: now, ID: caseID,
				})
			}
		}
		if target != "masuk_kelas" || toIndex < fromIndex {
			affected, err = q.SetOnboardingStage(ctx, queries.SetOnboardingStageParams{
				Stage: target, UpdatedAt: now, ID: caseID, ExpectedStage: state.Stage,
			})
		}
	}
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrOnboardingConflict
	}
	if err := q.LogOnboardingActivity(ctx, queries.LogOnboardingActivityParams{
		CaseID: caseID, ActorUserID: sql.NullInt64{Int64: actorID, Valid: true},
		EventType: event, FromStage: state.Stage, ToStage: target, Note: reason, CreatedAt: now,
	}); err != nil {
		return err
	}
	return tx.Commit()
}
