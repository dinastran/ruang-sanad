package services

import (
	"context"
	"fmt"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

func onboardingWIBDate(t time.Time) time.Time {
	local := t.In(onboardingWIB)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, onboardingWIB)
}

func onboardingCalendarDays(start, end time.Time) int {
	a := onboardingWIBDate(start)
	b := onboardingWIBDate(end)
	return int(b.Sub(a).Hours() / 24)
}
func onboardingPercent(a, b int64) *float64 {
	if b == 0 {
		return nil
	}
	result := 100 * float64(a) / float64(b)
	return &result
}

func computeOnboardingKPI(cases []queries.OnboardingMonitoringCaseRow, classes []queries.OnboardingClassControlRow, now time.Time) models.OnboardingKPIReport {
	r := models.OnboardingKPIReport{
		AsOf:  now.In(onboardingWIB).Format(time.RFC3339),
		Total: int64(len(cases)), Stages: map[string]int64{},
	}
	var waitingDays int64
	for _, c := range cases {
		r.Stages[c.Stage]++
		switch c.Stage {
		case "batal":
			r.Cancelled++
			continue
		case "masuk_kelas":
			r.Completed++
			// A completed case always enters the SLA denominator, including late starts.
			r.DueCases++
			if c.ActualStartedAt.Valid {
				if c.ActualStartedAt.Time.Before(c.DeadlineAt) {
					r.OnTimeCases++
				}
				days := onboardingCalendarDays(c.RegisteredAt, c.ActualStartedAt.Time)
				if days >= 0 {
					waitingDays += int64(days)
				}
			}
		default:
			r.InProgress++
			if !now.Before(c.DeadlineAt) {
				r.DueCases++
				r.Overdue++
			} else if onboardingCalendarDays(c.RegisteredAt, now) >= 11 {
				r.AtRisk++
			}
		}
	}
	r.SLARate = onboardingPercent(r.OnTimeCases, r.DueCases)
	if r.Completed > 0 {
		avg := float64(waitingDays) / float64(r.Completed)
		r.AvgWaitingDays = &avg
	}
	for _, c := range classes {
		if onboardingIsPrivate(c.Tipe) || !c.TargetKuota.Valid || c.TargetKuota.Int64 <= 0 {
			continue
		}
		anchor := c.FirstCaseAt
		if c.FirstCandidateAt.Valid {
			anchor = c.FirstCandidateAt.Time
		}
		day, _ := OnboardingClassPhase(anchor, now)
		// Current obligation: alert recorded OR class still below target.
		// Without a historical obligation ledger, a class filled before D4 is ineligible.
		required := c.VacancyAlertSentAt.Valid || c.JumlahSantri < c.TargetKuota.Int64
		if day < 4 || !required {
			continue
		}
		r.AlertEligible++
		if !c.VacancyAlertSentAt.Valid {
			r.AlertPending++
		}
		if day >= 8 {
			r.AlertDue++
			timely := c.VacancyAlertSentAt.Valid &&
				onboardingCalendarDays(anchor, c.VacancyAlertSentAt.Time) >= 4 &&
				onboardingCalendarDays(anchor, c.VacancyAlertSentAt.Time) <= 7
			if timely {
				r.AlertOnTime++
			} else {
				r.AlertLate++
			}
		}
	}
	r.AlertOnTimeRate = onboardingPercent(r.AlertOnTime, r.AlertDue)
	return r
}

// Monitoring reports are read-only. Cases with 'batal' are counted separately,
// not quietly treated as on-time completions.
func (s *OnboardingService) Monitoring(ctx context.Context) (models.OnboardingKPIReport, error) {
	cases, err := s.querier.ListOnboardingMonitoringCases(ctx)
	if err != nil {
		return models.OnboardingKPIReport{}, err
	}
	classes, err := s.querier.ListAllOnboardingClassControls(ctx)
	if err != nil {
		return models.OnboardingKPIReport{}, err
	}
	return computeOnboardingKPI(cases, classes, time.Now().UTC()), nil
}

// TickOnboardingNotifications creates one in-app notification per eligible user
// and milestone. Unique index 0092 guarantees retries and concurrent ticks
// cannot duplicate a notification. No external messaging is sent.
func (s *OnboardingService) TickOnboardingNotifications(ctx context.Context, now time.Time) (models.OnboardingMonitoringResult, error) {
	cases, err := s.querier.ListOnboardingMonitoringCases(ctx)
	if err != nil {
		return models.OnboardingMonitoringResult{}, err
	}
	classes, err := s.querier.ListAllOnboardingClassControls(ctx)
	if err != nil {
		return models.OnboardingMonitoringResult{}, err
	}
	var count int64
	sendRole := func(roles []string, kind, title, message, url, refType string, refID int64) error {
		n, err := s.querier.NotifyOnboardingRoles(ctx, roles, kind, title, message, url, refType, refID, now)
		count += n
		return err
	}
	sendPIC := func(id int64, kind, title, message, url string, refID int64) error {
		n, err := s.querier.NotifyOnboardingUser(ctx, id, kind, title, message, url, "onboarding_case", refID, now)
		count += n
		return err
	}
	for _, c := range cases {
		if err := ctx.Err(); err != nil {
			return models.OnboardingMonitoringResult{}, err
		}
		if c.Stage == "masuk_kelas" || c.Stage == "batal" {
			continue
		}
		days := onboardingCalendarDays(c.RegisteredAt, now)
		var typ, title, message string
		var roles []string
		if !now.Before(c.DeadlineAt) {
			typ, title = "onboarding_sla_overdue", "SLA onboarding terlewati"
			message = fmt.Sprintf("Kasus onboarding #%d sudah melewati batas 14 hari. Tinjau dan eskalasi segera.", c.ID)
			roles = []string{"admin_kelas", "super_admin"}
		} else if days >= 11 {
			typ, title = "onboarding_sla_warning", "SLA onboarding mendekati batas"
			message = fmt.Sprintf("Kasus onboarding #%d memasuki D%d. Pastikan kelas dimulai sebelum batas SLA.", c.ID, days)
			roles = []string{"admin_kelas"}
		} else {
			continue
		}
		if err := sendRole(roles, typ, title, message, "/app/onboarding", "onboarding_case", c.ID); err != nil {
			return models.OnboardingMonitoringResult{}, err
		}
		if c.PICUserID.Valid {
			if err := sendPIC(c.PICUserID.Int64, typ, title, message, "/app/onboarding", c.ID); err != nil {
				return models.OnboardingMonitoringResult{}, err
			}
		}
	}
	for _, c := range classes {
		if err := ctx.Err(); err != nil {
			return models.OnboardingMonitoringResult{}, err
		}
		anchor := c.FirstCaseAt
		if c.FirstCandidateAt.Valid {
			anchor = c.FirstCandidateAt.Time
		}
		days, _ := OnboardingClassPhase(anchor, now)
		if days < 4 {
			continue
		}
		if !c.TargetKuota.Valid || c.TargetKuota.Int64 <= 0 {
			if err := sendRole([]string{"super_admin"}, "onboarding_quota_unconfigured",
				"Target kuota belum dikonfigurasi",
				fmt.Sprintf("Kelas %s mencapai D%d tanpa target kuota. Tetapkan target operasional.", c.KelasNama, days),
				"/app/onboarding/classes", "onboarding_class", c.KelasID); err != nil {
				return models.OnboardingMonitoringResult{}, err
			}
			continue
		}
		if onboardingIsPrivate(c.Tipe) || c.JumlahSantri >= c.TargetKuota.Int64 || c.ExceptionApprovedAt.Valid {
			continue
		}
		if !c.VacancyAlertSentAt.Valid {
			if err := sendRole([]string{"admin_kelas"}, "onboarding_vacancy_due",
				"Vacancy Alert perlu dikirim",
				fmt.Sprintf("Kelas %s memasuki D%d dan kurang kuota. Kirim alert dan catat pengirimannya.", c.KelasNama, days),
				"/app/onboarding/classes", "onboarding_class", c.KelasID); err != nil {
				return models.OnboardingMonitoringResult{}, err
			}
		}
		if days >= 8 {
			if err := sendRole([]string{"admin_kelas", "cs"}, "onboarding_quota_push",
				"Tindak lanjut pengisian kuota",
				fmt.Sprintf("Kelas %s masih kurang kuota pada D%d. CS dan Admin Kelas perlu koordinasi Marketing.", c.KelasNama, days),
				"/app/onboarding/classes", "onboarding_class", c.KelasID); err != nil {
				return models.OnboardingMonitoringResult{}, err
			}
		}
		if days >= 11 {
			if err := sendRole([]string{"super_admin"}, "onboarding_quota_resolve",
				"Keputusan kelas diperlukan",
				fmt.Sprintf("Kelas %s sudah D%d dan masih kurang kuota. Super Admin perlu meninjau exception atau tindak lanjut.", c.KelasNama, days),
				"/app/onboarding/classes", "onboarding_class", c.KelasID); err != nil {
				return models.OnboardingMonitoringResult{}, err
			}
		}
	}
	return models.OnboardingMonitoringResult{NotificationsCreated: count}, nil
}

func (s *OnboardingService) UserNotices(ctx context.Context, userID int64) ([]models.OnboardingUserNotice, error) {
	if userID <= 0 {
		return nil, onboardingInputError("akun tidak valid")
	}
	rows, err := s.querier.ListOnboardingUserNotices(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]models.OnboardingUserNotice, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.OnboardingUserNotice{
			ID: r.ID, Type: r.Type, Title: r.Title, Message: r.Message,
			ActionURL: r.ActionURL, Read: r.ReadAt.Valid,
			CreatedAt: r.CreatedAt.In(onboardingWIB).Format(time.RFC3339),
		})
	}
	return out, nil
}
func (s *OnboardingService) ReadUserNotice(ctx context.Context, userID, id int64) error {
	if userID <= 0 || id <= 0 {
		return ErrOnboardingNotFound
	}
	affected, err := s.querier.MarkOnboardingUserNoticeRead(ctx, userID, id, time.Now().UTC())
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrOnboardingNotFound
	}
	return nil
}
