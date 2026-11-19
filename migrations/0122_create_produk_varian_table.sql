-- +goose Up
-- +goose StatementBegin
-- Varian harga per produk (mis. Tahsin Reguler 1x / 2x per pekan).
-- paket_kode disalin ke leads.paket_kode saat pendaftar memilih varian.
CREATE TABLE IF NOT EXISTS produk_varian (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    produk_id INTEGER NOT NULL REFERENCES produk(id) ON DELETE CASCADE,
    nama TEXT NOT NULL,
    harga INTEGER NOT NULL DEFAULT 0 CHECK (harga >= 0),
    paket_kode TEXT NOT NULL DEFAULT '',
    urutan INTEGER NOT NULL DEFAULT 0,
    is_aktif INTEGER NOT NULL DEFAULT 1 CHECK (is_aktif IN (0, 1)),
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_produk_varian_produk ON produk_varian(produk_id, urutan);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS produk_varian;
-- +goose StatementEnd
