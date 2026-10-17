-- +goose Up
-- +goose StatementBegin
-- Locked component lines behind each ujroh_guru row. Name and jenis are
-- snapshots so renaming a component never alters a locked month.
CREATE TABLE IF NOT EXISTS ujroh_guru_komponen (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bulan TEXT NOT NULL REFERENCES ujroh_bulan(bulan) ON DELETE CASCADE,
    guru_id INTEGER NOT NULL,
    komponen_id INTEGER NOT NULL,
    komponen_nama TEXT NOT NULL DEFAULT '',
    jenis TEXT NOT NULL DEFAULT '',
    jumlah INTEGER NOT NULL DEFAULT 1,
    tarif INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    catatan TEXT NOT NULL DEFAULT '',
    UNIQUE (bulan, guru_id, komponen_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ujroh_guru_komponen;
-- +goose StatementEnd
