-- +goose Up
-- +goose StatementBegin
CREATE TABLE class_monitoring_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    jadwal_pertemuan_id INTEGER NOT NULL REFERENCES jadwal_pertemuan(id) ON DELETE CASCADE,
    note TEXT NOT NULL,
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_class_monitoring_notes_schedule
ON class_monitoring_notes(jadwal_pertemuan_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE class_monitoring_notes;
-- +goose StatementEnd
