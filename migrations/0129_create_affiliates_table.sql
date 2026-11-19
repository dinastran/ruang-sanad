-- +goose Up
-- +goose StatementBegin
-- Affiliate: guru (otomatis) atau eksternal (daftar sendiri, disetujui admin).
CREATE TABLE IF NOT EXISTS affiliates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kode TEXT NOT NULL UNIQUE,
    tipe TEXT NOT NULL CHECK (tipe IN ('guru', 'eksternal')),
    guru_id INTEGER UNIQUE REFERENCES guru(id) ON DELETE SET NULL,
    user_id INTEGER UNIQUE REFERENCES users(id) ON DELETE SET NULL,
    nama TEXT NOT NULL,
    no_wa TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'menunggu' CHECK (status IN ('menunggu', 'aktif', 'ditolak', 'nonaktif')),
    bank_nama TEXT NOT NULL DEFAULT '',
    bank_rekening TEXT NOT NULL DEFAULT '',
    bank_atas_nama TEXT NOT NULL DEFAULT '',
    catatan_admin TEXT NOT NULL DEFAULT '',
    disetujui_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    disetujui_at DATETIME,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_affiliates_status ON affiliates(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS affiliates;
-- +goose StatementEnd
