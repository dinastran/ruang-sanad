package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

type crmAgentFixture struct {
	*leadFixture
	agent   *AIAgentService
	adminID int64
	tokenID int64
}

func setupCRMAgent(t *testing.T) *crmAgentFixture {
	t.Helper()
	f := setupLeadFixture(t)
	res, err := f.db.Exec(`INSERT INTO users (email, name, role) VALUES ('boss@example.com', 'Boss', 'super_admin')`)
	require.NoError(t, err)
	adminID, _ := res.LastInsertId()
	res, err = f.db.Exec(`INSERT INTO ai_tokens (user_id, nama, token_hash, prefix, expires_at, can_write) VALUES (?, 'Hermes', 'hash', 'rsai_x', '2027-01-01 00:00:00', 1)`, adminID)
	require.NoError(t, err)
	tokenID, _ := res.LastInsertId()
	agent := NewAIAgentService(f.querier, nil, nil, nil, nil, nil, f.service)
	agent.now = func() time.Time { return f.clock }
	return &crmAgentFixture{leadFixture: f, agent: agent, adminID: adminID, tokenID: tokenID}
}

func (c *crmAgentFixture) writer() *models.AIPrincipal {
	return &models.AIPrincipal{UserID: c.adminID, TokenID: c.tokenID, UserName: "Boss", Role: "super_admin", CanWrite: true, BaseURL: "https://dash.test"}
}

func (c *crmAgentFixture) call(t *testing.T, p *models.AIPrincipal, tool string, args any) (string, bool) {
	t.Helper()
	raw, err := json.Marshal(args)
	require.NoError(t, err)
	r := c.agent.Call(context.Background(), p, tool, raw)
	return r.Text, r.IsError
}

// seedConversation creates a lead with an inbound question and one AI reply.
func (c *crmAgentFixture) seedConversation(t *testing.T) int64 {
	t.Helper()
	c.ingest(t,
		leadEvent("evt_c1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "M1", "body": "Assalamu'alaikum, mau tanya kelas tahsin"}),
		leadEvent("evt_c2", "message.out", "2026-10-24T03:01:00Z", map[string]string{"wa_message_id": "M2", "body": "Wa'alaikumussalam Kak, ada kelas 1x dan 2x sepekan.", "sender": "ai"}),
	)
	return c.lead(t).ID
}

func validAnalysis(ref string, until int64) map[string]any {
	return map[string]any{
		"lead_ref": ref, "summary": "Mencari kelas tahsin untuk diri sendiri, belum memilih paket.",
		"persona": map[string]any{
			"untuk_siapa": "diri_sendiri", "level": "pemula", "tujuan": []string{"tajwid", "makhraj"},
			"preferensi_waktu": "malam", "paket_diminati": "belum_jelas", "hambatan": []string{"jadwal"},
			"gaya_komunikasi": "singkat", "catatan": "Responsif di malam hari.",
		},
		"interest_score": 70, "main_objection": "jadwal", "suggested_stage": "discovery",
		"suggested_next_action": "Tawarkan jadwal malam.", "confidence": "sedang",
		"analyzed_until_message_id": until,
	}
}

func TestAICRMWriteToolsNeedWritePermission(t *testing.T) {
	c := setupCRMAgent(t)
	id := c.seedConversation(t)
	ref := fmt.Sprintf("L-%d", id)

	reader := &models.AIPrincipal{UserID: c.csID, Role: "cs", BaseURL: "https://dash.test"}
	names := func(ts []AITool) []string {
		out := []string{}
		for _, tl := range ts {
			out = append(out, tl.Name)
		}
		return out
	}
	require.Subset(t, names(c.agent.ToolsFor("cs", false)), []string{"daftar_lead", "detail_lead", "laporan_funnel"})
	require.NotContains(t, names(c.agent.ToolsFor("cs", false)), "simpan_analisis_lead")
	require.Contains(t, names(c.agent.ToolsFor("cs", true)), "usulkan_followup")
	require.NotContains(t, names(c.agent.ToolsFor("admin_kelas", true)), "daftar_lead", "admin_kelas has no CRM access")

	for _, tool := range []string{"simpan_analisis_lead", "usulkan_followup", "tambah_label_lead"} {
		text, isErr := c.call(t, reader, tool, map[string]any{"lead_ref": ref})
		require.True(t, isErr, tool)
		require.Contains(t, text, "tidak punya izin menulis", tool)
	}
	// Write calls (allowed or not) are audited.
	text, isErr := c.call(t, c.writer(), "simpan_analisis_lead", validAnalysis(ref, 2))
	require.False(t, isErr, text)
	var n int
	require.NoError(t, c.db.QueryRow(`SELECT COUNT(*) FROM ai_audit_log WHERE tool IN ('simpan_analisis_lead','usulkan_followup','tambah_label_lead')`).Scan(&n))
	require.Equal(t, 4, n)
	require.NoError(t, c.db.QueryRow(`SELECT COUNT(*) FROM ai_audit_log WHERE tool = 'simpan_analisis_lead' AND status = 'ok' AND token_id = ?`, c.tokenID).Scan(&n))
	require.Equal(t, 1, n)
}

func TestAITokenWritePermissionOnlyForSuperAdmin(t *testing.T) {
	c := setupCRMAgent(t)
	tokens := NewAITokenService(c.querier)
	tokens.now = func() time.Time { return c.clock }
	_, err := tokens.Create(c.csID, "cs", "Hermes", 30, true)
	require.ErrorIs(t, err, ErrAITokenWriteDitolak)
	created, err := tokens.Create(c.adminID, "super_admin", "Hermes", 30, true)
	require.NoError(t, err)
	p, err := tokens.Authenticate(context.Background(), created.Plaintext)
	require.NoError(t, err)
	require.True(t, p.CanWrite)
	plain, err := tokens.Create(c.adminID, "super_admin", "Claude", 30, false)
	require.NoError(t, err)
	p, err = tokens.Authenticate(context.Background(), plain.Plaintext)
	require.NoError(t, err)
	require.False(t, p.CanWrite)
}

func TestAICRMDetailLeadNeverLeaksContact(t *testing.T) {
	c := setupCRMAgent(t)
	c.ingest(t,
		leadEvent("evt_d1", "message.in", "2026-10-24T03:00:00Z", map[string]string{"wa_message_id": "D1", "body": "Nomor suami saya 0812-3456-7890, email fulanah@mail.com, atau wa.me/6281299998888"}),
		leadEvent("evt_d2", "form.sent", "2026-10-24T03:05:00Z", map[string]string{}),
		leadEvent("evt_d3", "message.in", "2026-10-24T03:10:00Z", map[string]string{"wa_message_id": "D3",
			"body": "Nama: Fulanah binti Fulan\nUsia: 25 th\nDomisili: Bekasi Timur\nIkhwan/Akhwat: Akhwat\nPilihan kelas: 1x"}),
		leadEvent("evt_d4", "message.in", "2026-10-24T03:12:00Z", map[string]string{"wa_message_id": "D4", "body": "Saya di bekasi timur dekat stasiun"}),
	)
	l := c.lead(t)
	require.Equal(t, "Bekasi Timur", l.Domisili)

	for _, p := range []*models.AIPrincipal{c.writer(), {UserID: c.csID, Role: "cs", BaseURL: "https://dash.test"}} {
		text, isErr := c.call(t, p, "detail_lead", map[string]any{"lead_ref": fmt.Sprintf("L-%d", l.ID)})
		require.False(t, isErr, text)
		lower := strings.ToLower(text)
		for _, leak := range []string{"bekasi", "0812", "3456", "6281234567890", "s.whatsapp.net", "fulanah@mail.com", "wa.me", "6281299998888", "domisili\":"} {
			require.NotContains(t, lower, leak)
		}
		require.Contains(t, text, "[domisili disembunyikan]")
		require.Contains(t, text, "[nomor disembunyikan]")
		require.Contains(t, text, "\"nama_depan\":\"Fulanah\"")

		list, isErr := c.call(t, p, "daftar_lead", map[string]any{})
		require.False(t, isErr, list)
		require.NotContains(t, list, "6281234567890")
		require.NotContains(t, list, "binti")
	}
}

func TestAICRMAnalysisNeverChangesStage(t *testing.T) {
	c := setupCRMAgent(t)
	id := c.seedConversation(t)
	ref := fmt.Sprintf("L-%d", id)
	before := c.lead(t)

	list, _ := c.call(t, c.writer(), "daftar_lead", map[string]any{"butuh_analisis": true})
	require.Contains(t, list, ref)

	args := validAnalysis(ref, 2)
	args["suggested_stage"] = "rekomendasi"
	text, isErr := c.call(t, c.writer(), "simpan_analisis_lead", args)
	require.False(t, isErr, text)

	after := c.lead(t)
	require.Equal(t, before.Stage, after.Stage)
	require.Equal(t, before.Status, after.Status)
	require.EqualValues(t, 2, after.LastAnalyzedMessageID)
	an, err := c.querier.GetLatestLeadAnalysis(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "rekomendasi", an.SuggestedStage)
	require.JSONEq(t, `{"untuk_siapa":"diri_sendiri","level":"pemula","tujuan":["makhraj","tajwid"],"preferensi_waktu":"malam",
		"paket_diminati":"belum_jelas","hambatan":["jadwal"],"gaya_komunikasi":"singkat","catatan":"Responsif di malam hari."}`, an.Persona)
	var actor string
	require.NoError(t, c.db.QueryRow(`SELECT actor FROM lead_events WHERE type = 'analysis.saved' AND lead_id = ?`, id).Scan(&actor))
	require.Equal(t, fmt.Sprintf("ai:%d", c.tokenID), actor)

	// Analysed: no longer listed until a new message arrives.
	list, _ = c.call(t, c.writer(), "daftar_lead", map[string]any{"butuh_analisis": true})
	require.NotContains(t, list, ref)
	c.ingest(t, leadEvent("evt_c3", "message.in", "2026-10-24T04:00:00Z", map[string]string{"wa_message_id": "M3", "body": "Jadwal malam ada?"}))
	list, _ = c.call(t, c.writer(), "daftar_lead", map[string]any{"butuh_analisis": true})
	require.Contains(t, list, ref)

	// An older message id never moves the pointer back.
	args["analyzed_until_message_id"] = 1
	_, isErr = c.call(t, c.writer(), "simpan_analisis_lead", args)
	require.False(t, isErr)
	require.EqualValues(t, 2, c.lead(t).LastAnalyzedMessageID)

	// Stage terverifikasi cannot be suggested; a foreign message id is rejected.
	args["suggested_stage"] = "terverifikasi"
	text, isErr = c.call(t, c.writer(), "simpan_analisis_lead", args)
	require.True(t, isErr)
	require.Contains(t, text, "suggested_stage")
	args["suggested_stage"] = ""
	args["analyzed_until_message_id"] = 999
	text, isErr = c.call(t, c.writer(), "simpan_analisis_lead", args)
	require.True(t, isErr)
	require.Contains(t, text, "bukan pesan milik lead ini")
}

func TestAICRMPersonaValidation(t *testing.T) {
	c := setupCRMAgent(t)
	id := c.seedConversation(t)
	ref := fmt.Sprintf("L-%d", id)
	cases := []struct {
		name    string
		persona map[string]any
		want    string
	}{
		{"unknown field", map[string]any{"level": "pemula", "agama": "kuat"}, "persona.agama tidak dikenal"},
		{"bad enum", map[string]any{"level": "mahir"}, "persona.level harus salah satu"},
		{"bad list item", map[string]any{"hambatan": []string{"malas"}}, "persona.hambatan hanya boleh"},
		{"religious judgement", map[string]any{"catatan": "Kurang religius, perlu didorong"}, "kata terlarang \"religius\""},
		{"economic judgement", map[string]any{"catatan": "Sepertinya keluarga miskin"}, "kata terlarang \"miskin\""},
		{"phrase", map[string]any{"catatan": "Pendukung partai tertentu"}, "kata terlarang \"partai\""},
		{"too long", map[string]any{"catatan": strings.Repeat("a", 301)}, "maksimal 300"},
	}
	for _, tc := range cases {
		args := validAnalysis(ref, 2)
		args["persona"] = tc.persona
		text, isErr := c.call(t, c.writer(), "simpan_analisis_lead", args)
		require.True(t, isErr, tc.name)
		require.Contains(t, text, tc.want, tc.name)
	}
	// Whole words only: "merasa" does not trip "ras", "kayak" does not trip "kaya".
	args := validAnalysis(ref, 2)
	args["persona"] = map[string]any{"catatan": "Merasa kayak belum siap, ingin rutin"}
	text, isErr := c.call(t, c.writer(), "simpan_analisis_lead", args)
	require.False(t, isErr, text)
	var n int
	require.NoError(t, c.db.QueryRow(`SELECT COUNT(*) FROM lead_analyses`).Scan(&n))
	require.Equal(t, 1, n)

	// The keyword list is configurable.
	_, err := c.db.Exec(`UPDATE crm_settings SET persona_blocked_keywords = 'rutin'`)
	require.NoError(t, err)
	text, isErr = c.call(t, c.writer(), "simpan_analisis_lead", args)
	require.True(t, isErr)
	require.Contains(t, text, "\"rutin\"")
}

func TestAICRMLabelOnlyExisting(t *testing.T) {
	c := setupCRMAgent(t)
	id := c.seedConversation(t)
	ref := fmt.Sprintf("L-%d", id)
	// "prioritas" is seeded by the migration.
	text, isErr := c.call(t, c.writer(), "tambah_label_lead", map[string]any{"lead_ref": ref, "label": "vip"})
	require.True(t, isErr)
	require.Contains(t, text, "prioritas")
	text, isErr = c.call(t, c.writer(), "tambah_label_lead", map[string]any{"lead_ref": ref, "label": "Prioritas"})
	require.False(t, isErr, text)
	labels, err := c.querier.ListLabelsForLead(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	text, isErr = c.call(t, c.writer(), "tambah_label_lead", map[string]any{"lead_ref": "L-9999", "label": "prioritas"})
	require.True(t, isErr)
	require.Contains(t, text, "lead tidak ditemukan")
}

func TestAICRMLaporanFunnel(t *testing.T) {
	c := setupCRMAgent(t)
	c.seedConversation(t)
	text, isErr := c.call(t, c.writer(), "laporan_funnel", map[string]any{"dari": "2026-10-01", "sampai": "2026-10-31"})
	require.False(t, isErr, text)
	require.Contains(t, text, "\"total_leads\":1")
	require.NotContains(t, text, "6281234567890")
	_, isErr = c.call(t, c.writer(), "laporan_funnel", map[string]any{"dari": "kemarin"})
	require.True(t, isErr)
}
