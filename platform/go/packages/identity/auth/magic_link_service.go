package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shredbx/sbx-core/pkg/user"
)

type magicLinkService struct {
	cfg        AuthConfig
	users      user.UserStore
	magicLinks MagicLinkRepository
	sessions   SessionRepository
	cache      SessionCache
}

// NewMagicLinkService creates a magic link service with all dependencies.
func NewMagicLinkService(cfg AuthConfig, users user.UserStore, magicLinks MagicLinkRepository, sessions SessionRepository, cache SessionCache) MagicLinkService {
	return &magicLinkService{
		cfg:        cfg,
		users:      users,
		magicLinks: magicLinks,
		sessions:   sessions,
		cache:      cache,
	}
}

func (s *magicLinkService) Generate(ctx context.Context, email, role string) (*MagicLink, error) {
	if err := user.ValidateEmail(email); err != nil {
		return nil, err
	}

	// Check if user already exists
	existing, err := s.users.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		// User exists — handle based on status. M5: we no longer return the
		// "active link" idempotently because the plaintext token isn't persisted
		// (only its hash). createLink invalidates prior unused links (H5), so
		// callers always get a fresh token and the prior one becomes invalid
		// atomically — same end-user effect as the old idempotent return.
		if existing.Status == user.UserStatusInvited {
			return s.createLink(ctx, existing.ID, MagicLinkPurposeInvite)
		}
		// Active user — generate a login magic link (passwordless re-auth)
		if existing.Status == user.UserStatusActive {
			return s.createLink(ctx, existing.ID, MagicLinkPurposeLogin)
		}
		return nil, fmt.Errorf("create user: %w", user.ErrEmailExists)
	}

	// User doesn't exist — create invited user with invite-purpose link
	u, err := s.users.Create(ctx, user.CreateUserInput{
		Email: email,
		Role:  role,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.createLink(ctx, u.ID, MagicLinkPurposeInvite)
}

// GenerateForPurpose creates a magic link for a specific user and purpose
// without going through the email-lookup path. Used by the password-reset
// admin-approval flow (C1) where the user already exists.
func (s *magicLinkService) GenerateForPurpose(ctx context.Context, userID uuid.UUID, purpose MagicLinkPurpose) (*MagicLink, error) {
	return s.createLink(ctx, userID, purpose)
}

func (s *magicLinkService) createLink(ctx context.Context, userID uuid.UUID, purpose MagicLinkPurpose) (*MagicLink, error) {
	// H5: invalidate any prior unused links for this user before issuing a new
	// one. Prevents stale links from coexisting with the fresh one and being
	// replayed during the brief window before they naturally expire.
	if _, err := s.magicLinks.InvalidateUnusedForUser(ctx, userID); err != nil {
		return nil, fmt.Errorf("invalidate prior magic links: %w", err)
	}
	token, err := GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	ttl := s.cfg.MagicLinkTTL
	if purpose == MagicLinkPurposeReset && s.cfg.MagicLinkResetTTL > 0 {
		ttl = s.cfg.MagicLinkResetTTL
	}
	link := &MagicLink{
		ID:        uuid.New(),
		Token:     token,
		UserID:    userID,
		Purpose:   purpose,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
	if err := s.magicLinks.Create(ctx, link); err != nil {
		return nil, fmt.Errorf("create magic link: %w", err)
	}
	return link, nil
}

func (s *magicLinkService) Validate(ctx context.Context, token string) (*MagicLink, MagicLinkStatus, error) {
	link, err := s.magicLinks.GetByToken(ctx, token)
	if err != nil {
		return nil, "", ErrMagicLinkNotFound
	}

	status := link.Status()
	return link, status, nil
}

func (s *magicLinkService) CompleteOnboarding(ctx context.Context, token, password, fullName string) (*TokenPair, error) {
	link, err := s.magicLinks.GetByToken(ctx, token)
	if err != nil {
		return nil, ErrMagicLinkNotFound
	}

	switch link.Status() {
	case MagicLinkStatusExpired:
		return nil, ErrMagicLinkExpired
	case MagicLinkStatusUsed:
		return nil, ErrMagicLinkUsed
	}

	// Load user to validate purpose/status pairing BEFORE marking used —
	// avoid burning a valid token on a mismatch error.
	u, err := s.users.Get(ctx, link.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	purpose := link.Purpose
	if purpose == "" {
		// Backfill safety: pre-migration links have no purpose; treat as invite
		// (matches DB DEFAULT and historical behavior).
		purpose = MagicLinkPurposeInvite
	}

	// Enforce purpose ↔ status pairing — H2 defense-in-depth.
	switch purpose {
	case MagicLinkPurposeInvite:
		if u.Status != user.UserStatusInvited {
			return nil, ErrMagicLinkWrongPurpose
		}
		// L3: password is optional during invite, but if provided it must be strong.
		if password != "" {
			if vErr := ValidatePassword(password); vErr != nil {
				return nil, vErr
			}
		}
	case MagicLinkPurposeReset:
		if u.Status != user.UserStatusActive {
			return nil, ErrMagicLinkWrongPurpose
		}
		// L3: reset MUST provide a strong password.
		if vErr := ValidatePassword(password); vErr != nil {
			return nil, vErr
		}
	case MagicLinkPurposeLogin:
		if u.Status != user.UserStatusActive {
			return nil, ErrMagicLinkWrongPurpose
		}
	default:
		return nil, ErrMagicLinkWrongPurpose
	}

	// Mark link as used (atomic — prevents replay of the same token)
	if err := s.magicLinks.MarkUsed(ctx, link.ID); err != nil {
		return nil, fmt.Errorf("mark used: %w", err)
	}

	switch purpose {
	case MagicLinkPurposeInvite:
		// Activate the invited user (password optional during invite — they can set it later)
		var hash string
		if password != "" {
			var hashErr error
			hash, hashErr = HashPassword(password, s.cfg.BcryptCost)
			if hashErr != nil {
				return nil, fmt.Errorf("hash password: %w", hashErr)
			}
		}
		u, err = s.users.Activate(ctx, link.UserID, user.ActivateUserInput{
			PasswordHash: hash,
			FullName:     fullName,
		})
		if err != nil {
			return nil, fmt.Errorf("activate user: %w", err)
		}
	case MagicLinkPurposeReset:
		// C1 fix: actually update the password for an active user.
		hash, hashErr := HashPassword(password, s.cfg.BcryptCost)
		if hashErr != nil {
			return nil, fmt.Errorf("hash password: %w", hashErr)
		}
		if resetErr := s.users.ResetPassword(ctx, u.ID, hash); resetErr != nil {
			return nil, fmt.Errorf("reset password: %w", resetErr)
		}
	case MagicLinkPurposeLogin:
		// No mutation — passwordless re-auth path.
	}

	// Issue tokens + create session
	claims := &Claims{
		Sub:   u.ID,
		Email: u.Email,
		Role:  u.Role,
	}

	tp, err := NewTokenPair(claims, s.cfg)
	if err != nil {
		return nil, fmt.Errorf("create token pair: %w", err)
	}

	session := &Session{
		ID:           uuid.New(),
		UserID:       u.ID,
		RefreshToken: tp.RefreshToken,
		JTI:          claims.JTI,
		ExpiresAt:    tp.RefreshExpiresAt,
		CreatedAt:    time.Now(),
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	if s.cache != nil {
		_ = s.cache.SetSession(ctx, session)
	}

	return tp, nil
}
