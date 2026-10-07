-- +goose Up
-- +goose StatementBegin
-- Default ujroh (teacher pay) per completed meeting, per guru status.
CREATE TABLE IF NOT EXISTS ujroh_tarif (
    status TEXT PRIMARY KEY CHECK (status IN ('tetap', 'part_time')),
    nominal INTEGER NOT NULL CHECK (nominal >= 0),
    diubah_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO ujroh_tarif (status, nominal) VALUES ('tetap', 75000), ('part_time', 50000)
ON CONFLICT(status) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ujroh_tarif;
-- +goose StatementEnd
