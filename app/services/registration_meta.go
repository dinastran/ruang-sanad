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

// MetaPixelInfo is shown on the admin page: the main Pixel from .env and
// whether CAPI (server events) is on.
func (s *RegistrationService) MetaPixelInfo() models.MetaPixelInfo {
	return models.MetaPixelInfo{PixelID: s.meta.PixelID(), CAPIAktif: s.meta != nil && s.meta.Enabled()}
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

func newMetaViewEventID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "rs-vc-" + hex.EncodeToString(b)
}

func (s *RegistrationService) publicMeta(l queries.RegistrationLink) *models.PublicMeta {
	ids := s.linkPixelIDs(l)
	if len(ids) == 0 {
		return nil
	}
	return &models.PublicMeta{PixelIDs: ids, ViewEventID: newMetaViewEventID()}
}

// TrackPageView sends the CAPI copy of the page's ViewContent (async).
func (s *RegistrationService) TrackPageView(link *models.PublicRegistrationLink, sig MetaWebSignals) {
	if link == nil || link.Meta == nil || s.meta == nil {
		return
	}
	s.meta.SendViewContentAsync(link.Meta.ViewEventID, link.ProgramNama, sig)
}

// browserTracking lists the funnel events this submit reached, for the
// Pixel, with the CRM event_ids so Meta dedupes them against CAPI.
func browserTracking(l queries.Lead, events []string) *models.MetaBrowserTracking {
	out := &models.MetaBrowserTracking{Events: []models.MetaBrowserEvent{}, User: MetaBrowserUserData(l)}
	for _, name := range events {
		out.Events = append(out.Events, models.MetaBrowserEvent{
			Name: name, ID: metaEventID(l.ID, name), Value: max(l.PaymentAmount, 0), Currency: "IDR",
		})
	}
	return out
}

// RecordProofSignals stores browser signals at a proof upload, so the
// AddPaymentInfo CAPI event is sent as a website event.
func (s *RegistrationService) RecordProofSignals(ctx context.Context, ref string, sig MetaWebSignals) error {
	reg, err := s.querier.GetLeadRegistrationByRef(ctx, ref)
	if err != nil {
		return ErrRefTidakAda
	}
	link, err := s.querier.GetRegistrationLink(ctx, reg.LinkID)
	if err != nil || len(s.linkPixelIDs(link)) == 0 {
		return err
	}
	return saveWebSignals(ctx, s.querier, reg.LeadID, sig, s.nowUTC())
}

// bayarMeta is the Pixel config of the payment page.
func (s *RegistrationService) bayarMeta(link queries.RegistrationLink, l queries.Lead) *models.BayarMeta {
	ids := s.linkPixelIDs(link)
	if len(ids) == 0 {
		return nil
	}
	return &models.BayarMeta{
		PixelIDs: ids, AddPaymentInfoID: metaEventID(l.ID, MetaEventAddPaymentInfo),
		Value: max(l.PaymentAmount, 0), User: MetaBrowserUserData(l),
	}
}
