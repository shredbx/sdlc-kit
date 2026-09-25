package rss

import (
	"context"
	"sync"
	"time"
)

// Cache is a storage-agnostic byte cache with per-entry TTL. The shared engine
// never hard-couples to Redis (Rule #9): consumers wire a Redis-backed adapter
// in production; tests and lightweight callers use MemoryCache.
type Cache interface {
	// Get returns the cached value for key. found is false on a miss or an
	// expired entry. A non-nil error signals a backend failure (not a miss).
	Get(ctx context.Context, key string) (val []byte, found bool, err error)
	// Set stores val under key for ttl. A ttl <= 0 stores the value without
	// expiry.
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
}

// memEntry is a stored value plus its absolute expiry (zero = never).
type memEntry struct {
	val       []byte
	expiresAt time.Time
}

// MemoryCache is a concurrency-safe in-memory Cache with TTL expiry. Use
// NewMemoryCache to construct one.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]memEntry
}

// NewMemoryCache returns an empty, ready-to-use in-memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{entries: make(map[string]memEntry)}
}

// Get returns the value for key, treating expired entries as misses. The
// context is accepted for interface parity; the in-memory backend never blocks.
func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		// Lazily evict the expired entry.
		c.mu.Lock()
		if cur, still := c.entries[key]; still && cur.expiresAt.Equal(e.expiresAt) {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return nil, false, nil
	}
	// Return a copy so callers can't mutate the stored bytes.
	out := make([]byte, len(e.val))
	copy(out, e.val)
	return out, true, nil
}

// Set stores a copy of val under key for ttl (ttl <= 0 means no expiry).
func (c *MemoryCache) Set(_ context.Context, key string, val []byte, ttl time.Duration) error {
	stored := make([]byte, len(val))
	copy(stored, val)
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.entries[key] = memEntry{val: stored, expiresAt: exp}
	c.mu.Unlock()
	return nil
}
