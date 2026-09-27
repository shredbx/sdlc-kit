package transaction

import "fmt"

// typeFacets is the code mapper: TransactionType -> facet SET. This is engine
// CODE (not data) — a new variant (Rent for Bestays) adds one entry here and a
// new Facet const, with zero change to the transition/close/validate logic.
//
// Slices are returned by-copy (see FacetsFor) so callers can never mutate the
// canonical mapping.
var typeFacets = map[TransactionType][]Facet{
	TypeSale:  {FacetSale},
	TypeLease: {FacetLease},
	// TypeRent: {FacetRent}, // deferred
}

// FacetsFor returns the facet SET a deal type composes, in declaration order.
// An uncompiled code (e.g. "rent" before the Bestays consumer lands) returns
// nil — it has no facet behaviour, so it cannot be enabled or validated.
//
//	FacetsFor(TypeSale)              => []Facet{FacetSale}
//	FacetsFor(TransactionType("rent")) => nil
func FacetsFor(t TransactionType) []Facet {
	src, ok := typeFacets[t]
	if !ok {
		return nil
	}
	// Defensive copy — never expose the canonical slice for mutation.
	out := make([]Facet, len(src))
	copy(out, src)
	return out
}

// EnabledTypes is a consumer's enabled deal-type set (project-level config, in
// code — NOT a dictionary table). Constructed via EnabledSet; query via
// IsEnabled. The zero value enables nothing.
type EnabledTypes struct {
	enabled map[TransactionType]struct{}
}

// EnabledSet builds the consumer enablement from the type codes it turns on.
// It fails fast (panics) if any code has no compiled facet set — this call is
// version-controlled with the app and runs at startup, so an unknown code is a
// programming error to surface immediately (usage-spec 2d failure twin), not a
// runtime condition to handle.
//
//	cfg := EnabledSet(TypeSale, TypeLease)
//	cfg.IsEnabled(TypeLease) // => true
func EnabledSet(types ...TransactionType) EnabledTypes {
	set := EnabledTypes{enabled: make(map[TransactionType]struct{}, len(types))}
	for _, t := range types {
		if FacetsFor(t) == nil {
			panic(fmt.Sprintf("transaction type %q has no facet mapping", t))
		}
		set.enabled[t] = struct{}{}
	}
	return set
}

// IsEnabled reports whether type t is enabled for this consumer. A compiled but
// not-enabled code, and any uncompiled code, both report false.
func (e EnabledTypes) IsEnabled(t TransactionType) bool {
	_, ok := e.enabled[t]
	return ok
}
