package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shredbx/sbx-core/pkg/auth"
	"github.com/shredbx/sbx-core/pkg/user"
)

func setupMagicLinkService() (auth.MagicLinkService, *mockUserStore, *mockMagicLinkRepo, *mockSessionRepo) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	cfg := testConfig()

	svc := auth.NewMagicLinkService(cfg, users, magicLinks, sessions, cache)
	return svc, users, magicLinks, sessions
}

func seedInvitedUser(store *mockUserStore, email string) *user.User {
	u := &user.User{
		ID:        uuid.New(),
		Email:     email,
		FullName:  "Pending User",
		Role:      "admin",
		Status:    user.UserStatusInvited,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.mu.Lock()
	store.users[u.Email] = u
	store.mu.Unlock()
	return u
}

func seedActiveUserWithPassword(store *mockUserStore, email, plaintext string) *user.User {
	hash, _ := auth.HashPassword(plaintext, 4)
	u := &user.User{
		ID:           uuid.New(),
		Email:        email,
		FullName:     "Active User",
		PasswordHash: hash,
		Role:         "admin",
		Status:       user.UserStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.mu.Lock()
	store.users[u.Email] = u
	store.mu.Unlock()
	return u
}

func seedMagicLink(repo *mockMagicLinkRepo, userID uuid.UUID, purpose auth.MagicLinkPurpose, token string) *auth.MagicLink {
	link := &auth.MagicLink{
		ID:        uuid.New(),
		Token:     token,
		UserID:    userID,
		Purpose:   purpose,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	repo.mu.Lock()
	repo.links[token] = link
	repo.mu.Unlock()
	return link
}

// TC-C1: Password reset flow actually updates the password hash for active users.
// Before fix: CompleteOnboarding silently ignored the new password for active users.
// After fix: reset-purpose links call UserStore.ResetPassword.
func TestCompleteOnboarding_ResetFlow_UpdatesPassword(t *testing.T) {
	svc, users, links, _ := setupMagicLinkService()
	u := seedActiveUserWithPassword(users, "active@example.com", "old-password")
	oldHash := u.PasswordHash

	seedMagicLink(links, u.ID, auth.MagicLinkPurposeReset, "reset-token-1")

	tp, err := svc.CompleteOnboarding(t.Context(), "reset-token-1", "new-secure-pwd", "")
	require.NoError(t, err)
	require.NotNil(t, tp)
	assert.NotEmpty(t, tp.AccessToken)

	// PasswordHash must have changed (C1 fix)
	users.mu.RLock()
	updated := users.users["active@example.com"]
	users.mu.RUnlock()
	assert.NotEqual(t, oldHash, updated.PasswordHash, "password_hash must be updated after reset")

	// New password must verify against the new hash
	assert.NoError(t, auth.VerifyPassword(updated.PasswordHash, "new-secure-pwd"))
	// Old password must NOT verify
	assert.Error(t, auth.VerifyPassword(updated.PasswordHash, "old-password"))
}

// TC-C1 regression guard: empty password on reset flow is rejected (can't accidentally clear hash).
func TestCompleteOnboarding_ResetFlow_RejectsEmptyPassword(t *testing.T) {
	svc, users, links, _ := setupMagicLinkService()
	u := seedActiveUserWithPassword(users, "active@example.com", "old-password")
	oldHash := u.PasswordHash

	seedMagicLink(links, u.ID, auth.MagicLinkPurposeReset, "reset-token-empty")

	_, err := svc.CompleteOnboarding(t.Context(), "reset-token-empty", "", "")
	assert.ErrorIs(t, err, auth.ErrEmptyPassword)

	users.mu.RLock()
	current := users.users["active@example.com"]
	users.mu.RUnlock()
	assert.Equal(t, oldHash, current.PasswordHash, "empty-password reset must not modify hash")
}

// TC-H2 positive: invite link on invited user activates correctly (regression — keep existing flow working).
func TestCompleteOnboarding_InviteFlow_ActivatesInvitedUser(t *testing.T) {
	svc, users, links, _ := setupMagicLinkService()
	u := seedInvitedUser(users, "newcomer@example.com")

	seedMagicLink(links, u.ID, auth.MagicLinkPurposeInvite, "invite-token-1")

	tp, err := svc.CompleteOnboarding(t.Context(), "invite-token-1", "first-password", "Newcomer Name")
	require.NoError(t, err)
	require.NotNil(t, tp)

	users.mu.RLock()
	activated := users.users["newcomer@example.com"]
	users.mu.RUnlock()
	assert.Equal(t, user.UserStatusActive, activated.Status)
	assert.NoError(t, auth.VerifyPassword(activated.PasswordHash, "first-password"))
}

// TC-H2: reset-purpose link cannot activate an invited user (cross-purpose replay defense).
func TestCompleteOnboarding_RejectsResetLinkOnInvitedUser(t *testing.T) {
	svc, users, links, _ := setupMagicLinkService()
	u := seedInvitedUser(users, "still-invited@example.com")

	// Mint a reset-purpose link, but the user is still invited — purpose/status mismatch.
	seedMagicLink(links, u.ID, auth.MagicLinkPurposeReset, "reset-token-mismatch")

	_, err := svc.CompleteOnboarding(t.Context(), "reset-token-mismatch", "any-pwd", "")
	assert.ErrorIs(t, err, auth.ErrMagicLinkWrongPurpose)

	// Status must remain invited — no privilege escalation via wrong-purpose token.
	users.mu.RLock()
	current := users.users["still-invited@example.com"]
	users.mu.RUnlock()
	assert.Equal(t, user.UserStatusInvited, current.Status)
}

// TC-H2: invite-purpose link cannot reset an active user's password.
func TestCompleteOnboarding_RejectsInviteLinkOnActiveUser(t *testing.T) {
	svc, users, links, _ := setupMagicLinkService()
	u := seedActiveUserWithPassword(users, "active2@example.com", "real-password")
	oldHash := u.PasswordHash

	// Mint an invite-purpose link pointing at an already-active user — should refuse.
	seedMagicLink(links, u.ID, auth.MagicLinkPurposeInvite, "invite-on-active")

	_, err := svc.CompleteOnboarding(t.Context(), "invite-on-active", "attacker-chosen-pwd", "Mallory")
	assert.ErrorIs(t, err, auth.ErrMagicLinkWrongPurpose)

	users.mu.RLock()
	current := users.users["active2@example.com"]
	users.mu.RUnlock()
	assert.Equal(t, oldHash, current.PasswordHash, "active user's password must not be overwritten by invite-purpose token")
}

// TC-C1 follow-up: empty-password rejection on reset does NOT burn the token;
// user can retry with a real password. Verifies the rejection happens BEFORE MarkUsed.
func TestCompleteOnboarding_EmptyPasswordRetryAllowed(t *testing.T) {
	svc, users, links, _ := setupMagicLinkService()
	u := seedActiveUserWithPassword(users, "retry@example.com", "old-password")

	seedMagicLink(links, u.ID, auth.MagicLinkPurposeReset, "retry-token")

	// First attempt — empty password rejected
	_, err := svc.CompleteOnboarding(t.Context(), "retry-token", "", "")
	assert.ErrorIs(t, err, auth.ErrEmptyPassword)

	// Retry with real password must succeed (token not burned)
	tp, err := svc.CompleteOnboarding(t.Context(), "retry-token", "valid-new-pwd", "")
	require.NoError(t, err)
	require.NotNil(t, tp)

	users.mu.RLock()
	updated := users.users["retry@example.com"]
	users.mu.RUnlock()
	assert.NoError(t, auth.VerifyPassword(updated.PasswordHash, "valid-new-pwd"))
}

// TC-C1/H2 replay protection: a successful reset burns the token; reuse rejected.
func TestCompleteOnboarding_ResetFlow_ReplayRejected(t *testing.T) {
	svc, users, links, _ := setupMagicLinkService()
	u := seedActiveUserWithPassword(users, "replay@example.com", "old")

	seedMagicLink(links, u.ID, auth.MagicLinkPurposeReset, "replay-token")

	// First use succeeds
	_, err := svc.CompleteOnboarding(t.Context(), "replay-token", "new-pwd-1", "")
	require.NoError(t, err)

	// Token is now used — replay must fail. MarkUsed is atomic at the repo layer;
	// our mock's MarkUsed is a no-op so the Status() check is what catches the
	// replay (UsedAt would be set by a real repo). Simulate that here.
	links.mu.Lock()
	now := time.Now()
	links.links["replay-token"].UsedAt = &now
	links.mu.Unlock()

	_, err = svc.CompleteOnboarding(t.Context(), "replay-token", "attacker-pwd", "")
	assert.ErrorIs(t, err, auth.ErrMagicLinkUsed)

	// Password from the first (legit) reset is preserved
	users.mu.RLock()
	current := users.users["replay@example.com"]
	users.mu.RUnlock()
	assert.NoError(t, auth.VerifyPassword(current.PasswordHash, "new-pwd-1"))
}

// TC-H2 backfill safety: pre-migration link with empty Purpose is treated as
// invite (matches DB DEFAULT). Active users still rejected by status pairing.
func TestCompleteOnboarding_EmptyPurpose_TreatedAsInvite(t *testing.T) {
	svc, users, links, _ := setupMagicLinkService()
	u := seedInvitedUser(users, "legacy@example.com")

	// Link from before the migration — Purpose is the zero value
	link := &auth.MagicLink{
		ID:        uuid.New(),
		Token:     "legacy-token",
		UserID:    u.ID,
		Purpose:   "", // legacy / pre-migration
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	links.mu.Lock()
	links.links[link.Token] = link
	links.mu.Unlock()

	tp, err := svc.CompleteOnboarding(t.Context(), "legacy-token", "first-pwd", "Legacy User")
	require.NoError(t, err)
	require.NotNil(t, tp)

	users.mu.RLock()
	activated := users.users["legacy@example.com"]
	users.mu.RUnlock()
	assert.Equal(t, user.UserStatusActive, activated.Status)
}

// TC-H4: Activate is now status-aware via the mock — invite link on an already-active
// user fails BEFORE we reach Activate (purpose mismatch). But if a future bug got past
// the purpose check, the mock's Activate would still refuse — defense-in-depth.
func TestActivateMock_RejectsAlreadyActiveUser(t *testing.T) {
	users := newMockUserStore()
	u := seedActiveUserWithPassword(users, "already-active@example.com", "real-pwd")

	_, err := users.Activate(t.Context(), u.ID, user.ActivateUserInput{
		PasswordHash: "attacker-controlled-hash",
		FullName:     "Mallory",
	})
	assert.ErrorIs(t, err, user.ErrUserAlreadyActive)
}

// TC-H2 login purpose: magic-link login on active user succeeds without mutation.
func TestCompleteOnboarding_LoginPurpose_NoPasswordChange(t *testing.T) {
	svc, users, links, _ := setupMagicLinkService()
	u := seedActiveUserWithPassword(users, "login-link@example.com", "real-pwd")
	originalHash := u.PasswordHash

	seedMagicLink(links, u.ID, auth.MagicLinkPurposeLogin, "login-link-token")

	tp, err := svc.CompleteOnboarding(t.Context(), "login-link-token", "", "")
	require.NoError(t, err)
	require.NotNil(t, tp)
	assert.NotEmpty(t, tp.AccessToken)

	// Password unchanged — login link is re-auth, not mutation
	users.mu.RLock()
	current := users.users["login-link@example.com"]
	users.mu.RUnlock()
	assert.Equal(t, originalHash, current.PasswordHash)
}

// TC-H5: A fresh Generate for a user with prior unused links atomically
// invalidates the priors. Only one valid link can exist per user at a time.
func TestGenerate_InvalidatesPriorUnusedLinks(t *testing.T) {
	svc, users, links, _ := setupMagicLinkService()
	u := seedInvitedUser(users, "h5@example.com")

	// Seed a stale (but not yet expired) link directly — simulates a leftover from
	// a prior session that GetActiveByUserID would still return.
	staleToken := "stale-token-h5"
	staleLink := &auth.MagicLink{
		ID:        uuid.New(),
		Token:     staleToken,
		UserID:    u.ID,
		Purpose:   auth.MagicLinkPurposeInvite,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now().Add(-30 * time.Minute),
	}
	links.mu.Lock()
	links.links[staleToken] = staleLink
	links.mu.Unlock()

	// Force a path that bypasses idempotent return by marking the stale link as used
	// via a different mechanism — actually simpler: delete the link from GetActive's view
	// by setting it as used but keep it findable by token. Simulate the H5 race:
	// expired-but-not-marked-used link exists, then Generate is called.
	links.mu.Lock()
	staleLink.ExpiresAt = time.Now().Add(-1 * time.Hour) // now "expired" — GetActive returns nil
	links.mu.Unlock()

	// Generate a new link — Generate will see no active link and create a new one.
	// H5 fix: it must also invalidate the stale one even though it's expired.
	newLink, err := svc.Generate(t.Context(), "h5@example.com", "admin")
	require.NoError(t, err)
	require.NotEqual(t, staleToken, newLink.Token)

	// The stale link must now be marked used (UsedAt set) so it can't be replayed
	// even if its expiry hasn't logically arrived yet.
	links.mu.RLock()
	staleAfter := links.links[staleToken]
	links.mu.RUnlock()
	require.NotNil(t, staleAfter.UsedAt, "H5: prior unused link must be invalidated when a new one is issued")
}

// M5+H5 regression: a re-invite issues a FRESH token and invalidates the prior.
// (Pre-M5 returned the existing active link; pre-M5 design depended on
// plaintext-token storage which is now hashed. New design: always fresh, prior
// is atomically marked used so only one valid link exists at any time.)
func TestGenerate_ReinviteIssuesFreshAndInvalidatesPrior(t *testing.T) {
	svc, users, links, _ := setupMagicLinkService()
	u := seedInvitedUser(users, "reinvite@example.com")

	// Seed an active (unexpired, unused) invite link
	existingToken := "existing-active"
	existingLink := &auth.MagicLink{
		ID:        uuid.New(),
		Token:     existingToken,
		UserID:    u.ID,
		Purpose:   auth.MagicLinkPurposeInvite,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	links.mu.Lock()
	links.links[existingToken] = existingLink
	links.mu.Unlock()

	// Re-invite — must issue a FRESH token (not the existing one)
	link, err := svc.Generate(t.Context(), "reinvite@example.com", "admin")
	require.NoError(t, err)
	assert.NotEqual(t, existingToken, link.Token, "re-invite must issue a fresh token")

	// Prior link must now be marked used (H5)
	links.mu.RLock()
	prior := links.links[existingToken]
	links.mu.RUnlock()
	require.NotNil(t, prior.UsedAt, "prior link must be invalidated")
}

// Sanity: Generate (invite path) tags links with invite purpose for new users.
func TestGenerate_TagsInvitePurpose(t *testing.T) {
	svc, _, links, _ := setupMagicLinkService()

	link, err := svc.Generate(t.Context(), "fresh@example.com", "admin")
	require.NoError(t, err)
	assert.Equal(t, auth.MagicLinkPurposeInvite, link.Purpose)

	// And the persisted copy has the same purpose
	links.mu.RLock()
	stored := links.links[link.Token]
	links.mu.RUnlock()
	require.NotNil(t, stored)
	assert.Equal(t, auth.MagicLinkPurposeInvite, stored.Purpose)
}
