package services

import (
	"context"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
)

type PlanningReadinessIssue struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type PlanningReadinessReport struct {
	AsOf                          string                          `json:"as_of"`
	Counts                        queries.PlanningReadinessCounts `json:"counts"`
	ReadyForAutomaticRecommendation bool                           `json:"ready_for_automatic_recommendation"`
	Issues                        []PlanningReadinessIssue        `json:"issues"`
}

type PlanningReadinessService struct {
	querier *queries.Querier
}

func NewPlanningReadinessService(q *queries.Querier) *PlanningReadinessService {
	return &PlanningReadinessService{querier: q}
}

func planningReadinessFromCounts(date string, counts queries.PlanningReadinessCounts) PlanningReadinessReport {
	r := PlanningReadinessReport{AsOf: date, Counts: counts, Issues: make([]PlanningReadinessIssue, 0)}
	add := func(code, severity, message string) {
		r.Issues = append(r.Issues, PlanningReadinessIssue{code, severity, message})
	}
	if counts.CRMPaymentVerified == 0 {
		add("NO_VERIFIED_CRM_DEMAND", "blocker", "Belum ada kandidat kelas dengan pembayaran CRM terverifikasi. Santri tanpa lead CRM perlu pengecekan pembayaran tersendiri.")
	}
	if counts.UsableTeacherCount == 0 {
		add("NO_CONFIRMED_TEACHER_LEVEL_AND_AVAILABILITY", "blocker", "Belum ada guru aktif dengan slot available terkonfirmasi dan kompetensi level yang disetujui.")
	}
	if counts.ActiveClassesNoRoutineSlots > 0 {
		add("CLASS_SCHEDULE_COVERAGE_INCOMPLETE", "blocker", "Ada kelas aktif dengan guru tetapi jadwal rutin yang sedang berlaku belum terstruktur; konflik guru belum dapat dikesampingkan.")
	}
	if counts.CRMVerificationUnknown > 0 {
		add("CRM_PAYMENT_EVIDENCE_MISSING", "warning", "Sebagian kandidat tidak memiliki bukti pembayaran terverifikasi di CRM. Jangan menyimpulkan mereka belum membayar.")
	}
	if counts.LegacyPreassigned > 0 {
		add("LEGACY_PREASSIGNMENT", "warning", "Sebagian santri onboarding telah memiliki kelas_id walaupun belum masuk tahap belajar; bukan berarti mereka sudah mulai belajar.")
	}
	if counts.RegularClassesAboveMax > 0 {
		add("LEGACY_CAPACITY_OVER_10", "warning", "Ada kelas reguler existing berkapasitas di atas 10; jangan ubah massal, batasi penempatan baru sesuai kebijakan.")
	}
	// This only means the aggregate prerequisite checks pass. A future engine
	// must still check each candidate, level, payment, schedule and conflicts.
	r.ReadyForAutomaticRecommendation = counts.CRMPaymentVerified > 0 &&
		counts.UsableTeacherCount > 0 && counts.ActiveClassesNoRoutineSlots == 0
	return r
}

func (s *PlanningReadinessService) Report(ctx context.Context) (PlanningReadinessReport, error) {
	zone, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return PlanningReadinessReport{}, err
	}
	date := time.Now().In(zone).Format(time.DateOnly)
	counts, err := s.querier.ReadPlanningReadiness(ctx, date)
	if err != nil {
		return PlanningReadinessReport{}, err
	}
	return planningReadinessFromCounts(date, counts), nil
}
