// Package video provides a read-only, keyless video-link engine shared across
// SBX projects. Given a pasted social video URL it (1) recognizes the host,
// (2) parses a canonical URL + stable external id + orientation, and (3)
// resolves lightweight display metadata (title, author, thumbnail) via each
// platform's keyless oEmbed endpoint. It never hosts or transcodes media — it
// links out to the original, mirroring pkg/feed's link-out model.
//
// Design constraints (Decision-backed, task 2605-001):
//   - Zero external dependencies — stdlib only (net/http, encoding/json,
//     net/url, strings, time, context, errors, sync).
//   - Discriminators are dictionary-backed named types, never raw strings:
//     Platform, Orientation, Placement, Status.
//   - Recognition + parsing + resolution sit behind the Resolver interface
//     (one resolver_*.go adapter per platform). The Registry dispatches to the
//     matching resolver; adding a host = a new adapter, zero dispatch changes.
//   - Keyless resolve: YouTube/TikTok oEmbed need no API key; Instagram has no
//     keyless server metadata, so its Resolve returns an empty Meta (the UI
//     renders the client-side embed). A slow/erroring upstream yields an error
//     the caller stores as empty meta — a missing thumbnail never breaks a read.
//
// EXTENSIBILITY NOTE (intentionally NOT built now — all additive, same interface):
//   - API-based resolvers (view counts, exact length, channel auto-pull) arrive
//     as additive resolver_*_api.go files behind this same Resolver interface;
//     no caller changes. The richer fields (DurationSeconds, ViewCount) are
//     Phase-2 and are deliberately absent from Meta today.
//   - R2-owned video (our own poster + HLS, e.g. platform "bestie"/"r2") arrives
//     as an additive resolver adapter reading our own metadata. The upload +
//     transcode pipeline stays OUTSIDE pkg/video — it belongs to storage and the
//     `sbx media` processing gateway. pkg/video only ever reads/links.
//
// Example (consumer wiring):
//
//	reg := video.NewRegistry(video.DefaultHTTPClient(), video.NewMemoryCache())
//	p, err := reg.Parse("https://youtu.be/dQw4w9WgXcQ")
//	meta, err := reg.Resolve(ctx, p)
package video

import (
	"context"
	"errors"
)

// ErrUnsupportedURL is returned (wrapped) when a URL matches no registered
// resolver — an unknown host, a non-URL string, or an empty input.
var ErrUnsupportedURL = errors.New("video: unsupported or unrecognized URL")

// Parsed is the deterministic, network-free result of recognizing a video URL:
// the host platform, a stable external id, a canonical (query-stripped) URL,
// and the orientation inferred from the URL shape.
type Parsed struct {
	// Platform is the recognized host.
	Platform Platform
	// ExternalID is the host's stable id for the video (e.g. a YouTube videoId,
	// a TikTok numeric id, an Instagram shortcode).
	ExternalID string
	// CanonicalURL is the normalized public URL (query/timestamp stripped),
	// used as the oEmbed lookup key and the "watch on …" link-out target.
	CanonicalURL string
	// Orientation is inferred from the URL shape at parse time.
	Orientation Orientation
}

// Meta is the lightweight, keyless display metadata for a parsed video. Any
// field may be empty when the upstream omits it (or, for Instagram, when there
// is no keyless server metadata at all).
type Meta struct {
	// Title is the video title / caption.
	Title string
	// AuthorName is the uploader / channel display name.
	AuthorName string
	// ThumbnailURL is the best keyless poster image (e.g. YouTube hqdefault).
	ThumbnailURL string
}

// Resolver recognizes, parses and resolves the URLs of a single video platform.
// Each platform is one adapter (resolver_*.go); the Registry composes them.
type Resolver interface {
	// Match reports whether rawURL belongs to this resolver's platform. It is a
	// cheap host check and performs no network I/O.
	Match(rawURL string) bool
	// Parse extracts the platform, external id, canonical URL and orientation
	// from rawURL. It performs no network I/O. A URL this resolver cannot parse
	// yields a non-nil error.
	Parse(rawURL string) (Parsed, error)
	// Resolve fetches keyless display metadata for an already-parsed video. An
	// upstream failure (non-2xx, malformed body, timeout) returns an error;
	// platforms without keyless metadata return an empty Meta and a nil error.
	Resolve(ctx context.Context, p Parsed) (Meta, error)
}
