package visitoractivity_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	va "github.com/shredbx/sbx-core/pkg/visitoractivity"
)

// TC-A3-1: same UTC day → same non-empty salt (cached, not re-rolled).
func TestMemorySaltProvider_SameDayStable(t *testing.T) {
	day := "2026-05-27"
	p := va.NewMemorySaltProvider(func() string { return day })
	ctx := context.Background()

	s1, err := p.Salt(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, s1)

	s2, err := p.Salt(ctx)
	require.NoError(t, err)
	assert.Equal(t, s1, s2, "same day must return the same cached salt")
}

// TC-A3-2: next UTC day → different salt (rotation).
func TestMemorySaltProvider_RotatesNextDay(t *testing.T) {
	day := "2026-05-27"
	p := va.NewMemorySaltProvider(func() string { return day })
	ctx := context.Background()

	s1, err := p.Salt(ctx)
	require.NoError(t, err)

	day = "2026-05-28"
	s2, err := p.Salt(ctx)
	require.NoError(t, err)

	assert.NotEqual(t, s1, s2, "advancing the day must rotate the salt")
	assert.NotEmpty(t, s2)
}

// TC-A3-3: nil day-fn falls back to a real UTC date (non-empty, deterministic
// within the same call window).
func TestMemorySaltProvider_NilDayFnUsesUTC(t *testing.T) {
	p := va.NewMemorySaltProvider(nil)
	ctx := context.Background()

	s1, err := p.Salt(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, s1)

	s2, err := p.Salt(ctx)
	require.NoError(t, err)
	assert.Equal(t, s1, s2, "same real day must be stable")
}

// TC-A3-4: both providers satisfy the SaltProvider interface. RedisSaltProvider
// behaviour (SETNX-seed + GET, 48h TTL) is covered by the integration layer —
// no test redis is wired here and adding one would introduce a new dependency.
func TestSaltProviders_ImplementInterface(t *testing.T) {
	var _ va.SaltProvider = va.NewMemorySaltProvider(nil)
	var _ va.SaltProvider = va.NewRedisSaltProvider(nil, nil)
}
