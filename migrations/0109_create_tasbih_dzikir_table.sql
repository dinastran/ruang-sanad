-- +goose Up
-- +goose StatementBegin
-- Dzikir untuk tasbih digital. user_id NULL = dzikir bawaan (seed), selain itu
-- dzikir custom milik user tersebut.
CREATE TABLE IF NOT EXISTS tasbih_dzikir (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    slug TEXT UNIQUE,
    teks_arab TEXT NOT NULL DEFAULT '',
    latin TEXT NOT NULL,
    arti TEXT NOT NULL DEFAULT '',
    rujukan TEXT NOT NULL DEFAULT '',
    target_default INTEGER NOT NULL CHECK (target_default BETWEEN 1 AND 100000),
    urutan INTEGER NOT NULL DEFAULT 0,
    diarsipkan_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tasbih_dzikir_user ON tasbih_dzikir(user_id);

INSERT INTO tasbih_dzikir (slug, teks_arab, latin, arti, rujukan, target_default, urutan) VALUES
('subhanallah', 'سُبْحَانَ اللهِ', 'Subhanallah', 'Maha Suci Allah', 'HR. Muslim no. 597', 33, 1),
('alhamdulillah', 'الْحَمْدُ لِلَّهِ', 'Alhamdulillah', 'Segala puji bagi Allah', 'HR. Muslim no. 597', 33, 2),
('allahu-akbar', 'اللهُ أَكْبَرُ', 'Allahu Akbar', 'Allah Maha Besar', 'HR. Muslim no. 597', 33, 3),
('tahlil', 'لَا إِلَهَ إِلَّا اللهُ', 'Laa ilaaha illallah', 'Tiada sesembahan yang berhak disembah selain Allah', 'HR. Tirmidzi no. 3383', 100, 4),
('tahlil-lengkap', 'لَا إِلَهَ إِلَّا اللهُ وَحْدَهُ لَا شَرِيكَ لَهُ، لَهُ الْمُلْكُ وَلَهُ الْحَمْدُ، وَهُوَ عَلَى كُلِّ شَيْءٍ قَدِيرٌ', 'Laa ilaaha illallahu wahdahu laa syariika lah, lahul mulku wa lahul hamdu wa huwa ''alaa kulli syai-in qadiir', 'Tiada sesembahan yang berhak disembah selain Allah semata, tiada sekutu bagi-Nya. Milik-Nya kerajaan dan milik-Nya segala puji, dan Dia Maha Kuasa atas segala sesuatu', 'HR. Bukhari no. 6403, Muslim no. 2691', 100, 5),
('istighfar', 'أَسْتَغْفِرُ اللهَ', 'Astaghfirullah', 'Aku memohon ampun kepada Allah', 'HR. Muslim no. 2702', 100, 6),
('istighfar-taubat', 'أَسْتَغْفِرُ اللهَ وَأَتُوبُ إِلَيْهِ', 'Astaghfirullaha wa atuubu ilaih', 'Aku memohon ampun kepada Allah dan bertaubat kepada-Nya', 'HR. Bukhari no. 6307', 100, 7),
('tasbih-tahmid', 'سُبْحَانَ اللهِ وَبِحَمْدِهِ', 'Subhanallahi wa bihamdihi', 'Maha Suci Allah dan segala puji bagi-Nya', 'HR. Bukhari no. 6405, Muslim no. 2691', 100, 8),
('tasbih-azhim', 'سُبْحَانَ اللهِ وَبِحَمْدِهِ، سُبْحَانَ اللهِ الْعَظِيمِ', 'Subhanallahi wa bihamdihi, subhanallahil ''azhiim', 'Maha Suci Allah dan segala puji bagi-Nya, Maha Suci Allah Yang Maha Agung', 'HR. Bukhari no. 6406, Muslim no. 2694', 33, 9),
('hauqalah', 'لَا حَوْلَ وَلَا قُوَّةَ إِلَّا بِاللهِ', 'Laa haula wa laa quwwata illaa billaah', 'Tiada daya dan kekuatan kecuali dengan pertolongan Allah', 'HR. Bukhari no. 6384, Muslim no. 2704', 33, 10),
('shalawat', 'اللَّهُمَّ صَلِّ عَلَى مُحَمَّدٍ', 'Allahumma shalli ''alaa Muhammad', 'Ya Allah, limpahkanlah shalawat kepada Muhammad', 'HR. Muslim no. 408 (keutamaan bershalawat)', 100, 11),
('hasbunallah', 'حَسْبُنَا اللهُ وَنِعْمَ الْوَكِيلُ', 'Hasbunallaahu wa ni''mal wakiil', 'Cukuplah Allah bagi kami, dan Dia sebaik-baik pelindung', 'QS. Ali ''Imran: 173; HR. Bukhari no. 4563', 33, 12);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tasbih_dzikir_user;
DROP TABLE IF EXISTS tasbih_dzikir;
-- +goose StatementEnd
