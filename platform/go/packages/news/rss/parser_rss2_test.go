package rss_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/rss"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

var bangkokpostSource = rss.FeedSource{
	ID:       "bangkokpost-property",
	Name:     "Bangkok Post — Property",
	URL:      "https://www.bangkokpost.com/rss/data/property.xml",
	Language: rss.LanguageEN,
	Category: rss.CategoryProperty,
	Parser:   rss.ParserRSS2,
	Enabled:  true,
}

func TestRSS2Parser_FieldMapping(t *testing.T) {
	items, err := rss.RSS2Parser{}.Parse(readFixture(t, "bangkokpost-property.xml"), bangkokpostSource)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}

	first := items[0]
	if first.Title != "Sansiri, SCB eye young affluent homebuyers" {
		t.Errorf("title = %q", first.Title)
	}
	if first.Link != "https://www.bangkokpost.com/property/3262048/sansiri-scb-eye-young-affluent-homebuyers" {
		t.Errorf("link = %q", first.Link)
	}
	// Category + Language come from the source assignment, not the body.
	if first.Category != rss.CategoryProperty {
		t.Errorf("category = %q, want property", first.Category)
	}
	if first.Language != rss.LanguageEN {
		t.Errorf("language = %q, want en", first.Language)
	}
	if first.SourceID != "bangkokpost-property" {
		t.Errorf("sourceID = %q", first.SourceID)
	}
	// Bare RSS 2.0: no image, no author.
	if first.ImageURL != "" {
		t.Errorf("imageURL = %q, want empty (bare feed)", first.ImageURL)
	}
	if first.Author != "" {
		t.Errorf("author = %q, want empty (bare feed)", first.Author)
	}
	// pubDate parsed.
	want := time.Date(2026, 5, 28, 5, 56, 0, 0, time.FixedZone("", 7*3600))
	if !first.PublishedAt.Equal(want) {
		t.Errorf("publishedAt = %v, want %v", first.PublishedAt, want)
	}
}

func TestRSS2Parser_ExcerptIsPlainText(t *testing.T) {
	items, err := rss.RSS2Parser{}.Parse(readFixture(t, "bangkokpost-property.xml"), bangkokpostSource)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, it := range items {
		if strings.ContainsAny(it.Excerpt, "<>") {
			t.Errorf("excerpt has tags: %q", it.Excerpt)
		}
		if strings.Contains(it.Excerpt, "&amp;") || strings.Contains(it.Excerpt, "&mdash;") {
			t.Errorf("excerpt has escaped entities: %q", it.Excerpt)
		}
	}
	// The entity-heavy third item ("&amp;mdash;") must decode to an em dash.
	third := items[2]
	if !strings.Contains(third.Excerpt, "—") {
		t.Errorf("third excerpt should decode em dash: %q", third.Excerpt)
	}
}

func TestRSS2Parser_GUIDFallsBackToLink(t *testing.T) {
	// Bangkok Post items have no <guid> — GUID must fall back to the link.
	items, err := rss.RSS2Parser{}.Parse(readFixture(t, "bangkokpost-property.xml"), bangkokpostSource)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, it := range items {
		if it.GUID == "" {
			t.Errorf("GUID empty for %q; should fall back to link", it.Title)
		}
		if it.GUID != it.Link {
			t.Errorf("GUID %q should equal link %q when no <guid>", it.GUID, it.Link)
		}
	}
}

func TestRSS2Parser_EmptyFeed(t *testing.T) {
	items, err := rss.RSS2Parser{}.Parse(readFixture(t, "empty-feed.xml"), bangkokpostSource)
	if err != nil {
		t.Fatalf("empty feed should parse without error, got %v", err)
	}
	if len(items) != 0 {
		t.Errorf("empty feed should yield 0 items, got %d", len(items))
	}
}

func TestRSS2Parser_Malformed(t *testing.T) {
	// Must not panic; returns an error and no items.
	items, err := rss.RSS2Parser{}.Parse(readFixture(t, "malformed.xml"), bangkokpostSource)
	if err == nil {
		t.Error("malformed XML should return an error")
	}
	if len(items) != 0 {
		t.Errorf("malformed XML should yield 0 items, got %d", len(items))
	}
}

// rss2ImagesSource is the bare-RSS2 source for the image-extraction fixture.
var rss2ImagesSource = rss.FeedSource{
	ID:       "rss2-images",
	Name:     "RSS2 Images Sample",
	URL:      "https://example.com/rss2-images",
	Language: rss.LanguageEN,
	Category: rss.CategoryProperty,
	Parser:   rss.ParserRSS2,
	Enabled:  true,
}

// TC-06 (SC-NEWS-01) — rss2 image extraction: an <enclosure type="image/*">
// supplies the image; failing that, the first <img> in the description supplies
// it; a non-image enclosure (audio) is ignored so its item carries no image.
func TestRSS2_ImageExtraction_EnclosureAndImg(t *testing.T) {
	items, err := rss.RSS2Parser{}.Parse(readFixture(t, "rss2-images.xml"), rss2ImagesSource)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}

	// Item 1: enclosure image wins.
	if got, want := items[0].ImageURL, "https://cdn.example.com/a.jpg"; got != want {
		t.Errorf("item[0] enclosure image = %q, want %q", got, want)
	}
	// Item 2: no enclosure → first <img> in the HTML description.
	if got, want := items[1].ImageURL, "https://cdn.example.com/b.png"; got != want {
		t.Errorf("item[1] description-img image = %q, want %q", got, want)
	}
	// Item 3: non-image enclosure (audio) ignored, no inline img → empty.
	if items[2].ImageURL != "" {
		t.Errorf("item[2] image = %q, want empty (audio enclosure must be ignored)", items[2].ImageURL)
	}
}
