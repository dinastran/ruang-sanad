-- +goose Up
-- +goose StatementBegin
-- Recurring ("tetap") component amounts per guru, added to every month that is
-- not yet locked.
CREATE TABLE IF NOT EXISTS ujroh_komponen_guru (
    komponen_id INTEGER NOT NULL REFERENCES ujroh_komponen(id) ON DELETE CASCADE,
    guru_id INTEGER NOT NULL REFERENCES guru(id) ON DELETE CASCADE,
    nominal INTEGER NOT NULL CHECK (nominal >= 0),
    diubah_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (komponen_id, guru_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ujroh_komponen_guru;
-- +goose StatementEnd
