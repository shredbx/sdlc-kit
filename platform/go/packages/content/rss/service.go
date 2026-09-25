package rss

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"
)

// DefaultPageSize is used when Query.PageSize is unset (<= 0).
const DefaultPageSize = 20

// DefaultCacheTTL is the short TTL applied to a source's cached body/validators.
const DefaultCacheTTL = 5 * time.Minute

// Service is the feed aggregation engine: it fetches every enabled source
// (cached + conditional), parses each via its ParserKind adapter, then merges,
// filters, sorts (newest first) and paginates. It degrades gracefully — a
// source that errors or times out is logged and skipped, and List only returns
// an error when every source fails.
type Service struct {
	fetcher     *Fetcher
	cache       Cache
	sources     []FeedSource
	store       Store
	cacheTTL    time.Duration
	imageCacher ImageCacher
}

// Option configures a Service at construction (functional-options pattern).
type Option func(*Service)

// WithCacheTTL overrides the per-source cache TTL (default DefaultCacheTTL).
// A consumer that runs a real cache backend (e.g. BR's Redis adapter) uses this
// to choose its own short-TTL within the design's ~5–10 min envelope; a
// non-positive value is ignored (keeps the default).
func WithCacheTTL(d time.Duration) Option {
	return func(s *Service) {
		if d > 0 {
			s.cacheTTL = d
		}
	}
}

// WithStore injects a persistence backend (Rule #9 — storage stays out of the
// engine). With a Store wired, Refresh persists imported items and List /
// NewerCount read from the Store (DB-backed, zero upstream RSS on a page load);
// without one, List keeps its legacy live-fetch+merge behavior. A nil store is
// ignored (keeps the live-fetch path).
func WithStore(store Store) Option {
	return func(s *Service) {
		if store != nil {
			s.store = store
		}
	}
}

// WithImageCacher injects the R2-thumbnail caching backend (Rule #9 — the engine
// stays storage-agnostic). With a cacher wired, the background Refresh resolves
// each item's image (incl. the og:image fallback), caches it, and persists the
// resulting ImageR2URL. Without one, items keep their upstream ImageURL and
// ImageR2URL stays empty. A nil cacher is ignored. It is NEVER consulted on the
// request/List path — caching is a refresh-time, background concern only.
func WithImageCacher(c ImageCacher) Option {
	return func(s *Service) {
		if c != nil {
			s.imageCacher = c
		}
	}
}

// NewService wires the engine with an HTTP client, a cache backend, and the
// registered sources. The client may carry any RoundTripper (tests inject a
// fixture transport); production passes DefaultHTTPClient(). Optional Options
// (e.g. WithCacheTTL) tune behavior.
func NewService(client *http.Client, cache Cache, sources []FeedSource, opts ...Option) *Service {
	if client == nil {
		client = DefaultHTTPClient()
	}
	s := &Service{
		fetcher:  NewFetcher(client),
		cache:    cache,
		sources:  sources,
		cacheTTL: DefaultCacheTTL,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Sources returns the registered sources (registry view, including disabled).
func (s *Service) Sources() []FeedSource {
	out := make([]FeedSource, len(s.sources))
	copy(out, s.sources)
	return out
}

// cachedSource is the cache payload per source: the last good body plus the
// validators to replay, enabling 304 reuse across calls.
type cachedSource struct {
	Body         []byte `json:"body"`
	ETag         string `json:"etag"`
	LastModified string `json:"lastModified"`
}

// List assembles a filtered, sorted, paginated slice of the merged feed.
// Partial-serve: a source that fails to fetch or parse is logged and skipped;
// an error is returned only when every enabled source fails.
func (s *Service) List(ctx context.Context, q Query) (Result, error) {
	// Store-backed (DB) read: the persisted-feed path. Items were imported by
	// Refresh; the public listing reads them with ZERO upstream RSS calls.
	if s.store != nil {
		return s.store.ListItems(ctx, q)
	}

	// Legacy live-fetch path (no store wired): fetch+merge enabled sources.
	var (
		merged       []FeedItem
		enabledCount int
		failCount    int
	)

	for _, src := range s.sources {
		if !src.Enabled {
			continue
		}
		enabledCount++

		items, err := s.fetchAndParse(ctx, src)
		if err != nil {
			failCount++
			log.Printf("feed: source %q skipped: %v", src.ID, err)
			continue
		}
		merged = append(merged, items...)
	}

	// Only fail when every enabled source failed (and there was at least one).
	if enabledCount > 0 && failCount == enabledCount {
		return Result{}, fmt.Errorf("feed: all %d enabled sources failed", enabledCount)
	}

	filtered := applyFilters(merged, q)
	sortByPublishedDesc(filtered)
	return paginate(filtered, q), nil
}

// fetchAndParse fetches one source (conditional GET + cache reuse), then parses
// it with the source's adapter.
func (s *Service) fetchAndParse(ctx context.Context, src FeedSource) ([]FeedItem, error) {
	parser, err := ParserFor(src.Parser)
	if err != nil {
		return nil, err
	}

	prev := s.readCache(ctx, src)
	res, err := s.fetcher.Fetch(ctx, src.URL, prev.ETag, prev.LastModified)
	if err != nil {
		return nil, err
	}

	body := res.Body
	if res.NotModified {
		if len(prev.Body) == 0 {
			return nil, fmt.Errorf("304 with no cached body for %q", src.ID)
		}
		body = prev.Body
	}

	// Refresh the cache with the freshest body + validators.
	s.writeCache(ctx, src, cachedSource{
		Body:         body,
		ETag:         res.ETag,
		LastModified: res.LastModified,
	})

	return parser.Parse(body, src)
}

func (s *Service) cacheKey(src FeedSource) string { return "feed:source:" + src.ID }

func (s *Service) readCache(ctx context.Context, src FeedSource) cachedSource {
	if s.cache == nil {
		return cachedSource{}
	}
	raw, found, err := s.cache.Get(ctx, s.cacheKey(src))
	if err != nil || !found {
		return cachedSource{}
	}
	var c cachedSource
	if json.Unmarshal(raw, &c) != nil {
		return cachedSource{}
	}
	return c
}

func (s *Service) writeCache(ctx context.Context, src FeedSource, c cachedSource) {
	if s.cache == nil {
		return
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return
	}
	if err := s.cache.Set(ctx, s.cacheKey(src), raw, s.cacheTTL); err != nil {
		log.Printf("feed: cache set %q: %v", src.ID, err)
	}
}

// applyFilters keeps items matching the query. Empty / invalid filter values
// mean "no filter" so a bad query param never 500s or silently empties.
func applyFilters(items []FeedItem, q Query) []FeedItem {
	srcSet := map[string]bool{}
	for _, id := range q.Sources {
		srcSet[id] = true
	}
	catActive := q.Category.Valid()
	langActive := q.Language.Valid()

	out := make([]FeedItem, 0, len(items))
	for _, it := range items {
		if len(srcSet) > 0 && !srcSet[it.SourceID] {
			continue
		}
		if catActive && it.Category != q.Category {
			continue
		}
		if langActive && it.Language != q.Language {
			continue
		}
		out = append(out, it)
	}
	return out
}

// sortByPublishedDesc orders items newest first; ties break by GUID for a
// stable, deterministic order.
func sortByPublishedDesc(items []FeedItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].PublishedAt.Equal(items[j].PublishedAt) {
			return items[i].GUID > items[j].GUID
		}
		return items[i].PublishedAt.After(items[j].PublishedAt)
	})
}

// paginate slices items into the requested page and reports HasNext.
func paginate(items []FeedItem, q Query) Result {
	page := q.Page
	if page < 1 {
		page = 1
	}
	size := q.PageSize
	if size < 1 {
		size = DefaultPageSize
	}

	start := (page - 1) * size
	if start >= len(items) {
		return Result{Items: []FeedItem{}, Page: page, HasNext: false}
	}
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	return Result{
		Items:   items[start:end],
		Page:    page,
		HasNext: end < len(items),
	}
}
