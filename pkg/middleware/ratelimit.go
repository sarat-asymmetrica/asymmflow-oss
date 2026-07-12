package middleware

import (
	"context"

	"golang.org/x/time/rate"
)

// RateLimiter protects backend methods from abuse
type RateLimiter struct {
	limiter *rate.Limiter
	// No mutex - rate.Limiter is already thread-safe using atomics
}

// NewRateLimiter creates a limiter with r requests per second and burst b
func NewRateLimiter(r float64, b int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(r), b),
	}
}

// Allow checks if the request is allowed
func (rl *RateLimiter) Allow() bool {
	return rl.limiter.Allow()
}

// Wait blocks until the request is allowed
// Returns error if context is cancelled or deadline exceeded
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}
