package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var (
	ErrPlanWorkflowInvalid = errors.New("draft perencanaan tidak valid")
	ErrPlanWorkflowConflict = errors.New("versi atau status perencanaan sudah berubah")
	ErrPlanWorkflowForbidden = errors.New("role tidak diizinkan")
	ErrPlanWorkflowNotFound = errors.New("perencanaan tidak ditemukan")
)

type ClassPlanDraftInput struct {
	Angkatan string `json:"angkatan"`
	Gender string `json:"gender"`
	Level string `json:"level"`
	GuruID int64 `json:"guru_id"`
	Frequency int64 `json:"frequency"`
	Capacity int64 `json:"capacity"`
	TargetKelasID int64 `json:"target_kelas_id"`
	Slots []PlanningSlot `json:"slots"`
	CandidateIDs []int64 `json:"candidate_ids"`
}

type ClassPlanWorkflowService struct { q *queries.Querier }

func NewClassPlanWorkflowService(q *queries.Querier) *ClassPlanWorkflowService {
	return &ClassPlanWorkflowService{q:q}
}

func (s *ClassPlanWorkflowService) List(ctx context.Context,actorID int64) ([]queries.ClassPlanWorkflowRow,error) {
	if _,err:=s.actorRole(ctx,s.q,actorID,false,false);err!=nil{return nil,err}
	return s.q.ListClassPlanWorkflow(ctx)
}

func (s *ClassPlanWorkflowService) actorRole(ctx context.Context,q *queries.Querier,actorID int64,allowAdmin,allowCoordinator bool) (string,error) {
	if actorID<=0{return "",ErrPlanWorkflowForbidden}
	user,err:=q.GetUserByID(ctx,actorID)
	if err!=nil{return "",ErrPlanWorkflowForbidden}
	role:=string(user.Role)
	if role==string(models.RoleSuperAdmin) {return role,nil}
	if allowAdmin && role==string(models.RoleAdminKelas){return role,nil}
	if allowCoordinator && role==string(models.RoleKoordinator){return role,nil}
	if !allowAdmin&&!allowCoordinator &&
		(role==string(models.RoleAdminKelas)||role==string(models.RoleKoordinator)){return role,nil}
	return "",ErrPlanWorkflowForbidden
}

func normalizeClassPlanDraft(input *ClassPlanDraftInput) error {
	input.Angkatan=strings.TrimSpace(input.Angkatan)
	input.Level=strings.TrimSpace(input.Level)
	if input.Angkatan==""||len(input.Angkatan)>100||input.Level==""||len(input.Level)>50||
		(input.Gender!="L"&&input.Gender!="P")||input.GuruID<=0||
		(input.Frequency!=1&&input.Frequency!=2)||input.Capacity<8||input.Capacity>10||
		input.TargetKelasID<0||len(input.Slots)!=int(input.Frequency)||
		len(input.CandidateIDs)==0||len(input.CandidateIDs)>int(input.Capacity) {
		return ErrPlanWorkflowInvalid
	}
	seen:=make(map[int]bool)
	for _,slot:=range input.Slots {
		start,end,ok:=planningInterval(slot)
		if !ok||end-start!=planningLessonMinutes||seen[slot.Day]{return ErrPlanWorkflowInvalid}
		seen[slot.Day]=true
	}
	ids:=make(map[int64]bool)
	for _,id:=range input.CandidateIDs {
		if id<=0||ids[id]{return ErrPlanWorkflowInvalid}
		ids[id]=true
	}
	return nil
}

func (s *ClassPlanWorkflowService) Create(ctx context.Context,actorID int64,input ClassPlanDraftInput) (int64,error) {
	if _,err:=s.actorRole(ctx,s.q,actorID,true,false);err!=nil{return 0,err}
	if err:=normalizeClassPlanDraft(&input);err!=nil{return 0,err}
	tx,err:=s.q.BeginTx(ctx);if err!=nil{return 0,err}
	defer tx.Rollback()
	q:=s.q.WithTx(tx)
	teacherOK,err:=q.ClassPlanWorkflowTeacherValid(ctx,input.GuruID,input.Gender,input.Level)
	if err!=nil{return 0,err}
	if !teacherOK{return 0,fmt.Errorf("%w: guru tidak aktif atau level belum disetujui",ErrPlanWorkflowInvalid)}
	plan:=queries.ClassPlanWorkflowRow{Program:"reguler",Angkatan:input.Angkatan,Gender:input.Gender,
		GuruID:input.GuruID,Frequency:input.Frequency,MinQuota:8,Capacity:input.Capacity,TargetKelasID:input.TargetKelasID}
	if input.TargetKelasID!=0 {
		eligible,err:=q.ClassPlanWorkflowTargetValid(ctx,input.TargetKelasID,plan,input.Level)
		if err!=nil{return 0,err}
		if !eligible{return 0,fmt.Errorf("%w: kelas existing tidak cocok",ErrPlanWorkflowInvalid)}
	}
	for _,id:=range input.CandidateIDs{
		free,err:=q.ClassPlanWorkflowCandidateFree(ctx,0,id)
		if err!=nil{return 0,err}
		if !free{return 0,fmt.Errorf("%w: santri sudah berada pada perencanaan aktif",ErrPlanWorkflowConflict)}
	}
	slots:=make([]queries.ClassPlanWorkflowSlot,0,len(input.Slots))
	for _,slot:=range input.Slots{slots=append(slots,queries.ClassPlanWorkflowSlot{Day:slot.Day,Start:slot.Start,End:slot.End})}
	id,err:=q.CreateClassPlanWorkflow(ctx,actorID,plan,slots,input.CandidateIDs,input.Level)
	if err!=nil{return 0,err}
	if err:=q.AppendClassPlanWorkflowEvent(ctx,id,actorID,"planning.draft_created","","draft","Draft manual dibuat");err!=nil{return 0,err}
	if err:=tx.Commit();err!=nil{return 0,err}
	return id,nil
}

func sameClassPlanOffer(slots []PlanningSlot,offer queries.ClassPlanOfferEvidence) bool {
	days,err:=parseHari(offer.Days)
	if err!=nil||len(days)!=len(slots){return false}
	start,sOK:=planningMinutes(offer.Start)
	end,eOK:=planningMinutes(offer.End)
	if !sOK||!eOK||end-start!=planningLessonMinutes{return false}
	for _,slot:=range slots{
		found:=false
		for _,day:=range days{if slot.Day==day&&slot.Start==offer.Start&&slot.End==offer.End{found=true;break}}
		if !found{return false}
	}
	return true
}

func (s *ClassPlanWorkflowService) preflight(ctx context.Context,q *queries.Querier,plan queries.ClassPlanWorkflowRow) ([]PlanningViolation,error) {
	level,err:=q.ClassPlanWorkflowLevel(ctx,plan.ID);if err!=nil{return nil,err}
	savedSlots,err:=q.ClassPlanWorkflowSlots(ctx,plan.ID);if err!=nil{return nil,err}
	slots:=make([]PlanningSlot,0,len(savedSlots))
	for _,a:=range savedSlots{slots=append(slots,PlanningSlot{Day:a.Day,Start:a.Start,End:a.End})}
	candidates,err:=q.ClassPlanWorkflowCandidates(ctx,plan.ID);if err!=nil{return nil,err}
	date,err:=classPlanWIBDate();if err!=nil{return nil,err}
	windows,err:=q.ListPlanningTeacherWindows(ctx,date);if err!=nil{return nil,err}
	approved,err:=q.ListPlanningTeacherLevels(ctx);if err!=nil{return nil,err}
	busy,err:=q.ClassPlanWorkflowBusy(ctx,date,plan.ID,plan.TargetKelasID);if err!=nil{return nil,err}
	unsafe,err:=q.ListPlanningUnsafeTeacherIDs(ctx,date);if err!=nil{return nil,err}
	limits,err:=q.ListPlanningWorkloadLimits(ctx);if err!=nil{return nil,err}
	teacher:=PlanningTeacher{ID:plan.GuruID,ApprovedLevels:make(map[string]bool),
		Available:make([]PlanningSlot,0),Occupied:make([]PlanningSlot,0)}
	for _,a:=range windows {if a.GuruID==plan.GuruID{
		teacher.Active=true;teacher.Gender=a.Gender
		teacher.Available=append(teacher.Available,PlanningSlot{Day:int(a.Day),Start:a.Start,End:a.End})
	}}
	for _,a:=range approved{if a.GuruID==plan.GuruID{teacher.ApprovedLevels[a.Level]=true}}
	for _,a:=range busy{if a.GuruID==plan.GuruID{
		teacher.Occupied=append(teacher.Occupied,PlanningSlot{Day:int(a.Day),Start:a.Start,End:a.End})
	}}
	if limit,ok:=limits[plan.GuruID];ok&&limit.Valid{
		max:=int(limit.Int64);teacher.MaxWeeklyMinutes=&max
	}
	teacher.CurrentWeeklyMinutes=len(teacher.Occupied)*planningLessonMinutes
	if unsafe[plan.GuruID]{
		teacher.Active=false
	}
	eligibility:=make([]PlanningCandidate,0,len(candidates))
	for _,r:=range candidates {
		free,err:=q.ClassPlanWorkflowCandidateFree(ctx,plan.ID,r.ID)
		if err!=nil{return nil,err}
		eligible:=free&&r.Status=="aktif"&&!r.ClassID.Valid&&r.Angkatan==plan.Angkatan&&
			(r.Stage=="matching_kelas"||r.Stage=="waiting_kuota")&&r.Level==level
		offers,err:=q.ClassPlanCandidateOffers(ctx,r.ID);if err!=nil{return nil,err}
		scheduleApproved:=false
		for _,offer:=range offers{if sameClassPlanOffer(slots,offer){scheduleApproved=true;break}}
		eligibility=append(eligibility,PlanningCandidate{ID:r.ID,Gender:r.Gender,Level:r.Level,
			PlacementEligible:eligible,ScheduleApproved:scheduleApproved,CurrentClassID:r.ClassID.Int64})
	}
	exceptionApproved:=false
	issues:=ValidatePlanningActivation(PlanningProposal{
		Program:plan.Program,Gender:plan.Gender,Frequency:int(plan.Frequency),Capacity:int(plan.Capacity),
		Slots:slots,Teacher:teacher,Candidates:eligibility,ExceptionApproved:exceptionApproved,
	})
	if len(candidates)==0||int64(len(candidates))!=plan.StudentCount{
		issues=append(issues,PlanningViolation{Code:"ROSTER_MISMATCH",Message:"Kandidat draft kosong atau jumlah tidak konsisten"})
	}
	if len(candidates)>0&&len(candidates)>int(plan.Capacity){
		issues=append(issues,PlanningViolation{Code:"CAPACITY_EXCEEDED",Message:"Kandidat melebihi kapasitas"})
	}
	if plan.TargetKelasID>0{
		ok,err:=q.ClassPlanWorkflowTargetValid(ctx,plan.TargetKelasID,plan,level)
		if err!=nil{return nil,err}
		all,err:=q.ListPlanningExistingClasses(ctx);if err!=nil{return nil,err}
		found:=false
		if ok {
			classSlots,err:=q.ListPlanningCurrentClassSlots(ctx,date)
			if err!=nil{return nil,err}
			for _,k:=range all {
				if k.ID!=plan.TargetKelasID{continue}
				own:=make([]queries.PlanningClassSlot,0)
				for _,slot:=range classSlots {if slot.KelasID==k.ID{own=append(own,slot)}}
				max:=int(k.Capacity);if max>10{max=10}
				if planningSameSlots(slots,own)&&max-int(k.Occupancy)>=len(candidates){found=true}
			}
		}
		if !found{issues=append(issues,PlanningViolation{Code:"TARGET_CLASS_UNAVAILABLE",Message:"Kelas tujuan tidak cocok, jadwal berbeda, atau kapasitas tidak cukup"})}
	}
	return issues,nil
}

func classPlanWIBDate() (string,error) {
	zone,err:=time.LoadLocation("Asia/Jakarta");if err!=nil{return "",err}
	return time.Now().In(zone).Format(time.DateOnly),nil
}

// Submit advances only a valid persisted draft. A sub-quota request stays in
// pending_exception, which this phase deliberately cannot approve.
func (s *ClassPlanWorkflowService) Action(ctx context.Context,id,version,actorID int64,action,reason string) error {
	if id<=0||version<=0||actorID<=0{return ErrPlanWorkflowInvalid}
	reason=strings.TrimSpace(reason)
	if len(reason)>500{return ErrPlanWorkflowInvalid}
	var admin,coordinator bool
	switch action {
	case "submit","admin_approve":admin=true
	case "coordinator_confirm":coordinator=true
	default:return ErrPlanWorkflowInvalid
	}
	if _,err:=s.actorRole(ctx,s.q,actorID,admin,coordinator);err!=nil{return err}
	tx,err:=s.q.BeginTx(ctx);if err!=nil{return err}
	defer tx.Rollback()
	q:=s.q.WithTx(tx)
	plan,err:=q.GetClassPlanWorkflow(ctx,id)
	if errors.Is(err,sql.ErrNoRows){return ErrPlanWorkflowNotFound}
	if err!=nil{return err}
	if plan.Version!=version{return ErrPlanWorkflowConflict}
	from,to,event:="","",""
	switch action{
	case "submit":
		if plan.Status!="draft"{return ErrPlanWorkflowConflict}
		from="draft";to="pending_coordinator";event="planning.submitted"
	case "coordinator_confirm":
		if plan.Status!="pending_coordinator"{return ErrPlanWorkflowConflict}
		from="pending_coordinator";to="pending_admin";event="planning.coordinator_confirmed"
	case "admin_approve":
		if plan.Status!="pending_admin"||plan.ApprovedBy!=0||
			plan.ConfirmedBy==0||plan.ConfirmedVersion!=plan.Version||actorID==plan.ConfirmedBy{
			return ErrPlanWorkflowConflict
		}
		from="pending_admin";to="pending_admin";event="planning.admin_approved"
	}
	issues,err:=s.preflight(ctx,q,plan);if err!=nil{return err}
	if action=="submit"&&len(issues)>0 {
		// The only provisional state allowed is a valid sub-quota draft. It
		// cannot go through coordinator/admin approval in this phase.
		if len(issues)==1&&issues[0].Code=="MIN_QUOTA_NOT_MET"{
			to="pending_exception"
		}else{return fmt.Errorf("%w: %s",ErrPlanWorkflowInvalid,issues[0].Message)}
	}else if len(issues)>0 {
		return fmt.Errorf("%w: %s",ErrPlanWorkflowInvalid,issues[0].Message)
	}
	transition:=to
	if action=="admin_approve"{transition="admin_approved"}
	ok,err:=q.ClassPlanTransition(ctx,id,version,from,transition,actorID)
	if err!=nil{return err}
	if !ok{return ErrPlanWorkflowConflict}
	if reason==""{reason=event}
	if err:=q.AppendClassPlanWorkflowEvent(ctx,id,actorID,event,from,to,reason);err!=nil{return err}
	return tx.Commit()
}
