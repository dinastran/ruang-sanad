-- +goose Up
-- +goose StatementBegin

-- Todo koordinator (menggantikan Sheet 1 "Todo Koordinator Guru").
CREATE TABLE IF NOT EXISTS todo_koordinator (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    judul TEXT NOT NULL,
    teknis TEXT NOT NULL DEFAULT '',
    kebutuhan TEXT NOT NULL DEFAULT '',
    deadline DATE,
    pic TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'belum',   -- belum/proses/selesai/batal
    recurring TEXT NOT NULL DEFAULT 'none', -- none/harian/mingguan/bulanan/4bulanan
    link_pendukung TEXT NOT NULL DEFAULT '',
    catatan TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_todo_status ON todo_koordinator(status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS todo_koordinator;
-- +goose StatementEnd
