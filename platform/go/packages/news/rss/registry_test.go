package rss_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/rss"
)

// TestLoadSources_Valid loads the committed valid fixture and asserts every row
// maps onto the correct dictionary-backed typed fields (the whole point of the
// loader: raw YAML strings -> ParserKind/FeedCategory/FeedLanguage).
func TestLoadSources_Valid(t *testing.T) {
	sources, err := rss.LoadSources("testdata/feed-sources.valid.yml")
	if err != nil {
		t.Fatalf("LoadSources(valid) returned error: %v", err)
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}

	bp := sources[0]
	if bp.ID != "bangkokpost-property" {
		t.Errorf("source[0].ID = %q, want bangkokpost-property", bp.ID)
	}
	if bp.Name != "Bangkok Post — Property" {
		t.Errorf("source[0].Name = %q", bp.Name)
	}
	if bp.URL != "https://www.bangkokpost.com/rss/data/property.xml" {
		t.Errorf("source[0].URL = %q", bp.URL)
	}
	if bp.Language != rss.LanguageEN {
		t.Errorf("source[0].Language = %q, want en", bp.Language)
	}
	if bp.Category != rss.CategoryProperty {
		t.Errorf("source[0].Category = %q, want property", bp.Category)
	}
	if bp.Parser != rss.ParserRSS2 {
		t.Errorf("source[0].Parser = %q, want rss2", bp.Parser)
	}
	if !bp.Enabled {
		t.Error("source[0].Enabled = false, want true")
	}

	pc := sources[1]
	if pc.ID != "prachachat-business" {
		t.Errorf("source[1].ID = %q, want prachachat-business", pc.ID)
	}
	if pc.Language != rss.LanguageTH {
		t.Errorf("source[1].Language = %q, want th", pc.Language)
	}
	if pc.Category != rss.CategoryBusiness {
		t.Errorf("source[1].Category = %q, want business", pc.Category)
	}
	if pc.Parser != rss.ParserWordPress {
		t.Errorf("source[1].Parser = %q, want wordpress", pc.Parser)
	}
	if pc.Enabled {
		t.Error("source[1].Enabled = true, want false (disabled row must round-trip)")
	}
}

// TestLoadSources_InvalidParser loads the committed invalid fixture and asserts
// the error names the offending source.
func TestLoadSources_InvalidParser(t *testing.T) {
	_, err := rss.LoadSources("testdata/feed-sources.invalid.yml")
	if err == nil {
		t.Fatal("LoadSources(invalid parser) expected error, got nil")
	}
	if !strings.Contains(err.Error(), "bad-parser") {
		t.Errorf("error should name the offending source %q, got: %v", "bad-parser", err)
	}
	if !strings.Contains(err.Error(), "atom") {
		t.Errorf("error should name the bad parser value %q, got: %v", "atom", err)
	}
}

// TestLoadSources_Errors is the table-driven fail-fast matrix over inline YAML
// written to a temp file: duplicate id, missing url, bad category, bad
// language, non-https url, missing id. Each asserts an error mentioning the
// offending source / field.
func TestLoadSources_Errors(t *testing.T) {
	cases := []struct {
		name        string
		yaml        string
		wantInError string
	}{
		{
			name: "duplicate id",
			yaml: `sources:
  - id: dup
    name: A
    url: https://a.example.com/feed
    language: en
    category: property
    parser: rss2
    enabled: true
  - id: dup
    name: B
    url: https://b.example.com/feed
    language: en
    category: business
    parser: wordpress
    enabled: true
`,
			wantInError: "dup",
		},
		{
			name: "missing url",
			yaml: `sources:
  - id: no-url
    name: No URL
    url: ""
    language: en
    category: property
    parser: rss2
    enabled: true
`,
			wantInError: "no-url",
		},
		{
			name: "unknown category",
			yaml: `sources:
  - id: bad-cat
    name: Bad Category
    url: https://c.example.com/feed
    language: en
    category: sports
    parser: rss2
    enabled: true
`,
			wantInError: "bad-cat",
		},
		{
			name: "unknown language",
			yaml: `sources:
  - id: bad-lang
    name: Bad Language
    url: https://d.example.com/feed
    language: fr
    category: property
    parser: rss2
    enabled: true
`,
			wantInError: "bad-lang",
		},
		{
			name: "non-https url",
			yaml: `sources:
  - id: plain-http
    name: Plain HTTP
    url: http://e.example.com/feed
    language: en
    category: property
    parser: rss2
    enabled: true
`,
			wantInError: "plain-http",
		},
		{
			name: "missing id",
			yaml: `sources:
  - id: ""
    name: No ID
    url: https://f.example.com/feed
    language: en
    category: property
    parser: rss2
    enabled: true
`,
			wantInError: "empty id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "feed-sources.yml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o600); err != nil {
				t.Fatalf("write temp fixture: %v", err)
			}
			_, err := rss.LoadSources(path)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantInError) {
				t.Errorf("error %q should contain %q", err.Error(), tc.wantInError)
			}
		})
	}
}

// TestLoadSources_MissingFile asserts an unreadable path is a clear error,
// naming the path.
func TestLoadSources_MissingFile(t *testing.T) {
	_, err := rss.LoadSources("testdata/does-not-exist.yml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "does-not-exist.yml") {
		t.Errorf("error should name the path, got: %v", err)
	}
}
