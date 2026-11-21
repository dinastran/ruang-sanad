package services

import (
	"context"
	"strings"

	"database/sql"
	"fmt"
	"github.com/maulanashalihin/laju-go/app/models"
	"path/filepath"
	"testing"

	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newTasbihTestDB(t *testing.T) (*sql.DB, *TasbihService) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	return db, NewTasbihService(queries.NewQuerier(db))
}

var tasbihUserSeq int

func seedUser(t *testing.T, db *sql.DB, role string) int64 {
	t.Helper()
	tasbihUserSeq++
	res, err := db.Exec("INSERT INTO users (email, name, role) VALUES (?, ?, ?)",
		fmt.Sprintf("tasbih%d@test.local", tasbihUserSeq), "User Tasbih", role)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

func countRows(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.QueryRow(query, args...).Scan(&n))
	return n
}

func TestTasbihSeed(t *testing.T) {
	db, _ := newTasbihTestDB(t)
	require.EqualValues(t, 12, countRows(t, db, "SELECT COUNT(*) FROM tasbih_dzikir WHERE user_id IS NULL AND hanya_paket = 0"))
	require.EqualValues(t, 34, countRows(t, db, "SELECT COUNT(*) FROM tasbih_dzikir WHERE user_id IS NULL AND hanya_paket = 1"))
	require.EqualValues(t, 0, countRows(t, db, "SELECT COUNT(*) FROM tasbih_dzikir WHERE user_id IS NULL AND (teks_arab = '' OR arti = '' OR rujukan = '')"))
	require.EqualValues(t, 5, countRows(t, db, "SELECT COUNT(*) FROM tasbih_paket"))
	require.EqualValues(t, 4, countRows(t, db, "SELECT COUNT(*) FROM tasbih_paket_item i JOIN tasbih_paket p ON p.id = i.paket_id WHERE p.slug = 'badah-shalat'"))
	require.EqualValues(t, 1, countRows(t, db, "SELECT i.target FROM tasbih_paket_item i JOIN tasbih_paket p ON p.id = i.paket_id WHERE p.slug = 'badah-shalat' AND i.urutan = 4"))
	require.EqualValues(t, 34, countRows(t, db, "SELECT i.target FROM tasbih_paket_item i JOIN tasbih_paket p ON p.id = i.paket_id WHERE p.slug = 'tasbih-fatimah' AND i.urutan = 3"))
	require.EqualValues(t, 0, countRows(t, db, "SELECT COUNT(*) FROM tasbih_paket_item WHERE dzikir_id IS NULL"))
}

func dzikirIDBySlug(t *testing.T, db *sql.DB, slug string) int64 {
	t.Helper()
	return countRows(t, db, "SELECT id FROM tasbih_dzikir WHERE slug = ?", slug)
}

func TestKatalogGabungBawaanDanCustom(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	a, b := seedUser(t, db, "user"), seedUser(t, db, "user")
	_, err := svc.BuatDzikir(ctx, a, models.TasbihDzikirInput{Latin: "Ya Hayyu ya Qayyum", Target: 33})
	require.NoError(t, err)

	dz, paket, err := svc.Katalog(ctx, a)
	require.NoError(t, err)
	// 12 dzikir umum + 34 dzikir Al-Ma'tsurat (hanya_paket) + 1 custom.
	require.Len(t, dz, 47)
	require.Len(t, paket, 5)
	require.Len(t, paket[0].Items, 4)
	require.True(t, dz[46].Custom)
	require.False(t, dz[46].HanyaPaket)

	dzB, _, err := svc.Katalog(ctx, b)
	require.NoError(t, err)
	require.Len(t, dzB, 46)
}

// Al-Ma'tsurat Sughra: urutan, target, dan varian pagi/petang sesuai bacaan bertanda ٭.
func TestSeedMatsuratSughra(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	dz, paket, err := svc.Katalog(ctx, u)
	require.NoError(t, err)
	byID := map[int64]models.TasbihDzikir{}
	for _, d := range dz {
		byID[d.ID] = d
	}
	bySlug := map[string]models.TasbihPaket{}
	for _, p := range paket {
		bySlug[p.Slug] = p
	}
	pagi, petang := bySlug["matsurat-pagi"], bySlug["matsurat-petang"]
	require.Len(t, pagi.Items, 30)
	require.Len(t, petang.Items, 31)

	total := int64(0)
	for _, it := range pagi.Items {
		total += it.Target
		require.NotEmpty(t, byID[it.DzikirID].Arab, "teks Arab wajib ada")
	}
	require.EqualValues(t, 185, total)

	nama := func(p models.TasbihPaket, i int) string { return byID[p.Items[i].DzikirID].Latin }
	require.Equal(t, "Ta'awudz & Al-Fatihah", nama(pagi, 0))
	require.Equal(t, "Al-Baqarah 284–286", nama(pagi, 3))
	require.Equal(t, "Al-Ikhlas", nama(pagi, 4))
	require.EqualValues(t, 3, pagi.Items[4].Target)
	require.Equal(t, "Ashbahna wa ashbahal mulku lillah", nama(pagi, 7))
	require.Equal(t, "Amsaina wa amsal mulku lillah", nama(petang, 7))
	require.EqualValues(t, 100, pagi.Items[23].Target)
	require.Equal(t, dzikirIDBySlug(t, db, "tahlil-lengkap"), pagi.Items[24].DzikirID)
	require.EqualValues(t, 10, pagi.Items[24].Target)
	require.Equal(t, "Allahumma inna hadza iqbalu lailik", nama(petang, 29))
	require.Equal(t, "Doa Rabithah", nama(pagi, 29))
	require.Equal(t, "Doa Rabithah", nama(petang, 30))

	require.True(t, byID[pagi.Items[0].DzikirID].HanyaPaket)
	require.False(t, byID[pagi.Items[24].DzikirID].HanyaPaket, "tahlil-lengkap tetap dzikir satuan")
}

func TestTargetEfektifOverride(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	id := dzikirIDBySlug(t, db, "subhanallah")
	require.NoError(t, svc.SetTargetDzikir(ctx, u, id, 99))
	dz, _, err := svc.Katalog(ctx, u)
	require.NoError(t, err)
	for _, d := range dz {
		if d.ID == id {
			require.EqualValues(t, 99, d.Target)
			require.EqualValues(t, 33, d.TargetDefault)
		}
	}
	require.ErrorIs(t, svc.SetTargetDzikir(ctx, u, id, 0), ErrTasbihValidasi)
}

func TestUbahDzikirMilikLain(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	a, b := seedUser(t, db, "user"), seedUser(t, db, "user")
	id, err := svc.BuatDzikir(ctx, a, models.TasbihDzikirInput{Latin: "Dzikir A", Target: 10})
	require.NoError(t, err)
	in := models.TasbihDzikirInput{Latin: "Diubah", Target: 10}
	require.ErrorIs(t, svc.UbahDzikir(ctx, b, id, in), ErrTasbihTidakDitemukan)
	require.ErrorIs(t, svc.UbahDzikir(ctx, a, dzikirIDBySlug(t, db, "subhanallah"), in), ErrTasbihTidakDitemukan)
	require.ErrorIs(t, svc.ArsipkanDzikir(ctx, b, id), ErrTasbihTidakDitemukan)
	require.ErrorIs(t, svc.SetTargetDzikir(ctx, b, id, 5), ErrTasbihTidakDitemukan)
	require.NoError(t, svc.UbahDzikir(ctx, a, id, in))
}

func TestBuatDzikirValidasi(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	for _, in := range []models.TasbihDzikirInput{
		{Latin: "   ", Target: 33},
		{Latin: strings.Repeat("a", 301), Target: 33},
		{Latin: "ok", Arab: strings.Repeat("ب", 501), Target: 33},
		{Latin: "ok", Target: 0},
		{Latin: "ok", Target: 100001},
	} {
		_, err := svc.BuatDzikir(ctx, u, in)
		require.ErrorIs(t, err, ErrTasbihValidasi)
	}
	for i := 0; i < 50; i++ {
		_, err := svc.BuatDzikir(ctx, u, models.TasbihDzikirInput{Latin: fmt.Sprintf("D%d", i), Target: 1})
		require.NoError(t, err)
	}
	_, err := svc.BuatDzikir(ctx, u, models.TasbihDzikirInput{Latin: "ke-51", Target: 1})
	require.ErrorIs(t, err, ErrTasbihValidasi)
}

func TestArsipkanDzikir(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	id, err := svc.BuatDzikir(ctx, u, models.TasbihDzikirInput{Latin: "Arsip", Target: 3})
	require.NoError(t, err)
	require.NoError(t, svc.ArsipkanDzikir(ctx, u, id))
	dz, _, err := svc.Katalog(ctx, u)
	require.NoError(t, err)
	require.Len(t, dz, 46)
	require.EqualValues(t, 1, countRows(t, db, "SELECT COUNT(*) FROM tasbih_dzikir WHERE id = ?", id))
}

func TestTargetHarianDefault(t *testing.T) {
	db, svc := newTasbihTestDB(t)
	ctx := context.Background()
	u := seedUser(t, db, "user")
	v, err := svc.TargetHarian(ctx, u)
	require.NoError(t, err)
	require.EqualValues(t, 100, v)
	require.NoError(t, svc.SetTargetHarian(ctx, u, 300))
	v, err = svc.TargetHarian(ctx, u)
	require.NoError(t, err)
	require.EqualValues(t, 300, v)
	require.ErrorIs(t, svc.SetTargetHarian(ctx, u, 0), ErrTasbihValidasi)
}
