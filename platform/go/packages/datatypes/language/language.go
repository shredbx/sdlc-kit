// Package language is the shared, canonical reference for the world's languages.
//
// It answers one question for every project's content-localization write paths:
// "is this a real language?" — backed by the CLDR / ISO-639 registry in
// golang.org/x/text (already a workspace dependency), so there is NO hand-maintained
// table and no shipped list to drift. A project's own "languages" dictionary is the
// admin's OFFERED set (which languages to render for filling); it is NOT the gate for
// what may be stored. The gate is canonical validity — this package.
//
// Names / endonyms are a DISPLAY concern resolved at the edge (Intl.DisplayNames in
// the browser, or x/text/language/display server-side); this package deliberately
// owns only validity, the one thing a write path must enforce.
package language

import (
	"strings"

	"golang.org/x/text/language"
)

// IsValid reports whether code is a real, lowercase ISO-639-1 language subtag.
//
//   - shape gate (no lookup): exactly two lowercase ASCII letters — rejects "EN",
//     "english", "", "e1", "eng";
//   - registry gate: the code must parse to an ASSIGNED ISO-639 base language via
//     x/text — rejects well-formed-but-unassigned pairs like "zz" / "xx", and the
//     undefined language "und".
//
// It accepts any assigned two-letter language (en, th, ru, de, zh, ja, …) regardless
// of whether a given project offers it.
func IsValid(code string) bool {
	if !isLowerISO6391(code) {
		return false
	}
	if code == "und" { // the ISO-639-2 "undefined" tag — never a content language
		return false
	}
	base, err := language.ParseBase(code)
	if err != nil {
		return false
	}
	// ParseBase canonicalizes to the assigned form; require a round-trip to the same
	// lowercase two-letter code so a remapped/aliased or undefined input never slips
	// through as "valid".
	return strings.EqualFold(base.String(), code)
}

// isLowerISO6391 reports whether s is exactly two lowercase ASCII letters — the
// ISO-639-1 shape. Cheapest gate; runs before any registry lookup.
func isLowerISO6391(s string) bool {
	if len(s) != 2 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if c := s[i]; c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}
