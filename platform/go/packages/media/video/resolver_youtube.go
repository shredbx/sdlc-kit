package video

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// youTubeResolver recognizes, parses and resolves YouTube links: youtube.com,
// m.youtube.com and youtu.be. A /shorts/ path is portrait; everything else is
// landscape. Resolution uses the keyless oEmbed endpoint.
type youTubeResolver struct {
	doer httpDoer
}

// newYouTubeResolver builds a YouTube resolver over doer.
func newYouTubeResolver(doer httpDoer) *youTubeResolver {
	return &youTubeResolver{doer: doer}
}

// isYouTubeHost reports whether host (already lower-cased, port-stripped) is a
// YouTube web host or the youtu.be short-link host.
func isYouTubeHost(host string) bool {
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")
	return host == "youtube.com" || host == "youtu.be"
}

// Match reports whether rawURL is a YouTube link.
func (r *youTubeResolver) Match(rawURL string) bool {
	u, err := parseURL(rawURL)
	if err != nil {
		return false
	}
	return isYouTubeHost(hostOf(u))
}

// Parse extracts the videoId from any of the watch / youtu.be / shorts / embed
// shapes, strips query + timestamp, and infers orientation from the path.
func (r *youTubeResolver) Parse(rawURL string) (Parsed, error) {
	u, err := parseURL(rawURL)
	if err != nil {
		return Parsed{}, fmt.Errorf("video: parse youtube url: %w", err)
	}
	host := hostOf(u)
	if !isYouTubeHost(host) {
		return Parsed{}, fmt.Errorf("video: not a youtube url %q", rawURL)
	}

	var id string
	orientation := OrientationLandscape

	bareHost := strings.TrimPrefix(strings.TrimPrefix(host, "www."), "m.")
	segs := pathSegments(u.Path)

	switch {
	case bareHost == "youtu.be":
		// youtu.be/{id}
		if len(segs) >= 1 {
			id = segs[0]
		}
	case len(segs) >= 1 && segs[0] == "shorts":
		// /shorts/{id} → portrait
		if len(segs) >= 2 {
			id = segs[1]
			orientation = OrientationPortrait
		}
	case len(segs) >= 1 && segs[0] == "embed":
		// /embed/{id}
		if len(segs) >= 2 {
			id = segs[1]
		}
	default:
		// /watch?v={id}
		id = u.Query().Get("v")
	}

	if id == "" {
		return Parsed{}, fmt.Errorf("video: could not extract youtube id from %q", rawURL)
	}

	// Preserve the surface form in the canonical URL: a Short keeps its /shorts/
	// path (the stored link + public "watch" target should open the vertical
	// player), everything else normalizes to /watch?v=. Both forms resolve via
	// the keyless oEmbed endpoint (verified), so Resolve is unaffected.
	canonical := "https://www.youtube.com/watch?v=" + id
	if orientation == OrientationPortrait {
		canonical = "https://www.youtube.com/shorts/" + id
	}

	return Parsed{
		Platform:     PlatformYouTube,
		ExternalID:   id,
		CanonicalURL: canonical,
		Orientation:  orientation,
	}, nil
}

// Resolve fetches the keyless YouTube oEmbed metadata. The oEmbed thumbnail_url
// (hqdefault) is the reliable primary and is used as-is — maxresdefault is NOT
// hardcoded because it 404s for Shorts.
func (r *youTubeResolver) Resolve(ctx context.Context, p Parsed) (Meta, error) {
	endpoint := "https://www.youtube.com/oembed?format=json&url=" + url.QueryEscape(p.CanonicalURL)
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
