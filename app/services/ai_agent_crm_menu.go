package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// MCP tools for the rest of the CRM menu: Analisis CRM, Prioritas Lead,
// Follow-up, Sumber Lead, Performa Iklan, Meta CAPI, plus an ad → closing
// attribution view and the "safe" CS edits. Roles mirror routes/web.go:
// LeadRoles, FollowupApproverRoles, MetaAdsRoles and super_admin for CAPI.
// Contact data stays out: leads appear as lead_ref + first name.

var (
	aiRolesIklan       = []string{"cs", "super_admin"} // routes.MetaAdsRoles
	aiRolesCAPI        = []string{"super_admin"}
	aiRolesFollowupApv = FollowupApproverRoles
)

const (
	aiAdsListDefault  = 20
	aiAdsListMax      = 100
	aiCAPIEventsDef   = 20
	aiCAPIEventsMax   = 50
	aiLeadEventsLimit = 30
	aiDraftMaxRunes   = 1000
)

var errAIIklanTidakAda = aiArgError{"data Performa Iklan belum tersedia di server ini"}
var errAICAPITidakAda = aiArgError{"data Meta CAPI belum tersedia di server ini"}

// aiSafeFirstName is the first word of a display name, or "" when it looks
// like a phone number (leadDisplayName falls back to the phone).
func aiSafeFirstName(name string) string {
	f := strings.Fields(name)
	if len(f) == 0 {
		return ""
	}
	digits, other := 0, 0
	for _, r := range f[0] {
		switch {
		case unicode.IsDigit(r):
			digits++
		case r != '+' && r != '-' && r != '.' && r != '(' && r != ')':
			other++
		}
	}
	if digits >= 5 || (digits > 0 && other == 0) {
		return ""
	}
	return f[0]
}

// aiRatio returns a/b rounded to 2 decimals, or nil when b is 0.
func aiRatio(a, b float64) *float64 {
	if b == 0 {
		return nil
	}
	v := math.Round(a/b*100) / 100
	return &v
}

func aiPct(a, b float64) *float64 {
	if b == 0 {
		return nil
	}
	v := math.Round(a/b*10000) / 100
	return &v
}

// aiDateRange reads dari/sampai (YYYY-MM-DD, WIB). Both or neither must be
// set; maxDays bounds the inclusive length.
func (a aiArgs) dateRange(maxDays int) (string, string, error) {
	dari, sampai := a.str("dari"), a.str("sampai")
	if dari == "" && sampai == "" {
		return "", "", nil
	}
	if dari == "" || sampai == "" {
		return "", "", aiArgErr("isi dari dan sampai sekaligus (YYYY-MM-DD), atau kosongkan keduanya")
	}
	start, ok := wibDayStart(dari)
	if !ok {
		return "", "", aiArgErr("dari harus format YYYY-MM-DD")
	}
	end, ok := wibDayStart(sampai)
	if !ok {
		return "", "", aiArgErr("sampai harus format YYYY-MM-DD")
	}
	if start.After(end) || end.Sub(start) >= time.Duration(maxDays)*24*time.Hour {
		return "", "", aiArgErr("rentang tanggal harus 1 sampai %d hari dan dari ≤ sampai", maxDays)
	}
	return dari, sampai, nil
}

func aiRangeProps() map[string]any {
	return map[string]any{
		"dari":   aiProp("string", "Tanggal awal YYYY-MM-DD (WIB). Isi bersama sampai, atau kosongkan keduanya untuk default."),
		"sampai": aiProp("string", "Tanggal akhir YYYY-MM-DD (WIB)."),
	}
}

func aiMerge(a, b map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func (s *AIAgentService) crmMenuTools() []AITool {
	return []AITool{
		{
			Name:  "analisis_crm",
			Title: "Analisis CRM",
			Description: "Isi halaman Analisis CRM: lead masuk & closing per hari, banding periode sebelumnya, funnel, alasan hilang, keberatan, kinerja follow-up, " +
				"performa per sumber lead, antrean prioritas, dan insight otomatis. Default 7 hari terakhir.",
			InputSchema: aiSchema(aiRangeProps()),
			Roles:       aiRolesCRM,
			run:         s.toolAnalisisCRM,
		},
		{
			Name:  "prioritas_lead",
			Title: "Prioritas Lead",
			Description: "Antrean kerja CS lintas tanggal (halaman Prioritas Lead): lead dengan bukti bayar menunggu verifikasi, terverifikasi menunggu konversi, " +
				"data perlu ditinjau, atau pesan belum dibalas. 30 per halaman.",
			InputSchema: aiSchema(map[string]any{
				"alasan":  aiEnumProp("verify = bukti masuk, convert = perlu dikonversi, review = data perlu ditinjau, reply = belum dibalas. Kosong = semua.", "verify", "convert", "review", "reply"),
				"halaman": aiIntProp("Nomor halaman (default 1).", 1, 10000),
			}),
			Roles: aiRolesCRM,
			run:   s.toolPrioritasLead,
		},
		{
			Name:  "antrean_followup",
			Title: "Antrean follow-up",
			Description: "Isi halaman Follow-up: draft menunggu persetujuan, yang sudah disetujui/terjadwal, dan pengiriman gagal, plus pengaturan jam kirim & kuota harian. " +
				"BACA saja: menyetujui dan mengirim tetap dilakukan CS di dashboard.",
			InputSchema: aiSchema(map[string]any{}),
			Roles:       aiRolesFollowupApv,
			run:         s.toolAntreanFollowup,
		},
		{
			Name:        "daftar_sumber_lead",
			Title:       "Sumber lead",
			Description: "Katalog sumber lead (landing page / iklan): id, nama, URL, status aktif, teks pemicu pesan pertama, dan total lead. Pakai sumber_id untuk filter daftar_lead atau set_sumber_lead.",
			InputSchema: aiSchema(map[string]any{}),
			Roles:       aiRolesCRM,
			run:         s.toolDaftarSumberLead,
		},
		{
			Name:  "referensi_crm",
			Title: "Referensi CRM",
			Description: "Nilai yang valid untuk filter dan tool tulis CRM: tahap, status, alasan hilang, keberatan, label, sumber, dan daftar PIC (user_id staf CRM). " +
				"Panggil sebelum tugaskan_lead atau memfilter daftar_lead.",
			InputSchema: aiSchema(map[string]any{}),
			Roles:       aiRolesCRM,
			run:         s.toolReferensiCRM,
		},
		{
			Name:  "performa_iklan",
			Title: "Performa iklan Meta",
			Description: "Data Performa Iklan (Meta Ads) per kampanye, ad set, atau iklan: spend (Rupiah), impresi, klik, klik tautan, chat dimulai, CTR, CPC, CPM, biaya per chat, " +
				"banding periode sebelumnya, sumber lead yang dipetakan, dan status sinkron terakhir. Default 30 hari terakhir.",
			InputSchema: aiSchema(aiMerge(aiRangeProps(), map[string]any{
				"level":       aiEnumProp("Tingkat rincian (default kampanye).", "kampanye", "adset", "iklan"),
				"kampanye_id": aiProp("string", "Hanya baris dari kampanye ini (id dari hasil level kampanye)."),
				"limit":       aiIntProp("Jumlah baris teratas menurut spend (default 20).", 1, aiAdsListMax),
			})),
			Roles: aiRolesIklan,
			run:   s.toolPerformaIklan,
		},
		{
			Name:  "atribusi_iklan",
			Title: "Atribusi iklan ke closing",
			Description: "Gabungan iklan + CRM per sumber lead: spend kampanye yang dipetakan ke sumber, chat, lead masuk, jadi santri (won), biaya per lead (CPL), " +
				"biaya per santri, dan konversi, banding periode sebelumnya. Juga kampanye ber-spend yang belum dipetakan dan ringkasan event Meta CAPI (super_admin). " +
				"Lead dihitung per tanggal masuk (cohort); won = lead cohort itu yang sekarang sudah jadi santri.",
			InputSchema: aiSchema(aiRangeProps()),
			Roles:       aiRolesIklan,
			run:         s.toolAtribusiIklan,
		},
		{
			Name:  "status_meta_capi",
			Title: "Status Meta CAPI",
			Description: "Isi halaman Meta CAPI: konfigurasi (dataset tersamar, mode uji), antrean event (pending/sent/failed/skipped), pemetaan tahap CRM → event Meta, " +
				"sumber tanpa URL landing page, analitik event per hari & per jenis (maks 90 hari, default 14), dan event terbaru beserta error. " +
				"Ini event yang DIKIRIM CRM, bukan konversi yang diatribusikan Meta.",
			InputSchema: aiSchema(aiMerge(aiRangeProps(), map[string]any{
				"max_event": aiIntProp("Jumlah event terbaru (default 20).", 0, aiCAPIEventsMax),
				"status":    aiEnumProp("Saring event menurut status (hanya di antara 50 event terbaru; jumlah total per status ada di antrean_seluruhnya).", "pending", "sent", "failed", "skipped"),
			})),
			Roles: aiRolesCAPI,
			run:   s.toolStatusMetaCAPI,
		},
		{
			Name:  "ubah_tahap_lead",
			Title: "Ubah tahap lead",
			Description: "TULIS. Pindahkan lead open ke tahap lain (maju atau mundur), sama seperti tombol tahap di halaman lead. Tidak bisa ke/dari terverifikasi. " +
				"Wajib alasan dari bukti chat. Perhatian: tahap data_terisi, info_bayar_dikirim, dan bukti_masuk memicu event konversi Meta CAPI, jadi pindahkan hanya bila faktanya jelas.",
			InputSchema: aiSchema(map[string]any{
				"lead_ref": aiProp("string", "lead_ref, mis. L-12."),
				"tahap":    aiEnumProp("Tahap tujuan.", aiStagesWithout(LeadStageTerverifikasi)...),
				"alasan":   aiProp("string", "Bukti singkat 10–300 karakter, mis. \"Lead sudah mengirim form lengkap pada 2 Nov\"."),
			}, "lead_ref", "tahap", "alasan"),
			Roles: aiRolesCRM,
			Write: true,
			run:   s.toolUbahTahapLead,
		},
		{
			Name:  "tugaskan_lead",
			Title: "Tugaskan PIC lead",
			Description: "TULIS. Set PIC (penanggung jawab) lead. Token milik non-super_admin hanya bisa menugaskan ke pemilik token sendiri. " +
				"user_id dari referensi_crm; 0 = lepas PIC (super_admin saja).",
			InputSchema: aiSchema(map[string]any{
				"lead_ref": aiProp("string", "lead_ref, mis. L-12."),
				"user_id":  aiIntProp("user_id PIC dari referensi_crm.", 0, math.MaxInt64),
			}, "lead_ref", "user_id"),
			Roles: aiRolesCRM,
			Write: true,
			run:   s.toolTugaskanLead,
		},
		{
			Name:        "set_sumber_lead",
			Title:       "Set sumber lead",
			Description: "TULIS. Set sumber lead secara manual (mis. lead menyebut iklan tertentu di chat). sumber_id dari daftar_sumber_lead; 0 = tanpa sumber. Wajib alasan.",
			InputSchema: aiSchema(map[string]any{
				"lead_ref":  aiProp("string", "lead_ref, mis. L-12."),
				"sumber_id": aiIntProp("id sumber; 0 = tanpa sumber.", 0, math.MaxInt64),
				"alasan":    aiProp("string", "Bukti singkat 10–300 karakter."),
			}, "lead_ref", "sumber_id", "alasan"),
			Roles: aiRolesCRM,
			Write: true,
			run:   s.toolSetSumberLead,
		},
		{
			Name:  "petakan_kampanye_iklan",
			Title: "Petakan kampanye ke sumber",
			Description: "TULIS. Hubungkan kampanye Meta ke sumber lead agar spend-nya dihitung di atribusi_iklan (sama seperti pilihan sumber di Performa Iklan). " +
				"sumber_id 0 = lepas pemetaan.",
			InputSchema: aiSchema(map[string]any{
				"kampanye_id": aiProp("string", "id kampanye dari performa_iklan."),
				"sumber_id":   aiIntProp("id sumber dari daftar_sumber_lead; 0 = lepas.", 0, math.MaxInt64),
			}, "kampanye_id", "sumber_id"),
			Roles: aiRolesIklan,
			Write: true,
			run:   s.toolPetakanKampanye,
		},
	}
}

func aiStagesWithout(skip string) []string {
	out := []string{}
	for _, st := range LeadStages {
		if st != skip {
			out = append(out, st)
		}
	}
	return out
}

func (s *AIAgentService) priorityItems(p *models.AIPrincipal, items []models.LeadDashboardPriority) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		aktivitas := it.ActivityAt
		if t, err := time.Parse(time.RFC3339, it.ActivityAt); err == nil {
			aktivitas = aiWIB(t)
		}
		out = append(out, map[string]any{
			"lead_ref": aiLeadRef(it.ID), "nama_depan": aiSafeFirstName(it.Name), "tahap": it.Stage,
			"alasan": it.Reason, "aktivitas_terakhir": aktivitas, "link": s.leadLink(p, it.ID),
		})
	}
	return out
}

func (s *AIAgentService) toolAnalisisCRM(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	dari, sampai, err := a.dateRange(366)
	if err != nil {
		return nil, err
	}
	d, err := s.leads.Dashboard(ctx, dari, sampai)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"data_per": aiWIB(s.now()), "dari": d.From, "sampai": d.To,
		"lead_masuk": d.Report.TotalLeads, "lead_periode_sebelumnya": d.PreviousLeads,
		"closing": d.ClosingPeriod, "closing_periode_sebelumnya": d.PreviousClosing,
		"perlu_ditinjau": d.ReviewCount,
		"follow_up": map[string]any{
			"menunggu_persetujuan": d.FollowupPending, "disetujui_terjadwal": d.FollowupApproved, "gagal_kirim": d.FollowupFailed,
			"matang_72_jam": d.FollowupEligible, "dibalas": d.FollowupReplied, "persen_dibalas": aiPct(float64(d.FollowupReplied), float64(d.FollowupEligible)),
		},
		"harian":            d.Daily,
		"per_sumber":        d.Sources,
		"laporan_funnel":    d.Report,
		"prioritas_teratas": s.priorityItems(p, d.Priorities),
		"insight":           d.Insights,
		"catatan":           "lead_masuk & funnel = cohort lead yang masuk di periode; closing = lead yang dikonversi pada periode (tanggal kejadian).",
		"link":              aiLink(p, "/app/leads/dashboard?from="+d.From+"&to="+d.To),
	}, nil
}

func (s *AIAgentService) toolPrioritasLead(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	alasan, err := a.enum("alasan", "", "verify", "convert", "review", "reply")
	if err != nil {
		return nil, err
	}
	page, err := a.int("halaman", 1, 1, 10000)
	if err != nil {
		return nil, err
	}
	res, err := s.leads.PriorityPage(ctx, "", alasan, int(page))
	if err != nil {
		return nil, err
	}
	pages := (res.Total + int64(res.PerPage) - 1) / int64(res.PerPage)
	return map[string]any{
		"data_per": aiWIB(s.now()), "total": res.Total, "halaman": res.Page, "jumlah_halaman": pages, "per_halaman": res.PerPage,
		"lead": s.priorityItems(p, res.Items), "link": aiLink(p, "/app/leads/prioritas"),
	}, nil
}

func (s *AIAgentService) toolAntreanFollowup(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	q, err := s.leads.FollowupQueue(ctx, p.Role)
	if err != nil {
		return nil, err
	}
	// Each lead's domisili, so drafts get the same masking as detail_lead.
	domisili := map[int64]string{}
	for _, it := range append(append([]models.FollowupQueueItem{}, q.Pending...), q.Approved...) {
		if _, ok := domisili[it.LeadID]; ok {
			continue
		}
		l, err := s.querier.GetLeadByID(ctx, it.LeadID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		domisili[it.LeadID] = l.Domisili
	}
	item := func(it models.FollowupQueueItem) map[string]any {
		dom := domisili[it.LeadID]
		return map[string]any{
			"followup_id": it.ID, "lead_ref": aiLeadRef(it.LeadID), "nama_depan": aiSafeFirstName(it.LeadName), "tahap": it.LeadStage,
			"step": it.Step, "jadwal": it.ScheduledAt, "status": it.Status, "status_kirim": it.OutboxStatus,
			"draft": RedactLeadText(aiTruncate(it.DraftText, aiDraftMaxRunes), dom), "alasan": RedactLeadText(it.Reason, dom),
			"dibuat_oleh": it.CreatedByKind, "peringatan": it.Warnings, "diblokir": it.Blocked, "error": RedactContactText(it.Error),
			"link": s.leadLink(p, it.LeadID),
		}
	}
	pending := make([]map[string]any, 0, len(q.Pending))
	for _, it := range q.Pending {
		pending = append(pending, item(it))
	}
	approved := make([]map[string]any, 0, len(q.Approved))
	for _, it := range q.Approved {
		approved = append(approved, item(it))
	}
	failed := make([]map[string]any, 0, len(q.Failed))
	for _, f := range q.Failed {
		failed = append(failed, map[string]any{
			"outbox_id": f.OutboxID, "lead_ref": aiLeadRef(f.LeadID), "nama_depan": aiSafeFirstName(f.LeadName), "tujuan": f.Purpose,
			"step": f.Step, "error": RedactContactText(f.Error), "percobaan": f.Attempts, "diperbarui": f.UpdatedAt,
			"link": s.leadLink(p, f.LeadID),
		})
	}
	return map[string]any{
		"data_per": aiWIB(s.now()),
		"pengaturan": map[string]any{
			"followup_aktif": q.FollowupEnabled, "jam_kirim": q.SendStart + "–" + q.SendEnd + " WIB",
			"kuota_harian": q.DailyCap, "terkirim_hari_ini": q.SentToday, "hanya_label": q.OnlyLabel,
		},
		"menunggu_persetujuan": pending, "disetujui": approved, "gagal_kirim": failed,
		"catatan": "AI tidak bisa menyetujui atau mengirim; arahkan CS ke link halaman Follow-up.",
		"link":    aiLink(p, "/app/leads/followups"),
	}, nil
}

func (s *AIAgentService) toolDaftarSumberLead(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	srcs, err := s.leads.Sources(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(srcs))
	for _, src := range srcs {
		pemicu := make([]string, 0, len(src.Patterns))
		for _, pt := range src.Patterns {
			pemicu = append(pemicu, pt.Teks)
		}
		out = append(out, map[string]any{
			"sumber_id": src.ID, "nama": src.Nama, "keterangan": src.Keterangan, "url": src.URL,
			"aktif": src.IsAktif, "teks_pemicu": pemicu, "total_lead": src.TotalLeads,
		})
	}
	return map[string]any{"data_per": aiWIB(s.now()), "jumlah": len(out), "sumber": out, "link": aiLink(p, "/app/leads/sumber")}, nil
}

func (s *AIAgentService) toolReferensiCRM(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	labels, err := s.querier.ListLeadLabels(ctx)
	if err != nil {
		return nil, err
	}
	labelOut := make([]map[string]any, 0, len(labels))
	for _, lb := range labels {
		labelOut = append(labelOut, map[string]any{"label_id": lb.ID, "nama": lb.Nama})
	}
	srcs, err := s.leads.SourceOptions(ctx)
	if err != nil {
		return nil, err
	}
	srcOut := make([]map[string]any, 0, len(srcs))
	for _, src := range srcs {
		srcOut = append(srcOut, map[string]any{"sumber_id": src.ID, "nama": src.Nama, "aktif": src.IsAktif})
	}
	users, err := s.leads.AssignableUsers(ctx)
	if err != nil {
		return nil, err
	}
	pic := make([]map[string]any, 0, len(users))
	for _, u := range users {
		pic = append(pic, map[string]any{"user_id": u.ID, "nama": u.Name, "role": u.Role})
	}
	return map[string]any{
		"tahap": LeadStages, "status": []string{LeadStatusOpen, LeadStatusWon, LeadStatusLost, LeadStatusDoNotContact},
		"alasan_hilang": LeadLostReasons, "keberatan": LeadObjections, "label": labelOut, "sumber": srcOut, "pic": pic,
		"anda": map[string]any{"user_id": p.UserID, "role": p.Role, "bisa_tulis": p.CanWrite},
	}, nil
}

// --- Performa Iklan ----------------------------------------------------------

type aiAdMetrics struct {
	Spend, Impressions, Clicks, LinkClicks, Chats int64
}

func (m *aiAdMetrics) add(r models.MetaAdsRow) {
	m.Spend += r.Spend
	m.Impressions += r.Impressions
	m.Clicks += r.Clicks
	m.LinkClicks += r.LinkClicks
	m.Chats += r.Messaging
}

func (m aiAdMetrics) out() map[string]any {
	return map[string]any{
		"spend": m.Spend, "impresi": m.Impressions, "klik": m.Clicks, "klik_tautan": m.LinkClicks, "chat_dimulai": m.Chats,
		"ctr_persen": aiPct(float64(m.LinkClicks), float64(m.Impressions)), "ctr_semua_klik_persen": aiPct(float64(m.Clicks), float64(m.Impressions)), "cpc": aiRatio(float64(m.Spend), float64(m.LinkClicks)),
		"cpm": aiRatio(float64(m.Spend)*1000, float64(m.Impressions)), "biaya_per_chat": aiRatio(float64(m.Spend), float64(m.Chats)),
	}
}

func (s *AIAgentService) adsReport(ctx context.Context, a aiArgs) (*models.MetaAdsReport, error) {
	if s.metaAds == nil {
		return nil, errAIIklanTidakAda
	}
	dari, sampai, err := a.dateRange(366)
	if err != nil {
		return nil, err
	}
	return s.metaAds.Report(ctx, dari, sampai)
}

func aiSyncStatus(r *models.MetaAdsReport) map[string]any {
	out := map[string]any{"integrasi_aktif": r.Enabled, "akun": r.Accounts}
	if r.LastSuccess != nil {
		out["sinkron_sukses_terakhir"] = r.LastSuccess.FinishedAt
		out["data_sampai"] = r.LastSuccess.DateTo
	}
	if r.LastRun != nil {
		out["sinkron_terakhir_status"] = r.LastRun.Status
		if r.LastRun.Error != "" {
			out["sinkron_terakhir_error"] = aiTruncate(r.LastRun.Error, 300)
		}
	}
	return out
}

func (s *AIAgentService) toolPerformaIklan(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	level, err := a.enum("level", "kampanye", "kampanye", "adset", "iklan")
	if err != nil {
		return nil, err
	}
	limit, err := a.int("limit", aiAdsListDefault, 1, aiAdsListMax)
	if err != nil {
		return nil, err
	}
	onlyCampaign := a.str("kampanye_id")
	r, err := s.adsReport(ctx, a)
	if err != nil {
		return nil, err
	}
	campName, campSource := map[string]string{}, map[string]int64{}
	for _, c := range r.Campaigns {
		campName[c.ID], campSource[c.ID] = c.Name, c.SourceID
	}
	srcName := map[int64]string{}
	for _, src := range r.Sources {
		srcName[src.ID] = src.Nama
	}
	adInfo, adsetName := map[string]models.MetaAdsAdInfo{}, map[string]string{}
	for _, ad := range r.Ads {
		adInfo[ad.ID] = ad
		adsetName[ad.AdsetID] = ad.AdsetName
	}

	type row struct {
		id, campaign, adset string
		cur, prev           aiAdMetrics
	}
	rows := map[string]*row{}
	var total, prevTotal aiAdMetrics
	for _, x := range r.Rows {
		if onlyCampaign != "" && x.CampaignID != onlyCampaign {
			continue
		}
		cur := x.Day >= r.From && x.Day <= r.To
		if cur {
			total.add(x)
		} else {
			prevTotal.add(x)
		}
		key := x.CampaignID
		switch level {
		case "adset":
			key = x.AdsetID
		case "iklan":
			key = x.AdID
		}
		rw := rows[key]
		if rw == nil {
			rw = &row{id: key, campaign: x.CampaignID, adset: x.AdsetID}
			rows[key] = rw
		}
		if cur {
			rw.cur.add(x)
		} else {
			rw.prev.add(x)
		}
	}
	list := make([]*row, 0, len(rows))
	for _, rw := range rows {
		list = append(list, rw)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].cur.Spend != list[j].cur.Spend {
			return list[i].cur.Spend > list[j].cur.Spend
		}
		return list[i].id < list[j].id
	})
	truncated := int64(len(list)) > limit
	if truncated {
		list = list[:limit]
	}
	items := make([]map[string]any, 0, len(list))
	for _, rw := range list {
		m := map[string]any{"id": rw.id, "periode_ini": rw.cur.out(), "periode_sebelumnya": rw.prev.out()}
		switch level {
		case "kampanye":
			m["nama"] = campName[rw.id]
			if sid := campSource[rw.id]; sid > 0 {
				m["sumber_id"], m["sumber"] = sid, srcName[sid]
			} else {
				m["sumber"] = "belum dipetakan"
			}
		case "adset":
			m["nama"], m["kampanye_id"], m["kampanye"] = adsetName[rw.id], rw.campaign, campName[rw.campaign]
		case "iklan":
			info := adInfo[rw.id]
			m["nama"], m["adset"], m["kampanye_id"], m["kampanye"] = info.Name, info.AdsetName, rw.campaign, campName[rw.campaign]
		}
		items = append(items, m)
	}
	link := "/app/leads/iklan?from=" + r.From + "&to=" + r.To
	return map[string]any{
		"data_per": aiWIB(s.now()), "dari": r.From, "sampai": r.To, "periode_sebelumnya": r.PrevFrom + " s.d. " + r.PrevTo,
		"level": level, "sinkron": aiSyncStatus(r),
		"total": total.out(), "total_periode_sebelumnya": prevTotal.out(),
		"jumlah_baris": len(rows), "baris": items, "terpotong": truncated,
		"catatan": "spend dalam Rupiah. chat_dimulai = messaging conversations started (Meta), bukan lead CRM. Data kosong bisa berarti belum disinkron.",
		"link":    aiLink(p, link),
	}, nil
}

// --- Atribusi iklan → closing ------------------------------------------------

// paidRatio is spend/n for a source with mapped campaigns; nil for organic
// sources, whose leads cost nothing we can measure.
func paidRatio(st *aiSourceAttr, spend, n int64) *float64 {
	if st.campaigns == 0 {
		return nil
	}
	return aiRatio(float64(spend), float64(n))
}

type aiSourceAttr struct {
	cur, prev        aiAdMetrics
	leads, won       int64
	prevLeads, prevW int64
	campaigns        int
}

func (s *AIAgentService) toolAtribusiIklan(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	r, err := s.adsReport(ctx, a)
	if err != nil {
		return nil, err
	}
	campSource, campName := map[string]int64{}, map[string]string{}
	for _, c := range r.Campaigns {
		campSource[c.ID], campName[c.ID] = c.SourceID, c.Name
	}
	srcName := map[int64]string{0: "Tanpa sumber"}
	for _, src := range r.Sources {
		srcName[src.ID] = src.Nama
	}
	stats := map[int64]*aiSourceAttr{}
	get := func(id int64) *aiSourceAttr {
		st := stats[id]
		if st == nil {
			st = &aiSourceAttr{}
			stats[id] = st
		}
		return st
	}
	for _, c := range r.Campaigns {
		if c.SourceID > 0 {
			get(c.SourceID).campaigns++
		}
	}
	unmapped := map[string]*aiAdMetrics{}
	var unmappedSpend int64
	for _, x := range r.Rows {
		cur := x.Day >= r.From && x.Day <= r.To
		sid := campSource[x.CampaignID]
		if sid == 0 {
			if cur {
				m := unmapped[x.CampaignID]
				if m == nil {
					m = &aiAdMetrics{}
					unmapped[x.CampaignID] = m
				}
				m.add(x)
				unmappedSpend += x.Spend
			}
			continue
		}
		st := get(sid)
		if cur {
			st.cur.add(x)
		} else {
			st.prev.add(x)
		}
	}
	for _, l := range r.LeadDays {
		st := get(l.SourceID)
		if l.Day >= r.From && l.Day <= r.To {
			st.leads += l.Leads
			st.won += l.Won
		} else {
			st.prevLeads += l.Leads
			st.prevW += l.Won
		}
	}
	ids := make([]int64, 0, len(stats))
	for id := range stats {
		ids = append(ids, id)
	}
	sourceName := func(id int64) string {
		if n, ok := srcName[id]; ok {
			return n
		}
		return fmt.Sprintf("Sumber %d", id)
	}
	// Spend, then leads, then name — the Performa Iklan order.
	sort.Slice(ids, func(i, j int) bool {
		a, b := stats[ids[i]], stats[ids[j]]
		if a.cur.Spend != b.cur.Spend {
			return a.cur.Spend > b.cur.Spend
		}
		if a.leads != b.leads {
			return a.leads > b.leads
		}
		return sourceName(ids[i]) < sourceName(ids[j])
	})
	var totSpend, totLeads, totWon, paidLeads, paidWon int64
	rows := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		st := stats[id]
		totSpend += st.cur.Spend
		totLeads += st.leads
		totWon += st.won
		if st.campaigns > 0 {
			paidLeads += st.leads
			paidWon += st.won
		}
		rows = append(rows, map[string]any{
			"sumber_id": id, "sumber": sourceName(id), "kampanye_terpetakan": st.campaigns, "berbayar": st.campaigns > 0,
			"spend": st.cur.Spend, "chat_dimulai": st.cur.Chats, "lead": st.leads, "jadi_santri": st.won,
			"cpl": paidRatio(st, st.cur.Spend, st.leads), "biaya_per_santri": paidRatio(st, st.cur.Spend, st.won),
			"konversi_persen": aiPct(float64(st.won), float64(st.leads)),
			"periode_sebelumnya": map[string]any{
				"spend": st.prev.Spend, "lead": st.prevLeads, "jadi_santri": st.prevW,
				"cpl": paidRatio(st, st.prev.Spend, st.prevLeads), "biaya_per_santri": paidRatio(st, st.prev.Spend, st.prevW),
			},
		})
	}
	unList := make([]map[string]any, 0, len(unmapped))
	for id, m := range unmapped {
		if m.Spend == 0 {
			continue
		}
		unList = append(unList, map[string]any{"kampanye_id": id, "nama": campName[id], "spend": m.Spend, "chat_dimulai": m.Chats})
	}
	sort.Slice(unList, func(i, j int) bool { return unList[i]["spend"].(int64) > unList[j]["spend"].(int64) })

	out := map[string]any{
		"data_per": aiWIB(s.now()), "dari": r.From, "sampai": r.To, "periode_sebelumnya": r.PrevFrom + " s.d. " + r.PrevTo,
		"sinkron": aiSyncStatus(r),
		"ringkasan": map[string]any{
			"spend_terpetakan": totSpend, "spend_belum_dipetakan": unmappedSpend,
			"lead_total": totLeads, "jadi_santri_total": totWon,
			"lead_dari_sumber_berbayar": paidLeads, "jadi_santri_dari_sumber_berbayar": paidWon,
			"cpl_berbayar": aiRatio(float64(totSpend), float64(paidLeads)), "biaya_per_santri_berbayar": aiRatio(float64(totSpend), float64(paidWon)),
		},
		"per_sumber":               rows,
		"kampanye_belum_dipetakan": unList,
		"catatan":                  "Atribusi lewat pemetaan kampanye → sumber lead dan teks pemicu pesan pertama; lead tanpa sumber tidak bisa dikaitkan ke iklan. Spend kampanye belum dipetakan tidak masuk CPL. Petakan lewat petakan_kampanye_iklan.",
		"link":                     aiLink(p, "/app/leads/iklan?from="+r.From+"&to="+r.To),
	}
	if s.metaCAPI != nil && aiHasRole(p.Role, aiRolesCAPI) {
		if capi, err := s.capiAnalytics(ctx, r.From, r.To); err == nil {
			out["meta_capi"] = capi
		} else {
			out["meta_capi"] = map[string]any{"catatan": err.Error()}
		}
	}
	return out, nil
}

// capiAnalytics summarises CAPI events for a range; ranges over 90 days use
// the last 90 days of it.
func (s *AIAgentService) capiAnalytics(ctx context.Context, from, to string) (map[string]any, error) {
	if start, ok := wibDayStart(from); ok {
		if end, ok := wibDayStart(to); ok && end.Sub(start) >= 90*24*time.Hour {
			from = end.AddDate(0, 0, -89).Format(time.DateOnly)
		}
	}
	an, err := s.metaCAPI.Analytics(ctx, from, to)
	if err != nil {
		return nil, aiArgErr("analitik CAPI tidak tersedia untuk rentang ini (maks 90 hari)")
	}
	return map[string]any{
		"dari": an.From, "sampai": an.To, "total": an.Total, "terkirim_produksi": an.Sent, "terkirim_uji": an.TestSent,
		"pending": an.Pending, "gagal": an.Failed, "dilewati": an.Skipped, "nilai_purchase": an.PurchaseValue, "per_event": an.Events,
	}, nil
}

func (s *AIAgentService) toolStatusMetaCAPI(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	if s.metaCAPI == nil {
		return nil, errAICAPITidakAda
	}
	dari, sampai, err := a.dateRange(90)
	if err != nil {
		return nil, err
	}
	maxEv, err := a.int("max_event", aiCAPIEventsDef, 0, aiCAPIEventsMax)
	if err != nil {
		return nil, err
	}
	status, err := a.enum("status", "", "pending", "sent", "failed", "skipped")
	if err != nil {
		return nil, err
	}
	st, err := s.metaCAPI.Status(ctx)
	if err != nil {
		return nil, err
	}
	an, err := s.metaCAPI.Analytics(ctx, dari, sampai)
	if err != nil {
		return nil, err
	}
	events := make([]map[string]any, 0, maxEv)
	for _, e := range st.Events {
		if int64(len(events)) >= maxEv {
			break
		}
		if status != "" && e.Status != status {
			continue
		}
		events = append(events, map[string]any{
			"lead_ref": aiLeadRef(e.LeadID), "nama_depan": aiSafeFirstName(e.LeadNama), "event": e.Event, "status": e.Status,
			"waktu_event": e.EventTime, "terkirim": e.SentAt, "nilai": e.Value, "mode_uji": e.TestMode,
			"percobaan": e.Attempts, "error": aiTruncate(RedactContactText(e.LastError), 300),
		})
	}
	return map[string]any{
		"data_per": aiWIB(s.now()),
		"konfigurasi": map[string]any{
			"aktif": st.Enabled, "dataset": st.DatasetID, "versi_api": st.APIVersion, "mode_uji": st.TestMode,
			"app_secret_proof": st.AppSecretProof, "url_sumber_default": st.EventSourceURL,
		},
		"antrean_seluruhnya":     st.Counts,
		"event_uji_bisa_dikirim": st.ResendableTest,
		"sumber_tanpa_url":       st.SourcesWithoutURL,
		"pemetaan_tahap":         st.Mapping,
		"analitik": map[string]any{
			"dari": an.From, "sampai": an.To, "total": an.Total, "terkirim_produksi": an.Sent, "terkirim_uji": an.TestSent,
			"pending": an.Pending, "gagal": an.Failed, "dilewati": an.Skipped, "nilai_purchase": an.PurchaseValue,
			"harian": an.Days, "per_event": an.Events,
		},
		"event_terbaru": events,
		"catatan":       "Retry dan kirim ulang event uji hanya lewat tombol di dashboard (super_admin).",
		"link":          aiLink(p, "/app/leads/meta-capi"),
	}, nil
}

// --- write tools -------------------------------------------------------------

func (s *AIAgentService) toolUbahTahapLead(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.leadRef("lead_ref")
	if err != nil {
		return nil, err
	}
	tahap, err := a.enum("tahap", "", aiStagesWithout(LeadStageTerverifikasi)...)
	if err != nil {
		return nil, err
	}
	if tahap == "" {
		return nil, aiArgErr("tahap wajib diisi")
	}
	from, changed, err := s.leads.ChangeStageByAI(ctx, id, p.TokenID, tahap, a.str("alasan"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "lead_ref": aiLeadRef(id), "tahap_sebelumnya": from, "tahap": tahap, "berubah": changed, "link": s.leadLink(p, id)}, nil
}

func (s *AIAgentService) toolTugaskanLead(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.leadRef("lead_ref")
	if err != nil {
		return nil, err
	}
	if _, ok := a.m["user_id"]; !ok {
		return nil, aiArgErr("user_id wajib diisi (lihat pic di referensi_crm)")
	}
	userID, err := a.int("user_id", 0, 0, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	changed, err := s.leads.AssignByAI(ctx, id, p.TokenID, p.UserID, p.Role, userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "lead_ref": aiLeadRef(id), "pic_user_id": userID, "berubah": changed, "link": s.leadLink(p, id)}, nil
}

func (s *AIAgentService) toolSetSumberLead(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	id, err := a.leadRef("lead_ref")
	if err != nil {
		return nil, err
	}
	if _, ok := a.m["sumber_id"]; !ok {
		return nil, aiArgErr("sumber_id wajib diisi (lihat daftar_sumber_lead; 0 = tanpa sumber)")
	}
	sid, err := a.int("sumber_id", 0, 0, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	changed, err := s.leads.SetLeadSourceByAI(ctx, id, p.TokenID, sid, a.str("alasan"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "lead_ref": aiLeadRef(id), "sumber_id": sid, "berubah": changed, "link": s.leadLink(p, id)}, nil
}

func (s *AIAgentService) toolPetakanKampanye(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	if s.metaAds == nil {
		return nil, errAIIklanTidakAda
	}
	cid := a.str("kampanye_id")
	if cid == "" || len(cid) > 64 {
		return nil, aiArgErr("kampanye_id wajib diisi (id dari performa_iklan)")
	}
	if _, ok := a.m["sumber_id"]; !ok {
		return nil, aiArgErr("sumber_id wajib diisi (0 = lepas pemetaan)")
	}
	sid, err := a.int("sumber_id", 0, 0, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	known := false
	camps, err := s.querier.ListMetaCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range camps {
		if c.CampaignID == cid {
			known = true
			break
		}
	}
	if !known {
		return nil, aiArgErr("kampanye %q belum pernah tersinkron; pakai id dari performa_iklan", cid)
	}
	if err := s.metaAds.SetCampaignSource(ctx, cid, sid); err != nil {
		if errors.Is(err, ErrLeadSourceNotFound) {
			return nil, aiArgErr("sumber %d tidak ditemukan; lihat daftar_sumber_lead", sid)
		}
		return nil, err
	}
	return map[string]any{"ok": true, "kampanye_id": cid, "sumber_id": sid, "link": aiLink(p, "/app/leads/iklan")}, nil
}

// --- detail_lead enrichment --------------------------------------------------

// aiLeadEventPayloadKeys are the event payload fields shown to the AI.
var aiLeadEventPayloadKeys = []string{"stage_from", "stage_to", "status_from", "status_to", "reason", "label", "step", "fields", "alasan", "lead_source_from", "lead_source_to", "from_user_id", "to_user_id"}

func aiActorKind(actor string) string {
	switch {
	case strings.HasPrefix(actor, "user:"):
		return "staf"
	case strings.HasPrefix(actor, "ai:"):
		return "ai"
	case actor == "":
		return "sistem"
	}
	return actor
}

// leadExtra returns the CRM fields detail_lead adds beyond the form data:
// source, PIC, payment milestones and the recent event timeline.
func (s *AIAgentService) leadExtra(ctx context.Context, id int64, domisili string, sourceID sql.NullInt64, sourceBy string, assigned sql.NullInt64) (map[string]any, error) {
	out := map[string]any{"sumber_id": sourceID.Int64, "sumber": "", "sumber_oleh": sourceBy, "pic_user_id": assigned.Int64, "pic": ""}
	if sourceID.Valid && sourceID.Int64 > 0 {
		if src, err := s.querier.GetLeadSource(ctx, sourceID.Int64); err == nil {
			out["sumber"] = src.Nama
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	if assigned.Valid && assigned.Int64 > 0 {
		if u, err := s.querier.GetUserByID(ctx, assigned.Int64); err == nil {
			out["pic"] = u.Name
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	evs, err := s.querier.ListLeadEvents(ctx, queries.ListLeadEventsParams{LeadID: id, Limit: aiLeadEventsLimit})
	if err != nil {
		return nil, err
	}
	timeline := make([]map[string]any, 0, len(evs))
	for _, e := range evs {
		row := map[string]any{"jenis": e.Type, "oleh": aiActorKind(e.Actor), "waktu": aiWIB(e.CreatedAt)}
		payload := map[string]any{}
		_ = json.Unmarshal([]byte(e.Payload), &payload)
		for _, k := range aiLeadEventPayloadKeys {
			v, ok := payload[k]
			if !ok || (k == "reason" && e.Type != "lead.lost") {
				// Elsewhere "reason" is free text from the WA bridge (handoff,
				// ai.paused, outbox errors) that may name people or places.
				continue
			}
			if str, isStr := v.(string); isStr {
				v = RedactLeadText(str, domisili)
			}
			row[k] = v
		}
		timeline = append(timeline, row)
	}
	out["riwayat_terbaru"] = timeline
	return out, nil
}
