package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

func newRegistrationRef() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func leadRef(id int64) string { return "L-" + strconv.FormatInt(id, 10) }

func formActor(linkID int64) string { return "form:" + strconv.FormatInt(linkID, 10) }

// waURL builds the wa.me link to the CS number with a prefilled message.
func waURL(csNumber, text string) string {
	u := "https://wa.me/" + csNumber
	if text != "" {
		u += "?text=" + url.QueryEscape(text)
	}
	return u
}

const defaultWaTemplate = "Assalamu'alaikum, saya {nama} sudah isi form pendaftaran {program} ({ref})."

func fillWaTemplate(tpl, nama, program, jadwal, ref string) string {
	if strings.TrimSpace(tpl) == "" {
		tpl = defaultWaTemplate
	}
	return strings.NewReplacer("{nama}", nama, "{program}", program, "{jadwal}", jadwal, "{ref}", ref).Replace(tpl)
}

// CSWaURL is the plain WhatsApp link to CS (inactive-link fallback, mode wa).
func (s *RegistrationService) CSWaURL(ctx context.Context, text string) string {
	settings, err := s.Settings(ctx)
	if err != nil || settings.CsWaNumber == "" {
		return ""
	}
	return waURL(settings.CsWaNumber, text)
}

type submitInput struct {
	models.RegistrationSubmit
	phone       string
	jadwalLabel string
}

func (s *RegistrationService) validateSubmit(ctx context.Context, link queries.RegistrationLink, butuhJadwal bool, in *submitInput) error {
	in.Nama = strings.Join(strings.Fields(in.Nama), " ")
	if n := utf8.RuneCountInString(in.Nama); n < 2 || n > 80 {
		return &FieldError{Field: "nama", Pesan: "Nama lengkap wajib diisi (2–80 huruf)"}
	}
	in.phone = NormalizeLeadPhone(in.NoWA)
	if in.phone == "" || !strings.HasPrefix(in.phone, "62") || len(in.phone) < 10 {
		return &FieldError{Field: "no_wa", Pesan: "Nomor WhatsApp tidak valid"}
	}
	in.JenisKelamin = strings.ToUpper(strings.TrimSpace(in.JenisKelamin))
	if in.JenisKelamin != "L" && in.JenisKelamin != "P" {
		return &FieldError{Field: "jenis_kelamin", Pesan: "Pilih ikhwan atau akhwat"}
	}
	fields := effectiveFields(link.Mode, parseFieldConfig(link.FieldConfig))
	required := func(f string) bool { return fields[f].Wajib }
	if in.Usia != 0 && (in.Usia < 5 || in.Usia > 90) {
		return &FieldError{Field: "usia", Pesan: "Usia harus 5–90 tahun"}
	}
	if in.Usia == 0 && required("usia") {
		return &FieldError{Field: "usia", Pesan: "Usia wajib diisi"}
	}
	in.Domisili = truncateRunes(strings.TrimSpace(in.Domisili), 80)
	in.RiwayatBelajar = truncateRunes(strings.TrimSpace(in.RiwayatBelajar), 500)
	in.BelajarDimana = truncateRunes(strings.TrimSpace(in.BelajarDimana), 200)
	in.WaktuBelajar = truncateRunes(strings.TrimSpace(in.WaktuBelajar), 200)
	in.UntukSiapa = strings.TrimSpace(in.UntukSiapa)
	if !leadUntukSiapa[in.UntukSiapa] {
		return &FieldError{Field: "untuk_siapa", Pesan: "Pilihan tidak valid"}
	}
	for f, v := range map[string]string{"domisili": in.Domisili, "riwayat_belajar": in.RiwayatBelajar,
		"belajar_dimana": in.BelajarDimana, "untuk_siapa": in.UntukSiapa, "waktu_belajar": in.WaktuBelajar} {
		if v == "" && required(f) {
			return &FieldError{Field: f, Pesan: "Wajib diisi"}
		}
	}
	in.JadwalKode = strings.ToLower(strings.TrimSpace(in.JadwalKode))
	if !butuhJadwal {
		in.JadwalKode = ""
		return nil
	}
	if in.JadwalKode == "" {
		return &FieldError{Field: "jadwal_kode", Pesan: "Silakan pilih jadwal"}
	}
	open, err := s.jadwal.Open(ctx, in.JenisKelamin)
	if err != nil {
		return err
	}
	for _, j := range open {
		if j.Kode != in.JadwalKode {
			continue
		}
		sisa := j.SisaIkhwan
		if in.JenisKelamin == "P" {
			sisa = j.SisaAkhwat
		}
		if sisa > 0 {
			in.jadwalLabel = j.Label
			return nil
		}
	}
	return &FieldError{Field: "jadwal_kode", Pesan: "Jadwal ini sudah penuh, silakan pilih jadwal lain"}
}

// findOrCreateLead returns the lead for a phone number: by classic chat id,
// then by the phone column (covers @lid chats), else a new lead.
func findOrCreateLead(ctx context.Context, q *queries.Querier, phone string, now time.Time) (queries.Lead, bool, error) {
	chatID := phone + "@s.whatsapp.net"
	l, err := q.GetLeadByChatID(ctx, chatID)
	if err == nil {
		return l, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return l, false, err
	}
	l, err = q.GetLeadByPhone(ctx, phone)
	if err == nil {
		return l, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return l, false, err
	}
	l, err = q.CreateLead(ctx, queries.CreateLeadParams{WaChatID: chatID, Phone: phone, CreatedAt: now, UpdatedAt: now})
	return l, true, err
}

func fillEmpty(dst *string, v string) {
	if *dst == "" && v != "" {
		*dst = v
	}
}

// Submit records a public form submission as a lead (spec §5.3).
func (s *RegistrationService) Submit(ctx context.Context, slug string, raw models.RegistrationSubmit, meta models.RequestMeta) (*models.RegistrationResult, error) {
	link, prog, _, err := s.linkContext(ctx, slug)
	if err != nil {
		return nil, err
	}
	if link.Mode == "wa" {
		return nil, ErrLinkTidakAktif
	}
	if strings.TrimSpace(raw.Honeypot) != "" {
		// Bots get a believable answer and nothing is written.
		return &models.RegistrationResult{Next: "wa", WaURL: s.CSWaURL(ctx, "")}, nil
	}
	if err := s.CheckFormToken(slug, raw.FormToken); err != nil {
		return nil, err
	}
	in := submitInput{RegistrationSubmit: raw}
	if err := s.validateSubmit(ctx, link, prog.Jenis == "reguler", &in); err != nil {
		return nil, err
	}
	amount, varian, err := s.ResolveAmount(ctx, link, in.VarianID)
	if err != nil {
		return nil, err
	}
	if !s.allowPhone(in.phone) {
		return nil, &FieldError{Field: "no_wa", Pesan: "Nomor ini sudah beberapa kali mendaftar. Silakan lanjutkan lewat WhatsApp CS."}
	}
	settings, err := s.Settings(ctx)
	if err != nil {
		return nil, err
	}

	var attribution submitAttribution
	if s.affiliates != nil {
		attribution = s.affiliates.resolveSubmit(ctx, meta.AffCookie, in.KodeReferal)
	}
	res := &models.RegistrationResult{Next: link.AfterSubmit}
	actor := formActor(link.ID)
	err = s.leads.withTx(ctx, func(q *queries.Querier) error {
		now := s.nowUTC()
		l, created, err := findOrCreateLead(ctx, q, in.phone, now)
		if err != nil {
			return err
		}
		if created {
			if err := writeLeadEvent(ctx, q, l.ID, newLeadEventID("frm"), "lead.created", actor, map[string]any{"link_id": link.ID}, now); err != nil {
				return err
			}
		}
		res.LeadRef = leadRef(l.ID)

		switch l.Status {
		case LeadStatusWon, LeadStatusDoNotContact:
			l.NeedsReview = 1
			res.Next = "wa"
		case LeadStatusLost:
			from := l.Status
			l.Status, l.LostReason = LeadStatusOpen, ""
			if err := writeLeadEvent(ctx, q, l.ID, newLeadEventID("frm"), "lead.reopened", actor,
				map[string]any{"status_from": from, "status_to": LeadStatusOpen}, now); err != nil {
				return err
			}
		}
		open := l.Status == LeadStatusOpen

		if open {
			fillEmpty(&l.Nama, in.Nama)
			fillEmpty(&l.JenisKelamin, in.JenisKelamin)
			fillEmpty(&l.Domisili, in.Domisili)
			fillEmpty(&l.RiwayatBelajar, in.RiwayatBelajar)
			fillEmpty(&l.BelajarDimana, in.BelajarDimana)
			fillEmpty(&l.UntukSiapa, in.UntukSiapa)
			fillEmpty(&l.WaktuBelajar, in.WaktuBelajar)
			fillEmpty(&l.JadwalKode, in.JadwalKode)
			if !l.Usia.Valid && in.Usia > 0 {
				l.Usia = sql.NullInt64{Int64: in.Usia, Valid: true}
			}
			if varian != nil && l.PaketKode == "" && varian.PaketKode != "" {
				l.PaketKode = varian.PaketKode
				if l.KelasKode == "" {
					if crm, err := q.GetCRMSettings(ctx); err == nil {
						l.KelasKode = paketToKelasKode(crm, varian.PaketKode)
					}
				}
			}
			if !l.ProgramID.Valid {
				if err := q.SetLeadProgram(ctx, l.ID, link.ProgramID, now); err != nil {
					return err
				}
			}
			if !l.LeadSourceID.Valid {
				if err := q.SetLeadSource(ctx, queries.SetLeadSourceParams{
					LeadSourceID: sql.NullInt64{Int64: link.LeadSourceID, Valid: true}, LeadSourceBy: LeadSourceByForm, UpdatedAt: now, ID: l.ID,
				}); err != nil {
					return err
				}
			}
			target := LeadStageDataTerisi
			if link.AfterSubmit == "bayar" {
				target = LeadStageInfoBayarDikirim
				// The amount follows the link just submitted unless a proof is
				// already waiting or paid (then staff decide).
				if leadStageIndex(l.Stage) <= leadStageIndex(LeadStageInfoBayarDikirim) {
					l.PaymentAmount = amount
				}
			}
			if leadStageIndex(l.Stage) < leadStageIndex(target) {
				// Walk through data_terisi so CAPI sees each milestone.
				for _, st := range []string{LeadStageDataTerisi, target} {
					if leadStageIndex(l.Stage) >= leadStageIndex(st) {
						continue
					}
					from := l.Stage
					l.Stage = st
					if st == LeadStageInfoBayarDikirim && !l.PaymentInfoSentAt.Valid {
						l.PaymentInfoSentAt = sql.NullTime{Time: now, Valid: true}
					}
					if err := writeLeadEvent(ctx, q, l.ID, newLeadEventID("frm"), "stage.changed", actor,
						map[string]any{"stage_from": from, "stage_to": st}, now); err != nil {
						return err
					}
				}
			}
			if link.AfterSubmit == "bayar" && l.Stage != LeadStageTerverifikasi {
				res.Next = "bayar"
			} else if link.AfterSubmit == "bayar" {
				res.Next = "wa"
			}
		}
		if err := saveLead(ctx, q, l, now); err != nil {
			return err
		}

		if s.affiliates != nil {
			if err := attributeSubmitTx(ctx, q, l, attribution, now); err != nil {
				return err
			}
		}
		payload := in.RegistrationSubmit
		payload.FormToken, payload.Honeypot = "", ""
		body, _ := json.Marshal(payload)
		ref := newRegistrationRef()
		if _, err := q.CreateLeadRegistration(ctx, queries.CreateLeadRegistrationParams{
			LeadID: l.ID, LinkID: link.ID, Ref: ref, Payload: string(body),
			Ip: truncateRunes(meta.IP, 64), UserAgent: truncateRunes(meta.UserAgent, 300), CreatedAt: now,
		}); err != nil {
			return err
		}
		if res.Next == "bayar" {
			res.BayarRef = ref
		}
		if err := writeLeadEvent(ctx, q, l.ID, newLeadEventID("frm"), "registration.submitted", actor,
			map[string]any{"link_id": link.ID, "slug": link.Slug}, now); err != nil {
			return err
		}
		return q.BumpRegistrationStat(ctx, queries.BumpRegistrationStatParams{
			LinkID: link.ID, Tanggal: now.In(wib).Format(time.DateOnly), Submit: 1,
		})
	})
	if err != nil {
		return nil, err
	}
	res.WaURL = waURL(settings.CsWaNumber, fillWaTemplate(link.WaTemplate, in.Nama, prog.Nama, in.jadwalLabel, res.LeadRef))
	return res, nil
}
