package services

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
)

const (
	// alfaBeruntunAmbang menandai guru merah: alfa minimal sekian sesi terakhir berturut-turut.
	alfaBeruntunAmbang = 3
	// alfaBeruntunWaspada adalah peringatan dini sebelum ambang tercapai.
	alfaBeruntunWaspada = 2
	// analisisStripSesi adalah jumlah sesi terlaksana terakhir yang ditampilkan per guru.
	analisisStripSesi = 6
)

// kehadiranSesi adalah satu catatan absen guru pada sesi terlaksana.
type kehadiranSesi struct {
	GuruID    int64
	SesiID    int64
	Tanggal   time.Time
	Kehadiran string
}

// hitungAlfaBeruntun menerima status kehadiran terbaru lebih dulu. Hanya sesi
// yang punya catatan absen yang masuk ke sini, jadi sesi tanpa catatan tidak
// memutus maupun menambah hitungan.
func hitungAlfaBeruntun(statusTerbaru []string) int {
	n := 0
	for _, s := range statusTerbaru {
		if s != models.KehadiranAlfa {
			break
		}
		n++
	}
	return n
}

func levelAlfaBeruntun(n int) string {
	switch {
	case n >= alfaBeruntunAmbang:
		return "merah"
	case n >= alfaBeruntunWaspada:
		return "oranye"
	default:
		return ""
	}
}

// AnalisisAlfaPembinaan menganalisis alfa berturut-turut guru aktif pada pembinaan terlaksana.
func (s *KoordinatorService) AnalisisAlfaPembinaan() (*models.AnalisisKehadiran, error) {
	ctx := context.Background()
	sesiRows, err := s.querier.ListPembinaan(ctx)
	if err != nil {
		return nil, err
	}
	sesi := make([]models.AnalisisKehadiranSesi, 0, analisisStripSesi)
	for _, p := range sesiRows {
		if p.Status == "terlaksana" && len(sesi) < analisisStripSesi {
			sesi = append(sesi, models.AnalisisKehadiranSesi{ID: p.ID, Tanggal: p.Tanggal.Format("2006-01-02"), Judul: p.Topik})
		}
	}
	rows, err := s.querier.ListKehadiranPembinaanTerlaksana(ctx)
	if err != nil {
		return nil, err
	}
	catatan := make([]kehadiranSesi, 0, len(rows))
	for _, r := range rows {
		catatan = append(catatan, kehadiranSesi{GuruID: r.GuruID, SesiID: r.KegiatanID, Tanggal: r.Tanggal, Kehadiran: r.Kehadiran})
	}
	return s.susunAnalisisKehadiran(sesi, catatan)
}

// AnalisisAlfaRapat menganalisis alfa berturut-turut guru aktif pada rapat terlaksana.
func (s *KoordinatorService) AnalisisAlfaRapat() (*models.AnalisisKehadiran, error) {
	ctx := context.Background()
	sesiRows, err := s.querier.ListRapat(ctx)
	if err != nil {
		return nil, err
	}
	sesi := make([]models.AnalisisKehadiranSesi, 0, analisisStripSesi)
	for _, r := range sesiRows {
		if r.Status == "terlaksana" && len(sesi) < analisisStripSesi {
			sesi = append(sesi, models.AnalisisKehadiranSesi{ID: r.ID, Tanggal: r.Tanggal.Format("2006-01-02"), Judul: r.Judul})
		}
	}
	rows, err := s.querier.ListKehadiranRapatTerlaksana(ctx)
	if err != nil {
		return nil, err
	}
	catatan := make([]kehadiranSesi, 0, len(rows))
	for _, r := range rows {
		catatan = append(catatan, kehadiranSesi{GuruID: r.GuruID, SesiID: r.KegiatanID, Tanggal: r.Tanggal, Kehadiran: r.Kehadiran})
	}
	return s.susunAnalisisKehadiran(sesi, catatan)
}

// RingkasanAlfaBeruntun merangkum jumlah guru merah/waspada di pembinaan dan rapat.
func (s *KoordinatorService) RingkasanAlfaBeruntun() (*models.RingkasanAlfaBeruntun, error) {
	pembinaan, err := s.AnalisisAlfaPembinaan()
	if err != nil {
		return nil, err
	}
	rapat, err := s.AnalisisAlfaRapat()
	if err != nil {
		return nil, err
	}
	return &models.RingkasanAlfaBeruntun{
		AmbangAlfa:       alfaBeruntunAmbang,
		PembinaanMerah:   pembinaan.Merah,
		PembinaanWaspada: pembinaan.Waspada,
		RapatMerah:       rapat.Merah,
		RapatWaspada:     rapat.Waspada,
	}, nil
}

// susunAnalisisKehadiran: sesi = sesi terlaksana terbaru dulu (maks. strip),
// catatan = absen guru aktif di sesi terlaksana, urut per guru lalu terbaru dulu.
func (s *KoordinatorService) susunAnalisisKehadiran(sesi []models.AnalisisKehadiranSesi, catatan []kehadiranSesi) (*models.AnalisisKehadiran, error) {
	guruRows, err := s.querier.ListGuruAktifSimple(context.Background())
	if err != nil {
		return nil, err
	}

	perGuru := make(map[int64][]kehadiranSesi, len(guruRows))
	for _, c := range catatan {
		perGuru[c.GuruID] = append(perGuru[c.GuruID], c)
	}
	posisiSesi := make(map[int64]int, len(sesi))
	for i, ss := range sesi {
		posisiSesi[ss.ID] = i
	}

	out := &models.AnalisisKehadiran{
		AmbangAlfa:    alfaBeruntunAmbang,
		AmbangWaspada: alfaBeruntunWaspada,
		Sesi:          sesi,
		Guru:          make([]models.AnalisisKehadiranGuru, 0, len(guruRows)),
	}
	var totalHadir, totalDicatat int
	for _, g := range guruRows {
		list := perGuru[g.ID]
		status := make([]string, len(list))
		for i, c := range list {
			status[i] = c.Kehadiran
		}
		item := models.AnalisisKehadiranGuru{
			GuruID:       g.ID,
			Nama:         g.Nama,
			NoWa:         g.NoWa,
			AlfaBeruntun: hitungAlfaBeruntun(status),
			Riwayat:      make([]string, len(sesi)),
		}
		item.Level = levelAlfaBeruntun(item.AlfaBeruntun)
		for _, c := range list {
			if c.Kehadiran == models.KehadiranHadir && item.TerakhirHadir == "" {
				item.TerakhirHadir = c.Tanggal.Format("2006-01-02")
			}
			if i, ok := posisiSesi[c.SesiID]; ok {
				item.Riwayat[i] = c.Kehadiran
				item.DicatatTerakhir++
				if c.Kehadiran == models.KehadiranHadir {
					item.HadirTerakhir++
				}
			}
		}
		totalHadir += item.HadirTerakhir
		totalDicatat += item.DicatatTerakhir
		switch item.Level {
		case "merah":
			out.Merah++
		case "oranye":
			out.Waspada++
		}
		out.Guru = append(out.Guru, item)
	}
	if totalDicatat > 0 {
		persen := float64(totalHadir) / float64(totalDicatat) * 100
		out.PersenHadir = &persen
	}

	sort.SliceStable(out.Guru, func(i, j int) bool {
		if out.Guru[i].AlfaBeruntun != out.Guru[j].AlfaBeruntun {
			return out.Guru[i].AlfaBeruntun > out.Guru[j].AlfaBeruntun
		}
		return strings.ToLower(out.Guru[i].Nama) < strings.ToLower(out.Guru[j].Nama)
	})
	return out, nil
}
