-- +goose Up
-- +goose StatementBegin
-- Ganti level & ganti jadwal kelas oleh Admin Kelas.
--
-- pertemuan.pertemuan_ke tetap berurutan sepanjang umur kelas karena tagihan
-- (bulan_ke) dan UNIQUE(kelas_id, pertemuan_ke) bergantung padanya. Nomor yang
-- dilihat pengguna setelah kelas naik level adalah pertemuan_level_ke, yang
-- dihitung dari kelas.level_pertemuan_awal (pertemuan_ke terakhir di level lama).
ALTER TABLE kelas ADD COLUMN level_pertemuan_awal INTEGER NOT NULL DEFAULT 0;

ALTER TABLE pertemuan ADD COLUMN level_nama TEXT NOT NULL DEFAULT '';
ALTER TABLE pertemuan ADD COLUMN pertemuan_level_ke INTEGER NOT NULL DEFAULT 0;

UPDATE pertemuan
SET pertemuan_level_ke = pertemuan_ke,
    level_nama = COALESCE((
        SELECT COALESCE(NULLIF(l.nama, ''), k.level)
        FROM kelas k
        LEFT JOIN level l ON l.kode = k.level
        WHERE k.id = pertemuan.kelas_id
    ), '');

-- Sesi terjadwal yang dibuat sebelum jadwal rutin kelas diganti.
ALTER TABLE jadwal_pertemuan ADD COLUMN jadwal_kelas_berubah INTEGER NOT NULL DEFAULT 0;

-- Riwayat perubahan level/jadwal. jenis: level_kelas, jadwal_kelas, level_santri.
CREATE TABLE kelas_perubahan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kelas_id INTEGER REFERENCES kelas(id) ON DELETE SET NULL,
    santri_id INTEGER REFERENCES santri(id) ON DELETE CASCADE,
    kelas_tujuan_id INTEGER REFERENCES kelas(id) ON DELETE SET NULL,
    jenis TEXT NOT NULL,
    nilai_lama TEXT NOT NULL DEFAULT '',
    nilai_baru TEXT NOT NULL DEFAULT '',
    pertemuan_ke INTEGER NOT NULL DEFAULT 0,
    dibuat_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_kelas_perubahan_kelas ON kelas_perubahan(kelas_id, created_at DESC);
CREATE INDEX idx_kelas_perubahan_santri ON kelas_perubahan(santri_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE kelas_perubahan;
ALTER TABLE jadwal_pertemuan DROP COLUMN jadwal_kelas_berubah;
ALTER TABLE pertemuan DROP COLUMN pertemuan_level_ke;
ALTER TABLE pertemuan DROP COLUMN level_nama;
ALTER TABLE kelas DROP COLUMN level_pertemuan_awal;
-- +goose StatementEnd
