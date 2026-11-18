package services

import (
	"context"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// TasbihLevelAmbang: total hitungan minimal untuk level index+1.
var TasbihLevelAmbang = []int64{0, 100, 500, 1500, 4000, 10000, 25000, 50000, 100000, 250000}

const (
	tasbihHeatmapHari    = 365
	tasbihKomposisiTop   = 6
	tasbihRiwayatPerHlm  = 20
	tasbihTanggalLayout  = "2006-01-02"
	tasbihTanggalAwalSql = "0000-00-00"
)

func HitungLevel(total int64) models.TasbihLevel {
	lv := models.TasbihLevel{Level: 1, Total: total}
	for i, a := range TasbihLevelAmbang {
		if total >= a {
			lv.Level = i + 1
			lv.AmbangIni = a
		}
	}
	if lv.Level < len(TasbihLevelAmbang) {
		lv.AmbangBerikut = TasbihLevelAmbang[lv.Level]
	}
	return lv
}

func tasbihTanggal(t time.Time) string { return t.In(wib).Format(tasbihTanggalLayout) }

func tasbihGeser(tanggal string, hari int) string {
	t, err := time.ParseInLocation(tasbihTanggalLayout, tanggal, wib)
	if err != nil {
		return tanggal
	}
	return t.AddDate(0, 0, hari).Format(tasbihTanggalLayout)
}

// HitungStreak: hariTercapai berisi tanggal (asc, unik) yang mencapai target.
// Streak saat ini dihitung dari hari ini bila tercapai, jika tidak dari kemarin.
func HitungStreak(hariTercapai []string, hariIni string) models.TasbihStreak {
	var s models.TasbihStreak
	set := make(map[string]bool, len(hariTercapai))
	run, prev := 0, ""
	for _, d := range hariTercapai {
		set[d] = true
		if prev != "" && tasbihGeser(prev, 1) == d {
			run++
		} else {
			run = 1
		}
		if run > s.Terpanjang {
			s.Terpanjang = run
		}
		prev = d
	}
	s.HariIniTercapai = set[hariIni]
	d := hariIni
	if !s.HariIniTercapai {
		d = tasbihGeser(hariIni, -1)
	}
	for set[d] {
		s.Saat++
		d = tasbihGeser(d, -1)
	}
	return s
}

func (s *TasbihService) HariIni(ctx context.Context, userID int64, now time.Time) (models.TasbihHariIni, error) {
	hari := tasbihTanggal(now)
	total, err := s.querier.SumTasbihRentang(ctx, queries.SumTasbihRentangParams{UserID: userID, Dari: hari, Sampai: hari})
	if err != nil {
		return models.TasbihHariIni{}, err
	}
	target, err := s.TargetHarian(ctx, userID)
	if err != nil {
		return models.TasbihHariIni{}, err
	}
	return models.TasbihHariIni{Tanggal: hari, Total: total, Target: target}, nil
}

// Streak menghitung streak berdasarkan target harian saat ini.
func (s *TasbihService) Streak(ctx context.Context, userID int64, now time.Time) (models.TasbihStreak, error) {
	target, err := s.TargetHarian(ctx, userID)
	if err != nil {
		return models.TasbihStreak{}, err
	}
	hari, err := s.querier.ListTasbihHariTercapai(ctx, queries.ListTasbihHariTercapaiParams{UserID: userID, Target: target})
	if err != nil {
		return models.TasbihStreak{}, err
	}
	return HitungStreak(hari, tasbihTanggal(now)), nil
}

func (s *TasbihService) Level(ctx context.Context, userID int64) (models.TasbihLevel, error) {
	total, err := s.querier.SumTasbihRentang(ctx, queries.SumTasbihRentangParams{UserID: userID, Dari: tasbihTanggalAwalSql, Sampai: "9999-99-99"})
	if err != nil {
		return models.TasbihLevel{}, err
	}
	return HitungLevel(total), nil
}

func normalisasiRentang(r int) int {
	switch r {
	case 7, 30, 90, 365:
		return r
	}
	return 30
}

func (s *TasbihService) deretHarian(ctx context.Context, userID int64, sampai string, hari int) ([]models.TasbihTitikHari, int64, error) {
	dari := tasbihGeser(sampai, -(hari - 1))
	rows, err := s.querier.ListTasbihTotalHarian(ctx, queries.ListTasbihTotalHarianParams{UserID: userID, Dari: dari, Sampai: sampai})
	if err != nil {
		return nil, 0, err
	}
	byTgl := make(map[string]int64, len(rows))
	for _, r := range rows {
		byTgl[r.TanggalWib] = r.Total
	}
	out := make([]models.TasbihTitikHari, hari)
	var total int64
	for i := 0; i < hari; i++ {
		tgl := tasbihGeser(dari, i)
		out[i] = models.TasbihTitikHari{Tanggal: tgl, Total: byTgl[tgl]}
		total += byTgl[tgl]
	}
	return out, total, nil
}

func (s *TasbihService) Dashboard(ctx context.Context, userID int64, rentang int, now time.Time) (models.TasbihDashboard, error) {
	rentang = normalisasiRentang(rentang)
	d := models.TasbihDashboard{Rentang: rentang}
	hariIni := tasbihTanggal(now)
	dari := tasbihGeser(hariIni, -(rentang - 1))
	var err error

	if d.HariIni, err = s.HariIni(ctx, userID, now); err != nil {
		return d, err
	}
	if d.Streak, err = s.Streak(ctx, userID, now); err != nil {
		return d, err
	}
	if d.Level, err = s.Level(ctx, userID); err != nil {
		return d, err
	}
	if d.Harian, d.TotalRentang, err = s.deretHarian(ctx, userID, hariIni, rentang); err != nil {
		return d, err
	}
	d.RataRata = float64(d.TotalRentang) / float64(rentang)
	if d.TotalRentangSebelum, err = s.querier.SumTasbihRentang(ctx, queries.SumTasbihRentangParams{
		UserID: userID, Dari: tasbihGeser(dari, -rentang), Sampai: tasbihGeser(dari, -1),
	}); err != nil {
		return d, err
	}
	if d.Heatmap, _, err = s.deretHarian(ctx, userID, hariIni, tasbihHeatmapHari); err != nil {
		return d, err
	}

	jam, err := s.querier.ListTasbihPerJam(ctx, queries.ListTasbihPerJamParams{UserID: userID, Dari: dari, Sampai: hariIni})
	if err != nil {
		return d, err
	}
	for _, j := range jam {
		if j.JamWib >= 0 && j.JamWib < 24 {
			d.PerJam[j.JamWib] = j.Total
		}
	}

	komp, err := s.querier.ListTasbihKomposisi(ctx, queries.ListTasbihKomposisiParams{UserID: userID, Dari: dari, Sampai: hariIni})
	if err != nil {
		return d, err
	}
	d.Komposisi = []models.TasbihKomposisi{}
	var lainnya int64
	for i, k := range komp {
		if i < tasbihKomposisiTop {
			d.Komposisi = append(d.Komposisi, models.TasbihKomposisi{DzikirID: k.DzikirID, Latin: k.Latin, Total: k.Total})
		} else {
			lainnya += k.Total
		}
	}
	if lainnya > 0 {
		d.Komposisi = append(d.Komposisi, models.TasbihKomposisi{Latin: "Lainnya", Total: lainnya})
	}

	paket, err := s.querier.ListTasbihPutaranPaket(ctx, queries.ListTasbihPutaranPaketParams{UserID: userID, Dari: dari})
	if err != nil {
		return d, err
	}
	d.Paket = make([]models.TasbihPaketPutaran, 0, len(paket))
	for _, p := range paket {
		d.Paket = append(d.Paket, models.TasbihPaketPutaran{PaketID: p.ID, Nama: p.Nama, Selesai: p.Selesai})
	}

	sesi, err := s.querier.ListTasbihSesiRingkas(ctx, queries.ListTasbihSesiRingkasParams{UserID: userID, Dari: dari, Sampai: hariIni})
	if err != nil {
		return d, err
	}
	d.Sesi = HitungSesiStat(sesi)
	d.HariTerbaik = hariTerbanyak(d.Harian)
	d.RekorTahun = hariTerbanyak(d.Heatmap)
	if d.HariIniDzikir, err = s.KomposisiHariIni(ctx, userID, now); err != nil {
		return d, err
	}
	return d, nil
}

// tasbihDurasiMaksSesi membatasi durasi satu sesi agar sesi yang lupa ditutup
// (aplikasi dibiarkan terbuka) tidak merusak total durasi dan kecepatan.
const tasbihDurasiMaksSesi = 2 * time.Hour

func HitungSesiStat(rows []queries.ListTasbihSesiRingkasRow) models.TasbihSesiStat {
	var st models.TasbihSesiStat
	var total int64
	var dur time.Duration
	for _, r := range rows {
		st.Jumlah++
		total += r.Jumlah
		if r.SelesaiTarget == 1 {
			st.Tuntas++
		}
		if r.Jumlah > st.Terbesar {
			st.Terbesar = r.Jumlah
		}
		if dd := r.SelesaiAt.Sub(r.MulaiAt); dd > 0 {
			dur += min(dd, tasbihDurasiMaksSesi)
		}
	}
	st.DurasiDetik = int64(dur.Seconds())
	if st.Jumlah > 0 {
		st.RataPerSesi = float64(total) / float64(st.Jumlah)
	}
	if dur >= time.Minute {
		st.PerMenit = float64(total) / dur.Minutes()
	}
	return st
}

// hariTerbanyak: hari dengan total terbesar (tanggal terbaru bila seri); nil bila semua nol.
func hariTerbanyak(deret []models.TasbihTitikHari) *models.TasbihTitikHari {
	var best *models.TasbihTitikHari
	for i := range deret {
		if deret[i].Total > 0 && (best == nil || deret[i].Total >= best.Total) {
			h := deret[i]
			best = &h
		}
	}
	return best
}

// JumlahPutaranPaket: putaran paket selesai sejak tanggal (nil = sepanjang masa),
// untuk satu paket (paketID) atau semua paket (nil).
func (s *TasbihService) JumlahPutaranPaket(ctx context.Context, userID int64, paketID *int64, sejak *string) (int64, error) {
	dari := tasbihTanggalAwalSql
	if sejak != nil {
		dari = *sejak
	}
	rows, err := s.querier.ListTasbihPutaranPaket(ctx, queries.ListTasbihPutaranPaketParams{UserID: userID, Dari: dari})
	if err != nil {
		return 0, err
	}
	var n int64
	for _, r := range rows {
		if paketID == nil || r.ID == *paketID {
			n += r.Selesai
		}
	}
	return n, nil
}

func (s *TasbihService) Riwayat(ctx context.Context, userID int64, halaman int, dzikirID *int64) ([]models.TasbihRiwayatItem, bool, error) {
	if halaman < 1 {
		halaman = 1
	}
	var filter interface{}
	if dzikirID != nil {
		filter = *dzikirID
	}
	rows, err := s.querier.ListTasbihRiwayat(ctx, queries.ListTasbihRiwayatParams{
		UserID: userID, DzikirID: filter, Lim: tasbihRiwayatPerHlm + 1, Off: int64((halaman - 1) * tasbihRiwayatPerHlm),
	})
	if err != nil {
		return nil, false, err
	}
	adaLagi := len(rows) > tasbihRiwayatPerHlm
	if adaLagi {
		rows = rows[:tasbihRiwayatPerHlm]
	}
	out := make([]models.TasbihRiwayatItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.TasbihRiwayatItem{
			ID: r.ID, Tanggal: r.TanggalWib, Jam: r.MulaiAt.In(wib).Format("15:04"), DzikirLatin: r.Latin,
			Jumlah: r.Jumlah, Target: r.Target, DurasiDetik: int64(r.SelesaiAt.Sub(r.MulaiAt).Seconds()), PaketNama: r.PaketNama,
		})
	}
	return out, adaLagi, nil
}

// KomposisiHariIni: total per dzikir hari ini (WIB), semua dzikir, terbanyak dulu.
func (s *TasbihService) KomposisiHariIni(ctx context.Context, userID int64, now time.Time) ([]models.TasbihKomposisi, error) {
	hari := tasbihTanggal(now)
	rows, err := s.querier.ListTasbihKomposisi(ctx, queries.ListTasbihKomposisiParams{UserID: userID, Dari: hari, Sampai: hari})
	if err != nil {
		return nil, err
	}
	out := make([]models.TasbihKomposisi, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.TasbihKomposisi{DzikirID: r.DzikirID, Latin: r.Latin, Total: r.Total})
	}
	return out, nil
}
