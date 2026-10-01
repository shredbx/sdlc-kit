package video

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultResolveTimeout caps a single keyless oEmbed fetch (connect + read).
const DefaultResolveTimeout = 10 * time.Second

// defaultMaxBodyBytes caps an oEmbed response body to guard against a hostile or
// runaway upstream. oEmbed JSON is a few hundred bytes; 1 MB is generous.
const defaultMaxBodyBytes int64 = 1 << 20

// userAgent identifies the resolver to upstream oEmbed endpoints. Some hosts
// reject the default Go agent, so we present a browser-like UA.
const userAgent = "Mozilla/5.0 (compatible; SBXVideoBot/1.0)"

// httpDoer is the minimal client surface the resolvers depend on, so tests can
// inject a stub transport without a real network round-trip.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultHTTPClient returns an *http.Client with a sane timeout for outbound
// oEmbed fetches, mirroring the pkg/feed convention.
func DefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: DefaultResolveTimeout}
}

// oembedResponse is the subset of the oEmbed JSON shape the resolvers consume.
// Width/height vary in type across providers (number vs "100%"), so they are
// intentionally omitted — orientation is derived from the URL, not the body.
type oembedResponse struct {
	Title        string `json:"title"`
	AuthorName   string `json:"author_name"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// fetchOEmbed performs a keyless GET against oembedURL and decodes the shared
// oEmbed subset. It is the one network primitive every resolver reuses. A
// non-2xx status, a body over the cap, or malformed JSON is a wrapped error.
func fetchOEmbed(ctx context.Context, doer httpDoer, oembedURL string) (oembedResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, oembedURL, nil)
	if err != nil {
		return oembedResponse{}, fmt.Errorf("video: build request %s: %w", oembedURL, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := doer.Do(req)
	if err != nil {
		return oembedResponse{}, fmt.Errorf("video: fetch %s: %w", oembedURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oembedResponse{}, fmt.Errorf("video: fetch %s: unexpected status %d", oembedURL, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, defaultMaxBodyBytes))
	if err != nil {
		return oembedResponse{}, fmt.Errorf("video: read body %s: %w", oembedURL, err)
	}

	var out oembedResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return oembedResponse{}, fmt.Errorf("video: decode oembed %s: %w", oembedURL, err)
	}
	return out, nil
}

// parseURL trims surrounding whitespace and parses raw into an absolute URL. It
// rejects inputs that are not absolute (no scheme) or carry no host, so a bare
// string like "hello world" never masquerades as a recognized link.
func parseURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("video: empty url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("video: url %q is not http(s)", raw)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("video: url %q has no host", raw)
	}
	return u, nil
}

// hostOf returns the lower-cased, port-stripped host of u.
func hostOf(u *url.URL) string {
	return strings.ToLower(u.Hostname())
}

// pathSegments splits u.Path into non-empty segments, so leading/trailing
// slashes and a "  …/  " trailing slash never produce empty entries.
func pathSegments(p string) []string {
	parts := strings.Split(p, "/")
	out := make([]string, 0, len(parts))
	for _, s := range parts {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
