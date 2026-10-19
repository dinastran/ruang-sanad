package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
)

// List tools: return every matching row (with names) page by page, so the AI
// can answer "who exactly?" after a summary gives only a count.

const aiDaftarMaksPerHalaman = 200

func aiPaging(a aiArgs, def int64) (limit, offset, page int64, err error) {
	if limit, err = a.int("per_halaman", def, 1, aiDaftarMaksPerHalaman); err != nil {
		return
	}
	if page, err = a.int("halaman", 1, 1, math.MaxInt32); err != nil {
		return
	}
	return limit, (page - 1) * limit, page, nil
}

func aiPageInfo(total, page, limit int64, shown int) map[string]any {
	pages := (total + limit - 1) / limit
	return map[string]any{"total": total, "halaman": page, "per_halaman": limit, "jumlah_halaman": pages,
		"ditampilkan": shown, "ada_halaman_berikut": page < pages}
}

func aiDateArg(a aiArgs, key string) (string, error) {
	v := a.str(key)
	if v == "" {
		return "", nil
	}
	if _, err := time.Parse("2006-01-02", v); err != nil {
		return "", aiArgErr("%s harus format YYYY-MM-DD", key)
	}
	return v, nil
}

var aiPagingProps = map[string]any{
	"halaman":     aiIntProp("Nomor halaman, mulai 1.", 1, math.MaxInt32),
	"per_halaman": aiIntProp("Baris per halaman (default 100, maks 200).", 1, aiDaftarMaksPerHalaman),
}

func aiWithPaging(props map[string]any) map[string]any {
	for k, v := range aiPagingProps {
		props[k] = v
	}
	return props
}

func (s *AIAgentService) daftarTools() []AITool {
	return []AITool{
		{
			Name:  "daftar_santri",
			Title: "Daftar santri",
			Description: "Daftar lengkap santri (nama, ID, status, kelas, guru, level, tipe, angkatan, tanggal daftar) dengan filter bebas. " +
				"Pakai ini untuk menjawab \"siapa saja\" di balik angka ringkasan, mis. tanpa_kelas_aktif=true (santri aktif belum punya kelas aktif) " +
				"atau belum_lengkap=true (data belum lengkap). Hasil berhalaman; ikuti ada_halaman_berikut.",
			InputSchema: aiSchema(aiWithPaging(map[string]any{
				"status":            aiEnumProp("Status santri. Kosong = semua status.", "aktif", "cuti", "nonaktif", "tidak_lanjut"),
				"tanpa_kelas_aktif": aiProp("boolean", "true = hanya santri yang tidak punya kelas atau kelasnya nonaktif (definisi dashboard: pakai bersama status=aktif)."),
				"belum_lengkap":     aiProp("boolean", "true = hanya santri yang datanya belum lengkap (definisi dashboard: pakai bersama status=aktif)."),
				"kelas_id":          aiIntProp("Hanya santri di kelas ini.", 1, math.MaxInt32),
				"guru_id":           aiIntProp("Hanya santri di kelas guru ini.", 1, math.MaxInt32),
				"level":             aiProp("string", "Level santri persis, mis. \"Dasar\"."),
				"tipe":              aiProp("string", "Tipe kelas santri persis."),
				"frekuensi":         aiProp("string", "Frekuensi persis."),
				"angkatan":          aiProp("string", "Angkatan pendaftaran persis."),
				"angkatan_kelas":    aiProp("string", "Angkatan kelas persis."),
				"jenis_kelamin":     aiProp("string", "Jenis kelamin persis (mis. L / P)."),
				"kata":              aiProp("string", "Cari nama / ID mahasantri."),
				"daftar_dari":       aiProp("string", "Tanggal daftar mulai YYYY-MM-DD."),
				"daftar_sampai":     aiProp("string", "Tanggal daftar sampai YYYY-MM-DD."),
			})),
			Roles: aiRolesSantri,
			run:   s.toolDaftarSantri,
		},
		{
			Name:        "daftar_kelas",
			Title:       "Daftar kelas",
			Description: "Semua kelas beserta guru, jumlah santri aktif/cuti, kapasitas, pertemuan 30 hari, pertemuan belum ditutup, dan kehadiran 30 hari. Default hanya kelas aktif.",
			InputSchema: aiSchema(map[string]any{
				"termasuk_nonaktif": aiProp("boolean", "true = sertakan kelas nonaktif."),
				"guru_id":           aiIntProp("Hanya kelas guru ini.", 1, math.MaxInt32),
				"level":             aiProp("string", "Level persis."),
				"tipe":              aiProp("string", "Tipe persis."),
				"angkatan":          aiProp("string", "Angkatan persis."),
			}),
			Roles: aiRolesRiayah,
			run:   s.toolDaftarKelas,
		},
		{
			Name:        "daftar_guru",
			Title:       "Daftar guru",
			Description: "Semua guru (tanpa kontak) dengan jumlah kelas & santri aktif, pertemuan 30 hari, pertemuan belum ditutup, tindak lanjut kunjungan, dan kehadiran pembinaan/rapat 90 hari. Default hanya guru aktif.",
			InputSchema: aiSchema(map[string]any{
				"termasuk_nonaktif": aiProp("boolean", "true = sertakan guru nonaktif."),
			}),
			Roles: aiRolesKoordinator,
			run:   s.toolDaftarGuru,
		},
		{
			Name:        "daftar_tagihan",
			Title:       "Daftar tagihan",
			Description: "Daftar tagihan SPP per santri dengan filter status, terlambat, kelas, angkatan kelas, dan rentang tanggal tagih. Urut: belum bayar dengan jatuh tempo terlama dulu. Berhalaman.",
			InputSchema: aiSchema(aiWithPaging(map[string]any{
				"status":         aiEnumProp("Status tagihan.", "belum_bayar", "lunas", "batal"),
				"terlambat":      aiProp("boolean", "true = hanya belum_bayar yang lewat jatuh tempo."),
				"santri_id":      aiIntProp("Hanya tagihan santri ini.", 1, math.MaxInt32),
				"kelas_id":       aiIntProp("Hanya tagihan kelas ini.", 1, math.MaxInt32),
				"angkatan_kelas": aiProp("string", "Angkatan kelas persis."),
				"tagih_dari":     aiProp("string", "Tanggal tagih mulai YYYY-MM-DD."),
				"tagih_sampai":   aiProp("string", "Tanggal tagih sampai YYYY-MM-DD."),
			})),
			Roles: aiRolesKeuangan,
			run:   s.toolDaftarTagihan,
		},
	}
}

func (s *AIAgentService) toolDaftarSantri(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	var f models.AISantriDaftarFilter
	var err error
	if f.Status, err = a.enum("status", "", "aktif", "cuti", "nonaktif", "tidak_lanjut"); err != nil {
		return nil, err
	}
	if f.KelasID, err = a.int("kelas_id", 0, 0, math.MaxInt32); err != nil {
		return nil, err
	}
	if f.GuruID, err = a.int("guru_id", 0, 0, math.MaxInt32); err != nil {
		return nil, err
	}
	if f.DaftarDari, err = aiDateArg(a, "daftar_dari"); err != nil {
		return nil, err
	}
	if f.DaftarSampai, err = aiDateArg(a, "daftar_sampai"); err != nil {
		return nil, err
	}
	var page int64
	if f.Limit, f.Offset, page, err = aiPaging(a, 100); err != nil {
		return nil, err
	}
	f.Level, f.Tipe, f.Frekuensi = a.str("level"), a.str("tipe"), a.str("frekuensi")
	f.Angkatan, f.AngkatanKelas, f.JenisKelamin, f.Kata = a.str("angkatan"), a.str("angkatan_kelas"), a.str("jenis_kelamin"), a.str("kata")
	f.TanpaKelasAktif, f.BelumLengkap = a.bool("tanpa_kelas_aktif"), a.bool("belum_lengkap")

	rows, total, err := s.querier.AIDaftarSantri(ctx, f)
	if err != nil {
		return nil, err
	}
	hasil := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, map[string]any{"santri": r, "link": aiSantriLink(p, r.ID)})
	}
	out := aiPageInfo(total, page, f.Limit, len(hasil))
	out["data_per"] = s.today().Format("2006-01-02")
	out["santri"] = hasil
	if f.TanpaKelasAktif || f.BelumLengkap {
		out["catatan"] = "Definisi sama dengan angka di dashboard: tanpa kelas aktif = tidak punya kelas atau kelasnya nonaktif; belum lengkap = data Admin Kelas belum lengkap."
	}
	return out, nil
}

func (s *AIAgentService) toolDaftarKelas(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	guruID, err := a.int("guru_id", 0, 0, math.MaxInt32)
	if err != nil {
		return nil, err
	}
	now := s.today()
	metrik, err := s.querier.AIKelasMetrik(ctx, now.Format("2006-01-02"), now.AddDate(0, 0, -30).Format("2006-01-02"), 0, a.bool("termasuk_nonaktif"))
	if err != nil {
		return nil, err
	}
	level, tipe, angkatan := a.str("level"), a.str("tipe"), a.str("angkatan")
	hasil := make([]map[string]any, 0, len(metrik))
	for _, k := range metrik {
		if (guruID > 0 && k.GuruID != guruID) || (level != "" && k.Level != level) || (tipe != "" && k.Tipe != tipe) || (angkatan != "" && k.Angkatan != angkatan) {
			continue
		}
		k.PersenHadir30 = aiRoundPtr(k.PersenHadir30)
		hasil = append(hasil, map[string]any{"kelas": k, "link": aiKelasLink(p, k.ID)})
	}
	return map[string]any{"data_per": now.Format("2006-01-02"), "total": len(hasil), "kelas": hasil}, nil
}

func (s *AIAgentService) toolDaftarGuru(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	now := s.today()
	metrik, err := s.querier.AIGuruMetrik(ctx, now.Format("2006-01-02"), now.AddDate(0, 0, -30).Format("2006-01-02"),
		now.AddDate(0, 0, -90).Format("2006-01-02"), 0, a.bool("termasuk_nonaktif"))
	if err != nil {
		return nil, err
	}
	hasil := make([]map[string]any, 0, len(metrik))
	for _, g := range metrik {
		hasil = append(hasil, map[string]any{"guru": g, "link": aiGuruLink(p, g.ID)})
	}
	return map[string]any{"data_per": now.Format("2006-01-02"), "total": len(hasil), "guru": hasil}, nil
}

func (s *AIAgentService) toolDaftarTagihan(ctx context.Context, p *models.AIPrincipal, a aiArgs) (any, error) {
	now := s.today()
	f := models.AITagihanDaftarFilter{HariIni: now.Format("2006-01-02"), Terlambat: a.bool("terlambat"), AngkatanKelas: a.str("angkatan_kelas")}
	var err error
	if f.Status, err = a.enum("status", "", "belum_bayar", "lunas", "batal"); err != nil {
		return nil, err
	}
	if f.SantriID, err = a.int("santri_id", 0, 0, math.MaxInt32); err != nil {
		return nil, err
	}
	if f.KelasID, err = a.int("kelas_id", 0, 0, math.MaxInt32); err != nil {
		return nil, err
	}
	if f.TagihDari, err = aiDateArg(a, "tagih_dari"); err != nil {
		return nil, err
	}
	if f.TagihSampai, err = aiDateArg(a, "tagih_sampai"); err != nil {
		return nil, err
	}
	var page int64
	if f.Limit, f.Offset, page, err = aiPaging(a, 100); err != nil {
		return nil, err
	}
	rows, total, err := s.querier.AIDaftarTagihan(ctx, f)
	if err != nil {
		return nil, err
	}
	hasil := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, map[string]any{"tagihan": r, "link": aiLink(p, fmt.Sprintf("/app/keuangan/tagihan/%d", r.ID))})
	}
	out := aiPageInfo(total, page, f.Limit, len(hasil))
	out["data_per"] = f.HariIni
	out["tagihan"] = hasil
	return out, nil
}
