package models

import "encoding/json"

// --- Ingest dari bridge CS AI (POST /integrations/cs-ai/events) ---

// LeadIngestRequest is one batch of events sent by hermes-cs-bridge.
type LeadIngestRequest struct {
	Events []LeadIngestEvent `json:"events"`
}

// LeadIngestEvent is one bridge event. Data is decoded per type by the service.
type LeadIngestEvent struct {
	EventID    string          `json:"event_id"`
	Type       string          `json:"type"`
	OccurredAt string          `json:"occurred_at"`
	ChatID     string          `json:"chat_id"`
	Phone      string          `json:"phone"`
	WaName     string          `json:"wa_name"`
	Data       json.RawMessage `json:"data"`
}

type LeadIngestRejected struct {
	EventID string `json:"event_id"`
	Error   string `json:"error"`
}

type LeadIngestResult struct {
	Accepted   int                  `json:"accepted"`
	Duplicates int                  `json:"duplicates"`
	Rejected   []LeadIngestRejected `json:"rejected,omitempty"`
}

// --- Halaman CRM ---

type LeadLabelResponse struct {
	ID    int64  `json:"id"`
	Nama  string `json:"nama"`
	Warna string `json:"warna"`
}

// LeadListItem is one card in /app/leads.
type LeadListItem struct {
	ID             int64               `json:"id"`
	Nama           string              `json:"nama"`
	WaName         string              `json:"wa_name"`
	Phone          string              `json:"phone"`
	Stage          string              `json:"stage"`
	Status         string              `json:"status"`
	PaketKode      string              `json:"paket_kode"`
	NeedsReview    bool                `json:"needs_review"`
	AIPaused       bool                `json:"ai_paused"`
	AssignedUserID int64               `json:"assigned_user_id"`
	AssignedName   string              `json:"assigned_name"`
	LastInboundAt  string              `json:"last_inbound_at"`
	LastActivityAt string              `json:"last_activity_at"`
	Labels         []LeadLabelResponse `json:"labels"`
}

type LeadListParams struct {
	Status      string `query:"status" json:"status"`
	Stage       string `query:"stage" json:"stage"`
	ReachedStage string `query:"reached_stage" json:"reached_stage"`
	LostReason string `query:"lost_reason" json:"lost_reason"`
	LabelID     int64  `query:"label" json:"label"`
	Mine        bool   `query:"mine" json:"mine"`
	NeedsReview bool   `query:"review" json:"review"`
	Search      string `query:"q" json:"q"`
	DateField   string `query:"date_field" json:"date_field"`
	From        string `query:"from" json:"from"`
	To          string `query:"to" json:"to"`
	Cursor      string `query:"cursor" json:"cursor"`
	// Set by the handler from the session, not from the query string.
	UserID int64 `query:"-" json:"-"`
}

type LeadListResponse struct {
	Items      []LeadListItem `json:"items"`
	NextCursor string         `json:"next_cursor"`
	// Total and Counts cover every page: Counts is keyed by the current stage
	// for open leads and by status (won, lost, do_not_contact) otherwise.
	Total  int64            `json:"total"`
	Counts map[string]int64 `json:"counts"`
}

type LeadMessageResponse struct {
	ID        int64  `json:"id"`
	Direction string `json:"direction"`
	Sender    string `json:"sender"`
	Body      string `json:"body"`
	MediaType string `json:"media_type"`
	SentAt    string `json:"sent_at"`
}

type LeadEventResponse struct {
	ID        int64          `json:"id"`
	Type      string         `json:"type"`
	Actor     string         `json:"actor"`
	ActorName string         `json:"actor_name"`
	Payload   map[string]any `json:"payload"`
	CreatedAt string         `json:"created_at"`
}

type LeadAnalysisResponse struct {
	ID                  int64          `json:"id"`
	Summary             string         `json:"summary"`
	Persona             map[string]any `json:"persona"`
	InterestScore       int64          `json:"interest_score"`
	MainObjection       string         `json:"main_objection"`
	SuggestedStage      string         `json:"suggested_stage"`
	SuggestedNextAction string         `json:"suggested_next_action"`
	Confidence          string         `json:"confidence"`
	CreatedAt           string         `json:"created_at"`
}

type LeadFollowupResponse struct {
	ID          int64  `json:"id"`
	Step        string `json:"step"`
	ScheduledAt string `json:"scheduled_at"`
	DraftText   string `json:"draft_text"`
	Reason      string `json:"reason"`
	Status      string `json:"status"`
	SentAt      string `json:"sent_at"`
	Error       string `json:"error"`
	// CreatedByKind is "ai" or "user".
	CreatedByKind string   `json:"created_by_kind"`
	Warnings      []string `json:"warnings"`
}

// LeadDetail is everything /app/leads/:id shows about one lead.
type LeadDetail struct {
	ID                 int64                  `json:"id"`
	WaChatID           string                 `json:"wa_chat_id"`
	Phone              string                 `json:"phone"`
	WaName             string                 `json:"wa_name"`
	Nama               string                 `json:"nama"`
	Usia               int64                  `json:"usia"`
	Domisili           string                 `json:"domisili"`
	JenisKelamin       string                 `json:"jenis_kelamin"`
	PaketKode          string                 `json:"paket_kode"`
	KelasKode          string                 `json:"kelas_kode"`
	WaktuBelajar       string                 `json:"waktu_belajar"`
	RiwayatBelajar     string                 `json:"riwayat_belajar"`
	BelajarDimana      string                 `json:"belajar_dimana"`
	UntukSiapa         string                 `json:"untuk_siapa"`
	JadwalKode         string                 `json:"jadwal_kode"`
	JadwalLabel        string                 `json:"jadwal_label"`
	// JadwalOptions lists every schedule (for the select on the lead page).
	JadwalOptions []JadwalOption `json:"jadwal_options"`
	Stage              string                 `json:"stage"`
	Status             string                 `json:"status"`
	LostReason         string                 `json:"lost_reason"`
	FollowupPermission string                 `json:"followup_permission"`
	AssignedUserID     int64                  `json:"assigned_user_id"`
	AssignedName       string                 `json:"assigned_name"`
	PaymentAmount      int64                  `json:"payment_amount"`
	PaymentInfoSentAt  string                 `json:"payment_info_sent_at"`
	ProofReceivedAt    string                 `json:"proof_received_at"`
	VerifiedAt         string                 `json:"verified_at"`
	VerifiedAmount     int64                  `json:"verified_amount"`
	VerifiedDate       string                 `json:"verified_date"`
	ConvertedAt        string                 `json:"converted_at"`
	SantriID           int64                  `json:"santri_id"`
	NeedsReview        bool                   `json:"needs_review"`
	ReviewFields       []string               `json:"review_fields"`
	AIPaused           bool                   `json:"ai_paused"`
	AIPausedUntil      string                 `json:"ai_paused_until"`
	ContactedAgain     bool                   `json:"contacted_again"`
	LastInboundAt      string                 `json:"last_inbound_at"`
	LastOutboundAt     string                 `json:"last_outbound_at"`
	CreatedAt          string                 `json:"created_at"`
	Labels             []LeadLabelResponse    `json:"labels"`
	Messages           []LeadMessageResponse  `json:"messages"`
	HasOlderMessages   bool                   `json:"has_older_messages"`
	Events             []LeadEventResponse    `json:"events"`
	Analysis           *LeadAnalysisResponse  `json:"analysis"`
	Followups          []LeadFollowupResponse `json:"followups"`
	// ConfirmationReady: a payment confirmation template is set (PRD 6.3).
	ConfirmationReady bool `json:"confirmation_ready"`
}

type LeadUpdateRequest struct {
	Nama           string `json:"nama"`
	Usia           int64  `json:"usia"`
	Domisili       string `json:"domisili"`
	JenisKelamin   string `json:"jenis_kelamin"`
	PaketKode      string `json:"paket_kode"`
	KelasKode      string `json:"kelas_kode"`
	WaktuBelajar   string `json:"waktu_belajar"`
	RiwayatBelajar string `json:"riwayat_belajar"`
	BelajarDimana  string `json:"belajar_dimana"`
	UntukSiapa     string `json:"untuk_siapa"`
	// JadwalKode: nil = unchanged, "" = clear, otherwise a jadwal_penawaran kode.
	JadwalKode   *string `json:"jadwal_kode"`
	MarkReviewed bool    `json:"mark_reviewed"`
}

// JadwalOption is one schedule shown in a select.
type JadwalOption struct {
	Kode   string `json:"kode"`
	Label  string `json:"label"`
	IsBuka bool   `json:"is_buka"`
}

type LeadStageRequest struct {
	Stage string `json:"stage"`
}

type LeadStatusRequest struct {
	Status     string `json:"status"`
	LostReason string `json:"lost_reason"`
}

type LeadAssignRequest struct {
	UserID int64 `json:"user_id"`
}

type LeadLabelRequest struct {
	LabelID int64 `json:"label_id"`
}

type LeadVerifyRequest struct {
	Amount  int64  `json:"amount"`
	Date    string `json:"date"`
	Note    string `json:"note"`
	NoProof bool   `json:"no_proof_confirmed"`
	// SendConfirmation queues crm_settings.payment_verified_template to the
	// lead through the bridge (PRD 6.3, default off).
	SendConfirmation bool `json:"send_confirmation"`
}

type LeadConvertRequest struct {
	Nama          string `json:"nama"`
	JenisKelamin  string `json:"jenis_kelamin"`
	Usia          int64  `json:"usia"`
	Domisili      string `json:"domisili"`
	NoWA          string `json:"no_wa"`
	KelasKode     string `json:"kelas_kode"`
	Nominal       int64  `json:"nominal"`
	TanggalDaftar string `json:"tanggal_daftar"`
	Angkatan      string `json:"angkatan"`
	// LinkSantriID links the lead to an existing santri instead of creating one.
	LinkSantriID int64 `json:"link_santri_id"`
	// ForceNew confirms creating a new santri although one with the same no_wa exists.
	ForceNew bool `json:"force_new"`
}

type LeadDuplicateSantri struct {
	ID           int64  `json:"id"`
	Nama         string `json:"nama"`
	IDMahasantri string `json:"id_mahasantri"`
	NoWA         string `json:"no_wa"`
}

type LeadUserOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// --- Laporan ---

type LeadFunnelStage struct {
	Stage      string  `json:"stage"`
	Reached    int64   `json:"reached"`
	Conversion float64 `json:"conversion"`
}

type LeadCountByKey struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type LeadWeekRow struct {
	Week  string `json:"week"`
	Leads int64  `json:"leads"`
	Won   int64  `json:"won"`
}

type LeadReport struct {
	From             string            `json:"from"`
	To               string            `json:"to"`
	TotalLeads       int64             `json:"total_leads"`
	Won              int64             `json:"won"`
	Lost             int64             `json:"lost"`
	DoNotContact     int64             `json:"do_not_contact"`
	Open             int64             `json:"open"`
	Funnel           []LeadFunnelStage `json:"funnel"`
	MedianHoursToWon float64           `json:"median_hours_to_won"`
	LostReasons      []LeadCountByKey  `json:"lost_reasons"`
	Objections       []LeadCountByKey  `json:"objections"`
	Weekly           []LeadWeekRow     `json:"weekly"`
	FollowupSent     int64             `json:"followup_sent"`
	FollowupReplied  int64             `json:"followup_replied"`
	FollowupWon      int64             `json:"followup_won"`
}

// --- Integrasi WhatsApp (super_admin) ---

type CRMSettingsResponse struct {
	PaketKeKelasKode            map[string]string `json:"paket_ke_kelas_kode"`
	FollowupEnabled             bool              `json:"followup_enabled"`
	FollowupSendStart           string            `json:"followup_send_start"`
	FollowupSendEnd             string            `json:"followup_send_end"`
	FollowupMinGapSeconds       int64             `json:"followup_min_gap_seconds"`
	FollowupDailyCap            int64             `json:"followup_daily_cap"`
	TranscriptRetentionDaysLost int64             `json:"transcript_retention_days_lost"`
	PaymentVerifiedTemplate     string            `json:"payment_verified_template"`
	// PersonaBlockedKeywords: one word/phrase per line (PRD 9.3).
	PersonaBlockedKeywords string `json:"persona_blocked_keywords"`
	// FollowupOnlyLabel limits follow-up sends to leads with this label ("" = all).
	FollowupOnlyLabel string `json:"followup_only_label"`
}

type CRMSettingsRequest = CRMSettingsResponse

type IntegrationStatus struct {
	Name    string `json:"name"`
	Target  string `json:"target"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type IntegrationPage struct {
	Settings          CRMSettingsResponse `json:"settings"`
	IngestEnabled     bool                `json:"ingest_enabled"`
	OutboundReady     bool                `json:"outbound_ready"`
	LastBridgeEvent   string              `json:"last_bridge_event"`
	BridgeEventsToday int64               `json:"bridge_events_today"`
	Statuses          []IntegrationStatus `json:"statuses"`
}

// --- Follow-up (Fase D) ---

// LeadFollowupRequest creates or edits a follow-up. ScheduledAt is
// "YYYY-MM-DDTHH:MM" (or with a space) in WIB.
type LeadFollowupRequest struct {
	Step        string `json:"step"`
	ScheduledAt string `json:"scheduled_at"`
	DraftText   string `json:"draft_text"`
	Reason      string `json:"reason"`
}

type FollowupBulkRequest struct {
	IDs []int64 `json:"ids"`
}

type FollowupQueueItem struct {
	ID             int64                 `json:"id"`
	LeadID         int64                 `json:"lead_id"`
	LeadName       string                `json:"lead_name"`
	LeadStage      string                `json:"lead_stage"`
	Step           string                `json:"step"`
	ScheduledAt    string                `json:"scheduled_at"`
	DraftText      string                `json:"draft_text"`
	Reason         string                `json:"reason"`
	Status         string                `json:"status"`
	CreatedByKind  string                `json:"created_by_kind"`
	Error          string                `json:"error"`
	OutboxStatus   string                `json:"outbox_status"`
	Warnings       []string              `json:"warnings"`
	Blocked        bool                  `json:"blocked"`
	RecentMessages []LeadMessageResponse `json:"recent_messages"`
}

type FailedSendItem struct {
	OutboxID  int64  `json:"outbox_id"`
	LeadID    int64  `json:"lead_id"`
	LeadName  string `json:"lead_name"`
	Purpose   string `json:"purpose"`
	Step      string `json:"step"`
	Body      string `json:"body"`
	Error     string `json:"error"`
	Attempts  int64  `json:"attempts"`
	UpdatedAt string `json:"updated_at"`
}

type FollowupQueuePage struct {
	Pending         []FollowupQueueItem `json:"pending"`
	Approved        []FollowupQueueItem `json:"approved"`
	Failed          []FailedSendItem    `json:"failed"`
	FollowupEnabled bool                `json:"followup_enabled"`
	SendStart       string              `json:"send_start"`
	SendEnd         string              `json:"send_end"`
	DailyCap        int64               `json:"daily_cap"`
	SentToday       int64               `json:"sent_today"`
	OnlyLabel       string              `json:"only_label"`
	CanApprove      bool                `json:"can_approve"`
}
