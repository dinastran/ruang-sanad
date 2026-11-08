-- +goose Up
-- +goose StatementBegin
-- Aplikasi AI (Claude.ai, ChatGPT, ...) yang mendaftar sendiri lewat OAuth
-- Dynamic Client Registration (RFC 7591) untuk terhubung ke /mcp.
-- client_secret_hash kosong = public client (wajib PKCE, tanpa secret).
CREATE TABLE IF NOT EXISTS oauth_clients (
    client_id TEXT PRIMARY KEY,
    client_secret_hash TEXT NOT NULL DEFAULT '',
    client_name TEXT NOT NULL,
    redirect_uris TEXT NOT NULL,
    token_endpoint_auth_method TEXT NOT NULL DEFAULT 'none',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS oauth_clients;
-- +goose StatementEnd
