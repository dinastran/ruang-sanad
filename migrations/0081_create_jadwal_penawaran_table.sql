-- +goose Up
-- +goose StatementBegin
-- Jadwal kelas yang ditawarkan ke calon santri (CS AI + CRM). Satu baris per
-- slot. Kuota berlaku per gender; kursi terhitung terpakai saat pembayaran
-- lead diverifikasi sejak periode_mulai.
CREATE TABLE IF NOT EXISTS jadwal_penawaran (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kode TEXT NOT NULL UNIQUE,
    -- hari 1=Senin .. 7=Ahad, dipisah koma ("3,6" = Rabu & Sabtu).
    hari TEXT NOT NULL,
    jam_mulai TEXT NOT NULL,
    jam_selesai TEXT NOT NULL,
    -- '' = semua level, selain itu '1' atau '2'.
    level TEXT NOT NULL DEFAULT '',
    kuota_per_gender INTEGER NOT NULL DEFAULT 10,
    is_buka INTEGER NOT NULL DEFAULT 1,
    urutan INTEGER NOT NULL DEFAULT 0,
    catatan TEXT NOT NULL DEFAULT '',
    periode_mulai DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS jadwal_penawaran;
-- +goose StatementEnd
