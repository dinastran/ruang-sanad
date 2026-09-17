-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;
BEGIN IMMEDIATE;

CREATE TEMP TABLE kelas_sequence_backup AS
SELECT seq FROM sqlite_sequence WHERE name = 'kelas';

CREATE TABLE kelas_new (
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
    kapasitas INTEGER NOT NULL DEFAULT 15,
    jumlah_santri INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_aktif INTEGER NOT NULL DEFAULT 1,
    pertemuan_terakhir INTEGER NOT NULL DEFAULT 0,
    UNIQUE (kunci_kelas, sub_index),
    FOREIGN KEY (guru_id) REFERENCES guru(id) ON DELETE SET NULL
);

INSERT INTO kelas_new (
    id, kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal,
    sub_index, nama_kelas, guru_id, kapasitas, jumlah_santri, created_at,
    is_aktif, pertemuan_terakhir
)
SELECT
    id, kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal,
    sub_index, nama_kelas, guru_id,
    CASE WHEN kapasitas = 10 THEN 15 ELSE kapasitas END,
    jumlah_santri, created_at, is_aktif, pertemuan_terakhir
FROM kelas;

DROP TABLE kelas;
ALTER TABLE kelas_new RENAME TO kelas;
CREATE INDEX idx_kelas_angkatan ON kelas(angkatan);
CREATE INDEX idx_kelas_kunci ON kelas(kunci_kelas);

UPDATE sqlite_sequence
SET seq = MAX(seq, (SELECT seq FROM kelas_sequence_backup))
WHERE name = 'kelas';
INSERT INTO sqlite_sequence (name, seq)
SELECT 'kelas', seq FROM kelas_sequence_backup
WHERE NOT EXISTS (SELECT 1 FROM sqlite_sequence WHERE name = 'kelas');
DROP TABLE kelas_sequence_backup;

COMMIT;
PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;
BEGIN IMMEDIATE;

CREATE TEMP TABLE kelas_sequence_backup AS
SELECT seq FROM sqlite_sequence WHERE name = 'kelas';

CREATE TABLE kelas_old (
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
    is_aktif INTEGER NOT NULL DEFAULT 1,
    pertemuan_terakhir INTEGER NOT NULL DEFAULT 0,
    UNIQUE (kunci_kelas, sub_index),
    FOREIGN KEY (guru_id) REFERENCES guru(id) ON DELETE SET NULL
);

INSERT INTO kelas_old (
    id, kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal,
    sub_index, nama_kelas, guru_id, kapasitas, jumlah_santri, created_at,
    is_aktif, pertemuan_terakhir
)
SELECT
    id, kunci_kelas, angkatan, tipe, jenis_kelamin, level, frekuensi, jadwal,
    sub_index, nama_kelas, guru_id, kapasitas, jumlah_santri, created_at,
    is_aktif, pertemuan_terakhir
FROM kelas;

DROP TABLE kelas;
ALTER TABLE kelas_old RENAME TO kelas;
CREATE INDEX idx_kelas_angkatan ON kelas(angkatan);
CREATE INDEX idx_kelas_kunci ON kelas(kunci_kelas);

UPDATE sqlite_sequence
SET seq = MAX(seq, (SELECT seq FROM kelas_sequence_backup))
WHERE name = 'kelas';
INSERT INTO sqlite_sequence (name, seq)
SELECT 'kelas', seq FROM kelas_sequence_backup
WHERE NOT EXISTS (SELECT 1 FROM sqlite_sequence WHERE name = 'kelas');
DROP TABLE kelas_sequence_backup;

COMMIT;
PRAGMA foreign_keys = ON;
