package services

import (
	"testing"

	"github.com/maulanashalihin/laju-go/app/queries"
)

func previewAlternativeFixture() PlanningMatchSnapshot {
	s := matchingSnapshot(7)
	s.Classes = []queries.PlanningExistingClass{{
		ID: 20, Angkatan: "2026-A", Gender: "P", TeacherGender: "P",
		Level: "PRA", Frequency: "2x/pekan", Capacity: 10, Occupancy: 8, GuruID: 2,
	}}
	s.ClassSlots = []queries.PlanningClassSlot{
		{KelasID: 20, Day: 6, Start: "19:30"}, {KelasID: 20, Day: 7, Start: "19:30"},
	}
	s.ApprovedAlternatives = []queries.PlanningApprovedAlternative{
		{SantriID: 1, OfferCode: "WEEKEND", OfferDays: "6,7", OfferStart: "19:30", OfferEnd: "21:00"},
		{SantriID: 2, OfferCode: "WEEKEND", OfferDays: "6,7", OfferStart: "19:30", OfferEnd: "21:00"},
		{SantriID: 3, OfferCode: "WEEKEND", OfferDays: "6,7", OfferStart: "19:30", OfferEnd: "21:00"},
	}
	return s
}

func TestPlanningAlternateMatchingUsesSpareSeatsWithoutDoubleAllocation(t *testing.T) {
	s := previewAlternativeFixture()
	r := PlanMatchingPreview("2026-10-31", s)
	if r.ConfirmedDemand != 7 || r.EligibleDemand != 7 {
		t.Fatalf("alternatives must not inflate confirmed demand: %+v", r)
	}
	if len(r.AlternativeMatches) != 2 {
		t.Fatalf("only two seats available for approved alternatives: %+v", r.AlternativeMatches)
	}
	if r.AlternativeMatches[0].SantriID != 1 || r.AlternativeMatches[1].SantriID != 2 {
		t.Fatalf("deterministic selection broken: %+v", r.AlternativeMatches)
	}
	seen := make(map[int64]bool)
	for _, a := range r.AlternativeMatches {
		seen[a.SantriID] = true
	}
	for _, option := range r.Suggestions {
		for _, id := range option.CandidateIDs {
			if seen[id] {
				t.Fatalf("student %d double counted", id)
			}
			seen[id] = true
		}
	}
	if len(seen) != 7 {
		t.Fatalf("all seven original candidates should be represented once, got %d", len(seen))
	}
	if len(r.Suggestions) != 1 || r.Suggestions[0].State != "needs_quota" ||
		len(r.Suggestions[0].CandidateIDs) != 5 {
		t.Fatalf("primary shortage must still be shown: %+v", r.Suggestions)
	}
}

func TestPlanningAlternateMatchingNeverOverridesViablePrimary(t *testing.T) {
	s := previewAlternativeFixture()
	s.Demand = matchingDemand(8)
	r := PlanMatchingPreview("2026-10-31", s)
	if len(r.Suggestions) != 1 || r.Suggestions[0].State != "ready_preview" {
		t.Fatalf("valid original group must remain preferred: %+v", r.Suggestions)
	}
	if len(r.AlternativeMatches) != 0 {
		t.Fatalf("do not offer alternate for candidate already in viable primary: %+v", r.AlternativeMatches)
	}
}

func TestPlanningAlternateMatchingRejectsAmbiguousOrForeignPreferences(t *testing.T) {
	s := previewAlternativeFixture()
	s.ApprovedAlternatives = []queries.PlanningApprovedAlternative{
		{SantriID: 1, OfferCode: "BAD", OfferDays: "6,7", OfferStart: "19:30", OfferEnd: "20:00"},
		{SantriID: 2, OfferCode: "BAD", OfferDays: "6,6", OfferStart: "19:30", OfferEnd: "21:00"},
		{SantriID: 99, OfferCode: "WEEKEND", OfferDays: "6,7", OfferStart: "19:30", OfferEnd: "21:00"},
		{SantriID: 3, OfferCode: "SJ2", OfferDays: "6,7", OfferStart: "19:30", OfferEnd: "21:00"},
	}
	r := PlanMatchingPreview("2026-10-31", s)
	if len(r.AlternativeMatches) != 0 {
		t.Fatalf("unverified, duplicate-day, foreign and original choice must not match: %+v", r.AlternativeMatches)
	}
}

func TestPlanningMatchingExistingCapacitySharedAcrossOfferCodes(t *testing.T) {
	s := matchingSnapshot(4)
	for i := 2; i < 4; i++ {
		s.Demand[i].OfferCode = "SECOND"
	}
	s.Classes = []queries.PlanningExistingClass{{
		ID: 20, Angkatan: "2026-A", Gender: "P", TeacherGender: "P",
		Level: "PRA", Frequency: "2x/pekan", Capacity: 10, Occupancy: 8, GuruID: 2,
	}}
	s.ClassSlots = []queries.PlanningClassSlot{
		{KelasID: 20, Day: 2, Start: "19:30"}, {KelasID: 20, Day: 5, Start: "19:30"},
	}
	r := PlanMatchingPreview("2026-10-31", s)
	assigned := 0
	for _, o := range r.Suggestions {
		if o.Kind == "existing_class" && o.ClassID == 20 {
			assigned += len(o.CandidateIDs)
		}
	}
	if assigned != 2 {
		t.Fatalf("multiple primary offer codes overfilled 2 remaining seats: %d", assigned)
	}
}

func TestPlanningAlternativeDoesNotRecoverInvalidPrimaryDemand(t *testing.T) {
	s:=previewAlternativeFixture()
	s.Demand[0].OfferEnd="20:00"
	r:=PlanMatchingPreview("2026-10-31",s)
	for _,a:=range r.AlternativeMatches{
		if a.SantriID==1{t.Fatal("ambiguous original primary demand must remain excluded")}
	}
	if r.SkippedAmbiguous!=1{t.Fatalf("expected excluded primary row, got %+v",r)}
}
