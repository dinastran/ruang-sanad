package services

import (
	"context"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

func TestHitungLevel(t *testing.T) {
	cases := []struct {
		total int64
		level int
	}{{0, 1}, {99, 1}, {100, 2}, {1499, 3}, {1500, 4}, {250000, 10}, {999999, 10}}
	for _, c := range cases {
		require.Equal(t, c.level, HitungLevel(c.total).Level, "total %d", c.total)
	}
	l := HitungLevel(150)
	require.EqualValues(t, 100, l.AmbangIni)
	require.EqualValues(t, 500, l.AmbangBerikut)
	require.EqualValues(t, 0, HitungLevel(250000).AmbangBerikut)
}

func TestStreakHariIniBelumTercapai(t *testing.T) {
	s := HitungStreak([]string{"2026-11-13", "2026-11-14", "2026-11-15"}, "2026-11-16")
	require.Equal(t, 3, s.Saat)
	require.Equal(t, 3, s.Terpanjang)
	require.False(t, s.HariIniTercapai)
}

func TestStreakBolong(t *testing.T) {
	s := HitungStreak([]string{"2026-11-10", "2026-11-11", "2026-11-13", "2026-11-14", "2026-11-15", "2026-11-16"}, "2026-11-16")
	require.Equal(t, 4, s.Saat)
	require.Equal(t, 4, s.Terpanjang)
	require.True(t, s.HariIniTercapai)
}

func TestStreakPutus(t *testing.T) {
	s := HitungStreak([]string{"2026-11-01", "2026-11-02", "2026-11-03", "2026-11-14"}, "2026-11-16")
	require.Equal(t, 0, s.Saat)
	require.Equal(t, 3, s.Terpanjang)
	require.Equal(t, models.TasbihStreak{}, HitungStreak(nil, "2026-11-16"))
}

func TestDashboardAgregat(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u, lain := seedUser(t, db, "user"), seedUser(t, db, "user")
	d16 := time.Date(2026, 11, 16, 1, 0, 0, 0, time.UTC)  // 08 WIB
	d15 := time.Date(2026, 11, 15, 13, 0, 0, 0, time.UTC) // 20 WIB
	d05 := time.Date(2026, 11, 5, 3, 0, 0, 0, time.UTC)   // 10 WIB
	var batch []models.TasbihSesiInput
	for i, slug := range []string{"subhanallah", "alhamdulillah", "allahu-akbar", "tahlil", "istighfar", "tasbih-tahmid", "hauqalah", "shalawat"} {
		batch = append(batch, sesiT(t, db, slug, int64(80-10*i), 33, d16))
	}
	batch = append(batch, sesiT(t, db, "subhanallah", 40, 33, d15), sesiT(t, db, "subhanallah", 7, 33, d05))
	_, err := svc.Sync(ctx, u, batch, tasbihNow)
	require.NoError(t, err)
	_, err = svc.Sync(ctx, lain, []models.TasbihSesiInput{sesiT(t, db, "subhanallah", 500, 33, d16)}, tasbihNow)
	require.NoError(t, err)

	d, err := svc.Dashboard(ctx, u, 7, tasbihNow)
	require.NoError(t, err)
	require.Equal(t, 7, d.Rentang)
	require.Len(t, d.Harian, 7)
	require.Equal(t, "2026-11-10", d.Harian[0].Tanggal)
	require.Equal(t, "2026-11-16", d.Harian[6].Tanggal)
	require.EqualValues(t, 360, d.Harian[6].Total)
	require.EqualValues(t, 40, d.Harian[5].Total)
	require.EqualValues(t, 0, d.Harian[0].Total)
	require.EqualValues(t, 400, d.TotalRentang)
	require.EqualValues(t, 7, d.TotalRentangSebelum)
	require.InDelta(t, 400.0/7, d.RataRata, 0.001)
	require.EqualValues(t, 360, d.PerJam[8])
	require.EqualValues(t, 40, d.PerJam[20])
	require.Len(t, d.Komposisi, 7)
	require.Equal(t, "Subhanallah", d.Komposisi[0].Latin)
	require.EqualValues(t, 120, d.Komposisi[0].Total)
	require.Equal(t, "Lainnya", d.Komposisi[6].Latin)
	require.EqualValues(t, 30, d.Komposisi[6].Total)
	require.Len(t, d.Heatmap, 365)
	require.Equal(t, "2026-11-16", d.Heatmap[364].Tanggal)
	require.EqualValues(t, 360, d.HariIni.Total)
	require.EqualValues(t, 100, d.HariIni.Target)
	require.EqualValues(t, 407, d.Level.Total)
	require.Equal(t, 2, d.Level.Level)
	require.Equal(t, 1, d.Streak.Saat) // tgl 15 hanya 40 < target 100
	require.True(t, d.Streak.HariIniTercapai)

	// Ringkasan sesi rentang 7 hari: 8 sesi tgl 16 + 1 sesi tgl 15, @1 menit.
	require.EqualValues(t, 9, d.Sesi.Jumlah)
	require.EqualValues(t, 6, d.Sesi.Tuntas) // jumlah >= 33: 80,70,60,50,40 (tgl 16) + 40 (tgl 15)
	require.EqualValues(t, 9*60, d.Sesi.DurasiDetik)
	require.InDelta(t, 400.0/9, d.Sesi.PerMenit, 0.001)
	require.InDelta(t, 400.0/9, d.Sesi.RataPerSesi, 0.001)
	require.EqualValues(t, 80, d.Sesi.Terbesar)
	require.NotNil(t, d.HariTerbaik)
	require.Equal(t, models.TasbihTitikHari{Tanggal: "2026-11-16", Total: 360}, *d.HariTerbaik)
	require.Equal(t, models.TasbihTitikHari{Tanggal: "2026-11-16", Total: 360}, *d.RekorTahun)
	require.Len(t, d.HariIniDzikir, 8)
	require.Equal(t, "Subhanallah", d.HariIniDzikir[0].Latin)
}

func TestHitungSesiStatBatasiDurasi(t *testing.T) {
	m := time.Date(2026, 11, 16, 1, 0, 0, 0, time.UTC)
	st := HitungSesiStat([]queries.ListTasbihSesiRingkasRow{
		{Jumlah: 100, SelesaiTarget: 1, MulaiAt: m, SelesaiAt: m.Add(10 * time.Hour)}, // lupa ditutup → 2 jam
		{Jumlah: 20, MulaiAt: m, SelesaiAt: m.Add(-time.Minute)},                     // jam mundur → 0
	})
	require.EqualValues(t, 2, st.Jumlah)
	require.EqualValues(t, 1, st.Tuntas)
	require.EqualValues(t, 7200, st.DurasiDetik)
	require.InDelta(t, 1.0, st.PerMenit, 0.001)
	require.Nil(t, hariTerbanyak([]models.TasbihTitikHari{{Tanggal: "2026-11-16"}}))
}

func TestPutaranPaket(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	pid := paketIDBySlug(t, db, "badah-shalat")
	m := tasbihNow.Add(-time.Hour)
	run := func(slugs ...string) []models.TasbihSesiInput {
		id := uuidT()
		target := map[string]int64{"subhanallah": 33, "alhamdulillah": 33, "allahu-akbar": 33, "tahlil-lengkap": 1}
		var out []models.TasbihSesiInput
		for _, s := range slugs {
			x := sesiT(t, db, s, target[s], target[s], m)
			x.PaketID, x.PaketRunID = &pid, &id
			out = append(out, x)
		}
		return out
	}
	batch := append(run("subhanallah", "alhamdulillah", "allahu-akbar", "tahlil-lengkap"), run("subhanallah", "alhamdulillah", "allahu-akbar")...)
	_, err := svc.Sync(ctx, u, batch, tasbihNow)
	require.NoError(t, err)
	d, err := svc.Dashboard(ctx, u, 30, tasbihNow)
	require.NoError(t, err)
	require.Len(t, d.Paket, 5)
	require.Equal(t, pid, d.Paket[0].PaketID)
	require.EqualValues(t, 1, d.Paket[0].Selesai)
	n, err := svc.JumlahPutaranPaket(ctx, u, &pid, nil)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
}

func TestRiwayatPaginasi(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u, lain := seedUser(t, db, "user"), seedUser(t, db, "user")
	var batch []models.TasbihSesiInput
	for i := 0; i < 25; i++ {
		slug := "subhanallah"
		if i%5 == 0 {
			slug = "istighfar"
		}
		batch = append(batch, sesiT(t, db, slug, 10, 33, tasbihNow.Add(-time.Duration(i)*time.Hour)))
	}
	_, err := svc.Sync(ctx, u, batch, tasbihNow)
	require.NoError(t, err)
	_, err = svc.Sync(ctx, lain, []models.TasbihSesiInput{sesiT(t, db, "subhanallah", 1, 33, tasbihNow)}, tasbihNow)
	require.NoError(t, err)

	p1, lagi, err := svc.Riwayat(ctx, u, 1, nil)
	require.NoError(t, err)
	require.Len(t, p1, 20)
	require.True(t, lagi)
	require.Equal(t, batch[0].ID, p1[0].ID)
	require.Equal(t, "12:00", p1[0].Jam)
	require.EqualValues(t, 60, p1[0].DurasiDetik)
	p2, lagi, err := svc.Riwayat(ctx, u, 2, nil)
	require.NoError(t, err)
	require.Len(t, p2, 5)
	require.False(t, lagi)
	ist := dzikirIDBySlug(t, db, "istighfar")
	f, _, err := svc.Riwayat(ctx, u, 1, &ist)
	require.NoError(t, err)
	require.Len(t, f, 5)
}

// Review #3: item paket yang terbagi ke beberapa sesi (jeda > 5 menit) tetap
// terhitung tuntas bila jumlah per (run, dzikir) mencapai target item.
func TestPutaranPaketItemTerbagi(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	pid := paketIDBySlug(t, db, "badah-shalat")
	run := uuidT()
	m := tasbihNow.Add(-time.Hour)
	mk := func(slug string, jumlah, target int64) models.TasbihSesiInput {
		x := sesiT(t, db, slug, jumlah, target, m)
		x.PaketID, x.PaketRunID = &pid, &run
		return x
	}
	batch := []models.TasbihSesiInput{
		mk("subhanallah", 10, 33), mk("subhanallah", 23, 33),
		mk("alhamdulillah", 33, 33), mk("allahu-akbar", 33, 33), mk("tahlil-lengkap", 1, 1),
	}
	_, err := svc.Sync(ctx, u, batch, tasbihNow)
	require.NoError(t, err)
	n, err := svc.JumlahPutaranPaket(ctx, u, &pid, nil)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
}

func TestKomposisiHariIni(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u, lain := seedUser(t, db, "user"), seedUser(t, db, "user")
	kemarin := tasbihNow.Add(-24 * time.Hour)
	var b []models.TasbihSesiInput
	b = append(b, sesiT(t, db, "subhanallah", 33, 33, tasbihNow.Add(-2*time.Hour)), sesiT(t, db, "subhanallah", 33, 33, tasbihNow.Add(-time.Hour)),
		sesiT(t, db, "istighfar", 100, 100, tasbihNow.Add(-time.Hour)), sesiT(t, db, "hauqalah", 10, 33, kemarin))
	slugs := []string{"alhamdulillah", "allahu-akbar", "tahlil", "tasbih-tahmid", "shalawat", "hasbunallah", "tasbih-azhim"}
	for i, s := range slugs {
		b = append(b, sesiT(t, db, s, int64(i+1), 33, tasbihNow.Add(-time.Hour)))
	}
	_, err := svc.Sync(ctx, u, b, tasbihNow)
	require.NoError(t, err)
	_, err = svc.Sync(ctx, lain, []models.TasbihSesiInput{sesiT(t, db, "subhanallah", 500, 33, tasbihNow.Add(-time.Hour))}, tasbihNow)
	require.NoError(t, err)

	k, err := svc.KomposisiHariIni(ctx, u, tasbihNow)
	require.NoError(t, err)
	require.Len(t, k, 9, "semua dzikir hari ini, tanpa digabung 'Lainnya'")
	require.Equal(t, "Astaghfirullah", k[0].Latin)
	require.EqualValues(t, 100, k[0].Total)
	require.Equal(t, "Subhanallah", k[1].Latin)
	require.EqualValues(t, 66, k[1].Total)
	for _, x := range k {
		require.NotEqual(t, "Laa haula wa laa quwwata illaa billaah", x.Latin, "dzikir kemarin tidak ikut")
	}
}
