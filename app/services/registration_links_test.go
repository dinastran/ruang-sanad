package services

import (
	"errors"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

type registrationFixture struct {
	*leadFixture
	reg      *RegistrationService
	produkID int64
	sourceID int64
	adminID  int64
}

func setupRegistrationFixture(t *testing.T) *registrationFixture {
	t.Helper()
	lf := setupLeadFixture(t)
	f := &registrationFixture{leadFixture: lf}
	f.reg = NewRegistrationService(lf.querier, lf.service, NewJadwalPenawaranService(lf.querier), []byte("test-secret"))
	f.reg.now = func() time.Time { return lf.clock }
	f.reg.proofDir = t.TempDir()

	res, err := lf.db.Exec(`INSERT INTO produk (nama, kategori, harga) VALUES ('Tahsin Reguler', 'program', 150000)`)
	require.NoError(t, err)
	f.produkID, _ = res.LastInsertId()
	_, err = lf.db.Exec(`UPDATE lead_programs SET produk_id = ? WHERE id = 1`, f.produkID)
	require.NoError(t, err)
	require.NoError(t, lf.db.QueryRow(`SELECT id FROM lead_sources ORDER BY id LIMIT 1`).Scan(&f.sourceID))
	f.adminID = lf.csID
	require.NoError(t, f.reg.SaveSettings(t.Context(), models.RegistrationSettings{
		CsWaNumber: "6281100000000", PaymentExpiryHours: 2, BankManualInfo: "BSI 123 a.n. Yayasan",
	}, f.adminID))
	return f
}

func (f *registrationFixture) linkReq(slug string) models.RegistrationLinkRequest {
	return models.RegistrationLinkRequest{
		Slug: slug, Judul: "Daftar Tahsin", ProgramID: 1, LeadSourceID: f.sourceID,
		Mode: "short", AfterSubmit: "wa", IzinkanManual: true, IsAktif: true,
	}
}

func (f *registrationFixture) addVarian(t *testing.T, nama string, harga int64, paket string) int64 {
	t.Helper()
	res, err := f.db.Exec(`INSERT INTO produk_varian (produk_id, nama, harga, paket_kode, urutan, is_aktif, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, f.produkID, nama, harga, paket)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	return id
}

func TestCreateLinkValidation(t *testing.T) {
	f := setupRegistrationFixture(t)
	ctx := t.Context()

	bad := f.linkReq("Tahsin Reguler")
	_, err := f.reg.CreateLink(ctx, bad, f.adminID)
	require.ErrorContains(t, err, "slug")

	noSource := f.linkReq("tahsin-a")
	noSource.LeadSourceID = 0
	_, err = f.reg.CreateLink(ctx, noSource, f.adminID)
	require.ErrorContains(t, err, "sumber")

	_, err = f.db.Exec(`INSERT INTO lead_programs (nama, jenis) VALUES ('Tanpa Produk', 'program')`)
	require.NoError(t, err)
	var noProdukProgram int64
	require.NoError(t, f.db.QueryRow(`SELECT id FROM lead_programs WHERE nama = 'Tanpa Produk'`).Scan(&noProdukProgram))
	np := f.linkReq("tahsin-b")
	np.ProgramID = noProdukProgram
	_, err = f.reg.CreateLink(ctx, np, f.adminID)
	require.ErrorContains(t, err, "belum terhubung ke produk")

	require.NoError(t, f.reg.SaveSettings(ctx, models.RegistrationSettings{PaymentExpiryHours: 2}, f.adminID))
	wa := f.linkReq("tahsin-c")
	wa.Mode = "wa"
	_, err = f.reg.CreateLink(ctx, wa, f.adminID)
	require.ErrorContains(t, err, "nomor WA CS")
	require.NoError(t, f.reg.SaveSettings(ctx, models.RegistrationSettings{CsWaNumber: "081100000000", PaymentExpiryHours: 2}, f.adminID))

	id, err := f.reg.CreateLink(ctx, f.linkReq("tahsin-reguler"), f.adminID)
	require.NoError(t, err)
	require.Positive(t, id)
}

func TestResolveAmount(t *testing.T) {
	f := setupRegistrationFixture(t)
	ctx := t.Context()
	id, err := f.reg.CreateLink(ctx, f.linkReq("tahsin-reguler"), f.adminID)
	require.NoError(t, err)
	link, err := f.querier.GetRegistrationLink(ctx, id)
	require.NoError(t, err)

	amount, v, err := f.reg.ResolveAmount(ctx, link, 0)
	require.NoError(t, err)
	require.EqualValues(t, 150000, amount, "tanpa varian = harga produk")
	require.Nil(t, v)

	v1 := f.addVarian(t, "1x / pekan", 200000, "1x")
	v2 := f.addVarian(t, "2x / pekan", 350000, "2x")
	_, _, err = f.reg.ResolveAmount(ctx, link, 0)
	require.Error(t, err, "varian wajib dipilih bila ada varian aktif")
	amount, v, err = f.reg.ResolveAmount(ctx, link, v2)
	require.NoError(t, err)
	require.EqualValues(t, 350000, amount)
	require.Equal(t, "2x", v.PaketKode)

	res, err := f.db.Exec(`INSERT INTO produk (nama, kategori, harga) VALUES ('Lain', 'program', 1)`)
	require.NoError(t, err)
	other, _ := res.LastInsertId()
	res, err = f.db.Exec(`INSERT INTO produk_varian (produk_id, nama, harga, is_aktif, created_at, updated_at) VALUES (?, 'x', 5, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, other)
	require.NoError(t, err)
	foreign, _ := res.LastInsertId()
	_, _, err = f.reg.ResolveAmount(ctx, link, foreign)
	require.Error(t, err, "varian produk lain ditolak")

	req := f.linkReq("tahsin-reguler")
	override := int64(99000)
	req.NominalOverride = &override
	require.NoError(t, f.reg.UpdateLink(ctx, id, req))
	link, err = f.querier.GetRegistrationLink(ctx, id)
	require.NoError(t, err)
	amount, _, err = f.reg.ResolveAmount(ctx, link, v1)
	require.NoError(t, err)
	require.EqualValues(t, 99000, amount)
}

func TestPublicLinkInactive(t *testing.T) {
	f := setupRegistrationFixture(t)
	ctx := t.Context()
	id, err := f.reg.CreateLink(ctx, f.linkReq("tahsin-reguler"), f.adminID)
	require.NoError(t, err)

	pub, err := f.reg.PublicLink(ctx, "tahsin-reguler")
	require.NoError(t, err)
	require.True(t, pub.ButuhJadwal, "Tahsin Reguler = program reguler")
	require.EqualValues(t, 150000, pub.Harga)

	_, err = f.reg.PublicLink(ctx, "tidak-ada")
	require.True(t, errors.Is(err, ErrLinkTidakAktif))

	_, err = f.db.Exec(`UPDATE produk SET is_aktif = 0 WHERE id = ?`, f.produkID)
	require.NoError(t, err)
	_, err = f.reg.PublicLink(ctx, "tahsin-reguler")
	require.True(t, errors.Is(err, ErrLinkTidakAktif), "produk nonaktif")
	_, err = f.db.Exec(`UPDATE produk SET is_aktif = 1 WHERE id = ?`, f.produkID)
	require.NoError(t, err)

	_, err = f.db.Exec(`UPDATE lead_programs SET is_aktif = 0 WHERE id = 1`)
	require.NoError(t, err)
	_, err = f.reg.PublicLink(ctx, "tahsin-reguler")
	require.True(t, errors.Is(err, ErrLinkTidakAktif), "program nonaktif")
	_, err = f.db.Exec(`UPDATE lead_programs SET is_aktif = 1 WHERE id = 1`)
	require.NoError(t, err)

	_, err = f.db.Exec(`UPDATE registration_links SET is_aktif = 0 WHERE id = ?`, id)
	require.NoError(t, err)
	_, err = f.reg.PublicLink(ctx, "tahsin-reguler")
	require.True(t, errors.Is(err, ErrLinkTidakAktif), "link nonaktif")
}

func TestListLinksStats(t *testing.T) {
	f := setupRegistrationFixture(t)
	ctx := t.Context()
	id, err := f.reg.CreateLink(ctx, f.linkReq("tahsin-reguler"), f.adminID)
	require.NoError(t, err)
	today := f.clock.In(wib)
	for _, d := range []int{0, 1, 2} {
		_, err = f.db.Exec(`INSERT INTO registration_link_stats (link_id, tanggal, klik, submit, bayar) VALUES (?, ?, 10, 2, 1)`,
			id, today.AddDate(0, 0, -d).Format(time.DateOnly))
		require.NoError(t, err)
	}
	_, err = f.db.Exec(`INSERT INTO registration_link_stats (link_id, tanggal, klik) VALUES (?, ?, 500)`, id, today.AddDate(0, 0, -40).Format(time.DateOnly))
	require.NoError(t, err)

	rows, err := f.reg.ListLinks(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.EqualValues(t, 30, rows[0].Klik)
	require.EqualValues(t, 6, rows[0].Submit)
	require.EqualValues(t, 3, rows[0].Bayar)
}

func TestCountClickDedup(t *testing.T) {
	f := setupRegistrationFixture(t)
	id, err := f.reg.CreateLink(t.Context(), f.linkReq("tahsin-reguler"), f.adminID)
	require.NoError(t, err)
	require.NoError(t, f.reg.CountClick(t.Context(), id, "1.1.1.1", "ua-a"))
	require.NoError(t, f.reg.CountClick(t.Context(), id, "1.1.1.1", "ua-a"))
	require.NoError(t, f.reg.CountClick(t.Context(), id, "1.1.1.1", "ua-b"))
	var klik int
	require.NoError(t, f.db.QueryRow(`SELECT SUM(klik) FROM registration_link_stats WHERE link_id = ?`, id).Scan(&klik))
	require.Equal(t, 2, klik)
	f.clock = f.clock.Add(24 * time.Hour)
	require.NoError(t, f.reg.CountClick(t.Context(), id, "1.1.1.1", "ua-a"))
	require.NoError(t, f.db.QueryRow(`SELECT SUM(klik) FROM registration_link_stats WHERE link_id = ?`, id).Scan(&klik))
	require.Equal(t, 3, klik, "hari baru dihitung lagi")
}
