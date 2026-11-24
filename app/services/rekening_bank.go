package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/maulanashalihin/laju-go/app/models"
)

// Structured bank accounts for manual transfer (crm_settings.rekening_bank),
// so the payment page can copy the account number alone.

const maxRekeningBank = 5

var (
	reRekeningAllowed = regexp.MustCompile(`^[0-9][0-9 .\-]*[0-9]$`)
	reDigitRun        = regexp.MustCompile(`[0-9][0-9 .\-]{4,}[0-9]`)
	reAtasNama        = regexp.MustCompile(`(?i)^\s*(atas\s*nama|a\.?\s*n\.?)\s*[:.]?\s*`)
	reNoRekLabel      = regexp.MustCompile(`(?i)^\s*(no\.?\s*rek(ening)?|nomor\s*rek(ening)?|rek(ening)?)\s*[:.]?\s*`)
)

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normalizeRekening validates the admin's accounts; empty rows are dropped.
func normalizeRekening(in []models.RekeningBank) ([]models.RekeningBank, error) {
	out := []models.RekeningBank{}
	for i, r := range in {
		r.Bank = strings.Join(strings.Fields(r.Bank), " ")
		r.AtasNama = strings.Join(strings.Fields(r.AtasNama), " ")
		r.NoRekening = strings.TrimSpace(r.NoRekening)
		if r.Bank == "" && r.NoRekening == "" && r.AtasNama == "" {
			continue
		}
		n := i + 1
		if r.Bank == "" || utf8.RuneCountInString(r.Bank) > 40 {
			return nil, fmt.Errorf("rekening %d: nama bank wajib diisi (maks. 40 karakter)", n)
		}
		digits := onlyDigits(r.NoRekening)
		if !reRekeningAllowed.MatchString(r.NoRekening) || len(digits) < 5 || len(digits) > 30 {
			return nil, fmt.Errorf("rekening %d: nomor rekening hanya angka (5–30 digit)", n)
		}
		if r.AtasNama == "" || utf8.RuneCountInString(r.AtasNama) > 80 {
			return nil, fmt.Errorf("rekening %d: atas nama wajib diisi (maks. 80 karakter)", n)
		}
		out = append(out, r)
	}
	if len(out) > maxRekeningBank {
		return nil, errors.New("maksimal 5 rekening")
	}
	return out, nil
}

func parseRekeningJSON(raw string) []models.RekeningBank {
	var out []models.RekeningBank
	if strings.TrimSpace(raw) == "" || json.Unmarshal([]byte(raw), &out) != nil {
		return nil
	}
	clean, err := normalizeRekening(out)
	if err != nil {
		return nil
	}
	return clean
}

// RekeningDariTeks reads one account out of the old free-text bank info
// ("Bank Muamalat\nNo. Rek: 1110012352\nAtas Nama: Ruang Sanad Indonesia"),
// as a fallback until an admin saves structured accounts. nil when no
// account number is found.
func RekeningDariTeks(teks string) []models.RekeningBank {
	var r models.RekeningBank
	for _, line := range strings.Split(teks, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if loc := reAtasNama.FindStringIndex(line); loc != nil && r.AtasNama == "" {
			r.AtasNama = strings.TrimSpace(line[loc[1]:])
			continue
		}
		if m := reDigitRun.FindString(line); m != "" && r.NoRekening == "" {
			r.NoRekening = strings.TrimSpace(m)
			rest := strings.TrimSpace(strings.Replace(reNoRekLabel.ReplaceAllString(line, ""), m, "", 1))
			if r.Bank == "" && rest != "" && !strings.ContainsAny(rest, ":") {
				r.Bank = cleanBankName(rest)
			}
			continue
		}
		if r.Bank == "" {
			r.Bank = cleanBankName(line)
		}
	}
	if len(onlyDigits(r.NoRekening)) < 5 {
		return nil
	}
	if r.Bank == "" {
		r.Bank = "Bank"
	}
	return []models.RekeningBank{r}
}

// cleanBankName drops emoji/symbols around a bank name ("🏦 Bank Muamalat").
func cleanBankName(s string) string {
	s = strings.TrimFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	})
	return truncateRunes(strings.Join(strings.Fields(s), " "), 40)
}

// rekeningEfektif: the structured accounts, else the free-text fallback.
func rekeningEfektif(structured string, teks string) []models.RekeningBank {
	if r := parseRekeningJSON(structured); len(r) > 0 {
		return r
	}
	return RekeningDariTeks(teks)
}
