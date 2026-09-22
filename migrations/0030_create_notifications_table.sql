-- +goose Up
-- +goose StatementBegin
CREATE TABLE notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    action_url TEXT NOT NULL DEFAULT '',
    reference_type TEXT NOT NULL DEFAULT '',
    reference_id INTEGER,
    read_at DATETIME,
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_user_created
ON notifications(user_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE notifications;
-- +goose StatementEnd
