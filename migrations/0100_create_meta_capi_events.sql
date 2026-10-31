-- +goose Up
-- +goose StatementBegin
-- Antrean event Meta Conversions API (CAPI). Satu baris per lead per event
-- funnel (Lead, CompleteRegistration, InitiateCheckout, AddPaymentInfo,
-- Purchase); event_id unik sehingga event yang sama tidak pernah dobel.
-- status: pending | sent | failed | skipped.
CREATE TABLE IF NOT EXISTS meta_capi_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lead_id INTEGER NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    event_name TEXT NOT NULL,
    event_id TEXT NOT NULL UNIQUE,
    event_time DATETIME NOT NULL,
    value INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    fbtrace_id TEXT NOT NULL DEFAULT '',
    test_mode INTEGER NOT NULL DEFAULT 0,
    next_attempt_at DATETIME NOT NULL,
    sent_at DATETIME,
    created_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_meta_capi_events_due ON meta_capi_events(status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_meta_capi_events_lead ON meta_capi_events(lead_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS meta_capi_events;
-- +goose StatementEnd
