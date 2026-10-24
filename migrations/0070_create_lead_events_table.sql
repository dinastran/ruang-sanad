-- +goose Up
-- +goose StatementBegin
-- Timeline dan audit lead. event_id unik untuk idempotensi ingest dari bridge.
CREATE TABLE IF NOT EXISTS lead_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lead_id INTEGER NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    event_id TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    actor TEXT NOT NULL DEFAULT 'system',
    payload TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_lead_events_lead ON lead_events(lead_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_lead_events_created ON lead_events(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_lead_events_created;
DROP INDEX IF EXISTS idx_lead_events_lead;
DROP TABLE IF EXISTS lead_events;
-- +goose StatementEnd
