// Package guard holds the fraud logic: pure, in-memory, concurrency-safe
// adaptive statistics. Every decision is an explainable number.
package guard

import (
	"sync"
	"time"
)

// tokenBucket tracks available tokens and the timestamp of the last replenishemnt.
type tokenBucket struct {
	tokens float64
	last   time.Time
}

// Velocity enforces a per-key burst limit (key = phone number or IP)
// It catches OTP -bombing (many sends in a short window).
type Velocity struct {
	mu           sync.Mutex
	capacity     float64
	refillPerSec float64
	buckets      map[string]*tokenBucket
}

// NewVelocity initializes and returns a new Velocity rate limiter instance.
func NewVelocity(capacity, refillPerSec float64) *Velocity {
	return &Velocity{
		capacity:     capacity,
		refillPerSec: refillPerSec,
		buckets:      make(map[string]*tokenBucket),
	}
}

// Allow reports whether an event for key is permitted at time now, consuming a token
// when it is.
func (v *Velocity) Allow(key string, now time.Time) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	b, ok := v.buckets[key]
	if !ok {
		b = &tokenBucket{tokens: v.capacity, last: now}
		v.buckets[key] = b
	} else {
		elapsed := now.Sub(b.last).Seconds()
		if elapsed > 0 {
			b.tokens = min(v.capacity, b.tokens+elapsed*v.refillPerSec)
			b.last = now
		}
	}

	if b.tokens >= 1 {
		b.tokens--
		return true
	}

	return false
}

// min returns the lesser of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}

	return b
}
