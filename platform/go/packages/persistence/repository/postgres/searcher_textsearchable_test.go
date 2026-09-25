package postgres

import (
	"strings"
	"testing"
)

// legacyMapper does NOT implement TextSearchable — it must drive the historical
// property-shaped text-search SQL (alias p, title + loc.* fuzzy fields,
// p.search_vector) so existing property search behavior is unchanged.
type legacyMapper struct{}

func (legacyMapper) TableName() string           { return "s.things" }
func (legacyMapper) SelectFrom() string          { return "s.things p" }
func (legacyMapper) Columns() []string           { return []string{"p.id"} }
func (legacyMapper) FieldColumn(f string) string { return f }
func (legacyMapper) ToRow(struct{}) (map[string]any, error) {
	return map[string]any{}, nil
}
func (legacyMapper) FromRow(scan func(dest ...any) error) (struct{}, error) {
	return struct{}{}, nil
}

// customMapper implements TextSearchable — it must drive the text-search tiers
// from its own search-vector column + fuzzy text fields.
type customMapper struct{ legacyMapper }

func (customMapper) SearchVectorColumn() string { return "p.search_vector" }
func (customMapper) FuzzyTextFields() []string  { return []string{"p.title", "p.category"} }

// resolveTextSearch is the seam: given a mapper, it returns the search-vector
// column + fuzzy text fields the tiers should use. Mappers implementing
// TextSearchable drive it; all others fall back to the legacy property shape.
func TestResolveTextSearch_LegacyDefaults(t *testing.T) {
	sv, fields := resolveTextSearch[struct{}](legacyMapper{})
	if sv != "p.search_vector" {
		t.Errorf("legacy search vector = %q, want p.search_vector", sv)
	}
	want := []string{"p.title", "loc.sub_district", "loc.city", "loc.province"}
	if strings.Join(fields, ",") != strings.Join(want, ",") {
		t.Errorf("legacy fuzzy fields = %v, want %v", fields, want)
	}
}

func TestResolveTextSearch_CustomMapper(t *testing.T) {
	sv, fields := resolveTextSearch[struct{}](customMapper{})
	if sv != "p.search_vector" {
		t.Errorf("custom search vector = %q, want p.search_vector", sv)
	}
	want := []string{"p.title", "p.category"}
	if strings.Join(fields, ",") != strings.Join(want, ",") {
		t.Errorf("custom fuzzy fields = %v, want %v", fields, want)
	}
}

// buildFuzzyScoreExpr produces the GREATEST(word_similarity(...)) tier-2 SQL from
// the fuzzy fields. Single field (the cms case) must not emit a GREATEST wrapper
// with one arg that breaks — it must be a valid expression.
func TestBuildFuzzyScoreExpr_SingleField(t *testing.T) {
	got := buildFuzzyScoreExpr([]string{"p.title"}, 5)
	want := `word_similarity($5, COALESCE(p.title, ''))`
	if got != want {
		t.Errorf("single-field fuzzy = %q, want %q", got, want)
	}
}

func TestBuildFuzzyScoreExpr_MultiField(t *testing.T) {
	got := buildFuzzyScoreExpr([]string{"p.title", "loc.city"}, 3)
	if !strings.HasPrefix(got, "GREATEST(") {
		t.Errorf("multi-field fuzzy must wrap in GREATEST: %q", got)
	}
	if !strings.Contains(got, "word_similarity($3, COALESCE(p.title, ''))") ||
		!strings.Contains(got, "word_similarity($3, COALESCE(loc.city, ''))") {
		t.Errorf("multi-field fuzzy missing a field expr: %q", got)
	}
}

// buildIlikeCond produces the tier-3 ILIKE OR-chain from the fuzzy fields.
func TestBuildIlikeCond_SingleField(t *testing.T) {
	got := buildIlikeCond([]string{"p.title"}, 7)
	if !strings.Contains(got, `COALESCE(p.title, '') ILIKE '%' || $7 || '%'`) {
		t.Errorf("single-field ilike missing clause: %q", got)
	}
	if !strings.HasPrefix(got, "(") || !strings.HasSuffix(got, ")") {
		t.Errorf("single-field ilike must be parenthesized: %q", got)
	}
}

func TestBuildIlikeCond_MultiField(t *testing.T) {
	got := buildIlikeCond([]string{"p.title", "loc.city"}, 2)
	if !strings.Contains(got, `COALESCE(p.title, '') ILIKE '%' || $2 || '%'`) ||
		!strings.Contains(got, `COALESCE(loc.city, '') ILIKE '%' || $2 || '%'`) ||
		!strings.Contains(got, "OR ") {
		t.Errorf("multi-field ilike missing a clause / OR: %q", got)
	}
}
