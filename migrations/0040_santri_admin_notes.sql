-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS santri_admin_note (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    santri_id INTEGER NOT NULL REFERENCES santri(id) ON DELETE CASCADE,
    author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    updated_by_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    catatan TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_santri_admin_note_santri
ON santri_admin_note(santri_id, created_at DESC, id DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_santri_admin_note_santri;
DROP TABLE IF EXISTS santri_admin_note;
-- +goose StatementEnd
