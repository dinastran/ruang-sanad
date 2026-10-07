-- +goose Up
-- +goose StatementBegin
-- Optional per-guru ujroh rate that overrides the status default.
CREATE TABLE IF NOT EXISTS ujroh_tarif_guru (
    guru_id INTEGER PRIMARY KEY REFERENCES guru(id) ON DELETE CASCADE,
    nominal INTEGER NOT NULL CHECK (nominal >= 0),
    diubah_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ujroh_tarif_guru;
-- +goose StatementEnd
