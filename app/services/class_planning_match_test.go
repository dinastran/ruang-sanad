package services

import (
	"testing"

	"github.com/maulanashalihin/laju-go/app/queries"
)

func matchingDemand(n int) []queries.PlanningDemand {
	out:=make([]queries.PlanningDemand,n)
	for i:=range out {
		out[i]=queries.PlanningDemand{
			SantriID:int64(i+1),Angkatan:"2026-A",Gender:"P",
			Level:"PRA",Frequency:"2x/pekan",OfferCode:"SJ2",
			OfferDays:"2,5",OfferStart:"19:30",OfferEnd:"21:00",
		}
	}
	return out
}

func matchingSnapshot(n int) PlanningMatchSnapshot {
	return PlanningMatchSnapshot{
		Demand:matchingDemand(n),
		Classes:[]queries.PlanningExistingClass{},
		ClassSlots:[]queries.PlanningClassSlot{},
		Windows:[]queries.PlanningTeacherWindow{
			{GuruID:1,Gender:"L",Status:"tetap",Day:2,Start:"19:00",End:"22:00"},
			{GuruID:1,Gender:"L",Status:"tetap",Day:5,Start:"19:00",End:"22:00"},
			{GuruID:2,Gender:"P",Status:"part_time",Day:2,Start:"19:00",End:"22:00"},
			{GuruID:2,Gender:"P",Status:"part_time",Day:5,Start:"19:00",End:"22:00"},
		},
		Levels:[]queries.PlanningTeacherLevel{
			{GuruID:1,Level:"PRA"},{GuruID:2,Level:"PRA"},
		},
		UnsafeTeacher:map[int64]bool{},
		MaxWorkload:map[int64]int64{},
	}
}

func TestPlanningMatchingExistingClassFirstWithoutDoubleCounting(t *testing.T) {
	p:=matchingSnapshot(8)
	p.Classes=[]queries.PlanningExistingClass{
		{ID:10,Angkatan:"2026-A",Gender:"P",TeacherGender:"L",Level:"PRA",
			Frequency:"2x/pekan",Capacity:15,Occupancy:7,GuruID:1},
	}
	p.ClassSlots=[]queries.PlanningClassSlot{
		{KelasID:10,Day:2,Start:"19:30"},
		{KelasID:10,Day:5,Start:"19:30"},
	}
	r:=PlanMatchingPreview("2026-10-31",p)
	if len(r.Suggestions)!=2 {t.Fatalf("expected fill + remainder, got %+v",r.Suggestions)}
	existing:=r.Suggestions[0]
	if existing.Kind!="existing_class"||existing.ClassID!=10||len(existing.CandidateIDs)!=3{
		t.Fatalf("expected only three seats to legacy class, got %+v",existing)
	}
	newClass:=r.Suggestions[1]
	if newClass.Kind!="new_class"||newClass.State!="needs_quota"||len(newClass.CandidateIDs)!=5{
		t.Fatalf("remaining five should not activate: %+v",newClass)
	}
	seen:=map[int64]bool{}
	for _,option:=range r.Suggestions{
		for _,id:=range option.CandidateIDs{
			if seen[id]{t.Fatalf("santri %d duplicated",id)}
			seen[id]=true
		}
	}
	if len(seen)!=8 {t.Fatalf("missing candidates: %d",len(seen))}
}

func TestPlanningMatchingPrefersAkhwatTeacherWhenBothAvailable(t *testing.T) {
	p:=matchingSnapshot(8)
	r:=PlanMatchingPreview("2026-10-31",p)
	if len(r.Suggestions)!=1||r.Suggestions[0].State!="ready_preview"{
		t.Fatalf("expected one ready preview: %+v",r.Suggestions)
	}
	teachers:=r.Suggestions[0].TeacherOptions
	if len(teachers)!=2||teachers[0].GuruID!=2 {
		t.Fatalf("akhwat should lead for akhwat class, got %+v",teachers)
	}
}

func TestPlanningMatchingBlocksUnknownSchedulesAndIncompatibleGender(t *testing.T) {
	p:=matchingSnapshot(8)
	for i:=range p.Demand{p.Demand[i].Gender="L"}
	r:=PlanMatchingPreview("2026-10-31",p)
	for _,teacher:=range r.Suggestions[0].TeacherOptions{
		if teacher.GuruID==2 {t.Fatalf("akhwat teacher offered for ikhwan: %+v",teacher)}
	}
	p.Demand[0].OfferEnd="20:00"
	r=PlanMatchingPreview("2026-10-31",p)
	if r.SkippedAmbiguous!=1||r.EligibleDemand!=7{
		t.Fatalf("invalid time must be excluded: %+v",r)
	}
}

func TestPlanningMatchingConflictAndUnknownRoutineExcludeTeacher(t *testing.T) {
	p:=matchingSnapshot(8)
	p.Busy=[]queries.PlanningTeacherBusy{
		{GuruID:2,Day:2,Start:"20:00",End:"21:30"},
	}
	r:=PlanMatchingPreview("2026-10-31",p)
	if len(r.Suggestions)!=1||len(r.Suggestions[0].TeacherOptions)!=1||
		r.Suggestions[0].TeacherOptions[0].GuruID!=1 {
		t.Fatalf("busy teacher must be excluded: %+v",r.Suggestions)
	}
	p.UnsafeTeacher[1]=true
	r=PlanMatchingPreview("2026-10-31",p)
	if r.Suggestions[0].State!="needs_teacher"{
		t.Fatalf("no safe teacher must block class: %+v",r.Suggestions[0])
	}
}

func TestPlanningMatchingCanSplit16DemandIntoTwoViableClasses(t *testing.T) {
	p:=matchingSnapshot(16)
	r:=PlanMatchingPreview("2026-10-31",p)
	if len(r.Suggestions)!=2 {t.Fatalf("expected two previews, got %+v",r.Suggestions)}
	if len(r.Suggestions[0].CandidateIDs)!=8||len(r.Suggestions[1].CandidateIDs)!=8{
		t.Fatalf("16 should split 8+8: %+v",r.Suggestions)
	}
	if r.Suggestions[0].State!="ready_preview"||r.Suggestions[1].State!="ready_preview"{
		t.Fatalf("both should be viable: %+v",r.Suggestions)
	}
	if r.Suggestions[0].TeacherOptions[0].GuruID==r.Suggestions[1].TeacherOptions[0].GuruID{
		t.Fatalf("teacher must not be double-booked in same preview: %+v",r.Suggestions)
	}
}

func TestPlanningMatchingUnderQuotaNeverReady(t *testing.T) {
	r:=PlanMatchingPreview("2026-10-31",matchingSnapshot(7))
	if len(r.Suggestions)!=1||r.Suggestions[0].State!="needs_quota"{
		t.Fatalf("under quota cannot be ready: %+v",r.Suggestions)
	}
}

func TestPlanningMatchingPolicyMaximumEnforced(t *testing.T) {
	p:=matchingSnapshot(8)
	p.MaxWorkload[2]=90
	p.MaxWorkload[1]=90
	r:=PlanMatchingPreview("2026-10-31",p)
	if r.Suggestions[0].State!="needs_teacher"{
		t.Fatalf("teacher limit must be enforced: %+v",r.Suggestions)
	}
}
