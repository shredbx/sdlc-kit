package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// redisSessionCache implements SessionCache using Redis for fast session lookups
// and JTI revocation checks. All Redis errors are gracefully degraded — cache
// misses and unavailability never block authentication.
type redisSessionCache struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewRedisSessionCache returns a SessionCache backed by Redis.
// The ttl controls how long session data is cached; revocation markers
// persist for 2x ttl to outlive the sessions they invalidate.
func NewRedisSessionCache(rdb *redis.Client, ttl time.Duration) SessionCache {
	return &redisSessionCache{rdb: rdb, ttl: ttl}
}

// GetSession retrieves a cached session by ID.
// Returns (nil, nil) on cache miss or Redis error (graceful degradation).
func (c *redisSessionCache) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	key := fmt.Sprintf("session:%s", id.String())
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		log.Printf("auth: redis GetSession warning: %v", err)
		return nil, nil
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		log.Printf("auth: redis GetSession unmarshal warning: %v", err)
		return nil, nil
	}
	return &session, nil
}

// SetSession caches a session with the configured TTL.
func (c *redisSessionCache) SetSession(ctx context.Context, session *Session) error {
	key := fmt.Sprintf("session:%s", session.ID.String())
	// F1: never persist the refresh token in the cache — session lookups here are
	// by ID only. Strip it defensively so no caller can leak a token into Redis.
	cp := *session
	cp.RefreshToken = ""
	data, err := json.Marshal(&cp)
	if err != nil {
		log.Printf("auth: redis SetSession marshal warning: %v", err)
		return nil
	}

	if err := c.rdb.Set(ctx, key, data, c.ttl).Err(); err != nil {
		log.Printf("auth: redis SetSession warning: %v", err)
	}
	return nil
}

// DeleteSession removes a cached session.
func (c *redisSessionCache) DeleteSession(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("session:%s", id.String())
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		log.Printf("auth: redis DeleteSession warning: %v", err)
	}
	return nil
}

// IsRevoked checks whether a JWT ID has been revoked.
// Returns false on Redis error (graceful degradation — fail open).
func (c *redisSessionCache) IsRevoked(ctx context.Context, jti uuid.UUID) (bool, error) {
	key := fmt.Sprintf("revoked:%s", jti.String())
	exists, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		log.Printf("auth: redis IsRevoked warning: %v", err)
		return false, nil
	}
	return exists > 0, nil
}

// SetRevoked marks a JWT ID as revoked. The marker persists for 2x the
// session TTL so it outlives any cached session referencing this JTI.
func (c *redisSessionCache) SetRevoked(ctx context.Context, jti uuid.UUID) error {
	key := fmt.Sprintf("revoked:%s", jti.String())
	if err := c.rdb.Set(ctx, key, "1", 2*c.ttl).Err(); err != nil {
		log.Printf("auth: redis SetRevoked warning: %v", err)
	}
	return nil
}
