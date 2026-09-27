-- +goose Up
-- +goose StatementBegin

-- Menandai tagihan yang nominalnya dioverride manual oleh Keuangan.
-- Nilai efektif tetap disimpan di tagihan.nominal agar dashboard/laporan lama
-- tetap membaca nominal yang benar tanpa perubahan query agregasi.
CREATE TABLE IF NOT EXISTS tagihan_nominal_override (
    tagihan_id INTEGER PRIMARY KEY REFERENCES tagihan(id) ON DELETE CASCADE,
    nominal INTEGER NOT NULL CHECK (nominal >= 0),
    updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Riwayat setiap follow-up yang dibuka dari modul tagihan.
CREATE TABLE IF NOT EXISTS tagihan_follow_up_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tagihan_id INTEGER NOT NULL REFERENCES tagihan(id) ON DELETE CASCADE,
    template_id INTEGER REFERENCES wa_template(id) ON DELETE SET NULL,
    template_nama TEXT NOT NULL DEFAULT '',
    message_body TEXT NOT NULL DEFAULT '',
    dikirim_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tagihan_follow_up_log_tagihan
ON tagihan_follow_up_log(tagihan_id, created_at DESC);

-- Backfill sekali untuk memperbaiki tagihan terbuka yang masih menyimpan
-- snapshot nominal lama sebelum fitur sinkronisasi ini tersedia.
UPDATE tagihan
SET nominal = (
        SELECT s.nominal FROM santri s WHERE s.id = tagihan.santri_id
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE status = 'belum_bayar';

-- Jadikan tiga template bawaan tagihan sebagai tahapan FU awal.
UPDATE wa_template
SET nama = 'FU 1 - Pengingat Tagihan', is_aktif = 1, updated_at = CURRENT_TIMESTAMP
WHERE target_type = 'tagihan' AND nama = 'Tagihan SPP - Standar';

UPDATE wa_template
SET nama = 'FU 2 - Pengingat Kedua', is_aktif = 1, updated_at = CURRENT_TIMESTAMP
WHERE target_type = 'tagihan' AND nama = 'Tagihan SPP - Singkat';

UPDATE wa_template
SET nama = 'FU 3 - Jatuh Tempo', is_aktif = 1, updated_at = CURRENT_TIMESTAMP
WHERE target_type = 'tagihan' AND nama = 'Tagihan SPP - Jatuh Tempo';

-- Pastikan instalasi yang tidak memiliki seed lama tetap mendapat minimal 3 template.
INSERT INTO wa_template (nama, target_type, body, is_aktif)
SELECT 'FU 1 - Pengingat Tagihan', 'tagihan',
       'Assalamu''alaikum {nama}, kami informasikan tagihan SPP Ruang Sanad bulan ke-{bulan_ke} sebesar Rp{nominal} telah terbit pada {tanggal_tagih} untuk kelas {kelas}. Mohon konfirmasi pembayarannya. Jazaakumullah khairan.',
       1
WHERE NOT EXISTS (
    SELECT 1 FROM wa_template WHERE target_type = 'tagihan' AND nama = 'FU 1 - Pengingat Tagihan'
);

INSERT INTO wa_template (nama, target_type, body, is_aktif)
SELECT 'FU 2 - Pengingat Kedua', 'tagihan',
       'Assalamu''alaikum {nama}, izin mengingatkan kembali tagihan SPP bulan ke-{bulan_ke} sebesar Rp{nominal} untuk kelas {kelas}. Jatuh tempo: {jatuh_tempo}. Mohon konfirmasi jika pembayaran sudah dilakukan. Jazaakumullah khairan.',
       1
WHERE NOT EXISTS (
    SELECT 1 FROM wa_template WHERE target_type = 'tagihan' AND nama = 'FU 2 - Pengingat Kedua'
);

INSERT INTO wa_template (nama, target_type, body, is_aktif)
SELECT 'FU 3 - Jatuh Tempo', 'tagihan',
       'Assalamu''alaikum {nama}, kami mengingatkan tagihan SPP bulan ke-{bulan_ke} sebesar Rp{nominal} untuk kelas {kelas} yang jatuh tempo pada {jatuh_tempo}. Mohon kabari kami terkait status pembayarannya. Jazaakumullah khairan.',
       1
WHERE NOT EXISTS (
    SELECT 1 FROM wa_template WHERE target_type = 'tagihan' AND nama = 'FU 3 - Jatuh Tempo'
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tagihan_follow_up_log_tagihan;
DROP TABLE IF EXISTS tagihan_follow_up_log;
DROP TABLE IF EXISTS tagihan_nominal_override;
-- Template WA tidak dihapus pada down karena bisa sudah diedit pengguna.
-- +goose StatementEnd
