-- +goose Up
-- +goose StatementBegin
-- Lock state of a month's ujroh recap. A month is locked while dikunci_at is
-- set; unlocking keeps the row so the last unlock stays auditable.
CREATE TABLE IF NOT EXISTS ujroh_bulan (
    bulan TEXT PRIMARY KEY CHECK (length(bulan) = 7),
    dikunci_at DATETIME,
    dikunci_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    dibuka_at DATETIME,
    dibuka_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ujroh_bulan;
-- +goose StatementEnd
