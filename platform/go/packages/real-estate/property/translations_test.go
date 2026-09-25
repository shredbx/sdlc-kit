package property_test

import (
	"testing"

	"github.com/shredbx/sbx-core/pkg/property"
)

func strptr(s string) *string { return &s }

// TestTranslations_Resolve_LocalePresent — (a) returns the locale value when present.
func TestTranslations_Resolve_LocalePresent(t *testing.T) {
	tr := property.Translations{"text": {"th": "บ้านสวย"}}
	base := strptr("Beautiful house")
	if got := tr.Resolve("text", "th", base); got != "บ้านสวย" {
		t.Errorf("want localized value 'บ้านสวย', got %q", got)
	}
}

// TestTranslations_Resolve_LocaleMissing — (b) falls back to base when the
// locale key is absent.
func TestTranslations_Resolve_LocaleMissing(t *testing.T) {
	tr := property.Translations{"text": {"th": "บ้านสวย"}}
	base := strptr("Beautiful house")
	// "fr" not stored → base.
	if got := tr.Resolve("text", "fr", base); got != "Beautiful house" {
		t.Errorf("missing locale: want base 'Beautiful house', got %q", got)
	}
	// field not stored → base.
	if got := tr.Resolve("title", "th", base); got != "Beautiful house" {
		t.Errorf("missing field: want base 'Beautiful house', got %q", got)
	}
}

// TestTranslations_Resolve_EmptyValueFallsBack — (c) falls back when the stored
// locale value is the empty string.
func TestTranslations_Resolve_EmptyValueFallsBack(t *testing.T) {
	tr := property.Translations{"text": {"th": ""}}
	base := strptr("Beautiful house")
	if got := tr.Resolve("text", "th", base); got != "Beautiful house" {
		t.Errorf("empty stored value: want base fallback, got %q", got)
	}
}

// TestTranslations_Resolve_NilSafe — (d) nil-safe on nil Translations / nil base.
func TestTranslations_Resolve_NilSafe(t *testing.T) {
	var tr property.Translations // nil map

	// nil Translations + base → base.
	if got := tr.Resolve("text", "th", strptr("Beautiful house")); got != "Beautiful house" {
		t.Errorf("nil Translations: want base, got %q", got)
	}
	// nil Translations + nil base → "".
	if got := tr.Resolve("text", "th", nil); got != "" {
		t.Errorf("nil Translations + nil base: want empty string, got %q", got)
	}
	// present map but nil base, locale missing → "".
	tr2 := property.Translations{"text": {"th": "บ้านสวย"}}
	if got := tr2.Resolve("text", "fr", nil); got != "" {
		t.Errorf("nil base + missing locale: want empty string, got %q", got)
	}
	// present map, nil inner locale map → base (guard against nil inner map).
	tr3 := property.Translations{"text": nil}
	if got := tr3.Resolve("text", "th", strptr("Beautiful house")); got != "Beautiful house" {
		t.Errorf("nil inner locale map: want base, got %q", got)
	}
}

// TestTranslations_ValueScan_RoundTrip verifies JSONB persistence: a non-empty
// map marshals to JSON and scans back equal; the empty/nil map writes SQL NULL.
func TestTranslations_ValueScan_RoundTrip(t *testing.T) {
	orig := property.Translations{
		"text":  {"th": "บ้านสวย"},
		"title": {"th": "วิลล่า"},
	}
	v, err := orig.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	data, ok := v.([]byte)
	if !ok {
		t.Fatalf("Value: want []byte, got %T", v)
	}

	var back property.Translations
	if err := back.Scan(data); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if back["text"]["th"] != "บ้านสวย" || back["title"]["th"] != "วิลล่า" {
		t.Errorf("round-trip mismatch: %v", back)
	}
}

// TestTranslations_Value_EmptyIsNull — the zero/empty map persists as SQL NULL
// (driver value nil), never a literal "{}" or "null" (rule LOC-001).
func TestTranslations_Value_EmptyIsNull(t *testing.T) {
	var nilMap property.Translations
	if v, err := nilMap.Value(); err != nil || v != nil {
		t.Errorf("nil map: want (nil, nil), got (%v, %v)", v, err)
	}
	empty := property.Translations{}
	if v, err := empty.Value(); err != nil || v != nil {
		t.Errorf("empty map: want (nil, nil), got (%v, %v)", v, err)
	}
}

// TestTranslations_Scan_NullSource — a NULL column (nil src) and an empty / JSON
// "null" payload scan to a nil map, not an error.
func TestTranslations_Scan_NullSource(t *testing.T) {
	var tr property.Translations
	if err := tr.Scan(nil); err != nil || tr != nil {
		t.Errorf("Scan(nil): want (nil map, no error), got (%v, %v)", tr, err)
	}
	if err := tr.Scan([]byte("null")); err != nil || tr != nil {
		t.Errorf("Scan(\"null\"): want (nil map, no error), got (%v, %v)", tr, err)
	}
	if err := tr.Scan([]byte("")); err != nil || tr != nil {
		t.Errorf("Scan(empty): want (nil map, no error), got (%v, %v)", tr, err)
	}
}
