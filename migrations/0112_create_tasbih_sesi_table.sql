-- +goose Up
-- +goose StatementBegin
-- Satu baris = satu sesi hitungan dzikir. id adalah UUID dari klien (kunci
-- idempotensi sync offline). tanggal_wib/jam_wib dihitung server dari mulai_at.
CREATE TABLE IF NOT EXISTS tasbih_sesi (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    dzikir_id INTEGER NOT NULL REFERENCES tasbih_dzikir(id),
    paket_id INTEGER REFERENCES tasbih_paket(id),
    paket_run_id TEXT,
    jumlah INTEGER NOT NULL CHECK (jumlah BETWEEN 1 AND 100000),
    target INTEGER NOT NULL,
    selesai_target INTEGER NOT NULL DEFAULT 0,
    mulai_at DATETIME NOT NULL,
    selesai_at DATETIME NOT NULL,
    tanggal_wib TEXT NOT NULL,
    jam_wib INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tasbih_sesi_user_tanggal ON tasbih_sesi(user_id, tanggal_wib);
CREATE INDEX IF NOT EXISTS idx_tasbih_sesi_user_mulai ON tasbih_sesi(user_id, mulai_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tasbih_sesi_user_mulai;
DROP INDEX IF EXISTS idx_tasbih_sesi_user_tanggal;
DROP TABLE IF EXISTS tasbih_sesi;
-- +goose StatementEnd
