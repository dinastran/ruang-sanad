package models

type OnboardingQuotaBoard struct {
 Classes []OnboardingQuotaClass `json:"classes"`
}

type OnboardingQuotaClass struct {
 KelasID int64 `json:"kelas_id"`
 Nama string `json:"nama"`
 Tipe string `json:"tipe"`
 Private bool `json:"private"`
 Kapasitas int64 `json:"kapasitas"`
 JumlahSantri int64 `json:"jumlah_santri"`
 KandidatAktif int64 `json:"kandidat_aktif"`
 TargetKuota int64 `json:"target_kuota"`
 SuggestedTarget int64 `json:"suggested_target"`
 Kekurangan int64 `json:"kekurangan"`
 FirstCandidateAt string `json:"first_candidate_at"`
 Day int `json:"day"`
 Phase string `json:"phase"`
 VacancyRequired bool `json:"vacancy_required"`
 VacancySentAt string `json:"vacancy_sent_at"`
 CanSendAlert bool `json:"can_send_alert"`
 ExceptionApproved bool `json:"exception_approved"`
 ExceptionReason string `json:"exception_reason"`
 ExceptionApprovedAt string `json:"exception_approved_at"`
 ExceptionApprovedBy int64 `json:"exception_approved_by"`
 GuruConfirmed bool `json:"guru_confirmed"`
 JadwalConfirmed bool `json:"jadwal_confirmed"`
 CanStart bool `json:"can_start"`
 IsActive bool `json:"is_active"`
 VacancyText string `json:"vacancy_text"`
}

type OnboardingQuotaAction struct {
 ID int64 `json:"id"`
 EventType string `json:"event_type"`
 Note string `json:"note"`
 ActorName string `json:"actor_name"`
 CreatedAt string `json:"created_at"`
}
