package auth_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/shredbx/sbx-core/pkg/auth"
)

func TestHashPassword(t *testing.T) {
	t.Run("T-HP-1: valid password produces bcrypt hash", func(t *testing.T) {
		hash, err := auth.HashPassword("StrongP@ss1", 4)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("StrongP@ss1")))
	})

	t.Run("T-HP-2: empty password returns error", func(t *testing.T) {
		_, err := auth.HashPassword("", 4)
		assert.ErrorIs(t, err, auth.ErrEmptyPassword)
	})

	t.Run("T-HP-3: password exceeding 72 bytes returns error", func(t *testing.T) {
		longPass := strings.Repeat("a", 73)
		_, err := auth.HashPassword(longPass, 4)
		assert.ErrorIs(t, err, auth.ErrPasswordTooLong)
	})

	t.Run("T-HP-4: low cost produces valid hash", func(t *testing.T) {
		hash, err := auth.HashPassword("valid", 4)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})
}

func TestVerifyPassword(t *testing.T) {
	hash, _ := auth.HashPassword("correct", 4)

	t.Run("T-VP-1: matching password returns nil", func(t *testing.T) {
		err := auth.VerifyPassword(hash, "correct")
		assert.NoError(t, err)
	})

	t.Run("T-VP-2: wrong password returns error", func(t *testing.T) {
		err := auth.VerifyPassword(hash, "wrong")
		assert.Error(t, err)
	})

	t.Run("T-VP-4: invalid hash format returns error", func(t *testing.T) {
		err := auth.VerifyPassword("not-a-hash", "any")
		assert.Error(t, err)
	})
}
