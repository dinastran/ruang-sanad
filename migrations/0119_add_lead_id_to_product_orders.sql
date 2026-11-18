-- +goose Up
-- +goose StatementBegin
-- Pesanan program yang dibuat dari closing lead (peserta program pendek).
ALTER TABLE product_orders ADD COLUMN lead_id INTEGER REFERENCES leads(id) ON DELETE SET NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_product_orders_lead ON product_orders(lead_id) WHERE lead_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_product_orders_lead;
ALTER TABLE product_orders DROP COLUMN lead_id;
-- +goose StatementEnd
