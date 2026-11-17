package services

import (
	"context"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func kodeBadge(b []models.TasbihBadge) []string {
	out := []string{}
	for _, x := range b {
		out = append(out, x.Kode)
	}
	return out
}

func TestBadgeLangkahPertama(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	h, err := svc.Sync(ctx, u, []models.TasbihSesiInput{sesiT(t, db, "subhanallah", 5, 33, tasbihNow.Add(-time.Hour))}, tasbihNow)
	require.NoError(t, err)
	require.Contains(t, kodeBadge(h.BadgeBaru), "langkah-pertama")
	h, err = svc.Sync(ctx, u, []models.TasbihSesiInput{sesiT(t, db, "subhanallah", 5, 33, tasbihNow.Add(-time.Hour))}, tasbihNow)
	require.NoError(t, err)
	require.NotContains(t, kodeBadge(h.BadgeBaru), "langkah-pertama")
}

func TestBadgeTotal1k(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	h, err := svc.Sync(ctx, u, []models.TasbihSesiInput{sesiT(t, db, "tahlil", 999, 100, tasbihNow.Add(-time.Hour))}, tasbihNow)
	require.NoError(t, err)
	require.NotContains(t, kodeBadge(h.BadgeBaru), "total-1k")
	h, err = svc.Sync(ctx, u, []models.TasbihSesiInput{sesiT(t, db, "tahlil", 1, 100, tasbihNow.Add(-time.Hour))}, tasbihNow)
	require.NoError(t, err)
	require.Contains(t, kodeBadge(h.BadgeBaru), "total-1k")
}

func TestBadgeStreak3(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	var b []models.TasbihSesiInput
	for i := 0; i < 3; i++ {
		b = append(b, sesiT(t, db, "tahlil", 100, 100, tasbihNow.Add(-time.Duration(i)*24*time.Hour)))
	}
	h, err := svc.Sync(ctx, u, b, tasbihNow)
	require.NoError(t, err)
	require.Subset(t, kodeBadge(h.BadgeBaru), []string{"streak-3", "target-pertama"})
	require.NotContains(t, kodeBadge(h.BadgeBaru), "streak-7")
}

func TestBadgeSubuh7(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	subuh := time.Date(2026, 11, 15, 21, 30, 0, 0, time.UTC) // 04:30 WIB tgl 16
	var b []models.TasbihSesiInput
	for i := 0; i < 6; i++ {
		b = append(b, sesiT(t, db, "subhanallah", 1, 33, subuh.Add(-time.Duration(i)*24*time.Hour)))
	}
	h, err := svc.Sync(ctx, u, b, tasbihNow)
	require.NoError(t, err)
	require.NotContains(t, kodeBadge(h.BadgeBaru), "subuh-7")
	h, err = svc.Sync(ctx, u, []models.TasbihSesiInput{sesiT(t, db, "subhanallah", 1, 33, subuh.Add(-6*24*time.Hour))}, tasbihNow)
	require.NoError(t, err)
	require.Contains(t, kodeBadge(h.BadgeBaru), "subuh-7")
}

func TestBadgeCustomPertama(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	_, err := svc.BuatDzikir(ctx, u, models.TasbihDzikirInput{Latin: "Custom", Target: 3})
	require.NoError(t, err)
	b, err := svc.EvaluasiBadge(ctx, u, tasbihNow)
	require.NoError(t, err)
	require.Equal(t, []string{"custom-pertama"}, kodeBadge(b))
}

func TestBadgeTargetPertamaSetelahTurunkanTarget(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	h, err := svc.Sync(ctx, u, []models.TasbihSesiInput{sesiT(t, db, "subhanallah", 50, 33, tasbihNow.Add(-time.Hour))}, tasbihNow)
	require.NoError(t, err)
	require.NotContains(t, kodeBadge(h.BadgeBaru), "target-pertama")
	require.NoError(t, svc.SetTargetHarian(ctx, u, 50))
	b, err := svc.EvaluasiBadge(ctx, u, tasbihNow)
	require.NoError(t, err)
	require.Contains(t, kodeBadge(b), "target-pertama")
}

func TestBadgeIstighfar1k(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	h, err := svc.Sync(ctx, u, []models.TasbihSesiInput{
		sesiT(t, db, "istighfar", 600, 100, tasbihNow.Add(-time.Hour)),
		sesiT(t, db, "istighfar-taubat", 400, 100, tasbihNow.Add(-time.Hour)),
	}, tasbihNow)
	require.NoError(t, err)
	require.Contains(t, kodeBadge(h.BadgeBaru), "istighfar-1k")
}

func TestBadgeBadahShalat35(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	pid := paketIDBySlug(t, db, "badah-shalat")
	target := map[string]int64{"subhanallah": 33, "alhamdulillah": 33, "allahu-akbar": 33, "tahlil-lengkap": 1}
	kirim := func(n int) []string {
		var all []string
		for r := 0; r < n; r++ {
			id := uuidT()
			var b []models.TasbihSesiInput
			for _, slug := range []string{"subhanallah", "alhamdulillah", "allahu-akbar", "tahlil-lengkap"} {
				x := sesiT(t, db, slug, target[slug], target[slug], tasbihNow.Add(-time.Duration(r)*time.Hour))
				x.PaketID, x.PaketRunID = &pid, &id
				b = append(b, x)
			}
			h, err := svc.Sync(ctx, u, b, tasbihNow)
			require.NoError(t, err)
			all = append(all, kodeBadge(h.BadgeBaru)...)
		}
		return all
	}
	got := kirim(34)
	require.Contains(t, got, "paket-pertama")
	require.NotContains(t, got, "badah-shalat-35")
	require.Contains(t, kirim(1), "badah-shalat-35")
}

func TestBadgeTidakDicabut(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	_, err := svc.Sync(ctx, u, []models.TasbihSesiInput{sesiT(t, db, "tahlil", 100, 100, tasbihNow.Add(-time.Hour))}, tasbihNow)
	require.NoError(t, err)
	require.NoError(t, svc.SetTargetHarian(ctx, u, 5000))
	_, err = svc.EvaluasiBadge(ctx, u, tasbihNow)
	require.NoError(t, err)
	all, err := svc.DaftarBadge(ctx, u)
	require.NoError(t, err)
	for _, b := range all {
		if b.Kode == "target-pertama" {
			require.NotNil(t, b.DiraihAt)
		}
	}
}

func TestDaftarBadge(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	u := seedUser(t, db, "user")
	all, err := svc.DaftarBadge(context.Background(), u)
	require.NoError(t, err)
	require.Len(t, all, 15)
	for i, b := range all {
		require.Equal(t, TasbihBadges[i].Kode, b.Kode)
		require.Nil(t, b.DiraihAt)
		require.NotEmpty(t, b.Nama)
		require.NotEmpty(t, b.Syarat)
	}
}
