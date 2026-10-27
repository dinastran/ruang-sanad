-- Status kehadiran rinci (hadir/izin/sakit/alfa) untuk analisis alfa berturut-turut.
-- Kolom hadir tetap dipertahankan dan diisi sinkron agar rekap lama tidak berubah.
-- Data lama: hadir=1 -> hadir, selainnya -> alfa (koordinator bisa mengoreksi lewat halaman absen).

-- +goose Up
-- +goose StatementBegin
ALTER TABLE rapat_absen ADD COLUMN kehadiran TEXT NOT NULL DEFAULT 'alfa' CHECK (kehadiran IN ('hadir', 'izin', 'sakit', 'alfa'));
UPDATE rapat_absen SET kehadiran = CASE WHEN hadir = 1 THEN 'hadir' ELSE 'alfa' END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE rapat_absen DROP COLUMN kehadiran;
-- +goose StatementEnd
