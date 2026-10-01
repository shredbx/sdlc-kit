package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shredbx/sbx-core/pkg/user"
)

// requestResetMinDuration is the minimum wall-clock time RequestReset takes.
// Pads short paths (unknown email, has-pending) so they're indistinguishable
// from the full path (Create new request). H1 timing-attack defense. Combined
// with rate limiting (C2), email enumeration via timing becomes impractical.
const requestResetMinDuration = 30 * time.Millisecond

type resetRequestService struct {
	cfg        AuthConfig
	users      user.UserStore
	resets     ResetRequestRepository
	magicLinks MagicLinkRepository
}

// NewResetRequestService creates a password reset request service.
func NewResetRequestService(cfg AuthConfig, users user.UserStore, resets ResetRequestRepository, magicLinks MagicLinkRepository) ResetRequestService {
	if cfg.ResetRequestExpiry == 0 {
		cfg.ResetRequestExpiry = 7 * 24 * time.Hour
	}
	return &resetRequestService{
		cfg:        cfg,
		users:      users,
		resets:     resets,
		magicLinks: magicLinks,
	}
}

func (s *resetRequestService) RequestReset(ctx context.Context, email, reason string) error {
	// H1: pad wall-clock time so unknown vs. known email is timing-indistinguishable.
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed < requestResetMinDuration {
			time.Sleep(requestResetMinDuration - elapsed)
		}
	}()

	// Always return nil to prevent user enumeration via response body.
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil // no error even if user doesn't exist
	}

	// Check for existing pending request — idempotent
	existing, err := s.resets.GetPendingByUser(ctx, u.ID)
	if err == nil && existing != nil {
		return nil // already has a pending request
	}

	req := &PasswordResetRequest{
		ID:        uuid.New(),
		UserID:    u.ID,
		Email:     u.Email,
		Status:    ResetStatusPending,
		Reason:    reason,
		ExpiresAt: time.Now().Add(s.cfg.ResetRequestExpiry),
		CreatedAt: time.Now(),
	}

	return s.resets.Create(ctx, req)
}

func (s *resetRequestService) ListPending(ctx context.Context) ([]PasswordResetRequest, error) {
	return s.resets.ListPending(ctx)
}

func (s *resetRequestService) Approve(ctx context.Context, requestID, adminID uuid.UUID) (*MagicLink, error) {
	req, err := s.resets.Get(ctx, requestID)
	if err != nil {
		return nil, err
	}

	if req.Status != ResetStatusPending {
		return nil, ErrResetRequestNotFound
	}

	// Create magic link directly (user already exists — this is a reset, not an invite)
	token, err := GenerateSecureToken()
	if err != nil {
		return nil, err
	}

	ttl := s.cfg.MagicLinkTTL
	if s.cfg.MagicLinkResetTTL > 0 {
		ttl = s.cfg.MagicLinkResetTTL
	}
	link := &MagicLink{
		ID:        uuid.New(),
		Token:     token,
		UserID:    req.UserID,
		Purpose:   MagicLinkPurposeReset,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}

	// Persist the magic link so the token can be validated later
	if err := s.magicLinks.Create(ctx, link); err != nil {
		return nil, err
	}

	// Update request status
	if err := s.resets.UpdateStatus(ctx, requestID, ResetStatusApproved, adminID); err != nil {
		return nil, err
	}
	if err := s.resets.SetMagicLink(ctx, requestID, link.ID); err != nil {
		return nil, err
	}

	return link, nil
}

func (s *resetRequestService) Reject(ctx context.Context, requestID, adminID uuid.UUID) error {
	return s.resets.UpdateStatus(ctx, requestID, ResetStatusRejected, adminID)
}

func (s *resetRequestService) ExpireOld(ctx context.Context) (int64, error) {
	return s.resets.ExpireOld(ctx)
}
