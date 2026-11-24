-- +goose Up
-- +goose StatementBegin
-- Jenis kelamin dan kelas tiap peserta program pendek (peserta = pesanan
-- Produk CRM). kelas_id NULL = belum dapat kelas. cara: auto | manual
-- (manual = dipindah admin; tidak diubah otomatis).
CREATE TABLE IF NOT EXISTS program_peserta_kelas (
    program_id INTEGER NOT NULL REFERENCES lead_programs(id) ON DELETE CASCADE,
    order_id INTEGER NOT NULL REFERENCES product_orders(id) ON DELETE CASCADE,
    jenis_kelamin TEXT NOT NULL CHECK (jenis_kelamin IN ('L', 'P')),
    kelas_id INTEGER REFERENCES program_kelas(id) ON DELETE SET NULL,
    cara TEXT NOT NULL DEFAULT 'auto' CHECK (cara IN ('auto', 'manual')),
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (program_id, order_id)
);
CREATE INDEX IF NOT EXISTS idx_program_peserta_kelas_kelas ON program_peserta_kelas(kelas_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS program_peserta_kelas;
-- +goose StatementEnd
