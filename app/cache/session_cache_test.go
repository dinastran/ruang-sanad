package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSessionCache(t *testing.T) *SessionCache {
	t.Helper()
	return NewSessionCache()
}

func TestSessionCacheGetSet(t *testing.T) {
	c := newTestSessionCache(t)

	sessionID := "test-session-1"
	data := CachedSessionData{
		UserID:    1,
		Email:     "alice@example.com",
		Role:      "user",
		CSRFToken: "csrf-abc",
		IP:        "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	c.Set(sessionID, data)

	got, ok := c.Get(sessionID)
	require.True(t, ok)
	assert.Equal(t, int64(1), got.UserID)
	assert.Equal(t, "alice@example.com", got.Email)
	assert.Equal(t, "user", got.Role)
	assert.Equal(t, "csrf-abc", got.CSRFToken)
	assert.Equal(t, "192.168.1.1", got.IP)
}

func TestSessionCacheGetMiss(t *testing.T) {
	c := newTestSessionCache(t)

	_, ok := c.Get("nonexistent-session")
	assert.False(t, ok)
}

func TestSessionCacheInvalidate(t *testing.T) {
	c := newTestSessionCache(t)

	sessionID := "test-session-2"
	data := CachedSessionData{
		UserID:    2,
		Email:     "bob@example.com",
		Role:      "user",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	c.Set(sessionID, data)
	_, ok := c.Get(sessionID)
	assert.True(t, ok)

	c.Invalidate(sessionID)
	_, ok = c.Get(sessionID)
	assert.False(t, ok)
}

func TestSessionCacheExpiry(t *testing.T) {
	c := newTestSessionCache(t)

	sessionID := "test-session-3"
	data := CachedSessionData{
		UserID:    3,
		Email:     "carol@example.com",
		Role:      "user",
		ExpiresAt: time.Now().Add(50 * time.Millisecond),
	}
	c.Set(sessionID, data)
	_, ok := c.Get(sessionID)
	assert.True(t, ok)

	time.Sleep(60 * time.Millisecond)
	_, ok = c.Get(sessionID)
	assert.False(t, ok)
}

func TestSessionCacheClear(t *testing.T) {
	c := newTestSessionCache(t)

	c.Set("sid-1", CachedSessionData{
		UserID:    1,
		Email:     "one@example.com",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})
	c.Set("sid-2", CachedSessionData{
		UserID:    2,
		Email:     "two@example.com",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	_, ok1 := c.Get("sid-1")
	_, ok2 := c.Get("sid-2")
	assert.True(t, ok1)
	assert.True(t, ok2)

	c.Clear()

	_, ok1 = c.Get("sid-1")
	_, ok2 = c.Get("sid-2")
	assert.False(t, ok1)
	assert.False(t, ok2)
}

func TestSessionCacheOverwrite(t *testing.T) {
	c := newTestSessionCache(t)

	sessionID := "test-session-4"
	c.Set(sessionID, CachedSessionData{
		UserID:    1,
		Email:     "original@example.com",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})
	c.Set(sessionID, CachedSessionData{
		UserID:    1,
		Email:     "updated@example.com",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	got, ok := c.Get(sessionID)
	require.True(t, ok)
	assert.Equal(t, "updated@example.com", got.Email)
}

func TestSessionCacheTTLBuffer(t *testing.T) {
	c := newTestSessionCache(t)

	sessionID := "test-session-5"
	data := CachedSessionData{
		UserID:    5,
		Email:     "buffer@example.com",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	c.Set(sessionID, data)

	got, ok := c.Get(sessionID)
	require.True(t, ok)
	assert.Equal(t, int64(5), got.UserID)
}

func TestSessionCacheExpiredSessionAtSet(t *testing.T) {
	c := newTestSessionCache(t)

	sessionID := "test-session-6"
	data := CachedSessionData{
		UserID:    6,
		Email:     "expired@example.com",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // already expired
	}
	c.Set(sessionID, data)

	_, ok := c.Get(sessionID)
	assert.False(t, ok, "session expired in the past should not be retrievable")
}
