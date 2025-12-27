package virtual_key_middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/praveen001/uno/pkg/gateway"
)

// InMemoryRateLimiterStorage implements RateLimiterStorage using in-memory token buckets.
// This can be easily replaced with a Redis-based implementation for distributed rate limiting.
type InMemoryRateLimiterStorage struct {
	mu          sync.RWMutex
	buckets     map[string]*tokenBucket
	cleanup     *time.Ticker
	stopCleanup chan struct{}
}

// NewInMemoryRateLimiterStorage creates a new in-memory rate limiter storage.
// It includes a background cleanup goroutine to remove unused buckets.
func NewInMemoryRateLimiterStorage() *InMemoryRateLimiterStorage {
	storage := &InMemoryRateLimiterStorage{
		buckets:     make(map[string]*tokenBucket),
		cleanup:     time.NewTicker(5 * time.Minute),
		stopCleanup: make(chan struct{}),
	}

	// Start background cleanup goroutine
	go storage.cleanupUnusedBuckets()

	return storage
}

// Stop stops the background cleanup goroutine. Call this when shutting down.
func (s *InMemoryRateLimiterStorage) Stop() {
	s.cleanup.Stop()
	close(s.stopCleanup)
}

// Allow checks if a request is allowed and consumes a token if available.
func (s *InMemoryRateLimiterStorage) Allow(ctx context.Context, virtualKeyID string, rateLimit gateway.RateLimit) (bool, error) {
	bucketKey := s.getBucketKey(virtualKeyID, rateLimit.Unit)

	// Parse the rate limit unit to get duration
	duration, err := parseRateLimitUnit(rateLimit.Unit)
	if err != nil {
		return false, fmt.Errorf("invalid rate limit unit: %w", err)
	}

	// Get or create bucket
	s.mu.Lock()
	bucket, exists := s.buckets[bucketKey]
	if !exists {
		bucket = s.newTokenBucket(float64(rateLimit.Limit), duration)
		s.buckets[bucketKey] = bucket
	}
	s.mu.Unlock()

	// Try to consume a token
	return bucket.consume(1), nil
}

// getBucketKey generates a unique key for a virtual key + rate limit unit combination.
func (s *InMemoryRateLimiterStorage) getBucketKey(virtualKeyID string, unit string) string {
	return fmt.Sprintf("%s:%s", virtualKeyID, unit)
}

// newTokenBucket creates a new token bucket with the given capacity and window duration.
func (s *InMemoryRateLimiterStorage) newTokenBucket(capacity float64, windowDuration time.Duration) *tokenBucket {
	now := time.Now()
	// Refill rate is capacity / window duration in seconds
	refillRate := capacity / windowDuration.Seconds()

	return &tokenBucket{
		tokens:         capacity,
		lastRefill:     now,
		capacity:       capacity,
		refillRate:     refillRate,
		windowDuration: windowDuration,
	}
}

// cleanupUnusedBuckets periodically removes buckets that haven't been used recently.
func (s *InMemoryRateLimiterStorage) cleanupUnusedBuckets() {
	for {
		select {
		case <-s.cleanup.C:
			s.mu.Lock()
			now := time.Now()
			for key, bucket := range s.buckets {
				bucket.mu.Lock()
				// Remove buckets that haven't been used in 2x their window duration
				unusedDuration := now.Sub(bucket.lastRefill)
				if unusedDuration > bucket.windowDuration*2 {
					delete(s.buckets, key)
				}
				bucket.mu.Unlock()
			}
			s.mu.Unlock()
		case <-s.stopCleanup:
			return
		}
	}
}
