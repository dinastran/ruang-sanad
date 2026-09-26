package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// Rentang hari ke belakang untuk mencari pertemuan yang belum diabsen.
const berandaTertundaHari = 7

// GuruBerandaService menyusun dashboard guru: jadwal hari ini, absensi
// tertunda, kinerja pribadi, dan agenda koordinator.
type GuruBerandaService struct {
	querier *queries.Querier
	tsi     *TSIService
}

func NewGuruBerandaService(querier *queries.Querier, tsi *TSIService) *GuruBerandaService {
	return &GuruBerandaService{querier: querier, tsi: tsi}
}

var (
	hariJadwal = map[string]time.Weekday{
		"senin": time.Monday, "selasa": time.Tuesday, "rabu": time.Wednesday,
		"kamis": time.Thursday, "jumat": time.Friday, "jum'at": time.Friday,
		"sabtu": time.Saturday, "ahad": time.Sunday, "minggu": time.Sunday,
	}
	reKataJadwal = regexp.MustCompile(`[a-z']+`)
	reJamJadwal  = regexp.MustCompile(`(\d{1,2})[.:](\d{2})`)
)

// parseJadwalKelas membaca teks jadwal rutin seperti "Senin, jam 20.30 WIB"
// atau "Senin & Kamis, jam 19.00 WIB". Hari kosong berarti tidak terbaca.
func parseJadwalKelas(jadwal string) (map[time.Weekday]bool, string) {
	lower := strings.ToLower(jadwal)
	hari := map[time.Weekday]bool{}
	for _, kata := range reKataJadwal.FindAllString(lower, -1) {
		if d, ok := hariJadwal[kata]; ok {
			hari[d] = true
		}
	}
	jam := ""
	if m := reJamJadwal.FindStringSubmatch(lower); m != nil {
		h, _ := strconv.Atoi(m[1])
		jam = fmt.Sprintf("%02d:%s", h, m[2])
	}
	return hari, jam
}

type slotKey struct {
	kelasID int64
	tanggal string
}

// GetBeranda menyusun dashboard. guruID nil = semua kelas (admin), tanpa
// bagian kinerja pribadi.
func (s *GuruBerandaService) GetBeranda(guruID *int64, today time.Time) (*models.GuruBeranda, error) {
	ctx := context.Background()
	scope := guruScope(guruID)
	awalHariIni := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	hariIni := awalHariIni.Format("2006-01-02")
	mulai := awalHariIni.AddDate(0, 0, -berandaTertundaHari)
	besok := awalHariIni.AddDate(0, 0, 1).Format("2006-01-02")

	beranda := &models.GuruBeranda{
		Tanggal:          hariIni,
		HariIni:          []models.GuruSlotKelas{},
		Tertunda:         []models.GuruSlotKelas{},
		JadwalTakTerbaca: []models.GuruKelasJadwalTakTerbaca{},
		Agenda:           []models.GuruAgendaItem{},
	}

	kelasRows, err := s.querier.ListKelasAktifBeranda(ctx, scope)
	if err != nil {
		return nil, err
	}
	slots := map[slotKey]*models.GuruSlotKelas{}
	for _, k := range kelasRows {
		if k.JumlahSantri == 0 {
			continue // kelas tanpa santri aktif belum/tidak berjalan
		}
		hari, jam := parseJadwalKelas(k.Jadwal)
		if len(hari) == 0 {
			beranda.JadwalTakTerbaca = append(beranda.JadwalTakTerbaca, models.GuruKelasJadwalTakTerbaca{KelasID: k.ID, NamaKelas: k.NamaKelas, Jadwal: k.Jadwal})
			continue
		}
		dibuat := k.CreatedAt.In(today.Location()).Format("2006-01-02")
		for d := mulai; !d.After(awalHariIni); d = d.AddDate(0, 0, 1) {
			tanggal := d.Format("2006-01-02")
			if !hari[d.Weekday()] || tanggal < dibuat {
				continue
			}
			slots[slotKey{k.ID, tanggal}] = &models.GuruSlotKelas{
				KelasID:      k.ID,
				NamaKelas:    k.NamaKelas,
				Level:        k.Level,
				Tanggal:      tanggal,
				Jam:          jam,
				JumlahSantri: k.JumlahSantri,
				Sumber:       "rutin",
				Status:       models.SlotBelum,
			}
		}
	}

	// Jadwal Pertemuan menimpa jadwal rutin: reschedule memindahkan slot,
	// batal menghapusnya, badal mengubah siapa yang mengajar.
	jadwalRows, err := s.querier.ListJadwalPertemuanBeranda(ctx, queries.ListJadwalPertemuanBerandaParams{
		Mulai: mulai.Format("2006-01-02"), Selesai: besok, GuruID: scope,
	})
	if err != nil {
		return nil, err
	}
	mulaiStr := mulai.Format("2006-01-02")
	dalamRentang := func(t string) bool { return t >= mulaiStr && t < besok }
	jadwalSlots := map[slotKey]*models.GuruSlotKelas{}
	for _, j := range jadwalRows {
		tanggal := j.Tanggal.Format("2006-01-02")
		semula := ""
		if len(j.JadwalSemula) >= 10 {
			semula = j.JadwalSemula[:10]
		}
		if semula != "" && semula != tanggal && dalamRentang(semula) {
			delete(slots, slotKey{j.KelasID, semula})
		}
		if j.Status == "dibatalkan" {
			delete(slots, slotKey{j.KelasID, tanggal})
			continue
		}
		if !dalamRentang(tanggal) {
			continue
		}
		slot := &models.GuruSlotKelas{
			KelasID:      j.KelasID,
			NamaKelas:    j.NamaKelas,
			Level:        j.Level,
			Tanggal:      tanggal,
			Jam:          j.JamMulai,
			JumlahSantri: j.JumlahSantri,
			Sumber:       "jadwal",
			JadwalID:     j.ID,
			JadwalStatus: j.Status,
			Status:       models.SlotBelum,
			PertemuanID:  j.PertemuanID,
		}
		if j.IsReschedule == 1 && semula != "" {
			slot.Keterangan = "Dipindah dari " + semula
		}
		if j.GuruPenggantiID != 0 {
			if guruID != nil && j.GuruPenggantiID == *guruID && j.GuruUtamaID != *guruID {
				slot.SebagaiBadal = true
				slot.Keterangan = strings.TrimSpace("Menggantikan " + j.GuruUtamaNama)
			} else if guruID == nil || j.GuruPenggantiID != *guruID {
				slot.Status = models.SlotDibadalkan
				slot.Keterangan = strings.TrimSpace("Dibadalkan oleh " + j.GuruPenggantiNama)
			}
		}
		jadwalSlots[slotKey{j.KelasID, tanggal}] = slot
	}
	for k, v := range jadwalSlots {
		slots[k] = v
	}

	if err := s.isiStatusPertemuan(ctx, slots, mulai, besok); err != nil {
		return nil, err
	}

	for _, slot := range slots {
		switch {
		case slot.Tanggal == hariIni:
			beranda.HariIni = append(beranda.HariIni, *slot)
		case slot.Status == models.SlotBelum:
			beranda.Tertunda = append(beranda.Tertunda, *slot)
		}
	}
	sort.Slice(beranda.HariIni, func(i, j int) bool {
		a, b := beranda.HariIni[i], beranda.HariIni[j]
		if a.Jam != b.Jam {
			return a.Jam < b.Jam
		}
		return a.NamaKelas < b.NamaKelas
	})
	sort.Slice(beranda.Tertunda, func(i, j int) bool {
		a, b := beranda.Tertunda[i], beranda.Tertunda[j]
		if a.Tanggal != b.Tanggal {
			return a.Tanggal < b.Tanggal
		}
		return a.NamaKelas < b.NamaKelas
	})

	agenda, err := s.querier.ListAgendaGuruMendatang(ctx, hariIni)
	if err != nil {
		return nil, err
	}
	for _, a := range agenda {
		tanggal := a.Tanggal
		if len(tanggal) > 10 {
			tanggal = tanggal[:10]
		}
		beranda.Agenda = append(beranda.Agenda, models.GuruAgendaItem{Jenis: a.Jenis, Tanggal: tanggal, Judul: a.Judul})
	}

	if guruID != nil {
		kinerja, err := s.kinerja(ctx, *guruID, awalHariIni)
		if err != nil {
			return nil, err
		}
		beranda.Kinerja = kinerja
	}
	return beranda, nil
}

// isiStatusPertemuan menandai slot sebagai berlangsung/selesai bila ada
// pertemuan tercatat. Satu slot dianggap terpenuhi oleh pertemuan antara
// sehari sebelum slot (dimajukan) sampai sebelum slot berikutnya kelas itu.
func (s *GuruBerandaService) isiStatusPertemuan(ctx context.Context, slots map[slotKey]*models.GuruSlotKelas, mulai time.Time, besok string) error {
	perKelas := map[int64][]*models.GuruSlotKelas{}
	for _, slot := range slots {
		perKelas[slot.KelasID] = append(perKelas[slot.KelasID], slot)
	}
	if len(perKelas) == 0 {
		return nil
	}
	kelasIDs := make([]int64, 0, len(perKelas))
	for id := range perKelas {
		kelasIDs = append(kelasIDs, id)
	}
	rows, err := s.querier.ListPertemuanKelasSejak(ctx, queries.ListPertemuanKelasSejakParams{
		KelasIds: kelasIDs,
		Sejak:    mulai.AddDate(0, 0, -1).Format("2006-01-02"),
	})
	if err != nil {
		return err
	}
	pertemuanByID := map[int64]queries.ListPertemuanKelasSejakRow{}
	pertemuanKelas := map[int64][]queries.ListPertemuanKelasSejakRow{}
	for _, p := range rows {
		pertemuanByID[p.ID] = p
		pertemuanKelas[p.KelasID] = append(pertemuanKelas[p.KelasID], p)
	}

	for kelasID, list := range perKelas {
		sort.Slice(list, func(i, j int) bool { return list[i].Tanggal < list[j].Tanggal })
		for i, slot := range list {
			if slot.Status == models.SlotDibadalkan {
				continue
			}
			if slot.PertemuanID != 0 {
				if p, ok := pertemuanByID[slot.PertemuanID]; ok {
					slot.Status = statusSlotDariPertemuan(p.Status)
				}
				continue
			}
			awal := geserTanggal(slot.Tanggal, -1)
			if i > 0 && list[i-1].Tanggal >= awal {
				awal = geserTanggal(list[i-1].Tanggal, 1)
			}
			akhir := besok
			if i+1 < len(list) {
				akhir = list[i+1].Tanggal
			}
			for _, p := range pertemuanKelas[kelasID] {
				tanggal := p.Tanggal.Format("2006-01-02")
				if tanggal < awal || tanggal >= akhir {
					continue
				}
				slot.PertemuanID = p.ID
				slot.Status = statusSlotDariPertemuan(p.Status)
				if slot.Status == models.SlotBerlangsung {
					break
				}
			}
		}
	}
	return nil
}

func statusSlotDariPertemuan(status string) string {
	if status == "berlangsung" {
		return models.SlotBerlangsung
	}
	return models.SlotSelesai
}

func geserTanggal(tanggal string, hari int) string {
	t, err := time.Parse("2006-01-02", tanggal)
	if err != nil {
		return tanggal
	}
	return t.AddDate(0, 0, hari).Format("2006-01-02")
}

func (s *GuruBerandaService) kinerja(ctx context.Context, guruID int64, hariIni time.Time) (*models.GuruKinerja, error) {
	// tilawah_harian menyimpan tanggal sebagai tengah malam UTC.
	tgl := func(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }
	hari := tgl(hariIni)
	awalBulan := time.Date(hari.Year(), hari.Month(), 1, 0, 0, 0, 0, time.UTC)
	akhirBulan := awalBulan.AddDate(0, 1, -1)

	k := &models.GuruKinerja{}
	tilawah, err := s.querier.ListTilawahByGuruRange(ctx, queries.ListTilawahByGuruRangeParams{GuruID: guruID, Tanggal: hari.AddDate(0, 0, -90), Tanggal_2: hari})
	if err != nil {
		return nil, err
	}
	sudah := map[string]bool{}
	for _, t := range tilawah {
		d := t.Format("2006-01-02")
		sudah[d] = true
		if !t.Before(awalBulan) {
			k.TilawahBulanIni++
		}
	}
	k.SudahTilawah = sudah[hari.Format("2006-01-02")]
	// Streak tetap hidup bila hari ini belum check-in.
	d := hari
	if !k.SudahTilawah {
		d = d.AddDate(0, 0, -1)
	}
	for sudah[d.Format("2006-01-02")] {
		k.TilawahStreak++
		d = d.AddDate(0, 0, -1)
	}

	if bulan, err := s.querier.GetTsiFinalTerakhir(ctx, guruID); err == nil {
		if s.tsi != nil {
			if p, err := s.tsi.GetPenilaian(guruID, "", bulan); err == nil {
				k.TsiBulan = bulan
				total := p.Total
				k.TsiTotal = &total
				k.TsiPredikat = p.Predikat
			}
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if k.PembinaanTotal, err = s.querier.CountPembinaanTerlaksana(ctx, queries.CountPembinaanTerlaksanaParams{Tanggal: awalBulan, Tanggal_2: akhirBulan}); err != nil {
		return nil, err
	}
	if k.PembinaanHadir, err = s.querier.CountPembinaanHadirByGuru(ctx, queries.CountPembinaanHadirByGuruParams{GuruID: guruID, Tanggal: awalBulan, Tanggal_2: akhirBulan}); err != nil {
		return nil, err
	}
	if k.RapatHadir, err = s.querier.CountRapatHadirByGuru(ctx, queries.CountRapatHadirByGuruParams{GuruID: guruID, Tanggal: awalBulan, Tanggal_2: akhirBulan}); err != nil {
		return nil, err
	}
	return k, nil
}
