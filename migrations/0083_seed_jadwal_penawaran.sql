-- +goose Up
-- +goose StatementBegin
-- Jadwal awal Ruang Sanad (bisa diubah di /app/jadwal-penawaran).
INSERT OR IGNORE INTO jadwal_penawaran (kode, hari, jam_mulai, jam_selesai, level, kuota_per_gender, is_buka, urutan, periode_mulai, created_at, updated_at) VALUES
    ('selasa-malam', '2', '19:30', '21:00', '2', 10, 1, 10, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('jumat-malam', '5', '19:30', '21:00', '1', 10, 1, 20, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('ahad-pagi', '7', '05:30', '07:00', '', 10, 1, 30, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('rabu-sabtu-malam', '3,6', '19:30', '21:00', '', 10, 1, 40, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM jadwal_penawaran WHERE kode IN ('selasa-malam', 'jumat-malam', 'ahad-pagi', 'rabu-sabtu-malam');
-- +goose StatementEnd
