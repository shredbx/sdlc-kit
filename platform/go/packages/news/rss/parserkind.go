package rss

// ParserKind selects which Parser adapter handles a source's XML shape. It is a
// dictionary-backed discriminator (never a raw string): the authoritative value
// set lives in the feed.parser-kind dictionary YAML (feed-parser-kind.yml).
// Adding a new feed shape = a new ParserKind constant + a new parser_*.go
// adapter, with zero changes to the dispatch in ParserFor.
type ParserKind string

const (
	// ParserRSS2 is the bare RSS 2.0 shape: channel/item with
	// title, link, description, pubDate (Bangkok Post).
	ParserRSS2 ParserKind = "rss2"
	// ParserWordPress is RSS 2.0 plus the dc:, content:, category[] and
	// optional media: namespaces emitted by WordPress (the other sources).
	ParserWordPress ParserKind = "wordpress"
)

// ParseParserKind maps a raw string to a ParserKind, reporting whether the
// value is a known dictionary member. Unknown values map to ("", false) so a
// registry loader can fail fast at startup on a bad parser kind.
func ParseParserKind(s string) (ParserKind, bool) {
	k := ParserKind(s)
	if k.Valid() {
		return k, true
	}
	return "", false
}

// Valid reports whether k is one of the known feed.parser-kind dictionary values.
func (k ParserKind) Valid() bool {
	switch k {
	case ParserRSS2, ParserWordPress:
		return true
	}
	return false
}

// String returns the raw dictionary code.
func (k ParserKind) String() string { return string(k) }
