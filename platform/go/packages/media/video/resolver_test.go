package video_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/shredbx/sbx-core/pkg/video"
)

// roundTripFn adapts a function to http.RoundTripper so a *http.Client can be
// driven without a network. It counts calls so cache hits are observable.
type roundTripFn struct {
	mu    sync.Mutex
	calls int
	fn    func(*http.Request) (*http.Response, error)
}

func (rt *roundTripFn) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.mu.Lock()
	rt.calls++
	rt.mu.Unlock()
	return rt.fn(req)
}

func (rt *roundTripFn) Calls() int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.calls
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func jsonResp(status int, body string) *http.Response {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// clientWith builds an *http.Client whose transport serves body for every
// request and counts calls.
func clientWith(rt *roundTripFn) *http.Client {
	return &http.Client{Transport: rt}
}

func TestResolve_YouTube(t *testing.T) {
	body := mustReadFile(t, "testdata/youtube_shorts_oembed.json")
	rt := &roundTripFn{fn: func(*http.Request) (*http.Response, error) { return jsonResp(200, body), nil }}
	reg := video.NewRegistry(clientWith(rt), video.NewMemoryCache())

	p, err := reg.Parse("https://www.youtube.com/shorts/tPEE9ZwTmy0")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	meta, err := reg.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if meta.Title != "Shortest Video on Youtube" {
		t.Errorf("title = %q", meta.Title)
	}
	if meta.AuthorName != "Mylo the Cat" {
		t.Errorf("author = %q", meta.AuthorName)
	}
	if meta.ThumbnailURL != "https://i.ytimg.com/vi/tPEE9ZwTmy0/hqdefault.jpg" {
		t.Errorf("thumbnail = %q (must be the oEmbed hqdefault, not a hardcoded maxres)", meta.ThumbnailURL)
	}
}

func TestResolve_TikTok(t *testing.T) {
	body := mustReadFile(t, "testdata/tiktok_oembed.json")
	rt := &roundTripFn{fn: func(*http.Request) (*http.Response, error) { return jsonResp(200, body), nil }}
	reg := video.NewRegistry(clientWith(rt), video.NewMemoryCache())

	p, err := reg.Parse("https://www.tiktok.com/@scout2015/video/6718335390845095173")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	meta, err := reg.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.HasPrefix(meta.Title, "Scramble up ur name") {
		t.Errorf("title = %q", meta.Title)
	}
	if meta.AuthorName != "Scout, Suki & Stella" {
		t.Errorf("author = %q", meta.AuthorName)
	}
}

func TestResolve_Instagram_EmptyMetaNoError(t *testing.T) {
	// Instagram has no keyless server metadata; Resolve must NOT hit the network
	// and must return an empty Meta with a nil error.
	rt := &roundTripFn{fn: func(*http.Request) (*http.Response, error) {
		t.Error("Instagram Resolve must not perform any HTTP request")
		return jsonResp(200, "{}"), nil
	}}
	reg := video.NewRegistry(clientWith(rt), video.NewMemoryCache())

	p, err := reg.Parse("https://www.instagram.com/reel/CxYz123/")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	meta, err := reg.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("Resolve(instagram) err = %v, want nil", err)
	}
	if meta != (video.Meta{}) {
		t.Errorf("Resolve(instagram) = %+v, want empty Meta", meta)
	}
	if rt.Calls() != 0 {
		t.Errorf("Instagram Resolve made %d HTTP calls, want 0", rt.Calls())
	}
}

func TestResolve_Upstream5xxIsError(t *testing.T) {
	rt := &roundTripFn{fn: func(*http.Request) (*http.Response, error) { return jsonResp(500, "boom"), nil }}
	reg := video.NewRegistry(clientWith(rt), video.NewMemoryCache())

	p, _ := reg.Parse("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	if _, err := reg.Resolve(context.Background(), p); err == nil {
		t.Error("upstream 500 should return an error")
	}
}

func TestResolve_MalformedJSONIsError(t *testing.T) {
	rt := &roundTripFn{fn: func(*http.Request) (*http.Response, error) { return jsonResp(200, "{"), nil }}
	reg := video.NewRegistry(clientWith(rt), video.NewMemoryCache())

	p, _ := reg.Parse("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	if _, err := reg.Resolve(context.Background(), p); err == nil {
		t.Error("malformed JSON should return an error")
	}
}

func TestResolve_TransportError(t *testing.T) {
	rt := &roundTripFn{fn: func(*http.Request) (*http.Response, error) { return nil, errors.New("dial timeout") }}
	reg := video.NewRegistry(clientWith(rt), video.NewMemoryCache())

	p, _ := reg.Parse("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	if _, err := reg.Resolve(context.Background(), p); err == nil {
		t.Error("a transport error should propagate")
	}
}

func TestResolve_CacheServesSecondCall(t *testing.T) {
	body := mustReadFile(t, "testdata/youtube_watch_oembed.json")
	rt := &roundTripFn{fn: func(*http.Request) (*http.Response, error) { return jsonResp(200, body), nil }}
	reg := video.NewRegistry(clientWith(rt), video.NewMemoryCache())

	p, _ := reg.Parse("https://www.youtube.com/watch?v=dQw4w9WgXcQ")

	first, err := reg.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	second, err := reg.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if first != second {
		t.Errorf("cached Resolve differs: %+v vs %+v", first, second)
	}
	if rt.Calls() != 1 {
		t.Errorf("transport hit %d times across two Resolves, want exactly 1 (second served from cache)", rt.Calls())
	}
}
