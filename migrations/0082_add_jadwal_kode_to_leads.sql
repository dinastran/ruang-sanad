-- +goose Up
-- +goose StatementBegin
-- Jadwal pilihan calon santri (jadwal_penawaran.kode). Dicatat CS AI atau CS,
-- dikonfirmasi Admin Kelas saat penempatan.
ALTER TABLE leads ADD COLUMN jadwal_kode TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_leads_jadwal_kode ON leads(jadwal_kode);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_leads_jadwal_kode;
ALTER TABLE leads DROP COLUMN jadwal_kode;
-- +goose StatementEnd
