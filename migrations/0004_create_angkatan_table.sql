-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS angkatan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kode TEXT NOT NULL UNIQUE,
    keterangan TEXT NOT NULL DEFAULT '',
    is_aktif INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_angkatan_kode ON angkatan(kode);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_angkatan_kode;
DROP TABLE IF EXISTS angkatan;
-- +goose StatementEnd
