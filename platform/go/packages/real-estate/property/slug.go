package property

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gosimple/slug"
)

// =============================================================================
// SLUG — the property public URL key (a named type, never a raw string)
// =============================================================================

// MaxSlugLen is the entity.yml `slug` constraint maxLength = 200.
const MaxSlugLen = 200

// slugFallback is the non-empty slug used when a title transliterates to nothing
// (a pure emoji/symbol/blank title). It guarantees the public URL is never empty;
// EnsureUniqueSlug then disambiguates collisions ("listing", "listing-2", …) and
// the manager can always set a cleaner slug by hand.
const slugFallback = "listing"

// Slug is the unique, URL-safe identifier for a property — the /properties/{slug}
// public route segment, unique among LIVE (non-deleted) properties. It is a named
// type (never a raw string in signatures) so the value's invariant travels with
// it (standing rule: normalize discriminators). Mirrors collection.Slug.
type Slug string

// String renders the slug for SQL params and URLs.
func (s Slug) String() string { return string(s) }

// Validate enforces the entity.yml `slug` constraints: required (non-empty after
// trim) and bounded at MaxSlugLen runes. Charset is not constrained here —
// DeriveSlug already produces a clean transliterated kebab form, and a manager
// override is normalised by the handler before it reaches storage.
func (s Slug) Validate() error {
	v := strings.TrimSpace(string(s))
	if v == "" {
		return errors.New("slug is required")
	}
	if len([]rune(v)) > MaxSlugLen {
		return errors.New("slug must be at most 200 characters")
	}
	return nil
}

// DeriveSlug builds a clean, URL-safe slug from a property title. Unlike
// collection.DeriveSlug — whose `[^a-z0-9]+` strip empties a Thai-only title — this
// TRANSLITERATES via gosimple/slug, so Thai/Cyrillic/etc. titles become readable
// Latin (e.g. "บ้านวิลล่า…" → "baanwillaa…") rather than vanishing. A title that
// still transliterates to nothing (pure emoji/symbols/blank) falls back to a
// non-empty slug so the public URL is always valid. The result is lowercase
// kebab, capped at MaxSlugLen runes with any dangling hyphen trimmed.
func DeriveSlug(title string) Slug {
	s := slug.Make(title)
	if s == "" {
		s = slugFallback
	}
	if r := []rune(s); len(r) > MaxSlugLen {
		s = strings.Trim(string(r[:MaxSlugLen]), "-")
	}
	return Slug(s)
}

// maxSlugSuffixAttempts bounds EnsureUniqueSlug's numeric-suffix loop. The DB
// unique index is the FINAL arbiter of slug uniqueness: if a thousand suffixes
// all read "taken", something upstream is wrong (e.g. the probe is erroring and
// every candidate reads taken) — a loud unique-violation on insert beats an
// unbounded hot loop. Regression: a probe-error short-circuit that returned
// "taken" spun this loop for 23 minutes inside a property create (2607-014).
const maxSlugSuffixAttempts = 1000

// EnsureUniqueSlug returns base when it is free, otherwise appends -2, -3, … until
// the taken predicate reports a free candidate. taken(s) returns true when s is
// already used by another LIVE property; the caller supplies that DB existence
// check, keeping this function pure and unit-testable. Mirrors the dedup-suffix
// rule the migration backfill applies to existing rows. The loop is BOUNDED
// (maxSlugSuffixAttempts): past the cap the last candidate is returned and the
// DB unique constraint arbitrates — this function must never be able to hang.
func EnsureUniqueSlug(base Slug, taken func(Slug) bool) Slug {
	if !taken(base) {
		return base
	}
	cand := base
	for n := 2; n < 2+maxSlugSuffixAttempts; n++ {
		cand = Slug(base.String() + "-" + strconv.Itoa(n))
		if !taken(cand) {
			return cand
		}
	}
	return cand
}

// SlugCandidates is the graceful candidate ladder (FriendlyId-style): the readable
// slugs to TRY in order before falling back to a numeric suffix. The first is the
// title alone; the second qualifies the title with the property's area (sub-district
// or city) so a title that collides with a live/historical slug can still resolve to
// a readable, location-disambiguated form ("beachfront-villa-srithanu") rather than
// jumping straight to a numeric "-2". Both are transliterated via DeriveSlug (never
// empty). Identical candidates are deduped (a blank area, or an area that adds nothing
// after transliteration, collapses the ladder to one) and empties dropped — so the
// result is always ≥1 non-empty Slug. The numeric floor is ResolveSlug's job, not
// this ladder's.
func SlugCandidates(title, area string) []Slug {
	out := make([]Slug, 0, 2)
	seen := make(map[Slug]struct{}, 2)
	add := func(s Slug) {
		if s == "" {
			return
		}
		if _, dup := seen[s]; dup {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(DeriveSlug(title))
	if strings.TrimSpace(area) != "" {
		add(DeriveSlug(title + " " + area))
	}
	if len(out) == 0 {
		// Defensive: DeriveSlug never returns empty (it falls back to slugFallback),
		// so this is unreachable in practice — but it guarantees the postcondition
		// "always ≥1 candidate" no matter what DeriveSlug does.
		out = append(out, Slug(slugFallback))
	}
	return out
}

// ResolveSlug picks the final slug for a property from the candidate ladder. It
// returns the FIRST candidate the taken predicate reports free (the readable,
// un-suffixed form — including the B3 reclaim case where a candidate is a slug this
// property already owns, for which a self-excluding taken predicate returns false).
// When EVERY candidate is taken it falls back to the numeric floor: EnsureUniqueSlug
// on the FIRST candidate (title-derived), appending -2, -3, … until free. taken(s) is
// the caller's DB existence check (s in property_slug_history of another property OR
// s in another LIVE property's slug); keeping it injected keeps ResolveSlug pure and
// unit-testable. cands is assumed non-empty (SlugCandidates guarantees that); an empty
// slice is handled defensively via the fallback slug.
func ResolveSlug(cands []Slug, taken func(Slug) bool) Slug {
	if len(cands) == 0 {
		cands = []Slug{Slug(slugFallback)}
	}
	for _, c := range cands {
		if !taken(c) {
			return c
		}
	}
	return EnsureUniqueSlug(cands[0], taken)
}
