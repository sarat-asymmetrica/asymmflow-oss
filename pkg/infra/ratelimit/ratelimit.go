// Package ratelimit is a keyed token-bucket rate limiter.
//
// Promoted verbatim from package main's security_enhancements.go (Wave 4
// B.2 convergence): the bucket arithmetic is unchanged; only the home moved
// so verticals other than trading can rate-limit without importing the app
// shell.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter tracks one token bucket per key.
type Limiter struct {
	limiters sync.Map // map[string]*bucket
}

type bucket struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// New creates a rate limiter.
func New() *Limiter {
	return &Limiter{}
}

// Allow reports whether one more action under key is within the limit of
// maxTokens per bucket, refilling one token every refillRate. The first call
// for a key starts with a full bucket.
func (rl *Limiter) Allow(key string, maxTokens int, refillRate time.Duration) bool {
	val, _ := rl.limiters.LoadOrStore(key, &bucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	})

	b := val.(*bucket)
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	tokensToAdd := int(elapsed / b.refillRate)

	if tokensToAdd > 0 {
		b.tokens += tokensToAdd
		if b.tokens > b.maxTokens {
			b.tokens = b.maxTokens
		}
		b.lastRefill = now
	}

	if b.tokens > 0 {
		b.tokens--
		return true
	}
	return false
}
