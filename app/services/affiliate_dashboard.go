package services

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

func ruleInfo(r *queries.AffiliateRule) *models.AffiliateRuleInfo {
	if r == nil {
		return nil
	}
	out := &models.AffiliateRuleInfo{Pola: r.Pola}
	if r.Jenis == "persen" {
		v := strconv.FormatFloat(float64(r.Nilai)/100, 'f', -1, 64)
		out.Label = v + "%"
	} else {
		out.Label = rupiahLabel(r.Nilai)
	}
	if r.Pola == "bulanan" {
		if r.MaksBulan.Valid {
			out.Durasi = strconv.FormatInt(r.MaksBulan.Int64, 10) + " bulan"
		} else {
			out.Durasi = "selama santri aktif"
		}
	}
	return out
}

func rujukanStatus(r queries.ListAffiliateReferralsRow) string {
	switch {
	case r.Status == LeadStatusLost || r.Status == LeadStatusDoNotContact:
		return "batal"
	case r.SantriID.Valid:
		return "santri"
	case r.VerifiedAt.Valid:
		return "terverifikasi"
	}
	return "daftar"
}

// Dashboard is the "Affiliate Saya" page data. Only first names of referrals
// are exposed (privacy, spec §5.6).
func (s *AffiliateService) Dashboard(ctx context.Context, id int64) (*models.AffiliateDashboard, error) {
	a, err := s.get(ctx, id)
	if err != nil {
		return nil, err
	}
	st, err := s.Settings(ctx)
	if err != nil {
		return nil, err
	}
	d := &models.AffiliateDashboard{
		ID: a.ID, Kode: a.Kode, Nama: a.Nama, Tipe: a.Tipe, BankNama: a.BankNama, BankRekening: a.BankRekening,
		BankAtasNama: a.BankAtasNama, MasaBerlaku: st.MasaBerlakuHari, MasaTahan: st.MasaTahanHari,
		TemplateBagi: st.TemplateBagikan, Rujukan: []models.AffiliateRujukan{}, Tujuan: []models.AffiliateTujuan{},
	}
	since := s.now().In(wib).AddDate(0, 0, -29)
	clicks, err := s.querier.SumAffiliateClicksSince(ctx, since.Format(time.DateOnly))
	if err != nil {
		return nil, err
	}
	for _, c := range clicks {
		if c.AffiliateID == a.ID {
			d.Klik30 = c.Klik
		}
	}
	refs, err := s.querier.ListAffiliateReferrals(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	cutoff := since.UTC()
	for _, r := range refs {
		if !r.DikunciAt.Before(cutoff) {
			d.Daftar30++
		}
		status := rujukanStatus(r)
		if status == "santri" || status == "terverifikasi" {
			d.Santri++
		}
		nama := r.Nama
		if nama == "" {
			nama = r.WaName
		}
		d.Rujukan = append(d.Rujukan, models.AffiliateRujukan{
			NamaDepan: firstName(nama), Program: r.ProgramNama, Tanggal: r.DikunciAt.In(wib).Format(time.DateOnly),
			Status: status, Komisi: r.Komisi,
		})
	}
	sums, err := s.querier.AffiliateCommissionSums(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	for _, x := range sums {
		switch x.Status {
		case "pending":
			d.KomisiPending = x.Total
		case "siap":
			d.KomisiSiap = x.Total
		case "dibayar":
			d.KomisiDibayar = x.Total
		}
	}
	rule, err := s.ResolveRule(ctx, a.ID, 0) // affiliate rule, else default
	if err != nil {
		return nil, err
	}
	d.Aturan = ruleInfo(rule)
	links, err := s.querier.ListRegistrationLinks(ctx)
	if err != nil {
		return nil, err
	}
	for _, l := range links {
		if l.IsAktif == 1 {
			d.Tujuan = append(d.Tujuan, models.AffiliateTujuan{Slug: l.Slug, Judul: l.Judul})
			if st.LinkTujuanDefaultID.Valid && st.LinkTujuanDefaultID.Int64 == l.ID {
				d.TujuanDefault = l.Slug
			}
		}
	}
	if d.TujuanDefault == "" && len(d.Tujuan) > 0 {
		d.TujuanDefault = d.Tujuan[0].Slug
	}
	return d, nil
}

// AdminRows lists affiliates with their 30-day clicks and totals.
func (s *AffiliateService) AdminRows(ctx context.Context) ([]models.AffiliateRow, error) {
	list, err := s.querier.ListAffiliates(ctx)
	if err != nil {
		return nil, err
	}
	since := s.now().In(wib).AddDate(0, 0, -29)
	clicks, err := s.querier.SumAffiliateClicksSince(ctx, since.Format(time.DateOnly))
	if err != nil {
		return nil, err
	}
	klik := map[int64]int64{}
	for _, c := range clicks {
		klik[c.AffiliateID] = c.Klik
	}
	counts, err := s.querier.AffiliateReferralCounts(ctx, since.UTC())
	if err != nil {
		return nil, err
	}
	daftar, santri := map[int64]int64{}, map[int64]int64{}
	for _, c := range counts {
		daftar[c.AffiliateID], santri[c.AffiliateID] = c.Daftar, c.Santri
	}
	totals, err := s.querier.AffiliateCommissionTotals(ctx)
	if err != nil {
		return nil, err
	}
	siap, total := map[int64]int64{}, map[int64]int64{}
	for _, t := range totals {
		siap[t.AffiliateID] = t.Siap
		total[t.AffiliateID] = t.Siap + t.Pending + t.Dibayar
	}
	out := make([]models.AffiliateRow, 0, len(list))
	for _, a := range list {
		out = append(out, models.AffiliateRow{
			ID: a.ID, Kode: a.Kode, Tipe: a.Tipe, Nama: a.Nama, NoWA: a.NoWa, Email: a.Email, Status: a.Status,
			BankNama: a.BankNama, BankRekening: a.BankRekening, BankAtasNama: a.BankAtasNama, PunyaAkun: a.UserID.Valid,
			Klik: klik[a.ID], Daftar: daftar[a.ID], Santri: santri[a.ID], KomisiSiap: siap[a.ID], KomisiTotal: total[a.ID],
		})
	}
	return out, nil
}

// Ledger lists commissions (status "" = all), newest first.
func (s *AffiliateService) Ledger(ctx context.Context, status string) ([]models.AffiliateLedgerRow, error) {
	rows, err := s.querier.ListCommissionLedger(ctx, status)
	if err != nil {
		return nil, err
	}
	out := make([]models.AffiliateLedgerRow, 0, len(rows))
	for _, r := range rows {
		nama := r.LeadNama
		if nama == "" {
			nama = r.LeadWaName
		}
		rek := strings.TrimSpace(strings.Join([]string{r.BankNama, r.BankRekening, r.BankAtasNama}, " · "))
		if strings.Trim(rek, " ·") == "" {
			rek = ""
		}
		out = append(out, models.AffiliateLedgerRow{
			ID: r.ID, AffiliateID: r.AffiliateID, AffiliateNama: r.AffiliateNama, AffiliateKode: r.AffiliateKode,
			AffiliateTipe: r.AffiliateTipe, Rekening: rek, LeadID: r.LeadID, LeadNama: nama, Sumber: r.Sumber,
			BulanKe: r.BulanKe, Basis: r.Basis, Jumlah: r.Jumlah, Status: r.Status, MetodeBayar: r.MetodeBayar,
			UjrohBulan: r.UjrohBulan, Catatan: r.Catatan, AlasanBatal: r.AlasanBatal, CreatedAt: fmtTime(r.CreatedAt),
		})
	}
	return out, nil
}


// AffiliateAdminOptions are the pickers on the admin page.
type AffiliateAdminOptions struct {
	Programs []models.Option          `json:"programs"`
	Links    []models.AffiliateTujuan `json:"links"`
}

func (s *AffiliateService) AdminOptions(ctx context.Context) (AffiliateAdminOptions, error) {
	out := AffiliateAdminOptions{Programs: []models.Option{}, Links: []models.AffiliateTujuan{}}
	progs, err := s.querier.ListLeadPrograms(ctx)
	if err != nil {
		return out, err
	}
	for _, p := range progs {
		out.Programs = append(out.Programs, models.Option{ID: p.ID, Nama: p.Nama, Aktif: p.IsAktif})
	}
	links, err := s.querier.ListRegistrationLinks(ctx)
	if err != nil {
		return out, err
	}
	for _, l := range links {
		out.Links = append(out.Links, models.AffiliateTujuan{Slug: strconv.FormatInt(l.ID, 10), Judul: l.Judul + " (/daftar/" + l.Slug + ")"})
	}
	return out, nil
}

// LeadPanel is the affiliate info on a lead page plus active affiliates to pick.
func (s *AffiliateService) LeadPanel(ctx context.Context, leadID int64) (map[string]any, error) {
	out := map[string]any{"current": nil}
	if la, err := s.querier.GetLeadAffiliate(ctx, leadID); err == nil {
		out["current"] = map[string]any{"affiliate_id": la.AffiliateID, "kode": la.Kode, "nama": la.Nama, "tipe": la.Tipe,
			"sumber": la.Sumber, "dikunci_at": fmtTime(la.DikunciAt)}
	}
	list, err := s.querier.ListAffiliates(ctx)
	if err != nil {
		return nil, err
	}
	opts := []models.Option{}
	for _, a := range list {
		if a.Status == "aktif" {
			opts = append(opts, models.Option{ID: a.ID, Nama: a.Nama + " (" + a.Kode + ")", Aktif: true})
		}
	}
	out["options"] = opts
	return out, nil
}
