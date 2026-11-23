package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
)

// Browser side of Meta tracking for the public registration pages.
//
// The Pixel fires in the browser and the same events go to Meta through
// CAPI with the same event_id, so Meta counts each once (deduplication) and
// still gets the event when the Pixel is blocked by an ad blocker or iOS.
//
//   - ViewContent: page view of /daftar/:slug, sent directly (no lead yet).
//   - Lead, CompleteRegistration, InitiateCheckout, AddPaymentInfo: the CRM
//     funnel events of meta_capi.go. The browser fires them with the event_id
//     the CRM queue uses; the queued CAPI copy carries the browser signals
//     stored in lead_web_signals (fbp/fbc, IP, user agent, page URL).

// metaWebWindow: a CRM milestone reached within this time of the stored
// browser signals happened on the website (form submit / proof upload).
const metaWebWindow = 15 * time.Minute

// metaWebEvents are funnel events that can happen on the public pages.
var metaWebEvents = map[string]bool{
	MetaEventLead: true, MetaEventCompleteRegistration: true,
	MetaEventInitiateCheckout: true, MetaEventAddPaymentInfo: true,
}

var (
	metaPixelIDPattern = regexp.MustCompile(`^[0-9]{6,20}$`)
	metaFbclidPattern  = regexp.MustCompile(`^[A-Za-z0-9_\-.]{10,500}$`)
)

// PixelID is the main Pixel loaded in the browser: META_PIXEL_ID, else the
// CAPI dataset ID (a Meta dataset ID is the Pixel ID).
func (s *MetaCAPIService) PixelID() string {
	if s == nil {
		return ""
	}
	if s.cfg.PixelID != "" {
		return s.cfg.PixelID
	}
	return s.cfg.DatasetID
}

// ParseMetaPixelIDs keeps valid, unique Pixel IDs from a comma/space list.
func ParseMetaPixelIDs(raw string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, f := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' || r == ';' }) {
		if metaPixelIDPattern.MatchString(f) && !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}

// MetaFbc returns the _fbc value: the cookie when set, otherwise one built
// from the fbclid of the page URL (format fb.1.<ms>.<fbclid>).
func MetaFbc(cookie, pageURL string, now time.Time) string {
	if c := strings.TrimSpace(cookie); strings.HasPrefix(c, "fb.") && len(c) <= 600 {
		return c
	}
	u, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	id := u.Query().Get("fbclid")
	if !metaFbclidPattern.MatchString(id) {
		return ""
	}
	return "fb.1." + strconv.FormatInt(now.UnixMilli(), 10) + "." + id
}

// MetaFbp keeps a well-formed _fbp cookie.
func MetaFbp(cookie string) string {
	c := strings.TrimSpace(cookie)
	if strings.HasPrefix(c, "fb.") && len(c) <= 200 {
		return c
	}
	return ""
}

// SafePageURL keeps an http(s) URL of the given host, for event_source_url.
// Anything else falls back to base+path.
func SafePageURL(raw, fallback string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	f, ferr := url.Parse(fallback)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || (ferr == nil && f.Host != "" && u.Host != f.Host) {
		return fallback
	}
	u.Fragment = ""
	return truncateRunes(u.String(), 1000)
}

// MetaWebSignals is what the browser tells us about the visitor.
type MetaWebSignals struct {
	Fbp, Fbc, IP, UserAgent, SourceURL string
}

// SaveWebSignals stores the latest browser signals of a lead (tx-safe).
func saveWebSignals(ctx context.Context, q *queries.Querier, leadID int64, sig MetaWebSignals, now time.Time) error {
	return q.UpsertLeadWebSignals(ctx, queries.UpsertLeadWebSignalsParams{
		LeadID: leadID, Fbp: sig.Fbp, Fbc: sig.Fbc,
		ClientIp: truncateRunes(sig.IP, 64), UserAgent: truncateRunes(sig.UserAgent, 400),
		SourceUrl: truncateRunes(sig.SourceURL, 1000), UpdatedAt: now,
	})
}

// webSignals loads the stored browser signals of a lead; nil when none.
func (s *MetaCAPIService) webSignals(ctx context.Context, leadID int64) (*queries.LeadWebSignal, error) {
	sig, err := s.querier.GetLeadWebSignals(ctx, leadID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sig, nil
}

// applyWebSignals adds browser identifiers to a queued CRM event. fbp/fbc
// always help matching; an event that happened on the website (close in time
// to the signals) is sent as action_source=website with IP, user agent and
// page URL, so Meta can dedupe it against the Pixel event.
func applyWebSignals(out map[string]any, eventName string, eventTime time.Time, sig *queries.LeadWebSignal) {
	if sig == nil {
		return
	}
	ud, _ := out["user_data"].(map[string]any)
	if ud == nil {
		return
	}
	if sig.Fbp != "" {
		ud["fbp"] = sig.Fbp
	}
	if sig.Fbc != "" {
		ud["fbc"] = sig.Fbc
	}
	d := eventTime.Sub(sig.UpdatedAt)
	if d < 0 {
		d = -d
	}
	if !metaWebEvents[eventName] || d > metaWebWindow || sig.UserAgent == "" || sig.SourceUrl == "" {
		return
	}
	out["action_source"] = "website"
	out["event_source_url"] = sig.SourceUrl
	ud["client_user_agent"] = sig.UserAgent
	if sig.ClientIp != "" {
		ud["client_ip_address"] = sig.ClientIp
	}
}

// MetaBrowserUserData is the hashed advanced-matching data for the Pixel
// (same values as the CAPI user_data, so both sides match the same person).
func MetaBrowserUserData(l queries.Lead) map[string]string {
	out := map[string]string{}
	for k, v := range metaUserData(l) {
		if vs, ok := v.([]string); ok && len(vs) > 0 {
			out[k] = vs[0]
		}
	}
	return out
}

// webEventSlots caps concurrent direct sends (ViewContent per page view).
var webEventSlots = make(chan struct{}, 8)

// SendViewContentAsync sends a ViewContent server event for a page view,
// sharing eventID with the browser Pixel. It never blocks the request: when
// CAPI is off or too many sends are in flight, the event is skipped (the
// Pixel still fires).
func (s *MetaCAPIService) SendViewContentAsync(eventID, contentName string, sig MetaWebSignals) {
	if s == nil || !s.Enabled() || eventID == "" || sig.UserAgent == "" || sig.SourceURL == "" {
		return
	}
	select {
	case webEventSlots <- struct{}{}:
	default:
		return
	}
	ud := map[string]any{"client_user_agent": sig.UserAgent}
	if sig.IP != "" {
		ud["client_ip_address"] = sig.IP
	}
	if sig.Fbp != "" {
		ud["fbp"] = sig.Fbp
	}
	if sig.Fbc != "" {
		ud["fbc"] = sig.Fbc
	}
	event := map[string]any{
		"event_name": "ViewContent", "event_time": s.nowUTC().Unix(), "event_id": eventID,
		"action_source": "website", "event_source_url": sig.SourceURL, "user_data": ud,
		"custom_data": map[string]any{"content_name": truncateRunes(contentName, 120), "content_category": "pendaftaran"},
	}
	go func() {
		defer func() { <-webEventSlots }()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := s.post(ctx, event); err != nil {
			slog.Warn("Meta CAPI ViewContent gagal", "error", err)
		}
	}()
}
