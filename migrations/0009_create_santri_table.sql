-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS santri (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    id_mahasantri TEXT NOT NULL DEFAULT '',
    -- input CS
    kelas_kode TEXT NOT NULL DEFAULT '',
    nama TEXT NOT NULL,
    jenis_kelamin TEXT NOT NULL DEFAULT '',
    nominal INTEGER NOT NULL DEFAULT 0,
    tanggal_daftar DATE,
    angkatan TEXT NOT NULL DEFAULT '',
    usia INTEGER,
    domisili TEXT NOT NULL DEFAULT '',
    -- input Admin Kelas
    fu TEXT NOT NULL DEFAULT '',
    tanggal_vn DATE,
    hasil_vn TEXT NOT NULL DEFAULT '',
    masuk_grup TEXT NOT NULL DEFAULT '',
    mulai_belajar DATE,
    jumlah INTEGER,
    level TEXT NOT NULL DEFAULT '',
    jadwal TEXT NOT NULL DEFAULT '',
    guru TEXT NOT NULL DEFAULT '',
    jadwal_catatan TEXT NOT NULL DEFAULT '',
    -- input Keuangan
    infaq_terakhir TEXT NOT NULL DEFAULT '',
    keterangan_tidak_lanjut TEXT NOT NULL DEFAULT '',
    -- engine (auto)
    tipe TEXT NOT NULL DEFAULT '',
    frekuensi TEXT NOT NULL DEFAULT '',
    is_lengkap INTEGER NOT NULL DEFAULT 0,
    kelas_id INTEGER,
    status TEXT NOT NULL DEFAULT 'aktif',
    created_by INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (kelas_id) REFERENCES kelas(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_santri_angkatan ON santri(angkatan);
CREATE INDEX idx_santri_kelas ON santri(kelas_id);
CREATE INDEX idx_santri_lengkap ON santri(is_lengkap);
CREATE INDEX idx_santri_gender ON santri(jenis_kelamin);
CREATE INDEX idx_santri_status ON santri(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_santri_status;
DROP INDEX IF EXISTS idx_santri_gender;
DROP INDEX IF EXISTS idx_santri_lengkap;
DROP INDEX IF EXISTS idx_santri_kelas;
DROP INDEX IF EXISTS idx_santri_angkatan;
DROP TABLE IF EXISTS santri;
-- +goose StatementEnd
