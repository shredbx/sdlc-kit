package rss

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// WordPressParser is the adapter for RSS 2.0 feeds carrying the WordPress
// namespaces dc:creator, content:encoded, repeated <category>, and optional
// media:content/media:thumbnail (Thaiger, Prachachat, Khaosod). Author comes
// from dc:creator, SourceTags from the raw categories, ImageURL from the 3-step
// extractor (media:content > media:thumbnail > first content img > ""), and
// Category/Language from the source assignment.
type WordPressParser struct{}

// wpDoc is the WordPress RSS shape. Namespaced elements are matched by local
// name (encoding/xml ignores prefixes), which is sufficient here because the
// local names (creator, encoded, content, thumbnail) are unambiguous in feed
// items.
type wpDoc struct {
	XMLName xml.Name  `xml:"rss"`
	Channel wpChannel `xml:"channel"`
}

type wpChannel struct {
	Items []wpItem `xml:"item"`
}

type wpItem struct {
	Title      string         `xml:"title"`
	Link       string         `xml:"link"`
	GUID       string         `xml:"guid"`
	Creator    string         `xml:"creator"` // dc:creator
	PubDate    string         `xml:"pubDate"`
	Categories []string       `xml:"category"`
	Desc       string         `xml:"description"`
	Encoded    string         `xml:"encoded"` // content:encoded
	Media      []wpMediaBlock `xml:"content"` // media:content (may nest media:thumbnail)
	Thumbnail  wpMediaURL     `xml:"thumbnail"` // bare media:thumbnail
}

// wpMediaBlock is a media:content element; it may contain a nested
// media:thumbnail.
type wpMediaBlock struct {
	URL       string     `xml:"url,attr"`
	Medium    string     `xml:"medium,attr"`
	Thumbnail wpMediaURL `xml:"thumbnail"`
}

type wpMediaURL struct {
	URL string `xml:"url,attr"`
}

// Parse decodes WordPress RSS. A malformed document returns a wrapped error and
// no items; a well-formed document with zero items returns (nil, nil).
func (WordPressParser) Parse(data []byte, src FeedSource) ([]FeedItem, error) {
	var doc wpDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("feed: parse wordpress (%s): %w", src.ID, err)
	}

	items := make([]FeedItem, 0, len(doc.Channel.Items))
	for _, it := range doc.Channel.Items {
		tags := cleanTags(it.Categories)
		items = append(items, FeedItem{
			GUID:        firstNonEmpty(strings.TrimSpace(it.GUID), strings.TrimSpace(it.Link)),
			SourceID:    src.ID,
			Title:       strings.TrimSpace(it.Title),
			Link:        strings.TrimSpace(it.Link),
			Excerpt:     wpExcerpt(it),
			ImageURL:    wpImageURL(it),
			Author:      strings.TrimSpace(it.Creator),
			PublishedAt: parsePubDate(it.PubDate),
			Category:    src.Category,
			Language:    src.Language,
			SourceTags:  tags,
		})
	}
	return items, nil
}

// wpExcerpt prefers <description>, falling back to content:encoded, then
// cleans HTML to plain text.
func wpExcerpt(it wpItem) string {
	src := it.Desc
	if strings.TrimSpace(tagRe.ReplaceAllString(src, "")) == "" {
		src = it.Encoded
	}
	return Excerpt(src, DefaultExcerptLen)
}

// wpImageURL applies the 3-step fallback:
//  1. media:content url (or its nested media:thumbnail url)
//  2. a bare media:thumbnail url
//  3. the first <img src> in content:encoded
//
// Returns "" when none is found (UI applies a branded placeholder).
func wpImageURL(it wpItem) string {
	for _, m := range it.Media {
		if m.URL != "" {
			return m.URL
		}
		if m.Thumbnail.URL != "" {
			return m.Thumbnail.URL
		}
	}
	if it.Thumbnail.URL != "" {
		return it.Thumbnail.URL
	}
	return ExtractImageFromContent(it.Encoded)
}

// cleanTags trims and drops empty raw <category> values, preserving order.
func cleanTags(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, c := range raw {
		if c = strings.TrimSpace(c); c != "" {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
