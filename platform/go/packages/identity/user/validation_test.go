package user_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shredbx/sbx-core/pkg/user"
)

func TestValidateEmail(t *testing.T) {
	t.Run("T-VE-1: valid email", func(t *testing.T) {
		err := user.ValidateEmail("user@example.com")
		assert.NoError(t, err)
	})

	t.Run("T-VE-2: invalid format", func(t *testing.T) {
		err := user.ValidateEmail("not-an-email")
		assert.ErrorIs(t, err, user.ErrInvalidEmail)
	})

	t.Run("T-VE-3: empty string", func(t *testing.T) {
		err := user.ValidateEmail("")
		assert.ErrorIs(t, err, user.ErrInvalidEmail)
	})

	t.Run("T-VE-4: at 255 char boundary", func(t *testing.T) {
		// 255 total: local@domain format
		local := strings.Repeat("a", 243)
		email := local + "@example.com" // 243 + 12 = 255
		err := user.ValidateEmail(email)
		assert.NoError(t, err)
	})

	t.Run("T-VE-5: exceeds 255 chars", func(t *testing.T) {
		local := strings.Repeat("a", 244)
		email := local + "@example.com" // 244 + 12 = 256
		err := user.ValidateEmail(email)
		assert.ErrorIs(t, err, user.ErrEmailTooLong)
	})
}

func TestUserIsActive(t *testing.T) {
	t.Run("active user", func(t *testing.T) {
		u := testActiveUser()
		assert.True(t, u.IsActive())
	})

	t.Run("invited user", func(t *testing.T) {
		u := testInvitedUser()
		assert.False(t, u.IsActive())
	})

	t.Run("deleted user", func(t *testing.T) {
		u := testDeletedUser()
		assert.False(t, u.IsActive())
	})
}
