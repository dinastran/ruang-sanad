-- +goose Up
-- +goose StatementBegin
-- Transkrip chat CS per lead. File media tidak disimpan, hanya jenisnya.
CREATE TABLE IF NOT EXISTS lead_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lead_id INTEGER NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    wa_message_id TEXT NOT NULL DEFAULT '',
    direction TEXT NOT NULL,
    sender TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    media_type TEXT NOT NULL DEFAULT '',
    sent_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_lead_messages_wa_id ON lead_messages(lead_id, wa_message_id) WHERE wa_message_id <> '';
CREATE INDEX IF NOT EXISTS idx_lead_messages_lead_sent ON lead_messages(lead_id, sent_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_lead_messages_lead_sent;
DROP INDEX IF EXISTS uq_lead_messages_wa_id;
DROP TABLE IF EXISTS lead_messages;
-- +goose StatementEnd
