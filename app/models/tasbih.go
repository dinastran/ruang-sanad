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
