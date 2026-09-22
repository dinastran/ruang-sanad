package queries

import (
	"database/sql"
	"fmt"
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
		AngkatanKelas:         s.AngkatanKelas,
		Domisili:              s.Domisili,
		NoWA:                  s.NoWa,
		Email:                 s.Email,
		Fu:                    s.Fu,
		HasilVn:               s.HasilVn,
		KeteranganVn:          s.KeteranganVn,
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
		StatusAlasan:          s.StatusAlasan,
		CutiMulai:             s.CutiMulai,
		CutiSelesai:           s.CutiSelesai,
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
	if s.VoiceNoteUrl != "" {
		r.VoiceNoteURL = fmt.Sprintf("/app/santri/%d/voice-note/audio", s.ID)
	}
	return r
}

func (k Kela) ToResponse() models.KelasResponse {
	return kelasResponse(k.ID, k.KunciKelas, k.Angkatan, k.Tipe, k.JenisKelamin, k.Level, k.Frekuensi, k.Jadwal, k.SubIndex, k.NamaKelas, k.GuruID, k.Kapasitas, k.JumlahSantri, k.PertemuanTerakhir, k.MateriIndividual, k.IsAktif, k.CreatedAt)
}

func (k ListKelasByAngkatanRow) ToResponse() models.KelasResponse {
	return kelasResponse(k.ID, k.KunciKelas, k.Angkatan, k.Tipe, k.JenisKelamin, k.Level, k.Frekuensi, k.Jadwal, k.SubIndex, k.NamaKelas, k.GuruID, k.Kapasitas, k.JumlahSantri, k.PertemuanTerakhir, k.MateriIndividual, k.IsAktif, k.CreatedAt)
}

func (k ListKelasAllRow) ToResponse() models.KelasResponse {
	return kelasResponse(k.ID, k.KunciKelas, k.Angkatan, k.Tipe, k.JenisKelamin, k.Level, k.Frekuensi, k.Jadwal, k.SubIndex, k.NamaKelas, k.GuruID, k.Kapasitas, k.JumlahSantri, k.PertemuanTerakhir, k.MateriIndividual, k.IsAktif, k.CreatedAt)
}

func (k GetKelasByIDRow) ToResponse() models.KelasResponse {
	return kelasResponse(k.ID, k.KunciKelas, k.Angkatan, k.Tipe, k.JenisKelamin, k.Level, k.Frekuensi, k.Jadwal, k.SubIndex, k.NamaKelas, k.GuruID, k.Kapasitas, k.JumlahSantri, k.PertemuanTerakhir, k.MateriIndividual, k.IsAktif, k.CreatedAt)
}

func kelasResponse(id int64, kunciKelas, angkatan, tipe, jenisKelamin, level, frekuensi, jadwal string, subIndex int64, namaKelas string, guruID sql.NullInt64, kapasitas, jumlahSantri, pertemuanTerakhir, materiIndividual, isAktif int64, createdAt time.Time) models.KelasResponse {
	r := models.KelasResponse{
		ID:                id,
		KunciKelas:        kunciKelas,
		Angkatan:          angkatan,
		Tipe:              tipe,
		JenisKelamin:      jenisKelamin,
		Level:             level,
		Frekuensi:         frekuensi,
		Jadwal:            jadwal,
		SubIndex:          subIndex,
		NamaKelas:         namaKelas,
		Kapasitas:         kapasitas,
		JumlahSantri:      jumlahSantri,
		PertemuanTerakhir: pertemuanTerakhir,
		MateriIndividual:  materiIndividual == 1,
		IsAktif:           isAktif == 1,
		CreatedAt:         createdAt.Format("2006-01-02 15:04:05"),
	}
	if guruID.Valid {
		r.GuruID = &guruID.Int64
	}
	return r
}
