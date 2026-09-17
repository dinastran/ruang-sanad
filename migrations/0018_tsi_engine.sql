-- +goose Up
-- +goose StatementBegin

-- Master 18 indikator TSI. Bobot disimpan per kategori (40/30/20/10).
-- sumber: auto (dihitung sistem) / semi (auto+konfirmasi) / manual.
-- kode: kunci untuk indikator otomatis (kosong untuk manual/semi).
CREATE TABLE IF NOT EXISTS tsi_kriteria (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kategori TEXT NOT NULL,          -- kompetensi/kepuasan/kedisiplinan/kontribusi
    bobot INTEGER NOT NULL,          -- 40/30/20/10 (persen kategori)
    urutan INTEGER NOT NULL,
    sumber TEXT NOT NULL,            -- auto/semi/manual
    kode TEXT NOT NULL DEFAULT '',
    nama TEXT NOT NULL
);

-- Periode penilaian per guru per bulan.
CREATE TABLE IF NOT EXISTS tsi_periode (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guru_id INTEGER NOT NULL REFERENCES guru(id) ON DELETE CASCADE,
    bulan TEXT NOT NULL,             -- YYYY-MM
    status TEXT NOT NULL DEFAULT 'draft', -- draft/final
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(guru_id, bulan)
);
CREATE INDEX IF NOT EXISTS idx_tsi_periode_guru ON tsi_periode(guru_id);

-- Nilai per indikator per periode. nilai NULL = belum diisi (dikecualikan dari rata-rata).
CREATE TABLE IF NOT EXISTS tsi_nilai (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    periode_id INTEGER NOT NULL REFERENCES tsi_periode(id) ON DELETE CASCADE,
    kriteria_id INTEGER NOT NULL REFERENCES tsi_kriteria(id) ON DELETE CASCADE,
    nilai REAL,
    is_override INTEGER NOT NULL DEFAULT 0,
    alasan_override TEXT NOT NULL DEFAULT '',
    catatan TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(periode_id, kriteria_id)
);

-- Audit trail perubahan nilai (khususnya override nilai otomatis).
CREATE TABLE IF NOT EXISTS tsi_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    periode_id INTEGER NOT NULL REFERENCES tsi_periode(id) ON DELETE CASCADE,
    kriteria_id INTEGER NOT NULL REFERENCES tsi_kriteria(id) ON DELETE CASCADE,
    nilai_lama REAL,
    nilai_baru REAL,
    alasan TEXT NOT NULL DEFAULT '',
    oleh INTEGER REFERENCES users(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Seed 18 indikator (verbatim dari spreadsheet TSI).
INSERT INTO tsi_kriteria (kategori, bobot, urutan, sumber, kode, nama) VALUES
('kompetensi', 40, 1, 'manual', '', 'Kesesuaian dengan teknik mengajar metode AQU'),
('kompetensi', 40, 2, 'semi', '', 'Kesesuaian dengan kurikulum Ruang Sanad'),
('kompetensi', 40, 3, 'manual', '', 'Tampilan mengajar sesuai standar R Sanad'),
('kompetensi', 40, 4, 'auto', 'absen_riayah', 'Riayah grup dengan update absen setelah pertemuan'),
('kompetensi', 40, 5, 'semi', '', 'Menyapa grup di luar waktu mengajar (quotes, SS, kuis, PR, jawab pertanyaan)'),
('kompetensi', 40, 6, 'manual', '', 'Ketepatan pelafalan'),
('kepuasan', 30, 1, 'auto', 'retensi', 'Tidak banyak maha santri yang mundur'),
('kepuasan', 30, 2, 'manual', '', 'Kelas tidak dimerger'),
('kepuasan', 30, 3, 'manual', '', 'Tidak ada komplain dari maha santri di grup ataupun via admin kelas'),
('kepuasan', 30, 4, 'manual', '', 'Maha santri mengapresiasi guru dan berkenan terlibat mensosialisasikan ruang sanad'),
('kedisiplinan', 20, 1, 'auto', 'absen_dibuat', 'Membuat absen di kelas setiap selesai mengajar'),
('kedisiplinan', 20, 2, 'auto', 'no_reschedule', 'Tidak reschedule kelas'),
('kedisiplinan', 20, 3, 'auto', 'no_badal', 'Tidak sering dibadal'),
('kedisiplinan', 20, 4, 'manual', '', 'Memulai dan mengakhiri kelas sesuai waktu (tidak telat)'),
('kedisiplinan', 20, 5, 'auto', 'pembinaan', 'Mengikuti pembinaan'),
('kedisiplinan', 20, 6, 'auto', 'rapat', 'Menghadiri rapat guru'),
('kontribusi', 10, 1, 'auto', 'share_kalam', 'Share info Kalam Bersanad'),
('kontribusi', 10, 2, 'manual', '', 'Posting SS mengajar');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tsi_audit;
DROP TABLE IF EXISTS tsi_nilai;
DROP TABLE IF EXISTS tsi_periode;
DROP TABLE IF EXISTS tsi_kriteria;
-- +goose StatementEnd
