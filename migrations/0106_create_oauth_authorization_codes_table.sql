-- +goose Up
-- +goose StatementBegin
-- Kode otorisasi sekali pakai (berumur beberapa menit) yang diterbitkan
-- setelah pengguna menyetujui akses di halaman persetujuan. Hanya hash yang
-- disimpan; code_challenge untuk verifikasi PKCE (S256).
CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
    code_hash TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    can_write INTEGER NOT NULL DEFAULT 0,
    expires_at DATETIME NOT NULL,
    used_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS oauth_authorization_codes;
-- +goose StatementEnd
