-- +goose Up
-- +goose StatementBegin
-- Sesi program pendek (mis. Zoom 22, 25, 28 Nov). token = kode link absensi
-- publik /hadir/<token> yang dibagikan di chat Zoom.
CREATE TABLE IF NOT EXISTS program_sesi (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    program_id INTEGER NOT NULL REFERENCES lead_programs(id) ON DELETE CASCADE,
    ke INTEGER NOT NULL,
    judul TEXT NOT NULL DEFAULT '',
    tanggal TEXT NOT NULL,
    jam_mulai TEXT NOT NULL DEFAULT '',
    token TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (program_id, ke)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS program_sesi;
-- +goose StatementEnd
