-- +goose Up
-- +goose StatementBegin
ALTER TABLE produk ADD COLUMN minimum_stock INTEGER NOT NULL DEFAULT 5 CHECK (minimum_stock >= 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE produk DROP COLUMN minimum_stock;
-- +goose StatementEnd
