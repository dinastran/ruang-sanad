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
	ResendableTest int64 `json:"resendable_test"`
	// EventSourceURL is the default landing page (META_CAPI_EVENT_SOURCE_URL).
	EventSourceURL string `json:"event_source_url"`
	// SourcesWithoutURL lists active lead sources with no landing page URL:
	// their events fall back to EventSourceURL, or go without one.
	SourcesWithoutURL []string           `json:"sources_without_url"`
	Mapping           []MetaCAPIMapping  `json:"mapping"`
	Events            []MetaCAPIEventRow `json:"events"`
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
	EventID   string `json:"event_id"`
	FBTraceID string `json:"fbtrace_id"`
	EventTime string `json:"event_time"`
	Value     int64  `json:"value"`
	Status    string `json:"status"`
	Attempts  int64  `json:"attempts"`
	LastError string `json:"last_error"`
	TestMode  bool   `json:"test_mode"`
	SentAt    string `json:"sent_at"`
}

// MetaCAPIAnalytics measures outgoing CRM events by event_time (WIB).
// It does not represent Meta-attributed conversions, revenue, or match quality.
type MetaCAPIAnalytics struct {
 From string `json:"from"`
 To string `json:"to"`
 Total int64 `json:"total"`
 Sent int64 `json:"sent"` // sent to production
 TestSent int64 `json:"test_sent"`
 Pending int64 `json:"pending"`
 Failed int64 `json:"failed"`
 Skipped int64 `json:"skipped"`
 PurchaseValue int64 `json:"purchase_value"` // sent production Purchase events only
 Days []MetaCAPIDay `json:"days"`
 Events []MetaCAPIEventSummary `json:"events"`
}

type MetaCAPIDay struct {
 Day string `json:"day"`
 Sent int64 `json:"sent"`
 TestSent int64 `json:"test_sent"`
 Pending int64 `json:"pending"`
 Failed int64 `json:"failed"`
 Skipped int64 `json:"skipped"`
}

type MetaCAPIEventSummary struct {
 Event string `json:"event"`
 Total int64 `json:"total"`
 Sent int64 `json:"sent"`
 TestSent int64 `json:"test_sent"`
 Pending int64 `json:"pending"`
 Failed int64 `json:"failed"`
 Skipped int64 `json:"skipped"`
 PurchaseValue int64 `json:"purchase_value"`
}
