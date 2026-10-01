// Package feed provides a read-only RSS news-feed aggregation engine shared
// across SBX projects. It fetches a registered set of external RSS sources
// (conditional GET, body-size capped), parses each via a ParserKind-selected
// adapter, normalizes items into a common shape, then merges, filters, sorts
// (newest first) and paginates them.
//
// Design constraints (Decision-backed, task 2605-001):
//   - Zero external dependencies — stdlib encoding/xml only (no gofeed/x/net).
//   - Discriminators are dictionary-backed named types, never raw strings:
//     FeedCategory, FeedLanguage, ParserKind.
//   - Parsing is behind the Parser interface (two adapters: rss2, wordpress);
//     caching is behind the Cache interface (in-memory adapter here, Redis
//     wired by the consumer). Adding a source = a registry row; adding a feed
//     shape = one new parser_*.go.
//   - Link-out model: items carry a plain-text Excerpt only (HTML stripped),
//     never a rendered body — shrinking the XSS surface and copyright risk.
//   - Partial-serve: a slow/erroring source is logged and skipped; List only
//     fails when every source fails.
//
// Example (consumer wiring):
//
//	svc := rss.NewService(rss.DefaultHTTPClient(), redisCache, sources)
//	res, err := svc.List(ctx, rss.Query{
//	    Category: rss.CategoryProperty,
//	    Language: rss.LanguageEN,
//	    Page:     1,
//	})
package rss

import "time"

// FeedItem is a single normalized news entry. Body text is never hosted —
// Excerpt is a short plain-text summary and Link points at the original source.
type FeedItem struct {
	// ID is the persisted item identifier (the news_items row UUID). Empty on the
	// live-fetch path and during import (the store assigns it); populated on the
	// store-backed List path so the public projection carries a stable handle for
	// the reveal/lead UI.
	ID string
	// GUID is the source-provided stable identifier (falls back to Link).
	GUID string
	// SourceID is the registry id of the FeedSource that produced this item.
	SourceID string
	// Title is the headline (plain text).
	Title string
	// Link is the canonical URL of the original article (the "Read more" target).
	Link string
	// Excerpt is a short, HTML-stripped, whitespace-collapsed summary.
	Excerpt string
	// ImageURL is the best available image (media:* > enclosure > first content
	// img > og:image > ""). On the store-backed List path this already prefers the
	// cached R2 thumbnail (ImageR2URL) over the upstream hotlink; empty means the
	// UI applies a branded gradient placeholder.
	ImageURL string
	// ImageR2URL is the cached news-thumbnail URL on R2 (P3): the public listing
	// serves this rather than hotlinking the upstream image. Empty until the
	// background refresh caches it (per-item failure leaves it empty, non-fatal).
	// Populated by the refresh image pipeline and persisted by UpsertItemByGUID.
	ImageR2URL string
	// Author is the byline (from dc:creator on WordPress feeds; "" when absent).
	Author string
	// PublishedAt is the parsed pubDate; zero time when unparseable.
	PublishedAt time.Time
	// Category is the normalized, source-assigned category.
	Category FeedCategory
	// Language is the source-assigned content language.
	Language FeedLanguage
	// SourceTags are the raw <category> values (free text), used as tag chips.
	SourceTags []string
	// Tags are CURATOR-assigned labels (BR task 2606-001, US-RC-02) — the
	// editable twin of SourceTags, matched against property tags via the shared
	// slug formula. Distinct from SourceTags, which stay read-only RSS
	// provenance and are never matched. Empty on the live-fetch path; populated
	// only by store-backed consumers that persist curator tags (news_items.tags).
	Tags []string
	// Featured marks the single curated public lead. False on the import path
	// (curation is admin-only); populated on the store-backed List path so the
	// public projection can render the lead treatment.
	Featured bool
}

// FeedSource is one registered upstream RSS feed. The registry (YAML, code-
// edited) is the source of truth; Category and Language are assigned here
// rather than parsed, because upstream tags are unreliable.
type FeedSource struct {
	// ID is the stable registry identifier (e.g. "bangkokpost-property").
	ID string
	// Name is the human-readable label (e.g. "Bangkok Post — Property").
	Name string
	// URL is the absolute RSS endpoint (HTTPS).
	URL string
	// Language is assigned to every item this source produces.
	Language FeedLanguage
	// Category is the normalized category assigned to this source's items.
	Category FeedCategory
	// Parser selects which adapter handles this source's XML shape.
	Parser ParserKind
	// Enabled gates whether the source participates in List.
	Enabled bool
	// ETag is the conditional-GET ETag validator persisted from the last fetch
	// (empty when never fetched). Replayed by Refresh as If-None-Match.
	ETag string
	// LastModified is the conditional-GET Last-Modified validator persisted from
	// the last fetch (empty when never fetched). Replayed as If-Modified-Since.
	LastModified string
}

// Query describes a feed slice request: optional filters plus pagination.
// Zero-value filters (empty Sources, empty Category/Language) mean "no filter".
type Query struct {
	// Sources optionally restricts results to these source IDs (OR-set).
	Sources []string
	// Category optionally restricts results to one normalized category.
	Category FeedCategory
	// Language optionally restricts results to one language.
	Language FeedLanguage
	// Page is 1-based; values < 1 are treated as 1.
	Page int
	// PageSize is items per page; values < 1 fall back to DefaultPageSize.
	PageSize int
}

// Result is a single page of merged, filtered, sorted feed items.
type Result struct {
	// Items is the current page, newest first.
	Items []FeedItem
	// Page is the 1-based page number these items belong to.
	Page int
	// HasNext reports whether a further page exists after this one.
	HasNext bool
}
