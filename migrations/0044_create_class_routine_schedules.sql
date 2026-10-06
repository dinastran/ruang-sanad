-- +goose Up
-- +goose StatementBegin
CREATE TABLE kelas_jadwal_rutin (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kelas_id INTEGER NOT NULL REFERENCES kelas(id) ON DELETE CASCADE,
    hari INTEGER NOT NULL CHECK (hari BETWEEN 1 AND 7),
    jam_mulai TEXT NOT NULL,
    berlaku_mulai DATE NOT NULL,
    berlaku_sampai DATE,
    is_aktif INTEGER NOT NULL DEFAULT 1,
    dibuat_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_kelas_jadwal_rutin_kelas_aktif
ON kelas_jadwal_rutin(kelas_id, is_aktif, hari, jam_mulai);

CREATE UNIQUE INDEX uq_kelas_jadwal_rutin_versi
ON kelas_jadwal_rutin(kelas_id, hari, jam_mulai, berlaku_mulai);

ALTER TABLE jadwal_pertemuan ADD COLUMN jadwal_rutin_id INTEGER REFERENCES kelas_jadwal_rutin(id) ON DELETE SET NULL;
ALTER TABLE jadwal_pertemuan ADD COLUMN tanggal_rutin DATE;
ALTER TABLE jadwal_pertemuan ADD COLUMN is_otomatis INTEGER NOT NULL DEFAULT 0;
ALTER TABLE jadwal_pertemuan ADD COLUMN is_tambahan INTEGER NOT NULL DEFAULT 0;

CREATE UNIQUE INDEX uq_jadwal_pertemuan_rutin_occurrence
ON jadwal_pertemuan(jadwal_rutin_id, tanggal_rutin)
WHERE jadwal_rutin_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS uq_jadwal_pertemuan_rutin_occurrence;
ALTER TABLE jadwal_pertemuan DROP COLUMN is_tambahan;
ALTER TABLE jadwal_pertemuan DROP COLUMN is_otomatis;
ALTER TABLE jadwal_pertemuan DROP COLUMN tanggal_rutin;
ALTER TABLE jadwal_pertemuan DROP COLUMN jadwal_rutin_id;
DROP INDEX IF EXISTS uq_kelas_jadwal_rutin_versi;
DROP INDEX IF EXISTS idx_kelas_jadwal_rutin_kelas_aktif;
DROP TABLE kelas_jadwal_rutin;
-- +goose StatementEnd
