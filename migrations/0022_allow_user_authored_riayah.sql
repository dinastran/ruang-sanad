-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_riayah_guru;
DROP INDEX IF EXISTS idx_riayah_target;

ALTER TABLE catatan_riayah RENAME TO catatan_riayah_old;

CREATE TABLE catatan_riayah (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guru_id INTEGER REFERENCES guru(id) ON DELETE CASCADE,
    author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    target_type TEXT NOT NULL DEFAULT 'santri',
    target_id INTEGER NOT NULL,
    catatan TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO catatan_riayah (id, guru_id, author_user_id, target_type, target_id, catatan, created_at, updated_at)
SELECT cr.id, cr.guru_id, g.user_id, cr.target_type, cr.target_id, cr.catatan, cr.created_at, cr.updated_at
FROM catatan_riayah_old cr
LEFT JOIN guru g ON g.id = cr.guru_id;

DROP TABLE catatan_riayah_old;

CREATE INDEX idx_riayah_guru ON catatan_riayah(guru_id);
CREATE INDEX idx_riayah_author_user ON catatan_riayah(author_user_id);
CREATE INDEX idx_riayah_target ON catatan_riayah(target_type, target_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_riayah_guru;
DROP INDEX IF EXISTS idx_riayah_author_user;
DROP INDEX IF EXISTS idx_riayah_target;

ALTER TABLE catatan_riayah RENAME TO catatan_riayah_new;

CREATE TABLE catatan_riayah (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guru_id INTEGER NOT NULL REFERENCES guru(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL DEFAULT 'santri',
    target_id INTEGER NOT NULL,
    catatan TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO catatan_riayah (id, guru_id, target_type, target_id, catatan, created_at, updated_at)
SELECT id, guru_id, target_type, target_id, catatan, created_at, updated_at
FROM catatan_riayah_new
WHERE guru_id IS NOT NULL;

DROP TABLE catatan_riayah_new;

CREATE INDEX idx_riayah_guru ON catatan_riayah(guru_id);
CREATE INDEX idx_riayah_target ON catatan_riayah(target_type, target_id);
-- +goose StatementEnd
