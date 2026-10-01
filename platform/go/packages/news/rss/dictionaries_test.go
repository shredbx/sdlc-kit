package rss_test

import (
	"testing"

	"github.com/shredbx/sbx-core/pkg/rss"
)

func TestFeedCategory_Codes(t *testing.T) {
	cases := map[rss.FeedCategory]string{
		rss.CategoryProperty: "property",
		rss.CategoryBusiness: "business",
		rss.CategoryGeneral:  "general",
	}
	for c, want := range cases {
		if string(c) != want {
			t.Errorf("category code: got %q, want %q", string(c), want)
		}
		if !c.Valid() {
			t.Errorf("category %q should be valid", c)
		}
	}
}

func TestParseFeedCategory(t *testing.T) {
	cases := []struct {
		in     string
		want   rss.FeedCategory
		wantOK bool
	}{
		{"property", rss.CategoryProperty, true},
		{"business", rss.CategoryBusiness, true},
		{"general", rss.CategoryGeneral, true},
		{"sports", "", false},
		{"", "", false},
		{"Property", "", false}, // case-sensitive dictionary code
	}
	for _, tc := range cases {
		got, ok := rss.ParseFeedCategory(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("ParseFeedCategory(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestFeedLanguage_Codes(t *testing.T) {
	if string(rss.LanguageEN) != "en" {
		t.Errorf("LanguageEN: got %q, want en", rss.LanguageEN)
	}
	if string(rss.LanguageTH) != "th" {
		t.Errorf("LanguageTH: got %q, want th", rss.LanguageTH)
	}
}

func TestParseFeedLanguage(t *testing.T) {
	cases := []struct {
		in     string
		want   rss.FeedLanguage
		wantOK bool
	}{
		{"en", rss.LanguageEN, true},
		{"th", rss.LanguageTH, true},
		{"fr", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := rss.ParseFeedLanguage(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("ParseFeedLanguage(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestParserKind_Codes_And_Parse(t *testing.T) {
	if string(rss.ParserRSS2) != "rss2" {
		t.Errorf("ParserRSS2: got %q, want rss2", rss.ParserRSS2)
	}
	if string(rss.ParserWordPress) != "wordpress" {
		t.Errorf("ParserWordPress: got %q, want wordpress", rss.ParserWordPress)
	}
	cases := []struct {
		in     string
		want   rss.ParserKind
		wantOK bool
	}{
		{"rss2", rss.ParserRSS2, true},
		{"wordpress", rss.ParserWordPress, true},
		{"atom", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := rss.ParseParserKind(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("ParseParserKind(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestParserFor(t *testing.T) {
	if p, err := rss.ParserFor(rss.ParserRSS2); err != nil || p == nil {
		t.Errorf("ParserFor(rss2) = (%v, %v), want a parser and nil error", p, err)
	}
	if p, err := rss.ParserFor(rss.ParserWordPress); err != nil || p == nil {
		t.Errorf("ParserFor(wordpress) = (%v, %v), want a parser and nil error", p, err)
	}
	if _, err := rss.ParserFor(rss.ParserKind("nope")); err == nil {
		t.Error("ParserFor(unknown) should return an error")
	}
}
