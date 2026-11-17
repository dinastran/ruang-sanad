package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

var tasbihUUIDSeq int

func uuidT() string {
	tasbihUUIDSeq++
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", tasbihUUIDSeq)
}

// sesiT membuat input sesi valid untuk dzikir slug, mulai pada waktu m.
func sesiT(t *testing.T, db *sql.DB, slug string, jumlah, target int64, m time.Time) models.TasbihSesiInput {
	t.Helper()
	return models.TasbihSesiInput{ID: uuidT(), DzikirID: dzikirIDBySlug(t, db, slug), Jumlah: jumlah, Target: target, MulaiAt: m, SelesaiAt: m.Add(time.Minute)}
}

var tasbihNow = time.Date(2026, 11, 16, 5, 0, 0, 0, time.UTC) // 12:00 WIB

func TestSyncIdempoten(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	batch := []models.TasbihSesiInput{
		sesiT(t, db, "subhanallah", 33, 33, tasbihNow.Add(-time.Hour)),
		sesiT(t, db, "alhamdulillah", 10, 33, tasbihNow.Add(-30*time.Minute)),
	}
	for i := 0; i < 2; i++ {
		h, err := svc.Sync(ctx, u, batch, tasbihNow)
		require.NoError(t, err)
		require.ElementsMatch(t, []string{batch[0].ID, batch[1].ID}, h.Diterima)
		require.Empty(t, h.Ditolak)
		require.EqualValues(t, 43, h.HariIni.Total)
		require.Equal(t, "2026-11-16", h.HariIni.Tanggal)
	}
	require.EqualValues(t, 2, countRows(t, db, "SELECT COUNT(*) FROM tasbih_sesi WHERE user_id = ?", u))
	require.EqualValues(t, 1, countRows(t, db, "SELECT selesai_target FROM tasbih_sesi WHERE id = ?", batch[0].ID))
	require.EqualValues(t, 0, countRows(t, db, "SELECT selesai_target FROM tasbih_sesi WHERE id = ?", batch[1].ID))
}

func TestSyncIDMilikUserLain(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	a, b := seedUser(t, db, "user"), seedUser(t, db, "user")
	s := sesiT(t, db, "subhanallah", 33, 33, tasbihNow.Add(-time.Hour))
	_, err := svc.Sync(ctx, a, []models.TasbihSesiInput{s}, tasbihNow)
	require.NoError(t, err)
	s2 := s
	s2.Jumlah = 999
	h, err := svc.Sync(ctx, b, []models.TasbihSesiInput{s2}, tasbihNow)
	require.NoError(t, err)
	require.Empty(t, h.Diterima)
	require.Len(t, h.Ditolak, 1)
	require.EqualValues(t, 33, countRows(t, db, "SELECT jumlah FROM tasbih_sesi WHERE id = ?", s.ID))
	require.EqualValues(t, a, countRows(t, db, "SELECT user_id FROM tasbih_sesi WHERE id = ?", s.ID))
}

func TestSyncValidasi(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u, lain := seedUser(t, db, "user"), seedUser(t, db, "user")
	customLain, err := svc.BuatDzikir(ctx, lain, models.TasbihDzikirInput{Latin: "Milik lain", Target: 3})
	require.NoError(t, err)
	base := tasbihNow.Add(-time.Hour)
	mk := func(f func(*models.TasbihSesiInput)) models.TasbihSesiInput {
		s := sesiT(t, db, "subhanallah", 5, 33, base)
		f(&s)
		return s
	}
	tiada := int64(9999)
	buruk := []models.TasbihSesiInput{
		mk(func(s *models.TasbihSesiInput) { s.ID = "bukan-uuid" }),
		mk(func(s *models.TasbihSesiInput) { s.Jumlah = 0 }),
		mk(func(s *models.TasbihSesiInput) { s.Jumlah = 100001 }),
		mk(func(s *models.TasbihSesiInput) { s.SelesaiAt = s.MulaiAt.Add(-time.Second) }),
		mk(func(s *models.TasbihSesiInput) { s.MulaiAt = tasbihNow.Add(6 * time.Minute); s.SelesaiAt = s.MulaiAt }),
		mk(func(s *models.TasbihSesiInput) {
			s.MulaiAt = tasbihNow.Add(-401 * 24 * time.Hour)
			s.SelesaiAt = s.MulaiAt
		}),
		mk(func(s *models.TasbihSesiInput) { s.DzikirID = customLain }),
		mk(func(s *models.TasbihSesiInput) { s.PaketID = &tiada }),
		mk(func(s *models.TasbihSesiInput) { s.Target = 0 }),
	}
	baik := sesiT(t, db, "subhanallah", 5, 33, base)
	h, err := svc.Sync(ctx, u, append(buruk, baik), tasbihNow)
	require.NoError(t, err)
	require.Equal(t, []string{baik.ID}, h.Diterima)
	require.Len(t, h.Ditolak, len(buruk))
	for _, d := range h.Ditolak {
		require.NotEmpty(t, d.Alasan)
	}
}

func TestSyncDzikirDiarsipkanTetapDiterima(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	id, err := svc.BuatDzikir(ctx, u, models.TasbihDzikirInput{Latin: "Arsip", Target: 3})
	require.NoError(t, err)
	require.NoError(t, svc.ArsipkanDzikir(ctx, u, id))
	s := models.TasbihSesiInput{ID: uuidT(), DzikirID: id, Jumlah: 3, Target: 3, MulaiAt: tasbihNow.Add(-time.Hour), SelesaiAt: tasbihNow.Add(-time.Hour)}
	h, err := svc.Sync(ctx, u, []models.TasbihSesiInput{s}, tasbihNow)
	require.NoError(t, err)
	require.Equal(t, []string{s.ID}, h.Diterima)
}

func TestSyncTanggalWIBBoundary(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	a := sesiT(t, db, "subhanallah", 1, 33, time.Date(2026, 11, 15, 16, 59, 0, 0, time.UTC))
	b := sesiT(t, db, "subhanallah", 1, 33, time.Date(2026, 11, 15, 17, 1, 0, 0, time.UTC))
	_, err := svc.Sync(ctx, u, []models.TasbihSesiInput{a, b}, tasbihNow)
	require.NoError(t, err)
	var tgl string
	require.NoError(t, db.QueryRow("SELECT tanggal_wib FROM tasbih_sesi WHERE id = ?", a.ID).Scan(&tgl))
	require.Equal(t, "2026-11-15", tgl)
	require.EqualValues(t, 23, countRows(t, db, "SELECT jam_wib FROM tasbih_sesi WHERE id = ?", a.ID))
	require.NoError(t, db.QueryRow("SELECT tanggal_wib FROM tasbih_sesi WHERE id = ?", b.ID).Scan(&tgl))
	require.Equal(t, "2026-11-16", tgl)
	require.EqualValues(t, 0, countRows(t, db, "SELECT jam_wib FROM tasbih_sesi WHERE id = ?", b.ID))
}

func paketIDBySlug(t *testing.T, db *sql.DB, slug string) int64 {
	t.Helper()
	return countRows(t, db, "SELECT id FROM tasbih_paket WHERE slug = ?", slug)
}

func TestSyncPaketTargetTidakCocok(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	pid := paketIDBySlug(t, db, "badah-shalat")
	run := uuidT()
	s := sesiT(t, db, "tahlil-lengkap", 100, 100, tasbihNow.Add(-time.Hour))
	s.PaketID, s.PaketRunID = &pid, &run
	ok := sesiT(t, db, "subhanallah", 33, 33, tasbihNow.Add(-time.Hour))
	ok.PaketID, ok.PaketRunID = &pid, &run
	h, err := svc.Sync(ctx, u, []models.TasbihSesiInput{s, ok}, tasbihNow)
	require.NoError(t, err)
	require.Len(t, h.Diterima, 2)
	require.EqualValues(t, 1, countRows(t, db, "SELECT COUNT(*) FROM tasbih_sesi WHERE id = ? AND paket_id IS NULL AND paket_run_id IS NULL", s.ID))
	require.EqualValues(t, 1, countRows(t, db, "SELECT COUNT(*) FROM tasbih_sesi WHERE id = ? AND paket_id = ? AND paket_run_id = ?", ok.ID, pid, run))
}

func TestSyncBatchTerlaluBesar(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	u := seedUser(t, db, "user")
	batch := make([]models.TasbihSesiInput, 101)
	for i := range batch {
		batch[i] = sesiT(t, db, "subhanallah", 1, 33, tasbihNow.Add(-time.Hour))
	}
	_, err := svc.Sync(context.Background(), u, batch, tasbihNow)
	require.ErrorIs(t, err, ErrTasbihValidasi)
}
