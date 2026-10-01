package rss_test

import (
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/rss"
)

var thaigerSource = rss.FeedSource{
	ID:       "thethaiger-property",
	Name:     "Thaiger — Property",
	URL:      "https://thethaiger.com/thai-life/property/feed",
	Language: rss.LanguageEN,
	Category: rss.CategoryProperty,
	Parser:   rss.ParserWordPress,
	Enabled:  true,
}

var prachachatSource = rss.FeedSource{
	ID:       "prachachat-property",
	Name:     "Prachachat — Property",
	URL:      "https://www.prachachat.net/property/feed",
	Language: rss.LanguageTH,
	Category: rss.CategoryProperty,
	Parser:   rss.ParserWordPress,
	Enabled:  true,
}

var khaosodSource = rss.FeedSource{
	ID:       "khaosod",
	Name:     "Khaosod",
	URL:      "https://www.khaosod.co.th/feed",
	Language: rss.LanguageTH,
	Category: rss.CategoryGeneral,
	Parser:   rss.ParserWordPress,
	Enabled:  true,
}

func TestWordPressParser_Thaiger_AuthorAndTags(t *testing.T) {
	items, err := rss.WordPressParser{}.Parse(readFixture(t, "thethaiger-property.xml"), thaigerSource)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
	first := items[0]
	// Author from dc:creator (CDATA).
	if first.Author != "Thaiger" {
		t.Errorf("author = %q, want Thaiger", first.Author)
	}
	if items[1].Author != "Sophie Hartman" {
		t.Errorf("author[1] = %q, want Sophie Hartman", items[1].Author)
	}
	// SourceTags from repeated <category> (raw, CDATA).
	wantTags := []string{"Guides", "Property", "Bangkok", "Condos"}
	if len(first.SourceTags) != len(wantTags) {
		t.Fatalf("sourceTags = %v, want %v", first.SourceTags, wantTags)
	}
	for i, tag := range wantTags {
		if first.SourceTags[i] != tag {
			t.Errorf("sourceTags[%d] = %q, want %q", i, first.SourceTags[i], tag)
		}
	}
	// Category + Language from the source assignment.
	if first.Category != rss.CategoryProperty || first.Language != rss.LanguageEN {
		t.Errorf("category/lang = %q/%q, want property/en", first.Category, first.Language)
	}
}

func TestWordPressParser_Thaiger_ImageFromContent(t *testing.T) {
	items, err := rss.WordPressParser{}.Parse(readFixture(t, "thethaiger-property.xml"), thaigerSource)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// No media:*; image comes from the first <img> in content:encoded.
	if items[0].ImageURL != "https://thethaiger.com/wp-content/uploads/2026/05/bangkok-condo-listing.jpg" {
		t.Errorf("item0 imageURL = %q", items[0].ImageURL)
	}
	// Second item: img appears after a <p> — still extracted.
	if items[1].ImageURL != "https://thethaiger.com/wp-content/uploads/2026/05/bangkok-lease.jpg" {
		t.Errorf("item1 imageURL = %q", items[1].ImageURL)
	}
	// Third item: no img anywhere → empty (UI applies placeholder).
	if items[2].ImageURL != "" {
		t.Errorf("item2 imageURL = %q, want empty", items[2].ImageURL)
	}
}

func TestWordPressParser_Thaiger_ExcerptPlainNoEditorClasses(t *testing.T) {
	items, err := rss.WordPressParser{}.Parse(readFixture(t, "thethaiger-property.xml"), thaigerSource)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, it := range items {
		if strings.ContainsAny(it.Excerpt, "<>") {
			t.Errorf("excerpt has tags: %q", it.Excerpt)
		}
		if strings.Contains(it.Excerpt, "wp-post-image") || strings.Contains(it.Excerpt, "attachment-full") {
			t.Errorf("excerpt leaked WP image classes: %q", it.Excerpt)
		}
		if it.Excerpt == "" {
			t.Errorf("excerpt empty for %q", it.Title)
		}
	}
}

func TestWordPressParser_Prachachat_MediaContentImage(t *testing.T) {
	items, err := rss.WordPressParser{}.Parse(readFixture(t, "prachachat-property.xml"), prachachatSource)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
	// Step 1: media:content wins.
	if items[0].ImageURL != "https://www.prachachat.net/wp-content/uploads/2026/05/1-pokpop.jpg" {
		t.Errorf("item0 imageURL = %q (want media:content)", items[0].ImageURL)
	}
	// Author from dc:creator.
	if items[0].Author != "SUB_SU" {
		t.Errorf("item0 author = %q, want SUB_SU", items[0].Author)
	}
	// Step 3: item without media:* but with <img> in content:encoded.
	if items[2].ImageURL != "https://www.prachachat.net/wp-content/uploads/2026/05/3-rate.jpg" {
		t.Errorf("item2 imageURL = %q (want content img fallback)", items[2].ImageURL)
	}
	// Language assigned from source = th.
	if items[0].Language != rss.LanguageTH {
		t.Errorf("item0 language = %q, want th", items[0].Language)
	}
}

func TestWordPressParser_Prachachat_CleansLeakedEditorMarkup(t *testing.T) {
	items, err := rss.WordPressParser{}.Parse(readFixture(t, "prachachat-property.xml"), prachachatSource)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// Item 2 has leaked ChatGPT editor block + "Copy code".
	ex := items[1].Excerpt
	if strings.ContainsAny(ex, "<>") {
		t.Errorf("excerpt has tags: %q", ex)
	}
	if strings.Contains(ex, "chatgpt") || strings.Contains(ex, "message-author-role") || strings.Contains(ex, "Copy code") {
		t.Errorf("excerpt leaked editor markup: %q", ex)
	}
	if !strings.Contains(ex, "ธุรกิจรับสร้างบ้าน") {
		t.Errorf("excerpt dropped the real text: %q", ex)
	}
}

func TestWordPressParser_Khaosod_MediaFallbackChain(t *testing.T) {
	items, err := rss.WordPressParser{}.Parse(readFixture(t, "khaosod.xml"), khaosodSource)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
	// Item 0: full media:content (with nested thumbnail) → media:content url.
	if items[0].ImageURL != "https://www.khaosod.co.th/wpapp/uploads/2026/05/44455.jpg" {
		t.Errorf("item0 imageURL = %q", items[0].ImageURL)
	}
	// Item 1: only a bare media:thumbnail.
	if items[1].ImageURL != "https://www.khaosod.co.th/wpapp/uploads/2026/05/55566.jpg" {
		t.Errorf("item1 imageURL = %q (want media:thumbnail)", items[1].ImageURL)
	}
	// Item 2: no media, no content img → empty.
	if items[2].ImageURL != "" {
		t.Errorf("item2 imageURL = %q, want empty", items[2].ImageURL)
	}
	// Category assigned from source = general.
	if items[0].Category != rss.CategoryGeneral {
		t.Errorf("item0 category = %q, want general", items[0].Category)
	}
}

func TestWordPressParser_GUIDUsesGuidTag(t *testing.T) {
	items, err := rss.WordPressParser{}.Parse(readFixture(t, "thethaiger-property.xml"), thaigerSource)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if items[0].GUID != "https://thethaiger.com/?p=1004244" {
		t.Errorf("GUID = %q, want the <guid> value", items[0].GUID)
	}
}

func TestWordPressParser_EmptyAndMalformed(t *testing.T) {
	items, err := rss.WordPressParser{}.Parse(readFixture(t, "empty-feed.xml"), thaigerSource)
	if err != nil {
		t.Fatalf("empty feed should not error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("empty feed should yield 0 items, got %d", len(items))
	}

	items, err = rss.WordPressParser{}.Parse(readFixture(t, "malformed.xml"), thaigerSource)
	if err == nil {
		t.Error("malformed XML should return an error")
	}
	if len(items) != 0 {
		t.Errorf("malformed XML should yield 0 items, got %d", len(items))
	}
}
