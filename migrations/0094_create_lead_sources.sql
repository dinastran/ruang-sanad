-- +goose Up
-- +goose StatementBegin
-- Sumber lead: landing page / iklan asal calon santri. Dikenali dari teks
-- pemicu (lead_source_patterns) pada pesan WhatsApp pertama.
CREATE TABLE IF NOT EXISTS lead_sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nama TEXT NOT NULL UNIQUE,
    keterangan TEXT NOT NULL DEFAULT '',
    is_aktif INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS lead_sources;
-- +goose StatementEnd
