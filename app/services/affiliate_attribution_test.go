package services

import (
	"strings"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func (f *affiliateFixture) leadAffiliate(t *testing.T) (int64, string) {
	t.Helper()
	var aff int64
	var sumber string
	err := f.db.QueryRow(`SELECT la.affiliate_id, la.sumber FROM lead_affiliates la JOIN leads l ON l.id = la.lead_id WHERE l.wa_chat_id = ?`, testChatID).Scan(&aff, &sumber)
	if err != nil {
		return 0, ""
	}
	return aff, sumber
}

func (f *affiliateFixture) submitWith(t *testing.T, cookie, kode string) {
	t.Helper()
	setupJadwal(t, f.registrationFixture)
	in := validSubmit("081234567890")
	in.KodeReferal = kode
	in.FormToken = f.reg.FormToken("tahsin-reguler")
	f.clock = f.clock.Add(10 * time.Second)
	_, err := f.reg.Submit(t.Context(), "tahsin-reguler", in, models.RequestMeta{IP: "1.2.3.4", UserAgent: "t", AffCookie: cookie})
	require.NoError(t, err)
}

func TestAffCookieRoundTripAndExpiry(t *testing.T) {
	f := setupAffiliateFixture(t)
	f.external(t, "Ahmad", "081200000001")
	c := f.aff.AffCookie("AHMAD")
	kode, at, ok := f.aff.ParseAffCookie(t.Context(), c)
	require.True(t, ok)
	require.Equal(t, "AHMAD", kode)
	require.True(t, at.Equal(f.aff.nowUTC()))
	_, _, ok = f.aff.ParseAffCookie(t.Context(), strings.Replace(c, "AHMAD", "BUDI", 1))
	require.False(t, ok, "tanda tangan salah")
	f.clock = f.clock.Add(61 * 24 * time.Hour)
	_, _, ok = f.aff.ParseAffCookie(t.Context(), c)
	require.False(t, ok, "lewat masa berlaku 60 hari")
}

func TestTrackClickFirstClickKeepsCookie(t *testing.T) {
	f := setupAffiliateFixture(t)
	f.external(t, "Ahmad", "081200000001")
	f.external(t, "Budi", "081200000002")
	r1, err := f.aff.TrackClick(t.Context(), "ahmad", "1.1.1.1", "ua", "")
	require.NoError(t, err)
	require.NotEmpty(t, r1.Cookie)
	r2, err := f.aff.TrackClick(t.Context(), "BUDI", "1.1.1.1", "ua", r1.Cookie)
	require.NoError(t, err)
	require.Empty(t, r2.Cookie, "cookie affiliate pertama yang masih berlaku tidak ditimpa")
	_, err = f.aff.TrackClick(t.Context(), "BUDI", "1.1.1.1", "ua", r1.Cookie)
	require.NoError(t, err)
	require.Equal(t, 2, f.count(t, `SELECT COALESCE(SUM(klik),0) FROM affiliate_clicks`), "klik dedup per hari")
	r3, err := f.aff.TrackClick(t.Context(), "TIDAKADA", "1.1.1.1", "ua", "")
	require.NoError(t, err)
	require.False(t, r3.Found)
}

func TestSubmitLocksAffiliateFromCookie(t *testing.T) {
	f := setupAffiliateFixture(t)
	a := f.external(t, "Ahmad", "081200000001")
	f.submitWith(t, f.aff.AffCookie("AHMAD"), "")
	got, src := f.leadAffiliate(t)
	require.Equal(t, a, got)
	require.Equal(t, "cookie", src)
}

func TestSubmitKodeReferalAndFirstClick(t *testing.T) {
	f := setupAffiliateFixture(t)
	a := f.external(t, "Ahmad", "081200000001")
	f.external(t, "Budi", "081200000002")
	f.submitWith(t, "", "ahmad")
	got, src := f.leadAffiliate(t)
	require.Equal(t, a, got)
	require.Equal(t, "kode_form", src)
	f.submitWith(t, f.aff.AffCookie("BUDI"), "BUDI")
	got, _ = f.leadAffiliate(t)
	require.Equal(t, a, got, "affiliate pertama tetap")
}

func TestSubmitSelfReferralIgnored(t *testing.T) {
	f := setupAffiliateFixture(t)
	f.external(t, "Ahmad", "081234567890") // same number as the registrant
	f.submitWith(t, "", "AHMAD")
	got, _ := f.leadAffiliate(t)
	require.Zero(t, got)
}

func TestSubmitInactiveAffiliateIgnored(t *testing.T) {
	f := setupAffiliateFixture(t)
	a := f.external(t, "Ahmad", "081200000001")
	require.NoError(t, f.aff.SetStatus(t.Context(), a, "nonaktif", f.adminID))
	f.submitWith(t, "", "AHMAD")
	got, _ := f.leadAffiliate(t)
	require.Zero(t, got)
}

func TestIngestWARef(t *testing.T) {
	f := setupAffiliateFixture(t)
	a := f.external(t, "Ahmad", "081200000001")
	f.service.WithAffiliates(f.aff)
	f.ingest(t, leadEvent("w1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "W1", "body": "halo kak"}))
	f.ingest(t, leadEvent("w2", "message.in", "2026-10-24T03:05:00Z", map[string]string{"wa_message_id": "W2", "body": "Saya mau daftar (ref: ahmad)"}))
	got, src := f.leadAffiliate(t)
	require.Equal(t, a, got, "ref di pesan kedua tetap dikenali")
	require.Equal(t, "kode_wa", src)
}

func TestManualLeadAffiliate(t *testing.T) {
	f := setupAffiliateFixture(t)
	a := f.external(t, "Ahmad", "081200000001")
	b := f.external(t, "Budi", "081200000002")
	f.submitWith(t, "", "AHMAD")
	l, err := f.querier.GetLeadByChatID(t.Context(), testChatID)
	require.NoError(t, err)
	require.NoError(t, f.aff.SetLeadAffiliate(t.Context(), l.ID, b, f.csID))
	got, src := f.leadAffiliate(t)
	require.Equal(t, b, got)
	require.Equal(t, "manual", src)
	require.NoError(t, f.aff.SetLeadAffiliate(t.Context(), l.ID, 0, f.csID))
	got, _ = f.leadAffiliate(t)
	require.Zero(t, got)
	require.Equal(t, 2, f.count(t, `SELECT COUNT(*) FROM lead_events WHERE lead_id = ? AND type = 'affiliate.changed'`, l.ID))
	_ = a
}

func setupJadwal(t *testing.T, f *registrationFixture) {
	t.Helper()
	_, err := f.db.Exec(`INSERT OR IGNORE INTO jadwal_penawaran (kode, hari, jam_mulai, jam_selesai, kuota_per_gender, is_buka, periode_mulai, created_at, updated_at)
		VALUES (?, '1,4', '19:30', '20:30', 10, 1, '2026-01-01 00:00:00', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, testJadwal)
	require.NoError(t, err)
	if f.count(t, `SELECT COUNT(*) FROM registration_links WHERE slug = 'tahsin-reguler'`) == 0 {
		_, err = f.reg.CreateLink(t.Context(), f.linkReq("tahsin-reguler"), f.adminID)
		require.NoError(t, err)
	}
}
