package rss

import (
	"context"
	"log"
)

// ImageCacher resolves an upstream image URL into a project-hosted, cached
// thumbnail URL (P3: R2). It is the storage-agnostic seam that keeps pkg/feed
// from importing a concrete object store — mirroring how Cache and Store are
// injected (Rule #9). The BR app wires an R2-backed implementation
// (fetch → resize → upload via pkg/image); tests inject a stub.
//
// CacheImage runs ONLY in the background Refresh job, never the request path. It
// receives the SSRF-validated upstream srcURL and a stable itemKey (source+guid)
// for the object key, and returns the cached URL to persist on the item. A
// per-item failure is non-fatal at the call site: the engine logs it, leaves
// ImageR2URL empty, and keeps the item (the upstream ImageURL still serves).
type ImageCacher interface {
	CacheImage(ctx context.Context, itemKey ItemKey, srcURL string) (cachedURL string, err error)
}

// ItemKey identifies an item to the ImageCacher for deriving a stable object key
// (Decision #0271 system/news/{id}). It carries the natural dedup key so the
// adapter can mint a deterministic R2 path without depending on the DB row UUID.
type ItemKey struct {
	// SourceID is the registry id of the producing source.
	SourceID string
	// GUID is the source-provided stable item identifier.
	GUID string
}

// resolveAndCacheImage is the per-item P3 image pipeline run inside Refresh. It
// resolves the best image URL (applying the og:image fallback when the parser
// found none), then — when an ImageCacher is wired — enforces the idempotency
// rule (reuse the existing R2 thumbnail when the upstream URL is unchanged) and
// otherwise caches via the injected ImageCacher.
//
// It returns the RESOLVED upstream URL (which the caller persists as image_url,
// so the og:image fallback survives across runs and idempotency holds) and the
// cached R2 URL ("" when no cacher, no resolvable image, or a non-fatal failure).
// NEVER fatal — every failure logs and degrades to "".
func (s *Service) resolveAndCacheImage(ctx context.Context, it FeedItem) (imageURL, imageR2URL string) {
	srcURL := s.resolveImageURL(ctx, it)
	if srcURL == "" {
		return "", "" // no image anywhere → gradient placeholder downstream.
	}
	if s.imageCacher == nil {
		return srcURL, "" // caching disabled — persist resolved upstream only.
	}

	// Idempotency: if the item already has a cached thumbnail for this exact
	// upstream URL, reuse it — no re-fetch, no re-upload.
	if s.store != nil {
		if prev, found, err := s.store.ItemImageState(ctx, it.SourceID, it.GUID); err == nil && found {
			if prev.ImageR2URL != "" && prev.ImageURL == srcURL {
				return srcURL, prev.ImageR2URL
			}
		}
	}

	return srcURL, s.cacheItemImage(ctx, it, srcURL)
}

// resolveImageURL returns the best image URL for an item, applying the P3
// og:image fallback when the parser found none. It NEVER fetches when the item
// already carries an image (media:*/enclosure/content <img> already won) and
// NEVER fetches without a Link to read. The article fetch is SSRF-guarded (the
// same validatePublicURL gate as the source fetch) — a per-item fetch/parse
// failure yields "" (the caller keeps the item, image-less).
func (s *Service) resolveImageURL(ctx context.Context, it FeedItem) string {
	if it.ImageURL != "" {
		return it.ImageURL
	}
	if it.Link == "" {
		return ""
	}
	html, err := s.fetchArticleHTML(ctx, it.Link)
	if err != nil {
		log.Printf("feed: og:image fetch %q: %v", it.Link, err)
		return ""
	}
	return ExtractOGImage(html)
}

// ogFetchMaxBytes caps the article-page read for og:image extraction. A news
// article HEAD is small; 512 KB is generous and bounds a hostile page.
const ogFetchMaxBytes int64 = 512 << 10

// fetchArticleHTML GETs an article page (SSRF-guarded, size-capped) and returns
// its HTML for og:image extraction. It reuses the engine's guarded HTTP client
// (CheckRedirect re-validates hops) plus a pre-flight host check.
func (s *Service) fetchArticleHTML(ctx context.Context, link string) (string, error) {
	if err := validatePublicURL(link); err != nil {
		return "", err
	}
	res, err := s.fetcher.FetchRaw(ctx, link, ogFetchMaxBytes)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// cacheItemImage runs the injected ImageCacher for one item and returns the
// cached R2 URL, or "" when there is no cacher, no source URL, or the cache
// failed (non-fatal — the caller persists the item regardless). srcURL is the
// already-resolved upstream image (post og:image fallback).
func (s *Service) cacheItemImage(ctx context.Context, it FeedItem, srcURL string) string {
	if s.imageCacher == nil || srcURL == "" {
		return ""
	}
	if err := validatePublicURL(srcURL); err != nil {
		log.Printf("feed: image cache skip %q/%q: %v", it.SourceID, it.GUID, err)
		return ""
	}
	cached, err := s.imageCacher.CacheImage(ctx, ItemKey{SourceID: it.SourceID, GUID: it.GUID}, srcURL)
	if err != nil {
		log.Printf("feed: image cache %q/%q: %v", it.SourceID, it.GUID, err)
		return ""
	}
	return cached
}
