-- +goose Up
-- +goose StatementBegin
-- Access token (umur pendek) dan refresh token (dirotasi tiap dipakai) milik
-- satu koneksi OAuth (ai_tokens jenis 'oauth'). Hanya hash yang disimpan.
CREATE TABLE IF NOT EXISTS oauth_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ai_token_id INTEGER NOT NULL REFERENCES ai_tokens(id) ON DELETE CASCADE,
    jenis TEXT NOT NULL CHECK (jenis IN ('access', 'refresh')),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    revoked_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_oauth_tokens_grant ON oauth_tokens(ai_token_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_oauth_tokens_grant;
DROP TABLE IF EXISTS oauth_tokens;
-- +goose StatementEnd
