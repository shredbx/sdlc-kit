package visitoractivity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	va "github.com/shredbx/sbx-core/pkg/visitoractivity"
)

// TC-A1-1: EventType.Valid recognises exactly the three allowed event types.
func TestEventType_Valid(t *testing.T) {
	cases := []struct {
		in   va.EventType
		want bool
	}{
		{va.EventPropertyView, true},
		{va.EventListingView, true},
		{va.EventPageView, true},
		{va.EventClick, true},
		{va.EventType("property_view"), true},
		{va.EventType("page_view"), true},
		{va.EventType(""), false},
		{va.EventType("unknown"), false},
		{va.EventType("PROPERTY_VIEW"), false},
		{va.EventType("PAGE_VIEW"), false},
		{va.EventType("view"), false},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, c.in.Valid(), "EventType(%q).Valid()", string(c.in))
	}
}

// TC-A1-2: TargetType.Valid recognises exactly the two allowed target types.
func TestTargetType_Valid(t *testing.T) {
	cases := []struct {
		in   va.TargetType
		want bool
	}{
		{va.TargetProperty, true},
		{va.TargetListing, true},
		{va.TargetPage, true},
		{va.TargetType("property"), true},
		{va.TargetType("page"), true},
		{va.TargetType(""), false},
		{va.TargetType("user"), false},
		{va.TargetType("PROPERTY"), false},
		{va.TargetType("PAGE"), false},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, c.in.Valid(), "TargetType(%q).Valid()", string(c.in))
	}
}

// TC-A1-3: Const string values match the DB CHECK-constraint literals exactly.
func TestConstStringValues(t *testing.T) {
	assert.Equal(t, "property_view", string(va.EventPropertyView))
	assert.Equal(t, "listing_view", string(va.EventListingView))
	assert.Equal(t, "page_view", string(va.EventPageView))
	assert.Equal(t, "click", string(va.EventClick))
	assert.Equal(t, "property", string(va.TargetProperty))
	assert.Equal(t, "listing", string(va.TargetListing))
	assert.Equal(t, "page", string(va.TargetPage))
}

// TC-A1-4: DeviceType constants carry their canonical lowercase labels.
func TestDeviceTypeValues(t *testing.T) {
	assert.Equal(t, "mobile", string(va.DeviceMobile))
	assert.Equal(t, "tablet", string(va.DeviceTablet))
	assert.Equal(t, "desktop", string(va.DeviceDesktop))
	assert.Equal(t, "bot", string(va.DeviceBot))
	assert.Equal(t, "unknown", string(va.DeviceUnknown))
}
