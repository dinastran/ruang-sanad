package services

import (
 "context"
 "database/sql"
 "testing"
 "time"

 "github.com/stretchr/testify/require"
)

func quotaFixture(t *testing.T,tipe string) (*sql.DB,*OnboardingService,int64,int64,int64,int64) {
 t.Helper()
 db,s,caseID,picID,csID:=onboardingFixture(t)
 _,err:=db.Exec("INSERT INTO users(email,name,role) VALUES ('spv-pr3@example.com','SPV PR3','super_admin')")
 require.NoError(t,err)
 var superID int64
 require.NoError(t,db.QueryRow("SELECT id FROM users WHERE email='spv-pr3@example.com'").Scan(&superID))
 _,err=db.Exec("INSERT INTO guru(nama,jenis_kelamin,is_aktif) VALUES ('Guru PR3','L',1)")
 require.NoError(t,err)
 var guruID int64
 require.NoError(t,db.QueryRow("SELECT id FROM guru WHERE nama='Guru PR3'").Scan(&guruID))
 r,err:=db.Exec(`INSERT INTO kelas
 (kunci_kelas,angkatan,tipe,jenis_kelamin,level,frekuensi,jadwal,sub_index,nama_kelas,guru_id,kapasitas,jumlah_santri,is_aktif)
 VALUES ('PR3-TEST','AKA38',?,'L','Dasar','1x','Senin',1,'Kelas PR3',?,10,0,1)`,tipe,guruID)
 require.NoError(t,err)
 kelasID,err:=r.LastInsertId()
 require.NoError(t,err)
 _,err=db.Exec("UPDATE santri SET kelas_id=?, is_lengkap=1 WHERE id=(SELECT santri_id FROM onboarding_cases WHERE id=?)",kelasID,caseID)
 require.NoError(t,err)
 return db,s,kelasID,caseID,picID,superID
}

func TestOnboardingQuotaPhaseCalendarWIB(t *testing.T) {
 anchor:=time.Date(2026,10,29,23,59,0,0,onboardingWIB)
 cases:=[]struct{day int;phase string}{
  {0,"Matching"},{3,"Matching"},{4,"Fill"},{7,"Fill"},
  {8,"Push"},{10,"Push"},{11,"Resolve"},{13,"Resolve"},
  {14,"Start"},{15,"Overdue"},{20,"Overdue"},
 }
 for _,c:=range cases {
  now:=time.Date(2026,10,29,0,0,0,0,onboardingWIB).AddDate(0,0,c.day)
  day,phase:=OnboardingClassPhase(anchor,now)
  require.Equal(t,c.day,day);require.Equal(t,c.phase,phase)
 }
}

func TestOnboardingQuotaMustBeSuperAdminAndWithinCapacity(t *testing.T) {
 db,s,kelasID,_,picID,superID:=quotaFixture(t,"Reguler")
 ctx:=context.Background()
 require.ErrorIs(t,s.SetClassQuota(ctx,kelasID,picID,2),ErrOnboardingInvalid)
 require.ErrorIs(t,s.SetClassQuota(ctx,kelasID,superID,0),ErrOnboardingInvalid)
 require.ErrorIs(t,s.SetClassQuota(ctx,kelasID,superID,11),ErrOnboardingInvalid)
 require.NoError(t,s.SetClassQuota(ctx,kelasID,superID,5))
 before,err:=s.ClassControls(ctx);require.NoError(t,err)
 require.Len(t,before.Classes,1)
 require.EqualValues(t,5,before.Classes[0].TargetKuota)
 require.EqualValues(t,10,before.Classes[0].Kapasitas)
 require.EqualValues(t,1,before.Classes[0].JumlahSantri)
 require.EqualValues(t,4,before.Classes[0].Kekurangan)
 anchor:=before.Classes[0].FirstCandidateAt
 require.NoError(t,s.SetClassQuota(ctx,kelasID,superID,2))
 after,err:=s.ClassControls(ctx);require.NoError(t,err)
 require.Equal(t,anchor,after.Classes[0].FirstCandidateAt)
 require.EqualValues(t,2,after.Classes[0].TargetKuota)
 actions,err:=s.ClassActions(ctx,kelasID);require.NoError(t,err)
 require.Len(t,actions,2)
 require.Equal(t,"quota_set",actions[0].EventType)
 var deadline time.Time
 require.NoError(t,db.QueryRow("SELECT deadline_at FROM onboarding_cases LIMIT 1").Scan(&deadline))
 require.False(t,deadline.IsZero())
}

func TestOnboardingQuotaVacancyAlertAndFollowupsAudited(t *testing.T) {
 db,s,kelasID,_,picID,superID:=quotaFixture(t,"Reguler")
 ctx:=context.Background()
 require.NoError(t,s.SetClassQuota(ctx,kelasID,superID,3))
 require.ErrorIs(t,s.ClassAction(ctx,kelasID,picID,"vacancy_sent","WA ke CS"),ErrOnboardingInvalid)
 _,err:=db.Exec("UPDATE onboarding_class_controls SET first_candidate_at=? WHERE kelas_id=?",time.Now().UTC().AddDate(0,0,-8),kelasID)
 require.NoError(t,err)
 require.NoError(t,s.ClassAction(ctx,kelasID,picID,"vacancy_sent","Sudah dikirim melalui WA ke CS"))
 require.ErrorIs(t,s.ClassAction(ctx,kelasID,picID,"vacancy_sent","Duplikat"),ErrOnboardingConflict)
 require.NoError(t,s.ClassAction(ctx,kelasID,picID,"marketing_followup","Koordinasi promosi kelas"))
 require.NoError(t,s.ClassAction(ctx,kelasID,picID,"cs_followup","Follow up calon santri"))
 board,err:=s.ClassControls(ctx);require.NoError(t,err)
 require.False(t,board.Classes[0].CanSendAlert)
 require.NotEmpty(t,board.Classes[0].VacancySentAt)
 actions,err:=s.ClassActions(ctx,kelasID);require.NoError(t,err)
 require.Len(t,actions,4)
 require.Equal(t,"cs_followup",actions[0].EventType)
}

func TestOnboardingQuotaPrivateExemptAndExceptionRoleGate(t *testing.T) {
 db,s,kelasID,_,picID,superID:=quotaFixture(t,"Private")
 ctx:=context.Background()
 require.NoError(t,s.SetClassQuota(ctx,kelasID,superID,1))
 board,err:=s.ClassControls(ctx);require.NoError(t,err)
 require.True(t,board.Classes[0].Private)
 require.False(t,board.Classes[0].VacancyRequired)
 require.ErrorIs(t,s.ClassAction(ctx,kelasID,picID,"vacancy_sent","Sudah dikirim"),ErrOnboardingInvalid)
 require.ErrorIs(t,s.ClassAction(ctx,kelasID,picID,"exception_approved","Coba approve"),ErrOnboardingInvalid)
 _,err=db.Exec("UPDATE onboarding_class_controls SET first_candidate_at=? WHERE kelas_id=?",time.Now().UTC().AddDate(0,0,-12),kelasID)
 require.NoError(t,err)
 require.ErrorIs(t,s.ClassAction(ctx,kelasID,superID,"exception_approved","Tidak di bawah target"),ErrOnboardingInvalid)
}

func TestOnboardingQuotaExceptionUnblocksKanbanAndResetsOnTargetChange(t *testing.T) {
 db,s,kelasID,caseID,picID,superID:=quotaFixture(t,"Reguler")
 ctx:=context.Background()
 require.NoError(t,s.SetClassQuota(ctx,kelasID,superID,3))
 _,err:=db.Exec("UPDATE onboarding_class_controls SET first_candidate_at=? WHERE kelas_id=?",time.Now().UTC().AddDate(0,0,-12),kelasID)
 require.NoError(t,err)
 require.NoError(t,s.AssignPIC(ctx,caseID,picID,superID))
 require.NoError(t,s.Move(ctx,caseID,superID,"matching_kelas","",""))
 require.NoError(t,s.Move(ctx,caseID,superID,"waiting_kuota","",""))
 require.ErrorIs(t,s.Move(ctx,caseID,superID,"siap_mulai","",""),ErrOnboardingInvalid)
 require.ErrorIs(t,s.ClassAction(ctx,kelasID,picID,"exception_approved","Kebutuhan khusus"),ErrOnboardingInvalid)
 require.NoError(t,s.ClassAction(ctx,kelasID,superID,"exception_approved","Disetujui agar kelas tidak terlambat"))
 require.NoError(t,s.ClassAction(ctx,kelasID,superID,"exception_revoked","Review ulang"))
 require.ErrorIs(t,s.Move(ctx,caseID,superID,"siap_mulai","",""),ErrOnboardingInvalid)
 require.NoError(t,s.ClassAction(ctx,kelasID,superID,"exception_approved","Dikonfirmasi ulang"))
 require.NoError(t,s.SetClassQuota(ctx,kelasID,superID,2))
 require.ErrorIs(t,s.Move(ctx,caseID,superID,"siap_mulai","",""),ErrOnboardingInvalid,"quota update must revoke exception")
 require.NoError(t,s.ClassAction(ctx,kelasID,superID,"exception_approved","Memulai di bawah kuota"))
 require.NoError(t,s.Move(ctx,caseID,superID,"siap_mulai","",""))
 var stage string
 require.NoError(t,db.QueryRow("SELECT stage FROM onboarding_cases WHERE id=?",caseID).Scan(&stage))
 require.Equal(t,"siap_mulai",stage)
}

func TestOnboardingQuotaAuditFailureRollsBackAlert(t *testing.T) {
 db,s,kelasID,_,picID,superID:=quotaFixture(t,"Reguler")
 ctx:=context.Background()
 require.NoError(t,s.SetClassQuota(ctx,kelasID,superID,3))
 _,err:=db.Exec("UPDATE onboarding_class_controls SET first_candidate_at=? WHERE kelas_id=?",time.Now().UTC().AddDate(0,0,-8),kelasID)
 require.NoError(t,err)
 _,err=db.Exec(`CREATE TRIGGER fail_pr3_action BEFORE INSERT ON onboarding_class_action_logs
 BEGIN SELECT RAISE(ABORT,'audit pr3 unavailable'); END`)
 require.NoError(t,err)
 require.ErrorContains(t,s.ClassAction(ctx,kelasID,picID,"vacancy_sent","WA dikirim"),"audit pr3 unavailable")
 var sent sql.NullTime
 require.NoError(t,db.QueryRow("SELECT vacancy_alert_sent_at FROM onboarding_class_controls WHERE kelas_id=?",kelasID).Scan(&sent))
 require.False(t,sent.Valid)
}
