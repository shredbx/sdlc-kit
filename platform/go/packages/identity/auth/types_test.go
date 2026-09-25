package auth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/shredbx/sbx-core/pkg/auth"
)

// SC4: a zero RefreshReuseLeeway is normalized UP to the documented 30s grace
// (Okta default) so rotation grace is always on out of the box — a zero window
// would make every benign concurrent refresh a full-family logout, the bug this
// fixes (2607-125). An explicit non-zero value is preserved.
func TestAuthConfig_Defaults_RefreshReuseLeeway(t *testing.T) {
	var zero auth.AuthConfig
	zero.Defaults()
	assert.Equal(t, 30*time.Second, zero.RefreshReuseLeeway, "zero leeway must default to 30s")

	explicit := auth.AuthConfig{RefreshReuseLeeway: 10 * time.Second}
	explicit.Defaults()
	assert.Equal(t, 10*time.Second, explicit.RefreshReuseLeeway, "explicit leeway must be preserved")
}

func TestMagicLinkStatus(t *testing.T) {
	t.Run("T-MLS-1: valid link returns Valid", func(t *testing.T) {
		ml := testMagicLink(auth.MagicLinkStatusValid)
		assert.Equal(t, auth.MagicLinkStatusValid, ml.Status())
	})

	t.Run("T-MLS-2: used link returns Used", func(t *testing.T) {
		ml := testMagicLink(auth.MagicLinkStatusUsed)
		assert.Equal(t, auth.MagicLinkStatusUsed, ml.Status())
	})

	t.Run("T-MLS-3: expired link returns Expired", func(t *testing.T) {
		ml := testMagicLink(auth.MagicLinkStatusExpired)
		assert.Equal(t, auth.MagicLinkStatusExpired, ml.Status())
	})
}
