package models

// TsiKriteriaNilai is one indicator with its effective and raw-auto value.
type TsiKriteriaNilai struct {
	KriteriaID     int64    `json:"kriteria_id"`
	Kategori       string   `json:"kategori"`
	Bobot          int64    `json:"bobot"`
	Urutan         int64    `json:"urutan"`
	Sumber         string   `json:"sumber"`
	Kode           string   `json:"kode"`
	Nama           string   `json:"nama"`
	Nilai          *float64 `json:"nilai"`      // effective value used in the average
	AutoNilai      *float64 `json:"auto_nilai"` // computed automatic value (transparency)
	RawDisplay     string   `json:"raw_display"`
	IsOverride     bool     `json:"is_override"`
	AlasanOverride string   `json:"alasan_override"`
	Catatan        string   `json:"catatan"`
}

type TsiKategoriSkor struct {
	Kategori string   `json:"kategori"`
	Bobot    int64    `json:"bobot"`
	RataRata *float64 `json:"rata_rata"`
	Skor     float64  `json:"skor"`
	Terisi   int      `json:"terisi"`
	Total    int      `json:"total"`
}

type TsiPenilaianResponse struct {
	PeriodeID int64              `json:"periode_id"`
	GuruID    int64              `json:"guru_id"`
	GuruNama  string             `json:"guru_nama"`
	Bulan     string             `json:"bulan"`
	Status    string             `json:"status"`
	Kriteria  []TsiKriteriaNilai `json:"kriteria"`
	Kategori  []TsiKategoriSkor  `json:"kategori"`
	Total     float64            `json:"total"`
	Predikat  string             `json:"predikat"`
}

type TsiNilaiInput struct {
	KriteriaID     int64    `json:"kriteria_id"`
	Nilai          *float64 `json:"nilai"`
	IsOverride     bool     `json:"is_override"`
	AlasanOverride string   `json:"alasan_override"`
	Catatan        string   `json:"catatan"`
}

type TsiSaveRequest struct {
	Nilai []TsiNilaiInput `json:"nilai"`
}

type TsiRekapRow struct {
	GuruID       int64   `json:"guru_id"`
	GuruNama     string  `json:"guru_nama"`
	GuruStatus   string  `json:"guru_status"`
	Bulan        string  `json:"bulan"`
	Status       string  `json:"status"`
	Kompetensi   float64 `json:"kompetensi"`
	Kepuasan     float64 `json:"kepuasan"`
	Kedisiplinan float64 `json:"kedisiplinan"`
	Kontribusi   float64 `json:"kontribusi"`
	Total        float64 `json:"total"`
	Predikat     string  `json:"predikat"`
}
