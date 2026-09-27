package rss

// FeedCategory is the normalized, source-assigned classification of a feed
// item. It is a dictionary-backed discriminator (never a raw string): the
// authoritative value set lives in the feed.category dictionary YAML
// (.sbx/workspace/.../defaults/feed-category.yml). Source `<category>` tags are
// free-text and unreliable, so the normalized category is assigned per source
// in the feed-sources registry, not parsed from the feed body.
type FeedCategory string

const (
	// CategoryProperty groups real-estate / housing news.
	CategoryProperty FeedCategory = "property"
	// CategoryBusiness groups business / finance / economy news.
	CategoryBusiness FeedCategory = "business"
	// CategoryGeneral is the catch-all for sources without a tighter fit.
	CategoryGeneral FeedCategory = "general"
)

// ParseFeedCategory maps a raw string to a FeedCategory, reporting whether the
// value is a known dictionary member. Unknown values map to ("", false) so
// callers can ignore an out-of-range filter param rather than 500.
func ParseFeedCategory(s string) (FeedCategory, bool) {
	c := FeedCategory(s)
	if c.Valid() {
		return c, true
	}
	return "", false
}

// Valid reports whether c is one of the known feed.category dictionary values.
func (c FeedCategory) Valid() bool {
	switch c {
	case CategoryProperty, CategoryBusiness, CategoryGeneral:
		return true
	}
	return false
}

// String returns the raw dictionary code.
func (c FeedCategory) String() string { return string(c) }
