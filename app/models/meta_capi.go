package models

// MetaCAPIStatus is the Meta Conversions API settings page: configuration
// (secrets never included), queue counts and the latest events.
type MetaCAPIStatus struct {
	Enabled        bool             `json:"enabled"`
	DatasetID      string           `json:"dataset_id"` // masked
	APIVersion     string           `json:"api_version"`
	TestMode       bool             `json:"test_mode"`
	AppSecretProof bool             `json:"app_secret_proof"`
	Counts         map[string]int64 `json:"counts"` // pending, sent, failed, skipped
	// ResendableTest counts events sent in test mode that Meta would still
	// accept in production (under 7 days old).
	ResendableTest int64              `json:"resendable_test"`
	Mapping        []MetaCAPIMapping  `json:"mapping"`
	Events         []MetaCAPIEventRow `json:"events"`
}

// MetaCAPIMapping explains which CRM step sends which Meta event.
type MetaCAPIMapping struct {
	Event string `json:"event"`
	Tahap string `json:"tahap"`
}

type MetaCAPIEventRow struct {
	ID        int64  `json:"id"`
	LeadID    int64  `json:"lead_id"`
	LeadNama  string `json:"lead_nama"`
	Event     string `json:"event"`
	EventTime string `json:"event_time"`
	Value     int64  `json:"value"`
	Status    string `json:"status"`
	Attempts  int64  `json:"attempts"`
	LastError string `json:"last_error"`
	TestMode  bool   `json:"test_mode"`
	SentAt    string `json:"sent_at"`
}
