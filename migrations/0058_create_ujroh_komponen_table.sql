-- +goose Up
-- +goose StatementBegin
-- Master list of extra ujroh components (kuota, tunjangan kesehatan, MC,
-- Qiro'ah, ...). "tetap" components are paid every month to the guru assigned
-- in ujroh_komponen_guru; "insidental" ones are entered per month as
-- jumlah x tarif. Components are deactivated, never deleted, to keep history.
CREATE TABLE IF NOT EXISTS ujroh_komponen (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nama TEXT NOT NULL COLLATE NOCASE UNIQUE,
    jenis TEXT NOT NULL CHECK (jenis IN ('tetap', 'insidental')),
    nominal_default INTEGER NOT NULL DEFAULT 0 CHECK (nominal_default >= 0),
    is_aktif INTEGER NOT NULL DEFAULT 1,
    diubah_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ujroh_komponen;
-- +goose StatementEnd
