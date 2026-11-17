-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tasbih_badge_user (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_kode TEXT NOT NULL,
    diraih_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, badge_kode)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tasbih_badge_user;
-- +goose StatementEnd
