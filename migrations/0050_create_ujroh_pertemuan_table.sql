-- +goose Up
-- +goose StatementBegin
-- Locked meeting lines behind each ujroh_guru total, so a locked month can
-- still explain "which meetings was I paid for".
CREATE TABLE IF NOT EXISTS ujroh_pertemuan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bulan TEXT NOT NULL REFERENCES ujroh_bulan(bulan) ON DELETE CASCADE,
    guru_id INTEGER NOT NULL,
    pertemuan_id INTEGER NOT NULL,
    kelas_id INTEGER NOT NULL DEFAULT 0,
    kelas_nama TEXT NOT NULL DEFAULT '',
    tanggal TEXT NOT NULL DEFAULT '',
    pertemuan_ke INTEGER NOT NULL DEFAULT 0,
    is_badal INTEGER NOT NULL DEFAULT 0,
    tarif INTEGER NOT NULL DEFAULT 0,
    UNIQUE (bulan, pertemuan_id)
);

CREATE INDEX IF NOT EXISTS idx_ujroh_pertemuan_bulan_guru ON ujroh_pertemuan(bulan, guru_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_ujroh_pertemuan_bulan_guru;
DROP TABLE IF EXISTS ujroh_pertemuan;
-- +goose StatementEnd
