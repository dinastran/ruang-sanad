package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/maulanashalihin/laju-go/app/queries"
)

// AffCookieName is the first-party cookie that remembers an affiliate click.
const AffCookieName = "rs_aff"

var waRefRe = regexp.MustCompile(`(?i)\bref\s*:\s*([A-Z0-9]+(?:-[A-Z0-9]+)*)`)

func (s *AffiliateService) affSig(kode, ts string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte("aff|" + kode + "|" + ts))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))[:22]
}

// AffCookie signs "KODE.unix" so the cookie cannot be forged or extended.
func (s *AffiliateService) AffCookie(kode string) string {
	ts := strconv.FormatInt(s.nowUTC().Unix(), 10)
	kode = strings.ToUpper(kode)
	return kode + "." + ts + "." + s.affSig(kode, ts)
}

// ParseAffCookie returns the affiliate code and click time of a valid,
// unexpired cookie (expiry = masa_berlaku_hari setting).
func (s *AffiliateService) ParseAffCookie(ctx context.Context, v string) (string, time.Time, bool) {
	parts := strings.Split(v, ".")
	if len(parts) != 3 || !hmac.Equal([]byte(parts[2]), []byte(s.affSig(parts[0], parts[1]))) {
		return "", time.Time{}, false
	}
	unix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", time.Time{}, false
	}
	at := time.Unix(unix, 0).UTC()
	st, err := s.Settings(ctx)
	if err != nil {
		return "", time.Time{}, false
	}
	if s.nowUTC().After(at.Add(time.Duration(st.MasaBerlakuHari) * 24 * time.Hour)) {
		return "", time.Time{}, false
	}
	return parts[0], at, true
}

// ClickResult tells the /r/:kode handler what to do.
type ClickResult struct {
	Found  bool   // affiliate exists and is active
	Kode   string // canonical code
	Cookie string // new cookie value to set ("" = keep the existing one)
	MaxAge int    // seconds
}

// TrackClick counts a click (deduplicated per IP+UA per WIB day) and decides
// the cookie. First-click: a valid unexpired cookie is never overwritten.
func (s *AffiliateService) TrackClick(ctx context.Context, kode, ip, ua, existing string) (ClickResult, error) {
	kode = strings.ToUpper(strings.TrimSpace(kode))
	a, err := s.querier.GetAffiliateByKode(ctx, kode)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && a.Status != "aktif") {
		return ClickResult{}, nil
	}
	if err != nil {
		return ClickResult{}, err
	}
	res := ClickResult{Found: true, Kode: a.Kode}
	day := s.now().In(wib).Format(time.DateOnly)
	key := strconv.FormatInt(a.ID, 10) + "|" + ip + "|" + ua
	s.mu.Lock()
	if s.clickDay != day {
		s.clickDay, s.clickSeen = day, map[string]struct{}{}
	}
	_, seen := s.clickSeen[key]
	s.clickSeen[key] = struct{}{}
	s.mu.Unlock()
	if !seen {
		if err := s.querier.BumpAffiliateClick(ctx, queries.BumpAffiliateClickParams{AffiliateID: a.ID, Tanggal: day}); err != nil {
			return res, err
		}
	}
	if _, _, ok := s.ParseAffCookie(ctx, existing); ok {
		return res, nil
	}
	st, err := s.Settings(ctx)
	if err != nil {
		return res, err
	}
	res.Cookie = s.AffCookie(a.Kode)
	res.MaxAge = int(st.MasaBerlakuHari) * 24 * 3600
	return res, nil
}

// lockLeadTx attributes a lead to an affiliate inside the caller's
// transaction. First-click: an existing attribution is kept. Inactive
// affiliates and self-referrals (same WhatsApp number) are ignored.
// Returns true when a new attribution was written.
func lockLeadTx(ctx context.Context, q *queries.Querier, l queries.Lead, kode, sumber string, clickedAt sql.NullTime, now time.Time) (bool, error) {
	if kode == "" {
		return false, nil
	}
	a, err := q.GetAffiliateByKode(ctx, strings.ToUpper(kode))
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if a.Status != "aktif" || (a.NoWa != "" && a.NoWa == l.Phone) {
		return false, nil
	}
	n, err := q.InsertLeadAffiliate(ctx, queries.InsertLeadAffiliateParams{
		LeadID: l.ID, AffiliateID: a.ID, Sumber: sumber, DiklikAt: clickedAt, DikunciAt: now,
	})
	if err != nil || n == 0 {
		return false, err
	}
	return true, writeLeadEvent(ctx, q, l.ID, newLeadEventID("aff"), "affiliate.locked", "affiliate:"+strconv.FormatInt(a.ID, 10),
		map[string]any{"affiliate_id": a.ID, "kode": a.Kode, "sumber": sumber}, now)
}

// submitAttribution is resolved before the submit transaction (it reads
// settings; SQLite runs one connection, so no reads outside q inside a tx).
type submitAttribution struct {
	cookieKode string
	clickedAt  time.Time
	kodeForm   string
}

func (s *AffiliateService) resolveSubmit(ctx context.Context, cookie, kodeForm string) submitAttribution {
	out := submitAttribution{kodeForm: strings.TrimSpace(kodeForm)}
	if kode, at, ok := s.ParseAffCookie(ctx, cookie); ok {
		out.cookieKode, out.clickedAt = kode, at
	}
	return out
}

// attributeSubmitTx runs the form attribution order: cookie, then the
// "kode referal" field.
func attributeSubmitTx(ctx context.Context, q *queries.Querier, l queries.Lead, a submitAttribution, now time.Time) error {
	if a.cookieKode != "" {
		done, err := lockLeadTx(ctx, q, l, a.cookieKode, "cookie", sql.NullTime{Time: a.clickedAt, Valid: true}, now)
		if err != nil || done {
			return err
		}
	}
	_, err := lockLeadTx(ctx, q, l, a.kodeForm, "kode_form", sql.NullTime{}, now)
	return err
}

// detectWARef attributes a lead from "ref: KODE" in any inbound message
// while the lead has no affiliate yet.
func (s *AffiliateService) detectWARef(ctx context.Context, q *queries.Querier, l queries.Lead, body string, now time.Time) error {
	m := waRefRe.FindStringSubmatch(body)
	if m == nil {
		return nil
	}
	_, err := lockLeadTx(ctx, q, l, m[1], "kode_wa", sql.NullTime{}, now)
	return err
}

// SetLeadAffiliate lets staff set, change (affiliateID > 0) or clear (0) a
// lead's affiliate.
func (s *AffiliateService) SetLeadAffiliate(ctx context.Context, leadID, affiliateID, actorID int64) error {
	return s.leads.mutateLead(ctx, leadID, actorID, func(q *queries.Querier, l *queries.Lead, now time.Time) (string, map[string]any, error) {
		payload := map[string]any{}
		if prev, err := q.GetLeadAffiliate(ctx, l.ID); err == nil {
			payload["dari"] = prev.Kode
		}
		if affiliateID <= 0 {
			if err := q.DeleteLeadAffiliate(ctx, l.ID); err != nil {
				return "", nil, err
			}
			payload["ke"] = ""
			return "affiliate.changed", payload, nil
		}
		a, err := q.GetAffiliate(ctx, affiliateID)
		if err != nil {
			return "", nil, ErrAffiliateTidakAda
		}
		if err := q.ReplaceLeadAffiliate(ctx, queries.ReplaceLeadAffiliateParams{
			LeadID: l.ID, AffiliateID: a.ID, DikunciAt: now, DiubahOleh: sql.NullInt64{Int64: actorID, Valid: actorID > 0},
		}); err != nil {
			return "", nil, err
		}
		payload["ke"] = a.Kode
		return "affiliate.changed", payload, nil
	})
}

// Destination is where /r/:kode sends the visitor: the registration link
// "ke" (or the default link). A mode-wa link goes straight to WhatsApp with
// "(ref: KODE)" in the message so the chat is attributed too.
func (s *AffiliateService) Destination(ctx context.Context, ke, kode string) string {
	slug := strings.TrimSpace(ke)
	if slug == "" {
		st, err := s.Settings(ctx)
		if err != nil || !st.LinkTujuanDefaultID.Valid {
			return "/"
		}
		l, err := s.querier.GetRegistrationLink(ctx, st.LinkTujuanDefaultID.Int64)
		if err != nil {
			return "/"
		}
		slug = l.Slug
	}
	l, err := s.querier.GetRegistrationLinkBySlug(ctx, slug)
	if err != nil || l.IsAktif != 1 {
		return "/"
	}
	if l.Mode != "wa" {
		return "/daftar/" + l.Slug
	}
	settings, err := s.querier.GetRegistrationSettings(ctx)
	if err != nil || settings.CsWaNumber == "" {
		return "/daftar/" + l.Slug
	}
	program := ""
	if p, err := s.querier.GetLeadProgram(ctx, l.ProgramID); err == nil && p != nil {
		program = p.Nama
	}
	tpl := l.WaTemplate
	if strings.TrimSpace(tpl) == "" {
		tpl = "Assalamu'alaikum, saya ingin mendaftar {program}."
	}
	text := strings.TrimSpace(fillWaTemplate(tpl, "", program, "", ""))
	if kode != "" {
		text += " (ref: " + kode + ")"
	}
	return waURL(settings.CsWaNumber, text)
}
