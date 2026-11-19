-- +goose Up
-- +goose StatementBegin
-- Aturan komisi. scope: global | program | affiliate (yang paling spesifik menang).
-- nilai: persen dalam basis poin (1000 = 10%), flat dalam rupiah.
CREATE TABLE IF NOT EXISTS affiliate_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scope TEXT NOT NULL CHECK (scope IN ('global', 'program', 'affiliate')),
    target_id INTEGER NOT NULL DEFAULT 0,
    jenis TEXT NOT NULL CHECK (jenis IN ('persen', 'flat')),
    nilai INTEGER NOT NULL CHECK (nilai >= 0),
    pola TEXT NOT NULL CHECK (pola IN ('sekali', 'bulanan')),
    maks_bulan INTEGER CHECK (maks_bulan IS NULL OR maks_bulan >= 1),
    is_aktif INTEGER NOT NULL DEFAULT 1 CHECK (is_aktif IN (0, 1)),
    updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    updated_at DATETIME NOT NULL,
    UNIQUE (scope, target_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS affiliate_rules;
-- +goose StatementEnd
