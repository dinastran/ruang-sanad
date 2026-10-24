-- +goose Up
-- +goose StatementBegin
-- Riwayat analisis AI per lead. suggested_stage hanya saran, tidak mengubah leads.stage.
CREATE TABLE IF NOT EXISTS lead_analyses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lead_id INTEGER NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    ai_token_id INTEGER REFERENCES ai_tokens(id) ON DELETE SET NULL,
    summary TEXT NOT NULL DEFAULT '',
    persona TEXT NOT NULL DEFAULT '{}',
    interest_score INTEGER NOT NULL DEFAULT 0,
    main_objection TEXT NOT NULL DEFAULT '',
    suggested_stage TEXT NOT NULL DEFAULT '',
    suggested_next_action TEXT NOT NULL DEFAULT '',
    confidence TEXT NOT NULL DEFAULT 'rendah',
    analyzed_until_message_id INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_lead_analyses_lead ON lead_analyses(lead_id, id DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_lead_analyses_lead;
DROP TABLE IF EXISTS lead_analyses;
-- +goose StatementEnd
