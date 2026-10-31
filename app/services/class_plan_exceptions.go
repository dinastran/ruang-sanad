package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

func validateQuotaDecision(action,note string) error {
	if action!="exception_approve"&&action!="exception_reject"{
		return ErrPlanWorkflowInvalid
	}
	if len(strings.TrimSpace(note))<10 || len(note)>500{
		return fmt.Errorf("%w: catatan keputusan exception minimal 10 karakter",ErrPlanWorkflowInvalid)
	}
	return nil
}

func onlyQuotaViolation(issues []PlanningViolation) bool {
	return len(issues)==1&&issues[0].Code=="MIN_QUOTA_NOT_MET"
}

// DecideQuotaException is an explicit super-admin-only override of the
// minimum class size. A successful approval returns to pending_coordinator;
// it does not approve the teacher, authorize placement or activate the class.
func (s *ClassPlanWorkflowService) DecideQuotaException(ctx context.Context,id,version,actorID int64,action,note string) error {
	note=strings.TrimSpace(note)
	if id<=0||version<=0||actorID<=0{return ErrPlanWorkflowInvalid}
	if err:=validateQuotaDecision(action,note);err!=nil{return err}
	actor,err:=s.q.GetUserByID(ctx,actorID)
	if err!=nil||actor.Role!=models.RoleSuperAdmin{return ErrPlanWorkflowForbidden}
	tx,err:=s.q.BeginTx(ctx)
	if err!=nil{return err}
	defer tx.Rollback()
	q:=s.q.WithTx(tx)
	plan,err:=q.GetClassPlanWorkflow(ctx,id)
	if errors.Is(err,sql.ErrNoRows){return ErrPlanWorkflowNotFound}
	if err!=nil{return err}
	if plan.Status!="pending_exception"||plan.Version!=version||plan.ConfirmedBy!=0||plan.ApprovedBy!=0{
		return ErrPlanWorkflowConflict
	}
	record,err:=q.ClassPlanLatestException(ctx,id)
	if errors.Is(err,sql.ErrNoRows){return ErrPlanWorkflowConflict}
	if err!=nil{return err}
	if record.Status!="requested"||record.RequestedBy==actorID{
		return ErrPlanWorkflowConflict
	}
	// No approver may override a failed hard rule or an already-satisfied
	// quota. Unchanged plan version must still match the latest snapshot.
	if action=="exception_approve"{
		violations,err:=s.preflight(ctx,q,plan)
		if err!=nil{return err}
		if !onlyQuotaViolation(violations){
			if len(violations)>0{return fmt.Errorf("%w: %s",ErrPlanWorkflowInvalid,violations[0].Message)}
			return fmt.Errorf("%w: exception tidak diperlukan ketika kuota terpenuhi",ErrPlanWorkflowInvalid)
		}
	}
	approved:=action=="exception_approve"
	decision:="rejected"
	event:="planning.exception_rejected"
	next:="blocked"
	if approved {
		decision="approved"
		event="planning.exception_approved"
		next="pending_coordinator"
	}
	ok,err:=q.ClassPlanDecideQuotaException(ctx,id,record.ID,actorID,version,decision,note)
	if err!=nil{return err}
	if !ok{return ErrPlanWorkflowConflict}
	ok,err=q.ClassPlanTransitionQuotaDecision(ctx,id,version,record.ID,approved)
	if err!=nil{return err}
	if !ok{return ErrPlanWorkflowConflict}
	if err:=q.AppendClassPlanWorkflowEvent(ctx,id,actorID,event,"pending_exception",next,note);err!=nil{
		return err
	}
	return tx.Commit()
}

// The existing approver and coordinator checks remain independent of this
// exception. A rejected exception is terminal in this slice; create a fresh
// draft if new candidates or a revised schedule are required.
var _ = queries.ClassPlanExceptionReview{}
