-- +goose Up
-- +goose StatementBegin
-- Tindak lanjut hasil kunjungan kelas. Jenis 'koordinasi' bersifat internal
-- (tidak ditampilkan ke guru). Jenis 'monitoring' menautkan kunjungan
-- berikutnya yang dibuat otomatis.
CREATE TABLE IF NOT EXISTS kunjungan_tindak_lanjut (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kunjungan_id INTEGER NOT NULL REFERENCES kunjungan_kelas(id) ON DELETE CASCADE,
    jenis TEXT NOT NULL CHECK (jenis IN ('apresiasi', 'monitoring', 'koordinasi', 'coaching', 'pembinaan')),
    catatan TEXT NOT NULL DEFAULT '',
    target_tanggal DATE,
    status TEXT NOT NULL DEFAULT 'terbuka' CHECK (status IN ('terbuka', 'selesai')),
    selesai_at DATETIME,
    kunjungan_berikutnya_id INTEGER REFERENCES kunjungan_kelas(id) ON DELETE SET NULL,
    dibuat_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_kunjungan_tl_kunjungan ON kunjungan_tindak_lanjut(kunjungan_id);
CREATE INDEX IF NOT EXISTS idx_kunjungan_tl_status ON kunjungan_tindak_lanjut(status, target_tanggal);
CREATE INDEX IF NOT EXISTS idx_kunjungan_tl_berikutnya ON kunjungan_tindak_lanjut(kunjungan_berikutnya_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS kunjungan_tindak_lanjut;
-- +goose StatementEnd
