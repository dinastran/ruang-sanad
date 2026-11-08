-- +goose Up
-- +goose StatementBegin
-- Koneksi OAuth (Claude.ai / ChatGPT) dicatat sebagai baris ai_tokens
-- berjenis 'oauth' agar tampil, bisa dicabut, dan tercatat di audit seperti
-- token pribadi. Token akses/refresh-nya ada di oauth_tokens.
ALTER TABLE ai_tokens ADD COLUMN jenis TEXT NOT NULL DEFAULT 'pribadi';
ALTER TABLE ai_tokens ADD COLUMN oauth_client_id TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ai_tokens DROP COLUMN oauth_client_id;
ALTER TABLE ai_tokens DROP COLUMN jenis;
-- +goose StatementEnd
