-- +goose Up
-- +goose StatementBegin
-- Pembayaran lead: manual (bukti transfer) atau gateway (fase 1b).
CREATE TABLE IF NOT EXISTS payments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lead_id INTEGER NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    link_id INTEGER REFERENCES registration_links(id) ON DELETE SET NULL,
    produk_varian_id INTEGER REFERENCES produk_varian(id) ON DELETE SET NULL,
    metode TEXT NOT NULL CHECK (metode IN ('gateway', 'manual')),
    provider TEXT NOT NULL DEFAULT '',
    external_id TEXT UNIQUE,
    provider_ref TEXT NOT NULL DEFAULT '',
    checkout_url TEXT NOT NULL DEFAULT '',
    amount INTEGER NOT NULL CHECK (amount >= 0),
    status TEXT NOT NULL CHECK (status IN ('pending', 'paid', 'expired', 'failed', 'menunggu_verifikasi', 'ditolak')),
    paid_at DATETIME,
    expires_at DATETIME,
    channel TEXT NOT NULL DEFAULT '',
    bukti_path TEXT NOT NULL DEFAULT '',
    raw_webhook TEXT NOT NULL DEFAULT '',
    jadwal_kode TEXT NOT NULL DEFAULT '',
    jenis_kelamin TEXT NOT NULL DEFAULT '',
    perlu_tinjau INTEGER NOT NULL DEFAULT 0 CHECK (perlu_tinjau IN (0, 1)),
    catatan_sistem TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_payments_lead ON payments(lead_id, created_at);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status, expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS payments;
-- +goose StatementEnd
