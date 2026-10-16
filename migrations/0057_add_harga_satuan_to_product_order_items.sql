-- +goose Up
-- +goose StatementBegin
-- Snapshot harga per item saat order dibuat, agar perubahan harga produk tidak
-- mengubah nilai order lama. Order sebelum migrasi ini bernilai 0.
ALTER TABLE product_order_items ADD COLUMN harga_satuan INTEGER NOT NULL DEFAULT 0 CHECK (harga_satuan >= 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE product_order_items DROP COLUMN harga_satuan;
-- +goose StatementEnd
