package auth

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// redisRateLimiter implements RateLimiter using Redis for distributed
// rate limiting with exponential backoff and account lockout.
type redisRateLimiter struct {
	rdb *redis.Client
}

// NewRedisRateLimiter returns a RateLimiter backed by Redis.
// On Redis errors, it fails open (allows requests) to preserve availability.
func NewRedisRateLimiter(rdb *redis.Client) RateLimiter {
	return &redisRateLimiter{rdb: rdb}
}

// Allow checks whether a request identified by key is permitted under the
// given rate limit configuration. It does NOT increment the counter — call
// RecordFailure separately on authentication failure.
func (rl *redisRateLimiter) Allow(ctx context.Context, key string, cfg RateLimitConfig) (*RateLimitResult, error) {
	cfg.Defaults()

	lockedKey := fmt.Sprintf("locked:%s", key)
	rateKey := fmt.Sprintf("rate:%s", key)

	// Step 1: Check lockout.
	locked, err := rl.rdb.Exists(ctx, lockedKey).Result()
	if err != nil {
		log.Printf("auth: redis Allow lockout check warning: %v", err)
		return &RateLimitResult{Allowed: true, Remaining: cfg.MaxAttempts}, nil
	}
	if locked > 0 {
		ttl, err := rl.rdb.TTL(ctx, lockedKey).Result()
		if err != nil {
			log.Printf("auth: redis Allow lockout TTL warning: %v", err)
			ttl = cfg.LockoutDuration
		}
		retryAfter := int64(ttl.Seconds())
		if retryAfter <= 0 {
			retryAfter = 1
		}
		return &RateLimitResult{
			Allowed:    false,
			Remaining:  0,
			Locked:     true,
			RetryAfter: retryAfter,
		}, nil
	}

	// Step 2: Get current attempt count.
	val, err := rl.rdb.Get(ctx, rateKey).Result()
	if err == redis.Nil {
		val = "0"
	} else if err != nil {
		log.Printf("auth: redis Allow counter read warning: %v", err)
		return &RateLimitResult{Allowed: true, Remaining: cfg.MaxAttempts}, nil
	}

	count, _ := strconv.Atoi(val)

	// Step 3: Over limit.
	if count >= cfg.MaxAttempts {
		backoff := computeBackoff(count, cfg.BackoffBase, cfg.BackoffMax)
		return &RateLimitResult{
			Allowed:        false,
			Remaining:      0,
			Locked:         false,
			RetryAfter:     int64(backoff),
			BackoffSeconds: backoff,
		}, nil
	}

	// Step 4: Under limit — allowed.
	remaining := cfg.MaxAttempts - count - 1
	backoff := 0
	if count >= 2 {
		backoff = computeBackoff(count, cfg.BackoffBase, cfg.BackoffMax)
	}

	return &RateLimitResult{
		Allowed:        true,
		Remaining:      remaining,
		BackoffSeconds: backoff,
	}, nil
}

// RecordFailure increments the failure counter for key and triggers lockout
// if the threshold is reached.
func (rl *redisRateLimiter) RecordFailure(ctx context.Context, key string, cfg RateLimitConfig) error {
	cfg.Defaults()

	rateKey := fmt.Sprintf("rate:%s", key)

	// INCR atomically increments (creates with value 1 if absent).
	count, err := rl.rdb.Incr(ctx, rateKey).Result()
	if err != nil {
		log.Printf("auth: redis RecordFailure INCR warning: %v", err)
		return nil
	}

	// Set expiry on first increment.
	if count == 1 {
		if err := rl.rdb.Expire(ctx, rateKey, cfg.Window).Err(); err != nil {
			log.Printf("auth: redis RecordFailure Expire warning: %v", err)
		}
	}

	// Lockout check.
	if int(count) >= cfg.LockoutThreshold {
		lockedKey := fmt.Sprintf("locked:%s", key)
		if err := rl.rdb.Set(ctx, lockedKey, "1", cfg.LockoutDuration).Err(); err != nil {
			log.Printf("auth: redis RecordFailure lockout SET warning: %v", err)
		}
	}

	return nil
}

// Reset clears the failure counter and lockout for key (e.g., after successful login).
func (rl *redisRateLimiter) Reset(ctx context.Context, key string) error {
	rateKey := fmt.Sprintf("rate:%s", key)
	lockedKey := fmt.Sprintf("locked:%s", key)

	if err := rl.rdb.Del(ctx, rateKey, lockedKey).Err(); err != nil {
		log.Printf("auth: redis Reset warning: %v", err)
	}
	return nil
}

// computeBackoff returns min(base^count, max) seconds.
func computeBackoff(count, base, max int) int {
	val := int(math.Pow(float64(base), float64(count)))
	if val > max {
		return max
	}
	return val
}
