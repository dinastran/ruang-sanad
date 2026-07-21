-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS kode_kelas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kode TEXT NOT NULL UNIQUE,
    tipe TEXT NOT NULL DEFAULT '',
    frekuensi TEXT NOT NULL DEFAULT '',
    urutan INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_kode_kelas_kode ON kode_kelas(kode);

-- Seed the standard kode kelas. Tipe: P=Private, R=Reguler, SP=Semi Private.
INSERT INTO kode_kelas (kode, tipe, frekuensi, urutan) VALUES
    ('R 1X', 'Reguler', '1x/pekan', 1),
    ('R 2X', 'Reguler', '2x/pekan', 2),
    ('P 1X', 'Private', '1x/pekan', 3),
    ('P 2X', 'Private', '2x/pekan', 4),
    ('P 4X', 'Private', '4x pertemuan', 5),
    ('P 16X', 'Private', '16x pertemuan', 6),
    ('SP 1X', 'Semi Private', '1x/pekan', 7);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_kode_kelas_kode;
DROP TABLE IF EXISTS kode_kelas;
-- +goose StatementEnd
