-- +goose Up
-- +goose StatementBegin
-- Kehadiran peserta (pesanan program) per sesi. cara: 'link' = peserta isi
-- sendiri lewat link absensi, 'manual' = dicatat admin.
CREATE TABLE IF NOT EXISTS program_kehadiran (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sesi_id INTEGER NOT NULL REFERENCES program_sesi(id) ON DELETE CASCADE,
    order_id INTEGER NOT NULL REFERENCES product_orders(id) ON DELETE CASCADE,
    cara TEXT NOT NULL DEFAULT 'link' CHECK (cara IN ('link', 'manual')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (sesi_id, order_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS program_kehadiran;
-- +goose StatementEnd
