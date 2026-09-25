package rss

// FeedLanguage is the source-assigned content language of a feed item. It is a
// dictionary-backed discriminator (never a raw string): the authoritative value
// set lives in the feed.language dictionary YAML (feed-language.yml). Upstream
// <language> tags are unreliable (Bangkok Post leaves it empty; Thaiger reports
// "en-US"/"en-th"), so language is assigned per source in the registry.
type FeedLanguage string

const (
	// LanguageEN is English-language content.
	LanguageEN FeedLanguage = "en"
	// LanguageTH is Thai-language content.
	LanguageTH FeedLanguage = "th"
)

// ParseFeedLanguage maps a raw string to a FeedLanguage, reporting whether the
// value is a known dictionary member. Unknown values map to ("", false).
func ParseFeedLanguage(s string) (FeedLanguage, bool) {
	l := FeedLanguage(s)
	if l.Valid() {
		return l, true
	}
	return "", false
}

// Valid reports whether l is one of the known feed.language dictionary values.
func (l FeedLanguage) Valid() bool {
	switch l {
	case LanguageEN, LanguageTH:
		return true
	}
	return false
}

// String returns the raw dictionary code.
func (l FeedLanguage) String() string { return string(l) }
