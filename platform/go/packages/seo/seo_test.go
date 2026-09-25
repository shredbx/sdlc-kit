package seo

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestSeoMeta_ScanValue_JSONB_RoundTrip proves the JSONB column contract:
// a populated SeoMeta marshals via Value and re-hydrates identically via Scan;
// Scan(nil) yields the zero value; and an entirely-empty SeoMeta writes SQL NULL
// (Value returns nil) so unset entities never persist a literal "{}".
func TestSeoMeta_ScanValue_JSONB_RoundTrip(t *testing.T) {
	original := SeoMeta{
		MetaTitle:       "Beachfront Villa in Koh Phangan",
		MetaDescription: "A 3-bedroom pool villa steps from the sea.",
		MetaKeywords:    []string{"villa", "koh-phangan", "beachfront"},
		OgTitle:         "Beachfront Villa",
		OgDescription:   "3BR pool villa",
		OgImage:         "https://cdn.example.com/villa.jpg",
		CanonicalURL:    "https://example.com/properties/123",
		Noindex:         true,
	}

	v, err := original.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	data, ok := v.([]byte)
	if !ok {
		t.Fatalf("Value should return []byte for a non-empty SeoMeta, got %T", v)
	}

	var rescanned SeoMeta
	if err := rescanned.Scan(data); err != nil {
		t.Fatalf("Scan([]byte): %v", err)
	}
	if !reflect.DeepEqual(rescanned, original) {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", rescanned, original)
	}

	// Scan from string (defense — some drivers hand JSONB back as string).
	var fromString SeoMeta
	if err := fromString.Scan(string(data)); err != nil {
		t.Fatalf("Scan(string): %v", err)
	}
	if !reflect.DeepEqual(fromString, original) {
		t.Errorf("Scan(string) mismatch:\n got  %+v\n want %+v", fromString, original)
	}

	// Scan(nil) → zero value (a NULL column).
	var fromNil SeoMeta
	if err := fromNil.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if !fromNil.IsZero() {
		t.Errorf("Scan(nil) should produce the zero SeoMeta, got %+v", fromNil)
	}

	// Empty SeoMeta → Value() must return nil (stores SQL NULL, never "{}").
	emptyVal, err := (SeoMeta{}).Value()
	if err != nil {
		t.Fatalf("empty Value: %v", err)
	}
	if emptyVal != nil {
		t.Errorf("empty SeoMeta.Value() should be nil (SQL NULL), got %v", emptyVal)
	}

	// Noindex alone makes a SeoMeta non-zero: IsZero must be false and Value must
	// marshal (not NULL) so the noindex directive persists. A round-trip through
	// Scan re-hydrates the flag.
	noindexOnly := SeoMeta{Noindex: true}
	if noindexOnly.IsZero() {
		t.Error("SeoMeta{Noindex:true} should NOT be zero (the flag must persist)")
	}
	niVal, err := noindexOnly.Value()
	if err != nil {
		t.Fatalf("noindex-only Value: %v", err)
	}
	niBytes, ok := niVal.([]byte)
	if !ok {
		t.Fatalf("noindex-only Value should return []byte (not SQL NULL), got %T", niVal)
	}
	var niRescanned SeoMeta
	if err := niRescanned.Scan(niBytes); err != nil {
		t.Fatalf("noindex-only Scan: %v", err)
	}
	if !niRescanned.Noindex {
		t.Errorf("noindex flag should round-trip true, got %+v", niRescanned)
	}
}

// TestSeoMeta_Validate_LengthLimits proves the SERP length invariants: a valid
// SeoMeta passes; a 61-rune MetaTitle fails with ErrMetaTitleTooLong; a 161-rune
// MetaDescription fails with ErrMetaDescriptionTooLong. Boundary values (exactly
// 60 / 160) are accepted.
func TestSeoMeta_Validate_LengthLimits(t *testing.T) {
	valid := SeoMeta{
		MetaTitle:       strings.Repeat("a", MaxMetaTitleLen),       // exactly 60 — allowed
		MetaDescription: strings.Repeat("b", MaxMetaDescriptionLen), // exactly 160 — allowed
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("boundary-length SeoMeta should be valid, got %v", err)
	}

	titleTooLong := SeoMeta{MetaTitle: strings.Repeat("a", MaxMetaTitleLen+1)} // 61
	if err := titleTooLong.Validate(); !errors.Is(err, ErrMetaTitleTooLong) {
		t.Errorf("MetaTitle of %d runes should fail with ErrMetaTitleTooLong, got %v",
			MaxMetaTitleLen+1, err)
	}

	descTooLong := SeoMeta{MetaDescription: strings.Repeat("b", MaxMetaDescriptionLen+1)} // 161
	if err := descTooLong.Validate(); !errors.Is(err, ErrMetaDescriptionTooLong) {
		t.Errorf("MetaDescription of %d runes should fail with ErrMetaDescriptionTooLong, got %v",
			MaxMetaDescriptionLen+1, err)
	}
}
