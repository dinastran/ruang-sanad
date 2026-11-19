package services

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var (
	ErrAffiliateTidakAda = errors.New("affiliate tidak ditemukan")
	affiliateKodeRe      = regexp.MustCompile(`^[A-Z0-9]+(-[A-Z0-9]+)*$`)
)

// AffiliateService owns affiliates, commission rules, attribution of leads
// and the commission ledger (spec docs/superpowers/specs/2026-11-19-affiliate-design.md).
type AffiliateService struct {
	querier *queries.Querier
	leads   *LeadService
	secret  []byte
	now     func() time.Time

	mu        sync.Mutex
	clickDay  string
	clickSeen map[string]struct{}
}

func NewAffiliateService(q *queries.Querier, leads *LeadService, secret []byte) *AffiliateService {
	return &AffiliateService{querier: q, leads: leads, secret: secret, now: time.Now, clickSeen: map[string]struct{}{}}
}

func (s *AffiliateService) nowUTC() time.Time { return s.now().UTC().Truncate(time.Second) }

// kodeBase turns a name into an uppercase code stem ("Ahmad Fauzi" → "AHMAD").
func kodeBase(nama string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.Fields(nama + " x")[0]) {
		if (r >= 'A' && r <= 'Z') || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
		if b.Len() >= 12 {
			break
		}
	}
	if b.Len() < 3 {
		return "RS"
	}
	return b.String()
}

// uniqueKode returns stem, then stem-2, stem-3… until unused.
func (s *AffiliateService) uniqueKode(ctx context.Context, q *queries.Querier, stem string) (string, error) {
	for i := 1; i < 1000; i++ {
		k := stem
		if i > 1 || len(stem) < 3 {
			k = stem + "-" + strconv.Itoa(i)
		}
		_, err := q.GetAffiliateByKode(ctx, k)
		if errors.Is(err, sql.ErrNoRows) {
			return k, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", errors.New("gagal membuat kode affiliate")
}

// EnsureGuruAffiliates gives every active guru with a login an affiliate row.
func (s *AffiliateService) EnsureGuruAffiliates(ctx context.Context) (int, error) {
	created := 0
	err := s.leads.withTx(ctx, func(q *queries.Querier) error {
		gurus, err := q.ListGuruWithoutAffiliate(ctx)
		if err != nil {
			return err
		}
		now := s.nowUTC()
		for _, g := range gurus {
			kode, err := s.uniqueKode(ctx, q, kodeBase(g.Nama))
			if err != nil {
				return err
			}
			if _, err := q.CreateAffiliate(ctx, queries.CreateAffiliateParams{
				Kode: kode, Tipe: "guru", GuruID: sql.NullInt64{Int64: g.ID, Valid: true}, UserID: g.UserID,
				Nama: g.Nama, Status: "aktif", CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				return err
			}
			created++
		}
		return nil
	})
	return created, err
}

// Register records an external affiliate signup (status menunggu).
func (s *AffiliateService) Register(ctx context.Context, in models.AffiliateSignup) (int64, error) {
	in.Nama = strings.Join(strings.Fields(in.Nama), " ")
	if n := utf8.RuneCountInString(in.Nama); n < 3 || n > 80 {
		return 0, &FieldError{Field: "nama", Pesan: "Nama lengkap wajib diisi (3–80 huruf)"}
	}
	phone := NormalizeLeadPhone(in.NoWA)
	if phone == "" || !strings.HasPrefix(phone, "62") || len(phone) < 10 {
		return 0, &FieldError{Field: "no_wa", Pesan: "Nomor WhatsApp tidak valid"}
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if !strings.Contains(in.Email, "@") || len(in.Email) > 120 {
		return 0, &FieldError{Field: "email", Pesan: "Email wajib diisi untuk login ke halaman affiliate"}
	}
	var id int64
	err := s.leads.withTx(ctx, func(q *queries.Querier) error {
		list, err := q.ListAffiliates(ctx)
		if err != nil {
			return err
		}
		for _, a := range list {
			if a.NoWa == phone && a.Status != "ditolak" {
				return &FieldError{Field: "no_wa", Pesan: "Nomor ini sudah terdaftar sebagai affiliate"}
			}
			if a.Email == in.Email && a.Status != "ditolak" {
				return &FieldError{Field: "email", Pesan: "Email ini sudah terdaftar sebagai affiliate"}
			}
		}
		// Never attach an affiliate signup to an existing login: whoever
		// registers would receive that account's set-password link.
		if _, err := q.GetUserByEmail(ctx, in.Email); err == nil {
			return &FieldError{Field: "email", Pesan: "Email ini sudah dipakai akun lain. Gunakan email yang berbeda."}
		} else if !errors.Is(err, queries.ErrUserNotFound) {
			return err
		}
		kode, err := s.uniqueKode(ctx, q, kodeBase(in.Nama))
		if err != nil {
			return err
		}
		now := s.nowUTC()
		a, err := q.CreateAffiliate(ctx, queries.CreateAffiliateParams{
			Kode: kode, Tipe: "eksternal", Nama: in.Nama, NoWa: phone, Email: in.Email, Status: "menunggu",
			BankNama: truncateRunes(strings.TrimSpace(in.BankNama), 60), BankRekening: truncateRunes(strings.TrimSpace(in.BankRekening), 40),
			BankAtasNama: truncateRunes(strings.TrimSpace(in.BankAtasNama), 80), CreatedAt: now, UpdatedAt: now,
		})
		id = a.ID
		return err
	})
	return id, err
}

func (s *AffiliateService) get(ctx context.Context, id int64) (queries.Affiliate, error) {
	a, err := s.querier.GetAffiliate(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return a, ErrAffiliateTidakAda
	}
	return a, err
}

func (s *AffiliateService) save(ctx context.Context, a queries.Affiliate) error {
	return s.querier.UpdateAffiliate(ctx, queries.UpdateAffiliateParams{
		Kode: a.Kode, Nama: a.Nama, NoWa: a.NoWa, Email: a.Email, Status: a.Status, BankNama: a.BankNama,
		BankRekening: a.BankRekening, BankAtasNama: a.BankAtasNama, CatatanAdmin: a.CatatanAdmin, UserID: a.UserID,
		DisetujuiOleh: a.DisetujuiOleh, DisetujuiAt: a.DisetujuiAt, UpdatedAt: s.nowUTC(), ID: a.ID,
	})
}

// SetStatus changes an affiliate's status (aktif, ditolak, nonaktif).
func (s *AffiliateService) SetStatus(ctx context.Context, id int64, status string, actorID int64) error {
	switch status {
	case "aktif", "ditolak", "nonaktif":
	default:
		return errors.New("status tidak valid")
	}
	a, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	if status == "aktif" && !a.DisetujuiAt.Valid {
		a.DisetujuiAt = sql.NullTime{Time: s.nowUTC(), Valid: true}
		a.DisetujuiOleh = sql.NullInt64{Int64: actorID, Valid: actorID > 0}
	}
	a.Status = status
	return s.save(ctx, a)
}

// UpdateKode lets an admin change an affiliate's code.
func (s *AffiliateService) UpdateKode(ctx context.Context, id int64, kode string) error {
	kode = strings.ToUpper(strings.TrimSpace(kode))
	if len(kode) < 3 || len(kode) > 20 || !affiliateKodeRe.MatchString(kode) {
		return errors.New("kode hanya huruf, angka, dan tanda minus (3–20 karakter)")
	}
	a, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	if other, err := s.querier.GetAffiliateByKode(ctx, kode); err == nil && other.ID != id {
		return errors.New("kode sudah dipakai affiliate lain")
	}
	a.Kode = kode
	return s.save(ctx, a)
}

// UpdateBank lets an affiliate (or admin) change payout details.
func (s *AffiliateService) UpdateBank(ctx context.Context, id int64, nama, rekening, atasNama string) error {
	a, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	a.BankNama = truncateRunes(strings.TrimSpace(nama), 60)
	a.BankRekening = truncateRunes(strings.TrimSpace(rekening), 40)
	a.BankAtasNama = truncateRunes(strings.TrimSpace(atasNama), 80)
	return s.save(ctx, a)
}

func (s *AffiliateService) List(ctx context.Context) ([]queries.Affiliate, error) {
	return s.querier.ListAffiliates(ctx)
}

// --- Rules & settings ---

func (s *AffiliateService) SaveRule(ctx context.Context, r models.AffiliateRuleRequest, actorID int64) error {
	switch r.Scope {
	case "global":
		r.TargetID = 0
	case "program":
		if p, err := s.querier.GetLeadProgram(ctx, r.TargetID); err != nil || p == nil {
			return errors.New("program tidak ditemukan")
		}
	case "affiliate":
		if _, err := s.get(ctx, r.TargetID); err != nil {
			return errors.New("affiliate tidak ditemukan")
		}
	default:
		return errors.New("cakupan aturan tidak valid")
	}
	switch r.Jenis {
	case "persen":
		if r.Nilai < 0 || r.Nilai > 10000 {
			return errors.New("persen harus 0–100%")
		}
	case "flat":
		if r.Nilai < 0 {
			return errors.New("nominal tidak boleh negatif")
		}
	default:
		return errors.New("jenis komisi harus persen atau flat")
	}
	maks := sql.NullInt64{}
	switch r.Pola {
	case "sekali":
	case "bulanan":
		switch {
		case r.MaksBulan == nil:
			maks = sql.NullInt64{Int64: 6, Valid: true} // default (spec)
		case *r.MaksBulan == -1:
			maks = sql.NullInt64{} // selama santri aktif
		case *r.MaksBulan < 1:
			return errors.New("batas bulan minimal 1, atau pilih 'selama aktif'")
		default:
			maks = sql.NullInt64{Int64: *r.MaksBulan, Valid: true}
		}
	default:
		return errors.New("pola komisi harus sekali atau bulanan")
	}
	_, err := s.querier.UpsertAffiliateRule(ctx, queries.UpsertAffiliateRuleParams{
		Scope: r.Scope, TargetID: r.TargetID, Jenis: r.Jenis, Nilai: r.Nilai, Pola: r.Pola, MaksBulan: maks,
		IsAktif: boolInt(!r.Nonaktif), UpdatedBy: sql.NullInt64{Int64: actorID, Valid: actorID > 0}, UpdatedAt: s.nowUTC(),
	})
	return err
}

func (s *AffiliateService) DeleteRule(ctx context.Context, id int64) error {
	return s.querier.DeleteAffiliateRule(ctx, id)
}

func (s *AffiliateService) Rules(ctx context.Context) ([]queries.AffiliateRule, error) {
	return s.querier.ListAffiliateRules(ctx)
}

// ResolveRule returns the most specific active rule: affiliate > program >
// global. nil means no commission.
func ruleLookup(ctx context.Context, q *queries.Querier, affiliateID, programID int64) (*queries.AffiliateRule, error) {
	for _, k := range []queries.GetAffiliateRuleParams{
		{Scope: "affiliate", TargetID: affiliateID},
		{Scope: "program", TargetID: programID},
		{Scope: "global", TargetID: 0},
	} {
		if k.Scope == "program" && programID <= 0 {
			continue
		}
		r, err := q.GetAffiliateRule(ctx, k)
		if err == nil {
			return &r, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	return nil, nil
}

func (s *AffiliateService) ResolveRule(ctx context.Context, affiliateID, programID int64) (*queries.AffiliateRule, error) {
	return ruleLookup(ctx, s.querier, affiliateID, programID)
}

func (s *AffiliateService) Settings(ctx context.Context) (queries.AffiliateSetting, error) {
	return s.querier.GetAffiliateSettings(ctx)
}

func (s *AffiliateService) SaveSettings(ctx context.Context, in models.AffiliateSettings, actorID int64) error {
	switch in.MasaBerlakuHari {
	case 30, 60, 90:
	default:
		return errors.New("masa berlaku harus 30, 60, atau 90 hari")
	}
	if in.MasaTahanHari < 0 || in.MasaTahanHari > 90 {
		return errors.New("masa tahan harus 0–90 hari")
	}
	tujuan := sql.NullInt64{}
	if in.LinkTujuanDefaultID > 0 {
		if _, err := s.querier.GetRegistrationLink(ctx, in.LinkTujuanDefaultID); err != nil {
			return errors.New("link pendaftaran tujuan tidak ditemukan")
		}
		tujuan = sql.NullInt64{Int64: in.LinkTujuanDefaultID, Valid: true}
	}
	return s.querier.UpdateAffiliateSettings(ctx, queries.UpdateAffiliateSettingsParams{
		MasaBerlakuHari: in.MasaBerlakuHari, MasaTahanHari: in.MasaTahanHari, LinkTujuanDefaultID: tujuan,
		TemplateBagikan: truncateRunes(strings.TrimSpace(in.TemplateBagikan), 500),
		UpdatedBy:       sql.NullInt64{Int64: actorID, Valid: actorID > 0}, UpdatedAt: s.nowUTC(),
	})
}
