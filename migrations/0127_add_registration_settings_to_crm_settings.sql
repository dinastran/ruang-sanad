-- +goose Up
-- +goose StatementBegin
ALTER TABLE crm_settings ADD COLUMN cs_wa_number TEXT NOT NULL DEFAULT '';
ALTER TABLE crm_settings ADD COLUMN payment_expiry_hours INTEGER NOT NULL DEFAULT 2;
ALTER TABLE crm_settings ADD COLUMN bank_manual_info TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE crm_settings DROP COLUMN bank_manual_info;
ALTER TABLE crm_settings DROP COLUMN payment_expiry_hours;
ALTER TABLE crm_settings DROP COLUMN cs_wa_number;
-- +goose StatementEnd
