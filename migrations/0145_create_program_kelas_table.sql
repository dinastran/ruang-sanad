-- +goose Up
-- +goose StatementBegin
-- Kelas program pendek, dipisah ikhwan (L) / akhwat (P). Nama tampil
-- dibentuk dari jenis kelamin + urutan ("Akhwat 2"). Kelas dibuat otomatis
-- saat kelas sebelumnya penuh (lead_programs.kapasitas_kelas).
CREATE TABLE IF NOT EXISTS program_kelas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    program_id INTEGER NOT NULL REFERENCES lead_programs(id) ON DELETE CASCADE,
    jenis_kelamin TEXT NOT NULL CHECK (jenis_kelamin IN ('L', 'P')),
    urutan INTEGER NOT NULL,
    pengajar TEXT NOT NULL DEFAULT '',
    zoom_url TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (program_id, jenis_kelamin, urutan)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS program_kelas;
-- +goose StatementEnd
