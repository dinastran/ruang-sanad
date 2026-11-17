package services

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/models"
)

// AI write tools for data entry the staff asked Claude/ChatGPT to do for them:
// lead sources (Sumber Lead), the CRM label catalog, Produk CRM, and the
// Koordinator Guru agenda (pembinaan, rapat, todo, WA templates).
//
// Every tool calls the same service method as the dashboard button, so the
// validation is identical, and is limited to the roles that may use that page
// in the dashboard. Nothing here deletes data: sources and templates are
// deactivated instead (a wrong trigger text may be removed; it holds no lead
// data). Money (tagihan, ujroh) stays manual.

var (
	aiRolesSumberLead = []string{"cs", "super_admin"}          // routes.LeadSourceEditorRoles
	aiRolesProduk     = []string{"admin_kelas", "super_admin"} // Produk CRM (akRole)
)

// AIWriteRoles are the roles that have at least one AI write tool, so they may
// give an AI connection the write permission. Kept equal to the union of the
// write tools' roles (checked by a test).
var AIWriteRoles = []string{"super_admin", "cs", "keuangan", "admin_kelas", "koordinator_guru"}

// AIRoleCanWrite reports whether a role may own an AI connection that writes.
func AIRoleCanWrite(role string) bool { return slices.Contains(AIWriteRoles, role) }

// SetLayananTulis attaches the services behind the Produk CRM and Koordinator
// Guru tools; their tools are only offered when the service is set.
func (s *AIAgentService) SetLayananTulis(koordinator *KoordinatorService, produk *ProductCRMService) {
	s.koordinator = koordinator
	s.produk = produk
	s.tools = s.buildTools()
}

func (a aiArgs) ada(key string) bool {
	v, ok := a.m[key]
	return ok && v != nil
}

// strOr returns the argument when given, else the current value (partial update).
func (a aiArgs) strOr(key, current string) string {
	if a.ada(key) {
		return a.str(key)
	}
	return current
}

func (a aiArgs) boolOr(key string, current bool) bool {
	if a.ada(key) {
		return a.bool(key)
	}
	return current
}

func (a aiArgs) teks(key, label string, wajib bool, maks int) (string, error) {
	v := a.str(key)
	if wajib && v == "" {
		return "", aiArgErr("%s wajib diisi", label)
	}
	if utf8.RuneCountInString(v) > maks {
		return "", aiArgErr("%s maksimal %d karakter", label, maks)
	}
	return v, nil
}

func aiTanggal(v, label string, wajib bool) (string, error) {
	if v == "" {
		if wajib {
			return "", aiArgErr("%s wajib diisi (YYYY-MM-DD)", label)
		}
		return "", nil
	}
	if _, err := time.Parse("2006-01-02", v); err != nil {
		return "", aiArgErr("%s harus berformat YYYY-MM-DD", label)
	}
	return v, nil
}

// aiErrLayanan turns a service validation error into a message the AI can
// show; database errors stay internal (generic message, logged).
func aiErrLayanan(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return aiArgErr("data tidak ditemukan; cek id dari tool daftar terkait")
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	for _, internal := range []string{"sql", "database", "constraint", "no such", "context"} {
		if strings.Contains(low, internal) {
			return err
		}
	}
	return aiArgErr("%s", msg)
}

var aiNamaBulan = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func (s *AIAgentService) tulisTools() []AITool {
	tools := []AITool{
		{
			Name:  "buat_sumber_lead",
			Title: "Buat sumber lead",
			Description: "TULIS. Buat sumber lead baru (landing page / iklan / kanal) seperti tombol Tambah di halaman Sumber Lead, sekaligus teks pemicu " +
				"(kalimat khas pesan pertama dari LP/iklan) agar lead baru otomatis tercocokkan. Cek daftar_sumber_lead dulu agar tidak dobel.",
			InputSchema: aiSchema(map[string]any{
				"nama":        aiProp("string", "Nama sumber, unik, mis. \"LP Tahsin Ramadhan\"."),
				"url":         aiProp("string", "URL landing page lengkap (https://...), opsional. Dipakai Meta CAPI sebagai event_source_url."),
				"keterangan":  aiProp("string", "Catatan singkat, opsional."),
				"aktif":       aiProp("boolean", "Default true."),
				"teks_pemicu": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Maks 10 kalimat khas pesan pertama dari tombol WA di LP/iklan, mis. \"Assalamualaikum saya mau daftar Tahsin Ramadhan\"."},
			}, "nama"),
			Roles: aiRolesSumberLead,
			Write: true,
			run:   s.toolBuatSumberLead,
		},
		{
			Name:  "ubah_sumber_lead",
			Title: "Ubah sumber lead",
			Description: "TULIS. Ubah nama, URL, keterangan, atau status aktif sumber lead. Field yang tidak diisi tetap. " +
				"Untuk menghentikan sumber, set aktif=false (nonaktifkan); sumber tidak dihapus agar laporan tetap terbaca.",
			InputSchema: aiSchema(map[string]any{
				"sumber_id":  aiIntProp("id dari daftar_sumber_lead.", 1, math.MaxInt64),
				"nama":       aiProp("string", "Nama baru."),
				"url":        aiProp("string", "URL baru; string kosong = hapus URL."),
				"keterangan": aiProp("string", "Keterangan baru."),
				"aktif":      aiProp("boolean", "false = nonaktifkan, true = aktifkan."),
			}, "sumber_id"),
			Roles:       aiRolesSumberLead,
			Write:       true,
			Destructive: true,
			run:         s.toolUbahSumberLead,
		},
		{
			Name:        "tambah_teks_pemicu",
			Title:       "Tambah teks pemicu sumber",
			Description: "TULIS. Tambah satu teks pemicu (kalimat khas pesan pertama) ke sumber lead. Teks harus cukup spesifik dan belum dipakai sumber lain.",
			InputSchema: aiSchema(map[string]any{
				"sumber_id": aiIntProp("id dari daftar_sumber_lead.", 1, math.MaxInt64),
				"teks":      aiProp("string", "Kalimat pemicu, mis. \"Halo kak, saya tertarik kelas Talaqqi\"."),
			}, "sumber_id", "teks"),
			Roles: aiRolesSumberLead,
			Write: true,
			run:   s.toolTambahTeksPemicu,
		},
		{
			Name:        "hapus_teks_pemicu",
			Title:       "Hapus teks pemicu sumber",
			Description: "TULIS. Hapus satu teks pemicu yang salah (pola_id dari daftar_sumber_lead). Lead yang sudah tercocokkan tidak berubah.",
			InputSchema: aiSchema(map[string]any{
				"pola_id": aiIntProp("pola_id dari teks_pemicu di daftar_sumber_lead.", 1, math.MaxInt64),
			}, "pola_id"),
			Roles:       aiRolesSumberLead,
			Write:       true,
			Destructive: true,
			run:         s.toolHapusTeksPemicu,
		},
		{
			Name:  "cocokkan_ulang_sumber_lead",
			Title: "Cocokkan ulang sumber lead",
			Description: "TULIS. Isi sumber untuk lead yang BELUM punya sumber berdasarkan teks pemicu terbaru (sama seperti tombol Cocokkan di halaman Sumber Lead). " +
				"Lead yang sumbernya diatur manual tidak diubah. Panggil setelah menambah teks pemicu.",
			InputSchema: aiSchema(map[string]any{}),
			Roles:       aiRolesSumberLead,
			Write:       true,
			run:         s.toolCocokkanUlangSumber,
		},
		{
			Name:  "buat_label_lead",
			Title: "Buat label lead baru",
			Description: "TULIS. Tambah label baru ke katalog label CRM (berlaku untuk seluruh tim). Untuk memasang label ke lead, pakai tambah_label_lead. " +
				"Cek label yang ada di referensi_crm agar tidak dobel.",
			InputSchema: aiSchema(map[string]any{
				"nama":  aiProp("string", "Nama label, maks 60 karakter."),
				"warna": aiEnumProp("Warna (default neutral).", "brand", "secondary", "neutral"),
			}, "nama"),
			Roles: aiRolesCRM,
			Write: true,
			run:   s.toolBuatLabelLead,
		},
	}
	if s.produk != nil {
		tools = append(tools,
			AITool{
				Name:        "daftar_produk_crm",
				Title:       "Produk CRM",
				Description: "Katalog Produk CRM (buku/program): produk_id, nama, kategori, harga (Rupiah), aktif, stok, minimum stok, dan batch/gelombang.",
				InputSchema: aiSchema(map[string]any{}),
				Roles:       aiRolesProduk,
				run:         s.toolDaftarProduk,
			},
			AITool{
				Name:        "buat_produk_crm",
				Title:       "Buat produk CRM",
				Description: "TULIS. Tambah produk ke katalog Produk CRM seperti form di halaman Produk CRM. Kategori program tidak memakai stok.",
				InputSchema: aiSchema(map[string]any{
					"nama":         aiProp("string", "Nama produk."),
					"kategori":     aiEnumProp("Kategori.", "buku", "program"),
					"harga":        aiIntProp("Harga dalam Rupiah (default 0).", 0, math.MaxInt32),
					"track_stok":   aiProp("boolean", "Lacak stok (hanya buku)."),
					"minimum_stok": aiIntProp("Batas stok minimum untuk peringatan (hanya bila track_stok).", 0, math.MaxInt32),
				}, "nama", "kategori"),
				Roles: aiRolesProduk,
				Write: true,
				run:   s.toolBuatProduk,
			},
			AITool{
				Name:  "ubah_produk_crm",
				Title: "Ubah produk CRM",
				Description: "TULIS. Ubah nama, harga, minimum stok, atau status aktif produk; field yang tidak diisi tetap. aktif=false menonaktifkan produk. " +
					"Kategori & mode stok tidak bisa diubah bila produk sudah punya transaksi.",
				InputSchema: aiSchema(map[string]any{
					"produk_id":    aiIntProp("produk_id dari daftar_produk_crm.", 1, math.MaxInt64),
					"nama":         aiProp("string", "Nama baru."),
					"kategori":     aiEnumProp("Kategori.", "buku", "program"),
					"harga":        aiIntProp("Harga dalam Rupiah.", 0, math.MaxInt32),
					"track_stok":   aiProp("boolean", "Lacak stok."),
					"minimum_stok": aiIntProp("Batas stok minimum.", 0, math.MaxInt32),
					"aktif":        aiProp("boolean", "false = nonaktifkan."),
				}, "produk_id"),
				Roles:       aiRolesProduk,
				Write:       true,
				Destructive: true,
				run:         s.toolUbahProduk,
			},
		)
	}
	if s.koordinator != nil {
		tools = append(tools, s.koordinatorTools()...)
	}
	return tools
}

func (s *AIAgentService) koordinatorTools() []AITool {
	statusPembinaan := []string{"dijadwalkan", "terlaksana", "libur"}
	statusRapat := []string{"dijadwalkan", "terlaksana", "batal"}
	statusTodo := []string{"belum", "proses", "selesai", "batal"}
	recurring := []string{"none", "harian", "mingguan", "bulanan", "4bulanan"}
	pembinaanProps := map[string]any{
		"tanggal":    aiProp("string", "Tanggal YYYY-MM-DD."),
		"topik":      aiProp("string", "Topik pembinaan."),
		"keterangan": aiProp("string", "Keterangan, mis. link Zoom atau pemateri."),
		"bulan":      aiProp("string", "Nama bulan, mis. \"November\". Kosong = dari tanggal."),
		"pekan_ke":   aiIntProp("Pekan ke- dalam bulan. Kosong = dari tanggal.", 1, 6),
		"status":     aiEnumProp("Status (default dijadwalkan). Absensi diisi di dashboard; menyimpan absensi otomatis menandai terlaksana.", statusPembinaan...),
	}
	rapatProps := map[string]any{
		"tanggal": aiProp("string", "Tanggal YYYY-MM-DD."),
		"judul":   aiProp("string", "Judul/agenda rapat."),
		"catatan": aiProp("string", "Catatan, mis. link Zoom atau poin agenda."),
		"status":  aiEnumProp("Status (default dijadwalkan).", statusRapat...),
	}
	todoProps := map[string]any{
		"judul":          aiProp("string", "Judul tugas."),
		"teknis":         aiProp("string", "Teknis pelaksanaan."),
		"kebutuhan":      aiProp("string", "Kebutuhan/bahan."),
		"deadline":       aiProp("string", "Tenggat YYYY-MM-DD."),
		"pic":            aiProp("string", "Penanggung jawab (nama)."),
		"status":         aiEnumProp("Status (default belum).", statusTodo...),
		"recurring":      aiEnumProp("Pengulangan (default none).", recurring...),
		"link_pendukung": aiProp("string", "Link dokumen pendukung."),
		"catatan":        aiProp("string", "Catatan."),
	}
	templateProps := map[string]any{
		"nama":   aiProp("string", "Nama template."),
		"target": aiEnumProp("Untuk santri atau guru (default santri).", "santri", "guru"),
		"isi":    aiProp("string", "Isi pesan WhatsApp. Boleh memakai placeholder yang dipakai template lain, mis. {nama}."),
		"aktif":  aiProp("boolean", "Default true; false = nonaktifkan."),
	}
	withID := func(idKey, idDesc string, props map[string]any) map[string]any {
		return aiMerge(map[string]any{idKey: aiIntProp(idDesc, 1, math.MaxInt64)}, props)
	}
	return []AITool{
		{
			Name:  "daftar_agenda_koordinator",
			Title: "Agenda Koordinator Guru",
			Description: "Jadwal pembinaan dan rapat guru (id, tanggal, topik/judul, status), todo koordinator, dan template WA koordinator. " +
				"Pakai id-nya untuk tool ubah_*.",
			InputSchema: aiSchema(map[string]any{
				"jenis": aiEnumProp("Bagian yang diambil (default semua).", "semua", "pembinaan", "rapat", "todo", "template_wa"),
			}),
			Roles: aiRolesKoordinator,
			run:   s.toolDaftarAgendaKoordinator,
		},
		{
			Name: "buat_pembinaan", Title: "Buat jadwal pembinaan guru",
			Description: "TULIS. Jadwalkan sesi pembinaan guru seperti form di halaman Pembinaan.",
			InputSchema: aiSchema(pembinaanProps, "tanggal", "topik"),
			Roles:       aiRolesKoordinator, Write: true, run: s.toolBuatPembinaan,
		},
		{
			Name: "ubah_pembinaan", Title: "Ubah jadwal pembinaan guru",
			Description: "TULIS. Ubah sesi pembinaan; field yang tidak diisi tetap. Untuk membatalkan pakai status libur (tidak dihapus).",
			InputSchema: aiSchema(withID("pembinaan_id", "id dari daftar_agenda_koordinator.", pembinaanProps), "pembinaan_id"),
			Roles:       aiRolesKoordinator, Write: true, Destructive: true, run: s.toolUbahPembinaan,
		},
		{
			Name: "buat_rapat", Title: "Buat jadwal rapat guru",
			Description: "TULIS. Jadwalkan rapat guru seperti form di halaman Rapat Guru.",
			InputSchema: aiSchema(rapatProps, "tanggal", "judul"),
			Roles:       aiRolesKoordinator, Write: true, run: s.toolBuatRapat,
		},
		{
			Name: "ubah_rapat", Title: "Ubah jadwal rapat guru",
			Description: "TULIS. Ubah rapat; field yang tidak diisi tetap. Untuk membatalkan pakai status batal (tidak dihapus).",
			InputSchema: aiSchema(withID("rapat_id", "id dari daftar_agenda_koordinator.", rapatProps), "rapat_id"),
			Roles:       aiRolesKoordinator, Write: true, Destructive: true, run: s.toolUbahRapat,
		},
		{
			Name: "buat_todo_koordinator", Title: "Buat todo koordinator",
			Description: "TULIS. Tambah tugas ke Todo Koordinator.",
			InputSchema: aiSchema(todoProps, "judul"),
			Roles:       aiRolesKoordinator, Write: true, run: s.toolBuatTodo,
		},
		{
			Name: "ubah_todo_koordinator", Title: "Ubah todo koordinator",
			Description: "TULIS. Ubah tugas atau statusnya (mis. selesai); field yang tidak diisi tetap.",
			InputSchema: aiSchema(withID("todo_id", "id dari daftar_agenda_koordinator.", todoProps), "todo_id"),
			Roles:       aiRolesKoordinator, Write: true, Destructive: true, run: s.toolUbahTodo,
		},
		{
			Name: "buat_template_wa", Title: "Buat template WA koordinator",
			Description: "TULIS. Tambah template pesan WhatsApp untuk santri atau guru (dipakai tombol Sapa/WA di dashboard).",
			InputSchema: aiSchema(templateProps, "nama", "isi"),
			Roles:       aiRolesKoordinator, Write: true, run: s.toolBuatTemplateWA,
		},
		{
			Name: "ubah_template_wa", Title: "Ubah template WA koordinator",
			Description: "TULIS. Ubah template WA; field yang tidak diisi tetap. aktif=false menonaktifkan (tidak dihapus).",
			InputSchema: aiSchema(withID("template_id", "id dari daftar_agenda_koordinator.", templateProps), "template_id"),
			Roles:       aiRolesKoordinator, Write: true, Destructive: true, run: s.toolUbahTemplateWA,
		},
	}
}

// --- Sumber lead ---------------------------------------------------------------

func (s *AIAgentService) sumberByID(ctx context.Context, id int64) (*models.LeadSourceResponse, error) {
	srcs, err := s.leads.Sources(ctx)
	if err != nil {
		return nil, err
	}
	for i := range srcs {
		if srcs[i].ID == id {
			return &srcs[i], nil
		}
	}
	return nil, aiArgErr("sumber_id %d tidak ditemukan; lihat daftar_sumber_lead", id)
}

func (s *AIAgentService) toolBuatSumberLead(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	nama, err := a.teks("nama", "nama", true, 100)
	if err != nil {
		return nil, err
	}
	pemicu, err := a.strList("teks_pemicu")
	if err != nil {
		return nil, err
	}
	if len(pemicu) > 10 {
		return nil, aiArgErr("teks_pemicu maksimal 10")
	}
	aktif := true
	if a.ada("aktif") {
		aktif = a.bool("aktif")
	}
	id, err := s.leads.CreateSource(ctx, models.LeadSourceRequest{Nama: nama, URL: a.str("url"), Keterangan: a.str("keterangan"), IsAktif: aktif})
	if err != nil {
		return nil, aiErrLayanan(err)
	}
	hasil := make([]map[string]any, 0, len(pemicu))
	for _, t := range pemicu {
		if t == "" {
			continue
		}
		r := map[string]any{"teks": t, "ok": true}
		if err := s.leads.AddSourcePattern(ctx, id, t); err != nil {
			var ae aiArgError
			if e := aiErrLayanan(err); errors.As(e, &ae) {
				r["ok"], r["pesan"] = false, ae.msg
			} else {
				return nil, e
			}
		}
		hasil = append(hasil, r)
	}
	return map[string]any{
		"ok": true, "sumber_id": id, "nama": nama, "aktif": aktif, "teks_pemicu": hasil,
		"saran": "Panggil cocokkan_ulang_sumber_lead bila lead lama perlu ikut tercocokkan.",
		"link":  aiLink(p, "/app/leads/sumber"),
	}, nil
}

func (s *AIAgentService) toolUbahSumberLead(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.int("sumber_id", 0, 1, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	src, err := s.sumberByID(ctx, id)
	if err != nil {
		return nil, err
	}
	req := models.LeadSourceRequest{
		Nama: a.strOr("nama", src.Nama), URL: a.strOr("url", src.URL), Keterangan: a.strOr("keterangan", src.Keterangan), IsAktif: a.boolOr("aktif", src.IsAktif),
	}
	if err := s.leads.UpdateSource(ctx, id, req); err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{
		"ok": true, "sumber_id": id,
		"sebelum": map[string]any{"nama": src.Nama, "url": src.URL, "keterangan": src.Keterangan, "aktif": src.IsAktif},
		"sesudah": map[string]any{"nama": req.Nama, "url": req.URL, "keterangan": req.Keterangan, "aktif": req.IsAktif},
		"link":    aiLink(p, "/app/leads/sumber"),
	}, nil
}

func (s *AIAgentService) toolTambahTeksPemicu(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.int("sumber_id", 0, 1, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	teks, err := a.teks("teks", "teks", true, 300)
	if err != nil {
		return nil, err
	}
	if _, err := s.sumberByID(ctx, id); err != nil {
		return nil, err
	}
	if err := s.leads.AddSourcePattern(ctx, id, teks); err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{"ok": true, "sumber_id": id, "teks": teks, "link": aiLink(p, "/app/leads/sumber")}, nil
}

func (s *AIAgentService) toolHapusTeksPemicu(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.int("pola_id", 0, 1, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	if err := s.leads.DeleteSourcePattern(ctx, id); err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{"ok": true, "pola_id": id, "link": aiLink(p, "/app/leads/sumber")}, nil
}

func (s *AIAgentService) toolCocokkanUlangSumber(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	n, err := s.leads.RematchSources(ctx)
	if err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{"ok": true, "lead_tercocokkan": n, "link": aiLink(p, "/app/leads/sumber")}, nil
}

// --- Label ----------------------------------------------------------------------

func (s *AIAgentService) toolBuatLabelLead(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	nama, err := a.teks("nama", "nama", true, 60)
	if err != nil {
		return nil, err
	}
	warna, err := a.enum("warna", "neutral", "brand", "secondary", "neutral")
	if err != nil {
		return nil, err
	}
	if err := s.leads.SaveLabel(ctx, 0, nama, warna); err != nil {
		return nil, aiErrLayanan(err)
	}
	var labelID int64
	if labels, err := s.querier.ListLeadLabels(ctx); err == nil {
		for _, lb := range labels {
			if strings.EqualFold(lb.Nama, strings.Join(strings.Fields(nama), " ")) {
				labelID = lb.ID
			}
		}
	}
	return map[string]any{"ok": true, "label_id": labelID, "nama": nama, "warna": warna}, nil
}

// --- Produk CRM -----------------------------------------------------------------

func (s *AIAgentService) toolDaftarProduk(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	items, err := s.produk.Products()
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		batch := make([]map[string]any, 0, len(it.Batches))
		for _, b := range it.Batches {
			batch = append(batch, map[string]any{"batch_id": b.ID, "nama": b.Nama, "mulai": b.TanggalMulai})
		}
		out = append(out, map[string]any{
			"produk_id": it.ID, "nama": it.Nama, "kategori": it.Kategori, "harga": it.Harga, "aktif": it.IsAktif,
			"track_stok": it.TrackStok, "stok": it.Stok, "minimum_stok": it.MinimumStock, "batch": batch,
		})
	}
	return map[string]any{"data_per": aiWIB(s.now()), "jumlah": len(out), "produk": out, "link": aiLink(p, "/app/produk-crm")}, nil
}

func (s *AIAgentService) toolBuatProduk(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	nama, err := a.teks("nama", "nama", true, 120)
	if err != nil {
		return nil, err
	}
	kategori, err := a.enum("kategori", "", "buku", "program")
	if err != nil || kategori == "" {
		return nil, aiArgErr("kategori wajib: buku atau program")
	}
	harga, err := a.int("harga", 0, 0, math.MaxInt32)
	if err != nil {
		return nil, err
	}
	minStok, err := a.int("minimum_stok", 0, 0, math.MaxInt32)
	if err != nil {
		return nil, err
	}
	req := models.CreateProductRequest{Nama: nama, Kategori: kategori, TrackStok: a.bool("track_stok"), MinimumStock: minStok, Harga: harga}
	if err := s.produk.CreateProduct(req); err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{"ok": true, "nama": nama, "kategori": kategori, "harga": harga, "link": aiLink(p, "/app/produk-crm")}, nil
}

func (s *AIAgentService) toolUbahProduk(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.int("produk_id", 0, 1, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	items, err := s.produk.Products()
	if err != nil {
		return nil, err
	}
	var cur *models.Product
	for i := range items {
		if items[i].ID == id {
			cur = &items[i]
		}
	}
	if cur == nil {
		return nil, aiArgErr("produk_id %d tidak ditemukan; lihat daftar_produk_crm", id)
	}
	kategori := cur.Kategori
	if a.ada("kategori") {
		if kategori, err = a.enum("kategori", cur.Kategori, "buku", "program"); err != nil {
			return nil, err
		}
	}
	harga, err := a.int("harga", cur.Harga, 0, math.MaxInt32)
	if err != nil {
		return nil, err
	}
	minStok, err := a.int("minimum_stok", cur.MinimumStock, 0, math.MaxInt32)
	if err != nil {
		return nil, err
	}
	req := models.UpdateProductRequest{
		Nama: a.strOr("nama", cur.Nama), Kategori: kategori, TrackStok: a.boolOr("track_stok", cur.TrackStok),
		IsAktif: a.boolOr("aktif", cur.IsAktif), MinimumStock: minStok, Harga: harga,
	}
	if err := s.produk.UpdateProduct(id, req); err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{
		"ok": true, "produk_id": id,
		"sebelum": map[string]any{"nama": cur.Nama, "kategori": cur.Kategori, "harga": cur.Harga, "aktif": cur.IsAktif},
		"sesudah": map[string]any{"nama": req.Nama, "kategori": req.Kategori, "harga": req.Harga, "aktif": req.IsAktif},
		"link":    aiLink(p, "/app/produk-crm"),
	}, nil
}

// --- Koordinator Guru -------------------------------------------------------------

func (s *AIAgentService) toolDaftarAgendaKoordinator(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	jenis, err := a.enum("jenis", "semua", "semua", "pembinaan", "rapat", "todo", "template_wa")
	if err != nil {
		return nil, err
	}
	out := map[string]any{"data_per": aiWIB(s.now())}
	if jenis == "semua" || jenis == "pembinaan" {
		list, err := s.koordinator.ListPembinaan()
		if err != nil {
			return nil, err
		}
		rows := make([]map[string]any, 0, len(list))
		for _, x := range list {
			rows = append(rows, map[string]any{"pembinaan_id": x.ID, "tanggal": x.Tanggal, "topik": x.Topik, "bulan": x.Bulan, "pekan_ke": x.PekanKe, "keterangan": x.Keterangan, "status": x.Status})
		}
		out["pembinaan"], out["link_pembinaan"] = rows, aiLink(p, "/app/koordinator-guru/pembinaan")
	}
	if jenis == "semua" || jenis == "rapat" {
		list, err := s.koordinator.ListRapat()
		if err != nil {
			return nil, err
		}
		rows := make([]map[string]any, 0, len(list))
		for _, x := range list {
			rows = append(rows, map[string]any{"rapat_id": x.ID, "tanggal": x.Tanggal, "judul": x.Judul, "catatan": x.Catatan, "status": x.Status})
		}
		out["rapat"], out["link_rapat"] = rows, aiLink(p, "/app/koordinator-guru/rapat")
	}
	if jenis == "semua" || jenis == "todo" {
		list, err := s.koordinator.ListTodo()
		if err != nil {
			return nil, err
		}
		rows := make([]map[string]any, 0, len(list))
		for _, x := range list {
			rows = append(rows, map[string]any{"todo_id": x.ID, "judul": x.Judul, "deadline": x.Deadline, "pic": x.Pic, "status": x.Status, "recurring": x.Recurring})
		}
		out["todo"], out["link_todo"] = rows, aiLink(p, "/app/koordinator-guru/todo")
	}
	if jenis == "semua" || jenis == "template_wa" {
		list, err := s.koordinator.ListWaTemplate()
		if err != nil {
			return nil, err
		}
		rows := make([]map[string]any, 0, len(list))
		for _, x := range list {
			rows = append(rows, map[string]any{"template_id": x.ID, "nama": x.Nama, "target": x.TargetType, "isi": x.Body, "aktif": x.IsAktif})
		}
		out["template_wa"], out["link_template_wa"] = rows, aiLink(p, "/app/koordinator-guru/wa-template")
	}
	return out, nil
}

// pembinaanReq builds a request from the arguments over the current values.
func pembinaanReq(a aiArgs, cur models.PembinaanResponse, baru bool) (models.PembinaanRequest, error) {
	tanggal, err := aiTanggal(a.strOr("tanggal", cur.Tanggal), "tanggal", true)
	if err != nil {
		return models.PembinaanRequest{}, err
	}
	topik := a.strOr("topik", cur.Topik)
	if topik == "" {
		return models.PembinaanRequest{}, aiArgErr("topik wajib diisi")
	}
	status, err := a.enum("status", cur.Status, "dijadwalkan", "terlaksana", "libur")
	if err != nil {
		return models.PembinaanRequest{}, err
	}
	pekan, err := a.int("pekan_ke", cur.PekanKe, 1, 6)
	if err != nil {
		return models.PembinaanRequest{}, err
	}
	bulan := a.strOr("bulan", cur.Bulan)
	t, _ := time.Parse("2006-01-02", tanggal)
	// A new session (or a moved date) gets bulan/pekan from its date unless given.
	if (baru || a.ada("tanggal")) && !a.ada("bulan") {
		bulan = aiNamaBulan[t.Month()]
	}
	if (baru || a.ada("tanggal")) && !a.ada("pekan_ke") {
		pekan = int64((t.Day()-1)/7 + 1)
	}
	return models.PembinaanRequest{Tanggal: tanggal, Bulan: bulan, PekanKe: pekan, Topik: topik, Keterangan: a.strOr("keterangan", cur.Keterangan), Status: status}, nil
}

func (s *AIAgentService) toolBuatPembinaan(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	req, err := pembinaanReq(a, models.PembinaanResponse{Status: "dijadwalkan"}, true)
	if err != nil {
		return nil, err
	}
	id, err := s.koordinator.CreatePembinaan(req)
	if err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{"ok": true, "pembinaan_id": id, "pembinaan": req, "link": aiLink(p, "/app/koordinator-guru/pembinaan")}, nil
}

func (s *AIAgentService) toolUbahPembinaan(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.int("pembinaan_id", 0, 1, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	cur, err := s.koordinator.GetPembinaan(id)
	if err != nil {
		return nil, aiArgErr("pembinaan_id %d tidak ditemukan", id)
	}
	req, err := pembinaanReq(a, *cur, false)
	if err != nil {
		return nil, err
	}
	if err := s.koordinator.UpdatePembinaan(id, req); err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{"ok": true, "pembinaan_id": id, "sebelum": cur, "sesudah": req, "link": aiLink(p, "/app/koordinator-guru/pembinaan")}, nil
}

func rapatReq(a aiArgs, cur models.RapatResponse) (models.RapatRequest, error) {
	tanggal, err := aiTanggal(a.strOr("tanggal", cur.Tanggal), "tanggal", true)
	if err != nil {
		return models.RapatRequest{}, err
	}
	judul := a.strOr("judul", cur.Judul)
	if judul == "" {
		return models.RapatRequest{}, aiArgErr("judul wajib diisi")
	}
	status, err := a.enum("status", cur.Status, "dijadwalkan", "terlaksana", "batal")
	if err != nil {
		return models.RapatRequest{}, err
	}
	return models.RapatRequest{Tanggal: tanggal, Judul: judul, Catatan: a.strOr("catatan", cur.Catatan), Status: status}, nil
}

func (s *AIAgentService) toolBuatRapat(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	req, err := rapatReq(a, models.RapatResponse{Status: "dijadwalkan"})
	if err != nil {
		return nil, err
	}
	id, err := s.koordinator.CreateRapat(req)
	if err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{"ok": true, "rapat_id": id, "rapat": req, "link": aiLink(p, "/app/koordinator-guru/rapat")}, nil
}

func (s *AIAgentService) toolUbahRapat(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.int("rapat_id", 0, 1, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	cur, err := s.koordinator.GetRapat(id)
	if err != nil {
		return nil, aiArgErr("rapat_id %d tidak ditemukan", id)
	}
	req, err := rapatReq(a, *cur)
	if err != nil {
		return nil, err
	}
	if err := s.koordinator.UpdateRapat(id, req); err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{"ok": true, "rapat_id": id, "sebelum": cur, "sesudah": req, "link": aiLink(p, "/app/koordinator-guru/rapat")}, nil
}

func todoReq(a aiArgs, cur models.TodoResponse) (models.TodoRequest, error) {
	judul := a.strOr("judul", cur.Judul)
	if judul == "" {
		return models.TodoRequest{}, aiArgErr("judul wajib diisi")
	}
	deadline, err := aiTanggal(a.strOr("deadline", cur.Deadline), "deadline", false)
	if err != nil {
		return models.TodoRequest{}, err
	}
	status, err := a.enum("status", cur.Status, "belum", "proses", "selesai", "batal")
	if err != nil {
		return models.TodoRequest{}, err
	}
	rec, err := a.enum("recurring", cur.Recurring, "none", "harian", "mingguan", "bulanan", "4bulanan")
	if err != nil {
		return models.TodoRequest{}, err
	}
	return models.TodoRequest{
		Judul: judul, Teknis: a.strOr("teknis", cur.Teknis), Kebutuhan: a.strOr("kebutuhan", cur.Kebutuhan), Deadline: deadline,
		Pic: a.strOr("pic", cur.Pic), Status: status, Recurring: rec, LinkPendukung: a.strOr("link_pendukung", cur.LinkPendukung), Catatan: a.strOr("catatan", cur.Catatan),
	}, nil
}

func (s *AIAgentService) toolBuatTodo(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	req, err := todoReq(a, models.TodoResponse{Status: "belum", Recurring: "none"})
	if err != nil {
		return nil, err
	}
	id, err := s.koordinator.CreateTodo(req)
	if err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{"ok": true, "todo_id": id, "todo": req, "link": aiLink(p, "/app/koordinator-guru/todo")}, nil
}

func (s *AIAgentService) toolUbahTodo(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.int("todo_id", 0, 1, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	list, err := s.koordinator.ListTodo()
	if err != nil {
		return nil, err
	}
	var cur *models.TodoResponse
	for i := range list {
		if list[i].ID == id {
			cur = &list[i]
		}
	}
	if cur == nil {
		return nil, aiArgErr("todo_id %d tidak ditemukan", id)
	}
	req, err := todoReq(a, *cur)
	if err != nil {
		return nil, err
	}
	if err := s.koordinator.UpdateTodo(id, req); err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{"ok": true, "todo_id": id, "sebelum": cur, "sesudah": req, "link": aiLink(p, "/app/koordinator-guru/todo")}, nil
}

func templateReq(a aiArgs, cur models.WaTemplateResponse) (models.WaTemplateRequest, error) {
	nama := a.strOr("nama", cur.Nama)
	isi := a.strOr("isi", cur.Body)
	if nama == "" || isi == "" {
		return models.WaTemplateRequest{}, aiArgErr("nama dan isi template wajib diisi")
	}
	if utf8.RuneCountInString(nama) > 100 || utf8.RuneCountInString(isi) > 4000 {
		return models.WaTemplateRequest{}, aiArgErr("nama maksimal 100 karakter dan isi maksimal 4000 karakter")
	}
	target, err := a.enum("target", cur.TargetType, "santri", "guru")
	if err != nil {
		return models.WaTemplateRequest{}, err
	}
	return models.WaTemplateRequest{Nama: nama, TargetType: target, Body: isi, IsAktif: a.boolOr("aktif", cur.IsAktif)}, nil
}

func (s *AIAgentService) toolBuatTemplateWA(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	req, err := templateReq(a, models.WaTemplateResponse{TargetType: "santri", IsAktif: true})
	if err != nil {
		return nil, err
	}
	id, err := s.koordinator.CreateWaTemplate(req)
	if err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{"ok": true, "template_id": id, "template": req, "link": aiLink(p, "/app/koordinator-guru/wa-template")}, nil
}

func (s *AIAgentService) toolUbahTemplateWA(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.int("template_id", 0, 1, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	list, err := s.koordinator.ListWaTemplate()
	if err != nil {
		return nil, err
	}
	var cur *models.WaTemplateResponse
	for i := range list {
		if list[i].ID == id {
			cur = &list[i]
		}
	}
	if cur == nil {
		return nil, aiArgErr("template_id %d tidak ditemukan", id)
	}
	req, err := templateReq(a, *cur)
	if err != nil {
		return nil, err
	}
	if err := s.koordinator.UpdateWaTemplate(id, req); err != nil {
		return nil, aiErrLayanan(err)
	}
	return map[string]any{"ok": true, "template_id": id, "sebelum": cur, "sesudah": req, "link": aiLink(p, "/app/koordinator-guru/wa-template")}, nil
}

// aiTulisRingkas lists the write tools a principal can use, for the
// connection instructions.
func (s *AIAgentService) aiTulisRingkas(p *models.AIPrincipal) string {
	names := []string{}
	for _, t := range s.ToolsFor(p.Role, p.CanWrite) {
		if t.Write {
			names = append(names, t.Name)
		}
	}
	return strings.Join(names, ", ")
}
