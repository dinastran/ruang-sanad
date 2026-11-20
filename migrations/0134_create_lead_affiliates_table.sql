-- +goose Up
-- +goose StatementBegin
-- Atribusi lead ke affiliate (satu per lead, first-click).
CREATE TABLE IF NOT EXISTS lead_affiliates (
    lead_id INTEGER PRIMARY KEY REFERENCES leads(id) ON DELETE CASCADE,
    affiliate_id INTEGER NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    sumber TEXT NOT NULL CHECK (sumber IN ('cookie', 'kode_form', 'kode_wa', 'manual')),
    diklik_at DATETIME,
    dikunci_at DATETIME NOT NULL,
    diubah_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_lead_affiliates_affiliate ON lead_affiliates(affiliate_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS lead_affiliates;
-- +goose StatementEnd
