-- +goose Up
-- +goose StatementBegin
-- Harga jual default produk (rupiah). Dipakai sebagai harga awal item order.
ALTER TABLE produk ADD COLUMN harga INTEGER NOT NULL DEFAULT 0 CHECK (harga >= 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE produk DROP COLUMN harga;
-- +goose StatementEnd
