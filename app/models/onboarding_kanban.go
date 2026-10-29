package models

// OnboardingBoard is a paginated view of onboarding cases. Its stage counts
// describe the whole dataset, not only the current page.
type OnboardingBoard struct {
	Cases    []OnboardingBoardCard `json:"cases"`
	Counts   map[string]int64      `json:"counts"`
	PICs     []OnboardingPICOption `json:"pics"`
	Total    int64                 `json:"total"`
	Page     int64                 `json:"page"`
	PageSize int64                 `json:"page_size"`
}

type OnboardingBoardCard struct {
	ID              int64  `json:"id"`
	SantriID        int64  `json:"santri_id"`
	Nama            string `json:"nama"`
	IDMahasantri    string `json:"id_mahasantri"`
	Stage           string `json:"stage"`
	PICUserID       int64  `json:"pic_user_id"`
	PICName         string `json:"pic_name"`
	RegisteredAt    string `json:"registered_at"`
	DeadlineAt      string `json:"deadline_at"`
	ActualStartedAt string `json:"actual_started_at,omitempty"`
	KelasID         int64  `json:"kelas_id"`
	KelasNama       string `json:"kelas_nama"`
}

type OnboardingPICOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type OnboardingActivityItem struct {
	ID        int64  `json:"id"`
	EventType string `json:"event_type"`
	FromStage string `json:"from_stage"`
	ToStage   string `json:"to_stage"`
	Note      string `json:"note"`
	ActorName string `json:"actor_name"`
	CreatedAt string `json:"created_at"`
}
