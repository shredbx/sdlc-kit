package auth_test

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shredbx/sbx-core/pkg/auth"
	"github.com/shredbx/sbx-core/pkg/repository"
	"github.com/shredbx/sbx-core/pkg/user"
)

// --- Mock UserStore ---

type mockUserStore struct {
	mu    sync.RWMutex
	users map[string]*user.User // keyed by email
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{users: make(map[string]*user.User)}
}

func (m *mockUserStore) Create(_ context.Context, input user.CreateUserInput) (*user.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.users[input.Email]; exists {
		return nil, user.ErrEmailExists
	}
	u := &user.User{
		ID:       uuid.New(),
		Email:    input.Email,
		FullName: input.FullName,
		Role:     input.Role,
		Status:   user.UserStatusInvited,
	}
	m.users[input.Email] = u
	return u, nil
}

func (m *mockUserStore) Get(_ context.Context, id uuid.UUID) (*user.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, user.ErrUserNotFound
}

func (m *mockUserStore) GetByEmail(_ context.Context, email string) (*user.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[email]
	if !ok {
		return nil, user.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserStore) Update(_ context.Context, id uuid.UUID, input user.UpdateUserInput) (*user.User, error) {
	return nil, nil
}

func (m *mockUserStore) Activate(_ context.Context, id uuid.UUID, input user.ActivateUserInput) (*user.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.ID == id {
			// H4: status='invited' guard mirrors postgres impl
			if u.Status != user.UserStatusInvited {
				return nil, user.ErrUserAlreadyActive
			}
			u.PasswordHash = input.PasswordHash
			u.FullName = input.FullName
			u.Status = user.UserStatusActive
			return u, nil
		}
	}
	return nil, user.ErrUserNotFound
}

func (m *mockUserStore) ResetPassword(_ context.Context, id uuid.UUID, passwordHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.ID == id {
			if u.Status != user.UserStatusActive {
				return user.ErrUserNotActive
			}
			u.PasswordHash = passwordHash
			return nil
		}
	}
	return user.ErrUserNotFound
}

func (m *mockUserStore) UpdateLastLogin(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockUserStore) UpdateRole(_ context.Context, _ uuid.UUID, _ string) (*user.User, error) {
	return nil, nil
}
func (m *mockUserStore) SoftDelete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockUserStore) GetByEmailIncludingDeleted(_ context.Context, email string) (*user.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, user.ErrUserNotFound
}
func (m *mockUserStore) Reactivate(_ context.Context, id uuid.UUID) error {
	for _, u := range m.users {
		if u.ID == id && u.Status == user.UserStatusDeleted {
			u.Status = user.UserStatusInvited
			u.DeletedAt = nil
			u.PasswordHash = ""
			return nil
		}
	}
	return user.ErrUserNotFound
}
func (m *mockUserStore) List(_ context.Context, _ repository.ListOptions) ([]user.User, int, error) {
	return nil, 0, nil
}
func (m *mockUserStore) ListIncludingDeleted(_ context.Context, _ repository.ListOptions) ([]user.User, int, error) {
	return nil, 0, nil
}

// --- Mock SessionRepository ---

type mockSessionRepo struct {
	mu       sync.RWMutex
	sessions map[uuid.UUID]*auth.Session
	// Test hooks: when set, the corresponding method returns this error WITHOUT
	// mutating state — mirrors an atomic DB UPDATE that fails and writes nothing,
	// so the fail-open (RevokeForRotation) / failed-containment (RevokeAllForUser)
	// paths in RefreshToken can be exercised.
	revokeForRotationErr error
	revokeAllForUserErr  error
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{sessions: make(map[uuid.UUID]*auth.Session)}
}

func (m *mockSessionRepo) Create(_ context.Context, s *auth.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	return nil
}

func (m *mockSessionRepo) Get(_ context.Context, id uuid.UUID) (*auth.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, auth.ErrSessionNotFound
	}
	return s, nil
}

func (m *mockSessionRepo) GetByRefreshToken(_ context.Context, token string) (*auth.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.sessions {
		if s.RefreshToken == token {
			return s, nil
		}
	}
	return nil, auth.ErrSessionNotFound
}

func (m *mockSessionRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]auth.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []auth.Session
	for _, s := range m.sessions {
		if s.UserID == userID && s.RevokedAt == nil {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockSessionRepo) Revoke(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return auth.ErrSessionNotFound
	}
	now := time.Now()
	s.RevokedAt = &now
	return nil
}
func (m *mockSessionRepo) RevokeForRotation(_ context.Context, oldID, newID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.revokeForRotationErr != nil {
		return m.revokeForRotationErr
	}
	s, ok := m.sessions[oldID]
	if !ok {
		return auth.ErrSessionNotFound
	}
	now := time.Now()
	s.RevokedAt = &now
	s.ReplacedBySessionID = &newID
	return nil
}
func (m *mockSessionRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}
func (m *mockSessionRepo) DeleteExpired(_ context.Context) (int64, error) { return 0, nil }
func (m *mockSessionRepo) RevokeAllForUser(_ context.Context, userID uuid.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.revokeAllForUserErr != nil {
		return 0, m.revokeAllForUserErr
	}
	var count int64
	now := time.Now()
	for _, s := range m.sessions {
		if s.UserID == userID && s.RevokedAt == nil {
			s.RevokedAt = &now
			count++
		}
	}
	return count, nil
}

// --- Mock MagicLinkRepository ---

type mockMagicLinkRepo struct {
	mu    sync.RWMutex
	links map[string]*auth.MagicLink // keyed by token
}

func newMockMagicLinkRepo() *mockMagicLinkRepo {
	return &mockMagicLinkRepo{links: make(map[string]*auth.MagicLink)}
}

func (m *mockMagicLinkRepo) Create(_ context.Context, link *auth.MagicLink) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.links[link.Token] = link
	return nil
}

func (m *mockMagicLinkRepo) GetByToken(_ context.Context, token string) (*auth.MagicLink, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	l, ok := m.links[token]
	if !ok {
		return nil, auth.ErrMagicLinkNotFound
	}
	return l, nil
}

func (m *mockMagicLinkRepo) GetActiveByUserID(_ context.Context, userID uuid.UUID) (*auth.MagicLink, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, l := range m.links {
		if l.UserID == userID && l.UsedAt == nil && time.Now().Before(l.ExpiresAt) {
			return l, nil
		}
	}
	return nil, auth.ErrMagicLinkNotFound
}

func (m *mockMagicLinkRepo) MarkUsed(_ context.Context, _ uuid.UUID) error { return nil }

func (m *mockMagicLinkRepo) InvalidateUnusedForUser(_ context.Context, userID uuid.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	now := time.Now()
	for _, l := range m.links {
		if l.UserID == userID && l.UsedAt == nil {
			l.UsedAt = &now
			count++
		}
	}
	return count, nil
}

// --- Mock SessionCache ---

type mockSessionCache struct {
	mu       sync.RWMutex
	sessions map[uuid.UUID]*auth.Session
	revoked  map[uuid.UUID]bool
}

func newMockSessionCache() *mockSessionCache {
	return &mockSessionCache{
		sessions: make(map[uuid.UUID]*auth.Session),
		revoked:  make(map[uuid.UUID]bool),
	}
}

func (m *mockSessionCache) GetSession(_ context.Context, id uuid.UUID) (*auth.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, nil // cache miss
	}
	return s, nil
}

func (m *mockSessionCache) SetSession(_ context.Context, s *auth.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	return nil
}

func (m *mockSessionCache) DeleteSession(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}

func (m *mockSessionCache) IsRevoked(_ context.Context, jti uuid.UUID) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.revoked[jti], nil
}

func (m *mockSessionCache) SetRevoked(_ context.Context, jti uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.revoked[jti] = true
	return nil
}

// --- Mock RateLimiter ---

type mockRateLimiter struct {
	mu       sync.RWMutex
	counters map[string]int
	offset   time.Duration
}

func newMockRateLimiter() *mockRateLimiter {
	return &mockRateLimiter{counters: make(map[string]int)}
}

func newMockRateLimiterWithExpiry() *mockRateLimiter {
	return &mockRateLimiter{counters: make(map[string]int)}
}

func (m *mockRateLimiter) Allow(_ context.Context, key string, cfg auth.RateLimitConfig) (*auth.RateLimitResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := m.counters[key]
	if count >= cfg.MaxAttempts {
		backoff := 1 << min(count, 9) // 2^n capped
		if backoff > 300 {
			backoff = 300
		}
		return &auth.RateLimitResult{
			Allowed:        false,
			Remaining:      0,
			Locked:         count >= cfg.MaxAttempts,
			RetryAfter:     int64(cfg.Window.Seconds()),
			BackoffSeconds: backoff,
		}, nil
	}

	remaining := cfg.MaxAttempts - count - 1
	backoff := 0
	if count >= 2 {
		backoff = 1 << count
		if backoff > 300 {
			backoff = 300
		}
	}

	return &auth.RateLimitResult{
		Allowed:        true,
		Remaining:      remaining,
		BackoffSeconds: backoff,
	}, nil
}

func (m *mockRateLimiter) RecordFailure(_ context.Context, key string, _ auth.RateLimitConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[key]++
	return nil
}

func (m *mockRateLimiter) Reset(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.counters, key)
	return nil
}

func (m *mockRateLimiter) AdvanceTime(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.offset += d
	// Clear all counters to simulate window expiry
	m.counters = make(map[string]int)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
