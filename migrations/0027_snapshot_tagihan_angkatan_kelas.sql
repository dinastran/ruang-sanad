-- +goose Up
-- +goose StatementBegin
ALTER TABLE tagihan ADD COLUMN angkatan_kelas TEXT NOT NULL DEFAULT '';

UPDATE tagihan
SET angkatan_kelas = COALESCE(
    (SELECT kelas.angkatan FROM kelas WHERE kelas.id = tagihan.kelas_id),
    (SELECT santri.angkatan_kelas FROM santri WHERE santri.id = tagihan.santri_id),
    ''
);

CREATE INDEX idx_tagihan_angkatan_kelas ON tagihan(angkatan_kelas);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tagihan_angkatan_kelas;
ALTER TABLE tagihan DROP COLUMN angkatan_kelas;
-- +goose StatementEnd
