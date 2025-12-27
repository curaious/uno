package virtual_key_middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/praveen001/uno/pkg/gateway"
	"github.com/redis/go-redis/v9"
)

// RedisRateLimiterStorage implements RateLimiterStorage using Redis for distributed rate limiting.
// This allows rate limiting to work across multiple gateway instances.
//
// Usage example:
//
//	redisClient := redis.NewClient(&redis.Options{
//		Addr:     "localhost:6379",
//		Password: "", // no password set
//		DB:       0,  // use default DB
//	})
//
//	// Verify connection
//	if err := redisClient.Ping(context.Background()).Err(); err != nil {
//		log.Fatal("Failed to connect to Redis:", err)
//	}
//
//	// Create Redis rate limiter storage
//	rateLimiterStorage := NewRedisRateLimiterStorage(redisClient, "rate_limit:")
//
//	// Use it with VirtualKeyMiddleware
//	middleware := NewVirtualKeyMiddleware(configStore, rateLimiterStorage)
//
// The implementation uses Lua scripts for atomic operations, ensuring thread-safety
// and consistency across distributed systems. Token buckets are stored as Redis hashes
// with automatic expiration based on the rate limit window duration.
type RedisRateLimiterStorage struct {
	client    *redis.Client
	keyPrefix string
}

// NewRedisRateLimiterStorage creates a new Redis-based rate limiter storage.
// The client should be a configured Redis client that's ready to use.
// keyPrefix is optional and defaults to "rate_limit:" if empty.
func NewRedisRateLimiterStorage(client *redis.Client, keyPrefix string) *RedisRateLimiterStorage {
	if keyPrefix == "" {
		keyPrefix = "rate_limit:"
	}

	return &RedisRateLimiterStorage{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// Allow checks if a request is allowed and consumes a token if available.
// Uses a Lua script for atomic token bucket operations in Redis.
func (r *RedisRateLimiterStorage) Allow(ctx context.Context, virtualKeyID string, rateLimit gateway.RateLimit) (bool, error) {
	// Parse the rate limit unit to get duration
	duration, err := parseRateLimitUnit(rateLimit.Unit)
	if err != nil {
		return false, fmt.Errorf("invalid rate limit unit: %w", err)
	}

	bucketKey := r.getBucketKey(virtualKeyID, rateLimit.Unit)
	capacity := float64(rateLimit.Limit)
	refillRate := capacity / duration.Seconds()
	now := time.Now().UnixNano()

	// Lua script for atomic token bucket consumption
	// This ensures thread-safety and atomicity across distributed systems
	// The script atomically:
	// 1. Gets or initializes bucket state
	// 2. Refills tokens based on elapsed time
	// 3. Consumes a token if available
	// 4. Updates bucket state and expiration
	script := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local refillRate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local tokensToConsume = tonumber(ARGV[4])
		local windowSeconds = tonumber(ARGV[5])
		
		-- Get current bucket state
		local bucketData = redis.call('HMGET', key, 'tokens', 'lastRefill')
		local tokensStr = bucketData[1]
		local lastRefillStr = bucketData[2]
		
		-- Initialize or parse current state
		local tokens
		local lastRefill
		if tokensStr == false or tokensStr == nil then
			-- New bucket, start with full capacity
			tokens = capacity
			lastRefill = now
		else
			tokens = tonumber(tokensStr)
			if tokens == nil then
				tokens = capacity
			end
			lastRefill = tonumber(lastRefillStr)
			if lastRefill == nil then
				lastRefill = now
			end
		end
		
		-- Calculate elapsed time in seconds (nanoseconds to seconds)
		local elapsed = (now - lastRefill) / 1000000000
		
		-- Refill tokens based on elapsed time
		if elapsed > 0 then
			local tokensToAdd = elapsed * refillRate
			tokens = math.min(capacity, tokens + tokensToAdd)
		end
		
		-- Check if we have enough tokens
		if tokens >= tokensToConsume then
			-- Consume tokens
			tokens = tokens - tokensToConsume
			
			-- Update bucket state using HSET (modern Redis command)
			redis.call('HSET', key, 'tokens', tostring(tokens), 'lastRefill', tostring(now))
			
			-- Set expiration to window duration + 10% buffer to handle edge cases
			redis.call('EXPIRE', key, math.ceil(windowSeconds * 1.1))
			
			return 1  -- Allowed
		else
			-- Update lastRefill even if we can't consume (for accurate refill calculation)
			redis.call('HSET', key, 'tokens', tostring(tokens), 'lastRefill', tostring(now))
			redis.call('EXPIRE', key, math.ceil(windowSeconds * 1.1))
			
			return 0  -- Rate limited
		end
	`

	result, err := r.client.Eval(ctx, script, []string{bucketKey},
		capacity,
		refillRate,
		now,
		1.0, // tokens to consume
		duration.Seconds(),
	).Result()

	if err != nil {
		return false, fmt.Errorf("redis rate limit check failed: %w", err)
	}

	// Result is 1 if allowed, 0 if rate limited
	allowed := result.(int64) == 1
	return allowed, nil
}

// getBucketKey generates a unique key for a virtual key + rate limit unit combination.
func (r *RedisRateLimiterStorage) getBucketKey(virtualKeyID string, unit string) string {
	return fmt.Sprintf("%s%s:%s", r.keyPrefix, virtualKeyID, unit)
}

// Ping checks if the Redis connection is healthy.
func (r *RedisRateLimiterStorage) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the Redis client connection.
// Call this when shutting down to properly clean up resources.
func (r *RedisRateLimiterStorage) Close() error {
	return r.client.Close()
}
