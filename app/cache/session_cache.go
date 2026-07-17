package cache

import (
	"sync"
	"time"
)

// CachedSessionData holds the session fields we cache.
type CachedSessionData struct {
	UserID        int64     `json:"uid"`
	Name          string    `json:"name,omitempty"`
	Email         string    `json:"email"`
	Avatar        string    `json:"avatar,omitempty"`
	EmailVerified bool      `json:"ev,omitempty"`
	Role          string    `json:"role"`
	CSRFToken     string    `json:"csrf,omitempty"`
	CSRFExpiry    int64     `json:"csrf_exp,omitempty"`
	IP            string    `json:"ip,omitempty"`
	UserAgent     string    `json:"ua,omitempty"`
	ExpiresAt     time.Time `json:"exp"`
}

// cacheEntry holds a cached session with its metadata.
type cacheEntry struct {
	data CachedSessionData
}

// SessionCache provides in-memory session caching.
// Thread-safe via sync.RWMutex.
type SessionCache struct {
	mu   sync.RWMutex
	data map[string]cacheEntry
}

// NewSessionCache creates a new in-memory session cache.
func NewSessionCache() *SessionCache {
	return &SessionCache{
		data: make(map[string]cacheEntry),
	}
}

// Get retrieves a cached session. Returns nil + false if not found or expired.
func (c *SessionCache) Get(sessionID string) (*CachedSessionData, bool) {
	c.mu.RLock()
	entry, ok := c.data[sessionID]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Check session expiry
	if time.Now().After(entry.data.ExpiresAt) {
		c.Invalidate(sessionID)
		return nil, false
	}

	// Return a copy to avoid data races
	cp := entry.data
	return &cp, true
}

// Set stores a session in the in-memory cache.
func (c *SessionCache) Set(sessionID string, data CachedSessionData) {
	c.mu.Lock()
	c.data[sessionID] = cacheEntry{data: data}
	c.mu.Unlock()
}

// Invalidate removes a session from cache.
func (c *SessionCache) Invalidate(sessionID string) {
	c.mu.Lock()
	delete(c.data, sessionID)
	c.mu.Unlock()
}

// Clear removes all cached sessions.
func (c *SessionCache) Clear() {
	c.mu.Lock()
	c.data = make(map[string]cacheEntry)
	c.mu.Unlock()
}
