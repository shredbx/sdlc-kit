package auth_test

import (
	"context"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shredbx/sbx-core/pkg/auth"
)

// in-memory audit repo for testing
type mockAuditRepo struct {
	mu     sync.Mutex
	events []*auth.AuditEvent
}

func (m *mockAuditRepo) Append(_ context.Context, e *auth.AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	m.events = append(m.events, e)
	return nil
}

func (m *mockAuditRepo) ListEvents(_ context.Context, opts auth.AuditListOptions) ([]auth.AuditEvent, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var filtered []auth.AuditEvent
	for _, e := range m.events {
		if opts.Action != "" && e.Action != opts.Action {
			continue
		}
		if opts.ActorID != nil && (e.ActorID == nil || *e.ActorID != *opts.ActorID) {
			continue
		}
		if opts.TargetType != "" && e.TargetType != opts.TargetType {
			continue
		}
		if opts.TargetID != nil && (e.TargetID == nil || *e.TargetID != *opts.TargetID) {
			continue
		}
		if opts.Query != "" && !mockMatchesQuery(e, opts.Query) {
			continue
		}
		filtered = append(filtered, *e)
	}
	mockSortEvents(filtered, opts)
	total := len(filtered)
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	start := opts.Offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], total, nil
}

// mockMatchesQuery mirrors the store's ILIKE search group: case-insensitive
// substring across action, target_type, ip_address, and the string form of
// actor_id / target_id (payload + user_agent excluded, like the store).
func mockMatchesQuery(e *auth.AuditEvent, query string) bool {
	q := strings.ToLower(query)
	fields := []string{e.Action, e.TargetType, e.IPAddress}
	if e.ActorID != nil {
		fields = append(fields, e.ActorID.String())
	}
	if e.TargetID != nil {
		fields = append(fields, e.TargetID.String())
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

// mockSortEvents mirrors the store's ORDER BY: sort by the whitelisted column
// (unknown/empty → created_at), direction DESC when (Desc || Sort==""), with a
// created_at DESC then id DESC tiebreaker for determinism.
func mockSortEvents(events []auth.AuditEvent, opts auth.AuditListOptions) {
	col := opts.Sort
	switch col {
	case "created_at", "action", "actor_id", "target_type", "ip_address":
	default:
		col = "created_at"
	}
	desc := opts.Desc || opts.Sort == ""

	key := func(e auth.AuditEvent) string {
		switch col {
		case "action":
			return e.Action
		case "actor_id":
			if e.ActorID != nil {
				return e.ActorID.String()
			}
			return ""
		case "target_type":
			return e.TargetType
		case "ip_address":
			return e.IPAddress
		default:
			return e.CreatedAt.Format(time.RFC3339Nano)
		}
	}

	sort.SliceStable(events, func(i, j int) bool {
		ki, kj := key(events[i]), key(events[j])
		if ki != kj {
			if desc {
				return ki > kj
			}
			return ki < kj
		}
		// Tiebreaker: created_at DESC, then id DESC.
		ci := events[i].CreatedAt.Format(time.RFC3339Nano)
		cj := events[j].CreatedAt.Format(time.RFC3339Nano)
		if ci != cj {
			return ci > cj
		}
		return events[i].ID.String() > events[j].ID.String()
	})
}

func (m *mockAuditRepo) byAction(action string) []*auth.AuditEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*auth.AuditEvent
	for _, e := range m.events {
		if e.Action == action {
			out = append(out, e)
		}
	}
	return out
}

// TC-M7: Login failure emits an audit row with reason.
func TestLogin_FailureEmitsAudit(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	audit := &mockAuditRepo{}
	cfg := testConfig()

	svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, audit)
	seedActiveUser(users)

	// Wrong password — should emit AuditActionLoginFailure with reason=wrong_password
	_, err := svc.Login(t.Context(), "test@example.com", "incorrect")
	require.Error(t, err)

	failures := audit.byAction(auth.AuditActionLoginFailure)
	require.Len(t, failures, 1)
	assert.Equal(t, "wrong_password", failures[0].Payload["reason"])
	assert.NotNil(t, failures[0].ActorID)

	// Unknown email — should emit AuditActionLoginFailure with reason=unknown_user (no ActorID)
	_, err = svc.Login(t.Context(), "ghost@example.com", "any")
	require.Error(t, err)
	failures = audit.byAction(auth.AuditActionLoginFailure)
	require.Len(t, failures, 2)
	assert.Equal(t, "unknown_user", failures[1].Payload["reason"])
	assert.Nil(t, failures[1].ActorID, "anonymous login failure has no actor")
}

// TC-M7: Successful login emits success event.
func TestLogin_SuccessEmitsAudit(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	audit := &mockAuditRepo{}
	cfg := testConfig()

	svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, audit)
	seedActiveUser(users)

	_, err := svc.Login(t.Context(), "test@example.com", "correct-password")
	require.NoError(t, err)

	successes := audit.byAction(auth.AuditActionLoginSuccess)
	assert.Len(t, successes, 1)
	assert.NotNil(t, successes[0].ActorID)
}

// TC-M7: Token reuse detection emits audit event.
func TestRefreshToken_ReuseEmitsAudit(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	audit := &mockAuditRepo{}
	cfg := testConfig()

	svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, audit)
	seedActiveUser(users)

	tp1, _ := svc.Login(t.Context(), "test@example.com", "correct-password")
	_, _ = svc.RefreshToken(t.Context(), tp1.RefreshToken)

	// Age the rotated token past the rotation grace window so the replay is theft,
	// not a benign in-window race (2607-125) — otherwise it emits grace-replay.
	sessions.mu.Lock()
	for _, s := range sessions.sessions {
		if s.RefreshToken == auth.HashToken(tp1.RefreshToken) {
			aged := time.Now().Add(-2 * time.Minute)
			s.RevokedAt = &aged
		}
	}
	sessions.mu.Unlock()

	// Replay rotated token — must trigger theft event
	_, err := svc.RefreshToken(t.Context(), tp1.RefreshToken)
	assert.ErrorIs(t, err, auth.ErrRefreshTokenReused)

	theft := audit.byAction(auth.AuditActionTokenReuseDetected)
	require.Len(t, theft, 1)
	assert.Equal(t, true, theft[0].Payload["family_revoked"])
}
