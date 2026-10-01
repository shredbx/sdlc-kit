package rss

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultFetchTimeout caps a single upstream fetch (connect + read).
const DefaultFetchTimeout = 10 * time.Second

// DefaultMaxBodyBytes caps a fetched body to guard against hostile / runaway
// feeds (the largest live feed observed was ~560 KB; 5 MB is generous).
const DefaultMaxBodyBytes int64 = 5 << 20

// userAgent identifies the aggregator to upstream feeds. Some hosts reject the
// default Go agent, so we present a browser-like UA (matches the curl probe).
const userAgent = "Mozilla/5.0 (compatible; SBXFeedBot/1.0)"

// httpDoer is the minimal client surface Fetcher depends on, so tests can
// inject a stub without a real network round-trip.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultHTTPClient returns an *http.Client with a sane timeout for outbound
// feed fetches, mirroring the pkg/check convention. CheckRedirect re-validates
// every redirect hop against the SSRF guard (a 30x to an internal host is the
// classic one-time-preflight bypass) — see ssrf.go.
func DefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout:       DefaultFetchTimeout,
		CheckRedirect: guardedCheckRedirect,
		Transport:     safeTransport(),
	}
}

// CachedResponse is what a conditional fetch carries back: the body plus the
// validators to replay on the next request.
type CachedResponse struct {
	// Body is the response body (empty on a 304 Not Modified reuse).
	Body []byte
	// ETag is the upstream ETag validator, if any.
	ETag string
	// LastModified is the upstream Last-Modified validator, if any.
	LastModified string
	// NotModified is true when the upstream returned 304 and the caller should
	// reuse its previously cached body.
	NotModified bool
}

// StatusError is returned by Fetch when the upstream responds with a non-2xx
// (and non-304) status. It carries the numeric StatusCode so callers (e.g.
// Validate) can report it without string-parsing the error message.
type StatusError struct {
	// URL is the fetched endpoint.
	URL string
	// StatusCode is the upstream HTTP status (e.g. 402, 404, 500).
	StatusCode int
}

// Error implements error.
func (e *StatusError) Error() string {
	return fmt.Sprintf("feed: fetch %s: unexpected status %d", e.URL, e.StatusCode)
}

// Fetcher performs conditional GETs against feed URLs, capping body size and
// honoring ETag / Last-Modified validators for cheap refreshes.
type Fetcher struct {
	// Doer issues requests (an *http.Client in production).
	Doer httpDoer
	// MaxBytes caps the response body; <= 0 uses DefaultMaxBodyBytes.
	MaxBytes int64
}

// NewFetcher builds a Fetcher over doer.
func NewFetcher(doer httpDoer) *Fetcher {
	return &Fetcher{Doer: doer, MaxBytes: DefaultMaxBodyBytes}
}

// Fetch issues a conditional GET for url, sending If-None-Match / If-Modified-
// Since from the prior etag / lastModified. On 304 it returns NotModified=true
// with an empty Body; otherwise it returns the (size-capped) body and fresh
// validators. A non-2xx (and non-304) status is a wrapped error.
func (f *Fetcher) Fetch(ctx context.Context, url, etag, lastModified string) (CachedResponse, error) {
	// SSRF pre-flight: refuse non-https / internal targets before any dial.
	if err := validatePublicURL(url); err != nil {
		return CachedResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return CachedResponse{}, fmt.Errorf("feed: build request %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	resp, err := f.Doer.Do(req)
	if err != nil {
		return CachedResponse{}, fmt.Errorf("feed: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return CachedResponse{
			NotModified:  true,
			ETag:         firstNonEmpty(resp.Header.Get("ETag"), etag),
			LastModified: firstNonEmpty(resp.Header.Get("Last-Modified"), lastModified),
		}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return CachedResponse{}, &StatusError{URL: url, StatusCode: resp.StatusCode}
	}

	limit := f.MaxBytes
	if limit <= 0 {
		limit = DefaultMaxBodyBytes
	}
	// Read one extra byte past the cap so we can detect (and reject) oversized
	// bodies rather than silently truncating into invalid XML.
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return CachedResponse{}, fmt.Errorf("feed: read body %s: %w", url, err)
	}
	if int64(len(body)) > limit {
		return CachedResponse{}, fmt.Errorf("feed: fetch %s: body exceeds %d byte cap", url, limit)
	}

	return CachedResponse{
		Body:         body,
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
	}, nil
}

// FetchRaw issues a plain (non-conditional) SSRF-guarded GET and returns the
// size-capped body. It backs the P3 og:image fetch (an article HTML page) and
// the BR image-cache download — anything that needs a one-shot guarded read of
// an attacker-influenceable URL. maxBytes <= 0 falls back to the Fetcher's
// MaxBytes (then DefaultMaxBodyBytes). The SSRF pre-flight + the client's
// CheckRedirect together guard every hop.
func (f *Fetcher) FetchRaw(ctx context.Context, url string, maxBytes int64) ([]byte, error) {
	if err := validatePublicURL(url); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("feed: build request %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := f.Doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("feed: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &StatusError{URL: url, StatusCode: resp.StatusCode}
	}

	limit := maxBytes
	if limit <= 0 {
		limit = f.MaxBytes
	}
	if limit <= 0 {
		limit = DefaultMaxBodyBytes
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("feed: read body %s: %w", url, err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("feed: fetch %s: body exceeds %d byte cap", url, limit)
	}
	return body, nil
}

// SafeGet is a package-level guarded GET for consumers that hold their own
// *http.Client (e.g. the BR R2 ImageCacher downloading an image) but want the
// SAME SSRF protection the engine applies: an https-only, internal-host-blocking
// pre-flight, a redirect guard, and a body-size cap. The client SHOULD be
// DefaultHTTPClient() (which carries the redirect guard); SafeGet adds the
// pre-flight + cap so the caller never re-implements the policy.
func SafeGet(ctx context.Context, client *http.Client, url string, maxBytes int64) ([]byte, error) {
	if client == nil {
		client = DefaultHTTPClient()
	}
	f := &Fetcher{Doer: client, MaxBytes: maxBytes}
	return f.FetchRaw(ctx, url, maxBytes)
}
