-- +goose Up
-- +goose StatementBegin
INSERT INTO tasbih_paket (slug, nama, deskripsi, rujukan, urutan) VALUES
('matsurat-pagi', 'Al-Ma''tsurat Sughra · Pagi', 'Wirid pagi Hasan Al-Banna (versi sughra): Al-Fatihah, ayat-ayat Al-Baqarah, Al-Mu’awwidzat, dzikir dan doa pagi, ditutup Doa Rabithah.', 'Al-Ma''tsurat (Hasan Al-Banna)', 4),
('matsurat-petang', 'Al-Ma''tsurat Sughra · Petang', 'Wirid petang Hasan Al-Banna (versi sughra): lafaz pagi diganti versi petang (amsaina/amsaitu/amsa) dan ditambah doa iqbalu lailik.', 'Al-Ma''tsurat (Hasan Al-Banna)', 5);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM tasbih_paket WHERE slug IN ('matsurat-pagi', 'matsurat-petang');
-- +goose StatementEnd
