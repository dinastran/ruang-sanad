-- +goose Up
-- +goose StatementBegin
ALTER TABLE santri ADD COLUMN angkatan_kelas TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_santri_angkatan_kelas ON santri(angkatan_kelas);

-- Preserve useful unassigned legacy placement only while its old shared cohort
-- still exists in current master data. Historical codes missing from master are
-- not assumed to be current class placement.
UPDATE santri
SET angkatan_kelas = angkatan
WHERE EXISTS (SELECT 1 FROM angkatan AS master_angkatan WHERE master_angkatan.kode = santri.angkatan);

-- Issued IDs are the source of truth for legacy registration cohorts. The
-- sequence marker is anchored to the row ID so cohort codes may contain dots.
UPDATE santri
SET angkatan = substr(
        id_mahasantri,
        5,
        length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6
    )
WHERE substr(id_mahasantri, 1, 4) = 'MHS.'
  AND length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6 > 0
  AND substr(id_mahasantri, 5, length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6)
      = trim(substr(id_mahasantri, 5, length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6), ' ' || char(9) || char(10) || char(11) || char(12) || char(13))
  AND substr(
        id_mahasantri,
        5 + length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6
      ) = ('.' || printf('%04d', id) || '.' || substr(id_mahasantri, -6))
  AND substr(id_mahasantri, -6) GLOB '[0-9][0-9][0-9][0-9][0-9][0-9]'
  AND CAST(substr(id_mahasantri, -6, 2) AS INTEGER) BETWEEN 1 AND 12;

-- MMYYYY can disprove an exact date, but it can never supply an exact day.
UPDATE santri
SET tanggal_daftar = NULL
WHERE tanggal_daftar IS NOT NULL
  AND substr(id_mahasantri, 1, 4) = 'MHS.'
  AND length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6 > 0
  AND substr(id_mahasantri, 5, length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6)
      = trim(substr(id_mahasantri, 5, length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6), ' ' || char(9) || char(10) || char(11) || char(12) || char(13))
  AND substr(
        id_mahasantri,
        5 + length(id_mahasantri) - 4 - length('.' || printf('%04d', id) || '.') - 6
      ) = ('.' || printf('%04d', id) || '.' || substr(id_mahasantri, -6))
  AND substr(id_mahasantri, -6) GLOB '[0-9][0-9][0-9][0-9][0-9][0-9]'
  AND CAST(substr(id_mahasantri, -6, 2) AS INTEGER) BETWEEN 1 AND 12
  AND COALESCE(strftime('%m%Y', substr(CAST(tanggal_daftar AS TEXT), 1, 10), '+0 days'), '') != substr(id_mahasantri, -6);

UPDATE santri
SET angkatan_kelas = COALESCE((SELECT kelas.angkatan FROM kelas WHERE kelas.id = santri.kelas_id), '')
WHERE kelas_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Roll-forward only: dropping the structural column cannot restore the former
-- conflated registration/class semantics or values changed by the Up migration.
DROP INDEX IF EXISTS idx_santri_angkatan_kelas;
ALTER TABLE santri DROP COLUMN angkatan_kelas;
-- +goose StatementEnd
