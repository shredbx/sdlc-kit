package visitoractivity_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	va "github.com/shredbx/sbx-core/pkg/visitoractivity"
)

var hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// TC-A2-1: same inputs → stable id; output is 64 lowercase hex chars.
func TestComputeVisitorID_StableAndShaped(t *testing.T) {
	a := va.ComputeVisitorID("salt-1", "203.0.113.7", "Mozilla/5.0")
	b := va.ComputeVisitorID("salt-1", "203.0.113.7", "Mozilla/5.0")
	assert.Equal(t, a, b, "same inputs must yield same id")
	assert.Regexp(t, hex64, a, "id must be 64 lowercase hex chars (sha256)")
}

// TC-A2-2: salt rotation changes the id for identical ip/ua.
func TestComputeVisitorID_SaltRotationChangesID(t *testing.T) {
	day1 := va.ComputeVisitorID("salt-day-1", "203.0.113.7", "Mozilla/5.0")
	day2 := va.ComputeVisitorID("salt-day-2", "203.0.113.7", "Mozilla/5.0")
	assert.NotEqual(t, day1, day2, "rotating salt must change the id")
}

// TC-A2-3: no delimiter collision — splitting the boundary differently must
// produce a different id (the 0x1f separator prevents ip/ua concatenation
// ambiguity).
func TestComputeVisitorID_NoDelimiterCollision(t *testing.T) {
	x := va.ComputeVisitorID("s", "ab", "c")
	y := va.ComputeVisitorID("s", "a", "bc")
	assert.NotEqual(t, x, y, "ip/ua boundary must not collide")
}

// TC-A2-4: device classification table.
func TestClassifyDevice(t *testing.T) {
	cases := []struct {
		name     string
		ua       string
		wantType va.DeviceType
		wantBot  bool
	}{
		{"empty", "", va.DeviceUnknown, false},
		{"iphone", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)", va.DeviceMobile, false},
		{"android mobi", "Mozilla/5.0 (Linux; Android 13) Mobile Safari", va.DeviceMobile, false},
		{"android no-mobi", "Mozilla/5.0 (Linux; Android 13)", va.DeviceMobile, false},
		{"ipad", "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X)", va.DeviceTablet, false},
		{"tablet keyword", "Mozilla/5.0 (Linux; Android 13; Tablet)", va.DeviceTablet, false},
		{"windows desktop", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)", va.DeviceDesktop, false},
		{"mac desktop", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)", va.DeviceDesktop, false},
		{"googlebot", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", va.DeviceBot, true},
		{"generic crawler", "SomeCrawler/1.0", va.DeviceBot, true},
		{"spider", "BigSpider/2.0", va.DeviceBot, true},
		{"slurp", "Yahoo! Slurp", va.DeviceBot, true},
		{"facebook", "facebookexternalhit/1.1", va.DeviceBot, true},
		{"headless", "HeadlessChrome/120.0", va.DeviceBot, true},
		{"preview", "WhatsApp Preview Bot", va.DeviceBot, true},
	}
	for _, c := range cases {
		gotType, gotBot := va.ClassifyDevice(c.ua)
		assert.Equalf(t, c.wantType, gotType, "%s: device type", c.name)
		assert.Equalf(t, c.wantBot, gotBot, "%s: is-bot", c.name)
	}
}

// TC-A2-5: bot detection wins over device keywords (a bot UA that also says
// "mobile" still classifies as bot).
func TestClassifyDevice_BotBeatsDevice(t *testing.T) {
	got, isBot := va.ClassifyDevice("Mobile Googlebot Crawler")
	assert.Equal(t, va.DeviceBot, got)
	assert.True(t, isBot)
}

// TC2 (SE4): ClassifyClient reduces a UA to (device, os, browser, isBot) using
// the zero-dep keyword tables. Unknown segments fall into the "other" bucket;
// bot detection still wins over any device/os/browser keyword.
func TestClassifyClient(t *testing.T) {
	cases := []struct {
		name        string
		ua          string
		wantDevice  va.DeviceType
		wantOS      va.OSFamily
		wantBrowser va.BrowserFamily
		wantBot     bool
	}{
		{
			name:       "empty",
			ua:         "",
			wantDevice: va.DeviceUnknown, wantOS: va.OSOther, wantBrowser: va.BrowserOther, wantBot: false,
		},
		{
			name:       "iphone safari",
			ua:         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			wantDevice: va.DeviceMobile, wantOS: va.OSiOS, wantBrowser: va.BrowserSafari, wantBot: false,
		},
		{
			name:       "android chrome",
			ua:         "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			wantDevice: va.DeviceMobile, wantOS: va.OSAndroid, wantBrowser: va.BrowserChrome, wantBot: false,
		},
		{
			name:       "android samsung browser",
			ua:         "Mozilla/5.0 (Linux; Android 13; SAMSUNG SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/23.0 Chrome/115.0.0.0 Mobile Safari/537.36",
			wantDevice: va.DeviceMobile, wantOS: va.OSAndroid, wantBrowser: va.BrowserSamsung, wantBot: false,
		},
		{
			name:       "ipad safari",
			ua:         "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/604.1",
			wantDevice: va.DeviceTablet, wantOS: va.OSiOS, wantBrowser: va.BrowserSafari, wantBot: false,
		},
		{
			name:       "windows chrome",
			ua:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			wantDevice: va.DeviceDesktop, wantOS: va.OSWindows, wantBrowser: va.BrowserChrome, wantBot: false,
		},
		{
			name:       "windows edge",
			ua:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			wantDevice: va.DeviceDesktop, wantOS: va.OSWindows, wantBrowser: va.BrowserEdge, wantBot: false,
		},
		{
			name:       "mac firefox",
			ua:         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0",
			wantDevice: va.DeviceDesktop, wantOS: va.OSMacOS, wantBrowser: va.BrowserFirefox, wantBot: false,
		},
		{
			name:       "mac safari",
			ua:         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
			wantDevice: va.DeviceDesktop, wantOS: va.OSMacOS, wantBrowser: va.BrowserSafari, wantBot: false,
		},
		{
			name:       "linux chrome",
			ua:         "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			wantDevice: va.DeviceDesktop, wantOS: va.OSLinux, wantBrowser: va.BrowserChrome, wantBot: false,
		},
		{
			name:       "googlebot is bot",
			ua:         "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			wantDevice: va.DeviceBot, wantOS: va.OSOther, wantBrowser: va.BrowserOther, wantBot: true,
		},
		{
			name:       "unknown ua -> other buckets, desktop",
			ua:         "SomeWeirdClient/9.9",
			wantDevice: va.DeviceDesktop, wantOS: va.OSOther, wantBrowser: va.BrowserOther, wantBot: false,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			gotDevice, gotOS, gotBrowser, gotBot := va.ClassifyClient(c.ua)
			assert.Equal(t, c.wantDevice, gotDevice, "device")
			assert.Equal(t, c.wantOS, gotOS, "os")
			assert.Equal(t, c.wantBrowser, gotBrowser, "browser")
			assert.Equal(t, c.wantBot, gotBot, "isBot")
			assert.True(t, gotOS.Valid(), "os family must be in the closed set")
			assert.True(t, gotBrowser.Valid(), "browser family must be in the closed set")
		})
	}
}

// TC3 (SE4): ParseLanguage extracts the primary subtag of the first
// Accept-Language entry, lowercased, with q-values stripped. Empty/garbage → "".
func TestParseLanguage(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"simple", "en", "en"},
		{"region", "en-US", "en"},
		{"uppercase region", "EN-GB", "en"},
		{"multi with q", "en-US,en;q=0.9,fr;q=0.8", "en"},
		{"first is non-en", "fr-FR,fr;q=0.9,en;q=0.8", "fr"},
		{"wildcard only", "*", ""},
		{"leading space", "  de-DE , en;q=0.5", "de"},
		{"q-value on first", "ru;q=0.9", "ru"},
		{"garbage", ";;;", ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, va.ParseLanguage(c.in))
		})
	}
}

// TC4 (SE2): ClassifyChannel maps a sanitized referer_origin to one of the four
// traffic channels. This mirrors the read-side registry CASE so the Go classifier
// and the SQL stay in lockstep.
func TestClassifyChannel(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want va.Channel
	}{
		{"empty -> direct", "", va.ChannelDirect},
		{"google -> search", "https://www.google.com", va.ChannelSearch},
		{"bing -> search", "https://bing.com", va.ChannelSearch},
		{"duckduckgo -> search", "https://duckduckgo.com", va.ChannelSearch},
		{"facebook -> social", "https://facebook.com", va.ChannelSocial},
		{"instagram -> social", "https://instagram.com", va.ChannelSocial},
		{"t.co -> social", "https://t.co", va.ChannelSocial},
		{"tiktok -> social", "https://tiktok.com", va.ChannelSocial},
		{"line -> social", "https://line.me", va.ChannelSocial},
		{"other site -> referral", "https://some-blog.example", va.ChannelReferral},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := va.ClassifyChannel(c.in)
			assert.Equal(t, c.want, got)
			assert.True(t, got.Valid(), "channel must be in the closed set")
		})
	}
}
