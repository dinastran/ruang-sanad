package services

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Per-link Meta events. An admin picks which Pixel/CAPI events fire at each
// trigger of a registration link. The CRM funnel (meta_capi.go) stays global:
// a funnel event may only be chosen at the trigger where the CRM itself
// reaches that milestone, and then it shares the CRM event_id so Meta
// counts it once. Any other event gets its own event_id and is sent by the
// browser Pixel and (except langkah1) through CAPI.

const (
	MetaTriggerWA       = "wa"       // a mode-wa link is opened (before wa.me)
	MetaTriggerLangkah1 = "langkah1" // form step 1 completed (browser only)
	MetaTriggerSubmit   = "submit"   // form submitted
	MetaTriggerBukti    = "bukti"    // payment proof uploaded

	metaEventsPerTrigger = 5
)

// MetaTriggers in display order.
var MetaTriggers = []string{MetaTriggerWA, MetaTriggerLangkah1, MetaTriggerSubmit, MetaTriggerBukti}

// MetaStandardEvents are Meta's standard web events an admin can pick.
// Purchase is left out: the CRM sends it when a lead becomes a santri or
// participant.
var MetaStandardEvents = []string{
	"AddPaymentInfo", "AddToCart", "AddToWishlist", "CompleteRegistration", "Contact", "CustomizeProduct",
	"Donate", "FindLocation", "InitiateCheckout", "Lead", "Schedule", "Search", "StartTrial",
	"SubmitApplication", "Subscribe", "ViewContent",
}

// MetaFunnelTrigger: CRM funnel events and the only trigger they may be
// chosen at.
var MetaFunnelTrigger = map[string]string{
	MetaEventLead: MetaTriggerSubmit, MetaEventCompleteRegistration: MetaTriggerSubmit,
	MetaEventInitiateCheckout: MetaTriggerSubmit, MetaEventAddPaymentInfo: MetaTriggerBukti,
}

var metaCustomEventPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{1,39}$`)

// DefaultMetaEvents is the behavior before events were configurable.
func DefaultMetaEvents() map[string][]string {
	return map[string][]string{
		MetaTriggerWA:       {},
		MetaTriggerLangkah1: {"DaftarLangkah2"},
		MetaTriggerSubmit:   {MetaEventLead, MetaEventCompleteRegistration, MetaEventInitiateCheckout},
		MetaTriggerBukti:    {MetaEventAddPaymentInfo},
	}
}

// IsStandardMetaEvent reports whether name is a Meta standard event.
func IsStandardMetaEvent(name string) bool {
	for _, e := range MetaStandardEvents {
		if e == name {
			return true
		}
	}
	return name == MetaEventPurchase
}

func canonicalMetaEvent(name string) string {
	for _, e := range MetaStandardEvents {
		if strings.EqualFold(e, name) {
			return e
		}
	}
	if strings.EqualFold(name, MetaEventPurchase) {
		return MetaEventPurchase
	}
	return name
}

var metaTriggerLabel = map[string]string{
	MetaTriggerWA: "Mode WA", MetaTriggerLangkah1: "Lolos langkah 1", MetaTriggerSubmit: "Submit form", MetaTriggerBukti: "Unggah bukti bayar",
}

// NormalizeMetaEvents validates an admin's choice; nil means defaults.
func NormalizeMetaEvents(in map[string][]string) (map[string][]string, error) {
	if in == nil {
		return DefaultMetaEvents(), nil
	}
	out := map[string][]string{}
	for _, t := range MetaTriggers {
		seen := map[string]bool{}
		list := []string{}
		for _, raw := range in[t] {
			name := canonicalMetaEvent(strings.TrimSpace(raw))
			if name == "" || seen[name] {
				continue
			}
			if name == MetaEventPurchase {
				return nil, fmt.Errorf("%s: Purchase dikirim otomatis saat lead menjadi santri/peserta, tidak bisa dipilih di link", metaTriggerLabel[t])
			}
			if ft, ok := MetaFunnelTrigger[name]; ok && ft != t {
				return nil, fmt.Errorf("%s: %s adalah event funnel CRM dan hanya bisa dipilih di %s agar tidak terhitung dua kali", metaTriggerLabel[t], name, metaTriggerLabel[ft])
			}
			if !IsStandardMetaEvent(name) && !metaCustomEventPattern.MatchString(name) {
				return nil, fmt.Errorf("%s: nama event %q tidak valid (huruf, angka, garis bawah; 2–40 karakter, diawali huruf)", metaTriggerLabel[t], raw)
			}
			seen[name] = true
			list = append(list, name)
		}
		if len(list) > metaEventsPerTrigger {
			return nil, fmt.Errorf("%s: maksimal %d event", metaTriggerLabel[t], metaEventsPerTrigger)
		}
		out[t] = list
	}
	return out, nil
}

// ParseMetaEvents reads a link's stored config; empty or invalid = defaults.
func ParseMetaEvents(raw string) map[string][]string {
	if strings.TrimSpace(raw) == "" {
		return DefaultMetaEvents()
	}
	var in map[string][]string
	if json.Unmarshal([]byte(raw), &in) != nil || in == nil {
		return DefaultMetaEvents()
	}
	out, err := NormalizeMetaEvents(in)
	if err != nil {
		return DefaultMetaEvents()
	}
	return out
}
