-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS kelas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kunci_kelas TEXT NOT NULL,
    angkatan TEXT NOT NULL,
    tipe TEXT NOT NULL,
    jenis_kelamin TEXT NOT NULL,
    level TEXT NOT NULL,
    frekuensi TEXT NOT NULL,
    jadwal TEXT NOT NULL DEFAULT '',
    sub_index INTEGER NOT NULL DEFAULT 1,
    nama_kelas TEXT NOT NULL,
    guru_id INTEGER,
    kapasitas INTEGER NOT NULL DEFAULT 10,
    jumlah_santri INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (kunci_kelas, sub_index),
    FOREIGN KEY (guru_id) REFERENCES guru(id) ON DELETE SET NULL
);

CREATE INDEX idx_kelas_angkatan ON kelas(angkatan);
CREATE INDEX idx_kelas_kunci ON kelas(kunci_kelas);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_kelas_kunci;
DROP INDEX IF EXISTS idx_kelas_angkatan;
DROP TABLE IF EXISTS kelas;
-- +goose StatementEnd
