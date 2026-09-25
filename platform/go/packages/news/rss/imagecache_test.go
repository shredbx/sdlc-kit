package rss_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/shredbx/sbx-core/pkg/rss"
)

// stubCacher records every CacheImage call and returns a deterministic cached
// URL derived from the source URL — the in-test stand-in for the BR R2 uploader.
type stubCacher struct {
	mu       sync.Mutex
	calls    []cacheCall
	failURLs map[string]bool // srcURL -> return an error (per-item failure)
}

type cacheCall struct {
	key rss.ItemKey
	url string
}

func (c *stubCacher) CacheImage(_ context.Context, key rss.ItemKey, srcURL string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, cacheCall{key: key, url: srcURL})
	if c.failURLs[srcURL] {
		return "", io.ErrUnexpectedEOF
	}
	return "https://cdn.test/system/news/" + key.GUID + ".jpg", nil
}

func (c *stubCacher) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.calls)
}

// htmlTransport serves feed XML for source URLs and HTML for article links so the
// og:image fallback path is exercisable offline.
type htmlTransport struct {
	xml  map[string]string // url -> feed XML body
	html map[string]string // url -> article HTML body
}

func (t htmlTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u := req.URL.String()
	if body, ok := t.xml[u]; ok {
		return resp200(body), nil
	}
	if body, ok := t.html[u]; ok {
		return resp200(body), nil
	}
	return &http.Response{StatusCode: 404, Body: http.NoBody, Header: http.Header{}}, nil
}

func resp200(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
	}
}

// A minimal RSS2 feed whose single item carries NO image and a public https link,
// forcing the og:image fallback when an ImageCacher is wired.
const noImageFeed = `<?xml version="1.0"?>
<rss version="2.0"><channel><title>T</title>
  <item>
    <title>Article One</title>
    <link>https://news.test/article-1</link>
    <guid>https://news.test/article-1</guid>
    <description>excerpt</description>
    <pubDate>Wed, 28 May 2026 10:00:00 GMT</pubDate>
  </item>
</channel></rss>`

const articleWithOG = `<html><head>
  <meta property="og:image" content="https://img.test/article-1-og.jpg">
</head><body>...</body></html>`

func ogSource() rss.FeedSource {
	return rss.FeedSource{
		ID: "ogsrc", Name: "OG Source", URL: "https://news.test/feed.xml",
		Language: rss.LanguageEN, Category: rss.CategoryProperty,
		Parser: rss.ParserRSS2, Enabled: true,
	}
}

// TC-P3-01 — Refresh applies the og:image fallback and caches to R2: an item with
// no parser-found image triggers an article fetch, og:image extraction, and a
// CacheImage call; the persisted item carries the cached R2 URL.
func TestRefresh_OGImageFallback_CachesToR2(t *testing.T) {
	src := ogSource()
	tr := htmlTransport{
		xml:  map[string]string{src.URL: noImageFeed},
		html: map[string]string{"https://news.test/article-1": articleWithOG},
	}
	store := newStubStore(src)
	cacher := &stubCacher{}
	svc := rss.NewService(clientFor(tr), rss.NewMemoryCache(), nil,
		rss.WithStore(store), rss.WithImageCacher(cacher))

	rep, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if rep.Imported != 1 {
		t.Fatalf("imported = %d, want 1", rep.Imported)
	}
	if cacher.callCount() != 1 {
		t.Fatalf("CacheImage calls = %d, want 1", cacher.callCount())
	}
	if got := cacher.calls[0].url; got != "https://img.test/article-1-og.jpg" {
		t.Errorf("cached src = %q, want the og:image url", got)
	}

	// The persisted item serves the cached R2 URL on the public List path.
	res, _ := store.ListItems(context.Background(), rss.Query{})
	if len(res.Items) != 1 {
		t.Fatalf("list = %d items, want 1", len(res.Items))
	}
	if res.Items[0].ImageR2URL == "" {
		t.Error("ImageR2URL empty after cache, want the cdn url")
	}
}

// TC-P3-02 — idempotency: a second Refresh over the same unchanged item does NOT
// re-cache (no second CacheImage call) — the existing R2 thumbnail is reused.
func TestRefresh_ImageCache_Idempotent(t *testing.T) {
	src := ogSource()
	tr := htmlTransport{
		xml:  map[string]string{src.URL: noImageFeed},
		html: map[string]string{"https://news.test/article-1": articleWithOG},
	}
	store := newStubStore(src)
	cacher := &stubCacher{}
	svc := rss.NewService(clientFor(tr), rss.NewMemoryCache(), nil,
		rss.WithStore(store), rss.WithImageCacher(cacher))

	if _, err := svc.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh #1: %v", err)
	}
	if _, err := svc.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh #2: %v", err)
	}
	if cacher.callCount() != 1 {
		t.Errorf("CacheImage calls = %d, want 1 (idempotent — no re-upload)", cacher.callCount())
	}
}

// TC-P3-03 — per-item cache failure is non-fatal: the item is still imported,
// just with an empty ImageR2URL (the upstream/og image still persists).
func TestRefresh_ImageCacheFailure_NonFatal(t *testing.T) {
	src := ogSource()
	tr := htmlTransport{
		xml:  map[string]string{src.URL: noImageFeed},
		html: map[string]string{"https://news.test/article-1": articleWithOG},
	}
	store := newStubStore(src)
	cacher := &stubCacher{failURLs: map[string]bool{"https://img.test/article-1-og.jpg": true}}
	svc := rss.NewService(clientFor(tr), rss.NewMemoryCache(), nil,
		rss.WithStore(store), rss.WithImageCacher(cacher))

	rep, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if rep.Imported != 1 {
		t.Errorf("imported = %d, want 1 (failure non-fatal)", rep.Imported)
	}
	res, _ := store.ListItems(context.Background(), rss.Query{})
	if len(res.Items) != 1 {
		t.Fatalf("list = %d, want 1", len(res.Items))
	}
	if res.Items[0].ImageR2URL != "" {
		t.Errorf("ImageR2URL = %q, want empty after a cache failure", res.Items[0].ImageR2URL)
	}
	// The resolved og:image still persists as the upstream fallback.
	if res.Items[0].ImageURL != "https://img.test/article-1-og.jpg" {
		t.Errorf("ImageURL = %q, want resolved og:image", res.Items[0].ImageURL)
	}
}

// TC-P3-04 — no cacher wired: Refresh resolves the og:image into ImageURL but
// never touches R2 (ImageR2URL stays empty).
func TestRefresh_NoCacher_ResolvesButDoesNotCache(t *testing.T) {
	src := ogSource()
	tr := htmlTransport{
		xml:  map[string]string{src.URL: noImageFeed},
		html: map[string]string{"https://news.test/article-1": articleWithOG},
	}
	store := newStubStore(src)
	svc := rss.NewService(clientFor(tr), rss.NewMemoryCache(), nil, rss.WithStore(store))

	if _, err := svc.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	res, _ := store.ListItems(context.Background(), rss.Query{})
	if res.Items[0].ImageURL != "https://img.test/article-1-og.jpg" {
		t.Errorf("ImageURL = %q, want resolved og:image even without a cacher", res.Items[0].ImageURL)
	}
	if res.Items[0].ImageR2URL != "" {
		t.Errorf("ImageR2URL = %q, want empty (no cacher)", res.Items[0].ImageR2URL)
	}
}
