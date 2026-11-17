-- +goose Up
-- +goose StatementBegin
-- Paket wirid bawaan (urutan dzikir yang dibaca berurutan).
CREATE TABLE IF NOT EXISTS tasbih_paket (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    nama TEXT NOT NULL,
    deskripsi TEXT NOT NULL DEFAULT '',
    rujukan TEXT NOT NULL DEFAULT '',
    urutan INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO tasbih_paket (slug, nama, deskripsi, rujukan, urutan) VALUES
('badah-shalat', 'Dzikir Ba''da Shalat', 'Tasbih 33, tahmid 33, takbir 33, lalu ditutup tahlil sehingga genap 100.', 'HR. Muslim no. 597', 1),
('tasbih-fatimah', 'Tasbih Fatimah (Sebelum Tidur)', 'Tasbih 33, tahmid 33, takbir 34 menjelang tidur.', 'HR. Bukhari no. 3705, Muslim no. 2727', 2),
('istighfar-100', 'Istighfar 100', 'Istighfar seratus kali dalam sehari.', 'HR. Muslim no. 2702', 3);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tasbih_paket;
-- +goose StatementEnd
