package vcard_test

import (
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/contact"
	"github.com/shredbx/sbx-core/pkg/contact/vcard"
	"github.com/shredbx/sbx-core/pkg/socialnetwork"
)

func full() contact.Contact {
	c := contact.NewContact("Somchai", "Jaidee", "landlord", "owner")
	c.Title = "Khun"
	c.MiddleName = "P."
	c.Email = "somchai@example.com"
	c.PhoneCountryCode = "+66"
	c.PhoneNumber = "812345678"
	c.PhoneType = "mobile"
	c.Street = "12/3 Moo 5"
	c.Unit = "Apt 4"
	c.SubDistrict = "Ko Pha-ngan"
	c.City = "Surat Thani"
	c.Province = "Surat Thani"
	c.PostalCode = "84280"
	c.Country = "Thailand"
	c.DateOfBirth = "1985-03-12"
	c.SocialNetworks = socialnetwork.List{
		{Platform: "line", Handle: "somchai_pv"},
		{Platform: "instagram", Handle: "somchai", URL: "https://instagram.com/somchai"},
	}
	c.Notes = []contact.Note{
		{ID: "n1", ContactID: c.ID, Body: "Prefers afternoon viewings"},
	}
	return c
}

// assertContactEqual compares the wire-relevant fields a vCard carries. IDs and
// timestamps of child notes are NOT part of the vCard contract — the NOTE text is.
func assertContactEqual(t *testing.T, want, got contact.Contact) {
	t.Helper()
	if got.Title != want.Title {
		t.Errorf("Title: want %q, got %q", want.Title, got.Title)
	}
	if got.GivenName != want.GivenName {
		t.Errorf("GivenName: want %q, got %q", want.GivenName, got.GivenName)
	}
	if got.MiddleName != want.MiddleName {
		t.Errorf("MiddleName: want %q, got %q", want.MiddleName, got.MiddleName)
	}
	if got.Surname != want.Surname {
		t.Errorf("Surname: want %q, got %q", want.Surname, got.Surname)
	}
	if got.Email != want.Email {
		t.Errorf("Email: want %q, got %q", want.Email, got.Email)
	}
	if got.PhoneCountryCode != want.PhoneCountryCode {
		t.Errorf("PhoneCountryCode: want %q, got %q", want.PhoneCountryCode, got.PhoneCountryCode)
	}
	if got.PhoneNumber != want.PhoneNumber {
		t.Errorf("PhoneNumber: want %q, got %q", want.PhoneNumber, got.PhoneNumber)
	}
	if got.PhoneType != want.PhoneType {
		t.Errorf("PhoneType: want %q, got %q", want.PhoneType, got.PhoneType)
	}
	// Compare the postal fields the vCard ADR carries (address.Address itself is
	// not comparable — it embeds a GeoJSONPolygon).
	for _, f := range []struct{ name, want, got string }{
		{"Street", want.Street, got.Street},
		{"Unit", want.Unit, got.Unit},
		{"City", want.City, got.City},
		{"Province", want.Province, got.Province},
		{"PostalCode", want.PostalCode, got.PostalCode},
		{"Country", want.Country, got.Country},
	} {
		if f.want != f.got {
			t.Errorf("Address.%s: want %q, got %q", f.name, f.want, f.got)
		}
	}
	if got.DateOfBirth != want.DateOfBirth {
		t.Errorf("DateOfBirth: want %q, got %q", want.DateOfBirth, got.DateOfBirth)
	}
	if strings.Join(got.CategoryCodes, ",") != strings.Join(want.CategoryCodes, ",") {
		t.Errorf("CategoryCodes: want %v, got %v", want.CategoryCodes, got.CategoryCodes)
	}
	if len(got.SocialNetworks) != len(want.SocialNetworks) {
		t.Fatalf("SocialNetworks len: want %d, got %d", len(want.SocialNetworks), len(got.SocialNetworks))
	}
	for i := range want.SocialNetworks {
		if got.SocialNetworks[i] != want.SocialNetworks[i] {
			t.Errorf("SocialNetworks[%d]: want %+v, got %+v", i, want.SocialNetworks[i], got.SocialNetworks[i])
		}
	}
	// Notes round-trip as a single joined NOTE body.
	if wantNote, gotNote := joinedNotes(want), joinedNotes(got); wantNote != gotNote {
		t.Errorf("Notes body: want %q, got %q", wantNote, gotNote)
	}
}

func joinedNotes(c contact.Contact) string {
	parts := make([]string, 0, len(c.Notes))
	for _, n := range c.Notes {
		parts = append(parts, n.Body)
	}
	return strings.Join(parts, "\n")
}

func roundTrip(t *testing.T, c contact.Contact) contact.Contact {
	t.Helper()
	out := vcard.ToVCARD(c)
	got, err := vcard.FromVCARD(out)
	if err != nil {
		t.Fatalf("FromVCARD error: %v\n--- card ---\n%s", err, out)
	}
	return got
}

func TestRoundTrip_Full(t *testing.T) {
	c := full()
	got := roundTrip(t, c)
	assertContactEqual(t, c, got)
}

func TestRoundTrip_MinimalNameOnly(t *testing.T) {
	c := contact.NewContact("Jane", "Doe", "tenant")
	got := roundTrip(t, c)
	assertContactEqual(t, c, got)
}

func TestRoundTrip_EmptyOptionals(t *testing.T) {
	// Name + one category only — every optional absent. The card must omit the
	// optional properties and round-trip back to empty.
	c := contact.NewContact("A", "B", "vendor")
	card := vcard.ToVCARD(c)
	for _, prop := range []string{"EMAIL:", "TEL", "ADR", "BDAY:", "NOTE:", "URL", "X-SBX-SOCIAL"} {
		if strings.Contains(card, prop) {
			t.Errorf("empty optional %q should be omitted, got:\n%s", prop, card)
		}
	}
	got := roundTrip(t, c)
	assertContactEqual(t, c, got)
}

func TestRoundTrip_SocialWithoutURL(t *testing.T) {
	// A social with only platform+handle (no URL) must round-trip losslessly via
	// the X- property, not be dropped because it lacks a URL.
	c := contact.NewContact("Lek", "S", "guest")
	c.SocialNetworks = socialnetwork.List{{Platform: "line", Handle: "lek123"}}
	got := roundTrip(t, c)
	if len(got.SocialNetworks) != 1 {
		t.Fatalf("want 1 social, got %d", len(got.SocialNetworks))
	}
	if got.SocialNetworks[0].Platform != "line" || got.SocialNetworks[0].Handle != "lek123" {
		t.Errorf("social round-trip wrong: %+v", got.SocialNetworks[0])
	}
	if got.SocialNetworks[0].URL != "" {
		t.Errorf("URL should stay empty, got %q", got.SocialNetworks[0].URL)
	}
}

func TestRoundTrip_SpecialChars(t *testing.T) {
	cases := []struct {
		name    string
		surname string
		note    string
	}{
		{"comma", "Smith, Jr", "buyer, serious, cash"},
		{"semicolon", "O;Brien", "alarm; keys; manual"},
		{"newline", "Multi\nLine", "line1\nline2"},
		{"backslash", `Back\slash`, `path\to\x`},
		{"all", `a,b;c\d`, "x,y;z\nw\\v"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := contact.NewContact("Given", tc.surname, "landlord")
			c.Notes = []contact.Note{{ID: "n1", Body: tc.note}}
			got := roundTrip(t, c)
			if got.Surname != tc.surname {
				t.Errorf("surname escaping: want %q, got %q", tc.surname, got.Surname)
			}
			if joinedNotes(got) != tc.note {
				t.Errorf("note escaping: want %q, got %q", tc.note, joinedNotes(got))
			}
		})
	}
}

func TestFolding_LongLine(t *testing.T) {
	long := strings.Repeat("A very long note that easily exceeds seventy five octets per line. ", 4)
	c := contact.NewContact("Long", "Note", "landlord")
	c.Notes = []contact.Note{{ID: "n1", Body: long}}
	card := vcard.ToVCARD(c)
	for _, line := range strings.Split(card, "\r\n") {
		if len(line) > 75 {
			t.Errorf("line exceeds 75 octets (%d): %q", len(line), line)
		}
	}
	got := roundTrip(t, c)
	if joinedNotes(got) != long {
		t.Errorf("folded long note did not round-trip:\nwant %q\ngot  %q", long, joinedNotes(got))
	}
}

func TestToVCARD_StructuralLines(t *testing.T) {
	c := full()
	card := vcard.ToVCARD(c)
	mustContain := []string{
		"BEGIN:VCARD",
		"VERSION:4.0",
		"END:VCARD",
		"FN:Khun Somchai P. Jaidee", // FormalName/Full includes title? FN = full display
		"N:Jaidee;Somchai;P.;Khun;",
	}
	for _, want := range mustContain {
		if !strings.Contains(card, want) {
			t.Errorf("card missing %q:\n%s", want, card)
		}
	}
	if !strings.HasSuffix(card, "\r\n") {
		t.Error("card must end with CRLF")
	}
}

func TestFromVCARD_RequiresVersion(t *testing.T) {
	// A card without VERSION is malformed (RFC 6350 requires VERSION right after BEGIN).
	card := "BEGIN:VCARD\r\nFN:No Version\r\nN:Version;No;;;\r\nEND:VCARD\r\n"
	if _, err := vcard.FromVCARD(card); err == nil {
		t.Error("expected error for vCard without VERSION")
	}
}

func TestADR_StructuredComponents(t *testing.T) {
	c := full()
	card := vcard.ToVCARD(c)
	// ADR has 7 components: po-box;extended(unit);street;locality(city);region(province);postal;country
	want := "ADR;TYPE=home:;Apt 4;12/3 Moo 5;Surat Thani;Surat Thani;84280;Thailand"
	if !strings.Contains(card, want) {
		t.Errorf("ADR components wrong, want %q in:\n%s", want, card)
	}
}
