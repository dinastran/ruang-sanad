-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tasbih_pengaturan (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    target_harian INTEGER NOT NULL DEFAULT 100 CHECK (target_harian BETWEEN 1 AND 100000),
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tasbih_pengaturan;
-- +goose StatementEnd
