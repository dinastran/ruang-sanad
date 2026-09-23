-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS produk (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nama TEXT NOT NULL,
    kategori TEXT NOT NULL CHECK (kategori IN ('buku', 'program')),
    track_stok INTEGER NOT NULL DEFAULT 0 CHECK (track_stok IN (0, 1)),
    is_aktif INTEGER NOT NULL DEFAULT 1 CHECK (is_aktif IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_produk_nama_unique ON produk(LOWER(TRIM(nama)));
CREATE INDEX idx_produk_aktif ON produk(is_aktif, kategori);

CREATE TABLE IF NOT EXISTS produk_batch (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    produk_id INTEGER NOT NULL REFERENCES produk(id) ON DELETE CASCADE,
    nama TEXT NOT NULL,
    tanggal_mulai DATE,
    tanggal_selesai DATE,
    is_aktif INTEGER NOT NULL DEFAULT 1 CHECK (is_aktif IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(produk_id, nama)
);

CREATE INDEX idx_produk_batch_produk ON produk_batch(produk_id, is_aktif);
CREATE UNIQUE INDEX idx_produk_batch_nama_unique ON produk_batch(produk_id, LOWER(TRIM(nama)));

CREATE TABLE IF NOT EXISTS mahasantri_produk (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    santri_id INTEGER NOT NULL REFERENCES santri(id) ON DELETE CASCADE,
    produk_id INTEGER NOT NULL REFERENCES produk(id) ON DELETE RESTRICT,
    produk_batch_id INTEGER REFERENCES produk_batch(id) ON DELETE RESTRICT,
    tanggal DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'aktif' CHECK (status IN ('aktif', 'dibatalkan')),
    catatan TEXT NOT NULL DEFAULT '',
    dicatat_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    dibatalkan_at DATETIME,
    dibatalkan_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_mahasantri_produk_active_no_batch
    ON mahasantri_produk(santri_id, produk_id)
    WHERE produk_batch_id IS NULL AND status = 'aktif';

CREATE UNIQUE INDEX idx_mahasantri_produk_active_with_batch
    ON mahasantri_produk(santri_id, produk_id, produk_batch_id)
    WHERE produk_batch_id IS NOT NULL AND status = 'aktif';

CREATE INDEX idx_mahasantri_produk_santri ON mahasantri_produk(santri_id, status);
CREATE INDEX idx_mahasantri_produk_produk ON mahasantri_produk(produk_id, produk_batch_id, status);
CREATE INDEX idx_mahasantri_produk_tanggal ON mahasantri_produk(tanggal);

CREATE TABLE IF NOT EXISTS stok_mutasi (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    produk_id INTEGER NOT NULL REFERENCES produk(id) ON DELETE RESTRICT,
    produk_batch_id INTEGER REFERENCES produk_batch(id) ON DELETE RESTRICT,
    tipe TEXT NOT NULL CHECK (tipe IN ('stok_awal', 'stok_masuk', 'pembelian', 'pembatalan', 'opname')),
    qty INTEGER NOT NULL CHECK (qty != 0),
    referensi_type TEXT NOT NULL DEFAULT '',
    referensi_id INTEGER,
    catatan TEXT NOT NULL DEFAULT '',
    dicatat_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_stok_mutasi_produk ON stok_mutasi(produk_id, produk_batch_id, created_at);
CREATE INDEX idx_stok_mutasi_ref ON stok_mutasi(referensi_type, referensi_id);

CREATE TABLE IF NOT EXISTS stok_opname (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    produk_id INTEGER NOT NULL REFERENCES produk(id) ON DELETE RESTRICT,
    produk_batch_id INTEGER REFERENCES produk_batch(id) ON DELETE RESTRICT,
    stok_sistem INTEGER NOT NULL,
    stok_fisik INTEGER NOT NULL CHECK (stok_fisik >= 0),
    selisih INTEGER NOT NULL,
    catatan TEXT NOT NULL DEFAULT '',
    dicatat_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_stok_opname_produk ON stok_opname(produk_id, produk_batch_id, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_stok_opname_produk;
DROP TABLE IF EXISTS stok_opname;
DROP INDEX IF EXISTS idx_stok_mutasi_ref;
DROP INDEX IF EXISTS idx_stok_mutasi_produk;
DROP TABLE IF EXISTS stok_mutasi;
DROP INDEX IF EXISTS idx_mahasantri_produk_tanggal;
DROP INDEX IF EXISTS idx_mahasantri_produk_produk;
DROP INDEX IF EXISTS idx_mahasantri_produk_santri;
DROP INDEX IF EXISTS idx_mahasantri_produk_active_with_batch;
DROP INDEX IF EXISTS idx_mahasantri_produk_active_no_batch;
DROP TABLE IF EXISTS mahasantri_produk;
DROP INDEX IF EXISTS idx_produk_batch_nama_unique;
DROP INDEX IF EXISTS idx_produk_batch_produk;
DROP TABLE IF EXISTS produk_batch;
DROP INDEX IF EXISTS idx_produk_aktif;
DROP INDEX IF EXISTS idx_produk_nama_unique;
DROP TABLE IF EXISTS produk;
-- +goose StatementEnd
