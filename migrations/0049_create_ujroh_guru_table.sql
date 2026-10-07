-- +goose Up
-- +goose StatementBegin
-- Locked per-guru ujroh totals for a month, plus payment status. Name, status
-- and rate are snapshots so later master-data changes never alter paid months.
CREATE TABLE IF NOT EXISTS ujroh_guru (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bulan TEXT NOT NULL REFERENCES ujroh_bulan(bulan) ON DELETE CASCADE,
    guru_id INTEGER NOT NULL,
    guru_nama TEXT NOT NULL DEFAULT '',
    guru_status TEXT NOT NULL DEFAULT '',
    tarif INTEGER NOT NULL DEFAULT 0,
    jumlah_pertemuan INTEGER NOT NULL DEFAULT 0,
    jumlah_badal INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    dibayar_at DATETIME,
    dibayar_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    catatan_bayar TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (bulan, guru_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ujroh_guru;
-- +goose StatementEnd
