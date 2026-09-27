package cms

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/address"
	"github.com/shredbx/sbx-core/pkg/seo"
)

func strptr(s string) *string { return &s }

// TC-slug-validate — Slug is the URL key: required + kebab-case. Mirrors the
// entity.yml `slug` constraint (^[a-z0-9]+(?:-[a-z0-9]+)*$).
func TestSlug_Validate(t *testing.T) {
	tests := []struct {
		name    string
		slug    Slug
		wantErr bool
	}{
		{"simple", Slug("about"), false},
		{"kebab", Slug("contact-us"), false},
		{"with-digits", Slug("page-2"), false},
		{"empty", Slug(""), true},
		{"whitespace", Slug("   "), true},
		{"uppercase", Slug("About"), true},
		{"underscore", Slug("about_us"), true},
		{"leading-dash", Slug("-about"), true},
		{"trailing-dash", Slug("about-"), true},
		{"double-dash", Slug("about--us"), true},
		{"space inside", Slug("about us"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.slug.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() = nil; want error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() = %v; want nil", err)
			}
		})
	}
}

// TC-cmspage-validate — slug is the hard requirement; body_markdown (when
// present) must be within the length cap. Mirrors entity.yml constraints.
func TestCmsPage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		page    CmsPage
		wantErr bool
	}{
		{"valid minimal (slug only)", CmsPage{Slug: "about"}, false},
		{"valid with title + body", CmsPage{Slug: "about", Title: strptr("About"), BodyMarkdown: strptr("# Hi")}, false},
		{"missing slug", CmsPage{}, true},
		{"bad slug", CmsPage{Slug: "About Us"}, true},
		{"body too long", CmsPage{Slug: "about", BodyMarkdown: strptr(strings.Repeat("x", maxBodyMarkdownLen+1))}, true},
		{"body at cap", CmsPage{Slug: "about", BodyMarkdown: strptr(strings.Repeat("x", maxBodyMarkdownLen))}, false},
		// seo_meta SERP length invariant propagates through CmsPage.Validate (P0-A):
		// a meta_title > 60 runes is rejected at the save boundary (it renders in the
		// SERP), while a valid override passes.
		{"seo meta_title too long", CmsPage{Slug: "about", SeoMeta: seo.SeoMeta{MetaTitle: strings.Repeat("x", seo.MaxMetaTitleLen+1)}}, true},
		{"seo meta valid", CmsPage{Slug: "about", SeoMeta: seo.SeoMeta{MetaTitle: "About Bestie", MetaDescription: "Honest island property advice."}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.page.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() = nil; want error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() = %v; want nil", err)
			}
		})
	}
}

// TC-details-jsonb — PageDetails round-trips through its JSONB Scan/Value: a
// populated contact block marshals + scans back identically; an empty (zero)
// PageDetails Values to SQL NULL, and a NULL source Scans back to a zero value
// (Contact nil). Mirrors pkg/contact's SocialNetworkList/ExtensionData pattern.
func TestPageDetails_ScanValue_RoundTrip(t *testing.T) {
	d := PageDetails{Contact: &PageContact{
		Phone:    "+66 98-348-0292",
		WhatsApp: "66983480292",
		Email:    "bestierealestate@gmail.com",
		Hours:    "Mon–Sat 9:00–18:00",
		Address:  address.Address{City: "Koh Phangan", Province: "Surat Thani", Country: "TH"},
	}}
	v, err := d.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if v == nil {
		t.Fatal("Value of populated details = nil; want JSON bytes")
	}

	var got PageDetails
	if err := got.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got.Contact == nil {
		t.Fatal("scanned details.Contact = nil")
	}
	if got.Contact.Phone != d.Contact.Phone || got.Contact.WhatsApp != d.Contact.WhatsApp ||
		got.Contact.Email != d.Contact.Email || got.Contact.Hours != d.Contact.Hours {
		t.Errorf("contact round-trip mismatch: got %+v want %+v", *got.Contact, *d.Contact)
	}
	if got.Contact.Address.City != "Koh Phangan" || got.Contact.Address.Country != "TH" {
		t.Errorf("address round-trip mismatch: got %+v", got.Contact.Address)
	}

	// Empty (zero) details Values to SQL NULL — no row carries an empty blob.
	ev, err := (PageDetails{}).Value()
	if err != nil {
		t.Fatalf("Value(empty): %v", err)
	}
	if ev != nil {
		t.Errorf("Value(empty) = %v; want nil (SQL NULL)", ev)
	}

	// NULL source Scans to a zero value (Contact nil) without error.
	var z PageDetails
	if err := z.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if z.Contact != nil {
		t.Error("Scan(nil) should leave Contact nil")
	}
}

// TC-details-hero — PageDetails carries optional hero fields (headline +
// cover image) in the SAME details JSONB. IsZero stays true for fully-empty
// details (→ SQL NULL) but flips false when only Headline is set (a hero-only
// page still persists). The hero fields round-trip through Value()/Scan(), both
// alone and alongside a Contact block.
func TestPageDetails_Hero_IsZero(t *testing.T) {
	if !(PageDetails{}).IsZero() {
		t.Error("empty PageDetails should be IsZero")
	}
	if (PageDetails{Headline: strptr("Sell Your Property")}).IsZero() {
		t.Error("PageDetails with only Headline should NOT be IsZero")
	}
	if (PageDetails{CoverImageURL: strptr("https://r2/cover.webp")}).IsZero() {
		t.Error("PageDetails with only CoverImageURL should NOT be IsZero")
	}
	if (PageDetails{CoverImageURLs: []string{"https://r2/a.webp"}}).IsZero() {
		t.Error("PageDetails with only CoverImageURLs should NOT be IsZero")
	}
	if (PageDetails{SearchPrompt: strptr("Type a keyword above")}).IsZero() {
		t.Error("PageDetails with only SearchPrompt should NOT be IsZero")
	}
	if (PageDetails{Eyebrow: strptr("WATCH · FOLLOW · EXPLORE")}).IsZero() {
		t.Error("PageDetails with only Eyebrow should NOT be IsZero")
	}
	// Each optional hero color, set alone, flips IsZero false (a color-only
	// page still persists). Mirrors the SearchPrompt additive-JSONB rule.
	if (PageDetails{BgColor: strptr("#0D4F4F")}).IsZero() {
		t.Error("PageDetails with only BgColor should NOT be IsZero")
	}
	if (PageDetails{TextColor: strptr("#FFFFFF")}).IsZero() {
		t.Error("PageDetails with only TextColor should NOT be IsZero")
	}
	if (PageDetails{TitleColor: strptr("#C8A851")}).IsZero() {
		t.Error("PageDetails with only TitleColor should NOT be IsZero")
	}
	if (PageDetails{SubtitleColor: strptr("#E5E5E5")}).IsZero() {
		t.Error("PageDetails with only SubtitleColor should NOT be IsZero")
	}
	if (PageDetails{EyebrowColor: strptr("#C8A851")}).IsZero() {
		t.Error("PageDetails with only EyebrowColor should NOT be IsZero")
	}
}

// Slideshow behavior fields (2606-128 tail): each optional field, set alone,
// flips IsZero false — a details carrying ONLY a duration/zoom setting still
// persists (and must NOT round-trip as SQL NULL past the public projection gate).
func TestPageDetails_Slideshow_IsZero(t *testing.T) {
	dur := 5
	if (PageDetails{SlideDurationS: &dur}).IsZero() {
		t.Error("PageDetails with only SlideDurationS should NOT be IsZero")
	}
	on := true
	if (PageDetails{ZoomEnabled: &on}).IsZero() {
		t.Error("PageDetails with only ZoomEnabled should NOT be IsZero")
	}
	dir := ZoomDirectionIn
	if (PageDetails{ZoomDirection: &dir}).IsZero() {
		t.Error("PageDetails with only ZoomDirection should NOT be IsZero")
	}
	scale := 1.08
	if (PageDetails{ZoomScale: &scale}).IsZero() {
		t.Error("PageDetails with only ZoomScale should NOT be IsZero")
	}
}

// Validate pins the slideshow bounds (mirrored by the admin form's clamps and
// the web helper hero-slideshow.ts): duration 2–30s, direction in|out, scale
// 1.01–1.5. Boundary values pass; one step outside each bound rejects.
func TestPageDetails_Validate_Slideshow(t *testing.T) {
	ok := func(d PageDetails) {
		t.Helper()
		if err := d.Validate(); err != nil {
			t.Errorf("Validate(%+v) = %v; want nil", d, err)
		}
	}
	bad := func(d PageDetails) {
		t.Helper()
		if err := d.Validate(); err == nil {
			t.Errorf("Validate(%+v) = nil; want error", d)
		}
	}

	lo, hi, under, over := 2, 30, 1, 31
	ok(PageDetails{SlideDurationS: &lo})
	ok(PageDetails{SlideDurationS: &hi})
	bad(PageDetails{SlideDurationS: &under})
	bad(PageDetails{SlideDurationS: &over})

	in, out, diag := ZoomDirectionIn, ZoomDirectionOut, ZoomDirection("diagonal")
	ok(PageDetails{ZoomDirection: &in})
	ok(PageDetails{ZoomDirection: &out})
	bad(PageDetails{ZoomDirection: &diag})

	sLo, sHi, sUnder, sOver := 1.01, 1.5, 1.0, 1.51
	ok(PageDetails{ZoomScale: &sLo})
	ok(PageDetails{ZoomScale: &sHi})
	bad(PageDetails{ZoomScale: &sUnder})
	bad(PageDetails{ZoomScale: &sOver})
}

// BrowseFeaturedTypes (B5-6) — pointer three-state: nil (unset), &[] (explicitly
// none), &[codes]; Validate gates count + the dictionary-code grammar; IsZero
// counts a non-nil pointer (even empty) as data; JSON round-trips the empty list.
func TestPageDetails_BrowseFeaturedTypes(t *testing.T) {
	ok := func(d PageDetails) {
		t.Helper()
		if err := d.Validate(); err != nil {
			t.Errorf("Validate(%+v) = %v; want nil", d, err)
		}
	}
	bad := func(d PageDetails) {
		t.Helper()
		if err := d.Validate(); err == nil {
			t.Errorf("Validate(%+v) = nil; want error", d)
		}
	}

	ok(PageDetails{})                                 // nil = unset
	ok(PageDetails{BrowseFeaturedTypes: &[]string{}}) // explicit none
	ok(PageDetails{BrowseFeaturedTypes: &[]string{"business", "pool-villa"}})
	bad(PageDetails{BrowseFeaturedTypes: &[]string{""}})         // empty code
	bad(PageDetails{BrowseFeaturedTypes: &[]string{"Business"}}) // uppercase
	bad(PageDetails{BrowseFeaturedTypes: &[]string{"a;drop"}})   // charset
	seven := &[]string{"a", "b", "c", "d", "e", "f", "g"}
	bad(PageDetails{BrowseFeaturedTypes: seven}) // over cap

	// IsZero: a non-nil EMPTY list is data (explicit none must persist, not
	// collapse to NULL and resurrect the default).
	if (PageDetails{BrowseFeaturedTypes: &[]string{}}).IsZero() {
		t.Error("IsZero(&[]) = true; explicit-none must persist")
	}

	// JSON round-trip keeps the three states distinct.
	empty := PageDetails{BrowseFeaturedTypes: &[]string{}}
	b, err := json.Marshal(empty)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"browse_featured_types":[]`) {
		t.Errorf("explicit-none must serialize as []; got %s", b)
	}
	var back PageDetails
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.BrowseFeaturedTypes == nil || len(*back.BrowseFeaturedTypes) != 0 {
		t.Errorf("explicit-none did not round-trip: %+v", back.BrowseFeaturedTypes)
	}
	var unset PageDetails
	if err := json.Unmarshal([]byte(`{}`), &unset); err != nil {
		t.Fatalf("unmarshal {}: %v", err)
	}
	if unset.BrowseFeaturedTypes != nil {
		t.Error("absent key must decode to nil (unset)")
	}
}

// CoverImageURLs (2606-128) is the optional ORDERED hero-slideshow list. It
// round-trips through the details JSONB preserving order, and Validate caps the
// count and rejects empty entries (each renders as a public <img> layer).
func TestPageDetails_CoverImageURLs(t *testing.T) {
	// Round-trip through Value()/Scan() preserves order.
	d := PageDetails{CoverImageURLs: []string{"https://r2/1.webp", "https://r2/2.webp", "https://r2/3.webp"}}
	v, err := d.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	var got PageDetails
	if err := got.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got.CoverImageURLs) != 3 || got.CoverImageURLs[0] != "https://r2/1.webp" || got.CoverImageURLs[2] != "https://r2/3.webp" {
		t.Errorf("cover_image_urls round-trip mismatch: got %v", got.CoverImageURLs)
	}

	// Validate accepts an empty list (nil) and a normal list.
	if err := (PageDetails{}).Validate(); err != nil {
		t.Errorf("empty details Validate() = %v; want nil", err)
	}
	if err := (PageDetails{CoverImageURLs: []string{"https://r2/a.webp"}}).Validate(); err != nil {
		t.Errorf("single-image list Validate() = %v; want nil", err)
	}

	// Validate rejects an empty/whitespace entry.
	if err := (PageDetails{CoverImageURLs: []string{"https://r2/a.webp", "  "}}).Validate(); err == nil {
		t.Error("Validate() with an empty list entry = nil; want error")
	}

	// Validate rejects a list over the cap.
	over := make([]string, maxHeroImages+1)
	for i := range over {
		over[i] = "https://r2/x.webp"
	}
	if err := (PageDetails{CoverImageURLs: over}).Validate(); err == nil {
		t.Errorf("Validate() with %d images = nil; want error", maxHeroImages+1)
	}
}

// Homepage section order + FAQ block (2607-041 scope-08) — SectionsOrder is an
// opaque ordered token list and FaqHome an optional block config, both additive in
// the details JSONB. Each flips IsZero false alone, round-trips preserving order +
// fields, and Validate bounds the token count and the FAQ caps. Tokens are NEVER
// grammar-checked (web-owned) — an "unknown" token passes Validate.
func TestPageDetails_HomeSectionsOrder(t *testing.T) {
	// IsZero — either field, set alone, persists (must not collapse to SQL NULL).
	if (PageDetails{SectionsOrder: []string{"closing"}}).IsZero() {
		t.Error("PageDetails with only SectionsOrder should NOT be IsZero")
	}
	if (PageDetails{FaqHome: &FaqHomeConfig{}}).IsZero() {
		t.Error("PageDetails with only FaqHome (bare enable) should NOT be IsZero")
	}
	if !(PageDetails{}).IsZero() {
		t.Error("empty PageDetails should be IsZero")
	}

	// Round-trip through Value()/Scan() preserves token ORDER and every FAQ field.
	d := PageDetails{
		SectionsOrder: []string{"collection:abc", "faq", "collection:def", "closing"},
		FaqHome:       &FaqHomeConfig{CategorySlug: "buying", Limit: 6, Title: "Common questions"},
	}
	v, err := d.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	var got PageDetails
	if err := got.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got.SectionsOrder) != 4 || got.SectionsOrder[0] != "collection:abc" ||
		got.SectionsOrder[1] != "faq" || got.SectionsOrder[3] != "closing" {
		t.Errorf("sections_order round-trip mismatch: got %v", got.SectionsOrder)
	}
	if got.FaqHome == nil || got.FaqHome.CategorySlug != "buying" ||
		got.FaqHome.Limit != 6 || got.FaqHome.Title != "Common questions" {
		t.Errorf("faq_home round-trip mismatch: got %+v", got.FaqHome)
	}

	// Absent keys decode to the zero shape (nil FaqHome, empty SectionsOrder).
	var unset PageDetails
	if err := json.Unmarshal([]byte(`{}`), &unset); err != nil {
		t.Fatalf("unmarshal {}: %v", err)
	}
	if unset.FaqHome != nil || len(unset.SectionsOrder) != 0 {
		t.Errorf("absent keys must decode to zero: %+v / %v", unset.FaqHome, unset.SectionsOrder)
	}

	// A bare enable ({}) round-trips as an all-defaults FAQ block (omitempty fields).
	bare, err := json.Marshal(PageDetails{FaqHome: &FaqHomeConfig{}})
	if err != nil {
		t.Fatalf("marshal bare faq: %v", err)
	}
	if !strings.Contains(string(bare), `"faq_home":{}`) {
		t.Errorf("bare FaqHome must serialize as {}; got %s", bare)
	}

	// Validate — empty ok; a normal order + FAQ ok; over-cap tokens, out-of-range
	// limit and over-long copy reject. An UNKNOWN token is accepted (web-owned grammar).
	ok := func(d PageDetails) {
		t.Helper()
		if err := d.Validate(); err != nil {
			t.Errorf("Validate(%+v) = %v; want nil", d, err)
		}
	}
	bad := func(d PageDetails) {
		t.Helper()
		if err := d.Validate(); err == nil {
			t.Errorf("Validate(%+v) = nil; want error", d)
		}
	}
	ok(PageDetails{})
	ok(d)
	ok(PageDetails{SectionsOrder: []string{"totally-made-up-token"}}) // grammar is web-owned
	ok(PageDetails{FaqHome: &FaqHomeConfig{Limit: 0}})                // 0 = consumer default

	over := make([]string, maxHomeSectionTokens+1)
	for i := range over {
		over[i] = "closing"
	}
	bad(PageDetails{SectionsOrder: over})
	bad(PageDetails{FaqHome: &FaqHomeConfig{Limit: maxHomeFaqItems + 1}})
	bad(PageDetails{FaqHome: &FaqHomeConfig{Limit: -1}})
	bad(PageDetails{FaqHome: &FaqHomeConfig{Title: strings.Repeat("x", 201)}})
	bad(PageDetails{FaqHome: &FaqHomeConfig{CategorySlug: strings.Repeat("x", 101)}})
}

// Homepage map section (task 2607-133 R1, ruling D1) — MapHome is an optional block
// config additive in the details JSONB, exactly like FaqHome: flips IsZero alone,
// round-trips both fields, and Validate gates the level discriminator + title length.
func TestPageDetails_MapHome(t *testing.T) {
	if (PageDetails{MapHome: &MapHomeConfig{}}).IsZero() {
		t.Error("PageDetails with only MapHome (bare enable) should NOT be IsZero")
	}

	d := PageDetails{MapHome: &MapHomeConfig{Title: "Where we work", Level: MapHomeLevelSubDistrict}}
	v, err := d.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	var got PageDetails
	if err := got.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got.MapHome == nil || got.MapHome.Title != "Where we work" || got.MapHome.Level != MapHomeLevelSubDistrict {
		t.Errorf("map_home round-trip mismatch: got %+v", got.MapHome)
	}

	// A bare enable ({}) round-trips as an all-defaults block (omitempty fields).
	bare, err := json.Marshal(PageDetails{MapHome: &MapHomeConfig{}})
	if err != nil {
		t.Fatalf("marshal bare map: %v", err)
	}
	if !strings.Contains(string(bare), `"map_home":{}`) {
		t.Errorf("bare MapHome must serialize as {}; got %s", bare)
	}

	// Validate — empty and both known levels pass; an unknown level and over-long title reject.
	for _, okd := range []PageDetails{
		{MapHome: &MapHomeConfig{}},
		{MapHome: &MapHomeConfig{Level: MapHomeLevelDistrict}},
		{MapHome: &MapHomeConfig{Level: MapHomeLevelSubDistrict}},
	} {
		if err := okd.Validate(); err != nil {
			t.Errorf("Validate(%+v) = %v; want nil", okd, err)
		}
	}
	for _, badd := range []PageDetails{
		{MapHome: &MapHomeConfig{Level: "province"}},
		{MapHome: &MapHomeConfig{Title: strings.Repeat("x", 201)}},
	} {
		if err := badd.Validate(); err == nil {
			t.Errorf("Validate(%+v) = nil; want error", badd)
		}
	}
}

func TestPageDetails_Hero_ScanValue_RoundTrip(t *testing.T) {
	// Hero-only details (headline + cover image + search prompt + the five
	// optional hero colors, no contact).
	d := PageDetails{
		Eyebrow:       strptr("WATCH · FOLLOW · EXPLORE"),
		Headline:      strptr("Sell Your Property with Bestie"),
		CoverImageURL: strptr("https://assets.bestie/sell-cover.webp"),
		SearchPrompt:  strptr("Type a keyword above to search everything"),
		BgColor:       strptr("#0D4F4F"),
		TextColor:     strptr("#FFFFFF"),
		TitleColor:    strptr("#C8A851"),
		SubtitleColor: strptr("#E5E5E5"),
		EyebrowColor:  strptr("#C8A851"),
	}
	v, err := d.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if v == nil {
		t.Fatal("Value of hero-only details = nil; want JSON bytes")
	}

	var got PageDetails
	if err := got.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got.Eyebrow == nil || *got.Eyebrow != *d.Eyebrow {
		t.Errorf("eyebrow round-trip mismatch: got %v want %v", got.Eyebrow, *d.Eyebrow)
	}
	if got.Headline == nil || *got.Headline != *d.Headline {
		t.Errorf("headline round-trip mismatch: got %v want %v", got.Headline, *d.Headline)
	}
	if got.CoverImageURL == nil || *got.CoverImageURL != *d.CoverImageURL {
		t.Errorf("cover_image_url round-trip mismatch: got %v want %v", got.CoverImageURL, *d.CoverImageURL)
	}
	if got.SearchPrompt == nil || *got.SearchPrompt != *d.SearchPrompt {
		t.Errorf("search_prompt round-trip mismatch: got %v want %v", got.SearchPrompt, *d.SearchPrompt)
	}
	if got.BgColor == nil || *got.BgColor != *d.BgColor {
		t.Errorf("bg_color round-trip mismatch: got %v want %v", got.BgColor, *d.BgColor)
	}
	if got.TextColor == nil || *got.TextColor != *d.TextColor {
		t.Errorf("text_color round-trip mismatch: got %v want %v", got.TextColor, *d.TextColor)
	}
	if got.TitleColor == nil || *got.TitleColor != *d.TitleColor {
		t.Errorf("title_color round-trip mismatch: got %v want %v", got.TitleColor, *d.TitleColor)
	}
	if got.SubtitleColor == nil || *got.SubtitleColor != *d.SubtitleColor {
		t.Errorf("subtitle_color round-trip mismatch: got %v want %v", got.SubtitleColor, *d.SubtitleColor)
	}
	if got.EyebrowColor == nil || *got.EyebrowColor != *d.EyebrowColor {
		t.Errorf("eyebrow_color round-trip mismatch: got %v want %v", got.EyebrowColor, *d.EyebrowColor)
	}
	if got.Contact != nil {
		t.Error("hero-only details should scan Contact as nil")
	}

	// Contact + Headline together in one details blob.
	both := PageDetails{
		Contact:  &PageContact{Phone: "+66 98-348-0292", Email: "bestierealestate@gmail.com"},
		Headline: strptr("Get in touch"),
	}
	bv, err := both.Value()
	if err != nil {
		t.Fatalf("Value(both): %v", err)
	}
	if bv == nil {
		t.Fatal("Value of contact+headline details = nil; want JSON bytes")
	}
	var gotBoth PageDetails
	if err := gotBoth.Scan(bv); err != nil {
		t.Fatalf("Scan(both): %v", err)
	}
	if gotBoth.Contact == nil || gotBoth.Contact.Phone != both.Contact.Phone || gotBoth.Contact.Email != both.Contact.Email {
		t.Errorf("contact round-trip mismatch: got %+v", gotBoth.Contact)
	}
	if gotBoth.Headline == nil || *gotBoth.Headline != *both.Headline {
		t.Errorf("headline round-trip mismatch: got %v want %v", gotBoth.Headline, *both.Headline)
	}
}

// TC-details-color-validate — each non-nil hero color is a stored value rendered
// into a CSS custom property on the PUBLIC hero, so a malformed/hostile value is a
// stored-CSS-injection surface. PageDetails.Validate accepts only well-formed CSS
// colors (hex #rgb/#rrggbb/#rrggbbaa, rgb()/rgba(), hsl()/hsla()) and REJECTS
// anything carrying CSS-breaking chars (; { } < >) or otherwise unrecognized
// (url(), <script>, "red;}"). A nil color is always allowed (inherit default).
func TestPageDetails_Validate_Colors(t *testing.T) {
	valid := []string{
		"#fff", "#FFF", "#0D4F4F", "#0d4f4f", "#0D4F4FFF", "#0d4f4faa",
		"rgb(255, 0, 0)", "rgb(255,0,0)", "rgba(13, 79, 79, 0.5)", "RGB(0,0,0)",
		"hsl(120, 50%, 50%)", "hsla(120, 50%, 50%, 0.8)", "HSL(0,0%,0%)",
		"  #C8A851  ", // surrounding whitespace is trimmed, value is valid
	}
	for _, c := range valid {
		c := c
		t.Run("valid/"+c, func(t *testing.T) {
			d := PageDetails{BgColor: strptr(c)}
			if err := d.Validate(); err != nil {
				t.Errorf("Validate() = %v; want nil for valid color %q", err, c)
			}
		})
	}

	invalid := []string{
		"red;}",                       // CSS-breaking — terminates the declaration
		"url(x)",                      // not a color — an external fetch
		"url(javascript:alert(1))",    // hostile
		"<script>alert(1)</script>",   // injection
		"#0D4F4F; background: url(x)", // smuggled second declaration
		"red",                         // named colors not on the allowlist
		"expression(alert(1))",        // legacy IE CSS expression
		"#12",                         // malformed hex (wrong length)
		"#xyzxyz",                     // non-hex chars
		"}",                           // brace
		"a{color:red}",                // selector injection
		"#fff<",                       // angle bracket
	}
	for _, c := range invalid {
		c := c
		t.Run("invalid/"+c, func(t *testing.T) {
			d := PageDetails{BgColor: strptr(c)}
			if err := d.Validate(); err == nil {
				t.Errorf("Validate() = nil; want error for hostile/malformed color %q", c)
			}
		})
	}

	// A nil color (inherit the brand default) is always valid.
	if err := (PageDetails{}).Validate(); err != nil {
		t.Errorf("Validate() = %v; want nil for empty details", err)
	}

	// Every color slot is checked, not just BgColor: a hostile value in any one
	// slot is rejected.
	bad := "red;}"
	slots := []PageDetails{
		{BgColor: &bad},
		{TextColor: &bad},
		{TitleColor: &bad},
		{SubtitleColor: &bad},
		{EyebrowColor: &bad},
	}
	for i, d := range slots {
		d := d
		t.Run("each-slot-checked", func(t *testing.T) {
			if err := d.Validate(); err == nil {
				t.Errorf("slot %d: Validate() = nil; want error for hostile color in any slot", i)
			}
		})
	}
}

// TC-cmspage-validate-color — a hostile hero color reaches the upsert path through
// CmsPage.Validate (which the repository invokes before any write), so the whole
// entity is rejected (defense-in-depth for the publicly rendered CSS variable).
func TestCmsPage_Validate_RejectsHostileColor(t *testing.T) {
	hostile := CmsPage{Slug: "sell", Details: PageDetails{BgColor: strptr("red; } body { display:none")}}
	if err := hostile.Validate(); err == nil {
		t.Fatal("Validate() = nil; want error for hostile hero color")
	}
	clean := CmsPage{Slug: "sell", Details: PageDetails{BgColor: strptr("#0D4F4F"), TitleColor: strptr("rgb(200,168,81)")}}
	if err := clean.Validate(); err != nil {
		t.Errorf("Validate() = %v; want nil for well-formed hero colors", err)
	}
}

// TC-details-geo — a content page carrying a contact address must honor the
// never-skip coordinate invariant (both lat+lng, or neither). A half-set pair is
// rejected by Validate (which delegates to address.ValidateGeometry); a complete
// pair (or no coordinates) passes.
func TestCmsPage_Validate_RejectsHalfSetCoordinate(t *testing.T) {
	half := CmsPage{
		Slug: "contact",
		Details: PageDetails{Contact: &PageContact{
			Address: address.Address{City: "Koh Phangan", Country: "TH", Latitude: 9.7489}, // longitude missing
		}},
	}
	if err := half.Validate(); err == nil {
		t.Fatal("Validate() = nil; want error for half-set coordinate")
	}

	full := CmsPage{
		Slug: "contact",
		Details: PageDetails{Contact: &PageContact{
			Address: address.Address{City: "Koh Phangan", Country: "TH", Latitude: 9.7489, Longitude: 100.0310},
		}},
	}
	if err := full.Validate(); err != nil {
		t.Errorf("Validate() = %v; want nil for complete coordinate pair", err)
	}
}

// The mapper column order MUST stay in lock-step with FromRow's scan order and
// the migration. A drift here is the class of bug the integration test guards;
// assert the count + anchors so a reorder is caught early.
func TestPostgresMapper_Columns(t *testing.T) {
	m := NewPostgresMapper("bestierealestate")
	cols := m.Columns()
	// 11 base/foundation columns + draft_content + published_content (#0273) +
	// the published flag (#0281 / AE4 direct-save gate) + seo_meta (P0-A) = 15.
	if len(cols) != 15 {
		t.Fatalf("want 15 columns, got %d: %v", len(cols), cols)
	}
	if cols[0] != "c.id" {
		t.Errorf("first column = %q; want c.id", cols[0])
	}
	if m.TableName() != "bestierealestate.cms_pages" {
		t.Errorf("TableName() = %q; want bestierealestate.cms_pages", m.TableName())
	}
	// slug + version + the two #0273 section columns + the #0281 published flag
	// must all be present.
	var hasSlug, hasVersion, hasDraft, hasPublishedContent, hasPublished, hasSeoMeta bool
	for _, c := range cols {
		switch c {
		case "c.slug":
			hasSlug = true
		case "c.version":
			hasVersion = true
		case "c.draft_content":
			hasDraft = true
		case "c.published_content":
			hasPublishedContent = true
		case "c.published":
			hasPublished = true
		case "c.seo_meta":
			hasSeoMeta = true
		}
	}
	if !hasSlug || !hasVersion {
		t.Errorf("Columns() missing slug=%v / version=%v", hasSlug, hasVersion)
	}
	if !hasDraft || !hasPublishedContent {
		t.Errorf("Columns() missing draft_content=%v / published_content=%v", hasDraft, hasPublishedContent)
	}
	if !hasPublished {
		t.Error("Columns() missing the #0281 published flag column")
	}
	if !hasSeoMeta {
		t.Error("Columns() missing the P0-A seo_meta column")
	}
}

// ToRow must NOT emit deleted_at (DB default NULL) and must carry slug + version +
// the #0281 published flag (so the direct-save upsert can persist the live gate).
func TestPostgresMapper_ToRow(t *testing.T) {
	m := NewPostgresMapper("bestierealestate")
	row, err := m.ToRow(CmsPage{ID: "id-1", Slug: "about", Version: 3, Published: true})
	if err != nil {
		t.Fatalf("ToRow: %v", err)
	}
	if _, ok := row["deleted_at"]; ok {
		t.Error("ToRow must not include deleted_at")
	}
	if row["slug"] != "about" {
		t.Errorf("ToRow slug = %v; want about", row["slug"])
	}
	if row["version"] != 3 {
		t.Errorf("ToRow version = %v; want 3", row["version"])
	}
	if row["published"] != true {
		t.Errorf("ToRow published = %v; want true", row["published"])
	}
	if _, ok := row["seo_meta"]; !ok {
		t.Error("ToRow must include the P0-A seo_meta column")
	}
}

// TC-cmspage-published-roundtrip — the #0281 published flag survives the full
// mapper round-trip (ToRow → FromRow), so a save preserves the live gate. FromRow
// scans the columns in Columns() order; feed it a row matching that order with
// published=true and assert it lands on the struct field.
func TestPostgresMapper_FromRow_Published(t *testing.T) {
	m := NewPostgresMapper("bestierealestate")
	// Column order (Columns()): id, slug, title, body_markdown, details,
	// draft_content, published_content, published, seo_meta, version, created_by,
	// updated_by, created_at, updated_at, deleted_at.
	values := []any{
		"id-1",            // id
		Slug("about"),     // slug
		(*string)(nil),    // title
		(*string)(nil),    // body_markdown
		PageDetails{},     // details
		SectionList(nil),  // draft_content
		SectionList(nil),  // published_content
		true,              // published
		seo.SeoMeta{},     // seo_meta
		2,                 // version
		(*string)(nil),    // created_by
		(*string)(nil),    // updated_by
		time.Time{},       // created_at
		time.Time{},       // updated_at
		(*time.Time)(nil), // deleted_at
	}
	scan := func(dest ...any) error {
		if len(dest) != len(values) {
			t.Fatalf("FromRow scanned %d dests; want %d (Columns drift)", len(dest), len(values))
		}
		for i, d := range dest {
			switch p := d.(type) {
			case *string:
				if values[i] != nil {
					*p = values[i].(string)
				}
			case *Slug:
				*p = values[i].(Slug)
			case *PageDetails:
				*p = values[i].(PageDetails)
			case *SectionList:
				*p = values[i].(SectionList)
			case *seo.SeoMeta:
				*p = values[i].(seo.SeoMeta)
			case *bool:
				*p = values[i].(bool)
			case *int:
				*p = values[i].(int)
			case *time.Time:
				*p = values[i].(time.Time)
			case **string:
				// nullable text scanned as *string in the struct
			case **time.Time:
				// nullable timestamp
			default:
				t.Fatalf("FromRow dest %d has unexpected type %T", i, d)
			}
		}
		return nil
	}
	p, err := m.FromRow(scan)
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}
	if !p.Published {
		t.Error("FromRow did not scan published=true onto CmsPage.Published")
	}
	if p.Version != 2 {
		t.Errorf("FromRow version = %d; want 2", p.Version)
	}
}

// ErrValidation must wrap so handlers/repository can map it to 422.
func TestErrValidation_Wraps(t *testing.T) {
	wrapped := errors.New("x")
	if errors.Is(wrapped, ErrValidation) {
		t.Error("unrelated error must not match ErrValidation")
	}
}
