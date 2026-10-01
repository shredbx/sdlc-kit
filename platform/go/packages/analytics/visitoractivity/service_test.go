package visitoractivity_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	va "github.com/shredbx/sbx-core/pkg/visitoractivity"
)

// captureStore is a fake Store that records the last appended Event. The
// aggregation methods return zero values — Record never calls them.
type captureStore struct {
	last   *va.Event
	events []*va.Event
}

func (c *captureStore) Append(_ context.Context, e *va.Event) error {
	cp := *e
	c.last = &cp
	c.events = append(c.events, &cp)
	return nil
}

func (c *captureStore) TopProperties(context.Context, time.Time, int) ([]va.TargetStat, error) {
	return nil, nil
}

func (c *captureStore) PropertyStats(context.Context, uuid.UUID, time.Time, ...string) (*va.PropertyStats, error) {
	return &va.PropertyStats{}, nil
}

func (c *captureStore) Overview(context.Context, time.Time, ...string) (*va.Overview, error) {
	return &va.Overview{}, nil
}

func (c *captureStore) PathStats(context.Context, string, time.Time, ...string) (*va.PropertyStats, error) {
	return &va.PropertyStats{}, nil
}

func (c *captureStore) PathSummary(context.Context, string) (va.PathSummary, error) {
	return va.PathSummary{}, nil
}

// fixedSalt is a SaltProvider that always returns the same salt.
type fixedSalt struct{ s string }

func (f fixedSalt) Salt(context.Context) (string, error) { return f.s, nil }

func newTestService(salt string) (*va.Service, *captureStore) {
	store := &captureStore{}
	svc := va.NewService(store, fixedSalt{s: salt})
	return svc, store
}

const (
	testIP = "203.0.113.7"
	testUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)"
)

func baseInput() va.RecordInput {
	tid := uuid.New()
	return va.RecordInput{
		EventType:  va.EventPropertyView,
		TargetType: va.TargetProperty,
		TargetID:   &tid,
		Path:       "/p/123",
		IP:         testIP,
		UserAgent:  testUA,
	}
}

// TC-A5-1: an unrecognised event type is rejected with ErrInvalidEvent.
func TestRecord_InvalidEventType(t *testing.T) {
	svc, _ := newTestService("salt")
	in := baseInput()
	in.EventType = va.EventType("bogus")
	err := svc.Record(context.Background(), in)
	assert.ErrorIs(t, err, va.ErrInvalidEvent)
}

// TC-A5-1b: an unrecognised target type is rejected with ErrInvalidEvent.
func TestRecord_InvalidTargetType(t *testing.T) {
	svc, _ := newTestService("salt")
	in := baseInput()
	in.TargetType = va.TargetType("user")
	err := svc.Record(context.Background(), in)
	assert.ErrorIs(t, err, va.ErrInvalidEvent)
}

// TC-A5-2: a property target with a nil id is rejected with ErrTargetRequired.
func TestRecord_PropertyNilTarget(t *testing.T) {
	svc, _ := newTestService("salt")
	in := baseInput()
	in.TargetID = nil
	err := svc.Record(context.Background(), in)
	assert.ErrorIs(t, err, va.ErrTargetRequired)
}

// TC-A5-3: props that marshal larger than the cap are rejected with ErrPropsTooLarge.
func TestRecord_PropsTooLarge(t *testing.T) {
	svc, _ := newTestService("salt")
	in := baseInput()
	in.Props = map[string]any{"blob": strings.Repeat("x", 3000)}
	err := svc.Record(context.Background(), in)
	assert.ErrorIs(t, err, va.ErrPropsTooLarge)
}

// TC-A5-4: the derived VisitorID equals ComputeVisitorID(salt, ip, ua); raw
// ip/ua never appear on the captured Event.
func TestRecord_DerivesVisitorID_NoRawPII(t *testing.T) {
	salt := "the-daily-salt"
	svc, store := newTestService(salt)
	in := baseInput()
	require.NoError(t, svc.Record(context.Background(), in))
	require.NotNil(t, store.last)

	want := va.ComputeVisitorID(salt, testIP, testUA)
	assert.Equal(t, want, store.last.VisitorID)

	// No field of the stored Event may contain the raw ip or ua.
	assert.NotContains(t, store.last.VisitorID, testIP)
	assert.NotContains(t, store.last.Path, testIP)
	assert.NotContains(t, store.last.RefererOrigin, testIP)
	for k, v := range store.last.Props {
		assert.NotContains(t, k, testUA)
		assert.NotContains(t, k, testIP)
		if s, ok := v.(string); ok {
			assert.NotContains(t, s, testUA)
			assert.NotContains(t, s, testIP)
		}
	}

	// Symmetric, whole-Event invariant (data model §4): the rendered Event —
	// every field, not just the ones spot-checked above — must contain NEITHER
	// the raw IP NOR the raw user-agent anywhere. This locks "raw IP and UA are
	// never persisted" for the full serialized row.
	raw, err := json.Marshal(store.last)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), testIP, "serialized Event must not contain the raw IP")
	assert.NotContains(t, string(raw), testUA, "serialized Event must not contain the raw user-agent")
}

// TC-A5-5: DeviceType and IsBot are derived from the user-agent.
func TestRecord_ClassifiesDevice(t *testing.T) {
	svc, store := newTestService("salt")
	in := baseInput() // iPhone UA → mobile, not a bot
	require.NoError(t, svc.Record(context.Background(), in))
	assert.Equal(t, va.DeviceMobile, store.last.DeviceType)
	assert.False(t, store.last.IsBot)

	in2 := baseInput()
	in2.UserAgent = "Googlebot/2.1"
	require.NoError(t, svc.Record(context.Background(), in2))
	assert.Equal(t, va.DeviceBot, store.last.DeviceType)
	assert.True(t, store.last.IsBot)
}

// TC-A5-6: the referer is sanitized to scheme://host, dropping path + query.
func TestRecord_SanitizesReferer(t *testing.T) {
	svc, store := newTestService("salt")
	in := baseInput()
	in.RefererRaw = "https://g.com/x?token=abc"
	require.NoError(t, svc.Record(context.Background(), in))
	assert.Equal(t, "https://g.com", store.last.RefererOrigin)
}

// TC-A5-6b: a non-URL / empty referer sanitizes to "".
func TestRecord_SanitizesReferer_Garbage(t *testing.T) {
	svc, store := newTestService("salt")
	cases := []string{"", "not a url", "javascript:alert(1)"}
	for _, raw := range cases {
		in := baseInput()
		in.RefererRaw = raw
		require.NoError(t, svc.Record(context.Background(), in))
		assert.Emptyf(t, store.last.RefererOrigin, "referer %q must sanitize to empty", raw)
	}
}

// TC-A5-7: two Record calls with the same salt/ip/ua yield an identical VisitorID.
func TestRecord_StableVisitorIDAcrossCalls(t *testing.T) {
	svc, store := newTestService("salt")
	in := baseInput()
	require.NoError(t, svc.Record(context.Background(), in))
	first := store.last.VisitorID

	in2 := baseInput() // same ip/ua, different (random) target id
	require.NoError(t, svc.Record(context.Background(), in2))
	second := store.last.VisitorID

	assert.Equal(t, first, second, "same salt/ip/ua must produce the same visitor id")
}

// TC-A5-8: a valid listing view with a nil target is accepted (only property
// requires a target).
func TestRecord_ListingNilTargetOK(t *testing.T) {
	svc, store := newTestService("salt")
	in := va.RecordInput{
		EventType:  va.EventListingView,
		TargetType: va.TargetListing,
		TargetID:   nil,
		Path:       "/listings",
		IP:         testIP,
		UserAgent:  testUA,
	}
	require.NoError(t, svc.Record(context.Background(), in))
	require.NotNil(t, store.last)
	assert.Nil(t, store.last.TargetID)
}

// pageInput is the base input for a valid page view: keyed by Path, no TargetID.
func pageInput(path string) va.RecordInput {
	return va.RecordInput{
		EventType:  va.EventPageView,
		TargetType: va.TargetPage,
		TargetID:   nil,
		Path:       path,
		IP:         testIP,
		UserAgent:  testUA,
	}
}

// TC-A5-9: a valid page view (non-empty path, nil target id) is accepted — the
// path IS the identifier, so no TargetID is required.
func TestRecord_PageOK_NoTarget(t *testing.T) {
	svc, store := newTestService("salt")
	require.NoError(t, svc.Record(context.Background(), pageInput("/sell")))
	require.NotNil(t, store.last)
	assert.Equal(t, va.EventPageView, store.last.EventType)
	assert.Equal(t, va.TargetPage, store.last.TargetType)
	assert.Equal(t, "/sell", store.last.Path)
	assert.Nil(t, store.last.TargetID, "a page is keyed by path, not a target id")
}

// TC-A5-9b: the root path "/" is a valid page identifier.
func TestRecord_PageOK_RootPath(t *testing.T) {
	svc, store := newTestService("salt")
	require.NoError(t, svc.Record(context.Background(), pageInput("/")))
	require.NotNil(t, store.last)
	assert.Equal(t, "/", store.last.Path)
}

// TC-A5-10: a page event with an empty path is rejected with ErrPathRequired
// (the path is the identifier — it must be present).
func TestRecord_PageEmptyPathRejected(t *testing.T) {
	svc, _ := newTestService("salt")
	in := pageInput("")
	err := svc.Record(context.Background(), in)
	assert.ErrorIs(t, err, va.ErrPathRequired)
}

// TC-A5-11: a page event that ALSO carries a target id is rejected with
// ErrPathRequired (a page is keyed by its path, never a UUID).
func TestRecord_PageWithTargetIDRejected(t *testing.T) {
	svc, _ := newTestService("salt")
	in := pageInput("/sell")
	tid := uuid.New()
	in.TargetID = &tid
	err := svc.Record(context.Background(), in)
	assert.ErrorIs(t, err, va.ErrPathRequired)
}
