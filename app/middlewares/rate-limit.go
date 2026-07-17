package middlewares

import (
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RateLimiterConfig holds the rate limiter configuration
type RateLimiterConfig struct {
	MaxRequests int                     // Maximum number of requests allowed
	Window      time.Duration           // Time window for the rate limit
	Message     string                  // Message to return when rate limit is exceeded
	StatusCode  int                     // HTTP status code to return
	SkipFailed  bool                    // Whether to skip failed requests (non-2xx)
	CustomKeyFn func(*fiber.Ctx) string // Custom key function (optional)
}

// RateLimiter implements a simple in-memory rate limiter
type RateLimiter struct {
	mu      sync.RWMutex
	entries map[string]*rateLimitEntry
	config  RateLimiterConfig
}

type rateLimitEntry struct {
	count     int
	resetTime time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	if config.Message == "" {
		config.Message = "Too many requests, please try again later"
	}
	if config.StatusCode == 0 {
		config.StatusCode = fiber.StatusTooManyRequests
	}

	limiter := &RateLimiter{
		entries: make(map[string]*rateLimitEntry),
		config:  config,
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// Limit returns a middleware function that applies rate limiting
func (rl *RateLimiter) Limit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get client identifier (IP address by default)
		key := rl.getClientKey(c)

		rl.mu.Lock()
		entry, exists := rl.entries[key]
		now := time.Now()

		if !exists || now.After(entry.resetTime) {
			// Create new entry or reset expired one
			rl.entries[key] = &rateLimitEntry{
				count:     1,
				resetTime: now.Add(rl.config.Window),
			}
			rl.mu.Unlock()
			return c.Next()
		}

		if entry.count >= rl.config.MaxRequests {
			// Rate limit exceeded
			rl.mu.Unlock()
			c.Set("Retry-After", strconv.Itoa(int(entry.resetTime.Sub(now).Seconds())))
			return c.Status(rl.config.StatusCode).JSON(fiber.Map{
				"error": rl.config.Message,
			})
		}

		// Increment counter
		entry.count++
		rl.mu.Unlock()

		return c.Next()
	}
}

// getClientKey returns the client identifier
func (rl *RateLimiter) getClientKey(c *fiber.Ctx) string {
	if rl.config.CustomKeyFn != nil {
		return rl.config.CustomKeyFn(c)
	}
	// Default: use IP address
	return c.IP()
}

// cleanup removes expired entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, entry := range rl.entries {
			if now.After(entry.resetTime) {
				delete(rl.entries, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Reset clears all rate limit entries (useful for testing)
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.entries = make(map[string]*rateLimitEntry)
}

// Predefined rate limiters for common use cases
var (
	// AuthRateLimit: 5 requests per 15 minutes (login, register)
	AuthRateLimit = NewRateLimiter(RateLimiterConfig{
		MaxRequests: 5,
		Window:      15 * time.Minute,
		Message:     "Too many authentication attempts, please try again in 15 minutes",
	})

	// PasswordResetRateLimit: 3 requests per hour
	PasswordResetRateLimit = NewRateLimiter(RateLimiterConfig{
		MaxRequests: 3,
		Window:      time.Hour,
		Message:     "Too many password reset requests, please try again in an hour",
	})

	// APIRateLimit: 100 requests per 15 minutes
	APIRateLimit = NewRateLimiter(RateLimiterConfig{
		MaxRequests: 100,
		Window:      15 * time.Minute,
		Message:     "Too many requests, please slow down",
	})

	// UploadRateLimit: 50 requests per hour
	UploadRateLimit = NewRateLimiter(RateLimiterConfig{
		MaxRequests: 50,
		Window:      time.Hour,
		Message:     "Too many upload requests, please try again later",
	})
)
