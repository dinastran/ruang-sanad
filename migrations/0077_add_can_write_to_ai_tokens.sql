-- +goose Up
-- +goose StatementBegin
-- Izin tulis terbatas untuk tool CRM (simpan analisis, usulkan follow-up,
-- tambah label). Default 0: token lama dan token biasa tetap baca saja.
-- Hanya super_admin yang bisa membuat token dengan can_write = 1.
ALTER TABLE ai_tokens ADD COLUMN can_write INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ai_tokens DROP COLUMN can_write;
-- +goose StatementEnd
