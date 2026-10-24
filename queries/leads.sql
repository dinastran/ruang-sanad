-- CRM Leads (lead to closing). Dynamic list and report queries live in app/queries/leads_list.go.

-- name: GetLeadByID :one
SELECT * FROM leads WHERE id = ?;

-- name: GetLeadByChatID :one
SELECT * FROM leads WHERE wa_chat_id = ?;

-- name: CreateLead :one
INSERT INTO leads (wa_chat_id, phone, wa_name, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateLead :exec
UPDATE leads SET
    phone = ?, wa_name = ?, nama = ?, usia = ?, domisili = ?, jenis_kelamin = ?,
    paket_kode = ?, kelas_kode = ?, waktu_belajar = ?, riwayat_belajar = ?, belajar_dimana = ?,
    untuk_siapa = ?, stage = ?, status = ?, lost_reason = ?, followup_permission = ?,
    assigned_user_id = ?, payment_amount = ?, payment_info_sent_at = ?, proof_received_at = ?,
    verified_at = ?, verified_by = ?, needs_review = ?, last_inbound_at = ?, last_outbound_at = ?,
    updated_at = ?
WHERE id = ?;

-- name: LinkLeadSantri :execrows
UPDATE leads SET santri_id = ?, status = 'won', converted_at = ?, converted_by = ?, updated_at = ?
WHERE id = ? AND santri_id IS NULL AND status = 'open' AND stage = 'terverifikasi';

-- name: DeleteLead :execrows
DELETE FROM leads WHERE id = ?;

-- name: LeadEventExists :one
SELECT COUNT(*) FROM lead_events WHERE event_id = ?;

-- name: CreateLeadEvent :exec
INSERT INTO lead_events (lead_id, event_id, type, actor, payload, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ListLeadEvents :many
SELECT * FROM lead_events WHERE lead_id = ? ORDER BY created_at DESC, id DESC LIMIT ?;

-- name: CreateLeadMessage :execrows
INSERT OR IGNORE INTO lead_messages (lead_id, wa_message_id, direction, sender, body, media_type, sent_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListLeadMessagesLatest :many
SELECT * FROM lead_messages WHERE lead_id = ? ORDER BY sent_at DESC, id DESC LIMIT ?;

-- name: ListLeadMessagesBefore :many
SELECT * FROM lead_messages
WHERE lead_id = sqlc.arg(lead_id)
  AND (sent_at < sqlc.arg(before_sent_at) OR (sent_at = sqlc.arg(before_sent_at) AND id < sqlc.arg(before_id)))
ORDER BY sent_at DESC, id DESC
LIMIT sqlc.arg(row_limit);

-- name: GetLeadMessageByID :one
SELECT * FROM lead_messages WHERE id = ? AND lead_id = ?;

-- name: CountLeadMessages :one
SELECT COUNT(*) FROM lead_messages WHERE lead_id = ?;

-- name: CancelActiveLeadFollowups :execrows
UPDATE lead_followups SET status = 'cancelled', error = ?, updated_at = ?
WHERE lead_id = ? AND status IN ('pending_approval', 'approved');

-- name: CancelQueuedOutboxForLead :execrows
UPDATE wa_outbox SET status = 'cancelled', last_error = ?, updated_at = ?
WHERE status = 'queued' AND reference_type = 'lead_followup'
  AND reference_id IN (SELECT id FROM lead_followups WHERE lead_id = ?);

-- name: MarkLeadFollowupSent :execrows
UPDATE lead_followups SET status = 'sent', sent_at = ?, wa_message_id = ?, error = '', updated_at = ?
WHERE id = ? AND lead_id = ? AND status <> 'sent';

-- name: ListLeadFollowups :many
SELECT * FROM lead_followups WHERE lead_id = ? ORDER BY scheduled_at DESC, id DESC;

-- name: GetLatestLeadAnalysis :one
SELECT * FROM lead_analyses WHERE lead_id = ? ORDER BY id DESC LIMIT 1;

-- name: ListLeadLabels :many
SELECT * FROM lead_labels ORDER BY nama ASC;

-- name: GetLeadLabelByID :one
SELECT * FROM lead_labels WHERE id = ?;

-- name: AddLeadLabelLink :execrows
INSERT OR IGNORE INTO lead_label_links (lead_id, label_id, created_by, created_at) VALUES (?, ?, ?, ?);

-- name: RemoveLeadLabelLink :execrows
DELETE FROM lead_label_links WHERE lead_id = ? AND label_id = ?;

-- name: ListLabelsForLead :many
SELECT l.* FROM lead_labels l
JOIN lead_label_links k ON k.label_id = l.id
WHERE k.lead_id = ?
ORDER BY l.nama ASC;

-- name: GetCRMSettings :one
SELECT * FROM crm_settings WHERE id = 1;

-- name: UpdateCRMSettings :exec
UPDATE crm_settings SET
    paket_ke_kelas_kode = ?, followup_enabled = ?, followup_send_start = ?, followup_send_end = ?,
    followup_min_gap_seconds = ?, followup_daily_cap = ?, transcript_retention_days_lost = ?,
    payment_verified_template = ?, persona_blocked_keywords = ?, updated_by = ?, updated_at = ?
WHERE id = 1;

-- name: ListUserIDsByRole :many
SELECT id FROM users WHERE role = ? ORDER BY id;

-- name: ListUsersByRoles :many
SELECT id, name, role FROM users WHERE role IN (sqlc.slice(roles)) ORDER BY name;

-- name: ListAngkatanAktif :many
SELECT * FROM angkatan WHERE is_aktif = 1 ORDER BY id DESC;

-- name: GetLatestBridgeEvent :many
SELECT * FROM lead_events WHERE actor = 'bridge' ORDER BY id DESC LIMIT 1;

-- name: CountLeadEventsSince :one
SELECT COUNT(*) FROM lead_events WHERE actor = 'bridge' AND created_at >= ?;

-- name: ClearLeadEventNotesForClosedLeads :execrows
UPDATE lead_events SET payload = '{}'
WHERE type IN ('handoff', 'payment.verified', 'ai.paused', 'ai.resumed') AND payload <> '{}'
  AND lead_id IN (SELECT id FROM leads WHERE status IN ('lost', 'do_not_contact') AND updated_at < ?);

-- name: DeleteLeadMessagesForClosedLeads :execrows
DELETE FROM lead_messages WHERE lead_id IN (
    SELECT id FROM leads WHERE status IN ('lost', 'do_not_contact') AND updated_at < ?
);

-- name: CreateLeadAnalysis :one
INSERT INTO lead_analyses (
    lead_id, ai_token_id, summary, persona, interest_score, main_objection,
    suggested_stage, suggested_next_action, confidence, analyzed_until_message_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: SetLeadLastAnalyzedMessage :exec
UPDATE leads SET last_analyzed_message_id = sqlc.arg(message_id)
WHERE id = sqlc.arg(id) AND last_analyzed_message_id < sqlc.arg(message_id);

-- name: GetLeadLabelByName :one
SELECT * FROM lead_labels WHERE lower(nama) = lower(sqlc.arg(nama));

-- name: CreateLeadFollowup :one
INSERT INTO lead_followups (
    lead_id, step, scheduled_at, draft_text, reason, status, created_by_kind, created_by, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, 'pending_approval', ?, ?, ?, ?)
RETURNING id;
