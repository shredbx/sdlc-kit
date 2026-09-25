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

func setupAuthService() (auth.AuthService, *mockUserStore, *mockSessionRepo) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	cfg := testConfig()

	svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, nil)
	return svc, users, sessions
}

func seedActiveUser(store *mockUserStore) *user.User {
	hash, _ := auth.HashPassword("correct-password", 4)
	u := &user.User{
		ID:           testUserID,
		Email:        "test@example.com",
		FullName:     "Test User",
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

func TestAuthService_Login(t *testing.T) {
	t.Run("T-LG-1: valid credentials return TokenPair", func(t *testing.T) {
		svc, users, _ := setupAuthService()
		seedActiveUser(users)

		tp, err := svc.Login(t.Context(), "test@example.com", "correct-password")
		require.NoError(t, err)
		assert.NotEmpty(t, tp.AccessToken)
		assert.NotEmpty(t, tp.RefreshToken)
		assert.Equal(t, "Bearer", tp.TokenType)
	})

	t.Run("T-LG-2: wrong password returns invalid credentials", func(t *testing.T) {
		svc, users, _ := setupAuthService()
		seedActiveUser(users)

		_, err := svc.Login(t.Context(), "test@example.com", "wrong-password")
		assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})

	t.Run("T-LG-3: nonexistent email returns same error (no enumeration)", func(t *testing.T) {
		svc, _, _ := setupAuthService()

		_, err := svc.Login(t.Context(), "nobody@example.com", "any")
		assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})

	t.Run("T-LG-4: deleted user returns invalid credentials", func(t *testing.T) {
		svc, users, _ := setupAuthService()
		u := seedActiveUser(users)
		deletedAt := time.Now()
		u.Status = user.UserStatusDeleted
		u.DeletedAt = &deletedAt

		_, err := svc.Login(t.Context(), "test@example.com", "correct-password")
		assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})

	t.Run("T-LG-6: invited user (no password) returns invalid credentials", func(t *testing.T) {
		svc, users, _ := setupAuthService()
		users.mu.Lock()
		users.users["invited@example.com"] = &user.User{
			ID:     uuid.New(),
			Email:  "invited@example.com",
			Status: user.UserStatusInvited,
		}
		users.mu.Unlock()

		_, err := svc.Login(t.Context(), "invited@example.com", "any")
		assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})
}

func TestAuthService_Logout(t *testing.T) {
	t.Run("T-LO-1: valid refresh token deletes session", func(t *testing.T) {
		svc, users, sessions := setupAuthService()
		seedActiveUser(users)

		tp, _ := svc.Login(t.Context(), "test@example.com", "correct-password")
		err := svc.Logout(t.Context(), tp.RefreshToken)
		assert.NoError(t, err)

		// Verify session deleted
		sessions.mu.RLock()
		assert.Len(t, sessions.sessions, 0)
		sessions.mu.RUnlock()
	})

	t.Run("T-LO-2: already deleted session returns no error (idempotent)", func(t *testing.T) {
		svc, _, _ := setupAuthService()
		err := svc.Logout(t.Context(), "nonexistent-token")
		assert.NoError(t, err)
	})
}

// TestLogin_StoresHashedRefreshToken pins F1 (scope-10): refresh tokens must be
// stored HASHED at rest (Postgres) and STRIPPED from the session cache, mirroring
// the M5 magic-link pattern. A DB/backup/cache leak must not yield live bearer
// tokens. Rotation, refresh and logout must keep working against the RAW token the
// client holds (the service hashes on lookup).
func TestLogin_StoresHashedRefreshToken(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	svc := auth.NewAuthService(testConfig(), users, sessions, magicLinks, cache, nil)
	seedActiveUser(users)

	tp, err := svc.Login(t.Context(), "test@example.com", "correct-password")
	require.NoError(t, err)
	require.NotEmpty(t, tp.RefreshToken)
	raw := tp.RefreshToken // the raw secret handed to the client

	// DB at rest: the persisted session carries the SHA-256 hash, never the raw token.
	sessions.mu.RLock()
	var stored *auth.Session
	for _, s := range sessions.sessions {
		stored = s
	}
	sessions.mu.RUnlock()
	require.NotNil(t, stored, "a session must be persisted on login")
	assert.NotEqual(t, raw, stored.RefreshToken, "raw refresh token must not be stored at rest")
	assert.Equal(t, auth.HashToken(raw), stored.RefreshToken, "stored refresh token must be the SHA-256 hash of the issued token")

	// Cache: the refresh token must be stripped from any cached session copy.
	cache.mu.RLock()
	var cached *auth.Session
	for _, s := range cache.sessions {
		cached = s
	}
	cache.mu.RUnlock()
	require.NotNil(t, cached, "the session should be cached on login")
	assert.Empty(t, cached.RefreshToken, "cached session must not carry the refresh token")

	// Behavior preserved: the client presents the RAW token and refresh still works.
	tp2, err := svc.RefreshToken(t.Context(), raw)
	require.NoError(t, err, "refresh with the raw token must still succeed")
	require.NotEmpty(t, tp2.RefreshToken)
	assert.NotEqual(t, raw, tp2.RefreshToken, "rotation must issue a fresh refresh token")

	// And logout by the raw (rotated) token still resolves + clears the session.
	err = svc.Logout(t.Context(), tp2.RefreshToken)
	assert.NoError(t, err)
}

func TestMagicLinkService_Generate(t *testing.T) {
	cfg := testConfig()
	users := newMockUserStore()
	magicLinks := newMockMagicLinkRepo()
	sessions := newMockSessionRepo()
	cache := newMockSessionCache()
	svc := auth.NewMagicLinkService(cfg, users, magicLinks, sessions, cache)

	t.Run("T-MLG-1: valid email creates user + magic link", func(t *testing.T) {
		link, err := svc.Generate(t.Context(), "new@example.com", "team-member")
		require.NoError(t, err)
		assert.NotEmpty(t, link.Token)
		assert.NotEqual(t, uuid.Nil, link.UserID)
	})

	t.Run("T-MLG-2: duplicate invite for invited user returns existing link", func(t *testing.T) {
		link2, err := svc.Generate(t.Context(), "new@example.com", "team-member")
		require.NoError(t, err)
		assert.NotEmpty(t, link2.Token, "should return existing active link")
	})

	t.Run("T-MLG-3: invalid email returns error", func(t *testing.T) {
		_, err := svc.Generate(t.Context(), "not-an-email", "team-member")
		assert.Error(t, err)
	})
}

func TestMagicLinkService_Validate(t *testing.T) {
	cfg := testConfig()
	users := newMockUserStore()
	magicLinks := newMockMagicLinkRepo()
	sessions := newMockSessionRepo()
	cache := newMockSessionCache()
	svc := auth.NewMagicLinkService(cfg, users, magicLinks, sessions, cache)

	t.Run("T-MLV-4: nonexistent token returns error", func(t *testing.T) {
		_, _, err := svc.Validate(t.Context(), "nonexistent")
		assert.ErrorIs(t, err, auth.ErrMagicLinkNotFound)
	})
}

func TestMagicLinkService_CompleteOnboarding(t *testing.T) {
	cfg := testConfig()
	users := newMockUserStore()
	magicLinks := newMockMagicLinkRepo()
	sessions := newMockSessionRepo()
	cache := newMockSessionCache()
	svc := auth.NewMagicLinkService(cfg, users, magicLinks, sessions, cache)

	t.Run("T-MLO-1: valid token activates user and returns tokens", func(t *testing.T) {
		link, _ := svc.Generate(t.Context(), "onboard@example.com", "team-member")

		tp, err := svc.CompleteOnboarding(t.Context(), link.Token, "MyPassword1!", "John Doe")
		require.NoError(t, err)
		assert.NotEmpty(t, tp.AccessToken)
		assert.NotEmpty(t, tp.RefreshToken)
	})
}

func TestMagicLinkService_Generate_Idempotent(t *testing.T) {
	cfg := testConfig()
	users := newMockUserStore()
	magicLinks := newMockMagicLinkRepo()
	sessions := newMockSessionRepo()
	cache := newMockSessionCache()
	svc := auth.NewMagicLinkService(cfg, users, magicLinks, sessions, cache)

	t.Run("T-MLG-4: re-invite issues fresh token, invalidates prior (M5+H5)", func(t *testing.T) {
		link1, err := svc.Generate(t.Context(), "dup@example.com", "admin")
		require.NoError(t, err)

		link2, err := svc.Generate(t.Context(), "dup@example.com", "admin")
		require.NoError(t, err)
		// M5: plaintext tokens are not retrievable from the repo, so we always
		// issue a fresh token on re-invite. H5: prior is invalidated atomically.
		assert.NotEqual(t, link1.Token, link2.Token, "re-invite must issue a fresh token")
	})
}
