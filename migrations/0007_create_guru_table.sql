-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS guru (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nama TEXT NOT NULL UNIQUE,
    jenis_kelamin TEXT NOT NULL DEFAULT '',
    is_aktif INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX idx_guru_nama ON guru(nama);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_guru_nama;
DROP TABLE IF EXISTS guru;
-- +goose StatementEnd
