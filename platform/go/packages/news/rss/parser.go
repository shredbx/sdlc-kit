package rss

import "fmt"

// Parser turns raw feed bytes into normalized FeedItems for a given source.
// Each ParserKind has exactly one adapter; adding a feed shape = a new adapter
// + a new ParserKind, with no change to existing adapters (Rule #9).
type Parser interface {
	// Parse decodes data for src. A malformed document yields a wrapped error
	// and no items; a well-formed document with zero items yields (nil, nil).
	Parse(data []byte, src FeedSource) ([]FeedItem, error)
}

// ErrUnknownParserKind is returned by ParserFor for a kind with no adapter.
var ErrUnknownParserKind = fmt.Errorf("feed: unknown parser kind")

// ParserFor returns the adapter for kind, or ErrUnknownParserKind. A registry
// loader can call this at startup to fail fast on an unconfigured parser.
func ParserFor(kind ParserKind) (Parser, error) {
	switch kind {
	case ParserRSS2:
		return RSS2Parser{}, nil
	case ParserWordPress:
		return WordPressParser{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownParserKind, kind)
	}
}
