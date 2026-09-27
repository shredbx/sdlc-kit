// Package contact provides the universal business contact-book entity — a
// person we deal with (landlord, tenant, customer, guest, vendor) outside the
// auth/RBAC user system.
//
// Contact is intentionally NOT linked to pkg/auth, pkg/user, or pkg/rbac.
// Contacts are data, not actors — they have no login, no role, no permission
// surface. The same physical person may exist both as a user (with login) and
// as a contact (address-book entry) — these are separate rows in separate
// tables with no FK between them.
//
// Contact stores name, phone, and address as FLAT first-class fields (one
// column each). Value-object accessors (Name(), Phone(), PostalAddress())
// reconstruct the embedded primitives on demand for callers that prefer the
// value-object API. The flat storage is required by the postgres-mapper
// codegen template (`sbx generate mapper`) which accesses fields by direct
// struct path.
//
// Compactness: SocialNetworks is a JSONB array of pkg/socialnetwork.SocialNetwork.
// Data is a JSONB map for 1Password-style custom fields per category.
//
// Notes is an ORDERED child collection (creation-time ascending — newest appended
// at the bottom), each note its own row in the contact_notes table. It is managed
// append + delete only (no edit, no bulk save) via the NoteStore companion, so —
// like CategoryCodes — it is intentionally absent from the generated mapper and is
// NOT part of the create/update payload.
//
// Example:
//
//	c := contact.NewContact("Somchai", "Jaidee", "landlord", "owner")
//	c.PhoneCountryCode = "+66"
//	c.PhoneNumber = "812345678"
//	c.Email = "somchai@gmail.com"
//	c.SocialNetworks = contact.SocialNetworkList{
//	    {Platform: "line", Handle: "somchai_pv"},
//	}
//	if err := c.Validate(); err != nil { ... }
package contact

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shredbx/sbx-core/pkg/address"
	"github.com/shredbx/sbx-core/pkg/personname"
	"github.com/shredbx/sbx-core/pkg/phonenumber"
	"github.com/shredbx/sbx-core/pkg/socialnetwork"
)

// =============================================================================
// SENTINEL ERRORS
// =============================================================================

// ErrEmptyCategory indicates a contact was created (or left) with no category
// codes. Every contact must belong to AT LEAST ONE category (landlord, tenant,
// etc) so the directory UI can filter and per-category extension schemas can
// apply. This is the min-one invariant: the category set is never empty.
var ErrEmptyCategory = errors.New("contact must have at least one category_code")

// ErrInvalidSocialNetwork indicates one of the SocialNetworks entries failed
// its own validation (empty platform or handle).
var ErrInvalidSocialNetwork = errors.New("social network entry invalid")

// ErrEmptyName indicates a contact has neither a given name nor a surname.
// Contacts are lenient (a single name part is enough for a display label), but
// some identity is required so DisplayName() never renders blank in lists.
var ErrEmptyName = errors.New("contact must have at least a given name or surname")

// =============================================================================
// SOCIAL NETWORK LIST — JSONB array round-trip
// =============================================================================

// SocialNetworkList is the JSONB list of social-profile references stored on a
// contact. It is a TYPE ALIAS for the promoted socialnetwork.List — the single
// reusable wrapper now shared by contacts, agents, and the CMS business-contact
// block. Existing contact.SocialNetworkList{...} literals + the contacts mapper
// keep compiling unchanged; the Value/Scan/Validate methods live on
// socialnetwork.List (empty → SQL NULL, null/empty → nil).
type SocialNetworkList = socialnetwork.List

// =============================================================================
// EXTENSION DATA — JSONB map round-trip
// =============================================================================

// ExtensionData is a free-form key/value map stored as a JSONB column.
// Used for 1Password-style custom fields per category — bank account,
// ID number, preferred contact time, spouse name, etc.
type ExtensionData map[string]any

// Value implements driver.Valuer for JSONB columns.
func (d ExtensionData) Value() (driver.Value, error) {
	if len(d) == 0 {
		return nil, nil
	}
	return json.Marshal(d)
}

// Scan implements sql.Scanner. Accepts []byte (pgx default) or string.
func (d *ExtensionData) Scan(src interface{}) error {
	if src == nil {
		*d = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("ExtensionData.Scan: unsupported source type %T", src)
	}
	if len(data) == 0 || string(data) == "null" {
		*d = nil
		return nil
	}
	return json.Unmarshal(data, d)
}

// =============================================================================
// NOTE — an ordered child note
// =============================================================================

// Note is a single free-form note attached to a contact. Notes form an ordered
// collection (by CreatedAt ascending — newest appended at the bottom), each one
// its own row in the contact_notes table. They are append + delete only (no edit,
// no bulk save): a note is added via NoteStore.Add and removed via NoteStore.Delete,
// never mutated in place. The set is a normalized child collection (not a contacts
// column), so it is absent from the generated mapper and loaded via NoteStore.
type Note struct {
	ID        string    `json:"id" yaml:"id"`
	ContactID string    `json:"contact_id" yaml:"contact_id"`
	Body      string    `json:"body" yaml:"body"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
}

// =============================================================================
// CONTACT — the entity
// =============================================================================

// Contact is a business address-book entry. Document-type: has UUID identity,
// timestamps, soft delete. Fields are flat to match the postgres-mapper
// codegen template; reconstruct value objects via Name(), Phone(),
// PostalAddress() accessors.
type Contact struct {
	ID string `json:"id" yaml:"id"`

	// PersonName expansion (4 cols)
	Title      string `json:"title,omitempty" yaml:"title,omitempty"`
	GivenName  string `json:"given_name" yaml:"given_name"`
	MiddleName string `json:"middle_name,omitempty" yaml:"middle_name,omitempty"`
	Surname    string `json:"surname" yaml:"surname"`

	// Identity bits
	DateOfBirth string `json:"date_of_birth,omitempty" yaml:"date_of_birth,omitempty"`
	Nationality string `json:"nationality,omitempty" yaml:"nationality,omitempty"`

	// CategoryCodes is the set of contact-category codes this contact belongs to
	// (landlord, owner, buyer, seller, tenant). INVARIANT: non-empty (≥1).
	//
	// The set is a normalized many-to-many relation stored in the
	// contact_category_links join table, NOT a column on the contacts row — so it
	// is intentionally absent from the generated mapper (mirrors Property.Amenities)
	// and is loaded/written via the CategoryStore companion (Load/Save/Add/Remove).
	// CategoryStore.Load builds the slice ordered by the dictionary sort_order, so
	// the FIRST code is the primary (PrimaryCategory()).
	CategoryCodes []string `json:"category_codes" yaml:"category_codes"`

	Email string `json:"email,omitempty" yaml:"email,omitempty"`

	// PhoneNumber expansion (3 cols)
	PhoneCountryCode string `json:"phone_country_code,omitempty" yaml:"phone_country_code,omitempty"`
	PhoneNumber      string `json:"phone_number,omitempty" yaml:"phone_number,omitempty"`
	PhoneType        string `json:"phone_type,omitempty" yaml:"phone_type,omitempty"`

	// Address expansion (9 cols)
	Street      string  `json:"street,omitempty" yaml:"street,omitempty"`
	Unit        string  `json:"unit,omitempty" yaml:"unit,omitempty"`
	SubDistrict string  `json:"sub_district,omitempty" yaml:"sub_district,omitempty"`
	City        string  `json:"city,omitempty" yaml:"city,omitempty"`
	Province    string  `json:"province,omitempty" yaml:"province,omitempty"`
	PostalCode  string  `json:"postal_code,omitempty" yaml:"postal_code,omitempty"`
	Country     string  `json:"country,omitempty" yaml:"country,omitempty"`
	Latitude    float64 `json:"latitude,omitempty" yaml:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty" yaml:"longitude,omitempty"`

	// JSONB fields
	SocialNetworks SocialNetworkList `json:"social_networks,omitempty" yaml:"social_networks,omitempty"`
	Data           ExtensionData     `json:"data,omitempty" yaml:"data,omitempty"`

	// Identity refs
	PhotoID *string `json:"photo_id,omitempty" yaml:"photo_id,omitempty"`

	// Notes is the contact's ordered note collection (creation-time ascending —
	// newest at the bottom). INVARIANT (on delete only): a contact that HAS notes
	// keeps ≥1 — the last remaining note cannot be deleted (NoteStore.Delete returns
	// ErrLastNote); a fresh contact legitimately has ZERO notes.
	//
	// Notes are a normalized child collection stored in the contact_notes table, NOT
	// a column on the contacts row — so this field is intentionally absent from the
	// generated mapper (mirrors CategoryCodes) and is loaded via the NoteStore
	// companion (Load/Add/Delete). It is NOT part of the create/update payload;
	// notes are appended/deleted individually via dedicated endpoints.
	Notes []Note `json:"notes" yaml:"notes,omitempty"`

	// Document base
	CreatedAt time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" yaml:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty"`
}

// NewContact creates a Contact with required given/surname + at least one
// category. Pass one or more category codes; the min-one invariant is checked by
// Validate, not the constructor (so a partially built draft can be assembled).
func NewContact(givenName, surname string, categoryCodes ...string) Contact {
	return Contact{GivenName: givenName, Surname: surname, CategoryCodes: categoryCodes}
}

// Name returns the embedded PersonName value object reconstructed from the
// flat name fields. Use this when callers prefer the value-object API.
func (c Contact) Name() personname.PersonName {
	return personname.PersonName{
		Title:      c.Title,
		GivenName:  c.GivenName,
		MiddleName: c.MiddleName,
		Surname:    c.Surname,
	}
}

// Phone returns the embedded PhoneNumber value object reconstructed from the
// flat phone fields.
func (c Contact) Phone() phonenumber.PhoneNumber {
	return phonenumber.PhoneNumber{
		CountryCode: c.PhoneCountryCode,
		Number:      c.PhoneNumber,
		PhoneType:   c.PhoneType,
	}
}

// PostalAddress returns the embedded Address value object reconstructed from
// the flat address fields.
func (c Contact) PostalAddress() address.Address {
	return address.Address{
		Street:      c.Street,
		Unit:        c.Unit,
		SubDistrict: c.SubDistrict,
		City:        c.City,
		Province:    c.Province,
		PostalCode:  c.PostalCode,
		Country:     c.Country,
		Latitude:    c.Latitude,
		Longitude:   c.Longitude,
	}
}

// Validate checks that the contact has the required identity bits and that any
// embedded non-zero value object is itself valid.
//
// Required: Name (given_name + surname) + CategoryCode. Phone, Address, Email
// are optional; only validated when present. SocialNetworks entries are
// validated individually.
func (c Contact) Validate() error {
	// Name is intentionally lenient for contacts: at least ONE of given/surname is
	// enough for a usable display label (a landlord known only as "Khun Somchai", a
	// company, a nickname). The strict personname.Validate() (both parts required)
	// stays in force for pkg/person — only the contact surface is relaxed here.
	if strings.TrimSpace(c.GivenName) == "" && strings.TrimSpace(c.Surname) == "" {
		return ErrEmptyName
	}
	// Min-one invariant: the category set must contain at least one non-blank code.
	if !hasNonBlank(c.CategoryCodes) {
		return ErrEmptyCategory
	}
	// Phone is intentionally lenient for contacts: a contact book holds informal,
	// partial data, so a number without a country code (or a malformed/odd code)
	// must save without forcing the user to complete the pair. The strict
	// phonenumber.Validate() (requires "+country code" AND number) stays in force
	// for property/agent phones — only the contact surface is relaxed here.
	// Column widths (phone_country_code widened to VARCHAR(20) in migration
	// 20260527400000) are the only hard limit, so lenient input is stored, not
	// rejected with a 500.
	// Address is lenient for contacts: only the geometry invariants (coordinate
	// pair + polygon shape) are enforced — postal completeness (street/city/country)
	// is NOT forced, matching pkg/property's draft handling. The strict
	// Address.Validate() stays for property-postal / land / person.
	if !c.PostalAddress().IsZero() {
		if err := c.PostalAddress().ValidateGeometry(); err != nil {
			return err
		}
	}
	// Blank social rows are silently dropped (an agent who added an empty row and
	// left it should not be blocked). A partially filled row — platform without a
	// handle, or vice versa — is NOT blank and remains a validation error.
	for i, sn := range c.SocialNetworks {
		if sn.IsZero() {
			continue
		}
		if err := sn.Validate(); err != nil {
			return fmt.Errorf("%w: index %d: %v", ErrInvalidSocialNetwork, i, err)
		}
	}
	return nil
}

// IsZero reports whether the contact carries no usable data.
func (c Contact) IsZero() bool {
	return c.ID == "" &&
		c.Name().IsZero() &&
		c.Phone().IsZero() &&
		c.PostalAddress().IsZero() &&
		c.DateOfBirth == "" &&
		c.Nationality == "" &&
		len(c.CategoryCodes) == 0 &&
		c.Email == "" &&
		c.PhotoID == nil &&
		len(c.SocialNetworks) == 0 &&
		len(c.Notes) == 0 &&
		len(c.Data) == 0
}

// DisplayName returns a single-line display label using the embedded PersonName.
// Used by cached FK columns (e.g. property.landlord_contact_display_name).
func (c Contact) DisplayName() string {
	return c.Name().FullName()
}

// =============================================================================
// CATEGORY SET — helpers (contain the single→multi call-site churn)
// =============================================================================

// HasCategory reports whether the contact's category set contains code. This is
// the membership replacement for the old `c.CategoryCode == code` comparison —
// the section-reveal logic asks "is this contact a landlord?" via
// c.HasCategory("landlord").
func (c Contact) HasCategory(code string) bool {
	for _, got := range c.CategoryCodes {
		if got == code {
			return true
		}
	}
	return false
}

// PrimaryCategory returns the contact's primary category code, or "" when the
// set is empty.
//
// The set is built ordered by the contact_categories dictionary sort_order (see
// CategoryStore.Load), so the FIRST code is by construction the lowest-sort_order
// (primary) one. Callers use it for the default badge, the directory list sort
// key, and the suggested transaction role. No privileged "primary" code is
// persisted — the primary is derived from the set's order.
func (c Contact) PrimaryCategory() string {
	if len(c.CategoryCodes) == 0 {
		return ""
	}
	return c.CategoryCodes[0]
}

// CategoryLabels maps the contact's category codes to their human labels via the
// supplied dictionary, preserving set order (primary first). An unknown code
// falls back to the code itself so the UI never renders blank.
func (c Contact) CategoryLabels(d Dictionary) []string {
	if len(c.CategoryCodes) == 0 {
		return nil
	}
	out := make([]string, 0, len(c.CategoryCodes))
	for _, code := range c.CategoryCodes {
		if label, ok := d[code]; ok && label != "" {
			out = append(out, label)
			continue
		}
		out = append(out, code)
	}
	return out
}

// Dictionary maps a contact-category code to its human label. It is the minimal
// read surface the CategoryLabels helper needs — callers build it from the
// project-scoped contact_categories table (code → label). Keeping it a plain map
// (rather than importing a storage type) preserves pkg/contact's zero-storage
// dependency rule for the PD layer.
type Dictionary map[string]string

// hasNonBlank reports whether codes contains at least one non-whitespace entry.
func hasNonBlank(codes []string) bool {
	for _, code := range codes {
		if strings.TrimSpace(code) != "" {
			return true
		}
	}
	return false
}
