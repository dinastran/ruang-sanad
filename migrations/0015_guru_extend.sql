-- +goose Up
-- +goose StatementBegin

-- Extend guru table with user relationship and additional fields
ALTER TABLE guru ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE guru ADD COLUMN status TEXT NOT NULL DEFAULT 'tetap';
ALTER TABLE guru ADD COLUMN no_wa TEXT NOT NULL DEFAULT '';
ALTER TABLE guru ADD COLUMN email TEXT NOT NULL DEFAULT '';
ALTER TABLE guru ADD COLUMN tanggal_gabung DATE;
ALTER TABLE guru ADD COLUMN foto TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX idx_guru_user_id ON guru(user_id);

-- Pertemuan (class session)
CREATE TABLE IF NOT EXISTS pertemuan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kelas_id INTEGER NOT NULL REFERENCES kelas(id) ON DELETE CASCADE,
    pertemuan_ke INTEGER NOT NULL,
    tanggal DATE NOT NULL,
    jam_mulai TEXT NOT NULL DEFAULT '',
    jam_selesai TEXT NOT NULL DEFAULT '',
    materi TEXT NOT NULL DEFAULT '',
    catatan TEXT NOT NULL DEFAULT '',
    is_reschedule INTEGER NOT NULL DEFAULT 0,
    jadwal_semula TEXT NOT NULL DEFAULT '',
    alasan_reschedule TEXT NOT NULL DEFAULT '',
    is_badal INTEGER NOT NULL DEFAULT 0,
    guru_pengganti_id INTEGER REFERENCES guru(id) ON DELETE SET NULL,
    alasan_badal TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'selesai',
    dibuat_oleh INTEGER REFERENCES users(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(kelas_id, pertemuan_ke)
);

CREATE INDEX idx_pertemuan_kelas ON pertemuan(kelas_id);
CREATE INDEX idx_pertemuan_tanggal ON pertemuan(tanggal);

-- Absensi (attendance per student per session)
CREATE TABLE IF NOT EXISTS absensi (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pertemuan_id INTEGER NOT NULL REFERENCES pertemuan(id) ON DELETE CASCADE,
    santri_id INTEGER NOT NULL REFERENCES santri(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'hadir',
    catatan TEXT NOT NULL DEFAULT '',
    dibuat_oleh INTEGER REFERENCES users(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pertemuan_id, santri_id)
);

CREATE INDEX idx_absensi_pertemuan ON absensi(pertemuan_id);
CREATE INDEX idx_absensi_santri ON absensi(santri_id);

-- Catatan Riayah (personal notes on students)
CREATE TABLE IF NOT EXISTS catatan_riayah (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guru_id INTEGER NOT NULL REFERENCES guru(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL DEFAULT 'santri',
    target_id INTEGER NOT NULL,
    catatan TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_riayah_guru ON catatan_riayah(guru_id);
CREATE INDEX idx_riayah_target ON catatan_riayah(target_type, target_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS catatan_riayah;
DROP TABLE IF EXISTS absensi;
DROP TABLE IF EXISTS pertemuan;

ALTER TABLE guru DROP COLUMN foto;
ALTER TABLE guru DROP COLUMN tanggal_gabung;
ALTER TABLE guru DROP COLUMN email;
ALTER TABLE guru DROP COLUMN no_wa;
ALTER TABLE guru DROP COLUMN status;
ALTER TABLE guru DROP COLUMN user_id;

DROP INDEX IF EXISTS idx_guru_user_id;
DROP INDEX IF EXISTS idx_riayah_target;
DROP INDEX IF EXISTS idx_riayah_guru;
DROP INDEX IF EXISTS idx_absensi_santri;
DROP INDEX IF EXISTS idx_absensi_pertemuan;
DROP INDEX IF EXISTS idx_pertemuan_tanggal;
DROP INDEX IF EXISTS idx_pertemuan_kelas;
-- +goose StatementEnd
