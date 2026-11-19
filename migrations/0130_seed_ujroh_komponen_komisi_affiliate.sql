-- +goose Up
-- +goose StatementBegin
-- Komponen ujroh khusus untuk komisi affiliate guru (dikelola sistem).
INSERT OR IGNORE INTO ujroh_komponen (nama, jenis, nominal_default, is_aktif) VALUES ('Komisi Affiliate', 'insidental', 0, 1);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM ujroh_komponen WHERE nama = 'Komisi Affiliate' AND NOT EXISTS (SELECT 1 FROM ujroh_komponen_bulan b WHERE b.komponen_id = ujroh_komponen.id);
-- +goose StatementEnd
