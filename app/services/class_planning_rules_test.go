package services

import "testing"

func planningBase() PlanningProposal {
	students := make([]PlanningCandidate, 8)
	for i := range students {
		students[i] = PlanningCandidate{
			ID: int64(i + 1), Gender: "P", Level: "PRA",
			PlacementEligible: true, ScheduleApproved: true,
		}
	}
	return PlanningProposal{
		Program: "Reguler", Gender: "P", Frequency: 2, Capacity: 10,
		Slots: []PlanningSlot{{Day: 2, Start: "19:30", End: "21:00"}, {Day: 5, Start: "20:00", End: "21:30"}},
		Teacher: PlanningTeacher{
			ID: 1, Gender: "P", Active: true,
			ApprovedLevels: map[string]bool{"PRA": true, "TQ": true},
			Available:      []PlanningSlot{{Day: 2, Start: "18:00", End: "22:00"}, {Day: 5, Start: "19:00", End: "22:00"}},
		},
		Candidates: students,
	}
}

func hasPlanningCode(issues []PlanningViolation, code string) bool {
	for _, v := range issues {
		if v.Code == code {
			return true
		}
	}
	return false
}

func TestValidatePlanningActivationValidTwoDifferentTimes(t *testing.T) {
	if issues := ValidatePlanningActivation(planningBase()); len(issues) != 0 {
		t.Fatalf("valid plan rejected: %+v", issues)
	}
}

func TestValidatePlanningActivationBusinessRules(t *testing.T) {
	tests := []struct {
		name, code string
		edit       func(*PlanningProposal)
	}{
		{"under minimum", "MIN_QUOTA_NOT_MET", func(p *PlanningProposal) { p.Candidates = p.Candidates[:7] }},
		{"over capacity", "CAPACITY_EXCEEDED", func(p *PlanningProposal) {
			for i := 9; i <= 11; i++ {
				p.Candidates = append(p.Candidates, PlanningCandidate{ID: int64(i), Gender: "P", Level: "PRA", PlacementEligible: true, ScheduleApproved: true})
			}
		}},
		{"regular capacity above 10", "REGULAR_CAPACITY", func(p *PlanningProposal) { p.Capacity = 15 }},
		{"female teacher cannot teach ikhwan", "TEACHER_GENDER_MISMATCH", func(p *PlanningProposal) { p.Gender = "L" }},
		{"missing one day availability", "TEACHER_UNAVAILABLE", func(p *PlanningProposal) { p.Teacher.Available = p.Teacher.Available[:1] }},
		{"occupied overlaps", "SCHEDULE_CONFLICT", func(p *PlanningProposal) { p.Teacher.Occupied = []PlanningSlot{{Day: 2, Start: "20:00", End: "21:30"}} }},
		{"adjacent not overlap", "DUPLICATE_DAY", func(p *PlanningProposal) { p.Slots[1] = PlanningSlot{Day: 2, Start: "21:00", End: "22:30"} }},
		{"wrong duration", "INVALID_DURATION", func(p *PlanningProposal) { p.Slots[0].End = "20:30" }},
		{"invalid time", "INVALID_SLOT", func(p *PlanningProposal) { p.Slots[0].Start = "25:00" }},
		{"frequency mismatch", "SLOT_COUNT_MISMATCH", func(p *PlanningProposal) { p.Frequency = 1 }},
		{"unapproved level", "LEVEL_NOT_APPROVED", func(p *PlanningProposal) { p.Candidates[0].Level = "OTHER" }},
		{"missing schedule approval", "SCHEDULE_NOT_APPROVED", func(p *PlanningProposal) { p.Candidates[0].ScheduleApproved = false }},
		{"already assigned", "STUDENT_NOT_ELIGIBLE", func(p *PlanningProposal) { p.Candidates[0].CurrentClassID = 99 }},
		{"mixed santri", "STUDENT_GENDER_MISMATCH", func(p *PlanningProposal) { p.Candidates[0].Gender = "L" }},
		{"duplicate candidate", "DUPLICATE_CANDIDATE", func(p *PlanningProposal) { p.Candidates[1].ID = 1 }},
		{"inactive teacher", "TEACHER_INACTIVE", func(p *PlanningProposal) { p.Teacher.Active = false }},
		{"workload exceeded", "TEACHER_WORKLOAD_EXCEEDED", func(p *PlanningProposal) {
			limit := 270
			p.Teacher.MaxWeeklyMinutes = &limit
			p.Teacher.CurrentWeeklyMinutes = 180
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := planningBase()
			tc.edit(&p)
			issues := ValidatePlanningActivation(p)
			if !hasPlanningCode(issues, tc.code) {
				t.Fatalf("missing %s, got %+v", tc.code, issues)
			}
		})
	}
}

func TestValidatePlanningActivationQuotaException(t *testing.T) {
	p := planningBase()
	p.Candidates = p.Candidates[:6]
	p.ExceptionApproved = true
	if issues := ValidatePlanningActivation(p); len(issues) != 0 {
		t.Fatalf("approved exception rejected: %+v", issues)
	}
}

func TestValidatePlanningActivationMaleTeacherForAkhwat(t *testing.T) {
	p := planningBase()
	p.Teacher.Gender = "L"
	if issues := ValidatePlanningActivation(p); len(issues) != 0 {
		t.Fatalf("male teacher rejected for akhwat: %+v", issues)
	}
}

func TestPlanningOverlapsBoundary(t *testing.T) {
	a := PlanningSlot{Day: 2, Start: "19:30", End: "21:00"}
	b := PlanningSlot{Day: 2, Start: "21:00", End: "22:30"}
	if planningOverlaps(a, b) {
		t.Fatal("adjacent slots must not overlap")
	}
	b.Start = "20:59"
	if !planningOverlaps(a, b) {
		t.Fatal("intersecting slots must overlap")
	}
}
