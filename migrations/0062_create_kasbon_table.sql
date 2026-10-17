-- +goose Up
-- +goose StatementBegin
-- Cash advances to guru, repaid by deducting from monthly ujroh starting at
-- mulai_bulan: cicilan per month, or everything at once when cicilan = 0.
CREATE TABLE IF NOT EXISTS kasbon (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guru_id INTEGER NOT NULL REFERENCES guru(id) ON DELETE RESTRICT,
    tanggal TEXT NOT NULL CHECK (length(tanggal) = 10),
    nominal INTEGER NOT NULL CHECK (nominal > 0),
    cicilan INTEGER NOT NULL DEFAULT 0 CHECK (cicilan >= 0),
    mulai_bulan TEXT NOT NULL CHECK (length(mulai_bulan) = 7),
    keterangan TEXT NOT NULL DEFAULT '',
    dibuat_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    dibatalkan_at DATETIME,
    dibatalkan_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_kasbon_guru ON kasbon(guru_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_kasbon_guru;
DROP TABLE IF EXISTS kasbon;
-- +goose StatementEnd
