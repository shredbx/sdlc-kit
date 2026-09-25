package property_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/property"
)

func TestTagSlug(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"simple two words", "Sea View", "sea-view"},
		{"collapse + trim whitespace", "  Beach   House  ", "beach-house"},
		{"already kebab", "already-kebab", "already-kebab"},
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"strip surrounding hyphens", "- foo -", "foo"},
		{"case folds to same slug as spaced variant", "SEA   view", "sea-view"},
		{"thai preserved (no ascii stripping)", "วิว ทะเล", "วิว-ทะเล"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := property.TagSlug(c.in); got != c.want {
				t.Errorf("TagSlug(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestTagSlug_CapsAt50Runes(t *testing.T) {
	long := strings.Repeat("a", 80)
	got := property.TagSlug(long)
	if len([]rune(got)) > 50 {
		t.Errorf("TagSlug should cap at 50 runes, got %d", len([]rune(got)))
	}
}

func TestNormalizeTags_DedupesBySlugKeepsFirstLabel(t *testing.T) {
	in := []string{"Sea View", "sea view", "  ", "Pool", "POOL", ""}
	got := property.NormalizeTags(in)
	want := []string{"Sea View", "Pool"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("NormalizeTags(%v) = %v, want %v (dedupe by slug, first-seen label, order preserved)", in, got, want)
	}
}

func TestNormalizeTags_EmptyReturnsNil(t *testing.T) {
	if got := property.NormalizeTags(nil); got != nil {
		t.Errorf("NormalizeTags(nil) = %v, want nil", got)
	}
	if got := property.NormalizeTags([]string{"", "   ", "\t"}); got != nil {
		t.Errorf("NormalizeTags(all-empty) = %v, want nil", got)
	}
}

// TestNormalizeTags_SanitizesCharset pins the charset rule (task 2606-001): a stored
// tag LABEL may contain only Unicode letters (Thai/CJK included), digits, ASCII
// whitespace and hyphens. Every other rune — punctuation, symbols, emoji — is stripped
// on normalize so a stored label can only ever produce a clean [letter|digit|-] slug.
// This sanitizes the LABEL, NOT TagSlug — the slug formula (and its SQL twin) is
// unchanged, so there is no migration: existing symbol-bearing labels clean forward on
// their next save.
func TestNormalizeTags_SanitizesCharset(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // stored label after sanitize + trim ("" means dropped → nil)
	}{
		{"strip trailing symbol", "Sea View!", "Sea View"},
		{"strip percent, keep digits + space", "100% Pool", "100 Pool"},
		{"strip emoji then trim", "Pool 🏊", "Pool"},
		{"keep hyphen and digits", "Sea-View 2", "Sea-View 2"},
		{"thai preserved, symbol stripped", "วิว ทะเล!", "วิว ทะเล"},
		{"accented letter preserved", "Café", "Café"},
		{"all-symbol drops to nil", "!@#$%", ""},
		// Decimal-digit parity with the client \p{Nd}: a Thai decimal digit (Nd)
		// survives, a non-decimal numeral (½ is No) is stripped — both engines agree.
		{"thai decimal digit kept", "ห้อง ๒", "ห้อง ๒"},
		{"non-decimal numeral stripped", "Villa ½", "Villa"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := property.NormalizeTags([]string{c.in})
			if c.want == "" {
				if got != nil {
					t.Errorf("NormalizeTags([%q]) = %v, want nil (all-symbol sanitizes to empty)", c.in, got)
				}
				return
			}
			if len(got) != 1 || got[0] != c.want {
				t.Errorf("NormalizeTags([%q]) = %v, want [%q]", c.in, got, c.want)
			}
		})
	}
}

func TestNormalizeTags_CapsAtMaxTags(t *testing.T) {
	in := make([]string, property.MaxTags+10)
	for i := range in {
		in[i] = fmt.Sprintf("tag-%d", i)
	}
	got := property.NormalizeTags(in)
	if len(got) != property.MaxTags {
		t.Errorf("NormalizeTags should cap at MaxTags=%d, got %d", property.MaxTags, len(got))
	}
}

func TestNormalizeTags_TrimsLabels(t *testing.T) {
	got := property.NormalizeTags([]string{"  Quiet Area  "})
	if len(got) != 1 || got[0] != "Quiet Area" {
		t.Errorf("NormalizeTags should trim labels, got %v", got)
	}
}

func TestNormalizeTags_CapsLabelAt100Runes(t *testing.T) {
	long := strings.Repeat("น", 150) // 150 Thai runes
	got := property.NormalizeTags([]string{long})
	if len(got) != 1 {
		t.Fatalf("expected 1 tag, got %v", got)
	}
	if n := len([]rune(got[0])); n != property.MaxTagLabelLen {
		t.Errorf("NormalizeTags should cap label at MaxTagLabelLen=%d runes, got %d", property.MaxTagLabelLen, n)
	}
}

// TestTagSlug_CrossEngineGolden pins the slug outputs that the SQL seed in
// migration 20260531000002_property_tags.up.sql MUST also produce. These are the
// adversarial cases (Unicode whitespace, vertical tab, truncation boundary) where Go
// and SQL previously diverged — the divergence broke the #0279 create-on-add dedup
// invariant. If you change TagSlug, change the migration's slug expression the SAME
// way and update these expectations. Both engines: lower -> collapse ASCII whitespace
// (` \t\n\v\f\r`) to '-' -> LEFT(50 runes) -> trim '-'.
func TestTagSlug_CrossEngineGolden(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // identical in Go (TagSlug) and the SQL seed formula
	}{
		// Vertical tab: Go RE2 `\s` excluded it, POSIX [[:space:]] included it → the
		// explicit class now matches it in BOTH.
		{"vertical tab is a separator", "foo\vbar", "foo-bar"},
		// Truncation boundary: char 50 is a hyphen → both LEFT(50) then trim '-'.
		{"truncation re-trims trailing hyphen", strings.Repeat("a", 49) + "  bb", strings.Repeat("a", 49)},
		// Non-ASCII (NBSP) whitespace is NOT a separator in either engine → preserved
		// verbatim, identically, so the slug still matches (consistency over prettiness).
		{"nbsp (U+00A0) is not an ASCII separator", "\u00a0sea view", "\u00a0sea-view"},
		// Ordinary cases.
		{"multi word", "Foreign Ownership", "foreign-ownership"},
		{"already slug", "due-diligence", "due-diligence"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := property.TagSlug(c.in); got != c.want {
				t.Errorf("TagSlug(%q) = %q, want %q (MUST match the SQL seed formula)", c.in, got, c.want)
			}
		})
	}
}
