-- +goose Up
-- +goose StatementBegin
UPDATE santri
SET tanggal_daftar = CASE
    WHEN substr(CAST(tanggal_daftar AS TEXT), 1, 10) GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
         AND strftime('%Y-%m-%d', substr(CAST(tanggal_daftar AS TEXT), 1, 10), '+0 days') = substr(CAST(tanggal_daftar AS TEXT), 1, 10)
         AND (
             length(CAST(tanggal_daftar AS TEXT)) = 10
             OR (
                 substr(CAST(tanggal_daftar AS TEXT), 12, 8) GLOB '[0-9][0-9]:[0-9][0-9]:[0-9][0-9]'
                 AND CAST(substr(CAST(tanggal_daftar AS TEXT), 12, 2) AS INTEGER) <= 23
                 AND CAST(substr(CAST(tanggal_daftar AS TEXT), 15, 2) AS INTEGER) <= 59
                 AND CAST(substr(CAST(tanggal_daftar AS TEXT), 18, 2) AS INTEGER) <= 59
                 AND (
                     (length(CAST(tanggal_daftar AS TEXT)) = 19 AND substr(CAST(tanggal_daftar AS TEXT), 11, 1) = ' ')
                     OR (length(CAST(tanggal_daftar AS TEXT)) = 20 AND substr(CAST(tanggal_daftar AS TEXT), 11, 1) = 'T' AND substr(CAST(tanggal_daftar AS TEXT), 20, 1) = 'Z')
                     OR (length(CAST(tanggal_daftar AS TEXT)) = 25 AND substr(CAST(tanggal_daftar AS TEXT), 11, 1) = 'T'
                         AND substr(CAST(tanggal_daftar AS TEXT), 20, 1) IN ('+', '-')
                         AND substr(CAST(tanggal_daftar AS TEXT), 21, 5) GLOB '[0-9][0-9]:[0-9][0-9]'
                         AND CAST(substr(CAST(tanggal_daftar AS TEXT), 21, 2) AS INTEGER) <= 23
                         AND CAST(substr(CAST(tanggal_daftar AS TEXT), 24, 2) AS INTEGER) <= 59)
                     OR (length(CAST(tanggal_daftar AS TEXT)) = 29 AND substr(CAST(tanggal_daftar AS TEXT), 11, 1) = ' '
                         AND substr(CAST(tanggal_daftar AS TEXT), 20, 10) = ' +0000 UTC')
                 )
             )
         )
    THEN substr(CAST(tanggal_daftar AS TEXT), 1, 10)
    ELSE NULL
END
WHERE tanggal_daftar IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- Date normalization is intentionally irreversible.
SELECT 1;
