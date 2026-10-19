-- +goose Up
-- +goose StatementBegin
-- One row per AI agent tool call: who asked, which tool, with what arguments,
-- and whether it succeeded. Tool results are not stored.
CREATE TABLE IF NOT EXISTS ai_audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    token_id INTEGER REFERENCES ai_tokens(id) ON DELETE SET NULL,
    tool TEXT NOT NULL,
    arguments TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'ok',
    pesan TEXT NOT NULL DEFAULT '',
    durasi_ms INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ai_audit_log_user ON ai_audit_log(user_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_ai_audit_log_user;
DROP TABLE IF EXISTS ai_audit_log;
-- +goose StatementEnd
