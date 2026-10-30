package services

import (
 "context"
 "database/sql"
 "testing"
 "time"

 "github.com/maulanashalihin/laju-go/app/queries"
 "github.com/stretchr/testify/require"
)

func mockMonitoringCase(id int64,stage string,registered,started time.Time) queries.OnboardingMonitoringCaseRow {
 r:=queries.OnboardingMonitoringCaseRow{
  ID:id,Stage:stage,RegisteredAt:registered,DeadlineAt:OnboardingDeadline(registered),
 }
 if !started.IsZero(){r.ActualStartedAt=sql.NullTime{Time:started,Valid:true}}
 return r
}

func TestOnboardingMonitoringSLADenominatorAndCancel(t *testing.T) {
 now:=time.Date(2026,11,20,12,0,0,0,onboardingWIB)
 day:=func(n int)time.Time{return time.Date(2026,11,n,10,0,0,0,onboardingWIB)}
 cases:=[]queries.OnboardingMonitoringCaseRow{
  mockMonitoringCase(1,"masuk_kelas",day(1),day(10)),
  mockMonitoringCase(2,"masuk_kelas",day(1),day(19)),
  mockMonitoringCase(3,"batal",day(1),time.Time{}),
  mockMonitoringCase(4,"waiting_kuota",day(1),time.Time{}),
  mockMonitoringCase(5,"matching_kelas",day(9),time.Time{}),
  mockMonitoringCase(6,"onboarding_baru",day(20),time.Time{}),
 }
 report:=computeOnboardingKPI(cases,nil,now)
 require.EqualValues(t,6,report.Total)
 require.EqualValues(t,2,report.Completed)
 require.EqualValues(t,1,report.Cancelled)
 require.EqualValues(t,3,report.InProgress)
 require.EqualValues(t,3,report.DueCases)
 require.EqualValues(t,1,report.OnTimeCases)
 require.EqualValues(t,1,report.Overdue)
 require.EqualValues(t,1,report.AtRisk)
 require.NotNil(t,report.SLARate)
 require.InDelta(t,33.3333,*report.SLARate,0.01)
 require.NotNil(t,report.AvgWaitingDays)
 require.InDelta(t,13.5,*report.AvgWaitingDays,0.01)
 empty:=computeOnboardingKPI(nil,nil,now)
 require.Nil(t,empty.SLARate)
 require.Nil(t,empty.AlertOnTimeRate)
 require.Nil(t,empty.AvgWaitingDays)
}

func TestOnboardingMonitoringVacancyDenominatorAndPrivate(t *testing.T) {
 anchor:=time.Date(2026,10,1,18,0,0,0,onboardingWIB)
 now:=anchor.AddDate(0,0,10)
 sentD5:=anchor.AddDate(0,0,5)
 sentD9:=anchor.AddDate(0,0,9)
 makeClass:=func(id int64,tipe string,sent sql.NullTime)queries.OnboardingClassControlRow{
  return queries.OnboardingClassControlRow{
   KelasID:id,Tipe:tipe,TargetKuota:sql.NullInt64{Int64:10,Valid:true},
   FirstCaseAt:anchor,FirstCandidateAt:sql.NullTime{Time:anchor,Valid:true},
   JumlahSantri:3,VacancyAlertSentAt:sent,
  }
 }
 classes:=[]queries.OnboardingClassControlRow{
  makeClass(1,"Reguler",sql.NullTime{Time:sentD5,Valid:true}),
  makeClass(2,"Reguler",sql.NullTime{Time:sentD9,Valid:true}),
  makeClass(3,"Reguler",sql.NullTime{}),
  makeClass(4,"Private",sql.NullTime{}),
 }
 r:=computeOnboardingKPI(nil,classes,now)
 require.EqualValues(t,3,r.AlertEligible)
 require.EqualValues(t,3,r.AlertDue)
 require.EqualValues(t,1,r.AlertOnTime)
 require.EqualValues(t,1,r.AlertPending)
 require.EqualValues(t,2,r.AlertLate)
 require.InDelta(t,33.33,*r.AlertOnTimeRate,0.01)
}

func TestOnboardingMonitoringNotificationsIdempotentAndUserScoped(t *testing.T) {
 db,svc,kelasID,_,picID,superID:=quotaFixture(t,"Reguler")
 ctx:=context.Background()
 require.NoError(t,svc.SetClassQuota(ctx,kelasID,superID,3))
 now:=time.Now().UTC()
 _,err:=db.Exec("UPDATE onboarding_class_controls SET first_candidate_at=? WHERE kelas_id=?",now.AddDate(0,0,-12),kelasID)
 require.NoError(t,err)
 first,err:=svc.TickOnboardingNotifications(ctx,now)
 require.NoError(t,err)
 require.EqualValues(t,4,first.NotificationsCreated)
 again,err:=svc.TickOnboardingNotifications(ctx,now)
 require.NoError(t,err)
 require.Zero(t,again.NotificationsCreated)
 var rows int
 require.NoError(t,db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type LIKE 'onboarding_%'").Scan(&rows))
 require.Equal(t,4,rows)
 for _,x:=range []struct{id int64;count int}{
  {picID,2},{superID,1},
 }{
  ns,err:=svc.UserNotices(ctx,x.id)
  require.NoError(t,err)
  require.Len(t,ns,x.count)
 }
 notices,err:=svc.UserNotices(ctx,picID)
 require.NoError(t,err)
 require.ErrorIs(t,svc.ReadUserNotice(ctx,superID,notices[0].ID),ErrOnboardingNotFound)
 require.NoError(t,svc.ReadUserNotice(ctx,picID,notices[0].ID))
 require.ErrorIs(t,svc.ReadUserNotice(ctx,picID,notices[0].ID),ErrOnboardingNotFound)
 refresh,err:=svc.UserNotices(ctx,picID)
 require.NoError(t,err)
 require.True(t,refresh[0].Read)
}

func TestOnboardingMonitoringCaseWarningAndOverdueDeduplicated(t *testing.T) {
 db,svc,caseID,picID,_:=onboardingFixture(t)
 ctx:=context.Background()
 require.NoError(t,svc.AssignPIC(ctx,caseID,picID,1))
 now:=time.Now().UTC()
 reg12:=now.AddDate(0,0,-12)
 _,err:=db.Exec("UPDATE onboarding_cases SET registered_at=?,deadline_at=? WHERE id=?",reg12,OnboardingDeadline(reg12),caseID)
 require.NoError(t,err)
 first,err:=svc.TickOnboardingNotifications(ctx,now)
 require.NoError(t,err)
 require.EqualValues(t,1,first.NotificationsCreated)
 again,err:=svc.TickOnboardingNotifications(ctx,now)
 require.NoError(t,err)
 require.Zero(t,again.NotificationsCreated)
 reg16:=now.AddDate(0,0,-16)
 _,err=db.Exec("UPDATE onboarding_cases SET registered_at=?,deadline_at=? WHERE id=?",reg16,OnboardingDeadline(reg16),caseID)
 require.NoError(t,err)
 overdue,err:=svc.TickOnboardingNotifications(ctx,now)
 require.NoError(t,err)
 require.EqualValues(t,1,overdue.NotificationsCreated)
 var total int
 require.NoError(t,db.QueryRow("SELECT COUNT(*) FROM notifications WHERE reference_type='onboarding_case' AND reference_id=?",caseID).Scan(&total))
 require.Equal(t,2,total)
}
