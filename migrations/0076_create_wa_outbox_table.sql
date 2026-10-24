-- +goose Up
-- +goose StatementBegin
-- Antrean kirim WA bersama (follow-up sekarang; tagihan/reminder nanti).
CREATE TABLE IF NOT EXISTS wa_outbox (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    purpose TEXT NOT NULL,
    channel TEXT NOT NULL,
    to_chat_id TEXT NOT NULL,
    body TEXT NOT NULL,
    reference_type TEXT NOT NULL DEFAULT '',
    reference_id INTEGER,
    not_before DATETIME NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    sent_at DATETIME,
    wa_message_id TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_wa_outbox_status ON wa_outbox(status, not_before);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_wa_outbox_status;
DROP TABLE IF EXISTS wa_outbox;
-- +goose StatementEnd
