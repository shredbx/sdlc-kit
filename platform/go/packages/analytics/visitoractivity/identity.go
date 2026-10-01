package visitoractivity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ipUASeparator is a control byte (0x1f, ASCII Unit Separator) inserted between
// the IP and user-agent in the HMAC input. Without it, ("ab","c") and
// ("a","bc") would hash identical bytes; the separator makes the boundary
// unambiguous because 0x1f cannot appear in an IP literal or a UA string.
var ipUASeparator = []byte{0x1f}

// ComputeVisitorID is a keyed (HMAC-SHA256) per-day anonymous visitor id.
//
// Because the salt is rotated every UTC day (see SaltProvider), the same
// (ip, ua) pair maps to a DIFFERENT id tomorrow — the id is therefore not
// durable PII and cannot be correlated across days, nor reversed to the IP/UA
// without the day's secret salt. The raw IP and UA are never stored; only this
// digest is.
func ComputeVisitorID(salt, ip, ua string) string {
	m := hmac.New(sha256.New, []byte(salt))
	m.Write([]byte(ip))
	m.Write(ipUASeparator)
	m.Write([]byte(ua))
	return hex.EncodeToString(m.Sum(nil))
}

// botKeywords are case-insensitive substrings that mark a user-agent as an
// automated client. Matched before device buckets so a "mobile" bot still
// classifies as a bot (and is excluded from public view counts).
var botKeywords = []string{
	"bot",
	"crawl",
	"spider",
	"slurp",
	"facebookexternalhit",
	"headless",
	"preview",
}

// ClassifyDevice reduces a raw user-agent to a coarse DeviceType bucket plus an
// is-bot flag. The raw UA is used here only — it is never persisted.
//
// Order matters: bot detection runs first (bots win over any device keyword),
// then tablet (ipad/tablet) before mobile (so an iPad is a Tablet, not Mobile),
// then mobile (mobi/iphone/android). An empty UA is Unknown; anything else is
// Desktop.
func ClassifyDevice(ua string) (DeviceType, bool) {
	if ua == "" {
		return DeviceUnknown, false
	}
	lc := strings.ToLower(ua)

	for _, kw := range botKeywords {
		if strings.Contains(lc, kw) {
			return DeviceBot, true
		}
	}

	return classifyDeviceLC(lc), false
}

// classifyDeviceLC buckets a pre-lowercased, already-not-a-bot UA into a device
// type. Shared by ClassifyDevice and ClassifyClient so the device rules stay in
// one place.
func classifyDeviceLC(lc string) DeviceType {
	switch {
	case strings.Contains(lc, "ipad"), strings.Contains(lc, "tablet"):
		return DeviceTablet
	case strings.Contains(lc, "mobi"), strings.Contains(lc, "iphone"), strings.Contains(lc, "android"):
		return DeviceMobile
	default:
		return DeviceDesktop
	}
}

// classifyOS buckets a pre-lowercased UA into an OSFamily. iOS is matched before
// macOS ("mac os x" appears in iOS UAs too), and Android before Linux (Android
// UAs also contain "linux").
func classifyOS(lc string) OSFamily {
	switch {
	case strings.Contains(lc, "iphone"), strings.Contains(lc, "ipad"), strings.Contains(lc, "ipod"):
		return OSiOS
	case strings.Contains(lc, "android"):
		return OSAndroid
	case strings.Contains(lc, "windows"):
		return OSWindows
	case strings.Contains(lc, "mac os"), strings.Contains(lc, "macintosh"):
		return OSMacOS
	case strings.Contains(lc, "linux"), strings.Contains(lc, "x11"):
		return OSLinux
	default:
		return OSOther
	}
}

// classifyBrowser buckets a pre-lowercased UA into a BrowserFamily. Order is
// critical because browser UAs nest tokens: Edge/Samsung/Chrome all carry
// "chrome", and Chrome carries "safari". Most-specific token wins — Edge ("edg/"
// or "edga/edgios"), then Samsung ("samsungbrowser"), then Firefox, then Chrome,
// then Safari last (only a real Safari has "safari" without the others).
func classifyBrowser(lc string) BrowserFamily {
	switch {
	case strings.Contains(lc, "edg/"), strings.Contains(lc, "edga/"), strings.Contains(lc, "edgios/"), strings.Contains(lc, "edge/"):
		return BrowserEdge
	case strings.Contains(lc, "samsungbrowser"):
		return BrowserSamsung
	case strings.Contains(lc, "firefox"), strings.Contains(lc, "fxios"):
		return BrowserFirefox
	case strings.Contains(lc, "chrome"), strings.Contains(lc, "crios"), strings.Contains(lc, "chromium"):
		return BrowserChrome
	case strings.Contains(lc, "safari"):
		return BrowserSafari
	default:
		return BrowserOther
	}
}

// ClassifyClient reduces a raw user-agent to coarse (device, os, browser, isBot)
// buckets in one pass. The raw UA is used here only — it is never persisted.
//
// Like ClassifyDevice, bot detection wins first: a bot UA returns DeviceBot with
// os/browser left in the "other" bucket (a crawler's OS/browser is noise). An
// empty UA is fully unknown/other. Unknown segments fall into the "other" bucket
// — never an error — so the table stays zero-dependency and forgiving.
func ClassifyClient(ua string) (DeviceType, OSFamily, BrowserFamily, bool) {
	if ua == "" {
		return DeviceUnknown, OSOther, BrowserOther, false
	}
	lc := strings.ToLower(ua)

	for _, kw := range botKeywords {
		if strings.Contains(lc, kw) {
			return DeviceBot, OSOther, BrowserOther, true
		}
	}

	return classifyDeviceLC(lc), classifyOS(lc), classifyBrowser(lc), false
}

// ParseLanguage extracts the primary language subtag from a raw Accept-Language
// header value, lowercased. It takes the first comma-separated entry, drops any
// ";q=" weight, splits the region off the primary subtag (en-US → en), and
// returns "" for empty / wildcard / unparseable input. Zero dependency.
func ParseLanguage(acceptLanguage string) string {
	if acceptLanguage == "" {
		return ""
	}
	// First entry only (highest priority by header convention without sorting q).
	first := acceptLanguage
	if i := strings.IndexByte(first, ','); i >= 0 {
		first = first[:i]
	}
	// Drop any q-value.
	if i := strings.IndexByte(first, ';'); i >= 0 {
		first = first[:i]
	}
	first = strings.TrimSpace(first)
	// Primary subtag only (en-US → en).
	if i := strings.IndexByte(first, '-'); i >= 0 {
		first = first[:i]
	}
	first = strings.ToLower(strings.TrimSpace(first))
	if first == "" || first == "*" {
		return ""
	}
	return first
}

// searchEngineHosts are substrings of a referer_origin host that mark search
// traffic. Kept in lockstep with the "channel" SQL CASE in the dimension registry.
var searchEngineHosts = []string{"google", "bing", "duckduckgo", "yahoo"}

// socialHosts are substrings of a referer_origin host that mark social traffic.
// Kept in lockstep with the "channel" SQL CASE in the dimension registry.
var socialHosts = []string{"facebook", "instagram", "t.co", "twitter", "tiktok", "line", "lnk"}

// ClassifyChannel maps a sanitized referer_origin ("scheme://host" or "") to a
// traffic Channel. Empty → direct; a known search host → search; a known social
// host → social; anything else with a referer → referral. This mirrors the
// read-side registry CASE so the Go classifier and the SQL stay aligned.
func ClassifyChannel(refererOrigin string) Channel {
	if refererOrigin == "" {
		return ChannelDirect
	}
	lc := strings.ToLower(refererOrigin)
	for _, h := range searchEngineHosts {
		if strings.Contains(lc, h) {
			return ChannelSearch
		}
	}
	for _, h := range socialHosts {
		if strings.Contains(lc, h) {
			return ChannelSocial
		}
	}
	return ChannelReferral
}
