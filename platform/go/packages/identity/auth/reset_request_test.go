package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shredbx/sbx-core/pkg/auth"
)

// Ensure context import is used
var _ context.Context

func setupResetService() (auth.ResetRequestService, *mockUserStore, *mockResetRequestRepo, *mockMagicLinkRepo) {
	users := newMockUserStore()
	resets := newMockResetRequestRepo()
	magicLinks := newMockMagicLinkRepo()
	cfg := testConfig()

	svc := auth.NewResetRequestService(cfg, users, resets, magicLinks)
	return svc, users, resets, magicLinks
}

// TC-H1: RequestReset takes statistically indistinguishable time for
// known vs. unknown emails — timing-based email enumeration is impractical.
func TestRequestReset_TimingEqualization(t *testing.T) {
	svc, users, _, _ := setupResetService()
	seedActiveUser(users) // test@example.com

	measure := func(email string) time.Duration {
		start := time.Now()
		_ = svc.RequestReset(t.Context(), email, "test")
		return time.Since(start)
	}

	const trials = 10
	var knownTotal, unknownTotal time.Duration
	for i := 0; i < trials; i++ {
		knownTotal += measure("test@example.com")
		unknownTotal += measure("nonexistent@example.com")
	}
	known := knownTotal / trials
	unknown := unknownTotal / trials

	// Both must clear the 30ms baseline pad
	assert.GreaterOrEqual(t, known.Milliseconds(), int64(25),
		"known-email path must meet timing baseline (got %v)", known)
	assert.GreaterOrEqual(t, unknown.Milliseconds(), int64(25),
		"unknown-email path must meet timing baseline (got %v)", unknown)

	// Difference between known/unknown must be small (well within network jitter)
	diff := known - unknown
	if diff < 0 {
		diff = -diff
	}
	assert.LessOrEqual(t, diff.Milliseconds(), int64(15),
		"timing delta between known/unknown must be <15ms (got %v)", diff)
}

func TestResetRequestService_RequestReset(t *testing.T) {
	t.Run("T-RR-1: valid email creates pending request", func(t *testing.T) {
		svc, users, resets, _ := setupResetService()
		seedActiveUser(users)

		err := svc.RequestReset(t.Context(), "test@example.com", "Forgot my password")
		require.NoError(t, err)

		pending, err := resets.ListPending(t.Context())
		require.NoError(t, err)
		assert.Len(t, pending, 1)
		assert.Equal(t, auth.ResetStatusPending, pending[0].Status)
	})

	t.Run("T-RR-2: nonexistent email returns no error (anti-enumeration)", func(t *testing.T) {
		svc, _, _, _ := setupResetService()

		err := svc.RequestReset(t.Context(), "nobody@example.com", "")
		assert.NoError(t, err) // Always 200 — no user enumeration
	})

	t.Run("T-RR-3: duplicate pending request for same user is idempotent", func(t *testing.T) {
		svc, users, resets, _ := setupResetService()
		seedActiveUser(users)

		_ = svc.RequestReset(t.Context(), "test@example.com", "First request")
		_ = svc.RequestReset(t.Context(), "test@example.com", "Second request")

		pending, _ := resets.ListPending(t.Context())
		assert.Len(t, pending, 1) // Only one pending request
	})
}

func TestResetRequestService_ListPending(t *testing.T) {
	t.Run("T-RR-4: returns all pending requests with user info", func(t *testing.T) {
		svc, users, _, _ := setupResetService()
		seedActiveUser(users)
		_ = svc.RequestReset(t.Context(), "test@example.com", "reason")

		requests, err := svc.ListPending(t.Context())
		require.NoError(t, err)
		assert.Len(t, requests, 1)
		assert.Equal(t, "test@example.com", requests[0].Email)
	})
}

func TestResetRequestService_Approve(t *testing.T) {
	t.Run("T-RR-5: approve generates magic link", func(t *testing.T) {
		svc, users, resets, _ := setupResetService()
		seedActiveUser(users)
		_ = svc.RequestReset(t.Context(), "test@example.com", "")

		pending, _ := resets.ListPending(t.Context())
		require.Len(t, pending, 1)

		adminID := uuid.New()
		link, err := svc.Approve(t.Context(), pending[0].ID, adminID)
		require.NoError(t, err)
		assert.NotEmpty(t, link.Token)

		// Request status should be approved
		req, _ := resets.Get(t.Context(), pending[0].ID)
		assert.Equal(t, auth.ResetStatusApproved, req.Status)
		assert.NotNil(t, req.ResolvedAt)
		assert.Equal(t, adminID, *req.ResolvedBy)
	})

	t.Run("T-RR-5b: approve persists magic link to repository", func(t *testing.T) {
		svc, users, _, magicLinks := setupResetService()
		seedActiveUser(users)
		_ = svc.RequestReset(t.Context(), "test@example.com", "")

		// Approve the pending request
		pending, _ := svc.ListPending(t.Context())
		require.Len(t, pending, 1)

		adminID := uuid.New()
		link, err := svc.Approve(t.Context(), pending[0].ID, adminID)
		require.NoError(t, err)
		require.NotEmpty(t, link.Token)

		// The magic link must be retrievable from the repository by token
		persisted, err := magicLinks.GetByToken(t.Context(), link.Token)
		require.NoError(t, err, "magic link must be persisted to repository")
		assert.Equal(t, link.ID, persisted.ID)
		assert.Equal(t, link.Token, persisted.Token)
		assert.Equal(t, link.UserID, persisted.UserID)
	})

	t.Run("T-RR-6: approve nonexistent request returns error", func(t *testing.T) {
		svc, _, _, _ := setupResetService()
		_, err := svc.Approve(t.Context(), uuid.New(), uuid.New())
		assert.Error(t, err)
	})
}

func TestResetRequestService_Reject(t *testing.T) {
	t.Run("T-RR-7: reject sets status to rejected", func(t *testing.T) {
		svc, users, resets, _ := setupResetService()
		seedActiveUser(users)
		_ = svc.RequestReset(t.Context(), "test@example.com", "")

		pending, _ := resets.ListPending(t.Context())
		require.Len(t, pending, 1)

		adminID := uuid.New()
		err := svc.Reject(t.Context(), pending[0].ID, adminID)
		require.NoError(t, err)

		req, _ := resets.Get(t.Context(), pending[0].ID)
		assert.Equal(t, auth.ResetStatusRejected, req.Status)
	})
}

func TestResetRequestService_ExpireOld(t *testing.T) {
	t.Run("T-RR-8: expire old requests marks expired", func(t *testing.T) {
		svc, users, resets, _ := setupResetService()
		seedActiveUser(users)

		// Insert an expired request directly
		expired := &auth.PasswordResetRequest{
			ID:        uuid.New(),
			UserID:    testUserID,
			Status:    auth.ResetStatusPending,
			ExpiresAt: time.Now().Add(-1 * time.Hour),
			CreatedAt: time.Now().Add(-8 * 24 * time.Hour),
		}
		resets.Create(t.Context(), expired)

		count, err := svc.ExpireOld(t.Context())
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})
}

func TestPasswordResetRequest_Types(t *testing.T) {
	t.Run("T-RR-9: ResetStatusPending constant defined", func(t *testing.T) {
		assert.Equal(t, auth.ResetRequestStatus("pending"), auth.ResetStatusPending)
	})

	t.Run("T-RR-10: ResetStatusApproved constant defined", func(t *testing.T) {
		assert.Equal(t, auth.ResetRequestStatus("approved"), auth.ResetStatusApproved)
	})

	t.Run("T-RR-11: ResetStatusRejected constant defined", func(t *testing.T) {
		assert.Equal(t, auth.ResetRequestStatus("rejected"), auth.ResetStatusRejected)
	})
}

// --- Mock ResetRequestRepository ---

type mockResetRequestRepo struct {
	requests map[uuid.UUID]*auth.PasswordResetRequest
}

func newMockResetRequestRepo() *mockResetRequestRepo {
	return &mockResetRequestRepo{requests: make(map[uuid.UUID]*auth.PasswordResetRequest)}
}

func (m *mockResetRequestRepo) Create(_ context.Context, req *auth.PasswordResetRequest) error {
	m.requests[req.ID] = req
	return nil
}

func (m *mockResetRequestRepo) Get(_ context.Context, id uuid.UUID) (*auth.PasswordResetRequest, error) {
	r, ok := m.requests[id]
	if !ok {
		return nil, auth.ErrResetRequestNotFound
	}
	return r, nil
}

func (m *mockResetRequestRepo) GetPendingByUser(_ context.Context, userID uuid.UUID) (*auth.PasswordResetRequest, error) {
	for _, r := range m.requests {
		if r.UserID == userID && r.Status == auth.ResetStatusPending {
			return r, nil
		}
	}
	return nil, auth.ErrResetRequestNotFound
}

func (m *mockResetRequestRepo) ListPending(_ context.Context) ([]auth.PasswordResetRequest, error) {
	var result []auth.PasswordResetRequest
	for _, r := range m.requests {
		if r.Status == auth.ResetStatusPending {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *mockResetRequestRepo) UpdateStatus(_ context.Context, id uuid.UUID, status auth.ResetRequestStatus, resolvedBy uuid.UUID) error {
	r, ok := m.requests[id]
	if !ok {
		return auth.ErrResetRequestNotFound
	}
	r.Status = status
	now := time.Now()
	r.ResolvedAt = &now
	r.ResolvedBy = &resolvedBy
	return nil
}

func (m *mockResetRequestRepo) SetMagicLink(_ context.Context, id, magicLinkID uuid.UUID) error {
	r, ok := m.requests[id]
	if !ok {
		return auth.ErrResetRequestNotFound
	}
	r.MagicLinkID = &magicLinkID
	return nil
}

func (m *mockResetRequestRepo) ExpireOld(_ context.Context) (int64, error) {
	var count int64
	for _, r := range m.requests {
		if r.Status == auth.ResetStatusPending && time.Now().After(r.ExpiresAt) {
			r.Status = auth.ResetStatusExpired
			count++
		}
	}
	return count, nil
}

