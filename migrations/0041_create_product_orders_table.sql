-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS product_orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_no TEXT UNIQUE,
    tanggal DATE NOT NULL,
    nama_pembeli TEXT NOT NULL,
    no_wa TEXT NOT NULL DEFAULT '',
    sumber TEXT NOT NULL CHECK (sumber IN ('whatsapp', 'instagram', 'marketplace', 'offline', 'lainnya')),
    status TEXT NOT NULL DEFAULT 'aktif' CHECK (status IN ('aktif', 'dibatalkan')),
    catatan TEXT NOT NULL DEFAULT '',
    dicatat_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    dibatalkan_at DATETIME,
    dibatalkan_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_product_orders_tanggal ON product_orders(tanggal, id);
CREATE INDEX idx_product_orders_status ON product_orders(status, tanggal);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_product_orders_status;
DROP INDEX IF EXISTS idx_product_orders_tanggal;
DROP TABLE IF EXISTS product_orders;
-- +goose StatementEnd
