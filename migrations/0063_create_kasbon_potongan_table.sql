-- +goose Up
-- +goose StatementBegin
-- Kasbon deductions frozen when a month's ujroh is locked; removed again when
-- the month is unlocked. Sisa kasbon = nominal - SUM(potongan).
CREATE TABLE IF NOT EXISTS kasbon_potongan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bulan TEXT NOT NULL REFERENCES ujroh_bulan(bulan) ON DELETE CASCADE,
    kasbon_id INTEGER NOT NULL REFERENCES kasbon(id) ON DELETE CASCADE,
    guru_id INTEGER NOT NULL,
    nominal INTEGER NOT NULL CHECK (nominal >= 0),
    UNIQUE (bulan, kasbon_id)
);

CREATE INDEX IF NOT EXISTS idx_kasbon_potongan_kasbon ON kasbon_potongan(kasbon_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_kasbon_potongan_kasbon;
DROP TABLE IF EXISTS kasbon_potongan;
-- +goose StatementEnd
