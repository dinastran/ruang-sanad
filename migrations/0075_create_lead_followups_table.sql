-- +goose Up
-- +goose StatementBegin
-- Follow-up terjadwal (Script 30 Hari): disusun AI/CS, disetujui manusia, dikirim lewat bridge.
CREATE TABLE IF NOT EXISTS lead_followups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lead_id INTEGER NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    step TEXT NOT NULL,
    scheduled_at DATETIME NOT NULL,
    draft_text TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending_approval',
    created_by_kind TEXT NOT NULL DEFAULT 'user',
    created_by INTEGER,
    approved_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    approved_at DATETIME,
    sent_at DATETIME,
    wa_message_id TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_lead_followups_status ON lead_followups(status, scheduled_at);
CREATE INDEX IF NOT EXISTS idx_lead_followups_lead_step ON lead_followups(lead_id, step);
CREATE UNIQUE INDEX IF NOT EXISTS uq_lead_followups_active_step ON lead_followups(lead_id, step)
    WHERE status IN ('pending_approval', 'approved');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS uq_lead_followups_active_step;
DROP INDEX IF EXISTS idx_lead_followups_lead_step;
DROP INDEX IF EXISTS idx_lead_followups_status;
DROP TABLE IF EXISTS lead_followups;
-- +goose StatementEnd
