-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS lead_label_links (
    lead_id INTEGER NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    label_id INTEGER NOT NULL REFERENCES lead_labels(id) ON DELETE CASCADE,
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (lead_id, label_id)
);

CREATE INDEX IF NOT EXISTS idx_lead_label_links_label ON lead_label_links(label_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_lead_label_links_label;
DROP TABLE IF EXISTS lead_label_links;
-- +goose StatementEnd
