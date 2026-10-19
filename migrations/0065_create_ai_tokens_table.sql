-- +goose Up
-- +goose StatementBegin
-- Personal access tokens for AI agents (MCP). Only the SHA-256 hash of the
-- token is stored; the plaintext is shown to the user once at creation.
CREATE TABLE IF NOT EXISTS ai_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nama TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    prefix TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    last_used_at DATETIME,
    revoked_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ai_tokens_user ON ai_tokens(user_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_ai_tokens_user;
DROP TABLE IF EXISTS ai_tokens;
-- +goose StatementEnd
