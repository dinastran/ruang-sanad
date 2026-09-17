-- +goose Up
-- +goose StatementBegin

-- Kompetensi guru: satu baris per guru, 8 bidang, skala 0-3
-- (0=belum, 1=dasar, 2=mahir, 3=bersanad). Dipakai koordinator untuk
-- merekomendasikan penempatan guru sesuai level kelas.
CREATE TABLE IF NOT EXISTS guru_kompetensi (
    guru_id INTEGER PRIMARY KEY REFERENCES guru(id) ON DELETE CASCADE,
    hafalan_quran INTEGER NOT NULL DEFAULT 0,
    hafalan_tuhfah INTEGER NOT NULL DEFAULT 0,
    hafalan_jazariy INTEGER NOT NULL DEFAULT 0,
    hafalan_khaqaniy INTEGER NOT NULL DEFAULT 0,
    hafalan_syakhawiy INTEGER NOT NULL DEFAULT 0,
    sanad_qiroah INTEGER NOT NULL DEFAULT 0,
    bahasa_arab_pasif INTEGER NOT NULL DEFAULT 0,
    bahasa_arab_aktif INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Pembinaan (sesi pembinaan mingguan)
CREATE TABLE IF NOT EXISTS pembinaan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tanggal DATE NOT NULL,
    bulan TEXT NOT NULL DEFAULT '',
    pekan_ke INTEGER NOT NULL DEFAULT 0,
    topik TEXT NOT NULL DEFAULT '',
    keterangan TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'dijadwalkan', -- dijadwalkan/terlaksana/libur
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_pembinaan_tanggal ON pembinaan(tanggal);

CREATE TABLE IF NOT EXISTS pembinaan_absen (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pembinaan_id INTEGER NOT NULL REFERENCES pembinaan(id) ON DELETE CASCADE,
    guru_id INTEGER NOT NULL REFERENCES guru(id) ON DELETE CASCADE,
    hadir INTEGER NOT NULL DEFAULT 0,
    alasan TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pembinaan_id, guru_id)
);
CREATE INDEX IF NOT EXISTS idx_pembinaan_absen_guru ON pembinaan_absen(guru_id);

-- Rapat guru (bulanan)
CREATE TABLE IF NOT EXISTS rapat_guru (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tanggal DATE NOT NULL,
    judul TEXT NOT NULL DEFAULT '',
    catatan TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'dijadwalkan', -- dijadwalkan/terlaksana/batal
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_rapat_tanggal ON rapat_guru(tanggal);

CREATE TABLE IF NOT EXISTS rapat_absen (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rapat_id INTEGER NOT NULL REFERENCES rapat_guru(id) ON DELETE CASCADE,
    guru_id INTEGER NOT NULL REFERENCES guru(id) ON DELETE CASCADE,
    hadir INTEGER NOT NULL DEFAULT 0,
    alasan TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(rapat_id, guru_id)
);
CREATE INDEX IF NOT EXISTS idx_rapat_absen_guru ON rapat_absen(guru_id);

-- Tilawah harian (self check-in guru)
CREATE TABLE IF NOT EXISTS tilawah_harian (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guru_id INTEGER NOT NULL REFERENCES guru(id) ON DELETE CASCADE,
    tanggal DATE NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(guru_id, tanggal)
);
CREATE INDEX IF NOT EXISTS idx_tilawah_guru ON tilawah_harian(guru_id);

-- Kunjungan kelas
CREATE TABLE IF NOT EXISTS kunjungan_kelas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guru_id INTEGER NOT NULL REFERENCES guru(id) ON DELETE CASCADE,
    kelas_id INTEGER REFERENCES kelas(id) ON DELETE SET NULL,
    target_mulai DATE,
    target_selesai DATE,
    tanggal DATE,
    jam TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'dijadwalkan', -- dijadwalkan/terlaksana/ditunda/batal
    catatan TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_kunjungan_guru ON kunjungan_kelas(guru_id);

-- Kalam Bersanad (kajian pekanan)
CREATE TABLE IF NOT EXISTS kalam_bersanad (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tanggal DATE NOT NULL,
    topik TEXT NOT NULL DEFAULT '',
    kitab TEXT NOT NULL DEFAULT '',
    keterangan TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_kalam_tanggal ON kalam_bersanad(tanggal);

CREATE TABLE IF NOT EXISTS kalam_share_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kalam_id INTEGER NOT NULL REFERENCES kalam_bersanad(id) ON DELETE CASCADE,
    guru_id INTEGER NOT NULL REFERENCES guru(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(kalam_id, guru_id)
);
CREATE INDEX IF NOT EXISTS idx_kalam_share_guru ON kalam_share_log(guru_id);

-- Template pesan WA
CREATE TABLE IF NOT EXISTS wa_template (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nama TEXT NOT NULL,
    target_type TEXT NOT NULL DEFAULT 'santri', -- santri/guru
    body TEXT NOT NULL DEFAULT '',
    is_aktif INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS wa_template;
DROP TABLE IF EXISTS kalam_share_log;
DROP TABLE IF EXISTS kalam_bersanad;
DROP TABLE IF EXISTS kunjungan_kelas;
DROP TABLE IF EXISTS tilawah_harian;
DROP TABLE IF EXISTS rapat_absen;
DROP TABLE IF EXISTS rapat_guru;
DROP TABLE IF EXISTS pembinaan_absen;
DROP TABLE IF EXISTS pembinaan;
DROP TABLE IF EXISTS guru_kompetensi;
-- +goose StatementEnd
