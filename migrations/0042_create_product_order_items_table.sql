-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS product_order_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL REFERENCES product_orders(id) ON DELETE CASCADE,
    produk_id INTEGER NOT NULL REFERENCES produk(id) ON DELETE RESTRICT,
    produk_batch_id INTEGER REFERENCES produk_batch(id) ON DELETE RESTRICT,
    qty INTEGER NOT NULL CHECK (qty > 0),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_product_order_items_order ON product_order_items(order_id, id);
CREATE INDEX idx_product_order_items_product ON product_order_items(produk_id, produk_batch_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_product_order_items_product;
DROP INDEX IF EXISTS idx_product_order_items_order;
DROP TABLE IF EXISTS product_order_items;
-- +goose StatementEnd
