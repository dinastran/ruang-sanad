package models

// LeadDashboard is the read-only CRM operating and acquisition overview.
type LeadDashboardDay struct {
 Day string `json:"day"`
 Leads int64 `json:"leads"`
 Closing int64 `json:"closing"`
}

type LeadDashboardPriority struct {
 ID int64 `json:"id"`
 Name string `json:"name"`
 Stage string `json:"stage"`
 Reason string `json:"reason"`
 ActivityAt string `json:"activity_at"`
}

type LeadDashboard struct {
 From string `json:"from"`
 To string `json:"to"`
 Report *LeadReport `json:"report"`
 ClosingPeriod int64 `json:"closing_period"`
 PreviousLeads int64 `json:"previous_leads"`
 PreviousClosing int64 `json:"previous_closing"`
 ReviewCount int64 `json:"review_count"`
 Daily []LeadDashboardDay `json:"daily"`
 FollowupPending int64 `json:"followup_pending"`
 FollowupApproved int64 `json:"followup_approved"`
 FollowupFailed int64 `json:"followup_failed"`
 FollowupEligible int64 `json:"followup_eligible"`
 FollowupReplied int64 `json:"followup_replied"`
 Priorities []LeadDashboardPriority `json:"priorities"`
 // Sources is the period cohort per lead source (landing page / iklan).
 Sources []LeadSourceStat `json:"sources"`
 Insights []string `json:"insights"`
}
