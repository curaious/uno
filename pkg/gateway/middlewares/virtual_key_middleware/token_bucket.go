package virtual_key_middleware

import (
	"fmt"
	"sync"
	"time"
)

// tokenBucket represents a single token bucket for rate limiting.
type tokenBucket struct {
	mu             sync.Mutex
	tokens         float64
	lastRefill     time.Time
	capacity       float64
	refillRate     float64 // tokens per second
	windowDuration time.Duration
}

// consume attempts to consume the requested number of tokens.
// Returns true if tokens were available and consumed, false otherwise.
func (tb *tokenBucket) consume(tokens float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()

	// Refill tokens based on elapsed time
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokensToAdd := elapsed * tb.refillRate
	tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
	tb.lastRefill = now

	// Check if we have enough tokens
	if tb.tokens >= tokens {
		tb.tokens -= tokens
		return true
	}

	return false
}

// parseRateLimitUnit converts a rate limit unit string to a time.Duration.
// Supported units: 1min, 1h, 6h, 12h, 1d, 1w, 1mo
func parseRateLimitUnit(unit string) (time.Duration, error) {
	switch unit {
	case "1min":
		return time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "12h":
		return 12 * time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	case "1w":
		return 7 * 24 * time.Hour, nil
	case "1mo":
		return 30 * 24 * time.Hour, nil // Approximate month as 30 days
	default:
		return 0, fmt.Errorf("unsupported rate limit unit: %s", unit)
	}
}
