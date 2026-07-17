package cache

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func makeSessionData(id int64) CachedSessionData {
	return CachedSessionData{
		UserID:        id,
		Name:          fmt.Sprintf("User_%d", id),
		Email:         fmt.Sprintf("user%d@example.com", id),
		Avatar:        fmt.Sprintf("https://example.com/avatars/%d.jpg", id),
		EmailVerified: id%2 == 0,
		Role:          "user",
		CSRFToken:     fmt.Sprintf("csrf-token-%d", id),
		CSRFExpiry:    time.Now().Add(1 * time.Hour).Unix(),
		IP:            "192.168.1.100",
		UserAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
		ExpiresAt:     time.Now().Add(24 * time.Hour),
	}
}

func makeSessionID(i int) string {
	return fmt.Sprintf("bench-session-%016x", i)
}

func BenchmarkSessionCache_Set(b *testing.B) {
	c := NewSessionCache()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Set(makeSessionID(i), makeSessionData(int64(i)))
			i++
		}
	})
}

func BenchmarkSessionCache_Get_Hit(b *testing.B) {
	c := NewSessionCache()

	// Pre-populate
	for i := 0; i < b.N; i++ {
		c.Set(makeSessionID(i), makeSessionData(int64(i)))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(makeSessionID(i))
			i++
		}
	})
}

func BenchmarkSessionCache_Get_Miss(b *testing.B) {
	c := NewSessionCache()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get("nonexistent-" + makeSessionID(i))
			i++
		}
	})
}

func BenchmarkSessionCache_Set_Get_Invalidate(b *testing.B) {
	c := NewSessionCache()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			id := makeSessionID(i)
			data := makeSessionData(int64(i))
			c.Set(id, data)
			c.Get(id)
			c.Invalidate(id)
			i++
		}
	})
}

func BenchmarkSessionCache_Mixed_Workload(b *testing.B) {
	c := NewSessionCache()

	// Pre-populate
	const preload = 10_000
	for i := 0; i < preload; i++ {
		c.Set(makeSessionID(i), makeSessionData(int64(i)))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			idx := rng.Intn(preload * 2) // 50% hit, 50% miss
			id := makeSessionID(idx)
			op := rng.Intn(10)
			switch {
			case op < 6:
				c.Get(id)
			case op < 8:
				c.Set(id, makeSessionData(int64(idx)))
			default:
				c.Invalidate(id)
			}
		}
	})
}

func BenchmarkSessionCache_SustainedReads(b *testing.B) {
	c := NewSessionCache()

	const preload = 5000
	for i := 0; i < preload; i++ {
		c.Set(makeSessionID(i), makeSessionData(int64(i)))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			idx := rng.Intn(preload)
			c.Get(makeSessionID(idx))
		}
	})
}
