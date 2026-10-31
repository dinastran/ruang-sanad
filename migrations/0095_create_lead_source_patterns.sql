-- +goose Up
-- +goose StatementBegin
-- Teks pemicu sebuah sumber lead. teks_normal = huruf kecil tanpa tanda
-- baca, dipakai untuk pencocokan "mengandung" dan harus unik antar sumber.
CREATE TABLE IF NOT EXISTS lead_source_patterns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id INTEGER NOT NULL REFERENCES lead_sources(id) ON DELETE CASCADE,
    teks TEXT NOT NULL,
    teks_normal TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_lead_source_patterns_source ON lead_source_patterns(source_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS lead_source_patterns;
-- +goose StatementEnd
