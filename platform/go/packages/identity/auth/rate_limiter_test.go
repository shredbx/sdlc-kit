package auth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shredbx/sbx-core/pkg/auth"
)

func TestRateLimiter_Allow(t *testing.T) {
	t.Run("T-RL-1: first request is allowed", func(t *testing.T) {
		limiter := newMockRateLimiter()
		result, err := limiter.Allow(t.Context(), "192.168.1.1", auth.RateLimitConfig{
			MaxAttempts: 10,
			Window:      time.Minute,
		})
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 9, result.Remaining)
	})

	t.Run("T-RL-2: 10th request is allowed (at limit)", func(t *testing.T) {
		limiter := newMockRateLimiter()
		cfg := auth.RateLimitConfig{MaxAttempts: 10, Window: time.Minute}
		for i := 0; i < 9; i++ {
			limiter.RecordFailure(t.Context(), "192.168.1.1", cfg)
		}
		result, err := limiter.Allow(t.Context(), "192.168.1.1", cfg)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 0, result.Remaining)
	})

	t.Run("T-RL-3: 11th request is denied (over limit)", func(t *testing.T) {
		limiter := newMockRateLimiter()
		cfg := auth.RateLimitConfig{MaxAttempts: 10, Window: time.Minute}
		for i := 0; i < 10; i++ {
			limiter.RecordFailure(t.Context(), "192.168.1.1", cfg)
		}
		result, err := limiter.Allow(t.Context(), "192.168.1.1", cfg)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.True(t, result.RetryAfter > 0)
	})

	t.Run("T-RL-4: different IPs are independent", func(t *testing.T) {
		limiter := newMockRateLimiter()
		cfg := auth.RateLimitConfig{MaxAttempts: 10, Window: time.Minute}
		for i := 0; i < 10; i++ {
			limiter.RecordFailure(t.Context(), "192.168.1.1", cfg)
		}
		result, err := limiter.Allow(t.Context(), "192.168.1.2", cfg)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	})
}

func TestRateLimiter_RecordFailure(t *testing.T) {
	t.Run("T-RL-5: failure increments counter", func(t *testing.T) {
		limiter := newMockRateLimiter()
		cfg := auth.RateLimitConfig{MaxAttempts: 10, Window: time.Minute}
		err := limiter.RecordFailure(t.Context(), "192.168.1.1", cfg)
		require.NoError(t, err)
		result, _ := limiter.Allow(t.Context(), "192.168.1.1", cfg)
		assert.Equal(t, 8, result.Remaining)
	})

	t.Run("T-RL-6: exponential backoff delay on consecutive failures", func(t *testing.T) {
		limiter := newMockRateLimiter()
		cfg := auth.RateLimitConfig{MaxAttempts: 10, Window: time.Minute}
		// 3 failures → 4s delay (2^2)
		for i := 0; i < 3; i++ {
			limiter.RecordFailure(t.Context(), "192.168.1.1", cfg)
		}
		result, _ := limiter.Allow(t.Context(), "192.168.1.1", cfg)
		assert.True(t, result.Allowed)
		assert.True(t, result.BackoffSeconds >= 4, "expected 4s backoff after 3 failures, got %d", result.BackoffSeconds)
	})
}

func TestRateLimiter_Reset(t *testing.T) {
	t.Run("T-RL-7: reset clears failure counter", func(t *testing.T) {
		limiter := newMockRateLimiter()
		cfg := auth.RateLimitConfig{MaxAttempts: 10, Window: time.Minute}
		for i := 0; i < 5; i++ {
			limiter.RecordFailure(t.Context(), "192.168.1.1", cfg)
		}
		err := limiter.Reset(t.Context(), "192.168.1.1")
		require.NoError(t, err)
		result, _ := limiter.Allow(t.Context(), "192.168.1.1", cfg)
		assert.True(t, result.Allowed)
		assert.Equal(t, 9, result.Remaining)
	})
}

func TestRateLimiter_Lockout(t *testing.T) {
	t.Run("T-RL-8: account locked after max consecutive failures", func(t *testing.T) {
		limiter := newMockRateLimiter()
		cfg := auth.RateLimitConfig{MaxAttempts: 10, Window: time.Minute}
		for i := 0; i < 10; i++ {
			limiter.RecordFailure(t.Context(), "192.168.1.1", cfg)
		}
		result, err := limiter.Allow(t.Context(), "192.168.1.1", cfg)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.True(t, result.Locked)
		assert.True(t, result.RetryAfter > 0)
	})

	t.Run("T-RL-9: lockout expires after window", func(t *testing.T) {
		limiter := newMockRateLimiterWithExpiry()
		cfg := auth.RateLimitConfig{MaxAttempts: 10, Window: time.Minute}
		for i := 0; i < 10; i++ {
			limiter.RecordFailure(t.Context(), "192.168.1.1", cfg)
		}
		// Simulate window expiry
		limiter.AdvanceTime(2 * time.Minute)
		result, err := limiter.Allow(t.Context(), "192.168.1.1", cfg)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	})
}

func TestRateLimiter_InviteRateLimit(t *testing.T) {
	t.Run("T-RL-10: invite rate limit 3 per user per minute", func(t *testing.T) {
		limiter := newMockRateLimiter()
		cfg := auth.RateLimitConfig{MaxAttempts: 3, Window: time.Minute}
		for i := 0; i < 3; i++ {
			limiter.RecordFailure(t.Context(), "admin-user-id", cfg)
		}
		result, err := limiter.Allow(t.Context(), "admin-user-id", cfg)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
	})
}

func TestRateLimiter_BackoffCap(t *testing.T) {
	t.Run("T-RL-11: backoff capped at 300 seconds", func(t *testing.T) {
		limiter := newMockRateLimiter()
		cfg := auth.RateLimitConfig{MaxAttempts: 10, Window: time.Minute}
		// Many failures to exceed cap
		for i := 0; i < 10; i++ {
			limiter.RecordFailure(t.Context(), "192.168.1.1", cfg)
		}
		result, _ := limiter.Allow(t.Context(), "192.168.1.1", cfg)
		assert.True(t, result.BackoffSeconds <= 300, "backoff should be capped at 300s")
	})
}
