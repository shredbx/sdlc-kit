package rss

import (
	"fmt"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
)

// sourcesFile is the on-disk shape of the feed-sources registry YAML. It is the
// code-edited source of truth (Rule #10): one row per upstream feed. The wire
// fields are raw strings; LoadSources parses them into the dictionary-backed
// named types (ParserKind, FeedCategory, FeedLanguage) so an invalid value
// fails fast at load rather than per-request.
type sourcesFile struct {
	Sources []sourceRow `yaml:"sources"`
}

// sourceRow is one entry in the registry YAML.
type sourceRow struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Language string `yaml:"language"`
	Category string `yaml:"category"`
	Parser   string `yaml:"parser"`
	Enabled  bool   `yaml:"enabled"`
}

// LoadSources reads and validates the feed-sources registry at path, returning
// the parsed []FeedSource. It fails fast (Rule #10, usage-spec error contract:
// "config references unknown parser kind: fail fast at load, not per-request")
// on any of:
//   - unreadable / malformed YAML,
//   - empty id, name, or url,
//   - a url that is not an absolute HTTPS URL,
//   - a parser / category / language that is not a known dictionary member,
//   - a duplicate id.
//
// Every error names the offending source (by id, or by index when the id
// itself is missing) so a misconfiguration is actionable at startup.
func LoadSources(path string) ([]FeedSource, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("feed: read sources %s: %w", path, err)
	}

	var file sourcesFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("feed: parse sources %s: %w", path, err)
	}

	sources := make([]FeedSource, 0, len(file.Sources))
	seen := make(map[string]struct{}, len(file.Sources))

	for i, row := range file.Sources {
		// Identify the row for error messages: prefer the id, fall back to index.
		ref := row.ID
		if ref == "" {
			ref = fmt.Sprintf("index %d", i)
		}

		if row.ID == "" {
			return nil, fmt.Errorf("feed: source at %s: empty id", ref)
		}
		if row.Name == "" {
			return nil, fmt.Errorf("feed: source %q: empty name", ref)
		}
		if row.URL == "" {
			return nil, fmt.Errorf("feed: source %q: empty url", ref)
		}
		if err := validateHTTPSURL(row.URL); err != nil {
			return nil, fmt.Errorf("feed: source %q: %w", ref, err)
		}

		parser, ok := ParseParserKind(row.Parser)
		if !ok {
			return nil, fmt.Errorf("feed: source %q: unknown parser kind %q", ref, row.Parser)
		}
		category, ok := ParseFeedCategory(row.Category)
		if !ok {
			return nil, fmt.Errorf("feed: source %q: unknown category %q", ref, row.Category)
		}
		language, ok := ParseFeedLanguage(row.Language)
		if !ok {
			return nil, fmt.Errorf("feed: source %q: unknown language %q", ref, row.Language)
		}

		if _, dup := seen[row.ID]; dup {
			return nil, fmt.Errorf("feed: duplicate source id %q", row.ID)
		}
		seen[row.ID] = struct{}{}

		sources = append(sources, FeedSource{
			ID:       row.ID,
			Name:     row.Name,
			URL:      row.URL,
			Language: language,
			Category: category,
			Parser:   parser,
			Enabled:  row.Enabled,
		})
	}

	return sources, nil
}

// validateHTTPSURL ensures raw is an absolute HTTPS URL with a host. Plain-HTTP
// or scheme-less feed URLs are rejected so the outbound fetch is always TLS
// (usage-spec NFR: "outbound fetch over HTTPS only").
func validateHTTPSURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url %q: %w", raw, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("url %q must use https (got scheme %q)", raw, u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("url %q has no host", raw)
	}
	return nil
}
