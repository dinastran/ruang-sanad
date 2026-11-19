-- +goose Up
-- +goose StatementBegin
-- Jejak setiap submit form pendaftaran. ref = token acak halaman bayar.
CREATE TABLE IF NOT EXISTS lead_registrations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lead_id INTEGER NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    link_id INTEGER NOT NULL REFERENCES registration_links(id) ON DELETE CASCADE,
    ref TEXT NOT NULL UNIQUE,
    payload TEXT NOT NULL DEFAULT '{}',
    ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_lead_registrations_lead ON lead_registrations(lead_id, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS lead_registrations;
-- +goose StatementEnd
