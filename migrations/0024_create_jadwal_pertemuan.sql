-- +goose Up
-- +goose StatementBegin
CREATE TABLE jadwal_pertemuan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kelas_id INTEGER NOT NULL REFERENCES kelas(id) ON DELETE CASCADE,
    tanggal DATE NOT NULL,
    jam_mulai TEXT NOT NULL,
    catatan TEXT NOT NULL DEFAULT '',
    is_reschedule INTEGER NOT NULL DEFAULT 0,
    jadwal_semula TEXT NOT NULL DEFAULT '',
    alasan_reschedule TEXT NOT NULL DEFAULT '',
    guru_pengganti_id INTEGER REFERENCES guru(id) ON DELETE SET NULL,
    alasan_badal TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'dijadwalkan',
    pertemuan_id INTEGER REFERENCES pertemuan(id) ON DELETE SET NULL,
    dibuat_oleh INTEGER REFERENCES users(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_jadwal_pertemuan_kelas_status
ON jadwal_pertemuan(kelas_id, status, tanggal);

CREATE INDEX idx_jadwal_pertemuan_badal_status
ON jadwal_pertemuan(guru_pengganti_id, status, tanggal);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE jadwal_pertemuan;
-- +goose StatementEnd
