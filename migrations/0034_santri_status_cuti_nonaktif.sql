-- +goose Up
-- +goose StatementBegin
-- Status mahasantri oleh Admin Kelas: aktif, cuti, nonaktif. tidak_lanjut tetap
-- diatur Keuangan. Cuti memegang kursi kelas; nonaktif & tidak_lanjut melepasnya.
-- Tanggal cuti disimpan sebagai TEXT 'YYYY-MM-DD' agar perbandingan tanggal di
-- job auto-aktif tidak bergantung format waktu driver. cuti_selesai adalah hari
-- terakhir cuti; santri aktif kembali keesokan harinya.
ALTER TABLE santri ADD COLUMN status_alasan TEXT NOT NULL DEFAULT '';
ALTER TABLE santri ADD COLUMN cuti_mulai TEXT NOT NULL DEFAULT '';
ALTER TABLE santri ADD COLUMN cuti_selesai TEXT NOT NULL DEFAULT '';

-- dibuat_oleh NULL berarti perubahan oleh sistem (auto-aktif setelah cuti).
CREATE TABLE santri_status_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    santri_id INTEGER NOT NULL REFERENCES santri(id) ON DELETE CASCADE,
    kelas_id INTEGER REFERENCES kelas(id) ON DELETE SET NULL,
    status_lama TEXT NOT NULL,
    status_baru TEXT NOT NULL,
    alasan TEXT NOT NULL DEFAULT '',
    cuti_mulai TEXT NOT NULL DEFAULT '',
    cuti_selesai TEXT NOT NULL DEFAULT '',
    dibuat_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_santri_status_log_santri ON santri_status_log(santri_id, created_at DESC);
CREATE INDEX idx_santri_status_log_kelas ON santri_status_log(kelas_id, created_at DESC);
CREATE INDEX idx_santri_status_cuti ON santri(status, cuti_selesai);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_santri_status_cuti;
DROP INDEX IF EXISTS idx_santri_status_log_kelas;
DROP INDEX IF EXISTS idx_santri_status_log_santri;
DROP TABLE IF EXISTS santri_status_log;
ALTER TABLE santri DROP COLUMN cuti_selesai;
ALTER TABLE santri DROP COLUMN cuti_mulai;
ALTER TABLE santri DROP COLUMN status_alasan;
-- +goose StatementEnd
