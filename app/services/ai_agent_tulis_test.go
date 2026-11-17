package services

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/stretchr/testify/require"
)

func (c *crmAgentFixture) principal(role string, canWrite bool) *models.AIPrincipal {
	return &models.AIPrincipal{UserID: c.adminID, TokenID: c.tokenID, UserName: "Staf", Role: role, CanWrite: canWrite, BaseURL: "https://dash.test"}
}

func jsonMap(t *testing.T, text string) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &out), text)
	return out
}

// AIWriteRoles must stay equal to the roles of the write tools, so that the
// write permission is offered exactly to roles that can use it.
func TestAIWriteRolesMatchWriteTools(t *testing.T) {
	c := setupCRMAgent(t)
	c.agent.SetLayananTulis(NewKoordinatorService(c.querier), NewProductCRMService(c.querier))
	roles := map[string]bool{}
	for _, tool := range c.agent.tools {
		if tool.Write {
			for _, r := range tool.Roles {
				roles[r] = true
			}
		}
	}
	got := []string{}
	for r := range roles {
		got = append(got, r)
	}
	want := slices.Clone(AIWriteRoles)
	slices.Sort(got)
	slices.Sort(want)
	require.Equal(t, want, got)
	require.False(t, AIRoleCanWrite("admin"))
	require.False(t, AIRoleCanWrite("guru"))
}

func TestAISumberLeadTulis(t *testing.T) {
	c := setupCRMAgent(t)
	cs := c.principal("cs", true)

	// Without the write permission the tool is refused.
	text, isErr := c.call(t, c.principal("cs", false), "buat_sumber_lead", map[string]any{"nama": "LP Talaqqi"})
	require.True(t, isErr)
	require.Contains(t, text, "izin menulis")
	// keuangan may read the CRM but not edit sources (same as the dashboard).
	_, isErr = c.call(t, c.principal("keuangan", true), "buat_sumber_lead", map[string]any{"nama": "LP Talaqqi"})
	require.True(t, isErr)

	text, isErr = c.call(t, cs, "buat_sumber_lead", map[string]any{
		"nama": "LP Talaqqi Ramadhan", "url": "https://ruangsanad.com/talaqqi", "keterangan": "Iklan Meta Ramadhan",
		"teks_pemicu": []string{"Assalamualaikum saya mau daftar kelas Talaqqi Ramadhan", "hai"},
	})
	require.False(t, isErr, text)
	out := jsonMap(t, text)
	id := int64(out["sumber_id"].(float64))
	pemicu := out["teks_pemicu"].([]any)
	require.Equal(t, true, pemicu[0].(map[string]any)["ok"])
	require.Equal(t, false, pemicu[1].(map[string]any)["ok"], "too-short trigger is reported, not fatal")
	require.Contains(t, pemicu[1].(map[string]any)["pesan"], "terlalu pendek")

	// Validation from the dashboard service reaches the AI as-is.
	text, isErr = c.call(t, cs, "buat_sumber_lead", map[string]any{"nama": "LP Talaqqi Ramadhan"})
	require.True(t, isErr)
	require.Contains(t, text, "sudah ada")
	text, isErr = c.call(t, cs, "buat_sumber_lead", map[string]any{"nama": "LP Baru", "url": "ruangsanad.com"})
	require.True(t, isErr)
	require.Contains(t, text, "URL LP")

	// Partial update: only aktif changes; deactivate instead of delete.
	text, isErr = c.call(t, cs, "ubah_sumber_lead", map[string]any{"sumber_id": id, "aktif": false})
	require.False(t, isErr, text)
	src, err := c.agent.sumberByID(t.Context(), id)
	require.NoError(t, err)
	require.False(t, src.IsAktif)
	require.Equal(t, "https://ruangsanad.com/talaqqi", src.URL)
	require.Equal(t, "Iklan Meta Ramadhan", src.Keterangan)

	text, isErr = c.call(t, cs, "tambah_teks_pemicu", map[string]any{"sumber_id": id, "teks": "Halo kak saya tertarik program Talaqqi bulan Ramadhan"})
	require.False(t, isErr, text)
	text, _ = c.call(t, cs, "daftar_sumber_lead", map[string]any{})
	list := jsonMap(t, text)["sumber"].([]any)
	var polaID float64
	for _, s := range list {
		m := s.(map[string]any)
		if int64(m["sumber_id"].(float64)) == id {
			require.Len(t, m["teks_pemicu"], 2)
			polaID = m["teks_pemicu"].([]any)[1].(map[string]any)["pola_id"].(float64)
		}
	}
	require.NotZero(t, polaID)
	text, isErr = c.call(t, cs, "hapus_teks_pemicu", map[string]any{"pola_id": polaID})
	require.False(t, isErr, text)
	src, _ = c.agent.sumberByID(t.Context(), id)
	require.Len(t, src.Patterns, 1)

	text, isErr = c.call(t, cs, "cocokkan_ulang_sumber_lead", map[string]any{})
	require.False(t, isErr, text)
	require.Contains(t, text, "lead_tercocokkan")

	// Audit trail records every write call.
	var n int
	require.NoError(t, c.db.QueryRow(`SELECT COUNT(*) FROM ai_audit_log WHERE tool IN ('buat_sumber_lead','ubah_sumber_lead','hapus_teks_pemicu')`).Scan(&n))
	require.GreaterOrEqual(t, n, 4)
}

func TestAILabelProdukKoordinatorTulis(t *testing.T) {
	c := setupCRMAgent(t)
	c.agent.SetLayananTulis(NewKoordinatorService(c.querier), NewProductCRMService(c.querier))

	text, isErr := c.call(t, c.principal("keuangan", true), "buat_label_lead", map[string]any{"nama": "Minta diskon", "warna": "secondary"})
	require.False(t, isErr, text)
	require.NotZero(t, jsonMap(t, text)["label_id"])
	_, isErr = c.call(t, c.principal("keuangan", true), "buat_label_lead", map[string]any{"nama": "minta  diskon"})
	require.True(t, isErr, "duplicate label name")

	ak := c.principal("admin_kelas", true)
	text, isErr = c.call(t, ak, "buat_produk_crm", map[string]any{"nama": "Buku Tahsin Jilid 1", "kategori": "buku", "harga": 75000, "track_stok": true, "minimum_stok": 5})
	require.False(t, isErr, text)
	text, _ = c.call(t, ak, "daftar_produk_crm", map[string]any{})
	produk := jsonMap(t, text)["produk"].([]any)[0].(map[string]any)
	text, isErr = c.call(t, ak, "ubah_produk_crm", map[string]any{"produk_id": produk["produk_id"], "harga": 80000})
	require.False(t, isErr, text)
	sesudah := jsonMap(t, text)["sesudah"].(map[string]any)
	require.EqualValues(t, 80000, sesudah["harga"])
	require.Equal(t, "Buku Tahsin Jilid 1", sesudah["nama"])
	_, isErr = c.call(t, c.principal("cs", true), "buat_produk_crm", map[string]any{"nama": "X", "kategori": "buku"})
	require.True(t, isErr, "cs cannot edit Produk CRM")

	k := c.principal("koordinator_guru", true)
	text, isErr = c.call(t, k, "buat_pembinaan", map[string]any{"tanggal": "2026-11-20", "topik": "Adab mengajar anak", "keterangan": "Zoom 20.00"})
	require.False(t, isErr, text)
	p := jsonMap(t, text)
	pb := p["pembinaan"].(map[string]any)
	require.Equal(t, "November", pb["bulan"])
	require.EqualValues(t, 3, pb["pekan_ke"])
	require.Equal(t, "dijadwalkan", pb["status"])
	text, isErr = c.call(t, k, "ubah_pembinaan", map[string]any{"pembinaan_id": p["pembinaan_id"], "status": "libur"})
	require.False(t, isErr, text)
	require.Equal(t, "Adab mengajar anak", jsonMap(t, text)["sesudah"].(map[string]any)["topik"])
	_, isErr = c.call(t, k, "buat_pembinaan", map[string]any{"tanggal": "20-11-2026", "topik": "x"})
	require.True(t, isErr)

	text, isErr = c.call(t, k, "buat_rapat", map[string]any{"tanggal": "2026-11-25", "judul": "Evaluasi bulanan"})
	require.False(t, isErr, text)
	text, isErr = c.call(t, k, "buat_todo_koordinator", map[string]any{"judul": "Rekap TSI November", "deadline": "2026-11-30", "pic": "Ustadzah Koor"})
	require.False(t, isErr, text)
	todoID := jsonMap(t, text)["todo_id"]
	text, isErr = c.call(t, k, "ubah_todo_koordinator", map[string]any{"todo_id": todoID, "status": "selesai"})
	require.False(t, isErr, text)
	require.Equal(t, "Rekap TSI November", jsonMap(t, text)["sesudah"].(map[string]any)["judul"])
	text, isErr = c.call(t, k, "buat_template_wa", map[string]any{"nama": "Pengingat pembinaan", "target": "guru", "isi": "Assalamu'alaikum {nama}, pengingat pembinaan malam ini."})
	require.False(t, isErr, text)

	text, isErr = c.call(t, k, "daftar_agenda_koordinator", map[string]any{})
	require.False(t, isErr, text)
	agenda := jsonMap(t, text)
	require.Len(t, agenda["pembinaan"], 1)
	require.Len(t, agenda["rapat"], 1)
	require.Len(t, agenda["todo"], 1)
	require.Len(t, agenda["template_wa"], 1)

	// Instructions name the write tools the connection can use.
	instr := c.agent.Instructions(k)
	require.Contains(t, instr, "buat_pembinaan")
	require.NotContains(t, instr, "buat_sumber_lead")
	require.Contains(t, c.agent.Instructions(c.principal("koordinator_guru", false)), "BACA SAJA")
}
