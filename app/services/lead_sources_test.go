package services

import (
	"strconv"
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAndMatchLeadSource(t *testing.T) {
	require.Equal(t, "assalamu alaikum saya ingin tanya", NormalizeLeadSourceText("  Assalamu'alaikum,   saya ingin TANYA?! "))
	patterns := []queries.ListActiveLeadSourcePatternsRow{
		{SourceID: 1, TeksNormal: "kelas reguler tahsin"},
		{SourceID: 2, TeksNormal: "kelas reguler tahsin ruang sanad"},
		{SourceID: 3, TeksNormal: "info selengkapnya tentang ini"},
	}
	require.EqualValues(t, 2, matchLeadSource(patterns, "Assalamu'alaikum, saya ingin tanya tentang Kelas Reguler Tahsin Ruang Sanad."), "longest trigger wins")
	require.EqualValues(t, 1, matchLeadSource(patterns, "kelas reguler tahsin masih buka?"))
	require.EqualValues(t, 3, matchLeadSource(patterns, "Halo! Bisakah saya mendapatkan info selengkapnya tentang ini?"))
	require.Zero(t, matchLeadSource(patterns, "kelas reguler tahsinnya kapan"), "whole words only")
	require.Zero(t, matchLeadSource(patterns, "halo kak"))
}

func sourceIDByName(t *testing.T, f *leadFixture, nama string) int64 {
	t.Helper()
	var id int64
	require.NoError(t, f.db.QueryRow(`SELECT id FROM lead_sources WHERE nama = ?`, nama).Scan(&id))
	return id
}

func TestLeadSourceDetectedFromFirstMessageOnly(t *testing.T) {
	f := setupLeadFixture(t)
	reguler := sourceIDByName(t, f, "LP Kelas Reguler Tahsin")

	f.ingest(t, leadEvent("s1", "message.in", "2026-10-24T03:00:00Z", map[string]string{
		"wa_message_id": "S1", "body": "Assalamu'alaikum, saya ingin tanya tentang Kelas Reguler Tahsin Ruang Sanad.",
	}))
	l := f.lead(t)
	require.EqualValues(t, reguler, l.LeadSourceID.Int64)
	require.Equal(t, LeadSourceByAuto, l.LeadSourceBy)

	// A second lead whose first message matches nothing stays without a
	// source, even if a later message contains a trigger text.
	other := leadEvent("s2", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "S2", "body": "halo kak"})
	other.ChatID, other.Phone = "6289999999999@s.whatsapp.net", "6289999999999"
	later := leadEvent("s3", "message.in", "2026-10-24T03:05:00Z", map[string]string{"wa_message_id": "S3", "body": "Bisakah saya mendapatkan info selengkapnya tentang ini?"})
	later.ChatID, later.Phone = other.ChatID, other.Phone
	f.ingest(t, other, later)
	l2, err := f.querier.GetLeadByChatID(t.Context(), other.ChatID)
	require.NoError(t, err)
	require.False(t, l2.LeadSourceID.Valid)
	require.Equal(t, LeadSourceByAuto, l2.LeadSourceBy)

	// Rematch uses the first message, so it still finds nothing for lead 2.
	n, err := f.service.RematchSources(t.Context())
	require.NoError(t, err)
	require.Zero(t, n)
}

func TestLeadSourceCatalogRematchAndManual(t *testing.T) {
	f := setupLeadFixture(t)
	ctx := t.Context()
	f.ingest(t, leadEvent("m1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "M1", "body": "Saya mau daftar kelas tahsin online dewasa"}))
	l := f.lead(t)
	require.False(t, l.LeadSourceID.Valid)

	id, err := f.service.CreateSource(ctx, models.LeadSourceRequest{Nama: "  LP   Tahsin Dewasa ", IsAktif: true})
	require.NoError(t, err)
	_, err = f.service.CreateSource(ctx, models.LeadSourceRequest{Nama: "LP Tahsin Dewasa", IsAktif: true})
	require.ErrorContains(t, err, "sudah ada")

	require.ErrorContains(t, f.service.AddSourcePattern(ctx, id, "halo kak"), "terlalu pendek")
	require.NoError(t, f.service.AddSourcePattern(ctx, id, "kelas tahsin online dewasa"))
	require.ErrorContains(t, f.service.AddSourcePattern(ctx, id, "Kelas Tahsin, Online Dewasa!"), "sudah ada")
	reguler := sourceIDByName(t, f, "LP Kelas Reguler Tahsin")
	require.ErrorContains(t, f.service.AddSourcePattern(ctx, reguler, "kelas tahsin online dewasa"), "LP Tahsin Dewasa")

	n, err := f.service.RematchSources(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.EqualValues(t, id, f.lead(t).LeadSourceID.Int64)

	// A manual choice (including "tanpa sumber") is never overwritten.
	require.NoError(t, f.service.SetLeadSourceManual(ctx, l.ID, f.csID, 0))
	got := f.lead(t)
	require.False(t, got.LeadSourceID.Valid)
	require.Equal(t, LeadSourceByManual, got.LeadSourceBy)
	n, err = f.service.RematchSources(ctx)
	require.NoError(t, err)
	require.Zero(t, n)
	require.ErrorIs(t, f.service.SetLeadSourceManual(ctx, l.ID, f.csID, 999), ErrLeadSourceNotFound)

	require.NoError(t, f.service.SetLeadSourceManual(ctx, l.ID, f.csID, id))
	d, err := f.service.Detail(ctx, l.ID)
	require.NoError(t, err)
	require.Equal(t, "LP Tahsin Dewasa", d.LeadSourceNama)
	require.Equal(t, LeadSourceByManual, d.LeadSourceBy)
	require.Len(t, d.SourceOptions, 3)

	// A used source cannot be deleted, only deactivated; inactive sources do
	// not match new leads.
	require.ErrorContains(t, f.service.DeleteSource(ctx, id), "nonaktifkan")
	require.NoError(t, f.service.UpdateSource(ctx, id, models.LeadSourceRequest{Nama: "LP Tahsin Dewasa", IsAktif: false}))
	patterns, err := f.querier.ListActiveLeadSourcePatterns(ctx)
	require.NoError(t, err)
	require.Zero(t, matchLeadSource(patterns, "kelas tahsin online dewasa"))

	list, err := f.service.Sources(ctx)
	require.NoError(t, err)
	for _, s := range list {
		if s.ID == id {
			require.EqualValues(t, 1, s.TotalLeads)
			require.Len(t, s.Patterns, 1)
		}
	}
}

func TestLeadSourceListFilterAndDashboard(t *testing.T) {
	f := setupLeadFixture(t)
	reguler := sourceIDByName(t, f, "LP Kelas Reguler Tahsin")
	meta := sourceIDByName(t, f, "Iklan Meta (teks default)")
	for i, body := range []string{
		"Assalamu'alaikum, saya ingin tanya tentang Kelas Reguler Tahsin Ruang Sanad.",
		"Halo! Bisakah saya mendapatkan info selengkapnya tentang ini?",
		"Halo! Bisakah saya mendapatkan info selengkapnya tentang ini?",
		"assalamualaikum",
	} {
		ev := leadEvent("d"+string(rune('a'+i)), "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "D" + string(rune('a'+i)), "body": body})
		ev.ChatID = "62811" + string(rune('0'+i)) + "@s.whatsapp.net"
		ev.Phone = "62811" + string(rune('0'+i))
		f.ingest(t, ev)
	}

	byMeta, err := f.service.List(t.Context(), models.LeadListParams{Status: "all", Source: strconv.FormatInt(meta, 10)})
	require.NoError(t, err)
	require.EqualValues(t, 2, byMeta.Total)
	none, err := f.service.List(t.Context(), models.LeadListParams{Status: "all", Source: "none"})
	require.NoError(t, err)
	require.EqualValues(t, 1, none.Total)

	d, err := f.service.Dashboard(t.Context(), "2026-10-18", "2026-10-24")
	require.NoError(t, err)
	require.Equal(t, []models.LeadSourceStat{
		{ID: meta, Nama: "Iklan Meta (teks default)", Leads: 2, Open: 2},
		{ID: reguler, Nama: "LP Kelas Reguler Tahsin", Leads: 1, Open: 1},
		{ID: 0, Nama: "Tanpa sumber", Leads: 1, Open: 1},
	}, d.Sources)
}
