package services

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// LeadFormParse is the result of reading the registration form a lead sends
// back to the CS number (PRD 7.2). Only non-empty, normalized values are set.
type LeadFormParse struct {
	Nama           string
	Usia           int64
	Domisili       string
	JenisKelamin   string
	PaketKode      string
	WaktuBelajar   string
	RiwayatBelajar string
	BelajarDimana  string
	// Recognized lists every field whose label matched and value was non-empty.
	Recognized []string
	// Review lists fields that were present but could not be normalized.
	// They are left empty on purpose (never guessed).
	Review []string
}

// Filled reports whether the message counts as a filled form:
// at least 3 recognized fields and a non-empty nama.
func (p LeadFormParse) Filled() bool {
	return len(p.Recognized) >= 3 && p.Nama != ""
}

var leadFormLine = regexp.MustCompile(`^[\s*•\-\d.)]*([^:]+):\s*(.*)$`)
var leadFirstNumber = regexp.MustCompile(`\d+`)

func normalizeFormLabel(label string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(label) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func formFieldForLabel(label string) string {
	switch label {
	case "nama", "nama lengkap":
		return "nama"
	case "usia", "umur":
		return "usia"
	case "domisili", "alamat", "kota":
		return "domisili"
	case "ikhwan akhwat", "jenis kelamin", "gender":
		return "jenis_kelamin"
	case "pilihan kelas", "paket", "kelas":
		return "paket_kode"
	case "waktu belajar", "waktu":
		return "waktu_belajar"
	}
	if strings.Contains(label, "belajar dimana") || strings.Contains(label, "belajar di mana") {
		return "belajar_dimana"
	}
	if strings.HasPrefix(label, "apakah sebelumnya pernah belajar") || label == "pernah belajar" {
		return "riwayat_belajar"
	}
	return ""
}

func normalizeJenisKelamin(value string) (string, bool) {
	words := strings.Fields(normalizeFormLabel(value))
	male, female := false, false
	for _, w := range words {
		switch w {
		case "ikhwan", "laki", "pria", "l", "lakilaki":
			male = true
		case "akhwat", "perempuan", "wanita", "p":
			female = true
		}
	}
	if male == female {
		return "", false
	}
	if male {
		return "L", true
	}
	return "P", true
}

func normalizePaket(value string) (string, bool) {
	has1 := strings.Contains(value, "1")
	has2 := strings.Contains(value, "2")
	switch {
	case has1 && !has2:
		return "1x", true
	case has2 && !has1:
		return "2x", true
	}
	return "", false
}

func normalizeUsia(value string) (int64, bool) {
	match := leadFirstNumber.FindString(value)
	if match == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(match, 10, 64)
	if err != nil || n < 5 || n > 99 {
		return 0, false
	}
	return n, true
}

// ParseLeadForm reads one chat message line by line. It never guesses: a
// value that cannot be normalized is reported in Review and left empty.
func ParseLeadForm(body string) LeadFormParse {
	var out LeadFormParse
	seen := map[string]bool{}
	mark := func(field string, list *[]string) {
		for _, f := range *list {
			if f == field {
				return
			}
		}
		*list = append(*list, field)
	}
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		m := leadFormLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		field := formFieldForLabel(normalizeFormLabel(m[1]))
		value := strings.TrimSpace(m[2])
		if field == "" || value == "" || seen[field] {
			continue
		}
		seen[field] = true
		ok := true
		switch field {
		case "nama":
			out.Nama = value
		case "usia":
			out.Usia, ok = normalizeUsia(value)
		case "domisili":
			out.Domisili = value
		case "jenis_kelamin":
			out.JenisKelamin, ok = normalizeJenisKelamin(value)
		case "paket_kode":
			out.PaketKode, ok = normalizePaket(value)
		case "waktu_belajar":
			out.WaktuBelajar = value
		case "riwayat_belajar":
			out.RiwayatBelajar = value
		case "belajar_dimana":
			out.BelajarDimana = value
		}
		mark(field, &out.Recognized)
		if !ok {
			mark(field, &out.Review)
		}
	}
	return out
}

var leadStopPhrases = []string{
	"stop", "berhenti", "jangan hubungi", "jangan chat lagi", "unsubscribe",
	"tidak usah dihubungi", "jangan kirim pesan lagi",
}

func normalizeChatText(body string) string {
	return strings.Join(strings.Fields(strings.ToLower(body)), " ")
}

// IsLeadStopRequest reports whether a message equals or starts with one of
// the opt-out phrases (PRD 7.3). "belum dulu"/"nanti saja" are not opt-outs.
func IsLeadStopRequest(body string) bool {
	text := normalizeChatText(body)
	for _, phrase := range leadStopPhrases {
		if text == phrase {
			return true
		}
		if strings.HasPrefix(text, phrase) {
			next := []rune(text[len(phrase):])[0]
			if !unicode.IsLetter(next) && !unicode.IsDigit(next) {
				return true
			}
		}
	}
	return false
}

// IsLeadFollowupPermission detects the d14 permission-reset reply "boleh kabari".
func IsLeadFollowupPermission(body string) bool {
	return strings.Contains(normalizeChatText(body), "boleh kabari")
}

// NormalizeLeadPhone returns international digits (62…) or "" when unknown.
func NormalizeLeadPhone(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	switch {
	case digits == "":
		return ""
	case strings.HasPrefix(digits, "0"):
		digits = "62" + strings.TrimLeft(digits, "0")
	case strings.HasPrefix(digits, "8"):
		digits = "62" + digits
	}
	if len(digits) < 8 || len(digits) > 16 {
		return ""
	}
	return digits
}

// phoneFromChatID derives the phone number from a classic WA JID. LID chats
// (…@lid) carry no phone number.
func phoneFromChatID(chatID string) string {
	user, server, ok := strings.Cut(chatID, "@")
	if !ok || server != "s.whatsapp.net" {
		return ""
	}
	user, _, _ = strings.Cut(user, ":")
	return NormalizeLeadPhone(user)
}

// ParseRupiah turns "Rp149.000" / "Rp 149.000,00" into 149000. Returns 0 when
// no amount can be read.
func ParseRupiah(text string) int64 {
	text, _, _ = strings.Cut(text, ",")
	var b strings.Builder
	for _, r := range text {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	n, err := strconv.ParseInt(b.String(), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
