package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/maulanashalihin/laju-go/app/models"
	"github.com/maulanashalihin/laju-go/app/queries"
)

// WithMeta enables the Meta Pixel and CAPI web events on public pages.
func (s *RegistrationService) WithMeta(m *MetaCAPIService) *RegistrationService {
	s.meta = m
	return s
}

// MetaPixelInfo is shown on the admin page: the main Pixel from .env,
// whether CAPI (server events) is on, and the choices for the event picker.
func (s *RegistrationService) MetaPixelInfo() models.MetaPixelInfo {
	return models.MetaPixelInfo{
		PixelID: s.meta.PixelID(), CAPIAktif: s.meta != nil && s.meta.Enabled(),
		StandardEvents: MetaStandardEvents, FunnelTrigger: MetaFunnelTrigger, DefaultEvents: DefaultMetaEvents(),
	}
}

// linkPixelIDs is the main Pixel plus the link's extra Pixels; empty when
// tracking is off for the link.
func (s *RegistrationService) linkPixelIDs(l queries.RegistrationLink) []string {
	if s.meta == nil || l.MetaTracking != 1 {
		return nil
	}
	ids := []string{}
	if p := s.meta.PixelID(); p != "" {
		ids = append(ids, p)
	}
	for _, id := range ParseMetaPixelIDs(l.MetaPixelExtra) {
		if len(ids) > 0 && id == ids[0] {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func newMetaEventToken(prefix string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func newMetaViewEventID() string { return newMetaEventToken("rs-vc-") }

func (s *RegistrationService) publicMeta(l queries.RegistrationLink) *models.PublicMeta {
	ids := s.linkPixelIDs(l)
	if len(ids) == 0 {
		return nil
	}
	out := &models.PublicMeta{PixelIDs: ids, ViewEventID: newMetaViewEventID(), Langkah1: []models.MetaBrowserEvent{}}
	for _, name := range ParseMetaEvents(l.MetaEvents)[MetaTriggerLangkah1] {
		out.Langkah1 = append(out.Langkah1, models.MetaBrowserEvent{Name: name, Currency: "IDR", Custom: !IsStandardMetaEvent(name)})
	}
	return out
}

// TrackPageView sends the CAPI copy of the page's ViewContent (async).
func (s *RegistrationService) TrackPageView(link *models.PublicRegistrationLink, sig MetaWebSignals) {
	if link == nil || link.Meta == nil || s.meta == nil {
		return
	}
	s.meta.SendViewContentAsync(link.Meta.ViewEventID, link.ProgramNama, sig)
}

// triggerEvents builds the events of one trigger. A CRM funnel event is
// included only when this action reached it (reached), with the CRM
// event_id so Meta dedupes it against the queued CAPI copy. Any other event
// gets its own event_id and its CAPI copy is sent here (async).
func (s *RegistrationService) triggerEvents(names []string, reached map[string]bool, l *queries.Lead, value int64, content string, sig MetaWebSignals) []models.MetaBrowserEvent {
	out := []models.MetaBrowserEvent{}
	var ud map[string]any
	if l != nil {
		ud = metaUserData(*l)
	}
	for _, name := range names {
		ev := models.MetaBrowserEvent{Name: name, Value: max(value, 0), Currency: "IDR", Custom: !IsStandardMetaEvent(name)}
		if _, funnel := MetaFunnelTrigger[name]; funnel {
			if !reached[name] || l == nil {
				continue
			}
			ev.ID = metaEventID(l.ID, name)
			out = append(out, ev)
			continue
		}
		ev.ID = newMetaEventToken("rs-ev-")
		out = append(out, ev)
		if s.meta != nil {
			s.meta.SendWebEventAsync(toWebEvent(ev, content, ud, sig))
		}
	}
	return out
}

func toWebEvent(ev models.MetaBrowserEvent, content string, ud map[string]any, sig MetaWebSignals) MetaWebEvent {
	return MetaWebEvent{
		Name: ev.Name, ID: ev.ID, UserData: ud, Signals: sig,
		Custom: map[string]any{"currency": "IDR", "value": ev.Value, "content_name": truncateRunes(content, 120)},
	}
}

// browserTracking lists the Pixel events of a form submit.
func (s *RegistrationService) browserTracking(link queries.RegistrationLink, l queries.Lead, reached []string, content string, sig MetaWebSignals) *models.MetaBrowserTracking {
	got := map[string]bool{}
	for _, n := range reached {
		got[n] = true
	}
	names := ParseMetaEvents(link.MetaEvents)[MetaTriggerSubmit]
	return &models.MetaBrowserTracking{
		Events: s.triggerEvents(names, got, &l, l.PaymentAmount, content, sig),
		User:   MetaBrowserUserData(l),
	}
}

// WaTracking is the Pixel page shown before redirecting a mode-wa link to
// WhatsApp; nil when the link has no event for that trigger (then the visitor
// is redirected right away, as before).
func (s *RegistrationService) WaTracking(ctx context.Context, slug string, sig MetaWebSignals) *models.MetaBrowserTracking {
	l, prog, _, err := s.linkContext(ctx, slug)
	if err != nil {
		return nil
	}
	ids := s.linkPixelIDs(l)
	names := ParseMetaEvents(l.MetaEvents)[MetaTriggerWA]
	if len(ids) == 0 || len(names) == 0 {
		return nil
	}
	return &models.MetaBrowserTracking{
		PixelIDs: ids, User: map[string]string{},
		Events: s.triggerEvents(names, nil, nil, 0, prog.Nama, sig),
	}
}

// ProofTracking runs after a proof upload: it stores the browser signals (so
// the queued AddPaymentInfo goes out as a website event) and returns the
// Pixel events of the "bukti" trigger. nil when tracking is off.
func (s *RegistrationService) ProofTracking(ctx context.Context, ref string, sig MetaWebSignals) *models.MetaBrowserTracking {
	reg, err := s.querier.GetLeadRegistrationByRef(ctx, ref)
	if err != nil {
		return nil
	}
	link, err := s.querier.GetRegistrationLink(ctx, reg.LinkID)
	if err != nil || len(s.linkPixelIDs(link)) == 0 {
		return nil
	}
	l, err := s.querier.GetLeadByID(ctx, reg.LeadID)
	if err != nil || l.Status == LeadStatusDoNotContact {
		return nil
	}
	if err := saveWebSignals(ctx, s.querier, l.ID, sig, s.nowUTC()); err != nil {
		return nil
	}
	program := ""
	if prog, err := s.querier.GetLeadProgram(ctx, link.ProgramID); err == nil && prog != nil {
		program = prog.Nama
	}
	names := ParseMetaEvents(link.MetaEvents)[MetaTriggerBukti]
	// The upload just moved the lead to bukti_masuk: AddPaymentInfo reached.
	reached := map[string]bool{MetaEventAddPaymentInfo: l.ProofReceivedAt.Valid}
	return &models.MetaBrowserTracking{
		Events: s.triggerEvents(names, reached, &l, l.PaymentAmount, program, sig),
		User:   MetaBrowserUserData(l),
	}
}

// bayarMeta is the Pixel config of the payment page.
func (s *RegistrationService) bayarMeta(link queries.RegistrationLink, l queries.Lead) *models.BayarMeta {
	ids := s.linkPixelIDs(link)
	if len(ids) == 0 {
		return nil
	}
	return &models.BayarMeta{PixelIDs: ids, User: MetaBrowserUserData(l)}
}
