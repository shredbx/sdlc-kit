package video

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Registry composes the per-platform resolvers and dispatches Match/Parse/
// Resolve to whichever one recognizes a URL. Adding a platform = registering one
// more Resolver here; the dispatch logic never changes.
//
// Resolve results are memoized through a Cache keyed by canonical URL so a
// repeated lookup of the same video skips the upstream oEmbed round-trip.
type Registry struct {
	resolvers []Resolver
	cache     Cache
}

// resolveCacheTTL bounds how long a resolved Meta stays cached. oEmbed titles /
// thumbnails are effectively static, so a generous TTL keeps the upstream quiet.
const resolveCacheTTL = 24 * time.Hour

// NewRegistry wires the Phase-1 resolvers (YouTube, TikTok, Instagram) over the
// shared HTTP client and cache. A nil client falls back to DefaultHTTPClient;
// a nil cache falls back to an in-memory cache.
func NewRegistry(client httpDoer, cache Cache) *Registry {
	if client == nil {
		client = DefaultHTTPClient()
	}
	if cache == nil {
		cache = NewMemoryCache()
	}
	return &Registry{
		resolvers: []Resolver{
			newYouTubeResolver(client),
			newTikTokResolver(client),
			newInstagramResolver(client),
		},
		cache: cache,
	}
}

// Match returns the first resolver that recognizes rawURL, or (nil, false) if
// none do.
func (r *Registry) Match(rawURL string) (Resolver, bool) {
	for _, res := range r.resolvers {
		if res.Match(rawURL) {
			return res, true
		}
	}
	return nil, false
}

// Parse dispatches to the matching resolver. An unrecognized URL wraps
// ErrUnsupportedURL.
func (r *Registry) Parse(rawURL string) (Parsed, error) {
	res, ok := r.Match(rawURL)
	if !ok {
		return Parsed{}, fmt.Errorf("video: %q: %w", rawURL, ErrUnsupportedURL)
	}
	return res.Parse(rawURL)
}

// Resolve dispatches to the resolver for p.Platform, serving a cached Meta when
// one is present and fresh. A resolver that yields an upstream error is NOT
// cached, so a transient failure can be retried.
func (r *Registry) Resolve(ctx context.Context, p Parsed) (Meta, error) {
	if !p.Platform.Valid() {
		return Meta{}, fmt.Errorf("video: resolve: invalid platform %q: %w", p.Platform, ErrUnsupportedURL)
	}

	key := resolveCacheKey(p)
	if r.cache != nil {
		if raw, found, err := r.cache.Get(ctx, key); err == nil && found {
			var cached Meta
			if json.Unmarshal(raw, &cached) == nil {
				return cached, nil
			}
		}
	}

	// Dispatch is adapter-driven: the resolver that recognizes the canonical URL
	// is the one that owns the platform — no platform switch to maintain.
	res, ok := r.Match(p.CanonicalURL)
	if !ok {
		return Meta{}, fmt.Errorf("video: resolve: no resolver for platform %q: %w", p.Platform, ErrUnsupportedURL)
	}

	meta, err := res.Resolve(ctx, p)
	if err != nil {
		return Meta{}, err
	}

	if r.cache != nil {
		if raw, mErr := json.Marshal(meta); mErr == nil {
			_ = r.cache.Set(ctx, key, raw, resolveCacheTTL)
		}
	}
	return meta, nil
}

// resolveCacheKey namespaces the cache entry by platform + external id.
func resolveCacheKey(p Parsed) string {
	return "video:resolve:" + p.Platform.String() + ":" + p.ExternalID
}
