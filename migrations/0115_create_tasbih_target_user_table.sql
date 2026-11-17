-- +goose Up
-- +goose StatementBegin
-- Override target per dzikir per akun (berlaku di luar paket).
CREATE TABLE IF NOT EXISTS tasbih_target_user (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    dzikir_id INTEGER NOT NULL REFERENCES tasbih_dzikir(id) ON DELETE CASCADE,
    target INTEGER NOT NULL CHECK (target BETWEEN 1 AND 100000),
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, dzikir_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tasbih_target_user;
-- +goose StatementEnd
