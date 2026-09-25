package rss

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// RSS2Parser is the adapter for bare RSS 2.0 feeds (Bangkok Post): each
// channel/item carries title, link, description, pubDate and nothing else.
// Category and Language come from the source assignment, not the body.
type RSS2Parser struct{}

// rss2Doc is the minimal RSS 2.0 shape this adapter reads.
type rss2Doc struct {
	XMLName xml.Name    `xml:"rss"`
	Channel rss2Channel `xml:"channel"`
}

type rss2Channel struct {
	Items []rss2Item `xml:"item"`
}

type rss2Item struct {
	Title       string          `xml:"title"`
	Link        string          `xml:"link"`
	GUID        string          `xml:"guid"`
	Description string          `xml:"description"`
	PubDate     string          `xml:"pubDate"`
	Enclosures  []rss2Enclosure `xml:"enclosure"`
}

// rss2Enclosure is an RSS 2.0 <enclosure> (an optional item attachment). Only
// image/* enclosures are used as the item image; audio/video are ignored.
type rss2Enclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

// Parse decodes bare RSS 2.0. A malformed document returns a wrapped error and
// no items; a well-formed document with zero items returns (nil, nil).
func (RSS2Parser) Parse(data []byte, src FeedSource) ([]FeedItem, error) {
	var doc rss2Doc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("feed: parse rss2 (%s): %w", src.ID, err)
	}

	items := make([]FeedItem, 0, len(doc.Channel.Items))
	for _, it := range doc.Channel.Items {
		items = append(items, FeedItem{
			GUID:        firstNonEmpty(strings.TrimSpace(it.GUID), strings.TrimSpace(it.Link)),
			SourceID:    src.ID,
			Title:       strings.TrimSpace(it.Title),
			Link:        strings.TrimSpace(it.Link),
			Excerpt:     Excerpt(it.Description, DefaultExcerptLen),
			ImageURL:    rss2ImageURL(it),
			Author:      "", // bare RSS 2.0 carries no author
			PublishedAt: parsePubDate(it.PubDate),
			Category:    src.Category,
			Language:    src.Language,
			SourceTags:  nil,
		})
	}
	return items, nil
}

// rss2ImageURL applies the bare-RSS2 image fallback:
//  1. the first <enclosure> whose type is image/* (an explicit item image),
//  2. else the first <img src> inside the HTML description.
//
// Returns "" when neither is present (UI applies a branded placeholder). A
// non-image enclosure (audio/video) is ignored. og:image fetching is NOT done
// here — that is the P3 import-time pipeline, not the parser.
func rss2ImageURL(it rss2Item) string {
	for _, enc := range it.Enclosures {
		if enc.URL != "" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(enc.Type)), "image/") {
			return strings.TrimSpace(enc.URL)
		}
	}
	return ExtractImageFromContent(it.Description)
}

// firstNonEmpty returns the first non-empty string of its arguments.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// pubDateLayouts covers the RFC-1123 variants seen across the feeds, including
// Bangkok Post's non-standard zone-less form.
var pubDateLayouts = []string{
	time.RFC1123Z,               // Mon, 02 Jan 2006 15:04:05 -0700
	time.RFC1123,                // Mon, 02 Jan 2006 15:04:05 MST
	"Mon, 02 Jan 2006 15:04:05", // Bangkok Post channel pubDate (no zone)
	"Mon, _2 Jan 2006 15:04:05 -0700",
	"02 Jan 2006 15:04:05 -0700",
	time.RFC822Z,
	time.RFC822,
}

// parsePubDate parses an RSS pubDate across known layouts; returns the zero
// time when the value is empty or unparseable (item still surfaces, sorts last).
func parsePubDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range pubDateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
