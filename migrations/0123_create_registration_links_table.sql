-- +goose Up
-- +goose StatementBegin
-- Link pendaftaran publik (/daftar/:slug). mode: wa = langsung ke WA CS,
-- short/full = form. after_submit: wa = lanjut chat CS, bayar = halaman bayar.
-- nominal_override NULL = harga varian/produk program.
CREATE TABLE IF NOT EXISTS registration_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    judul TEXT NOT NULL,
    deskripsi TEXT NOT NULL DEFAULT '',
    program_id INTEGER NOT NULL REFERENCES lead_programs(id) ON DELETE RESTRICT,
    lead_source_id INTEGER NOT NULL REFERENCES lead_sources(id) ON DELETE RESTRICT,
    mode TEXT NOT NULL CHECK (mode IN ('wa', 'short', 'full')),
    after_submit TEXT NOT NULL DEFAULT 'wa' CHECK (after_submit IN ('wa', 'bayar')),
    nominal_override INTEGER CHECK (nominal_override IS NULL OR nominal_override >= 0),
    field_config TEXT NOT NULL DEFAULT '{}',
    wa_template TEXT NOT NULL DEFAULT '',
    izinkan_manual INTEGER NOT NULL DEFAULT 1 CHECK (izinkan_manual IN (0, 1)),
    is_aktif INTEGER NOT NULL DEFAULT 1 CHECK (is_aktif IN (0, 1)),
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS registration_links;
-- +goose StatementEnd
