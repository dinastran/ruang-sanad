-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tagihan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    santri_id INTEGER NOT NULL REFERENCES santri(id) ON DELETE CASCADE,
    kelas_id INTEGER REFERENCES kelas(id) ON DELETE SET NULL,
    pertemuan_id INTEGER REFERENCES pertemuan(id) ON DELETE SET NULL,
    bulan_ke INTEGER NOT NULL,
    pertemuan_ke INTEGER NOT NULL,
    nominal INTEGER NOT NULL DEFAULT 0,
    tanggal_tagih DATE NOT NULL,
    jatuh_tempo DATE,
    status TEXT NOT NULL DEFAULT 'belum_bayar',
    tanggal_bayar DATE,
    metode TEXT NOT NULL DEFAULT '',
    catatan TEXT NOT NULL DEFAULT '',
    dicatat_oleh INTEGER REFERENCES users(id) ON DELETE SET NULL,
    fu_terakhir DATETIME,
    fu_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(santri_id, bulan_ke)
);

ALTER TABLE santri ADD COLUMN pertemuan_awal INTEGER NOT NULL DEFAULT 0;

CREATE INDEX idx_tagihan_santri ON tagihan(santri_id);
CREATE INDEX idx_tagihan_kelas ON tagihan(kelas_id);
CREATE INDEX idx_tagihan_status ON tagihan(status);
CREATE INDEX idx_tagihan_tanggal ON tagihan(tanggal_tagih);

-- Editable from the existing Template WA manager. Only one template should be
-- active at a time; the first active template is used for follow-up links.
INSERT INTO wa_template (nama, target_type, body, is_aktif) VALUES
    ('Tagihan SPP - Standar', 'tagihan', 'Assalamu''alaikum {nama}, kami informasikan tagihan SPP Ruang Sanad bulan ke-{bulan_ke} sebesar Rp{nominal} telah terbit (tanggal {tanggal_tagih}) untuk kelas {kelas}. Mohon konfirmasi pembayarannya. Jazaakumullah khairan.', 1),
    ('Tagihan SPP - Singkat', 'tagihan', 'Assalamu''alaikum {nama}, tagihan SPP bulan ke-{bulan_ke} kelas {kelas} sebesar Rp{nominal} telah terbit pada {tanggal_tagih}. Mohon konfirmasi pembayaran. Jazaakumullah khairan.', 0),
    ('Tagihan SPP - Jatuh Tempo', 'tagihan', 'Assalamu''alaikum {nama}, kami mengingatkan tagihan SPP bulan ke-{bulan_ke} sebesar Rp{nominal}. Mohon konfirmasi pembayarannya. Jazaakumullah khairan.', 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tagihan_tanggal;
DROP INDEX IF EXISTS idx_tagihan_status;
DROP INDEX IF EXISTS idx_tagihan_kelas;
DROP INDEX IF EXISTS idx_tagihan_santri;
DROP TABLE IF EXISTS tagihan;
ALTER TABLE santri DROP COLUMN pertemuan_awal;
-- +goose StatementEnd
