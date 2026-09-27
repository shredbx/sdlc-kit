package visitoractivity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// saltRedisKeyPrefix namespaces the per-day salt in Redis.
const saltRedisKeyPrefix = "visitor_activity:salt:"

// saltTTL keeps a day's salt alive long enough to span clock skew and late
// requests at the day boundary, then lets it expire. 48h covers "yesterday"
// without retaining salts indefinitely (a shorter retention strengthens the
// non-durable-PII property).
const saltTTL = 48 * time.Hour

// saltBytes is the entropy of each daily salt (32 bytes → 64 hex chars).
const saltBytes = 32

// SaltProvider yields the current day's secret salt for visitor-id hashing.
// Implementations MUST return the same salt for the whole UTC day and a fresh
// one after the day rolls over.
type SaltProvider interface {
	Salt(ctx context.Context) (string, error)
}

// utcDay returns the current UTC date as YYYY-MM-DD. Used as the rotation key.
func utcDay() string {
	return time.Now().UTC().Format("2006-01-02")
}

// newSalt returns a fresh cryptographically-random hex salt.
func newSalt() (string, error) {
	b := make([]byte, saltBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("visitoractivity: generate salt: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// MemorySaltProvider holds the current day's salt in process memory. Suitable
// for single-instance deployments and tests. Multi-instance deployments should
// use RedisSaltProvider so all instances share the same daily salt (otherwise
// the same visitor counts as distinct per instance).
type MemorySaltProvider struct {
	mu    sync.Mutex
	dayFn func() string // injectable for tests; nil → real UTC date
	day   string
	salt  string
}

// NewMemorySaltProvider creates an in-memory provider. Pass nil for dayFn to
// use the real UTC date; tests inject a fixed/advancing day to exercise
// rotation deterministically.
func NewMemorySaltProvider(dayFn func() string) *MemorySaltProvider {
	if dayFn == nil {
		dayFn = utcDay
	}
	return &MemorySaltProvider{dayFn: dayFn}
}

// Salt returns today's salt, rolling a fresh one when the day changes.
func (p *MemorySaltProvider) Salt(_ context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	day := p.dayFn()
	if day == p.day && p.salt != "" {
		return p.salt, nil
	}
	s, err := newSalt()
	if err != nil {
		return "", err
	}
	p.day = day
	p.salt = s
	return s, nil
}

// RedisSaltProvider stores the day's salt in Redis so every instance of a
// horizontally-scaled service shares it. The key is
// "visitor_activity:salt:<UTC-date>"; a SETNX seeds a fresh salt exactly once
// per day across the fleet (first writer wins), then everyone GETs it.
type RedisSaltProvider struct {
	rdb   *redis.Client
	dayFn func() string // injectable for tests; nil → real UTC date
}

// NewRedisSaltProvider creates a Redis-backed provider. Pass nil for dayFn to
// use the real UTC date.
func NewRedisSaltProvider(rdb *redis.Client, dayFn func() string) *RedisSaltProvider {
	if dayFn == nil {
		dayFn = utcDay
	}
	return &RedisSaltProvider{rdb: rdb, dayFn: dayFn}
}

// Salt returns today's shared salt. It seeds a fresh salt via SETNX (so only
// the first caller of the day pays for generation) and then GETs the
// authoritative value — guaranteeing all instances agree even if two seed
// concurrently.
func (p *RedisSaltProvider) Salt(ctx context.Context) (string, error) {
	key := saltRedisKeyPrefix + p.dayFn()

	seed, err := newSalt()
	if err != nil {
		return "", err
	}
	// SetNX: only the first writer of the day stores its seed; TTL caps retention.
	if err := p.rdb.SetNX(ctx, key, seed, saltTTL).Err(); err != nil {
		return "", fmt.Errorf("visitoractivity: redis setnx salt: %w", err)
	}
	// GET the authoritative value (ours if we won the race, the winner's otherwise).
	val, err := p.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("visitoractivity: redis get salt: %w", err)
	}
	return val, nil
}
