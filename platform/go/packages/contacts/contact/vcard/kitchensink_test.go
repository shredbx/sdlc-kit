package vcard_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/contact"
	"github.com/shredbx/sbx-core/pkg/contact/vcard"
	"github.com/shredbx/sbx-core/pkg/socialnetwork"
)

// (losslessness gate — kitchen-sink round-trip tests for the vCard mapper)

// kitchenSink builds a Contact with EVERY field populated: every scalar non-zero,
// every slice with >=2 elements, special characters in every free-text field, and
// the JSONB Data map. It is the input to the losslessness gate below — if the
// mapper silently drops any field, the field-by-field comparison fails.
func kitchenSink() contact.Contact {
	photoID := "photo-9f8e7d6c"
	c := contact.NewContact("Somchai", "Jaidee", "landlord", "owner")
	c.Title = "Khun"
	c.MiddleName = "P."
	c.DateOfBirth = "1985-03-12"
	c.Nationality = "Thai"
	c.Email = "somchai@example.com"
	c.PhoneCountryCode = "+66"
	c.PhoneNumber = "812345678"
	c.PhoneType = "mobile"
	c.Street = "12/3 Moo 5, Soi 2; Baan Tai"
	c.Unit = `Apt 4\B`
	c.SubDistrict = "Ko Pha-ngan"
	c.City = "Surat Thani"
	c.Province = "Surat Thani"
	c.PostalCode = "84280"
	c.Country = "Thailand"
	c.Latitude = 9.748918
	c.Longitude = 100.031021
	c.SocialNetworks = socialnetwork.List{
		{Platform: "line", Handle: "somchai_pv"},
		{Platform: "instagram", Handle: "somchai", URL: "https://instagram.com/somchai"},
	}
	c.Data = contact.ExtensionData{
		"bank_account": "123-4-56789-0",
		"id_number":    "1-2345-67890-12-3",
		"note_special": `comma, semi; back\slash`,
	}
	c.PhotoID = &photoID
	c.Notes = []contact.Note{
		{ID: "n1", ContactID: c.ID, Body: "Prefers afternoon viewings; cash"},
		{ID: "n2", ContactID: c.ID, Body: "Owns 3 units, Ko Pha-ngan\nbeachfront"},
	}
	return c
}

// normalizedJSON marshals a value to canonical JSON bytes for equality. It is the
// correct comparison for a JSON-typed field (ExtensionData) whose Go-side identity
// is unstable under reflect.DeepEqual (a JSON number unmarshals as float64, so an
// int input would not DeepEqual its round-trip even though the wire bytes are
// identical). Comparing the marshalled forms tests the lossless property the sync
// layer actually relies on.
func normalizedJSON(t *testing.T, v contact.ExtensionData) string {
	t.Helper()
	if len(v) == 0 {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal ExtensionData: %v", err)
	}
	return string(b)
}

// assertContactDeepEqual compares every field of two contacts. It is the
// losslessness contract: it must fail if ANY field of want is not reproduced in
// got.
//
// The ONLY allowed exceptions, each asserted/skipped explicitly with a comment:
//   - Child-Note identity (Note.ID, Note.ContactID, Note.CreatedAt): only the NOTE
//     *body* is a vCard wire field (RFC 6350 NOTE is free text with no identity).
//     The notes round-trip as a single joined NOTE, so per-note IDs/timestamps are
//     intentionally NOT carried — compared as the joined body only, below.
//   - Contact Document base (Contact.ID, CreatedAt, UpdatedAt, DeletedAt): no vCard
//     wire slot (UID is reserved for cross-device identity which the sync layer owns
//     out of band, not the address-book card body). Excluded on purpose.
func assertContactDeepEqual(t *testing.T, want, got contact.Contact) {
	t.Helper()

	scalars := []struct{ name, want, got string }{
		{"Title", want.Title, got.Title},
		{"GivenName", want.GivenName, got.GivenName},
		{"MiddleName", want.MiddleName, got.MiddleName},
		{"Surname", want.Surname, got.Surname},
		{"DateOfBirth", want.DateOfBirth, got.DateOfBirth},
		{"Nationality", want.Nationality, got.Nationality},
		{"Email", want.Email, got.Email},
		{"PhoneCountryCode", want.PhoneCountryCode, got.PhoneCountryCode},
		{"PhoneNumber", want.PhoneNumber, got.PhoneNumber},
		{"PhoneType", want.PhoneType, got.PhoneType},
		{"Street", want.Street, got.Street},
		{"Unit", want.Unit, got.Unit},
		{"SubDistrict", want.SubDistrict, got.SubDistrict},
		{"City", want.City, got.City},
		{"Province", want.Province, got.Province},
		{"PostalCode", want.PostalCode, got.PostalCode},
		{"Country", want.Country, got.Country},
	}
	for _, f := range scalars {
		if f.want != f.got {
			t.Errorf("%s: want %q, got %q", f.name, f.want, f.got)
		}
	}

	// Coordinates — exact float equality (FormatFloat 'f',-1,64 round-trips bit-exact).
	if got.Latitude != want.Latitude {
		t.Errorf("Latitude: want %v, got %v", want.Latitude, got.Latitude)
	}
	if got.Longitude != want.Longitude {
		t.Errorf("Longitude: want %v, got %v", want.Longitude, got.Longitude)
	}

	// PhotoID (*string) — compare presence + value.
	switch {
	case want.PhotoID == nil && got.PhotoID != nil:
		t.Errorf("PhotoID: want nil, got %q", *got.PhotoID)
	case want.PhotoID != nil && got.PhotoID == nil:
		t.Errorf("PhotoID: want %q, got nil", *want.PhotoID)
	case want.PhotoID != nil && got.PhotoID != nil && *want.PhotoID != *got.PhotoID:
		t.Errorf("PhotoID: want %q, got %q", *want.PhotoID, *got.PhotoID)
	}

	// CategoryCodes — order-preserving.
	if strings.Join(got.CategoryCodes, ",") != strings.Join(want.CategoryCodes, ",") {
		t.Errorf("CategoryCodes: want %v, got %v", want.CategoryCodes, got.CategoryCodes)
	}

	// SocialNetworks — full struct, in order.
	if len(got.SocialNetworks) != len(want.SocialNetworks) {
		t.Fatalf("SocialNetworks len: want %d, got %d", len(want.SocialNetworks), len(got.SocialNetworks))
	}
	for i := range want.SocialNetworks {
		if got.SocialNetworks[i] != want.SocialNetworks[i] {
			t.Errorf("SocialNetworks[%d]: want %+v, got %+v", i, want.SocialNetworks[i], got.SocialNetworks[i])
		}
	}

	// Data (JSONB map) — compared as normalized JSON (see normalizedJSON doc).
	if w, g := normalizedJSON(t, want.Data), normalizedJSON(t, got.Data); w != g {
		t.Errorf("Data: want %s, got %s", w, g)
	}

	// Notes — only the joined body is a wire field (identity carve-out above).
	if w, g := joinedNotes(want), joinedNotes(got); w != g {
		t.Errorf("Notes body: want %q, got %q", w, g)
	}
}

// TestKitchenSink_RoundTrip is the losslessness gate for the vCard mapper: every
// field of a fully-populated Contact must survive FromVCARD(ToVCARD(c)). A
// field-SUBSET test would pass while silently dropping fields — this test would
// not. It is the regression guard for the GEO/SubDistrict/Nationality/PhotoID/Data
// gaps closed in this pass.
func TestKitchenSink_RoundTrip(t *testing.T) {
	c := kitchenSink()
	out := vcard.ToVCARD(c)
	got, err := vcard.FromVCARD(out)
	if err != nil {
		t.Fatalf("FromVCARD error: %v\n--- card ---\n%s", err, out)
	}
	assertContactDeepEqual(t, c, got)
}

// TestKitchenSink_StandardGEO asserts the standard GEO carrier the review requires:
// GEO:geo:<lat>,<lng> with FormatFloat(-1) precision so the floats round-trip
// exactly (a foreign app reads coordinates without parsing an X- line).
func TestKitchenSink_StandardGEO(t *testing.T) {
	c := kitchenSink() // 9.748918, 100.031021
	card := vcard.ToVCARD(c)
	want := "GEO:geo:9.748918,100.031021"
	if !strings.Contains(card, want) {
		t.Errorf("want standard %q in:\n%s", want, card)
	}
}

// TestKitchenSink_GEOOmittedWhenZero asserts GEO is omitted when both coordinates
// are zero (no spurious GEO:geo:0,0 line).
func TestKitchenSink_GEOOmittedWhenZero(t *testing.T) {
	c := contact.NewContact("Zero", "Coords", "vendor")
	card := vcard.ToVCARD(c)
	if strings.Contains(card, "GEO:") {
		t.Errorf("zero coordinates must omit GEO, got:\n%s", card)
	}
	got, err := vcard.FromVCARD(card)
	if err != nil {
		t.Fatalf("FromVCARD error: %v", err)
	}
	if got.Latitude != 0 || got.Longitude != 0 {
		t.Errorf("coords should stay zero, got %v,%v", got.Latitude, got.Longitude)
	}
}

// TestGEO_NegativeCoords proves GEO round-trips Southern/Western-hemisphere
// coordinates (negative lat/lng) bit-exact — the kitchen-sink uses positive values,
// so this guards the sign path.
func TestGEO_NegativeCoords(t *testing.T) {
	c := contact.NewContact("South", "West", "vendor")
	c.Latitude = -33.8688197
	c.Longitude = -151.2092955
	out := vcard.ToVCARD(c)
	if !strings.Contains(out, "GEO:geo:-33.8688197,-151.2092955") {
		t.Errorf("negative GEO not emitted exactly, got:\n%s", out)
	}
	got, err := vcard.FromVCARD(out)
	if err != nil {
		t.Fatalf("FromVCARD error: %v", err)
	}
	if got.Latitude != c.Latitude || got.Longitude != c.Longitude {
		t.Errorf("negative coords: want %v,%v got %v,%v", c.Latitude, c.Longitude, got.Latitude, got.Longitude)
	}
}

// TestData_PreservesValuesNormalized proves the Data carrier is lossless at the
// wire level even for values whose Go type is unstable under DeepEqual: a JSON
// number decodes as float64, and nested structure decodes as map[string]any. The
// normalized-JSON comparison (what the sync layer relies on) holds.
func TestData_PreservesValuesNormalized(t *testing.T) {
	c := contact.NewContact("Data", "Rich", "owner")
	c.Data = contact.ExtensionData{
		"count":  float64(3), // a JSON number — would break reflect.DeepEqual as int
		"active": true,
		"tags":   []any{"a", "b"},
		"meta":   map[string]any{"k": "v, with; specials\\"},
	}
	out := vcard.ToVCARD(c)
	got, err := vcard.FromVCARD(out)
	if err != nil {
		t.Fatalf("FromVCARD error: %v\n--- card ---\n%s", err, out)
	}
	if w, g := normalizedJSON(t, c.Data), normalizedJSON(t, got.Data); w != g {
		t.Errorf("Data normalized round-trip:\n want %s\n got  %s", w, g)
	}
}

// TestKitchenSink_PhotoIDOmittedWhenNil asserts the X-SBX-PHOTO-ID line is omitted
// when PhotoID is nil and round-trips back to nil.
func TestKitchenSink_PhotoIDOmittedWhenNil(t *testing.T) {
	c := contact.NewContact("No", "Photo", "tenant")
	card := vcard.ToVCARD(c)
	if strings.Contains(card, "X-SBX-PHOTO-ID") {
		t.Errorf("nil PhotoID must omit X-SBX-PHOTO-ID, got:\n%s", card)
	}
	got, err := vcard.FromVCARD(card)
	if err != nil {
		t.Fatalf("FromVCARD error: %v", err)
	}
	if got.PhotoID != nil {
		t.Errorf("PhotoID should stay nil, got %q", *got.PhotoID)
	}
}
