package language

import "testing"

// TestIsValid pins the canonical gate: any assigned ISO-639-1 language is valid
// regardless of any project's offered set; junk, uppercase, non-two-letter, and
// well-formed-but-unassigned codes are not.
func TestIsValid(t *testing.T) {
	valid := []string{"en", "th", "ru", "de", "zh", "fr", "ja", "es", "ar", "hi"}
	for _, c := range valid {
		if !IsValid(c) {
			t.Errorf("IsValid(%q) = false, want true (a real ISO-639-1 language)", c)
		}
	}

	invalid := []string{
		"zz", "xx", // well-formed shape, unassigned — not real languages
		"EN", "De", // not lowercase
		"english", "eng", // not a two-letter code
		"", "e", "e1", "e-", // malformed
		"und", // the ISO-639-2 "undefined" tag
	}
	for _, c := range invalid {
		if IsValid(c) {
			t.Errorf("IsValid(%q) = true, want false (not a real lowercase ISO-639-1 language)", c)
		}
	}
}
