-- +goose Up
-- +goose StatementBegin
-- Riwayat sinkron data iklan Meta (otomatis harian atau manual).
-- status: running | success | failed.
CREATE TABLE IF NOT EXISTS meta_ads_sync_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trigger TEXT NOT NULL,
    date_from TEXT NOT NULL,
    date_to TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    rows INTEGER NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    started_at DATETIME NOT NULL,
    finished_at DATETIME
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS meta_ads_sync_runs;
-- +goose StatementEnd
