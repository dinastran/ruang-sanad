package queries

import (
	"context"
	"testing"
)

func createQuotaTestPlan(t *testing.T,q *Querier) int64 {
	t.Helper()
	id,err:=q.CreateClassPlanWorkflow(context.Background(),17,ClassPlanWorkflowRow{
		Program:"reguler",Angkatan:"2026-A",Gender:"P",GuruID:12,
		Frequency:1,MinQuota:8,Capacity:10,
	},[]ClassPlanWorkflowSlot{{Day:6,Start:"19:30",End:"21:00"}},[]int64{1,2,3,4,5,6,7},"1")
	if err!=nil{t.Fatal(err)}
	ok,err:=q.ClassPlanTransition(context.Background(),id,1,"draft","pending_exception",17)
	if err!=nil||!ok{t.Fatalf("unable to request exception: %v %v",ok,err)}
	return id
}

func TestClassPlanExceptionApprovalRequiresDistinctActorAndCurrentVersion(t *testing.T) {
	ctx:=context.Background()
	db:=workflowQueryDB(t)
	q:=NewQuerier(db)
	id:=createQuotaTestPlan(t,q)
	if err:=q.ClassPlanRequestQuotaException(ctx,id,17,"Menunggu santri lain, kelas harus dimulai");err!=nil{t.Fatal(err)}
	req,err:=q.ClassPlanLatestException(ctx,id)
	if err!=nil||req.Status!="requested"||req.RequestedBy!=17{t.Fatalf("request missing: %+v %v",req,err)}
	if ok,err:=q.ClassPlanDecideQuotaException(ctx,id,req.ID,17,1,"approved","Approved by same requester");err!=nil||ok{
		t.Fatalf("self-approval should fail: %v %v",ok,err)
	}
	if ok,err:=q.ClassPlanDecideQuotaException(ctx,id,req.ID,99,2,"approved","Stale version not valid");err!=nil||ok{
		t.Fatalf("stale version should fail: %v %v",ok,err)
	}
	if ok,err:=q.ClassPlanDecideQuotaException(ctx,id,req.ID,99,1,"approved","Disetujui dengan catatan lengkap");err!=nil||!ok{
		t.Fatalf("super-admin decision CAS failed: %v %v",ok,err)
	}
	if ok,err:=q.ClassPlanTransitionQuotaDecision(ctx,id,1,req.ID,true);err!=nil||!ok{
		t.Fatalf("approved exception did not move to coordinator: %v %v",ok,err)
	}
	approved,err:=q.ClassPlanHasApprovedQuotaException(ctx,id)
	if err!=nil||!approved{t.Fatalf("approved exception not found: %v %v",approved,err)}
	if ok,err:=q.ClassPlanDecideQuotaException(ctx,id,req.ID,99,1,"rejected","Trying a second decision");err!=nil||ok{
		t.Fatalf("duplicate decision should fail: %v %v",ok,err)
	}
	plan,err:=q.GetClassPlanWorkflow(ctx,id)
	if err!=nil||plan.Status!="pending_coordinator"{t.Fatalf("must still need coordinator review: %+v %v",plan,err)}
}

func TestClassPlanExceptionRejectedBlocksCoordinator(t *testing.T) {
	ctx:=context.Background()
	q:=NewQuerier(workflowQueryDB(t))
	id:=createQuotaTestPlan(t,q)
	if err:=q.ClassPlanRequestQuotaException(ctx,id,17,"Jumlah santri baru tujuh orang");err!=nil{t.Fatal(err)}
	rec,err:=q.ClassPlanLatestException(ctx,id)
	if err!=nil{t.Fatal(err)}
	ok,err:=q.ClassPlanDecideQuotaException(ctx,id,rec.ID,99,1,"rejected","Belum ada justifikasi yang cukup")
	if err!=nil||!ok{t.Fatalf("rejection write: %v %v",ok,err)}
	ok,err=q.ClassPlanTransitionQuotaDecision(ctx,id,1,rec.ID,false)
	if err!=nil||!ok{t.Fatalf("reject transition: %v %v",ok,err)}
	plan,err:=q.GetClassPlanWorkflow(ctx,id)
	if err!=nil||plan.Status!="blocked"{t.Fatalf("rejection must block: %+v %v",plan,err)}
	if ok,err:=q.ClassPlanTransition(ctx,id,1,"pending_coordinator","pending_admin",23);err!=nil||ok{
		t.Fatalf("rejected exception bypassed coordinator: %v %v",ok,err)
	}
	if approved,err:=q.ClassPlanHasApprovedQuotaException(ctx,id);err!=nil||approved{
		t.Fatalf("rejected exception must not waive quota: %v %v",approved,err)
	}
}

func TestClassPlanExceptionRequiresApprovedDecisionForTransition(t *testing.T) {
	ctx:=context.Background()
	q:=NewQuerier(workflowQueryDB(t))
	id:=createQuotaTestPlan(t,q)
	if err:=q.ClassPlanRequestQuotaException(ctx,id,17,"Kelas butuh dimulai pekan ini");err!=nil{t.Fatal(err)}
	rec,err:=q.ClassPlanLatestException(ctx,id)
	if err!=nil{t.Fatal(err)}
	if ok,err:=q.ClassPlanTransitionQuotaDecision(ctx,id,1,rec.ID,true);err!=nil||ok{
		t.Fatalf("unreviewed exception must not move: %v %v",ok,err)
	}
}
