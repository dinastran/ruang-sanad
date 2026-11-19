package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

var (
	ErrFormTerlaluCepat = errors.New("form dikirim terlalu cepat, silakan coba lagi")
	ErrFormKedaluwarsa  = errors.New("halaman sudah terlalu lama dibuka, muat ulang halaman lalu kirim lagi")
	ErrFormTidakValid   = errors.New("form tidak valid, muat ulang halaman lalu kirim lagi")
)

const (
	formMinAge        = 3 * time.Second
	formMaxAge        = 2 * time.Hour
	phoneSubmitLimit  = 3
	phoneSubmitWindow = time.Hour
)

func (s *RegistrationService) formSig(slug, ts string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte("daftar|" + slug + "|" + ts))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// FormToken signs the render time of a registration form, so a submit can
// be rejected when it comes too fast (bots) or too late (stale tab).
func (s *RegistrationService) FormToken(slug string) string {
	ts := strconv.FormatInt(s.now().Unix(), 10)
	return ts + "." + s.formSig(slug, ts)
}

func (s *RegistrationService) CheckFormToken(slug, token string) error {
	ts, sig, ok := strings.Cut(token, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.formSig(slug, ts))) {
		return ErrFormTidakValid
	}
	unix, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return ErrFormTidakValid
	}
	age := s.now().Sub(time.Unix(unix, 0))
	switch {
	case age < formMinAge:
		return ErrFormTerlaluCepat
	case age > formMaxAge:
		return ErrFormKedaluwarsa
	}
	return nil
}

// allowPhone limits submissions per WhatsApp number (in memory).
func (s *RegistrationService) allowPhone(phone string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	cutoff := now.Add(-phoneSubmitWindow)
	for p, hits := range s.phoneHits {
		kept := hits[:0]
		for _, h := range hits {
			if h.After(cutoff) {
				kept = append(kept, h)
			}
		}
		if len(kept) == 0 {
			delete(s.phoneHits, p)
		} else {
			s.phoneHits[p] = kept
		}
	}
	if len(s.phoneHits[phone]) >= phoneSubmitLimit {
		return false
	}
	s.phoneHits[phone] = append(s.phoneHits[phone], now)
	return true
}
