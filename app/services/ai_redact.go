package services

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
)

// Contact data never leaves the server through the AI agent. Tool results use
// DTOs without contact fields; this file is the second line of defence:
//  1. keys that hold contact data are dropped wherever they appear;
//  2. phone numbers, e-mail addresses and WhatsApp links inside free text
//     (riayah notes, admin notes, ...) are masked.

var aiForbiddenKeys = map[string]bool{
	"no_wa":             true,
	"nowa":              true,
	"no_hp":             true,
	"telepon":           true,
	"email":             true,
	"linked_user_email": true,
	"voice_note_url":    true,
	"keterangan_vn":     true,
	"domisili":          true,
	"phone":             true,
	"wa_chat_id":        true,
	"chat_id":           true,
	"wa_link":           true,
	"nama_pembeli":      true,
	"foto":              true,
	"avatar":            true,
	"password":          true,
}

const (
	aiMaskPhone = "[nomor disembunyikan]"
	aiMaskEmail = "[email disembunyikan]"
	aiMaskLink  = "[tautan WhatsApp disembunyikan]"
)

var (
	// wa.me/628..., api.whatsapp.com/send?phone=628..., chat.whatsapp.com/...
	aiWALinkRe = regexp.MustCompile(`(?i)(?:https?://)?(?:wa\.me|api\.whatsapp\.com|chat\.whatsapp\.com|web\.whatsapp\.com)/\S*`)
	aiEmailRe  = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	// Indonesian mobile numbers: 08xx / +628xx / 628xx, 9–14 digits, with
	// optional spaces, dots or dashes between digit groups.
	aiPhoneRe = regexp.MustCompile(`(?:\+?62|\b0)[\s.\-]?8\d(?:[\s.\-]?\d){6,11}\b`)
)

// RedactContactText masks phone numbers, e-mail addresses and WhatsApp links.
func RedactContactText(s string) string {
	if s == "" {
		return s
	}
	s = aiWALinkRe.ReplaceAllString(s, aiMaskLink)
	s = aiEmailRe.ReplaceAllString(s, aiMaskEmail)
	s = aiPhoneRe.ReplaceAllString(s, aiMaskPhone)
	return s
}

func aiScrub(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if aiForbiddenKeys[strings.ToLower(k)] {
				delete(t, k)
				continue
			}
			t[k] = aiScrub(val)
		}
		return t
	case []any:
		for i := range t {
			t[i] = aiScrub(t[i])
		}
		return t
	case string:
		return RedactContactText(t)
	default:
		return v
	}
}

// SanitizeForAI converts a tool result to JSON with contact data removed.
func SanitizeForAI(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	// UseNumber keeps large integers (nominal rupiah) exact.
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var generic any
	if err := dec.Decode(&generic); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // keep "<" and ">" readable for the AI
	if err := enc.Encode(aiScrub(generic)); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}
