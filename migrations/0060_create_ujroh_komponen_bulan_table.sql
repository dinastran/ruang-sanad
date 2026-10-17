-- +goose Up
-- +goose StatementBegin
-- Per-month component input for a guru: an insidental entry (jumlah x tarif)
-- or an adjustment of a tetap component for that month only (jumlah 0 = not
-- given this month). Editable only while the month is unlocked.
CREATE TABLE IF NOT EXISTS ujroh_komponen_bulan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bulan TEXT NOT NULL CHECK (length(bulan) = 7),
    guru_id INTEGER NOT NULL REFERENCES guru(id) ON DELETE CASCADE,
    komponen_id INTEGER NOT NULL REFERENCES ujroh_komponen(id) ON DELETE CASCADE,
    jumlah INTEGER NOT NULL DEFAULT 1 CHECK (jumlah >= 0),
    tarif INTEGER NOT NULL DEFAULT 0 CHECK (tarif >= 0),
    catatan TEXT NOT NULL DEFAULT '',
    diubah_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (bulan, guru_id, komponen_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ujroh_komponen_bulan;
-- +goose StatementEnd
