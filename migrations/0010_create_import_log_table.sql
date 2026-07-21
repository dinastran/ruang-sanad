-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS import_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    nama_file TEXT NOT NULL,
    total_baris INTEGER NOT NULL DEFAULT 0,
    berhasil INTEGER NOT NULL DEFAULT 0,
    gagal INTEGER NOT NULL DEFAULT 0,
    catatan TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_import_log_user ON import_log(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_import_log_user;
DROP TABLE IF EXISTS import_log;
-- +goose StatementEnd
