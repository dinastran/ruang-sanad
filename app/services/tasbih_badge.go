package services

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

type TasbihBadgeDef struct {
	Kode, Nama, Syarat, Ikon string // Ikon = nama ikon lucide
}

// TasbihBadges: urutan tampil. Copy menggambarkan kebiasaan, bukan nilai ibadah.
var TasbihBadges = []TasbihBadgeDef{
	{"langkah-pertama", "Langkah Pertama", "Menyelesaikan sesi dzikir pertama", "Footprints"},
	{"target-pertama", "Target Pertama", "Mencapai target harian untuk pertama kali", "Target"},
	{"paket-pertama", "Wirid Tuntas", "Menuntaskan satu putaran paket wirid", "ListChecks"},
	{"custom-pertama", "Dzikir Pilihan", "Menambahkan dzikir custom pertama", "PenLine"},
	{"total-1k", "Seribu", "Total 1.000 hitungan", "Sparkles"},
	{"total-10k", "Sepuluh Ribu", "Total 10.000 hitungan", "Star"},
	{"total-100k", "Seratus Ribu", "Total 100.000 hitungan", "Gem"},
	{"total-1m", "Sejuta", "Total 1.000.000 hitungan", "Crown"},
	{"streak-3", "3 Hari Beruntun", "Mencapai target harian 3 hari berturut-turut", "Flame"},
	{"streak-7", "7 Hari Beruntun", "Mencapai target harian 7 hari berturut-turut", "Flame"},
	{"streak-30", "30 Hari Beruntun", "Mencapai target harian 30 hari berturut-turut", "Flame"},
	{"streak-100", "100 Hari Beruntun", "Mencapai target harian 100 hari berturut-turut", "Flame"},
	{"badah-shalat-35", "Sepekan Ba'da Shalat", "35 putaran Dzikir Ba'da Shalat dalam 7 hari terakhir", "Repeat"},
	{"subuh-7", "Pagi Hari", "Berdzikir pukul 03.00–05.59 WIB pada 7 hari berbeda", "Sunrise"},
	{"istighfar-1k", "Seribu Istighfar", "Total 1.000 istighfar", "Droplets"},
}

func badgeDef(kode string) (TasbihBadgeDef, bool) {
	for _, b := range TasbihBadges {
		if b.Kode == kode {
			return b, true
		}
	}
	return TasbihBadgeDef{}, false
}

func badgeModel(d TasbihBadgeDef, at *time.Time) models.TasbihBadge {
	return models.TasbihBadge{Kode: d.Kode, Nama: d.Nama, Syarat: d.Syarat, Ikon: d.Ikon, DiraihAt: at}
}

// BadgeDariKode memetakan daftar kode (mis. dari flash) ke model badge.
func BadgeDariKode(kode []string) []models.TasbihBadge {
	out := []models.TasbihBadge{}
	for _, k := range kode {
		if d, ok := badgeDef(k); ok {
			out = append(out, badgeModel(d, nil))
		}
	}
	return out
}

// EvaluasiBadge memberi badge yang syaratnya terpenuhi dan mengembalikan yang baru.
// Badge tidak pernah dicabut.
func (s *TasbihService) EvaluasiBadge(ctx context.Context, userID int64, now time.Time) ([]models.TasbihBadge, error) {
	q := s.querier
	sesi, err := q.CountTasbihSesi(ctx, userID)
	if err != nil {
		return nil, err
	}
	custom, err := q.CountTasbihDzikirCustomPernah(ctx, nullUser(userID))
	if err != nil {
		return nil, err
	}
	lv, err := s.Level(ctx, userID)
	if err != nil {
		return nil, err
	}
	streak, err := s.Streak(ctx, userID, now)
	if err != nil {
		return nil, err
	}
	putaran, err := s.JumlahPutaranPaket(ctx, userID, nil, nil)
	if err != nil {
		return nil, err
	}
	var badah int64
	pid, err := q.GetTasbihPaketIDBySlug(ctx, "badah-shalat")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		sejak := tasbihGeser(tasbihTanggal(now), -6)
		if badah, err = s.JumlahPutaranPaket(ctx, userID, &pid, &sejak); err != nil {
			return nil, err
		}
	}
	subuh, err := q.CountTasbihHariJam(ctx, queries.CountTasbihHariJamParams{UserID: userID, JamDari: 3, JamSampai: 5})
	if err != nil {
		return nil, err
	}
	istighfar, err := q.SumTasbihBySlug(ctx, queries.SumTasbihBySlugParams{
		UserID: userID, Slug1: sql.NullString{String: "istighfar", Valid: true}, Slug2: sql.NullString{String: "istighfar-taubat", Valid: true},
	})
	if err != nil {
		return nil, err
	}

	syarat := map[string]bool{
		"langkah-pertama": sesi >= 1,
		"target-pertama":  streak.Terpanjang >= 1,
		"paket-pertama":   putaran >= 1,
		"custom-pertama":  custom >= 1,
		"total-1k":        lv.Total >= 1000,
		"total-10k":       lv.Total >= 10000,
		"total-100k":      lv.Total >= 100000,
		"total-1m":        lv.Total >= 1000000,
		"streak-3":        streak.Terpanjang >= 3,
		"streak-7":        streak.Terpanjang >= 7,
		"streak-30":       streak.Terpanjang >= 30,
		"streak-100":      streak.Terpanjang >= 100,
		"badah-shalat-35": badah >= 35,
		"subuh-7":         subuh >= 7,
		"istighfar-1k":    istighfar >= 1000,
	}
	baru := []models.TasbihBadge{}
	for _, d := range TasbihBadges {
		if !syarat[d.Kode] {
			continue
		}
		n, err := q.InsertTasbihBadge(ctx, queries.InsertTasbihBadgeParams{UserID: userID, BadgeKode: d.Kode, DiraihAt: now.UTC()})
		if err != nil {
			return nil, err
		}
		if n > 0 {
			at := now
			baru = append(baru, badgeModel(d, &at))
		}
	}
	return baru, nil
}

// DaftarBadge: semua badge (urutan TasbihBadges), DiraihAt nil bila terkunci.
func (s *TasbihService) DaftarBadge(ctx context.Context, userID int64) ([]models.TasbihBadge, error) {
	rows, err := s.querier.ListTasbihBadgeUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	diraih := make(map[string]time.Time, len(rows))
	for _, r := range rows {
		diraih[r.BadgeKode] = r.DiraihAt
	}
	out := make([]models.TasbihBadge, 0, len(TasbihBadges))
	for _, d := range TasbihBadges {
		var at *time.Time
		if t, ok := diraih[d.Kode]; ok {
			t := t
			at = &t
		}
		out = append(out, badgeModel(d, at))
	}
	return out, nil
}
