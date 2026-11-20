-- +goose Up
-- +goose StatementBegin
-- Token buat-password untuk affiliate eksternal yang disetujui (72 jam, sekali pakai).
CREATE TABLE IF NOT EXISTS affiliate_invites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    affiliate_id INTEGER NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    used_at DATETIME,
    created_at DATETIME NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS affiliate_invites;
-- +goose StatementEnd
