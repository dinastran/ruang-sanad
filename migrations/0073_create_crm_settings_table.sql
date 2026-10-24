-- +goose Up
-- +goose StatementBegin
-- Pengaturan CRM non-secret. Satu baris (id = 1), diubah super_admin.
-- Secret (token bridge, akun GOWA) tetap di .env.
CREATE TABLE IF NOT EXISTS crm_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    paket_ke_kelas_kode TEXT NOT NULL DEFAULT '{"1x":"R 1X","2x":"R 2X"}',
    followup_enabled INTEGER NOT NULL DEFAULT 0,
    followup_send_start TEXT NOT NULL DEFAULT '08:00',
    followup_send_end TEXT NOT NULL DEFAULT '20:00',
    followup_min_gap_seconds INTEGER NOT NULL DEFAULT 45,
    followup_daily_cap INTEGER NOT NULL DEFAULT 40,
    transcript_retention_days_lost INTEGER NOT NULL DEFAULT 180,
    gowa_devices TEXT NOT NULL DEFAULT '{"keuangan":"keuangan","admin_kelas":"admin-kelas","koordinator":"koord-guru"}',
    payment_verified_template TEXT NOT NULL DEFAULT '',
    updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO crm_settings (id) VALUES (1);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS crm_settings;
-- +goose StatementEnd
