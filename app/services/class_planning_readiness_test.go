package services

import (
	"testing"

	"github.com/maulanashalihin/laju-go/app/queries"
)

func readinessHasIssue(r PlanningReadinessReport, code string) bool {
	for _, item := range r.Issues {
		if item.Code == code { return true }
	}
	return false
}

func TestPlanningReadinessBlocksUnsafeRecommendation(t *testing.T) {
	c := queries.PlanningReadinessCounts{
		AwaitingOnboarding: 10, LegacyPreassigned: 4,
		CandidateDataComplete: 7, CRMPaymentVerified: 0,
		CRMVerificationUnknown: 7, ActiveTeachers: 10,
		ConfirmedAvailabilityGuru: 5, ApprovedTeachingLevelGuru: 0,
		UsableTeacherCount: 0, ActiveClassesNoRoutineSlots: 3,
		RegularClassesAboveMax: 1,
	}
	r := planningReadinessFromCounts("2026-10-31", c)
	if r.ReadyForAutomaticRecommendation { t.Fatal("unsafe baseline marked ready") }
	for _, code := range []string{
		"NO_VERIFIED_CRM_DEMAND", "NO_CONFIRMED_TEACHER_LEVEL_AND_AVAILABILITY",
		"CLASS_SCHEDULE_COVERAGE_INCOMPLETE", "CRM_PAYMENT_EVIDENCE_MISSING",
		"LEGACY_PREASSIGNMENT", "LEGACY_CAPACITY_OVER_10",
	} {
		if !readinessHasIssue(r, code) { t.Errorf("missing issue %s", code) }
	}
}

func TestPlanningReadinessAggregateReadyIsNotActivationApproval(t *testing.T) {
	c := queries.PlanningReadinessCounts{CRMPaymentVerified: 8, UsableTeacherCount: 2}
	r := planningReadinessFromCounts("2026-10-31", c)
	if !r.ReadyForAutomaticRecommendation || len(r.Issues) != 0 {
		t.Fatalf("aggregate readiness mismatch: %+v", r)
	}
}
