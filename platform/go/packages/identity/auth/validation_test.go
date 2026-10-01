package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/shredbx/sbx-core/pkg/auth"
)

// TestAuthConfig_Validate pins F2 (scope-10): a short HS256 secret is offline-
// bruteable, so config construction must reject secrets under 32 bytes (and empty).
func TestAuthConfig_Validate(t *testing.T) {
	base := auth.AuthConfig{
		JWTSecret:       strings.Repeat("x", 32),
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	t.Run("T-CV-1: a >=32-byte secret passes", func(t *testing.T) {
		assert.NoError(t, base.Validate())
	})

	t.Run("T-CV-2: empty secret is rejected", func(t *testing.T) {
		cfg := base
		cfg.JWTSecret = ""
		assert.ErrorIs(t, cfg.Validate(), auth.ErrEmptyJWTSecret)
	})

	t.Run("T-CV-3: a short secret is rejected", func(t *testing.T) {
		cfg := base
		cfg.JWTSecret = "too-short-secret" // 16 bytes
		assert.ErrorIs(t, cfg.Validate(), auth.ErrJWTSecretTooShort)
	})

	t.Run("T-CV-4: boundary — 31 rejected, 32 accepted", func(t *testing.T) {
		cfg := base
		cfg.JWTSecret = strings.Repeat("a", 31)
		assert.ErrorIs(t, cfg.Validate(), auth.ErrJWTSecretTooShort)
		cfg.JWTSecret = strings.Repeat("a", 32)
		assert.NoError(t, cfg.Validate())
	})
}

func TestValidateEmail(t *testing.T) {
	t.Run("T-VE-1: valid email passes", func(t *testing.T) {
		assert.NoError(t, auth.ValidateEmail("user@example.com"))
	})

	t.Run("T-VE-2: empty email fails", func(t *testing.T) {
		assert.Error(t, auth.ValidateEmail(""))
	})

	t.Run("T-VE-3: no @ symbol fails", func(t *testing.T) {
		assert.Error(t, auth.ValidateEmail("notanemail"))
	})

	t.Run("T-VE-4: no domain fails", func(t *testing.T) {
		assert.Error(t, auth.ValidateEmail("user@"))
	})
}

func TestValidatePassword(t *testing.T) {
	t.Run("T-VPW-1: strong password passes", func(t *testing.T) {
		assert.NoError(t, auth.ValidatePassword("MyStr0ng!Pass"))
	})

	t.Run("T-VPW-2: empty password fails", func(t *testing.T) {
		assert.Error(t, auth.ValidatePassword(""))
	})

	t.Run("T-VPW-3: too short password fails (< 8 chars)", func(t *testing.T) {
		assert.Error(t, auth.ValidatePassword("Ab1!"))
	})

	t.Run("T-VPW-4: exceeds 72 bytes fails", func(t *testing.T) {
		longPass := make([]byte, 73)
		for i := range longPass {
			longPass[i] = 'a'
		}
		assert.Error(t, auth.ValidatePassword(string(longPass)))
	})
}

// TC-M6: ValidateRole requires an explicit allowlist — no implicit default.
// Each caller must pass its project-specific roles, making the security boundary explicit.
func TestValidateRole(t *testing.T) {
	brRoles := []string{"super-admin", "admin", "agent"}

	t.Run("T-VR-1: admin valid with explicit BR roles", func(t *testing.T) {
		assert.NoError(t, auth.ValidateRole("admin", brRoles))
	})

	t.Run("T-VR-2: agent valid with explicit BR roles", func(t *testing.T) {
		assert.NoError(t, auth.ValidateRole("agent", brRoles))
	})

	t.Run("T-VR-3: super-admin valid with explicit BR roles", func(t *testing.T) {
		assert.NoError(t, auth.ValidateRole("super-admin", brRoles))
	})

	t.Run("T-VR-4: unknown role rejected", func(t *testing.T) {
		assert.Error(t, auth.ValidateRole("editor", brRoles))
	})

	t.Run("T-VR-5: empty role rejected", func(t *testing.T) {
		assert.Error(t, auth.ValidateRole("", brRoles))
	})

	t.Run("T-VR-6: empty allowlist rejects ALL roles (M6: no implicit default)", func(t *testing.T) {
		assert.Error(t, auth.ValidateRole("admin", nil))
		assert.Error(t, auth.ValidateRole("admin", []string{}))
	})
}

// TC-L3: common weak passwords are rejected.
func TestValidatePassword_RejectsCommonWeak(t *testing.T) {
	weak := []string{"password", "Password", "PASSWORD", "password123",
		"12345678", "qwerty123", "admin", "letmein", "test-password-123"}
	for _, w := range weak {
		assert.Error(t, auth.ValidatePassword(w),
			"common weak password %q must be rejected", w)
	}
}
