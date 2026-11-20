package services

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormToken(t *testing.T) {
	clock := time.Date(2026, 11, 19, 3, 0, 0, 0, time.UTC)
	s := NewRegistrationService(nil, nil, nil, []byte("secret"))
	s.now = func() time.Time { return clock }
	tok := s.FormToken("tahsin")

	at := func(d time.Duration) { s.now = func() time.Time { return clock.Add(d) } }
	at(5 * time.Second)
	require.True(t, errors.Is(s.CheckFormToken("lain", tok), ErrFormTidakValid), "slug lain")
	at(2 * time.Second)
	require.True(t, errors.Is(s.CheckFormToken("tahsin", tok), ErrFormTerlaluCepat))
	at(5 * time.Second)
	require.NoError(t, s.CheckFormToken("tahsin", tok))
	at(2*time.Hour + time.Second)
	require.True(t, errors.Is(s.CheckFormToken("tahsin", tok), ErrFormKedaluwarsa))
	at(5 * time.Second)
	tampered := []byte(tok)
	tampered[len(tampered)-1] ^= 1
	require.True(t, errors.Is(s.CheckFormToken("tahsin", string(tampered)), ErrFormTidakValid))
	require.True(t, errors.Is(s.CheckFormToken("tahsin", ""), ErrFormTidakValid))
}

func TestAllowPhone(t *testing.T) {
	clock := time.Date(2026, 11, 19, 3, 0, 0, 0, time.UTC)
	s := NewRegistrationService(nil, nil, nil, []byte("secret"))
	s.now = func() time.Time { return clock }
	for range 3 {
		require.True(t, s.allowPhone("6281234567890"))
	}
	require.False(t, s.allowPhone("6281234567890"), "ke-4 dalam 1 jam")
	require.True(t, s.allowPhone("6289999999999"), "nomor lain tidak terpengaruh")
	clock = clock.Add(61 * time.Minute)
	require.True(t, s.allowPhone("6281234567890"))
}
