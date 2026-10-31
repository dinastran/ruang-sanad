package services

import (
	"errors"
	"testing"

	"github.com/maulanashalihin/laju-go/app/queries"
)

func validManualDraft() ClassPlanDraftInput {
	return ClassPlanDraftInput{
		Angkatan: "2026-A", Gender: "P", Level: "1", GuruID: 12,
		Frequency: 2, Capacity: 10,
		Slots:        []PlanningSlot{{Day: 2, Start: "19:30", End: "21:00"}, {Day: 5, Start: "20:00", End: "21:30"}},
		CandidateIDs: []int64{1, 2, 3, 4, 5, 6, 7, 8},
	}
}

func TestClassPlanWorkflowDraftInputValidation(t *testing.T) {
	cases := []struct {
		name string
		edit func(*ClassPlanDraftInput)
	}{
		{"bad gender", func(p *ClassPlanDraftInput) { p.Gender = "unknown" }},
		{"wrong frequency", func(p *ClassPlanDraftInput) { p.Frequency = 3 }},
		{"capacity over max", func(p *ClassPlanDraftInput) { p.Capacity = 15 }},
		{"capacity under min", func(p *ClassPlanDraftInput) { p.Capacity = 7 }},
		{"cross midnight", func(p *ClassPlanDraftInput) { p.Slots[0].End = "01:00" }},
		{"wrong session count", func(p *ClassPlanDraftInput) { p.Slots = p.Slots[:1] }},
		{"duplicate day", func(p *ClassPlanDraftInput) { p.Slots[1].Day = 2 }},
		{"duplicate santri", func(p *ClassPlanDraftInput) { p.CandidateIDs[0] = 2 }},
		{"invalid teacher", func(p *ClassPlanDraftInput) { p.GuruID = -1 }},
		{"invalid cohort", func(p *ClassPlanDraftInput) { p.Angkatan = "" }},
		{"existing class out of scope", func(p *ClassPlanDraftInput) { p.TargetKelasID = 17 }},
	}
	if err := normalizeClassPlanDraft(ptrDraft(validManualDraft())); err != nil {
		t.Fatalf("valid draft rejected: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validManualDraft()
			tc.edit(&p)
			if err := normalizeClassPlanDraft(&p); !errors.Is(err, ErrPlanWorkflowInvalid) {
				t.Fatalf("expected invalid: %v", err)
			}
		})
	}
}
func ptrDraft(p ClassPlanDraftInput) *ClassPlanDraftInput { return &p }

func TestClassPlanOfferMatchesExactSessionsAndDifferentTimes(t *testing.T) {
	days := []PlanningSlot{{Day: 2, Start: "19:30", End: "21:00"}, {Day: 5, Start: "19:30", End: "21:00"}}
	if !sameClassPlanOffer(days, queries.ClassPlanOfferEvidence{Days: "5,2", Start: "19:30", End: "21:00"}) {
		t.Fatal("order-independent day code should match")
	}
	for _, offer := range []queries.ClassPlanOfferEvidence{
		{Days: "2,5", Start: "20:00", End: "21:30"},
		{Days: "2,2", Start: "19:30", End: "21:00"},
		{Days: "2", Start: "19:30", End: "21:00"},
	} {
		if sameClassPlanOffer(days, offer) {
			t.Fatalf("mismatched schedule accepted: %+v", offer)
		}
	}
	days[1].Start = "20:00"
	days[1].End = "21:30"
	if sameClassPlanOffer(days, queries.ClassPlanOfferEvidence{Days: "2,5", Start: "19:30", End: "21:00"}) {
		t.Fatal("one-time offer cannot prove consent to different time on second day")
	}
}
