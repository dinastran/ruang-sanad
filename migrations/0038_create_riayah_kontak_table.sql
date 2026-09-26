-- +goose Up
-- +goose StatementBegin
-- Log kontak guru ke santri (menyapa, menanyakan kabar, kirim rapor).
-- tanggal disimpan sebagai TEXT 'YYYY-MM-DD' agar mudah dibandingkan.
CREATE TABLE riayah_kontak (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    santri_id INTEGER NOT NULL REFERENCES santri(id) ON DELETE CASCADE,
    guru_id INTEGER REFERENCES guru(id) ON DELETE SET NULL,
    author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    tanggal TEXT NOT NULL,
    media TEXT NOT NULL DEFAULT 'wa',   -- wa / telepon / tatap_muka / lainnya
    jenis TEXT NOT NULL DEFAULT 'sapa', -- sapa / rapor
    periode TEXT NOT NULL DEFAULT '',   -- 'YYYY-MM' untuk rapor bulanan
    catatan TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_riayah_kontak_santri ON riayah_kontak(santri_id, tanggal DESC);
CREATE INDEX idx_riayah_kontak_rapor ON riayah_kontak(jenis, periode);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE riayah_kontak;
-- +goose StatementEnd
