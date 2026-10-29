package models

// OnboardingCaseListItem is the read model for PR 1.
type OnboardingCaseListItem struct {
	ID           int64  `json:"id"`
	SantriID     int64  `json:"santri_id"`
	Nama         string `json:"nama"`
	IDMahasantri string `json:"id_mahasantri"`
	Stage        string `json:"stage"`
	RegisteredAt string `json:"registered_at"`
	DeadlineAt   string `json:"deadline_at"`
}
