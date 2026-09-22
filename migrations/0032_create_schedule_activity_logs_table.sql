-- +goose Up
-- +goose StatementBegin
CREATE TABLE schedule_activity_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    jadwal_pertemuan_id INTEGER NOT NULL REFERENCES jadwal_pertemuan(id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT '',
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_schedule_activity_logs_schedule
ON schedule_activity_logs(jadwal_pertemuan_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE schedule_activity_logs;
-- +goose StatementEnd
