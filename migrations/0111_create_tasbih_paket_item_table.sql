-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tasbih_paket_item (
    paket_id INTEGER NOT NULL REFERENCES tasbih_paket(id) ON DELETE CASCADE,
    urutan INTEGER NOT NULL,
    dzikir_id INTEGER NOT NULL REFERENCES tasbih_dzikir(id),
    target INTEGER NOT NULL CHECK (target BETWEEN 1 AND 100000),
    PRIMARY KEY (paket_id, urutan)
);

INSERT INTO tasbih_paket_item (paket_id, urutan, dzikir_id, target) VALUES
((SELECT id FROM tasbih_paket WHERE slug = 'badah-shalat'), 1, (SELECT id FROM tasbih_dzikir WHERE slug = 'subhanallah'), 33),
((SELECT id FROM tasbih_paket WHERE slug = 'badah-shalat'), 2, (SELECT id FROM tasbih_dzikir WHERE slug = 'alhamdulillah'), 33),
((SELECT id FROM tasbih_paket WHERE slug = 'badah-shalat'), 3, (SELECT id FROM tasbih_dzikir WHERE slug = 'allahu-akbar'), 33),
((SELECT id FROM tasbih_paket WHERE slug = 'badah-shalat'), 4, (SELECT id FROM tasbih_dzikir WHERE slug = 'tahlil-lengkap'), 1),
((SELECT id FROM tasbih_paket WHERE slug = 'tasbih-fatimah'), 1, (SELECT id FROM tasbih_dzikir WHERE slug = 'subhanallah'), 33),
((SELECT id FROM tasbih_paket WHERE slug = 'tasbih-fatimah'), 2, (SELECT id FROM tasbih_dzikir WHERE slug = 'alhamdulillah'), 33),
((SELECT id FROM tasbih_paket WHERE slug = 'tasbih-fatimah'), 3, (SELECT id FROM tasbih_dzikir WHERE slug = 'allahu-akbar'), 34),
((SELECT id FROM tasbih_paket WHERE slug = 'istighfar-100'), 1, (SELECT id FROM tasbih_dzikir WHERE slug = 'istighfar'), 100);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tasbih_paket_item;
-- +goose StatementEnd
