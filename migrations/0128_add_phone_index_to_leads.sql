-- +goose Up
-- +goose StatementBegin
-- Dedup lead dari form pendaftaran berdasarkan nomor HP.
CREATE INDEX IF NOT EXISTS idx_leads_phone ON leads(phone);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_leads_phone;
-- +goose StatementEnd
