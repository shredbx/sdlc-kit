package video

import (
	"context"
	"fmt"
	"strings"
)

// instagramResolver recognizes and parses Instagram reel/post links. A /reel/
// is portrait; a /p/ post is landscape. Instagram exposes no keyless server
// metadata, so Resolve returns an empty Meta with a nil error — the consumer UI
// renders the client-side embed (embed.js).
type instagramResolver struct {
	doer httpDoer
}

// newInstagramResolver builds an Instagram resolver over doer.
func newInstagramResolver(doer httpDoer) *instagramResolver {
	return &instagramResolver{doer: doer}
}

// isInstagramHost reports whether host (lower-cased) is an Instagram host.
func isInstagramHost(host string) bool {
	host = strings.TrimPrefix(host, "www.")
	return host == "instagram.com"
}

// Match reports whether rawURL is an Instagram link.
func (r *instagramResolver) Match(rawURL string) bool {
	u, err := parseURL(rawURL)
	if err != nil {
		return false
	}
	return isInstagramHost(hostOf(u))
}

// Parse extracts the shortcode from /reel/{code} (portrait) or /p/{code}
// (landscape).
func (r *instagramResolver) Parse(rawURL string) (Parsed, error) {
	u, err := parseURL(rawURL)
	if err != nil {
		return Parsed{}, fmt.Errorf("video: parse instagram url: %w", err)
	}
	if !isInstagramHost(hostOf(u)) {
		return Parsed{}, fmt.Errorf("video: not an instagram url %q", rawURL)
	}

	segs := pathSegments(u.Path)
	if len(segs) < 2 {
		return Parsed{}, fmt.Errorf("video: could not extract instagram code from %q", rawURL)
	}

	var orientation Orientation
	switch segs[0] {
	case "reel", "reels":
		orientation = OrientationPortrait
	case "p":
		orientation = OrientationLandscape
	default:
		return Parsed{}, fmt.Errorf("video: unsupported instagram path %q", u.Path)
	}
	code := segs[1]

	return Parsed{
		Platform:     PlatformInstagram,
		ExternalID:   code,
		CanonicalURL: "https://www.instagram.com/" + segs[0] + "/" + code + "/",
		Orientation:  orientation,
	}, nil
}

// Resolve returns an empty Meta and a nil error: Instagram has no keyless server
// metadata, so the consumer renders the client-side embed instead.
func (r *instagramResolver) Resolve(_ context.Context, _ Parsed) (Meta, error) {
	return Meta{}, nil
}
