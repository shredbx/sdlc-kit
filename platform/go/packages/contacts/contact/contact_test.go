package contact_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/shredbx/sbx-core/pkg/contact"
	"github.com/shredbx/sbx-core/pkg/socialnetwork"
)

func TestNewContact(t *testing.T) {
	c := contact.NewContact("Somchai", "Jaidee", "landlord")
	if c.GivenName != "Somchai" {
		t.Errorf("want given_name=Somchai, got %q", c.GivenName)
	}
	if c.Surname != "Jaidee" {
		t.Errorf("want surname=Jaidee, got %q", c.Surname)
	}
	if c.PrimaryCategory() != "landlord" {
		t.Errorf("want category=landlord, got %q", c.PrimaryCategory())
	}
}

func TestContact_Validate_RequiresName(t *testing.T) {
	c := contact.Contact{CategoryCodes: []string{"landlord"}}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error on empty name, got nil")
	}
}

func TestContact_Validate_RequiresCategory(t *testing.T) {
	c := contact.Contact{GivenName: "Somchai", Surname: "Jaidee"}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error on empty category, got nil")
	}
	if !errors.Is(err, contact.ErrEmptyCategory) {
		t.Errorf("want ErrEmptyCategory, got %v", err)
	}
}

func TestContact_Validate_PhoneOnlyWhenSet(t *testing.T) {
	c := contact.NewContact("Somchai", "Jaidee", "landlord")
	if err := c.Validate(); err != nil {
		t.Errorf("zero phone should pass validation, got %v", err)
	}
}

// Phone validation is intentionally WEAK for contacts (informal address book):
// lenient/partial/odd phone data must save without forcing the user to complete
// the country-code+number pair. Storage safety is handled by column width, not
// validation. The strict phonenumber.Validate() still guards property/agent phones.
func TestContact_Validate_PhoneLenient(t *testing.T) {
	c := contact.NewContact("Somchai", "Jaidee", "landlord")
	c.PhoneCountryCode = "66" // missing '+' — tolerated for contacts
	c.PhoneNumber = "812345678"
	if err := c.Validate(); err != nil {
		t.Errorf("contact phone validation should be lenient, got %v", err)
	}

	// Number without a country code must also pass (no forcing).
	c2 := contact.NewContact("A", "B", "landlord")
	c2.PhoneNumber = "812345678"
	if err := c2.Validate(); err != nil {
		t.Errorf("number-without-country-code should pass for contacts, got %v", err)
	}
}

// Address is intentionally LENIENT for contacts: a contact book holds informal,
// partial data. Setting any single address field (or just a map pin) must NOT
// force the full postal triplet (street + city + ISO-2 country). The strict
// address.Address.Validate() stays in force for property/land/person.
func TestContact_Validate_AddressLenient(t *testing.T) {
	cases := []struct {
		name  string
		apply func(*contact.Contact)
	}{
		{"country only (free text, not ISO-2)", func(c *contact.Contact) { c.Country = "Thailand" }},
		{"city only", func(c *contact.Contact) { c.City = "Koh Phangan" }},
		{"map pin only", func(c *contact.Contact) { c.Latitude = 9.7489; c.Longitude = 100.031 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := contact.NewContact("Somchai", "Jaidee", "landlord")
			tc.apply(&c)
			if err := c.Validate(); err != nil {
				t.Errorf("partial contact address should pass, got %v", err)
			}
		})
	}

	// A genuinely broken geometry (half-set coordinate pair) must STILL fail —
	// geometry invariants are the only address guard that remains.
	t.Run("half-set coordinate pair still rejected", func(t *testing.T) {
		c := contact.NewContact("Somchai", "Jaidee", "landlord")
		c.Latitude = 9.7489 // longitude left at 0
		if err := c.Validate(); err == nil {
			t.Error("half-set coordinate pair should still fail")
		}
	})
}

// Name requires AT LEAST ONE part (given OR surname) — enough for a display label —
// rather than forcing both. No name at all is still rejected.
func TestContact_Validate_NameAtLeastOnePart(t *testing.T) {
	t.Run("given name only passes", func(t *testing.T) {
		c := contact.NewContact("Somchai", "", "landlord")
		if err := c.Validate(); err != nil {
			t.Errorf("given-name-only contact should pass, got %v", err)
		}
	})
	t.Run("surname only passes", func(t *testing.T) {
		c := contact.NewContact("", "Jaidee", "landlord")
		if err := c.Validate(); err != nil {
			t.Errorf("surname-only contact should pass, got %v", err)
		}
	})
	t.Run("no name at all fails", func(t *testing.T) {
		c := contact.NewContact("", "", "landlord")
		if err := c.Validate(); err == nil {
			t.Error("contact with no name part should fail validation")
		}
	})
}

// A fully blank social row must be silently dropped, not rejected. A row that is
// partially filled (platform set, handle missing) is NOT blank and stays an error.
func TestContact_Validate_BlankSocialDropped(t *testing.T) {
	t.Run("blank row dropped", func(t *testing.T) {
		c := contact.NewContact("Somchai", "Jaidee", "landlord")
		c.SocialNetworks = contact.SocialNetworkList{{}}
		if err := c.Validate(); err != nil {
			t.Errorf("a blank social row should be ignored, got %v", err)
		}
	})
	t.Run("partially filled row still rejected", func(t *testing.T) {
		c := contact.NewContact("Somchai", "Jaidee", "landlord")
		c.SocialNetworks = contact.SocialNetworkList{{Platform: "line"}} // handle missing
		if err := c.Validate(); err == nil {
			t.Error("a half-filled social row (platform without handle) should still fail")
		}
	})
}

func TestContact_Validate_AddressZeroSkipped(t *testing.T) {
	c := contact.NewContact("Somchai", "Jaidee", "landlord")
	if err := c.Validate(); err != nil {
		t.Errorf("zero address should pass validation, got %v", err)
	}
}

func TestContact_Validate_SocialNetworks_Valid(t *testing.T) {
	c := contact.NewContact("Somchai", "Jaidee", "landlord")
	c.SocialNetworks = contact.SocialNetworkList{
		{Platform: "line", Handle: "somchai_pv"},
		{Platform: "whatsapp", Handle: "+66812345678"},
	}
	if err := c.Validate(); err != nil {
		t.Errorf("valid social networks: unexpected error: %v", err)
	}
}

func TestContact_Validate_SocialNetworks_EmptyPlatform(t *testing.T) {
	c := contact.NewContact("Somchai", "Jaidee", "landlord")
	c.SocialNetworks = contact.SocialNetworkList{
		{Platform: "", Handle: "somchai"},
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error on empty platform, got nil")
	}
	if !errors.Is(err, contact.ErrInvalidSocialNetwork) {
		t.Errorf("want ErrInvalidSocialNetwork, got %v", err)
	}
}

func TestContact_IsZero(t *testing.T) {
	if !(contact.Contact{}).IsZero() {
		t.Error("default Contact{} should be IsZero")
	}
	c := contact.NewContact("Somchai", "Jaidee", "landlord")
	if c.IsZero() {
		t.Error("Contact with name should NOT be IsZero")
	}
}

func TestContact_DisplayName(t *testing.T) {
	c := contact.NewContact("Somchai", "Jaidee", "landlord")
	if got := c.DisplayName(); got != "Somchai Jaidee" {
		t.Errorf("want 'Somchai Jaidee', got %q", got)
	}
}

func TestContact_Name_ReturnsPersonName(t *testing.T) {
	c := contact.NewContact("Somchai", "Jaidee", "landlord")
	c.Title = "Mr"
	c.MiddleName = "Phan"
	name := c.Name()
	if name.Title != "Mr" || name.GivenName != "Somchai" || name.MiddleName != "Phan" || name.Surname != "Jaidee" {
		t.Errorf("Name() round-trip mismatch: %+v", name)
	}
}

func TestContact_Phone_ReturnsPhoneNumber(t *testing.T) {
	c := contact.NewContact("Somchai", "Jaidee", "landlord")
	c.PhoneCountryCode = "+66"
	c.PhoneNumber = "812345678"
	c.PhoneType = "mobile"
	ph := c.Phone()
	if ph.CountryCode != "+66" || ph.Number != "812345678" || ph.PhoneType != "mobile" {
		t.Errorf("Phone() round-trip mismatch: %+v", ph)
	}
}

func TestContact_PostalAddress_ReturnsAddress(t *testing.T) {
	c := contact.NewContact("Somchai", "Jaidee", "landlord")
	c.Street = "88/12 Moo 6"
	c.City = "Koh Phangan"
	c.Country = "TH"
	c.Latitude = 9.7489
	c.Longitude = 100.0310
	a := c.PostalAddress()
	if a.Street != "88/12 Moo 6" || a.City != "Koh Phangan" || a.Country != "TH" {
		t.Errorf("PostalAddress() round-trip mismatch: %+v", a)
	}
	if a.Latitude != 9.7489 || a.Longitude != 100.0310 {
		t.Errorf("PostalAddress() coords mismatch: %+v", a)
	}
}

func TestSocialNetworkList_Value_Empty(t *testing.T) {
	v, err := contact.SocialNetworkList{}.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if v != nil {
		t.Errorf("empty list should Value()=nil, got %v", v)
	}
}

func TestSocialNetworkList_Value_NonEmpty(t *testing.T) {
	l := contact.SocialNetworkList{
		{Platform: "line", Handle: "somchai", URL: "https://line.me/ti/p/X"},
	}
	v, err := l.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	b, ok := v.([]byte)
	if !ok {
		t.Fatalf("want []byte, got %T", v)
	}
	var out []socialnetwork.SocialNetwork
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 1 || out[0].Platform != "line" {
		t.Errorf("round-trip mismatch: %+v", out)
	}
}

func TestSocialNetworkList_Scan_RoundTrip(t *testing.T) {
	original := contact.SocialNetworkList{
		{Platform: "line", Handle: "somchai"},
		{Platform: "instagram", Handle: "somchai_pv"},
	}
	v, err := original.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	var rescanned contact.SocialNetworkList
	if err := rescanned.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(rescanned) != 2 {
		t.Fatalf("want 2 entries, got %d", len(rescanned))
	}
	if rescanned[0].Platform != "line" || rescanned[1].Handle != "somchai_pv" {
		t.Errorf("round-trip mismatch: %+v", rescanned)
	}
}

func TestSocialNetworkList_Scan_Nil(t *testing.T) {
	var l contact.SocialNetworkList
	if err := l.Scan(nil); err != nil {
		t.Fatalf("Scan nil: %v", err)
	}
	if l != nil {
		t.Errorf("Scan(nil) should produce nil list, got %+v", l)
	}
}

func TestExtensionData_Value_Empty(t *testing.T) {
	v, err := contact.ExtensionData{}.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if v != nil {
		t.Errorf("empty data should Value()=nil, got %v", v)
	}
}

func TestExtensionData_RoundTrip(t *testing.T) {
	original := contact.ExtensionData{
		"preferred_time": "7pm",
		"spouse":         "Wanida",
		"vip":            true,
	}
	v, err := original.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	var rescanned contact.ExtensionData
	if err := rescanned.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if rescanned["preferred_time"] != "7pm" || rescanned["spouse"] != "Wanida" {
		t.Errorf("round-trip mismatch: %+v", rescanned)
	}
}
