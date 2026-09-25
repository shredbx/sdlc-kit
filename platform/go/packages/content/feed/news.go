// Package feed is the sbx-core domain for a persisted content stream: NewsSource
// (a registered upstream feed + its conditional-GET fetch state), NewsCategory
// (a normalized dictionary classification), and NewsItem (a deduplicated,
// curate-able archived article).
//
// Promoted from BR internal/news to sbx-core/pkg/feed (Fabric FF1) for
// cross-vertical reuse. The BR consumer still owns persistence (repositories in
// api-chi/internal/repository) and the entity governance:
//   entities/news-source/platforms/api-chi/md.yml
//   entities/news-category/platforms/api-chi/md.yml
//   entities/news-item/platforms/api-chi/md.yml
// Keep those repository column lists in exact sync with the md.yml files and the
// migration (news_sources / news_categories / news_items).
//
// The shared pkg/rss engine owns the fetch/parse/merge logic and the
// FeedCategory / FeedLanguage / ParserKind named types (reused here, never
// re-declared). The repositories implement rss.Store so the engine stays
// storage-agnostic and BR injects the postgres backend.
package feed

import (
	"time"

	"github.com/shredbx/sbx-core/pkg/rss"
)

// Category is a normalized news classification dictionary row. Its Code is the
// shared rss.FeedCategory named type (property | business | general) — never a
// raw string.
type Category struct {
	ID        string           `json:"id"`
	Code      rss.FeedCategory `json:"code"`
	Label     string           `json:"label"`
	// SourceCount is the number of news_sources referencing this category,
	// computed by the List read path (LEFT JOIN count). It is not a stored
	// column — Get/Create/Update leave it zero.
	SourceCount int       `json:"sourceCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Source is a registered upstream feed plus its conditional-GET fetch state. The
// discriminators (Language, Parser) and Category are shared feed named types.
type Source struct {
	ID         string            `json:"id"`
	SourceKey  string            `json:"sourceKey"`
	Name       string            `json:"name"`
	URL        string            `json:"url"`
	Language   rss.FeedLanguage `json:"language"`
	CategoryID string            `json:"categoryId"`
	// CategoryCode is the resolved category code (FeedCategory) for projection,
	// derived from CategoryID at read time — not a stored news_sources column.
	CategoryCode rss.FeedCategory `json:"category,omitempty"`
	Parser       rss.ParserKind   `json:"parser"`
	Enabled    bool              `json:"enabled"`

	// Conditional-GET fetch state (written by the refresh job).
	LastFetchedAt *time.Time `json:"lastFetchedAt,omitempty"`
	LastStatus    *string    `json:"lastStatus,omitempty"`
	ETag          *string    `json:"etag,omitempty"`
	LastModified  *string    `json:"lastModified,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Settings is the BR-local singleton row that governs the cron self-throttle for
// the auto-fetch refresh. A fixed cadence Dokploy cron fires
// `./refresh-feeds --if-due`; the CLI loads these settings and only runs a
// Refresh when auto-fetch is enabled AND the configured interval has elapsed
// since LastAutoFetchAt. Changing the cadence is a DB write here — never a cron
// reconfiguration.
//
// It is a projection of:
//
//	entities/news-settings/platforms/api-chi/md.yml
//
// The table holds exactly one row (singleton, enforced by a fixed boolean PK).
type Settings struct {
	AutoFetchEnabled         bool       `json:"autoFetchEnabled"`
	AutoFetchIntervalMinutes int        `json:"autoFetchIntervalMinutes"`
	LastAutoFetchAt          *time.Time `json:"lastAutoFetchAt,omitempty"`
	UpdatedAt                time.Time  `json:"updatedAt"`
}

// MinAutoFetchIntervalMinutes is the floor for the configurable auto-fetch
// interval — mirrored by the news_settings CHECK constraint and validated at the
// settings PUT endpoint. A too-frequent cadence risks upstream rate limits.
const MinAutoFetchIntervalMinutes = 5

// Item is one persisted, deduplicated archived article. Dedup is on
// (SourceID, GUID). HiddenAt NULL = visible; Featured is single-lead.
type Item struct {
	ID         string            `json:"id"`
	SourceID   string            `json:"sourceId"`
	GUID       string            `json:"guid"`
	Title      string            `json:"title"`
	Link       string            `json:"link"`
	Excerpt    string            `json:"excerpt"`
	ImageURL   *string           `json:"imageUrl,omitempty"`
	ImageR2URL *string           `json:"imageR2Url,omitempty"`
	Author     *string           `json:"author,omitempty"`
	PublishedAt *time.Time       `json:"publishedAt,omitempty"`
	CategoryID *string           `json:"categoryId,omitempty"`
	// CategoryCode is the resolved category code (FeedCategory) for the projection
	// — derived from CategoryID at read time, not a stored column.
	CategoryCode rss.FeedCategory `json:"category,omitempty"`
	Language     rss.FeedLanguage `json:"language"`
	SourceTags []string          `json:"sourceTags"`
	// Tags are curator-assigned labels (US-RC-02, task 2606-001) matched against
	// property tags via the shared slug formula. Distinct from SourceTags — the
	// raw RSS provenance chips, which are never matched.
	Tags       []string          `json:"tags"`
	HiddenAt   *time.Time        `json:"hiddenAt,omitempty"`
	Featured   bool              `json:"featured"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}
