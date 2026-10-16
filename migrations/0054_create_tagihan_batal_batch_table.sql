-- +goose Up
-- +goose StatementBegin
-- Satu baris per aksi "Batalkan massal" di halaman Tagihan. Filter yang dipakai
-- disimpan sebagai JSON agar riwayat bisa menampilkan cakupan pembatalan.
-- dipulihkan_at terisi saat batch di-undo; batch hanya bisa dipulihkan sekali.
CREATE TABLE IF NOT EXISTS tagihan_batal_batch (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alasan TEXT NOT NULL,
    filter_json TEXT NOT NULL DEFAULT '{}',
    jumlah INTEGER NOT NULL DEFAULT 0,
    total_nominal INTEGER NOT NULL DEFAULT 0,
    dibuat_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    dipulihkan_at DATETIME,
    dipulihkan_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    jumlah_dipulihkan INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_tagihan_batal_batch_created ON tagihan_batal_batch(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tagihan_batal_batch;
-- +goose StatementEnd
