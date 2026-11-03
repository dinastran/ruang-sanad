package services

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

// setupCRMMenuAgent is setupCRMAgent with Performa Iklan and Meta CAPI wired
// in and their clocks pinned to the lead fixture clock.
func setupCRMMenuAgent(t *testing.T) *crmAgentFixture {
	t.Helper()
	c := setupCRMAgent(t)
	ads := NewMetaAdsService(c.querier, MetaAdsConfig{})
	ads.now = func() time.Time { return c.clock }
	capi := NewMetaCAPIService(c.querier, MetaCAPIConfig{})
	capi.now = func() time.Time { return c.clock }
	c.agent.metaAds, c.agent.metaCAPI = ads, capi
	return c
}

func (c *crmAgentFixture) source(t *testing.T, nama string) int64 {
	t.Helper()
	res, err := c.db.Exec(`INSERT INTO lead_sources (nama, keterangan, is_aktif, url, created_at, updated_at) VALUES (?, '', 1, '', ?, ?)`, nama, c.clock, c.clock)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	return id
}

func decodeAI(t *testing.T, text string) map[string]any {
	t.Helper()
	out := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(text), &out), text)
	return out
}

func TestAISafeFirstNameHidesPhoneFallback(t *testing.T) {
	require.Equal(t, "", aiSafeFirstName("6281234567890"))
	require.Equal(t, "", aiSafeFirstName("0812 3456 7890"))
	require.Equal(t, "", aiSafeFirstName("+62"))
	require.Equal(t, "Fulanah", aiSafeFirstName("Fulanah binti Fulan"))
	require.Equal(t, "", aiSafeFirstName(""))
}

func TestAICRMMenuWriteToolsActAsAI(t *testing.T) {
	c := setupCRMMenuAgent(t)
	id := c.seedConversation(t)
	ref := fmt.Sprintf("L-%d", id)
	w := c.writer()

	// Stage: alasan is required, terverifikasi is off limits.
	text, isErr := c.call(t, w, "ubah_tahap_lead", map[string]any{"lead_ref": ref, "tahap": "form_dikirim", "alasan": "ok"})
	require.True(t, isErr)
	require.Contains(t, text, "alasan")
	text, isErr = c.call(t, w, "ubah_tahap_lead", map[string]any{"lead_ref": ref, "tahap": LeadStageTerverifikasi, "alasan": "Bukti transfer sudah dikirim lead"})
	require.True(t, isErr, text)
	text, isErr = c.call(t, w, "ubah_tahap_lead", map[string]any{"lead_ref": ref, "tahap": "form_dikirim", "alasan": "CS sudah mengirim link form pendaftaran"})
	require.False(t, isErr, text)
	require.Equal(t, "form_dikirim", c.lead(t).Stage)
	var actor, payload string
	require.NoError(t, c.db.QueryRow(`SELECT actor, payload FROM lead_events WHERE lead_id = ? AND type = 'stage.changed' ORDER BY id DESC LIMIT 1`, id).Scan(&actor, &payload))
	require.Equal(t, fmt.Sprintf("ai:%d", c.tokenID), actor)
	require.Contains(t, payload, `"stage_to":"form_dikirim"`, "CAPI reads stage_to")

	// PIC: super_admin token may assign anyone with a CRM role; others only themselves.
	text, isErr = c.call(t, w, "tugaskan_lead", map[string]any{"lead_ref": ref, "user_id": c.csID})
	require.False(t, isErr, text)
	require.EqualValues(t, c.csID, c.lead(t).AssignedUserID.Int64)
	text, isErr = c.call(t, w, "tugaskan_lead", map[string]any{"lead_ref": ref, "user_id": c.adminKelas1})
	require.True(t, isErr)
	require.Contains(t, text, "CS, Keuangan, atau super admin")
	csWriter := &models.AIPrincipal{UserID: c.csID, TokenID: c.tokenID, Role: "cs", CanWrite: true, BaseURL: "https://dash.test"}
	text, isErr = c.call(t, csWriter, "tugaskan_lead", map[string]any{"lead_ref": ref, "user_id": c.keuanganID})
	require.True(t, isErr)
	require.Contains(t, text, "pemilik token")

	// Lead source.
	src := c.source(t, "Iklan Ramadhan")
	text, isErr = c.call(t, w, "set_sumber_lead", map[string]any{"lead_ref": ref, "sumber_id": 9999, "alasan": "Lead menyebut iklan Ramadhan"})
	require.True(t, isErr)
	require.Contains(t, text, "tidak ditemukan")
	text, isErr = c.call(t, w, "set_sumber_lead", map[string]any{"lead_ref": ref, "sumber_id": src, "alasan": "Lead menyebut iklan Ramadhan di chat"})
	require.False(t, isErr, text)
	require.EqualValues(t, src, c.lead(t).LeadSourceID.Int64)
	require.Equal(t, LeadSourceByManual, c.lead(t).LeadSourceBy)

	// daftar_lead filters on the new fields.
	for args, want := range map[string]int{
		fmt.Sprintf(`{"sumber_id":%d}`, src):      1,
		`{"sumber_id":0}`:                         0,
		fmt.Sprintf(`{"pic_user_id":%d}`, c.csID): 1,
		`{"pic_user_id":0}`:                       0,
	} {
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(args), &m))
		text, isErr = c.call(t, w, "daftar_lead", m)
		require.False(t, isErr, text)
		require.EqualValues(t, want, decodeAI(t, text)["jumlah"], args)
	}
	text, isErr = c.call(t, w, "daftar_lead", map[string]any{"label": "tidak-ada"})
	require.True(t, isErr)
	require.Contains(t, text, "tidak ada")

	// detail_lead shows source, PIC and the AI-made timeline.
	text, isErr = c.call(t, w, "detail_lead", map[string]any{"lead_ref": ref})
	require.False(t, isErr, text)
	d := decodeAI(t, text)
	require.Equal(t, "Iklan Ramadhan", d["sumber"])
	require.Equal(t, "User cs", d["pic"])
	require.Contains(t, text, `"oleh":"ai"`)
	require.NotContains(t, text, "6281234567890")

	// Without write permission every new write tool is refused.
	reader := &models.AIPrincipal{UserID: c.adminID, Role: "super_admin", BaseURL: "https://dash.test"}
	for _, tool := range []string{"ubah_tahap_lead", "tugaskan_lead", "set_sumber_lead", "petakan_kampanye_iklan"} {
		text, isErr := c.call(t, reader, tool, map[string]any{"lead_ref": ref})
		require.True(t, isErr, tool)
		require.Contains(t, text, "tidak punya izin menulis", tool)
	}
}

func TestAICRMAdsAttributionAndRoles(t *testing.T) {
	c := setupCRMMenuAgent(t)
	id := c.seedConversation(t)
	src := c.source(t, "Iklan Tahsin")
	_, err := c.db.Exec(`UPDATE leads SET lead_source_id = ?, lead_source_by = 'manual' WHERE id = ?`, src, id)
	require.NoError(t, err)
	for _, r := range []struct {
		day, camp, ad string
		spend, chats  int64
	}{
		{"2026-10-23", "C1", "A1", 60000, 3}, {"2026-10-24", "C1", "A2", 40000, 2}, {"2026-10-24", "C2", "A3", 50000, 1},
		{"2026-09-20", "C1", "A1", 30000, 1}, // previous period
	} {
		_, err := c.db.Exec(`INSERT INTO meta_ad_insights_daily (tanggal, ad_account_id, campaign_id, campaign_name, adset_id, adset_name, ad_id, ad_name, spend, impressions, clicks, link_clicks, messaging_started, synced_at)
			VALUES (?, 'act_1', ?, ?, ?, 'Set '||?, ?, 'Iklan '||?, ?, 1000, 50, 20, ?, ?)`, r.day, r.camp, "Kampanye "+r.camp, "S"+r.camp, r.camp, r.ad, r.ad, r.spend, r.chats, c.clock)
		require.NoError(t, err)
	}
	w := c.writer()

	text, isErr := c.call(t, w, "petakan_kampanye_iklan", map[string]any{"kampanye_id": "C404", "sumber_id": src})
	require.True(t, isErr)
	require.Contains(t, text, "belum pernah tersinkron")
	text, isErr = c.call(t, w, "petakan_kampanye_iklan", map[string]any{"kampanye_id": "C1", "sumber_id": src})
	require.False(t, isErr, text)

	text, isErr = c.call(t, w, "performa_iklan", map[string]any{})
	require.False(t, isErr, text)
	perf := decodeAI(t, text)
	baris := perf["baris"].([]any)
	require.Len(t, baris, 2)
	first := baris[0].(map[string]any)
	require.Equal(t, "C1", first["id"])
	require.Equal(t, "Iklan Tahsin", first["sumber"])
	require.EqualValues(t, 100000, first["periode_ini"].(map[string]any)["spend"])
	require.EqualValues(t, 30000, first["periode_sebelumnya"].(map[string]any)["spend"])
	require.EqualValues(t, 20000, first["periode_ini"].(map[string]any)["biaya_per_chat"])
	require.EqualValues(t, 2, first["periode_ini"].(map[string]any)["ctr_persen"], "CTR uses link clicks like the dashboard")

	text, isErr = c.call(t, w, "performa_iklan", map[string]any{"level": "iklan", "kampanye_id": "C1"})
	require.False(t, isErr, text)
	require.Len(t, decodeAI(t, text)["baris"], 2)

	text, isErr = c.call(t, w, "atribusi_iklan", map[string]any{})
	require.False(t, isErr, text)
	at := decodeAI(t, text)
	ring := at["ringkasan"].(map[string]any)
	require.EqualValues(t, 100000, ring["spend_terpetakan"])
	require.EqualValues(t, 50000, ring["spend_belum_dipetakan"])
	require.EqualValues(t, 1, ring["lead_dari_sumber_berbayar"])
	row := at["per_sumber"].([]any)[0].(map[string]any)
	require.Equal(t, "Iklan Tahsin", row["sumber"])
	require.EqualValues(t, 100000, row["cpl"])
	require.Nil(t, row["biaya_per_santri"], "no santri yet")
	for _, raw := range at["per_sumber"].([]any) {
		if r := raw.(map[string]any); r["berbayar"] == false {
			require.Nil(t, r["cpl"], "organic source has no CPL, not Rp0")
		}
	}
	require.Len(t, at["kampanye_belum_dipetakan"], 1)
	require.Contains(t, at, "meta_capi", "super_admin also gets the CAPI summary")

	// Date range validation.
	text, isErr = c.call(t, w, "atribusi_iklan", map[string]any{"dari": "2026-10-01"})
	require.True(t, isErr)
	require.Contains(t, text, "dari dan sampai")

	// Role gates mirror the dashboard.
	keu := &models.AIPrincipal{UserID: c.keuanganID, Role: "keuangan", BaseURL: "https://dash.test"}
	cs := &models.AIPrincipal{UserID: c.csID, Role: "cs", BaseURL: "https://dash.test"}
	for _, tool := range []string{"performa_iklan", "atribusi_iklan", "status_meta_capi", "antrean_followup"} {
		text, isErr := c.call(t, keu, tool, map[string]any{})
		require.True(t, isErr, tool)
		require.Contains(t, text, "tidak memiliki akses", tool)
	}
	text, isErr = c.call(t, cs, "status_meta_capi", map[string]any{})
	require.True(t, isErr)
	require.Contains(t, text, "tidak memiliki akses")
	text, isErr = c.call(t, cs, "atribusi_iklan", map[string]any{})
	require.False(t, isErr, text)
	require.NotContains(t, decodeAI(t, text), "meta_capi", "cs does not see CAPI")
}

func TestAICRMMenuReadToolsNeverLeakContact(t *testing.T) {
	c := setupCRMMenuAgent(t)
	id := c.seedConversation(t)
	ref := fmt.Sprintf("L-%d", id)
	_, err := c.db.Exec(`UPDATE leads SET nama = '', wa_name = '', domisili = 'Cibubur' WHERE id = ?`, id) // display name falls back to the phone
	require.NoError(t, err)
	// Free-text bridge reason (may name people/places) and a staff-only enum reason.
	_, err = c.db.Exec(`INSERT INTO lead_events (lead_id, event_id, type, actor, payload, created_at) VALUES
		(?, 'e-h', 'handoff', '', '{"reason":"Ibu Siti Aminah minta bicara admin"}', ?),
		(?, 'e-s', 'stage.changed', 'user:1', '{"stage_from":"baru","stage_to":"discovery","alasan":"tinggal di Cibubur"}', ?)`, id, c.clock, id, c.clock)
	require.NoError(t, err)
	_, err = c.db.Exec(`INSERT INTO meta_capi_events (lead_id, event_name, event_id, event_time, status, attempts, test_mode, last_error, next_attempt_at, created_at)
		VALUES (?, 'Lead', 'ev1', ?, 'failed', 2, 0, 'invalid phone 6281234567890', ?, ?)`, id, c.clock, c.clock, c.clock)
	require.NoError(t, err)
	_, err = c.db.Exec(`INSERT INTO lead_followups (lead_id, step, scheduled_at, draft_text, reason, status, created_by_kind, created_at, updated_at)
		VALUES (?, 'd1', ?, 'Kak di Cibubur, boleh hubungi 0812-3456-7890 ya', '', 'pending_approval', 'ai', ?, ?)`, id, c.clock.Add(26*time.Hour), c.clock, c.clock)
	require.NoError(t, err)

	for tool, args := range map[string]map[string]any{
		"analisis_crm": {}, "prioritas_lead": {}, "antrean_followup": {}, "daftar_sumber_lead": {},
		"referensi_crm": {}, "status_meta_capi": {"max_event": 50}, "atribusi_iklan": {}, "performa_iklan": {},
		"detail_lead": {"lead_ref": ref},
	} {
		text, isErr := c.call(t, c.writer(), tool, args)
		require.False(t, isErr, "%s: %s", tool, text)
		for _, leak := range []string{"6281234567890", "0812-3456-7890", "s.whatsapp.net", "Cibubur", "Siti Aminah"} {
			require.NotContains(t, text, leak, "%s leaked %q", tool, leak)
		}
		if tool == "antrean_followup" {
			require.Len(t, decodeAI(t, text)["menunggu_persetujuan"], 1)
			require.Contains(t, text, aiMaskPhone)
		}
		if tool == "status_meta_capi" {
			require.Contains(t, text, ref)
		}
	}
}
