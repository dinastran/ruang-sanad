-- +goose Up
-- +goose StatementBegin
-- Penilaian rubrik (skala 1-4) per aspek, status kirim ke guru, dan
-- tanggapan guru untuk kunjungan kelas.
ALTER TABLE kunjungan_kelas ADD COLUMN nilai_kedisiplinan INTEGER CHECK (nilai_kedisiplinan BETWEEN 1 AND 4);
ALTER TABLE kunjungan_kelas ADD COLUMN nilai_materi INTEGER CHECK (nilai_materi BETWEEN 1 AND 4);
ALTER TABLE kunjungan_kelas ADD COLUMN nilai_metode INTEGER CHECK (nilai_metode BETWEEN 1 AND 4);
ALTER TABLE kunjungan_kelas ADD COLUMN nilai_interaksi INTEGER CHECK (nilai_interaksi BETWEEN 1 AND 4);
ALTER TABLE kunjungan_kelas ADD COLUMN dikirim_at DATETIME;
ALTER TABLE kunjungan_kelas ADD COLUMN dikirim_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE kunjungan_kelas ADD COLUMN dibaca_at DATETIME;
ALTER TABLE kunjungan_kelas ADD COLUMN tanggapan_guru TEXT NOT NULL DEFAULT '';
ALTER TABLE kunjungan_kelas ADD COLUMN tanggapan_at DATETIME;
CREATE INDEX IF NOT EXISTS idx_kunjungan_guru_dikirim ON kunjungan_kelas(guru_id, dikirim_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_kunjungan_guru_dikirim;
ALTER TABLE kunjungan_kelas DROP COLUMN tanggapan_at;
ALTER TABLE kunjungan_kelas DROP COLUMN tanggapan_guru;
ALTER TABLE kunjungan_kelas DROP COLUMN dibaca_at;
ALTER TABLE kunjungan_kelas DROP COLUMN dikirim_oleh;
ALTER TABLE kunjungan_kelas DROP COLUMN dikirim_at;
ALTER TABLE kunjungan_kelas DROP COLUMN nilai_interaksi;
ALTER TABLE kunjungan_kelas DROP COLUMN nilai_metode;
ALTER TABLE kunjungan_kelas DROP COLUMN nilai_materi;
ALTER TABLE kunjungan_kelas DROP COLUMN nilai_kedisiplinan;
-- +goose StatementEnd
