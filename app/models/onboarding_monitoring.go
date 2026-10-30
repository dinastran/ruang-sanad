package models

// Aggregate KPIs are calculated from all onboarding cases, not a paginated
// Kanban subset. Nil rates mean no eligible denominator (displayed as N/A).
type OnboardingKPIReport struct {
 AsOf string `json:"as_of"`
 Total int64 `json:"total"`
 InProgress int64 `json:"in_progress"`
 Completed int64 `json:"completed"`
 Cancelled int64 `json:"cancelled"`
 DueCases int64 `json:"due_cases"`
 OnTimeCases int64 `json:"on_time_cases"`
 SLARate *float64 `json:"sla_rate"`
 AtRisk int64 `json:"at_risk"`
 Overdue int64 `json:"overdue"`
 AvgWaitingDays *float64 `json:"avg_waiting_days"`
 AlertEligible int64 `json:"alert_eligible"`
 AlertDue int64 `json:"alert_due"`
 AlertOnTime int64 `json:"alert_on_time"`
 AlertPending int64 `json:"alert_pending"`
 AlertLate int64 `json:"alert_late"`
 AlertOnTimeRate *float64 `json:"alert_on_time_rate"`
 Stages map[string]int64 `json:"stages"`
}

type OnboardingMonitoringResult struct {
 NotificationsCreated int64 `json:"notifications_created"`
}

type OnboardingUserNotice struct {
 ID int64 `json:"id"`
 Type string `json:"type"`
 Title string `json:"title"`
 Message string `json:"message"`
 ActionURL string `json:"action_url"`
 Read bool `json:"read"`
 CreatedAt string `json:"created_at"`
}
