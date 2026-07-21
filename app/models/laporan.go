package models

type DashboardStats struct {
	TotalSantri        int64             `json:"total_santri"`
	SantriLengkap      int64             `json:"santri_lengkap"`
	SantriPerluLengkap int64             `json:"santri_perlu_lengkap"`
	SantriTidakLanjut  int64             `json:"santri_tidak_lanjut"`
	TotalKelas         int64             `json:"total_kelas"`
	TotalNominal       float64           `json:"total_nominal"`
	SantriLaki         int64             `json:"santri_laki"`
	SantriPerempuan    int64             `json:"santri_perempuan"`
	PerAngkatan        []AngkatanStat    `json:"per_angkatan"`
	PerLevel           []LevelStat       `json:"per_level"`
	PerTipe            []TipeStat        `json:"per_tipe"`
	NominalPerAngkatan []NominalAngkatan `json:"nominal_per_angkatan"`
}

type AngkatanStat struct {
	Angkatan string `json:"angkatan"`
	Total    int64  `json:"total"`
}

type LevelStat struct {
	Level string `json:"level"`
	Total int64  `json:"total"`
}

type TipeStat struct {
	Tipe  string `json:"tipe"`
	Total int64  `json:"total"`
}

type NominalAngkatan struct {
	Angkatan     string  `json:"angkatan"`
	TotalNominal float64 `json:"total_nominal"`
}

type LaporanKeuangan struct {
	PerAngkatan []NominalAngkatan `json:"per_angkatan"`
	TidakLanjut int64             `json:"tidak_lanjut"`
}
