-- +goose Up
-- +goose StatementBegin
-- Label untuk uji follow-up bertahap (PRD 15.2): isi crm_settings.followup_only_label
-- dengan "uji-followup" lalu pasang label ini ke sebagian kecil lead.
INSERT OR IGNORE INTO lead_labels (nama, warna) VALUES ('uji-followup', 'secondary');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM lead_labels WHERE nama = 'uji-followup'
  AND NOT EXISTS (SELECT 1 FROM lead_label_links k WHERE k.label_id = lead_labels.id);
-- +goose StatementEnd
