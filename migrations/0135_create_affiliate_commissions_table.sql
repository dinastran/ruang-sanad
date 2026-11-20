-- +goose Up
-- +goose StatementBegin
-- Buku komisi. sumber pendaftaran: sumber_id = lead_id; spp: sumber_id = tagihan.id.
CREATE TABLE IF NOT EXISTS affiliate_commissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    affiliate_id INTEGER NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    lead_id INTEGER NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    santri_id INTEGER REFERENCES santri(id) ON DELETE SET NULL,
    sumber TEXT NOT NULL CHECK (sumber IN ('pendaftaran', 'spp')),
    sumber_id INTEGER NOT NULL,
    bulan_ke INTEGER NOT NULL DEFAULT 1,
    basis INTEGER NOT NULL DEFAULT 0,
    jenis TEXT NOT NULL DEFAULT '',
    nilai INTEGER NOT NULL DEFAULT 0,
    pola TEXT NOT NULL DEFAULT '',
    maks_bulan INTEGER,
    jumlah INTEGER NOT NULL DEFAULT 0 CHECK (jumlah >= 0),
    status TEXT NOT NULL CHECK (status IN ('pending', 'siap', 'dibayar', 'batal')),
    siap_at DATETIME,
    dibayar_at DATETIME,
    dibayar_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    metode_bayar TEXT NOT NULL DEFAULT '',
    ujroh_bulan TEXT NOT NULL DEFAULT '',
    catatan TEXT NOT NULL DEFAULT '',
    alasan_batal TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    UNIQUE (sumber, sumber_id)
);
CREATE INDEX IF NOT EXISTS idx_affiliate_commissions_affiliate ON affiliate_commissions(affiliate_id, status);
CREATE INDEX IF NOT EXISTS idx_affiliate_commissions_status ON affiliate_commissions(status, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS affiliate_commissions;
-- +goose StatementEnd
