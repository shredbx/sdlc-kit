package property

import (
	"regexp"
	"strings"
	"unicode"
)

// Tag limits. MaxTags mirrors the Property.Tags `max=50` validate tag; MaxTagLabelLen
// mirrors the per-label `max=100`. The dict code column is VARCHAR(50), so a slug is
// capped at 50 runes.
const (
	MaxTags        = 50
	MaxTagLabelLen = 100
	maxTagSlugLen  = 50
)

// asciiWhitespace is the EXACT whitespace set both TagSlug and the SQL seed treat as
// separators — ASCII only ` \t\n\v\f\r`. It deliberately does NOT include Unicode
// whitespace (NBSP, U+3000, …): pure-SQL Unicode-whitespace handling is unreliable
// (POSIX `[[:space:]]` is locale-dependent), so to keep the Go and SQL slug formulas
// PROVABLY identical (Decision #0279's create-on-add dedup invariant) BOTH sides
// treat only ASCII whitespace as a separator and preserve any exotic whitespace
// verbatim. Used for both trimming and the slug regex.
const asciiWhitespace = " \t\n\v\f\r"

// tagSlugWhitespace collapses runs of ASCII whitespace to a single hyphen. Mirrors
// the SQL seed's `regexp_replace(LOWER(t), E'[ \t\n\v\f\r]+', '-', 'g')`.
var tagSlugWhitespace = regexp.MustCompile(`[ \t\n\v\f\r]+`)

// TagSlug derives a tag's stable identity from its label: lowercase, collapse ASCII
// whitespace runs to single hyphens, cap at 50 runes, strip leading/trailing hyphens.
// This is BOTH the dedup key for NormalizeTags AND the `tags` dictionary code used by
// create-on-add. The SQL seed in migration 20260531000002_property_tags.up.sql uses
// the IDENTICAL formula — same ASCII-whitespace class, same operation ORDER
// (lower → replace → LEFT(50) → trim '-') — so a Go-created tag can never produce a
// code that collides-yet-differs with a seeded one (the invariant the reviewer must
// keep true; see tags_test.go's cross-engine golden cases). Non-ASCII letters (Thai,
// CJK) are preserved — only ASCII whitespace is rewritten.
func TagSlug(label string) string {
	s := tagSlugWhitespace.ReplaceAllString(strings.ToLower(label), "-")
	if r := []rune(s); len(r) > maxTagSlugLen {
		s = string(r[:maxTagSlugLen])
	}
	return strings.Trim(s, "-")
}

// allowedTagRune reports whether r may appear in a stored tag LABEL: a Unicode letter,
// COMBINING MARK or DECIMAL digit (so Thai/CJK survive — Thai vowel/tone marks are
// Mn/Mc, not letters, so IsMark is required to keep e.g. "วิว" intact — matching
// TagSlug's non-ASCII-preserving contract), an ASCII whitespace separator
// (` \t\n\v\f\r` — collapsed to '-' by TagSlug), or a hyphen. Everything else —
// punctuation, symbols, emoji — is not allowed. unicode.IsDigit is the Nd
// (decimal-number) category EXACTLY, mirroring the client's \p{Nd} so the two charsets
// stay byte-identical (Roman numerals / fractions / other Numbers are stripped on both
// sides) (task 2606-001).
func allowedTagRune(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsDigit(r) {
		return true
	}
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', '-':
		return true
	}
	return false
}

// sanitizeTagLabel strips every rune outside the allowed tag charset (allowedTagRune),
// preserving letters (incl. Thai), digits, ASCII whitespace and hyphens. It cleans the
// LABEL, deliberately NOT TagSlug: by removing symbols before the label is stored, the
// existing TagSlug formula keeps producing a clean [letter|digit|-] code with NO change
// to the SQL twin (so no migration). Existing symbol-bearing labels are cleaned forward
// on their next NormalizeTags pass (every write re-runs it).
func sanitizeTagLabel(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if allowedTagRune(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NormalizeTags sanitizes each label to the allowed charset (allowedTagRune — letters
// incl. Thai, digits, ASCII whitespace, hyphen), trims it (ASCII whitespace, matching
// the slug contract), drops empties, caps each label at MaxTagLabelLen runes, and
// de-duplicates by slug identity (case- and whitespace-insensitive), preserving the
// FIRST-seen label form and input order. Per Decision #0279 the STORED value is the
// trimmed LABEL — the slug is only the dedup key, never persisted on the entity. The
// per-label cap is the defense-in-depth enforcement of the struct's `max=100` validate
// tag for non-API callers (the API boundary already rejects with 422). Caps total at
// MaxTags. Returns nil for an all-empty input so callers can treat "no tags" uniformly.
func NormalizeTags(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		label := strings.Trim(sanitizeTagLabel(r), asciiWhitespace)
		if label == "" {
			continue
		}
		if rs := []rune(label); len(rs) > MaxTagLabelLen {
			label = string(rs[:MaxTagLabelLen])
		}
		slug := TagSlug(label)
		if slug == "" {
			continue
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		out = append(out, label)
		if len(out) >= MaxTags {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
