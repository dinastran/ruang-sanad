package models

import "time"

// TasbihDzikir: dzikir bawaan atau custom milik user. Target = target efektif
// (override user bila ada, selain itu TargetDefault).
type TasbihDzikir struct {
	ID            int64  `json:"id"`
	Custom        bool   `json:"custom"`
	Arab          string `json:"arab"`
	Latin         string `json:"latin"`
	Arti          string `json:"arti"`
	Rujukan       string `json:"rujukan"`
	Target        int64  `json:"target"`
	TargetDefault int64  `json:"target_default"`
	// HanyaPaket: dzikir bawaan yang hanya dibaca lewat paket (tidak tampil di daftar satuan).
	HanyaPaket bool `json:"hanya_paket"`
}

type TasbihPaketItem struct {
	DzikirID int64 `json:"dzikir_id"`
	Target   int64 `json:"target"`
}

type TasbihPaket struct {
	ID        int64             `json:"id"`
	Slug      string            `json:"slug"`
	Nama      string            `json:"nama"`
	Deskripsi string            `json:"deskripsi"`
	Rujukan   string            `json:"rujukan"`
	Items     []TasbihPaketItem `json:"items"`
}

type TasbihDzikirInput struct {
	Arab   string `json:"arab"`
	Latin  string `json:"latin"`
	Arti   string `json:"arti"`
	Target int64  `json:"target"`
}

// TasbihBadge: definisi badge plus waktu diraih (nil = terkunci).
type TasbihBadge struct {
	Kode     string     `json:"kode"`
	Nama     string     `json:"nama"`
	Syarat   string     `json:"syarat"`
	Ikon     string     `json:"ikon"`
	DiraihAt *time.Time `json:"diraih_at"`
}

type TasbihHariIni struct {
	Tanggal string `json:"tanggal"`
	Total   int64  `json:"total"`
	Target  int64  `json:"target"`
}

// TasbihSesiInput: satu sesi hitungan yang dikirim klien saat sync.
type TasbihSesiInput struct {
	ID         string    `json:"id"`
	DzikirID   int64     `json:"dzikir_id"`
	PaketID    *int64    `json:"paket_id"`
	PaketRunID *string   `json:"paket_run_id"`
	Jumlah     int64     `json:"jumlah"`
	Target     int64     `json:"target"`
	MulaiAt    time.Time `json:"mulai_at"`
	SelesaiAt  time.Time `json:"selesai_at"`
}

type TasbihSesiDitolak struct {
	ID     string `json:"id"`
	Alasan string `json:"alasan"`
}

type TasbihSyncHasil struct {
	Diterima  []string            `json:"diterima"`
	Ditolak   []TasbihSesiDitolak `json:"ditolak"`
	BadgeBaru []TasbihBadge       `json:"badge_baru"`
	HariIni   TasbihHariIni       `json:"hari_ini"`
}

type TasbihLevel struct {
	Level         int   `json:"level"`
	Total         int64 `json:"total"`
	AmbangIni     int64 `json:"ambang_ini"`
	AmbangBerikut int64 `json:"ambang_berikut"` // 0 bila level maksimal
}

type TasbihStreak struct {
	Saat            int  `json:"saat"`
	Terpanjang      int  `json:"terpanjang"`
	HariIniTercapai bool `json:"hari_ini_tercapai"`
}

type TasbihTitikHari struct {
	Tanggal string `json:"tanggal"`
	Total   int64  `json:"total"`
}

type TasbihKomposisi struct {
	DzikirID int64  `json:"dzikir_id"`
	Latin    string `json:"latin"`
	Total    int64  `json:"total"`
}

type TasbihPaketPutaran struct {
	PaketID int64  `json:"paket_id"`
	Nama    string `json:"nama"`
	Selesai int64  `json:"selesai"`
}

type TasbihDashboard struct {
	Rentang             int                  `json:"rentang"`
	HariIni             TasbihHariIni        `json:"hari_ini"`
	Streak              TasbihStreak         `json:"streak"`
	Level               TasbihLevel          `json:"level"`
	RataRata            float64              `json:"rata_rata"`
	Harian              []TasbihTitikHari    `json:"harian"`
	TotalRentang        int64                `json:"total_rentang"`
	TotalRentangSebelum int64                `json:"total_rentang_sebelum"`
	Heatmap             []TasbihTitikHari    `json:"heatmap"`
	Komposisi           []TasbihKomposisi    `json:"komposisi"`
	PerJam              [24]int64            `json:"per_jam"`
	Paket               []TasbihPaketPutaran `json:"paket"`
	Sesi                TasbihSesiStat       `json:"sesi"`
	HariTerbaik         *TasbihTitikHari     `json:"hari_terbaik"` // nil bila rentang kosong
	RekorTahun          *TasbihTitikHari     `json:"rekor_tahun"`  // hari terbanyak 365 hari terakhir
	HariIniDzikir       []TasbihKomposisi    `json:"hari_ini_dzikir"`
}

// TasbihSesiStat: ringkasan sesi dalam rentang dashboard.
type TasbihSesiStat struct {
	Jumlah      int64   `json:"jumlah"`       // banyak sesi
	Tuntas      int64   `json:"tuntas"`       // sesi yang mencapai targetnya
	DurasiDetik int64   `json:"durasi_detik"` // total durasi (per sesi dibatasi, lihat HitungSesiStat)
	PerMenit    float64 `json:"per_menit"`    // hitungan per menit (0 bila durasi 0)
	RataPerSesi float64 `json:"rata_per_sesi"`
	Terbesar    int64   `json:"terbesar"` // hitungan terbanyak dalam satu sesi
}

type TasbihRiwayatItem struct {
	ID          string `json:"id"`
	Tanggal     string `json:"tanggal"`
	Jam         string `json:"jam"`
	DzikirLatin string `json:"dzikir_latin"`
	Jumlah      int64  `json:"jumlah"`
	Target      int64  `json:"target"`
	DurasiDetik int64  `json:"durasi_detik"`
	PaketNama   string `json:"paket_nama"`
}
