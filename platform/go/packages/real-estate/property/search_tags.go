// Tag-search param normalization (Decision #0280 M-B Unit 2).
//
// The `?tags=` facet on the property List endpoints matches on SLUG identity, not
// the stored label: a request of "Sea View" must match a property tagged "Sea View"
// via their shared slug "sea-view". This file owns the ONE read-side normalizer
// every listing surface (the BR public List handler, the admin ManageList, the
// shared filter builder) calls, so the slug + dedup + cap rules can never drift
// between surfaces. Moved here from the BR app's handler package (2607-004 — the
// original file said "logically belongs next to TagSlug in shared core"; it now is).
package property

// MaxSearchTags caps how many distinct tag slugs a single search request may carry.
// A `?tags=` filter is an AND/narrowing `@>` containment — beyond a handful of slugs
// the result is almost always empty, so an unbounded list only burns parameter space
// and log noise. 20 is generous headroom over any realistic facet UI while keeping
// the bound parameter small. Slugs beyond the cap are TRUNCATED (the first
// MaxSearchTags distinct slugs in input order are kept), not rejected — a too-long
// URL still returns a sensible (narrower-than-asked) result rather than a 400.
const MaxSearchTags = 20

// NormalizeSearchTags turns the comma-split values of a `?tags=` param into the
// canonical slug identities used for tag search. Each raw value is mapped through
// TagSlug — the SAME formula the SQL expression index
// (bestierealestate.tag_slug_array) uses — so the query expression and the index
// expression agree. Empty slugs are dropped, duplicates removed (preserving
// first-seen input order), and the result capped at MaxSearchTags. Returns nil when
// nothing usable remains so callers can treat "no tag filter" uniformly.
func NormalizeSearchTags(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		slug := TagSlug(r)
		if slug == "" {
			continue
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		out = append(out, slug)
		if len(out) >= MaxSearchTags {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
