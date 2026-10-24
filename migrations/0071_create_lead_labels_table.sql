-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS lead_labels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nama TEXT NOT NULL UNIQUE,
    warna TEXT NOT NULL DEFAULT 'neutral',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO lead_labels (nama, warna) VALUES
    ('prioritas', 'brand'),
    ('butuh telepon', 'secondary'),
    ('wali/anak', 'neutral'),
    ('minta jadwal khusus', 'neutral');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS lead_labels;
-- +goose StatementEnd
