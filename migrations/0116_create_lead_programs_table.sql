-- +goose Up
-- +goose StatementBegin
-- Program yang dipasarkan lewat CRM WhatsApp. jenis 'reguler' = closing
-- menjadi santri (kelas, SPP, onboarding); jenis 'program' = program pendek
-- (mis. Sepekan Bersama Al-Fatihah): closing menjadi peserta, dicatat sebagai
-- pesanan produk program di Produk CRM, tanpa kelas/SPP.
CREATE TABLE IF NOT EXISTS lead_programs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nama TEXT NOT NULL UNIQUE COLLATE NOCASE,
    jenis TEXT NOT NULL DEFAULT 'program' CHECK (jenis IN ('reguler', 'program')),
    produk_id INTEGER REFERENCES produk(id) ON DELETE SET NULL,
    produk_batch_id INTEGER REFERENCES produk_batch(id) ON DELETE SET NULL,
    keterangan TEXT NOT NULL DEFAULT '',
    is_aktif INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- id 1 = program bawaan: lead tanpa program masuk ke sini.
INSERT OR IGNORE INTO lead_programs (id, nama, jenis, keterangan) VALUES (1, 'Tahsin Reguler', 'reguler', 'Program bawaan: closing menjadi santri kelas reguler.');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS lead_programs;
-- +goose StatementEnd
