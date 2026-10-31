-- +goose Up
-- +goose StatementBegin
-- Sumber awal dari contoh teks yang masuk. Bisa diubah di /app/leads/sumber.
INSERT OR IGNORE INTO lead_sources (nama, keterangan, is_aktif, created_at, updated_at) VALUES
    ('LP Kelas Reguler Tahsin', 'Landing page Kelas Reguler Tahsin', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('Iklan Meta (teks default)', 'Teks bawaan iklan Click-to-WhatsApp Meta. Tidak membedakan iklan/LP; beri teks unik di setiap iklan agar bisa dibedakan.', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT OR IGNORE INTO lead_source_patterns (source_id, teks, teks_normal, created_at)
SELECT id, 'saya ingin tanya tentang Kelas Reguler Tahsin Ruang Sanad', 'saya ingin tanya tentang kelas reguler tahsin ruang sanad', CURRENT_TIMESTAMP
FROM lead_sources WHERE nama = 'LP Kelas Reguler Tahsin';
INSERT OR IGNORE INTO lead_source_patterns (source_id, teks, teks_normal, created_at)
SELECT id, 'Bisakah saya mendapatkan info selengkapnya tentang ini?', 'bisakah saya mendapatkan info selengkapnya tentang ini', CURRENT_TIMESTAMP
FROM lead_sources WHERE nama = 'Iklan Meta (teks default)';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM lead_sources WHERE nama IN ('LP Kelas Reguler Tahsin', 'Iklan Meta (teks default)');
-- +goose StatementEnd
