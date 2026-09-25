package video

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// tikTokResolver recognizes, parses and resolves TikTok links. The canonical
// form is /@{user}/video/{id}; TikTok videos are always portrait. The
// vm.tiktok.com short-link host is recognized but cannot be parsed without
// following a redirect, so Parse returns a clear Phase-1 error rather than
// silently dropping the link.
type tikTokResolver struct {
	doer httpDoer
}

// newTikTokResolver builds a TikTok resolver over doer.
func newTikTokResolver(doer httpDoer) *tikTokResolver {
	return &tikTokResolver{doer: doer}
}

// isTikTokHost reports whether host (lower-cased) is any tiktok.com host,
// including the vm./vt. short-link hosts.
func isTikTokHost(host string) bool {
	return host == "tiktok.com" || strings.HasSuffix(host, ".tiktok.com")
}

// isTikTokShortLinkHost reports whether host is a redirect-only short-link host.
func isTikTokShortLinkHost(host string) bool {
	return host == "vm.tiktok.com" || host == "vt.tiktok.com"
}

// Match reports whether rawURL is a TikTok link (canonical or short-link).
func (r *tikTokResolver) Match(rawURL string) bool {
	u, err := parseURL(rawURL)
	if err != nil {
		return false
	}
	return isTikTokHost(hostOf(u))
}

// Parse extracts the numeric video id from a canonical /@{user}/video/{id} URL.
// orientation is always portrait. A vm./vt. short link is recognized but errors
// (Phase 1 needs the canonical URL; the redirect is not followed at parse time).
func (r *tikTokResolver) Parse(rawURL string) (Parsed, error) {
	u, err := parseURL(rawURL)
	if err != nil {
		return Parsed{}, fmt.Errorf("video: parse tiktok url: %w", err)
	}
	host := hostOf(u)
	if !isTikTokHost(host) {
		return Parsed{}, fmt.Errorf("video: not a tiktok url %q", rawURL)
	}
	if isTikTokShortLinkHost(host) {
		return Parsed{}, fmt.Errorf("video: tiktok short link %q must be expanded to its canonical /@user/video/{id} URL (Phase 1 does not follow redirects)", rawURL)
	}

	segs := pathSegments(u.Path)
	// Expect: ["@user", "video", "{id}"]
	var id string
	for i := 0; i+1 < len(segs); i++ {
		if segs[i] == "video" {
			id = segs[i+1]
			break
		}
	}
	if id == "" {
		return Parsed{}, fmt.Errorf("video: could not extract tiktok id from %q", rawURL)
	}

	canonical := rawURL
	if u2, err := parseURL(rawURL); err == nil {
		u2.RawQuery = ""
		u2.Fragment = ""
		canonical = u2.String()
	}

	return Parsed{
		Platform:     PlatformTikTok,
		ExternalID:   id,
		CanonicalURL: canonical,
		Orientation:  OrientationPortrait,
	}, nil
}

// Resolve fetches the keyless TikTok oEmbed metadata.
func (r *tikTokResolver) Resolve(ctx context.Context, p Parsed) (Meta, error) {
	endpoint := "https://www.tiktok.com/oembed?url=" + url.QueryEscape(p.CanonicalURL)
	res, err := fetchOEmbed(ctx, r.doer, endpoint)
	if err != nil {
		return Meta{}, err
	}
	return Meta{
		Title:        res.Title,
		AuthorName:   res.AuthorName,
		ThumbnailURL: res.ThumbnailURL,
	}, nil
}
