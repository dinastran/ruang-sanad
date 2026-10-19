package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// Role sets mirror the route guards in routes/web.go for the same data.
var (
	aiRolesRingkasan   = []string{"super_admin", "admin", "keuangan"}
	aiRolesKeuangan    = []string{"super_admin", "admin", "keuangan"}
	aiRolesRiayah      = []string{"super_admin", "admin_kelas", "koordinator_guru"}
	aiRolesSantri      = []string{"super_admin", "admin_kelas", "koordinator_guru", "cs"}
	aiRolesManajemen   = []string{"super_admin"}
	aiRolesKoordinator = []string{"super_admin", "koordinator_guru"}
)

func aiHasRole(role string, roles []string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func aiCanRiayah(role string) bool    { return aiHasRole(role, aiRolesRiayah) }
func aiCanFinance(role string) bool   { return OverviewCanFinance(role) }
func aiCanAdminNote(role string) bool { return role == "super_admin" || role == "admin_kelas" }

// Notes each role may search; mirrors which dashboard pages show them.
var aiCatatanPerRole = map[string][]string{
	"super_admin":      queries.AICatatanSourceNames,
	"admin_kelas":      {"riayah", "kontak_riayah", "catatan_admin", "absensi", "pertemuan", "status_santri", "kalam"},
	"koordinator_guru": {"riayah", "kontak_riayah", "absensi", "pertemuan", "monitoring_kelas", "kunjungan", "tindak_lanjut", "pembinaan", "rapat", "kalam", "todo"},
}

func aiSantriLink(p *models.AIPrincipal, id int64) string {
	if aiCanRiayah(p.Role) {
		return aiLink(p, fmt.Sprintf("/app/guru/santri/%d", id))
	}
	return aiLink(p, fmt.Sprintf("/app/santri/%d", id))
}

func aiKelasLink(p *models.AIPrincipal, id int64) string {
	if p.Role == "koordinator_guru" {
		return aiLink(p, "/app/koordinator-guru/monitoring-kelas")
	}
	return aiLink(p, fmt.Sprintf("/app/kelas/%d", id))
}

func aiGuruLink(p *models.AIPrincipal, id int64) string {
	return aiLink(p, fmt.Sprintf("/app/koordinator-guru/guru/%d", id))
}

func aiTruncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

func aiRound(v float64) float64 { return math.Round(v*10) / 10 }

func aiRoundPtr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	r := aiRound(*v)
	return &r
}

func aiPercent(part, total int64) *float64 {
	if total <= 0 {
		return nil
	}
	v := aiRound(float64(part) * 100 / float64(total))
	return &v
}

func (s *AIAgentService) today() time.Time { return s.now().In(ZonaWaktuRiayah) }

func (s *AIAgentService) buildTools() []AITool {
	return append([]AITool{
		{
			Name:  "ringkasan_kondisi",
			Title: "Ringkasan kondisi Ruang Sanad",
			Description: "Gambaran menyeluruh satu bulan: daftar prioritas tindak lanjut, jumlah santri per status, tren pendaftar, program (tipe kelas), " +
				"operasional kelas/guru, riayah, produk, dan ringkasan keuangan (sesuai role). Pakai ini PERTAMA untuk pertanyaan seperti " +
				"\"apa masalah paling mendesak minggu ini?\", lalu telusuri dengan tool rincian.",
			InputSchema: aiSchema(map[string]any{
				"bulan": aiProp("string", "Bulan YYYY-MM. Kosong = bulan berjalan."),
			}),
			Roles: aiRolesRingkasan,
			run:   s.toolRingkasan,
		},
		{
			Name:  "santri_perlu_tindak_lanjut",
			Title: "Santri yang perlu ditindaklanjuti",
			Description: "Daftar santri aktif yang perlu ditindaklanjuti beserta alasan dan bukti: kehadiran rendah/alpa beruntun, lama tidak dikontak, " +
				"progres materi macet (penanda riayah), serta tunggakan SPP bila role boleh melihat keuangan. Urut dari paling mendesak.",
			InputSchema: aiSchema(map[string]any{
				"penanda":  aiEnumProp("Saring jenis alasan. Kosong = semua.", "kehadiran", "kontak", "progres", "tunggakan"),
				"kelas_id": aiIntProp("Saring satu kelas.", 1, math.MaxInt32),
				"guru":     aiProp("string", "Saring nama guru (sebagian nama boleh)."),
				"batas":    aiIntProp("Jumlah maksimal baris (default 25).", 1, 100),
			}),
			Roles: aiRolesRiayah,
			run:   s.toolSantriTindakLanjut,
		},
		{
			Name:        "cari_santri",
			Title:       "Cari santri",
			Description: "Cari santri berdasarkan nama atau ID mahasantri. Mengembalikan santri_id untuk dipakai di detail_santri.",
			InputSchema: aiSchema(map[string]any{
				"kata":   aiProp("string", "Nama atau ID mahasantri (minimal 2 huruf)."),
				"status": aiEnumProp("Saring status santri.", "aktif", "cuti", "nonaktif", "tidak_lanjut"),
				"batas":  aiIntProp("Jumlah maksimal (default 20).", 1, 50),
			}, "kata"),
			Roles: aiRolesSantri,
			run:   s.toolCariSantri,
		},
		{
			Name:  "detail_santri",
			Title: "Detail santri",
			Description: "Profil satu santri (tanpa kontak) dengan riwayat status, rekap kehadiran, penanda riayah, timeline pertemuan/catatan/kontak, " +
				"catatan admin, dan tagihan — bagian yang tampil mengikuti role.",
			InputSchema: aiSchema(map[string]any{
				"santri_id": aiIntProp("ID santri (dari cari_santri atau tool lain).", 1, math.MaxInt32),
			}, "santri_id"),
			Roles: aiRolesSantri,
			run:   s.toolDetailSantri,
		},
		{
			Name:  "tren_program",
			Title: "Tren program",
			Description: "Pertumbuhan atau pelemahan program per bulan. Dimensi level/tipe/frekuensi/angkatan: santri per status, pendaftar baru, " +
				"berhenti/cuti, dan persentase kehadiran per bulan. Dimensi produk: penjualan (qty, omzet) dan penugasan produk ke mahasantri.",
			InputSchema: aiSchema(map[string]any{
				"dimensi": aiEnumProp("Cara mengelompokkan program.", "level", "tipe", "frekuensi", "angkatan", "produk"),
				"bulan":   aiIntProp("Jumlah bulan ke belakang termasuk bulan berjalan (default 6).", 2, 24),
			}, "dimensi"),
			Roles: aiRolesManajemen,
			run:   s.toolTrenProgram,
		},
		{
			Name:  "rekap_tunggakan",
			Title: "Rekap tunggakan SPP",
			Description: "Total piutang, umur tunggakan (aging), tren kolektibilitas 6 bulan, piutang per kelas/angkatan, dan daftar prioritas " +
				"follow-up (paling lama telat, nominal terbesar, paling jarang di-follow-up).",
			InputSchema: aiSchema(map[string]any{
				"bulan": aiProp("string", "Bulan acuan tren YYYY-MM. Kosong = bulan berjalan."),
				"batas": aiIntProp("Jumlah baris prioritas follow-up (default 20).", 1, 100),
			}),
			Roles: aiRolesKeuangan,
			run:   s.toolRekapTunggakan,
		},
		{
			Name:  "kelas_perlu_perhatian",
			Title: "Kelas yang perlu perhatian",
			Description: "Kelas aktif dengan masalah: tanpa guru aktif, pertemuan lama belum ditutup, kehadiran 30 hari rendah, lama tanpa pertemuan, " +
				"banyak santri berpenanda riayah, kosong, atau melebihi kapasitas. Setiap kelas disertai alasan dan angka bukti.",
			InputSchema: aiSchema(map[string]any{
				"semua": aiProp("boolean", "true = tampilkan semua kelas aktif, termasuk yang tanpa masalah."),
				"batas": aiIntProp("Jumlah maksimal (default 30).", 1, 200),
			}),
			Roles: aiRolesRiayah,
			run:   s.toolKelasPerhatian,
		},
		{
			Name:        "detail_kelas",
			Title:       "Detail kelas",
			Description: "Satu kelas: metrik 30 hari, daftar santri aktif dengan kehadiran dan penanda riayah, 10 pertemuan terakhir, dan catatan monitoring/kunjungan terkait.",
			InputSchema: aiSchema(map[string]any{
				"kelas_id": aiIntProp("ID kelas.", 1, math.MaxInt32),
			}, "kelas_id"),
			Roles: aiRolesRiayah,
			run:   s.toolDetailKelas,
		},
		{
			Name:  "guru_perlu_perhatian",
			Title: "Guru yang perlu perhatian",
			Description: "Guru aktif dengan sinyal masalah: pertemuan belum ditutup, tindak lanjut kunjungan lewat target, cakupan riayah rendah, " +
				"nilai TSI < 70, kehadiran pembinaan/rapat rendah, atau banyak santri berpenanda. Disertai angka bukti.",
			InputSchema: aiSchema(map[string]any{
				"bulan_tsi": aiProp("string", "Bulan TSI YYYY-MM. Kosong = bulan terakhir yang punya data TSI."),
				"semua":     aiProp("boolean", "true = tampilkan semua guru aktif."),
				"batas":     aiIntProp("Jumlah maksimal (default 30).", 1, 200),
			}),
			Roles: aiRolesKoordinator,
			run:   s.toolGuruPerhatian,
		},
		{
			Name:        "detail_guru",
			Title:       "Detail guru",
			Description: "Satu guru (tanpa kontak): metrik mengajar, kelas yang diampu, riayah, nilai TSI 3 bulan terakhir, kunjungan kelas, dan catatan terkait.",
			InputSchema: aiSchema(map[string]any{
				"guru_id": aiIntProp("ID guru.", 1, math.MaxInt32),
			}, "guru_id"),
			Roles: aiRolesKoordinator,
			run:   s.toolDetailGuru,
		},
		{
			Name:  "cari_catatan",
			Title: "Cari catatan",
			Description: "Pencarian teks di catatan dashboard: riayah, kontak riayah, catatan admin santri, catatan absensi/pertemuan, monitoring kelas, " +
				"kunjungan & tindak lanjut, perubahan status santri, pembinaan, rapat, kalam bersanad, todo koordinator. Semua kata harus muncul.",
			InputSchema: aiSchema(map[string]any{
				"kata":      aiProp("string", "Kata kunci (dipisah spasi; semua kata harus ada)."),
				"sumber":    aiProp("string", "Opsional, daftar sumber dipisah koma: "+strings.Join(queries.AICatatanSourceNames, ", ")),
				"sejak":     aiProp("string", "Opsional, tanggal YYYY-MM-DD."),
				"santri_id": aiIntProp("Opsional, hanya catatan santri ini.", 1, math.MaxInt32),
				"kelas_id":  aiIntProp("Opsional, hanya catatan kelas ini.", 1, math.MaxInt32),
				"guru_id":   aiIntProp("Opsional, hanya catatan guru ini.", 1, math.MaxInt32),
				"batas":     aiIntProp("Jumlah maksimal (default 20).", 1, 50),
			}),
			Roles: []string{"super_admin", "admin_kelas", "koordinator_guru"},
			run:   s.toolCariCatatan,
		},
	}, s.daftarTools()...)
}

// --- ringkasan_kondisi -------------------------------------------------------

// aiRincianPrioritas tells the AI which tool call lists the rows behind each
// overview priority (titles from OverviewService.Get).
var aiRincianPrioritas = map[string]map[string]any{
	"Data santri belum lengkap":            {"tool": "daftar_santri", "argumen": map[string]any{"status": "aktif", "belum_lengkap": true}},
	"Santri aktif belum punya kelas aktif": {"tool": "daftar_santri", "argumen": map[string]any{"status": "aktif", "tanpa_kelas_aktif": true}},
	"Kelas membutuhkan guru":               {"tool": "kelas_perlu_perhatian", "argumen": map[string]any{}},
	"Pertemuan lama belum ditutup":         {"tool": "kelas_perlu_perhatian", "argumen": map[string]any{}},
	"Santri membutuhkan perhatian":         {"tool": "santri_perlu_tindak_lanjut", "argumen": map[string]any{}},
	"Tindak lanjut guru jatuh tempo":       {"tool": "guru_perlu_perhatian", "argumen": map[string]any{}},
	"Tagihan melewati jatuh tempo":         {"tool": "daftar_tagihan", "argumen": map[string]any{"terlambat": true}},
}

func (s *AIAgentService) toolRingkasan(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	now := s.today()
	bulan, err := a.month("bulan", now)
	if err != nil {
		return nil, err
	}
	ov, err := s.overview.Get(ctx, bulan, p.Role)
	if err != nil {
		return nil, aiArgErr("%s", err.Error())
	}
	prioritas := make([]map[string]any, 0, len(ov.Priorities))
	for _, pr := range ov.Priorities {
		item := map[string]any{"judul": pr.Title, "detail": pr.Detail, "jumlah": pr.Count, "link": aiLink(p, pr.Href)}
		// Only point at a tool this role can call; otherwise say why there is
		// no drill-down instead of naming a tool the client will not list.
		if r, ok := aiRincianPrioritas[pr.Title]; ok {
			if s.toolAllowed(r["tool"].(string), p.Role) {
				item["rincian"] = r
			} else {
				item["rincian"] = map[string]any{"tersedia": false, "alasan": "rincian nama untuk prioritas ini tidak tersedia bagi role " + p.Role}
			}
		}
		prioritas = append(prioritas, item)
	}
	out := map[string]any{
		"data_per":    now.Format("2006-01-02 15:04") + " WIB",
		"bulan":       ov.Bulan,
		"bulan_label": ov.BulanLabel,
		"prioritas":   prioritas,
	}
	if ov.Santri != nil {
		out["santri"] = ov.Santri
		out["pendaftar_baru_per_bulan"] = ov.Growth
		out["program_per_tipe"] = ov.Programs
	}
	if ov.Operations != nil {
		out["operasional"] = map[string]any{
			"santri_aktif_data_belum_lengkap":     ov.Operations.Incomplete,
			"guru_aktif":                          ov.Operations.Teachers,
			"kelas_aktif":                         ov.Operations.Classes,
			"kelas_tanpa_guru_aktif":              ov.Operations.WithoutTeacher,
			"santri_aktif_tanpa_kelas_aktif":      ov.Operations.WithoutClass,
			"pertemuan_lama_belum_ditutup":        ov.Operations.OpenMeetings,
			"tindak_lanjut_kunjungan_jatuh_tempo": ov.Operations.FollowUps,
			"todo_koordinator_jatuh_tempo":        ov.Operations.Todos,
			"pertemuan_selesai_bulan_ini":         ov.Operations.Finished,
			"pertemuan_badal_bulan_ini":           ov.Operations.Badal,
			"pertemuan_reschedule_bulan_ini":      ov.Operations.Rescheduled,
		}
	}
	if ov.Riayah != nil {
		out["riayah"] = ov.Riayah
	}
	if ov.Products != nil {
		out["produk"] = ov.Products
	}
	if ov.Finance != nil {
		f := ov.Finance
		out["keuangan"] = map[string]any{
			"total_piutang":            f.TotalPiutang,
			"jumlah_tagihan_piutang":   f.JumlahPiutang,
			"jumlah_tagihan_terlambat": f.JumlahTerlambat,
			"aging":                    f.Aging,
			"bulan_ini":                f.Sekarang,
			"bulan_sebelumnya":         f.Sebelumnya,
			"ujroh":                    f.Ujroh,
			"produk":                   f.Produk,
			"rincian":                  "Pakai rekap_tunggakan untuk daftar santri yang perlu di-follow-up.",
		}
	}
	if p.Role == "super_admin" {
		if kelas, err := s.kelasPerhatian(ctx, p, now, false); err == nil {
			out["kelas_perlu_perhatian"] = map[string]any{"jumlah": len(kelas), "teratas": aiFirst(kelas, 5)}
		} else {
			ov.Errors = append(ov.Errors, "Kelas perlu perhatian")
		}
		if guru, _, err := s.guruPerhatian(ctx, p, now, "", false); err == nil {
			out["guru_perlu_perhatian"] = map[string]any{"jumlah": len(guru), "teratas": aiFirst(guru, 5)}
		} else {
			ov.Errors = append(ov.Errors, "Guru perlu perhatian")
		}
	}
	out["sumber_gagal_dimuat"] = ov.Errors
	out["langkah_lanjut"] = "Untuk nama-nama di balik angka, panggil tool di field 'rincian' tiap prioritas. " +
		"Bila tool itu tidak ada di daftar tool Anda, daftar tool di aplikasi AI sudah usang: minta pengguna menyambung ulang (restart/reconnect) server MCP Ruang Sanad."
	out["tool_untuk_role_ini"] = s.toolNames(p.Role)
	return out, nil
}

func aiFirst[T any](xs []T, n int) []T {
	if len(xs) <= n {
		return xs
	}
	return xs[:n]
}

// --- santri_perlu_tindak_lanjut ---------------------------------------------

type aiSantriTL struct {
	SantriID     int64                     `json:"santri_id"`
	Nama         string                    `json:"nama"`
	IDMahasantri string                    `json:"id_mahasantri"`
	Kelas        string                    `json:"kelas"`
	KelasID      int64                     `json:"kelas_id"`
	Level        string                    `json:"level"`
	Guru         string                    `json:"guru"`
	Skor         int                       `json:"skor_prioritas"`
	Alasan       []string                  `json:"alasan"`
	Bukti        map[string]any            `json:"bukti"`
	Tunggakan    *models.AITunggakanSantri `json:"tunggakan,omitempty"`
	Link         string                    `json:"link"`
}

func aiPenandaSkor(level string) int {
	switch level {
	case models.LevelMerah:
		return 3
	case models.LevelOranye:
		return 2
	default:
		return 1
	}
}

func (s *AIAgentService) toolSantriTindakLanjut(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	penanda, err := a.enum("penanda", "", "kehadiran", "kontak", "progres", "tunggakan")
	if err != nil {
		return nil, err
	}
	kelasID, err := a.int("kelas_id", 0, 0, math.MaxInt32)
	if err != nil {
		return nil, err
	}
	batas, err := a.int("batas", 25, 1, 100)
	if err != nil {
		return nil, err
	}
	guru := strings.ToLower(a.str("guru"))
	finance := aiCanFinance(p.Role)
	if penanda == "tunggakan" && !finance {
		return nil, aiArgErr("role %s tidak dapat melihat data tunggakan", p.Role)
	}
	now := s.today()
	items, ringkasan, err := s.riayah.ListPerhatian(models.RiayahViewer{}, now)
	if err != nil {
		return nil, err
	}
	var tunggakan map[int64]models.AITunggakanSantri
	if finance {
		if tunggakan, err = s.querier.AITunggakanPerSantri(ctx, now.Format("2006-01-02")); err != nil {
			return nil, err
		}
	}
	rows := make([]aiSantriTL, 0)
	for _, it := range items {
		if kelasID > 0 && it.KelasID != kelasID {
			continue
		}
		if guru != "" && !strings.Contains(strings.ToLower(it.GuruNama), guru) {
			continue
		}
		row := aiSantriTL{SantriID: it.ID, Nama: it.Nama, IDMahasantri: it.IDMahasantri, Kelas: it.NamaKelas, KelasID: it.KelasID,
			Level: it.Level, Guru: it.GuruNama, Alasan: []string{}, Link: aiSantriLink(p, it.ID),
			Bukti: map[string]any{
				"persen_hadir_30_hari":      aiRoundPtr(it.PersenHadir30),
				"pertemuan_30_hari":         it.TotalPertemuan30,
				"kontak_terakhir":           it.KontakTerakhir,
				"batas_materi_terakhir":     it.BatasMateriTerakhir,
				"rapor_bulan_lalu_terkirim": it.RaporTerkirim,
			}}
		match := penanda == ""
		for _, pn := range it.Penanda {
			row.Alasan = append(row.Alasan, pn.Kode+" ("+pn.Level+"): "+pn.Alasan)
			row.Skor += aiPenandaSkor(pn.Level)
			if pn.Kode == penanda {
				match = true
			}
		}
		if t, ok := tunggakan[it.ID]; ok && t.HariTelatMaks >= s.ambang.TunggakanHariPrioritas {
			tt := t
			row.Tunggakan = &tt
			row.Alasan = append(row.Alasan, fmt.Sprintf("tunggakan: %d tagihan Rp%d, terlama %d hari", t.JumlahTagihan, t.Nominal, t.HariTelatMaks))
			if t.HariTelatMaks >= 2*s.ambang.TunggakanHariPrioritas {
				row.Skor += 3
			} else {
				row.Skor += 2
			}
			if penanda == "tunggakan" {
				match = true
			}
		}
		if len(row.Alasan) == 0 || !match {
			continue
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Skor != rows[j].Skor {
			return rows[i].Skor > rows[j].Skor
		}
		return strings.ToLower(rows[i].Nama) < strings.ToLower(rows[j].Nama)
	})
	total := len(rows)
	return map[string]any{
		"data_per": now.Format("2006-01-02"),
		"cakupan":  "Santri berstatus aktif. Tunggakan santri nonaktif/cuti ada di rekap_tunggakan.",
		"ambang": map[string]any{
			"kehadiran":          fmt.Sprintf("kehadiran 30 hari < %.0f%% (min. %d pertemuan) atau alpa %d kali berturut-turut", riayahBatasPersenHadir, riayahMinPertemuan, riayahAlpaBerturut),
			"kontak":             fmt.Sprintf("belum dikontak/disapa > %d hari", riayahBelumDisapaHari),
			"progres":            fmt.Sprintf("batas materi tidak bergerak %d pertemuan", riayahProgresMacetJumlah),
			"tunggakan":          fmt.Sprintf("tagihan telat >= %d hari", s.ambang.TunggakanHariPrioritas),
			"tunggakan_dihitung": finance,
		},
		"ringkasan_riayah": ringkasan,
		"total_cocok":      total,
		"ditampilkan":      min(int64(total), batas),
		"santri":           aiFirst(rows, int(batas)),
	}, nil
}

// --- cari_santri / detail_santri --------------------------------------------

func (s *AIAgentService) toolCariSantri(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	kata := a.str("kata")
	if utf8.RuneCountInString(kata) < 2 {
		return nil, aiArgErr("kata minimal 2 huruf")
	}
	status, err := a.enum("status", "", "aktif", "cuti", "nonaktif", "tidak_lanjut")
	if err != nil {
		return nil, err
	}
	batas, err := a.int("batas", 20, 1, 50)
	if err != nil {
		return nil, err
	}
	rows, err := s.querier.AICariSantri(ctx, kata, status, batas)
	if err != nil {
		return nil, err
	}
	hasil := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, map[string]any{"santri": r, "link": aiSantriLink(p, r.ID)})
	}
	return map[string]any{"kata": kata, "jumlah": len(hasil), "hasil": hasil}, nil
}

func (s *AIAgentService) toolDetailSantri(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.int("santri_id", 0, 1, math.MaxInt32)
	if err != nil || id == 0 {
		return nil, aiArgErr("santri_id wajib diisi")
	}
	prof, err := s.querier.AIGetSantriProfil(ctx, id)
	if err != nil {
		return nil, err
	}
	if prof == nil {
		return nil, aiArgErr("santri dengan id %d tidak ditemukan", id)
	}
	now := s.today()
	tidakTersedia := []string{}
	// Nominal SPP is on the santri page, which koordinator_guru cannot open.
	if !aiHasRole(p.Role, []string{"super_admin", "cs", "admin_kelas"}) {
		prof.Nominal = nil
		tidakTersedia = append(tidakTersedia, "nominal SPP")
	}
	out := map[string]any{"data_per": now.Format("2006-01-02"), "profil": prof, "link": aiSantriLink(p, id)}
	if logs, err := s.querier.AIListSantriStatusLog(ctx, id, 10); err == nil {
		out["riwayat_status"] = logs
	} else {
		return nil, err
	}
	if aiCanRiayah(p.Role) {
		profil, err := s.riayah.GetProfil(models.RiayahViewer{}, id, now)
		if err != nil {
			return nil, err
		}
		timeline := make([]map[string]any, 0)
		for _, t := range aiFirst(profil.Timeline, 30) {
			timeline = append(timeline, map[string]any{"jenis": t.Jenis, "tanggal": t.Tanggal, "judul": t.Judul, "status": t.Status,
				"isi": aiTruncate(t.Isi, 600), "materi": t.Materi, "batas_materi": t.Batas, "media": t.Media, "penulis": t.Penulis})
		}
		out["rekap_kehadiran_seluruhnya"] = profil.Rekap
		out["timeline_terbaru"] = timeline
		if items, _, err := s.riayah.ListPerhatian(models.RiayahViewer{}, now); err == nil {
			for _, it := range items {
				if it.ID == id {
					out["penanda_riayah"] = it.Penanda
					out["persen_hadir_30_hari"] = aiRoundPtr(it.PersenHadir30)
					out["kontak_terakhir"] = it.KontakTerakhir
					break
				}
			}
		}
	} else {
		tidakTersedia = append(tidakTersedia, "riayah & timeline")
	}
	if aiCanAdminNote(p.Role) {
		notes, err := s.santri.ListAdminNotes(id)
		if err != nil {
			return nil, err
		}
		catatan := make([]map[string]any, 0)
		for _, n := range aiFirst(notes, 20) {
			catatan = append(catatan, map[string]any{"tanggal": n.CreatedAt, "penulis": n.AuthorName, "catatan": aiTruncate(n.Catatan, 600)})
		}
		out["catatan_admin"] = catatan
	} else {
		tidakTersedia = append(tidakTersedia, "catatan admin")
	}
	if aiCanFinance(p.Role) {
		tagihan, err := s.querier.AIListTagihanSantri(ctx, id, now.Format("2006-01-02"), 12)
		if err != nil {
			return nil, err
		}
		out["tagihan_terbaru"] = tagihan
	} else {
		tidakTersedia = append(tidakTersedia, "tagihan")
	}
	out["tidak_tersedia_untuk_role_anda"] = tidakTersedia
	return out, nil
}

// --- tren_program -------------------------------------------------------------

func aiMonths(now time.Time, n int64) []string {
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -int(n-1), 0)
	out := make([]string, 0, n)
	for i := int64(0); i < n; i++ {
		out = append(out, start.AddDate(0, int(i), 0).Format("2006-01"))
	}
	return out
}

func (s *AIAgentService) toolTrenProgram(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	dim, err := a.enum("dimensi", "", "level", "tipe", "frekuensi", "angkatan", "produk")
	if err != nil {
		return nil, err
	}
	if dim == "" {
		return nil, aiArgErr("dimensi wajib diisi")
	}
	n, err := a.int("bulan", 6, 2, 24)
	if err != nil {
		return nil, err
	}
	now := s.today()
	months := aiMonths(now, n)
	dari := months[0]
	half := len(months) / 2
	out := map[string]any{"data_per": now.Format("2006-01-02"), "dimensi": dim, "bulan": months,
		"catatan_bulan_berjalan": "Bulan terakhir (" + months[len(months)-1] + ") belum selesai; jangan bandingkan langsung dengan bulan penuh."}

	// a,b,c,d,e: per-dimension monthly measures (meaning depends on dimensi).
	type cell struct{ a, b, c, d, e int64 }
	series := map[string]map[string]*cell{}
	get := func(nilai, bulan string) *cell {
		if series[nilai] == nil {
			series[nilai] = map[string]*cell{}
		}
		if series[nilai][bulan] == nil {
			series[nilai][bulan] = &cell{}
		}
		return series[nilai][bulan]
	}
	sumRange := func(nilai string, from, to int, f func(*cell) int64) int64 {
		var t int64
		for _, m := range months[from:to] {
			if c := series[nilai][m]; c != nil {
				t += f(c)
			}
		}
		return t
	}

	if dim == "produk" {
		jual, err := s.querier.AIProdukPenjualan(ctx, dari)
		if err != nil {
			return nil, err
		}
		tugas, err := s.querier.AIProdukPenugasan(ctx, dari)
		if err != nil {
			return nil, err
		}
		for _, r := range jual {
			c := get(r.Nilai, r.Bulan)
			c.a, c.b = r.Angka, r.Angka2
		}
		for _, r := range tugas {
			get(r.Nilai, r.Bulan).c = r.Angka
		}
		produk := make([]map[string]any, 0, len(series))
		for nilai := range series {
			perBulan := make([]map[string]any, 0, len(months))
			for _, m := range months {
				c := series[nilai][m]
				if c == nil {
					c = &cell{}
				}
				perBulan = append(perBulan, map[string]any{"bulan": m, "qty_terjual": c.a, "omzet": c.b, "penugasan_ke_mahasantri": c.c})
			}
			produk = append(produk, map[string]any{
				"produk":    nilai,
				"per_bulan": perBulan,
				"perbandingan": map[string]any{
					"periode_awal":  months[0] + " s.d. " + months[half-1],
					"periode_akhir": months[half] + " s.d. " + months[len(months)-1],
					"omzet_awal":    sumRange(nilai, 0, half, func(c *cell) int64 { return c.b }),
					"omzet_akhir":   sumRange(nilai, half, len(months), func(c *cell) int64 { return c.b }),
					"qty_awal":      sumRange(nilai, 0, half, func(c *cell) int64 { return c.a }),
					"qty_akhir":     sumRange(nilai, half, len(months), func(c *cell) int64 { return c.a }),
				},
			})
		}
		sort.Slice(produk, func(i, j int) bool { return produk[i]["produk"].(string) < produk[j]["produk"].(string) })
		out["produk"] = produk
		out["catatan_metode"] = "Penjualan dari pesanan aktif (tanggal pesanan). Penugasan = produk yang dicatat untuk mahasantri."
		return out, nil
	}

	snap, err := s.querier.AIProgramSnapshot(ctx, dim)
	if err != nil {
		return nil, err
	}
	daftar, err := s.querier.AIProgramPendaftar(ctx, dim, dari)
	if err != nil {
		return nil, err
	}
	berhenti, err := s.querier.AIProgramBerhenti(ctx, dim, dari)
	if err != nil {
		return nil, err
	}
	hadir, err := s.querier.AIProgramKehadiran(ctx, dim, dari)
	if err != nil {
		return nil, err
	}
	for _, r := range daftar {
		get(r.Nilai, r.Bulan).a = r.Angka
	}
	for _, r := range berhenti {
		c := get(r.Nilai, r.Bulan)
		c.b, c.c = r.Angka, r.Angka2
	}
	for _, r := range hadir {
		c := get(r.Nilai, r.Bulan)
		c.d, c.e = r.Angka, r.Angka2
	}
	program := make([]map[string]any, 0, len(snap))
	for _, sn := range snap {
		nilai := sn.Nilai
		perBulan := make([]map[string]any, 0, len(months))
		for _, m := range months {
			c := series[nilai][m]
			if c == nil {
				c = &cell{}
			}
			perBulan = append(perBulan, map[string]any{"bulan": m, "pendaftar_baru": c.a, "berhenti": c.b, "mulai_cuti": c.c,
				"absensi_tercatat": c.d, "hadir": c.e, "persen_hadir": aiPercent(c.e, c.d)})
		}
		hadirRange := func(from, to int) *float64 {
			h := sumRange(nilai, from, to, func(c *cell) int64 { return c.e })
			t := sumRange(nilai, from, to, func(c *cell) int64 { return c.d })
			return aiPercent(h, t)
		}
		program = append(program, map[string]any{
			"nilai":     nilai,
			"saat_ini":  sn,
			"per_bulan": perBulan,
			"perbandingan": map[string]any{
				"periode_awal":       months[0] + " s.d. " + months[half-1],
				"periode_akhir":      months[half] + " s.d. " + months[len(months)-1],
				"pendaftar_awal":     sumRange(nilai, 0, half, func(c *cell) int64 { return c.a }),
				"pendaftar_akhir":    sumRange(nilai, half, len(months), func(c *cell) int64 { return c.a }),
				"berhenti_awal":      sumRange(nilai, 0, half, func(c *cell) int64 { return c.b }),
				"berhenti_akhir":     sumRange(nilai, half, len(months), func(c *cell) int64 { return c.b }),
				"persen_hadir_awal":  hadirRange(0, half),
				"persen_hadir_akhir": hadirRange(half, len(months)),
			},
		})
	}
	out["program"] = program
	out["catatan_metode"] = "Pengelompokan memakai " + dim + " santri SAAT INI (riwayat perpindahan level/tipe tidak dilacak). " +
		"Pendaftar dari tanggal_daftar; berhenti/cuti dari log perubahan status; kehadiran = hadir+telat dari pertemuan selesai. " +
		"Nilai 'Belum diisi' berarti data santri belum lengkap."
	return out, nil
}

// --- rekap_tunggakan ------------------------------------------------------------

func (s *AIAgentService) toolRekapTunggakan(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	now := s.today()
	bulan, err := a.month("bulan", now)
	if err != nil {
		return nil, err
	}
	batas, err := a.int("batas", 20, 1, 100)
	if err != nil {
		return nil, err
	}
	f, err := s.finance.Get(bulan)
	if err != nil {
		return nil, aiArgErr("%s", err.Error())
	}
	rows, err := s.querier.KeuanganPrioritasFollowUp(ctx, queries.KeuanganPrioritasFollowUpParams{HariIni: f.HariIni, Batas: batas})
	if err != nil {
		return nil, err
	}
	prioritas := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		jatuh := r.TanggalTagih.Format("2006-01-02")
		if r.JatuhTempo.Valid {
			jatuh = r.JatuhTempo.Time.Format("2006-01-02")
		}
		fu := ""
		if r.FuTerakhir.Valid {
			fu = r.FuTerakhir.Time.Format("2006-01-02")
		}
		prioritas = append(prioritas, map[string]any{
			"tagihan_id": r.ID, "santri_id": r.SantriID, "nama": r.SantriNama, "id_mahasantri": r.IDMahasantri,
			"kelas": r.NamaKelas.String, "guru": r.GuruNama, "angkatan": r.Angkatan, "frekuensi": r.Frekuensi,
			"bulan_ke": r.BulanKe, "nominal": r.Nominal, "jatuh_tempo": jatuh, "hari_telat": r.HariTelat,
			"jumlah_follow_up": r.FuCount, "follow_up_terakhir": fu,
			"link_tagihan": aiLink(p, fmt.Sprintf("/app/keuangan/tagihan/%d", r.ID)),
		})
	}
	tren := make([]map[string]any, 0, len(f.Tren))
	for _, t := range f.Tren {
		tren = append(tren, map[string]any{"bulan": t.Bulan, "terbit_nominal": t.TerbitNominal, "terbit_lunas_nominal": t.TerbitLunas,
			"kolektibilitas_persen": aiRound(t.Kolektibilitas), "kas_masuk_spp": t.KasMasukSpp, "jumlah_bayar_spp": t.JumlahBayarSpp})
	}
	return map[string]any{
		"data_per":                 f.HariIni,
		"definisi_terlambat":       "status belum_bayar dan lewat jatuh tempo (atau tanggal tagih bila jatuh tempo kosong)",
		"total_piutang":            f.TotalPiutang,
		"jumlah_tagihan_piutang":   f.JumlahPiutang,
		"jumlah_tagihan_terlambat": f.JumlahTerlambat,
		"aging":                    f.Aging,
		"tren_kolektibilitas":      tren,
		"piutang_per_kelas":        f.PiutangKelas,
		"piutang_per_angkatan":     f.PiutangAngkatan,
		"prioritas_follow_up":      prioritas,
		"link":                     aiLink(p, "/app/keuangan?bulan="+bulan),
	}, nil
}

// --- kelas ----------------------------------------------------------------------

type aiKelasRow struct {
	Kelas           models.AIKelasMetrik `json:"kelas"`
	SantriPerhatian int64                `json:"santri_berpenanda_riayah"`
	Skor            int                  `json:"skor_prioritas"`
	Alasan          []string             `json:"alasan"`
	Link            string               `json:"link"`
}

func (s *AIAgentService) riayahPerKelas(now time.Time) (map[int64]int64, []models.RiayahSantriItem, error) {
	items, _, err := s.riayah.ListPerhatian(models.RiayahViewer{}, now)
	if err != nil {
		return nil, nil, err
	}
	out := map[int64]int64{}
	for _, it := range items {
		if len(it.Penanda) > 0 {
			out[it.KelasID]++
		}
	}
	return out, items, nil
}

func (s *AIAgentService) evaluateKelas(p *models.AIPrincipal, k models.AIKelasMetrik, flagged int64, now time.Time) aiKelasRow {
	row := aiKelasRow{Kelas: k, SantriPerhatian: flagged, Alasan: []string{}, Link: aiKelasLink(p, k.ID)}
	row.Kelas.PersenHadir30 = aiRoundPtr(k.PersenHadir30)
	add := func(skor int, format string, a ...any) {
		row.Skor += skor
		row.Alasan = append(row.Alasan, fmt.Sprintf(format, a...))
	}
	switch {
	case k.GuruID == 0:
		add(3, "belum ada guru")
	case !k.GuruAktif:
		add(3, "guru %s tidak aktif", k.GuruNama)
	}
	if k.PertemuanBelumTutup > 0 {
		add(2, "%d pertemuan sebelum hari ini masih berstatus berlangsung (belum ditutup)", k.PertemuanBelumTutup)
	}
	if k.SantriAktif == 0 {
		add(1, "kelas aktif tanpa santri aktif")
	}
	if k.SantriAktif > 0 {
		batas := now.AddDate(0, 0, -s.ambang.KelasTanpaPertemuanHari).Format("2006-01-02")
		if k.PertemuanTerakhir == "" {
			add(2, "belum ada pertemuan selesai yang tercatat")
		} else if k.PertemuanTerakhir < batas {
			add(2, "pertemuan selesai terakhir %s (lebih dari %d hari)", k.PertemuanTerakhir, s.ambang.KelasTanpaPertemuanHari)
		}
	}
	if k.PersenHadir30 != nil && k.Absensi30 >= 5 && *k.PersenHadir30 < s.ambang.KelasPersenHadirMin {
		add(2, "kehadiran 30 hari %.1f%% (%d dari %d absensi), di bawah %.0f%%", *k.PersenHadir30, k.Hadir30, k.Absensi30, s.ambang.KelasPersenHadirMin)
	}
	if k.SantriAktif > 0 && flagged >= 2 && float64(flagged)*100/float64(k.SantriAktif) >= s.ambang.KelasPersenSantriPerhatian {
		add(2, "%d dari %d santri aktif berpenanda riayah", flagged, k.SantriAktif)
	}
	if k.Kapasitas > 0 && k.SantriAktif > k.Kapasitas {
		add(1, "santri aktif %d melebihi kapasitas %d", k.SantriAktif, k.Kapasitas)
	}
	return row
}

func (s *AIAgentService) kelasPerhatian(ctx context.Context, p *models.AIPrincipal, now time.Time, semua bool) ([]aiKelasRow, error) {
	sejak := now.AddDate(0, 0, -30).Format("2006-01-02")
	metrik, err := s.querier.AIKelasMetrik(ctx, now.Format("2006-01-02"), sejak, 0, false)
	if err != nil {
		return nil, err
	}
	perKelas, _, err := s.riayahPerKelas(now)
	if err != nil {
		return nil, err
	}
	rows := make([]aiKelasRow, 0)
	for _, k := range metrik {
		row := s.evaluateKelas(p, k, perKelas[k.ID], now)
		if semua || len(row.Alasan) > 0 {
			rows = append(rows, row)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Skor > rows[j].Skor })
	return rows, nil
}

func (s *AIAgentService) toolKelasPerhatian(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	batas, err := a.int("batas", 30, 1, 200)
	if err != nil {
		return nil, err
	}
	now := s.today()
	rows, err := s.kelasPerhatian(ctx, p, now, a.bool("semua"))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"data_per": now.Format("2006-01-02"),
		"ambang": map[string]any{
			"kehadiran_min_persen":         s.ambang.KelasPersenHadirMin,
			"tanpa_pertemuan_hari":         s.ambang.KelasTanpaPertemuanHari,
			"santri_berpenanda_min_persen": s.ambang.KelasPersenSantriPerhatian,
		},
		"total": len(rows), "ditampilkan": min(int64(len(rows)), batas), "kelas": aiFirst(rows, int(batas)),
	}, nil
}

func (s *AIAgentService) toolDetailKelas(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.int("kelas_id", 0, 1, math.MaxInt32)
	if err != nil || id == 0 {
		return nil, aiArgErr("kelas_id wajib diisi")
	}
	now := s.today()
	metrik, err := s.querier.AIKelasMetrik(ctx, now.Format("2006-01-02"), now.AddDate(0, 0, -30).Format("2006-01-02"), id, false)
	if err != nil {
		return nil, err
	}
	if len(metrik) == 0 {
		return nil, aiArgErr("kelas dengan id %d tidak ditemukan", id)
	}
	perKelas, items, err := s.riayahPerKelas(now)
	if err != nil {
		return nil, err
	}
	santri := make([]map[string]any, 0)
	for _, it := range items {
		if it.KelasID != id {
			continue
		}
		santri = append(santri, map[string]any{"santri_id": it.ID, "nama": it.Nama, "persen_hadir_30_hari": aiRoundPtr(it.PersenHadir30),
			"pertemuan_30_hari": it.TotalPertemuan30, "batas_materi_terakhir": it.BatasMateriTerakhir, "kontak_terakhir": it.KontakTerakhir,
			"penanda": it.Penanda, "link": aiSantriLink(p, it.ID)})
	}
	pertemuan, err := s.querier.AIListPertemuanKelas(ctx, id, 10)
	if err != nil {
		return nil, err
	}
	for i := range pertemuan {
		pertemuan[i].Catatan = aiTruncate(pertemuan[i].Catatan, 400)
	}
	sumber := []string{"monitoring_kelas", "pertemuan"}
	if aiHasRole(p.Role, aiRolesKoordinator) {
		sumber = append(sumber, "kunjungan", "tindak_lanjut")
	}
	catatan, err := s.querier.AICariCatatan(ctx, queries.AICatatanFilter{Sumber: sumber, KelasID: id, Limit: 15})
	if err != nil {
		return nil, err
	}
	for i := range catatan {
		catatan[i].Isi = aiTruncate(catatan[i].Isi, 400)
	}
	return map[string]any{
		"data_per":           now.Format("2006-01-02"),
		"evaluasi":           s.evaluateKelas(p, metrik[0], perKelas[id], now),
		"santri_aktif":       santri,
		"pertemuan_terakhir": pertemuan,
		"catatan_terkait":    catatan,
	}, nil
}

// --- guru -------------------------------------------------------------------------

type aiGuruRow struct {
	Guru   models.AIGuruMetrik          `json:"guru"`
	Riayah *models.RiayahGuruMonitoring `json:"riayah,omitempty"`
	TSI    *models.TsiRekapRow          `json:"tsi,omitempty"`
	Skor   int                          `json:"skor_prioritas"`
	Alasan []string                     `json:"alasan"`
	Link   string                       `json:"link"`
}

// latestTSI returns the TSI recap for bulan, or for the current month and
// then the previous month when bulan is empty.
func (s *AIAgentService) latestTSI(now time.Time, bulan string) (string, map[int64]models.TsiRekapRow, error) {
	candidates := []string{bulan}
	if bulan == "" {
		candidates = []string{now.Format("2006-01"), now.AddDate(0, -1, 0).Format("2006-01")}
	}
	for _, b := range candidates {
		rows, err := s.tsi.GetRekap(b)
		if err != nil {
			return "", nil, err
		}
		if len(rows) > 0 || bulan != "" {
			out := map[int64]models.TsiRekapRow{}
			for _, r := range rows {
				out[r.GuruID] = r
			}
			return b, out, nil
		}
	}
	return "", map[int64]models.TsiRekapRow{}, nil
}

func (s *AIAgentService) evaluateGuru(p *models.AIPrincipal, g models.AIGuruMetrik, mon *models.RiayahGuruMonitoring, tsi *models.TsiRekapRow) aiGuruRow {
	row := aiGuruRow{Guru: g, Riayah: mon, TSI: tsi, Alasan: []string{}, Link: aiGuruLink(p, g.ID)}
	add := func(skor int, format string, a ...any) {
		row.Skor += skor
		row.Alasan = append(row.Alasan, fmt.Sprintf(format, a...))
	}
	if g.PertemuanBelumTutup > 0 {
		add(2, "%d pertemuan lama belum ditutup", g.PertemuanBelumTutup)
	}
	if g.TindakLanjutLewat > 0 {
		add(2, "%d tindak lanjut kunjungan lewat target", g.TindakLanjutLewat)
	}
	if tsi != nil && tsi.Total < s.ambang.GuruTSIMin {
		add(3, "nilai TSI %s %.1f (%s)", tsi.Bulan, tsi.Total, tsi.Predikat)
	}
	if mon != nil && mon.TotalSantri > 0 {
		if mon.CoverageRiayah < s.ambang.GuruCoverageRiayahMin {
			add(2, "cakupan riayah %.0f%% (%d santri belum disapa), di bawah %.0f%%", mon.CoverageRiayah, mon.BelumDisapa, s.ambang.GuruCoverageRiayahMin)
		}
		if mon.PerluPerhatian >= 2 && float64(mon.PerluPerhatian)*100/float64(mon.TotalSantri) >= s.ambang.KelasPersenSantriPerhatian {
			add(1, "%d dari %d santri berpenanda riayah", mon.PerluPerhatian, mon.TotalSantri)
		}
	}
	hadir, total := g.PembinaanHadir+g.RapatHadir, g.PembinaanTotal+g.RapatTotal
	if total >= 2 && float64(hadir)*100/float64(total) < s.ambang.GuruKehadiranForumMin {
		add(1, "hadir pembinaan/rapat %d dari %d (90 hari)", hadir, total)
	}
	if g.Aktif && g.KelasAktif == 0 {
		add(0, "guru aktif tanpa kelas aktif (info)")
	}
	return row
}

func (s *AIAgentService) guruPerhatian(ctx context.Context, p *models.AIPrincipal, now time.Time, bulanTSI string, semua bool) ([]aiGuruRow, string, error) {
	hariIni := now.Format("2006-01-02")
	metrik, err := s.querier.AIGuruMetrik(ctx, hariIni, now.AddDate(0, 0, -30).Format("2006-01-02"), now.AddDate(0, 0, -90).Format("2006-01-02"), 0, false)
	if err != nil {
		return nil, "", err
	}
	items, _, err := s.riayah.ListPerhatian(models.RiayahViewer{}, now)
	if err != nil {
		return nil, "", err
	}
	_, mons, err := s.riayah.MonitoringGuru(items, now)
	if err != nil {
		return nil, "", err
	}
	monBy := map[int64]models.RiayahGuruMonitoring{}
	for _, m := range mons {
		monBy[m.GuruID] = m
	}
	bulan, tsi, err := s.latestTSI(now, bulanTSI)
	if err != nil {
		return nil, "", err
	}
	rows := make([]aiGuruRow, 0)
	for _, g := range metrik {
		var mon *models.RiayahGuruMonitoring
		if m, ok := monBy[g.ID]; ok {
			mon = &m
		}
		var t *models.TsiRekapRow
		if r, ok := tsi[g.ID]; ok {
			t = &r
		}
		row := s.evaluateGuru(p, g, mon, t)
		if semua || row.Skor > 0 {
			rows = append(rows, row)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Skor > rows[j].Skor })
	return rows, bulan, nil
}

func (s *AIAgentService) toolGuruPerhatian(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	now := s.today()
	bulanTSI := a.str("bulan_tsi")
	if bulanTSI != "" {
		var err error
		if bulanTSI, err = a.month("bulan_tsi", now); err != nil {
			return nil, err
		}
	}
	batas, err := a.int("batas", 30, 1, 200)
	if err != nil {
		return nil, err
	}
	rows, bulan, err := s.guruPerhatian(ctx, p, now, bulanTSI, a.bool("semua"))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"data_per":  now.Format("2006-01-02"),
		"bulan_tsi": bulan,
		"ambang": map[string]any{
			"tsi_min":                    s.ambang.GuruTSIMin,
			"cakupan_riayah_min_persen":  s.ambang.GuruCoverageRiayahMin,
			"kehadiran_forum_min_persen": s.ambang.GuruKehadiranForumMin,
		},
		"total": len(rows), "ditampilkan": min(int64(len(rows)), batas), "guru": aiFirst(rows, int(batas)),
	}, nil
}

func (s *AIAgentService) toolDetailGuru(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.int("guru_id", 0, 1, math.MaxInt32)
	if err != nil || id == 0 {
		return nil, aiArgErr("guru_id wajib diisi")
	}
	now := s.today()
	hariIni := now.Format("2006-01-02")
	metrik, err := s.querier.AIGuruMetrik(ctx, hariIni, now.AddDate(0, 0, -30).Format("2006-01-02"), now.AddDate(0, 0, -90).Format("2006-01-02"), id, false)
	if err != nil {
		return nil, err
	}
	if len(metrik) == 0 {
		return nil, aiArgErr("guru dengan id %d tidak ditemukan", id)
	}
	items, _, err := s.riayah.ListPerhatian(models.RiayahViewer{}, now)
	if err != nil {
		return nil, err
	}
	_, mons, err := s.riayah.MonitoringGuru(items, now)
	if err != nil {
		return nil, err
	}
	var mon *models.RiayahGuruMonitoring
	for i := range mons {
		if mons[i].GuruID == id {
			mon = &mons[i]
		}
	}
	tsiRiwayat := make([]models.TsiRekapRow, 0, 3)
	var tsiTerakhir *models.TsiRekapRow
	for i := 0; i < 3; i++ {
		rows, err := s.tsi.GetRekap(now.AddDate(0, -i, 0).Format("2006-01"))
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			if r.GuruID == id {
				tsiRiwayat = append(tsiRiwayat, r)
				if tsiTerakhir == nil {
					rr := r
					tsiTerakhir = &rr
				}
			}
		}
	}
	semuaKelas, err := s.querier.AIKelasMetrik(ctx, hariIni, now.AddDate(0, 0, -30).Format("2006-01-02"), 0, false)
	if err != nil {
		return nil, err
	}
	kelas := make([]models.AIKelasMetrik, 0)
	for _, k := range semuaKelas {
		if k.GuruID == id {
			k.PersenHadir30 = aiRoundPtr(k.PersenHadir30)
			kelas = append(kelas, k)
		}
	}
	kunjungan, err := s.querier.AIListKunjunganGuru(ctx, id, 10)
	if err != nil {
		return nil, err
	}
	for i := range kunjungan {
		kunjungan[i].Catatan = aiTruncate(kunjungan[i].Catatan, 400)
		kunjungan[i].TanggapanGuru = aiTruncate(kunjungan[i].TanggapanGuru, 400)
	}
	catatan, err := s.querier.AICariCatatan(ctx, queries.AICatatanFilter{Sumber: []string{"tindak_lanjut", "monitoring_kelas", "pertemuan", "riayah"}, GuruID: id, Limit: 15})
	if err != nil {
		return nil, err
	}
	for i := range catatan {
		catatan[i].Isi = aiTruncate(catatan[i].Isi, 400)
	}
	return map[string]any{
		"data_per":           hariIni,
		"evaluasi":           s.evaluateGuru(p, metrik[0], mon, tsiTerakhir),
		"tsi_3_bulan":        tsiRiwayat,
		"kelas_aktif":        kelas,
		"kunjungan_terakhir": kunjungan,
		"catatan_terkait":    catatan,
	}, nil
}

// --- cari_catatan -------------------------------------------------------------------

func (s *AIAgentService) toolCariCatatan(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	allowed := aiCatatanPerRole[p.Role]
	if len(allowed) == 0 {
		return nil, ErrAIToolDitolak
	}
	f := queries.AICatatanFilter{Kata: strings.Fields(a.str("kata"))}
	if len(f.Kata) > 8 {
		return nil, aiArgErr("maksimal 8 kata kunci")
	}
	var err error
	if f.SantriID, err = a.int("santri_id", 0, 0, math.MaxInt32); err != nil {
		return nil, err
	}
	if f.KelasID, err = a.int("kelas_id", 0, 0, math.MaxInt32); err != nil {
		return nil, err
	}
	if f.GuruID, err = a.int("guru_id", 0, 0, math.MaxInt32); err != nil {
		return nil, err
	}
	if f.Limit, err = a.int("batas", 20, 1, 50); err != nil {
		return nil, err
	}
	if len(f.Kata) == 0 && f.SantriID == 0 && f.KelasID == 0 && f.GuruID == 0 {
		return nil, aiArgErr("isi kata, atau santri_id/kelas_id/guru_id")
	}
	if sejak := a.str("sejak"); sejak != "" {
		if _, err := time.Parse("2006-01-02", sejak); err != nil {
			return nil, aiArgErr("sejak harus format YYYY-MM-DD")
		}
		f.Sejak = sejak
	}
	allowedSet := map[string]bool{}
	for _, x := range allowed {
		allowedSet[x] = true
	}
	if raw := a.str("sumber"); raw != "" {
		for _, x := range strings.Split(raw, ",") {
			x = strings.TrimSpace(x)
			if x == "" {
				continue
			}
			if !allowedSet[x] {
				return nil, aiArgErr("sumber %q tidak tersedia untuk role Anda. Pilihan: %s", x, strings.Join(allowed, ", "))
			}
			f.Sumber = append(f.Sumber, x)
		}
	}
	if len(f.Sumber) == 0 {
		f.Sumber = allowed
	}
	rows, err := s.querier.AICariCatatan(ctx, f)
	if err != nil {
		return nil, err
	}
	hasil := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		r.Isi = aiTruncate(r.Isi, 600)
		item := map[string]any{"catatan": r}
		switch {
		case r.SantriID > 0:
			item["link"] = aiSantriLink(p, r.SantriID)
		case r.KelasID > 0:
			item["link"] = aiKelasLink(p, r.KelasID)
		case r.GuruID > 0 && aiHasRole(p.Role, aiRolesKoordinator):
			item["link"] = aiGuruLink(p, r.GuruID)
		}
		hasil = append(hasil, item)
	}
	return map[string]any{"sumber_dicari": f.Sumber, "jumlah": len(hasil), "hasil": hasil,
		"catatan": "Isi catatan ditulis staf dan bisa tidak lengkap; nomor HP/email di dalamnya disembunyikan."}, nil
}
