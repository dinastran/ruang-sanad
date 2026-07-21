package queries

import (
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
)

func (s Santri) ToResponse() models.SantriResponse {
	r := models.SantriResponse{
		ID:                    s.ID,
		IDMahasantri:          s.IDMahasantri,
		KelasKode:             s.KelasKode,
		Nama:                  s.Nama,
		JenisKelamin:          s.JenisKelamin,
		Nominal:               s.Nominal,
		Angkatan:              s.Angkatan,
		Domisili:              s.Domisili,
		Fu:                    s.Fu,
		HasilVn:               s.HasilVn,
		MasukGrup:             s.MasukGrup,
		Level:                 s.Level,
		Jadwal:                s.Jadwal,
		Guru:                  s.Guru,
		JadwalCatatan:         s.JadwalCatatan,
		InfaqTerakhir:         s.InfaqTerakhir,
		KeteranganTidakLanjut: s.KeteranganTidakLanjut,
		Tipe:                  s.Tipe,
		Frekuensi:             s.Frekuensi,
		IsLengkap:             s.IsLengkap == 1,
		Status:                s.Status,
		CreatedAt:             s.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:             s.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if s.TanggalDaftar.Valid {
		r.TanggalDaftar = s.TanggalDaftar.Time.Format(time.DateOnly)
	}
	if s.Usia.Valid {
		r.Usia = s.Usia.Int64
	}
	if s.TanggalVn.Valid {
		r.TanggalVn = s.TanggalVn.Time.Format(time.DateOnly)
	}
	if s.MulaiBelajar.Valid {
		r.MulaiBelajar = s.MulaiBelajar.Time.Format(time.DateOnly)
	}
	if s.Jumlah.Valid {
		r.Jumlah = s.Jumlah.Int64
	}
	if s.KelasID.Valid {
		r.KelasID = &s.KelasID.Int64
	}
	if s.CreatedBy.Valid {
		r.CreatedBy = &s.CreatedBy.Int64
	}
	return r
}

func (k Kela) ToResponse() models.KelasResponse {
	r := models.KelasResponse{
		ID:           k.ID,
		KunciKelas:   k.KunciKelas,
		Angkatan:     k.Angkatan,
		Tipe:         k.Tipe,
		JenisKelamin: k.JenisKelamin,
		Level:        k.Level,
		Frekuensi:    k.Frekuensi,
		Jadwal:       k.Jadwal,
		SubIndex:     k.SubIndex,
		NamaKelas:    k.NamaKelas,
		Kapasitas:    k.Kapasitas,
		JumlahSantri: k.JumlahSantri,
		IsAktif:      k.IsAktif == 1,
		CreatedAt:    k.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if k.GuruID.Valid {
		r.GuruID = &k.GuruID.Int64
	}
	return r
}
