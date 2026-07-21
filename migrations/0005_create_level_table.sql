-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS level (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kode TEXT NOT NULL UNIQUE,
    nama TEXT NOT NULL DEFAULT '',
    urutan INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_level_kode ON level(kode);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_level_kode;
DROP TABLE IF EXISTS level;
-- +goose StatementEnd
