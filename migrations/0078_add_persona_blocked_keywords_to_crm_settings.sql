-- +goose Up
-- +goose StatementBegin
-- Kata/frasa yang tidak boleh muncul di persona.catatan analisis AI (PRD 9.3):
-- penilaian tingkat keagamaan, ekonomi, kesehatan, etnis, politik.
-- Satu kata/frasa per baris, dicocokkan per kata utuh tanpa beda huruf besar-kecil.
ALTER TABLE crm_settings ADD COLUMN persona_blocked_keywords TEXT NOT NULL DEFAULT 'religius
alim
awam agama
kurang iman
lemah iman
fasik
munafik
saleh
sholeh
taat beragama
tidak taat
miskin
kaya
ekonomi lemah
ekonomi rendah
ekonomi menengah
penghasilan
pengangguran
tidak mampu secara ekonomi
penyakit
disabilitas
cacat
depresi
gangguan jiwa
suku
etnis
ras
keturunan
pribumi
tionghoa
politik
partai
pemilu
capres';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE crm_settings DROP COLUMN persona_blocked_keywords;
-- +goose StatementEnd
