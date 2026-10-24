package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupAIAgent(t *testing.T) (*AIAgentService, *AITokenService, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	q := queries.NewQuerier(db)
	now := func() time.Time { return time.Date(2026, 3, 15, 9, 0, 0, 0, ZonaWaktuRiayah) }
	ujroh := NewUjrohService(q)
	ujroh.now = now
	finance := NewKeuanganDashboardService(q, ujroh)
	finance.now = now
	riayah := NewRiayahSantriService(q)
	overview := NewOverviewService(q, finance, riayah)
	overview.now = now
	santri := NewSantriService(q, NewKelasEngineService(q))
	agent := NewAIAgentService(q, overview, finance, riayah, NewTSIService(q), santri, NewLeadService(q, santri))
	agent.now = now
	tokens := NewAITokenService(q)
	tokens.now = func() time.Time { return time.Date(2026, 3, 15, 2, 0, 0, 0, time.UTC) }

	for _, st := range []string{
		`INSERT INTO users (id,email,name,password,role) VALUES (1,'boss@x.id','Boss','x','super_admin'),(2,'ku@x.id','Keu','x','keuangan'),
		 (3,'ak@x.id','AdminKelas','x','admin_kelas'),(4,'ko@x.id','Koor','x','koordinator_guru'),(5,'cs@x.id','CS','x','cs'),(6,'gu@x.id','Guru','x','guru')`,
		`INSERT INTO guru (id,nama,is_aktif,no_wa,email) VALUES (1,'Ustadz Ali',1,'081234567890','ali@guru.id'),(2,'Ustadzah Nonaktif',0,'',''),(3,'Ustadz Tanpa Kelas',1,'','')`,
		`INSERT INTO kelas (id,kunci_kelas,angkatan,tipe,jenis_kelamin,level,frekuensi,nama_kelas,guru_id,kapasitas,is_aktif) VALUES
		 (1,'A','2026','Reguler','L','Dasar','1x/pekan','Kelas A',1,10,1),(2,'B','2026','Reguler','P','Lanjut','2x/pekan','Kelas B',2,10,1)`,
		`INSERT INTO santri (id,nama,id_mahasantri,tipe,level,frekuensi,angkatan,status,kelas_id,tanggal_daftar,no_wa,email,domisili,voice_note_url) VALUES
		 (1,'Ahmad','MHS.2026.0001.012026','Reguler','Dasar','1x/pekan','2026','aktif',1,'2026-01-10','081111111111','ahmad@mail.id','Bandung','/voice/1.mp3'),
		 (2,'Budi','MHS.2026.0002.022026','Reguler','Dasar','1x/pekan','2026','aktif',1,'2026-02-10','082222222222','','Jakarta',''),
		 (3,'Citra','MHS.2026.0003.032026','Reguler','Lanjut','2x/pekan','2026','tidak_lanjut',2,'2026-03-02','','','',''),
		 (4,'Dewi','MHS.2026.0004.032026','Private','Lanjut','2x/pekan','2026','aktif',2,'2026-03-03','','','','')`,
		`INSERT INTO pertemuan (id,kelas_id,pertemuan_ke,tanggal,status,catatan) VALUES
		 (1,1,1,'2026-02-20','selesai','Santri diminta hubungi 0812-3456-7890'),(2,1,2,'2026-03-01','selesai',''),(3,1,3,'2026-03-08','selesai',''),
		 (4,2,1,'2026-03-10','berlangsung','')`,
		`INSERT INTO absensi (pertemuan_id,santri_id,status) VALUES (1,1,'alpa'),(2,1,'alpa'),(3,1,'alpa'),(1,2,'hadir'),(2,2,'hadir'),(3,2,'telat')`,
		`INSERT INTO tagihan (santri_id,kelas_id,bulan_ke,pertemuan_ke,nominal,tanggal_tagih,jatuh_tempo,status) VALUES
		 (1,1,1,1,150000,'2026-01-10','2026-01-17','belum_bayar'),(2,1,1,1,150000,'2026-02-10','2026-02-17','lunas'),(4,2,1,1,200000,'2026-03-14','2026-03-20','belum_bayar')`,
		`INSERT INTO catatan_riayah (guru_id,author_user_id,target_type,target_id,catatan) VALUES (1,6,'santri',1,'Orang tua minta dihubungi di +62 813 9999 8888 atau ortu@mail.com soal absen')`,
		`INSERT INTO santri_admin_note (santri_id,author_user_id,catatan) VALUES (1,3,'Pindah jadwal, kontak wa.me/6281300000000')`,
		`INSERT INTO santri_status_log (santri_id,kelas_id,status_lama,status_baru,alasan,created_at) VALUES (3,2,'aktif','tidak_lanjut','Sibuk kerja','2026-03-05 10:00:00')`,
		`INSERT INTO kunjungan_kelas (id,guru_id,kelas_id,tanggal,status,catatan) VALUES (1,1,1,'2026-03-02','terlaksana','Metode bagus, tajwid perlu diperkuat')`,
		`INSERT INTO kunjungan_tindak_lanjut (kunjungan_id,jenis,catatan,target_tanggal,status) VALUES (1,'coaching','Coaching tajwid','2026-03-10','terbuka')`,
		`INSERT INTO produk (id,nama,kategori) VALUES (1,'Buku Tahsin','buku')`,
		`INSERT INTO product_orders (id,tanggal,nama_pembeli,no_wa,sumber) VALUES (1,'2026-03-03','Pembeli Rahasia','089999999999','whatsapp')`,
		`INSERT INTO product_order_items (order_id,produk_id,qty,harga_satuan) VALUES (1,1,2,50000)`,
	} {
		_, err := db.Exec(st)
		require.NoError(t, err, st)
	}
	return agent, tokens, db
}

func aiPrincipal(role string, userID int64) *models.AIPrincipal {
	return &models.AIPrincipal{UserID: userID, UserName: "Tester", Role: role, BaseURL: "https://dash.test"}
}

func callTool(t *testing.T, s *AIAgentService, p *models.AIPrincipal, name, args string) (string, bool) {
	t.Helper()
	r := s.call(context.Background(), p, name, json.RawMessage(args))
	return r.Text, r.IsError
}

func TestAIAgentToolsRunReadOnlyWithoutContactData(t *testing.T) {
	s, _, db := setupAIAgent(t)
	p := aiPrincipal("super_admin", 1)
	calls := map[string]string{
		"ringkasan_kondisi":          `{}`,
		"santri_perlu_tindak_lanjut": `{}`,
		"cari_santri":                `{"kata":"ahm"}`,
		"detail_santri":              `{"santri_id":1}`,
		"rekap_tunggakan":            `{}`,
		"kelas_perlu_perhatian":      `{"semua":true}`,
		"detail_kelas":               `{"kelas_id":1}`,
		"guru_perlu_perhatian":       `{"semua":true}`,
		"detail_guru":                `{"guru_id":1}`,
		"cari_catatan":               `{"kata":"hubungi"}`,
		"daftar_santri":              `{}`,
		"daftar_kelas":               `{"termasuk_nonaktif":true}`,
		"daftar_guru":                `{"termasuk_nonaktif":true}`,
		"daftar_tagihan":             `{}`,
		"daftar_lead":                `{}`,
		"laporan_funnel":             `{}`,
	}
	for _, dim := range []string{"level", "tipe", "frekuensi", "angkatan", "produk"} {
		calls["tren_program/"+dim] = `{"dimensi":"` + dim + `","bulan":3}`
	}
	require.Len(t, s.Tools("super_admin"), 18)

	var before, after int64
	require.NoError(t, db.QueryRow(`SELECT total_changes()`).Scan(&before))
	outputs := map[string]string{}
	for key, args := range calls {
		name := strings.Split(key, "/")[0]
		text, isErr := callTool(t, s, p, name, args)
		require.False(t, isErr, "%s: %s", key, text)
		require.True(t, json.Valid([]byte(text)), key)
		outputs[key] = text
		for _, leak := range []string{"081111111111", "082222222222", "081234567890", "ahmad@mail.id", "ali@guru.id", "/voice/1.mp3",
			"Bandung", "0812-3456-7890", "813 9999 8888", "ortu@mail.com", "wa.me/", "Pembeli Rahasia", "089999999999"} {
			require.NotContains(t, text, leak, "%s leaked %q", key, leak)
		}
	}
	require.NoError(t, db.QueryRow(`SELECT total_changes()`).Scan(&after))
	require.Equal(t, before, after, "AI tools must not write")

	require.Contains(t, outputs["cari_catatan"], aiMaskPhone)
	require.Contains(t, outputs["detail_santri"], aiMaskEmail)
	require.Contains(t, outputs["detail_santri"], aiMaskLink)
	require.Contains(t, outputs["detail_santri"], "https://dash.test/app/guru/santri/1")

	// Ahmad: 3x alpa + tunggakan Jan → top of follow-up list with evidence.
	var tl struct {
		Santri []struct {
			SantriID  int64    `json:"santri_id"`
			Alasan    []string `json:"alasan"`
			Tunggakan *struct {
				Nominal int64 `json:"nominal"`
			} `json:"tunggakan"`
		} `json:"santri"`
	}
	require.NoError(t, json.Unmarshal([]byte(outputs["santri_perlu_tindak_lanjut"]), &tl))
	require.NotEmpty(t, tl.Santri)
	require.EqualValues(t, 1, tl.Santri[0].SantriID)
	require.NotNil(t, tl.Santri[0].Tunggakan)
	require.EqualValues(t, 150000, tl.Santri[0].Tunggakan.Nominal)

	// Kelas B: guru nonaktif and a past meeting still open.
	require.Contains(t, outputs["kelas_perlu_perhatian"], "guru Ustadzah Nonaktif tidak aktif")
	require.Contains(t, outputs["kelas_perlu_perhatian"], "belum ditutup")
	require.Contains(t, outputs["guru_perlu_perhatian"], "tindak lanjut kunjungan lewat target")
	require.Contains(t, outputs["tren_program/produk"], "Buku Tahsin")
	require.Contains(t, outputs["tren_program/level"], "Lanjut")
}

func TestAIAgentRoleGating(t *testing.T) {
	s, _, _ := setupAIAgent(t)
	names := func(role string) []string {
		out := []string{}
		for _, tool := range s.Tools(role) {
			out = append(out, tool.Name)
		}
		return out
	}
	crmRead := []string{"daftar_lead", "detail_lead", "laporan_funnel"}
	require.ElementsMatch(t, append([]string{"ringkasan_kondisi", "rekap_tunggakan", "daftar_tagihan"}, crmRead...), names("keuangan"))
	require.ElementsMatch(t, append([]string{"cari_santri", "detail_santri", "daftar_santri"}, crmRead...), names("cs"))
	require.NotContains(t, names("admin_kelas"), "daftar_lead")
	require.NotContains(t, names("koordinator_guru"), "detail_lead")
	require.NotContains(t, names("admin_kelas"), "daftar_guru")
	require.NotContains(t, names("koordinator_guru"), "daftar_tagihan")
	require.NotContains(t, names("admin_kelas"), "rekap_tunggakan")
	require.NotContains(t, names("koordinator_guru"), "tren_program")
	require.Empty(t, names("guru"))

	text, isErr := callTool(t, s, aiPrincipal("cs", 5), "rekap_tunggakan", `{}`)
	require.True(t, isErr)
	require.Contains(t, text, "tidak memiliki akses")

	// admin_kelas sees riayah and admin notes but no tagihan.
	text, isErr = callTool(t, s, aiPrincipal("admin_kelas", 3), "detail_santri", `{"santri_id":1}`)
	require.False(t, isErr, text)
	require.NotContains(t, text, "tagihan_terbaru")
	require.Contains(t, text, "catatan_admin")
	require.Contains(t, text, "Ahmad")

	// cs gets the profile only.
	text, isErr = callTool(t, s, aiPrincipal("cs", 5), "detail_santri", `{"santri_id":1}`)
	require.False(t, isErr, text)
	require.NotContains(t, text, "timeline_terbaru")
	require.NotContains(t, text, "catatan_admin")
	require.Contains(t, text, "https://dash.test/app/santri/1")

	// koordinator sees riayah but not the SPP nominal.
	text, isErr = callTool(t, s, aiPrincipal("koordinator_guru", 4), "detail_santri", `{"santri_id":1}`)
	require.False(t, isErr, text)
	require.NotContains(t, text, "nominal_spp")
	require.Contains(t, text, "timeline_terbaru")

	// koordinator may not search admin notes and gets no tunggakan.
	text, isErr = callTool(t, s, aiPrincipal("koordinator_guru", 4), "cari_catatan", `{"kata":"jadwal","sumber":"catatan_admin"}`)
	require.True(t, isErr)
	require.Contains(t, text, "tidak tersedia")
	text, isErr = callTool(t, s, aiPrincipal("koordinator_guru", 4), "santri_perlu_tindak_lanjut", `{"penanda":"tunggakan"}`)
	require.True(t, isErr, text)

	// Validation errors are returned to the AI, not swallowed.
	text, isErr = callTool(t, s, aiPrincipal("super_admin", 1), "tren_program", `{"dimensi":"kota"}`)
	require.True(t, isErr)
	require.Contains(t, text, "dimensi harus salah satu")
	text, isErr = callTool(t, s, aiPrincipal("super_admin", 1), "detail_santri", `{"santri_id":999}`)
	require.True(t, isErr)
	require.Contains(t, text, "tidak ditemukan")
}

func TestAIAgentCallWritesAudit(t *testing.T) {
	s, _, db := setupAIAgent(t)
	r := s.Call(context.Background(), aiPrincipal("super_admin", 1), "cari_santri", json.RawMessage(`{"kata":"0811 1111 1111"}`))
	require.False(t, r.IsError, r.Text)
	s.Call(context.Background(), aiPrincipal("cs", 5), "rekap_tunggakan", nil)
	var n int64
	var args, status string
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM ai_audit_log`).Scan(&n))
	require.EqualValues(t, 2, n)
	require.NoError(t, db.QueryRow(`SELECT arguments, status FROM ai_audit_log WHERE tool='cari_santri'`).Scan(&args, &status))
	require.Equal(t, "ok", status)
	require.NotContains(t, args, "0811 1111 1111")
	require.NoError(t, db.QueryRow(`SELECT status FROM ai_audit_log WHERE tool='rekap_tunggakan'`).Scan(&status))
	require.Equal(t, "error", status)
}

func TestAITokenLifecycle(t *testing.T) {
	_, tokens, db := setupAIAgent(t)
	ctx := context.Background()

	_, err := tokens.Create(6, "guru", "Hermes", 30, false)
	require.ErrorIs(t, err, ErrAIRoleDitolak)

	created, err := tokens.Create(1, "super_admin", "Claude Desktop", 30, false)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(created.Plaintext, AITokenPrefix))
	var stored string
	require.NoError(t, db.QueryRow(`SELECT token_hash FROM ai_tokens WHERE id = ?`, created.Token.ID).Scan(&stored))
	require.NotEqual(t, created.Plaintext, stored, "plaintext must not be stored")

	p, err := tokens.Authenticate(ctx, created.Plaintext)
	require.NoError(t, err)
	require.Equal(t, "super_admin", p.Role)
	require.EqualValues(t, 1, p.UserID)

	_, err = tokens.Authenticate(ctx, created.Plaintext+"x")
	require.ErrorIs(t, err, ErrAITokenInvalid)
	_, err = tokens.Authenticate(ctx, "Bearer nonsense")
	require.ErrorIs(t, err, ErrAITokenInvalid)

	// Demoting the owner cuts access immediately.
	_, err = db.Exec(`UPDATE users SET role = 'guru' WHERE id = 1`)
	require.NoError(t, err)
	_, err = tokens.Authenticate(ctx, created.Plaintext)
	require.ErrorIs(t, err, ErrAIRoleDitolak)
	_, err = db.Exec(`UPDATE users SET role = 'super_admin' WHERE id = 1`)
	require.NoError(t, err)

	// Only the owner can revoke; revoked tokens stop working.
	require.ErrorIs(t, tokens.Revoke(2, created.Token.ID), ErrAITokenNotFound)
	require.NoError(t, tokens.Revoke(1, created.Token.ID))
	_, err = tokens.Authenticate(ctx, created.Plaintext)
	require.ErrorIs(t, err, ErrAITokenInvalid)

	// Expiry.
	short, err := tokens.Create(1, "super_admin", "Singkat", 1, false)
	require.NoError(t, err)
	tokens.now = func() time.Time { return time.Date(2026, 3, 17, 2, 0, 0, 0, time.UTC) }
	_, err = tokens.Authenticate(ctx, short.Plaintext)
	require.ErrorIs(t, err, ErrAITokenInvalid)

	list, err := tokens.List(1)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.NotNil(t, list[1].RevokedAt)
}

func TestRedactContactText(t *testing.T) {
	cases := map[string]string{
		"hubungi 081234567890":          "hubungi " + aiMaskPhone,
		"wa +62 812-3456-7890 ya":       "wa " + aiMaskPhone + " ya",
		"628123456789":                  aiMaskPhone,
		"email a.b@c.co.id":             "email " + aiMaskEmail,
		"https://wa.me/628123 cek":      aiMaskLink + " cek",
		"ID MHS.2026.0012.032026 aman":  "ID MHS.2026.0012.032026 aman",
		"nominal 150000 dan 2026-03-15": "nominal 150000 dan 2026-03-15",
	}
	for in, want := range cases {
		require.Equal(t, want, RedactContactText(in), in)
	}
}

func TestAIAgentDaftarToolsListNamesBehindOverviewCounts(t *testing.T) {
	s, _, db := setupAIAgent(t)
	_, err := db.Exec(`INSERT INTO kelas (id,kunci_kelas,angkatan,tipe,jenis_kelamin,level,frekuensi,nama_kelas,guru_id,kapasitas,is_aktif) VALUES (3,'C','2026','Reguler','L','Dasar','1x/pekan','Kelas Tutup',1,10,0)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO santri (id,nama,status,kelas_id,tanggal_daftar,no_wa) VALUES
	 (10,'Eko Tanpa Kelas','aktif',NULL,'2026-03-12','085555555555'),(11,'Fani Kelas Tutup','aktif',3,'2026-03-13',''),(12,'Gina Cuti','cuti',NULL,'2026-03-13','')`)
	require.NoError(t, err)
	p := aiPrincipal("super_admin", 1)

	// The overview count and the list use the same definition.
	text, isErr := callTool(t, s, p, "ringkasan_kondisi", `{}`)
	require.False(t, isErr, text)
	var ov struct {
		Operasional struct {
			WithoutClass int64 `json:"santri_aktif_tanpa_kelas_aktif"`
		} `json:"operasional"`
		Prioritas []struct {
			Judul   string         `json:"judul"`
			Rincian map[string]any `json:"rincian"`
		} `json:"prioritas"`
	}
	require.NoError(t, json.Unmarshal([]byte(text), &ov))
	require.EqualValues(t, 2, ov.Operasional.WithoutClass)
	var hint map[string]any
	for _, pr := range ov.Prioritas {
		if pr.Judul == "Santri aktif belum punya kelas aktif" {
			hint = pr.Rincian
		}
	}
	require.Equal(t, "daftar_santri", hint["tool"])
	args, _ := json.Marshal(hint["argumen"])

	text, isErr = callTool(t, s, p, "daftar_santri", string(args))
	require.False(t, isErr, text)
	var list struct {
		Total  int64 `json:"total"`
		Santri []struct {
			Santri struct {
				Nama string `json:"nama"`
			} `json:"santri"`
		} `json:"santri"`
	}
	require.NoError(t, json.Unmarshal([]byte(text), &list))
	require.EqualValues(t, ov.Operasional.WithoutClass, list.Total)
	names := []string{}
	for _, r := range list.Santri {
		names = append(names, r.Santri.Nama)
	}
	require.ElementsMatch(t, []string{"Eko Tanpa Kelas", "Fani Kelas Tutup"}, names)
	require.NotContains(t, text, "085555555555")

	// Paging.
	text, isErr = callTool(t, s, p, "daftar_santri", `{"per_halaman":2,"halaman":2}`)
	require.False(t, isErr, text)
	require.Contains(t, text, `"halaman":2`)
	require.Contains(t, text, `"total":7`)

	// Registration date filter works on driver-written and plain dates.
	text, _ = callTool(t, s, p, "daftar_santri", `{"daftar_dari":"2026-03-12","daftar_sampai":"2026-03-12"}`)
	require.Contains(t, text, "Eko Tanpa Kelas")
	require.NotContains(t, text, "Fani")

	text, isErr = callTool(t, s, p, "daftar_tagihan", `{"terlambat":true}`)
	require.False(t, isErr, text)
	require.Contains(t, text, `"total":1`)
	require.Contains(t, text, "Ahmad")

	text, _ = callTool(t, s, p, "daftar_kelas", `{}`)
	require.NotContains(t, text, "Kelas Tutup")
	text, _ = callTool(t, s, p, "daftar_kelas", `{"termasuk_nonaktif":true}`)
	require.Contains(t, text, "Kelas Tutup")
}

func TestAIAgentRincianOnlyNamesToolsTheRoleHas(t *testing.T) {
	s, _, db := setupAIAgent(t)
	_, err := db.Exec(`INSERT INTO santri (nama,status,kelas_id) VALUES ('Tanpa Kelas','aktif',NULL)`)
	require.NoError(t, err)
	for role, wantTool := range map[string]bool{"super_admin": true, "keuangan": false} {
		text, isErr := callTool(t, s, aiPrincipal(role, 1), "ringkasan_kondisi", `{}`)
		require.False(t, isErr, text)
		var ov struct {
			Prioritas []struct {
				Judul   string         `json:"judul"`
				Rincian map[string]any `json:"rincian"`
			} `json:"prioritas"`
			Tools []string `json:"tool_untuk_role_ini"`
		}
		require.NoError(t, json.Unmarshal([]byte(text), &ov))
		for _, pr := range ov.Prioritas {
			if tool, ok := pr.Rincian["tool"].(string); ok {
				require.Contains(t, ov.Tools, tool, "%s: rincian names a tool the role cannot call", role)
			}
		}
		require.Equal(t, wantTool, contains(ov.Tools, "daftar_santri"), role)
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
