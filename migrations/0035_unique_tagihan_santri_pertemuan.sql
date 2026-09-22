-- +goose Up
-- +goose StatementBegin
-- Satu tagihan per santri per pertemuan. bulan_ke santri pindahan bisa
-- dilanjutkan (bukan dari nomor pertemuan kelas), jadi (santri_id, bulan_ke)
-- saja tidak lagi mencegah tagihan ganda untuk pertemuan yang sama.
CREATE UNIQUE INDEX IF NOT EXISTS idx_tagihan_santri_pertemuan
ON tagihan(santri_id, pertemuan_id) WHERE pertemuan_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tagihan_santri_pertemuan;
-- +goose StatementEnd
