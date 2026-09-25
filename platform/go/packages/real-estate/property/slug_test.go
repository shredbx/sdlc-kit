package property

import (
	"regexp"
	"testing"
)

// kebabASCII is the shape every derived slug must satisfy: lowercase ASCII
// letters/digits and single hyphens — never empty, never raw Unicode.
var kebabASCII = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// SC1/SC4 — a readable, stable slug for ASCII titles (exact match: ASCII input
// is deterministic and must not drift).
func TestDeriveSlug_ASCII(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"english", "Beachfront Villa in Koh Phangan", "beachfront-villa-in-koh-phangan"},
		{"punct", "Luxury 3-Bed Pool Villa (Sea View!)", "luxury-3-bed-pool-villa-sea-view"},
		{"numeric", "1234", "1234"},
	}
	for _, c := range cases {
		if got := DeriveSlug(c.in).String(); got != c.want {
			t.Errorf("%s: DeriveSlug(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// SC2 — a Thai-only title must TRANSLITERATE to a non-empty, URL-safe Latin slug,
// never be stripped to empty. The exact romanization is the library's business;
// the invariant we own is "non-empty, lowercase-ASCII-kebab".
func TestDeriveSlug_Transliterates(t *testing.T) {
	for _, in := range []string{
		"บ้านวิลล่าริมทะเลเกาะพะงัน", // Thai
		"Вилла у моря",              // Cyrillic
	} {
		got := DeriveSlug(in).String()
		if !kebabASCII.MatchString(got) {
			t.Errorf("DeriveSlug(%q) = %q, want non-empty lowercase-ASCII-kebab", in, got)
		}
	}
}

// SC3 — a title that transliterates to nothing (emoji/symbols/blank) must fall
// back to a non-empty slug so the public URL is always valid.
func TestDeriveSlug_EmptyFallsBackNonEmpty(t *testing.T) {
	for _, in := range []string{"🏖️🏠✨", "", "   ", "—/#"} {
		got := DeriveSlug(in).String()
		if got == "" || !kebabASCII.MatchString(got) {
			t.Errorf("DeriveSlug(%q) = %q, want non-empty fallback slug", in, got)
		}
	}
}

// SC4 — collisions get a numeric suffix until unique among live slugs; a free
// base is returned unchanged.
func TestEnsureUniqueSlug(t *testing.T) {
	taken := map[string]bool{"beachfront-villa": true, "beachfront-villa-2": true}
	got := EnsureUniqueSlug(Slug("beachfront-villa"), func(s Slug) bool { return taken[s.String()] })
	if got.String() != "beachfront-villa-3" {
		t.Errorf("EnsureUniqueSlug(collision) = %q, want beachfront-villa-3", got)
	}
	free := EnsureUniqueSlug(Slug("unique-one"), func(Slug) bool { return false })
	if free.String() != "unique-one" {
		t.Errorf("EnsureUniqueSlug(free) = %q, want unique-one", free)
	}
}

// SlugCandidates — the graceful ladder: title alone, then title qualified by area;
// identical/blank candidates dedupe so the ladder is always ≥1 non-empty slug.
func TestSlugCandidates(t *testing.T) {
	// Ladder order: title-only first, then title+area.
	got := SlugCandidates("Beachfront Villa", "Srithanu")
	want := []string{"beachfront-villa", "beachfront-villa-srithanu"}
	if len(got) != len(want) {
		t.Fatalf("SlugCandidates len = %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i].String() != want[i] {
			t.Errorf("SlugCandidates[%d] = %q, want %q", i, got[i].String(), want[i])
		}
	}

	// Blank area collapses the ladder to a single title-only candidate.
	if got := SlugCandidates("Beachfront Villa", ""); len(got) != 1 || got[0].String() != "beachfront-villa" {
		t.Errorf("SlugCandidates(blank area) = %v, want [beachfront-villa]", got)
	}
	if got := SlugCandidates("Beachfront Villa", "   "); len(got) != 1 {
		t.Errorf("SlugCandidates(whitespace area) = %v, want a single candidate", got)
	}

	// An area that adds nothing after transliteration dedupes to one candidate
	// (here a symbol-only area derives to the same fallback contribution).
	if got := SlugCandidates("Villa", "!!!"); len(got) < 1 {
		t.Errorf("SlugCandidates always yields ≥1 candidate, got %v", got)
	}

	// A non-empty, never-empty postcondition even for an empty title (fallback).
	if got := SlugCandidates("", ""); len(got) != 1 || got[0].String() == "" {
		t.Errorf("SlugCandidates(empty) = %v, want one non-empty fallback candidate", got)
	}
}

// ResolveSlug — first-free wins; when all candidates are taken, fall to the numeric
// floor on the FIRST candidate; the B3 reclaim case (a candidate this property
// already owns, for which a self-excluding predicate returns free) takes it un-suffixed.
func TestResolveSlug(t *testing.T) {
	cands := SlugCandidates("Beachfront Villa", "Srithanu") // [beachfront-villa, beachfront-villa-srithanu]

	// First candidate free → returned unchanged (no area qualifier, no suffix).
	if got := ResolveSlug(cands, func(Slug) bool { return false }); got.String() != "beachfront-villa" {
		t.Errorf("ResolveSlug(all free) = %q, want beachfront-villa", got)
	}

	// Title-only taken, area candidate free → the readable area-qualified form wins
	// BEFORE any numeric suffix (graceful ladder).
	areaWins := ResolveSlug(cands, func(s Slug) bool { return s.String() == "beachfront-villa" })
	if areaWins.String() != "beachfront-villa-srithanu" {
		t.Errorf("ResolveSlug(area fallback) = %q, want beachfront-villa-srithanu", areaWins)
	}

	// Both candidates taken → numeric floor on the FIRST candidate.
	taken := map[string]bool{"beachfront-villa": true, "beachfront-villa-srithanu": true}
	floor := ResolveSlug(cands, func(s Slug) bool { return taken[s.String()] })
	if floor.String() != "beachfront-villa-2" {
		t.Errorf("ResolveSlug(numeric floor) = %q, want beachfront-villa-2", floor)
	}

	// B3 reclaim: a property re-deriving its OWN prior slug. The DB-backed `taken`
	// predicate excludes self, so the property's own historical slug reports FREE
	// and is reclaimed un-suffixed (no -2, no self-redirect). Here the slug is
	// "taken" by the global namespace but the self-excluding predicate treats it as
	// free — exactly what SlugTaken(slug, selfID) returns for a self-owned slug.
	reclaim := ResolveSlug(
		SlugCandidates("Beachfront Villa", "Srithanu"),
		func(s Slug) bool {
			// owned-by-self → free; owned-by-others → taken. The first candidate is
			// the property's own prior slug, so it resolves un-suffixed.
			return s.String() != "beachfront-villa"
		},
	)
	if reclaim.String() != "beachfront-villa" {
		t.Errorf("ResolveSlug(B3 reclaim) = %q, want beachfront-villa (un-suffixed)", reclaim)
	}
}

// Slug.Validate: required + bounded (mirrors collection.Slug).
func TestSlugValidate(t *testing.T) {
	if err := Slug("ok-slug").Validate(); err != nil {
		t.Errorf("valid slug rejected: %v", err)
	}
	if err := Slug("").Validate(); err == nil {
		t.Error("empty slug accepted; want error")
	}
}

// 2607-014 — an always-taken predicate (the exact shape a DB-error short-circuit
// used to produce) must NEVER hang the suffix loop: EnsureUniqueSlug is bounded
// and hands final arbitration to the DB unique index. Regression for the 23-minute
// hot spin in TestTagFilter_RoundTrip (probe error → taken=true forever).
func TestEnsureUniqueSlug_AlwaysTakenTerminates(t *testing.T) {
	calls := 0
	got := EnsureUniqueSlug("villa", func(Slug) bool { calls++; return true })
	if got == "" {
		t.Fatal("expected a non-empty bounded-fallback candidate")
	}
	if calls > maxSlugSuffixAttempts+2 {
		t.Fatalf("taken called %d times — the suffix loop is not bounded", calls)
	}
}

// 2607-014 — ResolveSlug with an erroring probe (predicate reports FREE per the
// generateSlug contract) terminates on the FIRST candidate — the ladder never
// probes past an error.
func TestResolveSlug_ErrorReadsAsFree_TerminatesOnFirst(t *testing.T) {
	probes := 0
	free := func(Slug) bool { probes++; return false } // the error-path shape
	got := ResolveSlug([]Slug{"beachfront-villa", "beachfront-villa-srithanu"}, free)
	if got.String() != "beachfront-villa" {
		t.Errorf("ResolveSlug = %q, want first candidate", got)
	}
	if probes != 1 {
		t.Errorf("probes = %d, want 1 (terminate on first free)", probes)
	}
}
