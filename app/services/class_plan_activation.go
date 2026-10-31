package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/maulanashalihin/laju-go/app/queries"
)

func classPlanFrequencyLabel(f int64) string { if f==1 {return "1x/pekan"}; return "2x/pekan" }

var ErrPlanActivationConflict=errors.New("aktivasi kelas tidak lagi memenuhi syarat")

var planningWeekdays=[8]string{"","Senin","Selasa","Rabu","Kamis","Jumat","Sabtu","Ahad"}

func classPlanSchedule(slots []queries.ClassPlanWorkflowSlot) (string,error) {
	if len(slots)==0||len(slots)>2{return "",ErrPlanWorkflowInvalid}
	seen:=make(map[int]bool,len(slots))
	parts:=make([]string,0,len(slots))
	for _,slot:=range slots{
		s,e,ok:=planningInterval(PlanningSlot{Day:slot.Day,Start:slot.Start,End:slot.End})
		if !ok||e-s!=planningLessonMinutes||seen[slot.Day]{
			return "",ErrPlanWorkflowInvalid
		}
		seen[slot.Day]=true
		parts=append(parts,fmt.Sprintf("%s %s-%s",planningWeekdays[slot.Day],slot.Start,slot.End))
	}
	return strings.Join(parts,", "),nil
}

// Activate is the only operational write path for planned new regular classes.
// All source data are re-read inside one SQL transaction. A failure rolls back
// class, class routines, roster, plan state and audit event together.
func (s *ClassPlanWorkflowService) Activate(ctx context.Context,planID,version,actorID int64) (int64,error) {
	if planID<=0||version<=0||actorID<=0{return 0,ErrPlanWorkflowInvalid}
	tx,err:=s.q.BeginTx(ctx)
	if err!=nil{return 0,err}
	defer tx.Rollback()
	q:=s.q.WithTx(tx)
	if _,err:=s.actorRole(ctx,q,actorID,true,false);err!=nil{return 0,err}
	plan,err:=q.GetClassPlanWorkflow(ctx,planID)
	if errors.Is(err,sql.ErrNoRows){return 0,ErrPlanWorkflowNotFound}
	if err!=nil{return 0,err}
	if plan.Version!=version{return 0,ErrPlanWorkflowConflict}
	// Retry after a committed transaction: no new class or audit event.
	if plan.Status=="activated"&&plan.ActivatedKelasID>0&&
		plan.ConfirmedBy!=0&&plan.ApprovedBy!=0 {
		return plan.ActivatedKelasID,nil
	}
	if plan.Status!="pending_admin" || plan.TargetKelasID!=0 ||
		plan.ActivatedKelasID!=0||plan.Program!="reguler"||
		plan.ConfirmedBy==0||plan.ConfirmedVersion!=plan.Version||
		plan.ApprovedBy==0||actorID==plan.ConfirmedBy||
		plan.StudentCount<=0||plan.ReservedCount!=plan.StudentCount||
		plan.StudentCount>plan.Capacity{
		return 0,ErrPlanWorkflowConflict
	}
	violations,err:=s.preflight(ctx,q,plan)
	if err!=nil{return 0,err}
	if len(violations)>0{
		return 0,fmt.Errorf("%w: %s",ErrPlanWorkflowInvalid,violations[0].Message)
	}
	level,err:=q.ClassPlanWorkflowLevel(ctx,plan.ID)
	if err!=nil{return 0,err}
	slots,err:=q.ClassPlanWorkflowSlots(ctx,plan.ID)
	if err!=nil{return 0,err}
	if len(slots)!=int(plan.Frequency){return 0,ErrPlanWorkflowInvalid}
	schedule,err:=classPlanSchedule(slots)
	if err!=nil{return 0,err}
	asOf,err:=classPlanWIBDate()
	if err!=nil{return 0,err}
	candidates,err:=q.ClassPlanWorkflowCandidates(ctx,plan.ID)
	if err!=nil{return 0,err}
	if int64(len(candidates))!=plan.StudentCount {
		return 0,ErrPlanWorkflowConflict
	}
	classID,err:=q.CreateActivatedPlanningClass(ctx,plan,level,schedule)
	if err!=nil{return 0,err}
	if err:=q.InsertPlanningClassLevel(ctx,classID,level);err!=nil{return 0,err}
	for _,slot:=range slots {
		if err:=q.InsertPlanningClassRoutine(ctx,classID,actorID,slot.Day,slot.Start,asOf);err!=nil{
			return 0,err
		}
	}
	n,err:=q.CommitPlanningReservations(ctx,plan.ID)
	if err!=nil{return 0,err}
	if n!=plan.StudentCount{return 0,ErrPlanWorkflowConflict}
	for _,student:=range candidates{
		assigned,err:=q.AssignActivatedPlanningStudent(ctx,plan.ID,student.ID,classID,
			plan.Gender,plan.Angkatan,level,classPlanFrequencyLabel(plan.Frequency),schedule)
		if err!=nil{return 0,err}
		if !assigned{return 0,ErrPlanActivationConflict}
	}
	if err:=q.SetPlanningClassRosterCount(ctx,classID,n);err!=nil{return 0,err}
	activated,err:=q.MarkClassPlanActivated(ctx,plan.ID,version,classID,n)
	if err!=nil{return 0,err}
	if !activated{return 0,ErrPlanWorkflowConflict}
	if err:=q.AppendClassPlanWorkflowEvent(ctx,plan.ID,actorID,"planning.activated",
		"pending_admin","activated",fmt.Sprintf("Kelas #%d diaktifkan dengan %d santri",classID,n));err!=nil{
		return 0,err
	}
	if err:=tx.Commit();err!=nil{return 0,err}
	return classID,nil
}
