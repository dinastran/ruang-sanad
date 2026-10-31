package services

import (
	"errors"
	"testing"
)

func TestClassPlanExceptionDecisionValidation(t *testing.T) {
	for _,tc:=range []struct{
		action string
		reason string
		valid bool
	}{
		{"exception_approve","Santri sudah konfirmasi untuk mulai pekan ini",true},
		{"exception_reject","Belum memenuhi alasan operasional",true},
		{"exception_approve","singkat",false},
		{"exception_reject","",false},
		{"admin_approve","Alasan sudah cukup panjang",false},
	}{
		err:=validateQuotaDecision(tc.action,tc.reason)
		if tc.valid&&err!=nil{t.Errorf("%s rejected: %v",tc.action,err)}
		if !tc.valid&&!errors.Is(err,ErrPlanWorkflowInvalid){t.Errorf("%s should fail: %v",tc.action,err)}
	}
}

func TestClassPlanExceptionOnlyWaivesMinimumQuota(t *testing.T) {
	if !onlyQuotaViolation([]PlanningViolation{{Code:"MIN_QUOTA_NOT_MET"}}){
		t.Fatal("strictly under-quota should be reviewable")
	}
	for _,violations:=range [][]PlanningViolation{
		nil,
		{{Code:"TEACHER_UNAVAILABLE"}},
		{{Code:"MIN_QUOTA_NOT_MET"},{Code:"SCHEDULE_CONFLICT"}},
		{{Code:"MIN_QUOTA_NOT_MET"},{Code:"SCHEDULE_NOT_APPROVED"}},
	}{
		if onlyQuotaViolation(violations){t.Fatalf("hard violations may not be waived: %+v",violations)}
	}
}
