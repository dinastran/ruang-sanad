package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

var (
	ErrLinkTidakAktif = errors.New("link pendaftaran tidak aktif")
	slugPattern       = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// Optional form fields an admin may show/require per link. Nama, no WA,
	// gender and (for reguler) jadwal are always shown and required.
	registrationOptionalFields = []string{"usia", "domisili", "riwayat_belajar", "belajar_dimana", "untuk_siapa", "waktu_belajar"}
	// Fields a short form shows by default; the rest only appear in full mode.
	registrationShortFields = map[string]bool{"usia": true, "domisili": true}
)

// DefaultProofDir keeps payment proofs outside ./storage, which is served
// publicly by app.Static; staff open them through an authenticated route.
const DefaultProofDir = "data/payment_proofs"

// RegistrationService owns public registration links, form submissions and
// manual payment proofs. Lead changes go through LeadService.mutateLeadAs.
type RegistrationService struct {
	querier  *queries.Querier
	leads    *LeadService
	jadwal   *JadwalPenawaranService
	secret   []byte
	now      func() time.Time
	proofDir string

	gateway    PaymentGateway    // nil = online payment off
	affiliates *AffiliateService // nil = no affiliate attribution
	meta       *MetaCAPIService  // nil = no Meta Pixel / CAPI web events
	appURL     string

	mu        sync.Mutex
	phoneHits map[string][]time.Time
	clickSeen map[string]struct{}
	starting  map[string]bool // refs whose gateway session is being created
	clickDay  string
}

func NewRegistrationService(q *queries.Querier, leads *LeadService, jadwal *JadwalPenawaranService, secret []byte) *RegistrationService {
	return &RegistrationService{
		querier: q, leads: leads, jadwal: jadwal, secret: secret,
		now: time.Now, proofDir: DefaultProofDir,
		phoneHits: map[string][]time.Time{}, clickSeen: map[string]struct{}{}, starting: map[string]bool{},
	}
}

func (s *RegistrationService) nowUTC() time.Time { return s.now().UTC().Truncate(time.Second) }

func (s *RegistrationService) Settings(ctx context.Context) (models.RegistrationSettings, error) {
	r, err := s.querier.GetRegistrationSettings(ctx)
	if err != nil {
		return models.RegistrationSettings{}, err
	}
	out := models.RegistrationSettings{CsWaNumber: r.CsWaNumber, PaymentExpiryHours: r.PaymentExpiryHours, BankManualInfo: r.BankManualInfo}
	// Before structured accounts are saved, prefill them from the old text.
	out.Rekening = rekeningEfektif(r.RekeningBank, r.BankManualInfo)
	if out.Rekening == nil {
		out.Rekening = []models.RekeningBank{}
	}
	return out, nil
}

func (s *RegistrationService) SaveSettings(ctx context.Context, in models.RegistrationSettings, actorID int64) error {
	in.CsWaNumber = strings.TrimSpace(in.CsWaNumber)
	if in.CsWaNumber != "" {
		in.CsWaNumber = NormalizeLeadPhone(in.CsWaNumber)
		if in.CsWaNumber == "" {
			return errors.New("nomor WA CS tidak valid")
		}
	}
	if in.PaymentExpiryHours < 1 || in.PaymentExpiryHours > 72 {
		return errors.New("masa berlaku tagihan harus 1–72 jam")
	}
	rekening, err := normalizeRekening(in.Rekening)
	if err != nil {
		return err
	}
	rekeningJSON := ""
	if len(rekening) > 0 {
		b, _ := json.Marshal(rekening)
		rekeningJSON = string(b)
	}
	return s.querier.UpdateRegistrationSettings(ctx, queries.UpdateRegistrationSettingsParams{
		CsWaNumber: in.CsWaNumber, PaymentExpiryHours: in.PaymentExpiryHours, RekeningBank: rekeningJSON,
		BankManualInfo: truncateRunes(strings.TrimSpace(in.BankManualInfo), 1000),
		UpdatedBy:      sql.NullInt64{Int64: actorID, Valid: actorID > 0}, UpdatedAt: s.nowUTC(),
	})
}

func normalizeFieldConfig(in map[string]models.FieldRule) map[string]models.FieldRule {
	out := make(map[string]models.FieldRule, len(registrationOptionalFields))
	for _, f := range registrationOptionalFields {
		r, ok := in[f]
		if !ok {
			continue
		}
		if r.Wajib {
			r.Tampil = true
		}
		out[f] = r
	}
	return out
}

func parseFieldConfig(raw string) map[string]models.FieldRule {
	out := map[string]models.FieldRule{}
	_ = json.Unmarshal([]byte(raw), &out)
	return normalizeFieldConfig(out)
}

// effectiveFields resolves which optional fields a link shows: the admin's
// field_config wins, otherwise short mode shows usia + domisili (optional)
// and full mode shows all of them (optional).
func effectiveFields(mode string, cfg map[string]models.FieldRule) map[string]models.FieldRule {
	out := map[string]models.FieldRule{}
	for _, f := range registrationOptionalFields {
		if r, ok := cfg[f]; ok {
			if r.Tampil {
				out[f] = r
			}
			continue
		}
		if mode == "full" || registrationShortFields[f] {
			out[f] = models.FieldRule{Tampil: true}
		}
	}
	return out
}

func (s *RegistrationService) validateLink(ctx context.Context, req *models.RegistrationLinkRequest) error {
	req.Slug = strings.TrimSpace(req.Slug)
	req.Judul = truncateRunes(strings.TrimSpace(req.Judul), 120)
	req.Deskripsi = truncateRunes(strings.TrimSpace(req.Deskripsi), 1000)
	req.WaTemplate = truncateRunes(strings.TrimSpace(req.WaTemplate), 500)
	if len(req.Slug) < 3 || len(req.Slug) > 60 || !slugPattern.MatchString(req.Slug) {
		return errors.New("slug hanya huruf kecil, angka, dan tanda minus (3–60 karakter)")
	}
	if req.Judul == "" {
		return errors.New("judul wajib diisi")
	}
	switch req.Mode {
	case "wa", "short", "full":
	default:
		return errors.New("mode harus wa, short, atau full")
	}
	if req.AfterSubmit != "wa" && req.AfterSubmit != "bayar" {
		return errors.New("aksi setelah submit harus wa atau bayar")
	}
	if req.NominalOverride != nil && *req.NominalOverride < 0 {
		return errors.New("nominal tidak boleh negatif")
	}
	if req.LeadSourceID <= 0 {
		return errors.New("sumber lead wajib dipilih")
	}
	if _, err := s.querier.GetLeadSource(ctx, req.LeadSourceID); err != nil {
		return errors.New("sumber lead tidak ditemukan")
	}
	prog, err := s.querier.GetLeadProgram(ctx, req.ProgramID)
	if err != nil || prog == nil {
		return errors.New("program tidak ditemukan")
	}
	if prog.ProdukID <= 0 {
		return errors.New("program belum terhubung ke produk; tautkan produk di menu Program dulu")
	}
	settings, err := s.Settings(ctx)
	if err != nil {
		return err
	}
	if settings.CsWaNumber == "" {
		return errors.New("isi nomor WA CS di pengaturan Link Pendaftaran dulu")
	}
	req.FieldConfig = normalizeFieldConfig(req.FieldConfig)
	extra := strings.TrimSpace(req.MetaPixelExtra)
	req.MetaPixelExtra = strings.Join(ParseMetaPixelIDs(extra), ",")
	if extra != "" && req.MetaPixelExtra == "" {
		return errors.New("Pixel ID tambahan harus angka (6–20 digit), pisahkan dengan koma")
	}
	if len(ParseMetaPixelIDs(req.MetaPixelExtra)) > 3 {
		return errors.New("maksimal 3 Pixel ID tambahan")
	}
	events, err := NormalizeMetaEvents(req.MetaEvents)
	if err != nil {
		return err
	}
	req.MetaEvents = events
	return nil
}

func nullAmount(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

func (s *RegistrationService) CreateLink(ctx context.Context, req models.RegistrationLinkRequest, actorID int64) (int64, error) {
	if err := s.validateLink(ctx, &req); err != nil {
		return 0, err
	}
	cfg, _ := json.Marshal(req.FieldConfig)
	metaEvents, _ := json.Marshal(req.MetaEvents)
	now := s.nowUTC()
	link, err := s.querier.CreateRegistrationLink(ctx, queries.CreateRegistrationLinkParams{
		Slug: req.Slug, Judul: req.Judul, Deskripsi: req.Deskripsi, ProgramID: req.ProgramID, LeadSourceID: req.LeadSourceID,
		Mode: req.Mode, AfterSubmit: req.AfterSubmit, NominalOverride: nullAmount(req.NominalOverride),
		FieldConfig: string(cfg), WaTemplate: req.WaTemplate, IzinkanManual: boolInt(req.IzinkanManual),
		IsAktif: boolInt(req.IsAktif), CreatedBy: sql.NullInt64{Int64: actorID, Valid: actorID > 0}, CreatedAt: now, UpdatedAt: now,
		TampilKodeReferal: boolInt(req.TampilKodeReferal == nil || *req.TampilKodeReferal),
		MetaTracking:      boolInt(req.MetaTracking == nil || *req.MetaTracking), MetaPixelExtra: req.MetaPixelExtra,
		MetaEvents: string(metaEvents),
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return 0, errors.New("slug sudah dipakai link lain")
		}
		return 0, err
	}
	return link.ID, nil
}

func (s *RegistrationService) UpdateLink(ctx context.Context, id int64, req models.RegistrationLinkRequest) error {
	if _, err := s.querier.GetRegistrationLink(ctx, id); err != nil {
		return errors.New("link tidak ditemukan")
	}
	if err := s.validateLink(ctx, &req); err != nil {
		return err
	}
	cfg, _ := json.Marshal(req.FieldConfig)
	metaEvents, _ := json.Marshal(req.MetaEvents)
	err := s.querier.UpdateRegistrationLink(ctx, queries.UpdateRegistrationLinkParams{
		Slug: req.Slug, Judul: req.Judul, Deskripsi: req.Deskripsi, ProgramID: req.ProgramID, LeadSourceID: req.LeadSourceID,
		Mode: req.Mode, AfterSubmit: req.AfterSubmit, NominalOverride: nullAmount(req.NominalOverride),
		FieldConfig: string(cfg), WaTemplate: req.WaTemplate, IzinkanManual: boolInt(req.IzinkanManual),
		IsAktif: boolInt(req.IsAktif), UpdatedAt: s.nowUTC(), ID: id,
		TampilKodeReferal: boolInt(req.TampilKodeReferal == nil || *req.TampilKodeReferal),
		MetaTracking:      boolInt(req.MetaTracking == nil || *req.MetaTracking), MetaPixelExtra: req.MetaPixelExtra,
		MetaEvents: string(metaEvents),
	})
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return errors.New("slug sudah dipakai link lain")
	}
	return err
}

func linkToRequest(l queries.RegistrationLink) models.RegistrationLinkRequest {
	tampil := l.TampilKodeReferal == 1
	tracking := l.MetaTracking == 1
	var override *int64
	if l.NominalOverride.Valid {
		v := l.NominalOverride.Int64
		override = &v
	}
	return models.RegistrationLinkRequest{
		Slug: l.Slug, Judul: l.Judul, Deskripsi: l.Deskripsi, ProgramID: l.ProgramID, LeadSourceID: l.LeadSourceID,
		Mode: l.Mode, AfterSubmit: l.AfterSubmit, NominalOverride: override, FieldConfig: parseFieldConfig(l.FieldConfig),
		WaTemplate: l.WaTemplate, IzinkanManual: l.IzinkanManual == 1, IsAktif: l.IsAktif == 1,
		TampilKodeReferal: &tampil, MetaTracking: &tracking, MetaPixelExtra: l.MetaPixelExtra,
		MetaEvents: ParseMetaEvents(l.MetaEvents),
	}
}

func rupiahLabel(n int64) string {
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return "Rp" + b.String()
}

// activeVarian returns the active price options of a produk.
func (s *RegistrationService) activeVarian(ctx context.Context, produkID int64) ([]queries.ProdukVarian, error) {
	rows, err := s.querier.ListProdukVarianByProduk(ctx, produkID)
	if err != nil {
		return nil, err
	}
	out := rows[:0]
	for _, r := range rows {
		if r.IsAktif == 1 {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *RegistrationService) ListLinks(ctx context.Context) ([]models.RegistrationLinkRow, error) {
	links, err := s.querier.ListRegistrationLinks(ctx)
	if err != nil {
		return nil, err
	}
	since := s.now().In(wib).AddDate(0, 0, -29).Format(time.DateOnly)
	stats, err := s.querier.ListRegistrationStatsSince(ctx, since)
	if err != nil {
		return nil, err
	}
	byLink := map[int64]queries.ListRegistrationStatsSinceRow{}
	for _, st := range stats {
		byLink[st.LinkID] = st
	}
	out := make([]models.RegistrationLinkRow, 0, len(links))
	for _, l := range links {
		row := models.RegistrationLinkRow{RegistrationLinkRequest: linkToRequest(l), ID: l.ID}
		if prog, err := s.querier.GetLeadProgram(ctx, l.ProgramID); err == nil && prog != nil {
			row.ProgramNama = prog.Nama
			row.HargaLabel = s.hargaLabel(ctx, l, prog.ProdukID)
		}
		if src, err := s.querier.GetLeadSource(ctx, l.LeadSourceID); err == nil {
			row.SumberNama = src.Nama
		}
		st := byLink[l.ID]
		row.Klik, row.Submit, row.Bayar = st.Klik, st.Submit, st.Bayar
		out = append(out, row)
	}
	return out, nil
}

func (s *RegistrationService) hargaLabel(ctx context.Context, l queries.RegistrationLink, produkID int64) string {
	if l.NominalOverride.Valid {
		return rupiahLabel(l.NominalOverride.Int64)
	}
	vars, err := s.activeVarian(ctx, produkID)
	if err == nil && len(vars) > 0 {
		lo, hi := vars[0].Harga, vars[0].Harga
		for _, v := range vars {
			lo, hi = min(lo, v.Harga), max(hi, v.Harga)
		}
		if lo == hi {
			return rupiahLabel(lo)
		}
		return rupiahLabel(lo) + "–" + rupiahLabel(hi)
	}
	p, err := s.querier.GetProduk(ctx, produkID)
	if err != nil {
		return "-"
	}
	return rupiahLabel(p.Harga)
}

// linkContext loads a link with its program and produk, failing with
// ErrLinkTidakAktif when any of them is missing or inactive.
func (s *RegistrationService) linkContext(ctx context.Context, slug string) (queries.RegistrationLink, *models.LeadProgram, models.Product, error) {
	l, err := s.querier.GetRegistrationLinkBySlug(ctx, slug)
	if err != nil || l.IsAktif != 1 {
		return l, nil, models.Product{}, ErrLinkTidakAktif
	}
	prog, err := s.querier.GetLeadProgram(ctx, l.ProgramID)
	if err != nil || prog == nil || !prog.IsAktif || prog.ProdukID <= 0 {
		return l, nil, models.Product{}, ErrLinkTidakAktif
	}
	p, err := s.querier.GetProduk(ctx, prog.ProdukID)
	if err != nil || !p.IsAktif {
		return l, nil, models.Product{}, ErrLinkTidakAktif
	}
	return l, prog, p, nil
}

func (s *RegistrationService) PublicLink(ctx context.Context, slug string) (*models.PublicRegistrationLink, error) {
	l, prog, produk, err := s.linkContext(ctx, slug)
	if err != nil {
		return nil, err
	}
	out := &models.PublicRegistrationLink{
		ID: l.ID, Slug: l.Slug, Judul: l.Judul, Deskripsi: l.Deskripsi, Mode: l.Mode, AfterSubmit: l.AfterSubmit,
		ProgramNama: prog.Nama, ButuhJadwal: prog.Jenis == "reguler", Harga: produk.Harga,
		FieldConfig: effectiveFields(l.Mode, parseFieldConfig(l.FieldConfig)), IzinkanManual: l.IzinkanManual == 1,
		KodeReferal: s.affiliates != nil && l.TampilKodeReferal == 1,
		Varian:      []models.PublicVarian{},
		Meta:        s.publicMeta(l),
	}
	if l.NominalOverride.Valid {
		out.Harga = l.NominalOverride.Int64
	}
	vars, err := s.activeVarian(ctx, prog.ProdukID)
	if err != nil {
		return nil, err
	}
	for _, v := range vars {
		harga := v.Harga
		if l.NominalOverride.Valid {
			harga = l.NominalOverride.Int64
		}
		out.Varian = append(out.Varian, models.PublicVarian{ID: v.ID, Nama: v.Nama, Harga: harga})
	}
	return out, nil
}

// ResolveAmount computes the amount due: link override > varian > produk.
// varianID is required when the produk has active varian.
func (s *RegistrationService) ResolveAmount(ctx context.Context, link queries.RegistrationLink, varianID int64) (int64, *queries.ProdukVarian, error) {
	prog, err := s.querier.GetLeadProgram(ctx, link.ProgramID)
	if err != nil || prog == nil || prog.ProdukID <= 0 {
		return 0, nil, ErrLinkTidakAktif
	}
	vars, err := s.activeVarian(ctx, prog.ProdukID)
	if err != nil {
		return 0, nil, err
	}
	var chosen *queries.ProdukVarian
	if len(vars) > 0 {
		for i := range vars {
			if vars[i].ID == varianID {
				chosen = &vars[i]
			}
		}
		if chosen == nil {
			return 0, nil, &FieldError{Field: "varian_id", Pesan: "Silakan pilih paket"}
		}
	}
	switch {
	case link.NominalOverride.Valid:
		return link.NominalOverride.Int64, chosen, nil
	case chosen != nil:
		return chosen.Harga, chosen, nil
	}
	p, err := s.querier.GetProduk(ctx, prog.ProdukID)
	if err != nil {
		return 0, nil, err
	}
	return p.Harga, nil, nil
}

// FieldError is a validation error tied to one form field.
type FieldError struct {
	Field string
	Pesan string
}

func (e *FieldError) Error() string { return e.Pesan }

// WithAffiliates enables affiliate attribution on form submits.
func (s *RegistrationService) WithAffiliates(a *AffiliateService) *RegistrationService {
	s.affiliates = a
	return s
}
