// Package inquiry is the shared sbx-core domain for the Inquiry entity — an
// inbound lead captured from any consuming app's page form (for BR: Sell,
// Services, a property enquiry, the Guides "Ask Bestie" card). It records the
// visitor's contact details + message, the page/source it came from, an optional
// property reference, and tracks an email-style triage lifecycle
// (new → read → handled).
//
// Inquiry lives in the shared github.com/shredbx/sbx-core module
// (pkg/inquiry) — consumers import it (BR's apps/api-chi wires the handler +
// repository against it). The struct, its validation, and the postgres Mapper
// are a projection of BR's entities/inquiry/platforms/api-chi/md.yml — keep the
// column list here in exact sync with that md.yml and the migration (inquiries).
//
// Contact details REUSE pkg/contact's proven flat-phone convention: phone is
// stored as two columns (PhoneCountryCode + PhoneNumber) and reconstructed into
// a phonenumber.PhoneNumber via the Phone() accessor; socials are a JSONB
// socialnetwork.List (Value/Scan). Nullable text columns are *string (NULL-safe
// scan + omitempty JSON), matching internal/cms + pkg/property.
package inquiry

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shredbx/sbx-core/pkg/phonenumber"
	"github.com/shredbx/sbx-core/pkg/repository"
	"github.com/shredbx/sbx-core/pkg/repository/postgres"
	"github.com/shredbx/sbx-core/pkg/socialnetwork"
)

// ErrValidation wraps every validation failure. Callers test with
// errors.Is(err, inquiry.ErrValidation). Mirrors internal/cms.ErrValidation.
var ErrValidation = errors.New("validation")

// =============================================================================
// SOURCE — which kind of page/CTA produced the lead (a named type, never a raw
// string)
// =============================================================================

// InquirySource identifies the page/CTA that produced a lead. It is a named
// type — never a raw string in signatures — so the closed set travels with the
// value (standing rule: normalize discriminators).
type InquirySource string

// The v1 source set, mirroring entities/inquiry-source.
const (
	SourceSell     InquirySource = "sell"
	SourceService  InquirySource = "service"
	SourceProperty InquirySource = "property"
	SourceAsk      InquirySource = "ask"
)

// LocationOther is the sentinel location value meaning "not in the list" — it
// requires LocationCustom to carry the free-text location the visitor typed.
const LocationOther = "other"

// PropertyTypeOther is the sentinel property_type meaning "not in the list" — it
// requires PropertyTypeCustom to carry the free-text type the visitor described.
const PropertyTypeOther = "other"

// String renders the source for SQL params.
func (s InquirySource) String() string { return string(s) }

// Valid reports whether s is one of the known sources.
func (s InquirySource) Valid() bool {
	switch s {
	case SourceSell, SourceService, SourceProperty, SourceAsk:
		return true
	default:
		return false
	}
}

// Validate returns ErrValidation when s is not a known source.
func (s InquirySource) Validate() error {
	if !s.Valid() {
		return errors.New("source must be one of sell|service|property|ask")
	}
	return nil
}

// =============================================================================
// STATUS — the email-style triage lifecycle (a named type)
// =============================================================================

// InquiryStatus is the triage state of a lead. Named type — the closed triad
// (new → read → handled) travels with the value.
type InquiryStatus string

// The v1 triad, mirroring entities/inquiry-status.
const (
	StatusNew     InquiryStatus = "new"
	StatusRead    InquiryStatus = "read"
	StatusHandled InquiryStatus = "handled"
)

// String renders the status for SQL params.
func (s InquiryStatus) String() string { return string(s) }

// Valid reports whether s is one of the known triage states.
func (s InquiryStatus) Valid() bool {
	switch s {
	case StatusNew, StatusRead, StatusHandled:
		return true
	default:
		return false
	}
}

// =============================================================================
// INQUIRY — the entity
// =============================================================================

// emailPattern is a pragmatic email-shape check (a non-empty local part, an @,
// a dotted domain). It mirrors the lenient "looks like an email" intent of the
// entity.yml `contact_email` format constraint — not full RFC 5322.
var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// maxMessageLen is the entity.yml `message` maxLength constraint.
const maxMessageLen = 5000

// Inquiry is a single inbound lead. Document-type: UUID identity, timestamps,
// soft delete, optimistic version. Field order in the struct is cosmetic; the
// storage column order is fixed by PostgresMapper.Columns()/FromRow().
type Inquiry struct {
	ID string `json:"id" yaml:"id"`

	Source InquirySource `json:"source" yaml:"source"`

	ContactName  *string `json:"contact_name,omitempty" yaml:"contact_name,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty" yaml:"contact_email,omitempty"`

	// Phone stored FLAT (country code + national number), reconstructed into a
	// phonenumber.PhoneNumber via Phone() — the proven pkg/contact convention.
	PhoneCountryCode *string `json:"contact_phone_country_code,omitempty" yaml:"contact_phone_country_code,omitempty"`
	PhoneNumber      *string `json:"contact_phone_number,omitempty" yaml:"contact_phone_number,omitempty"`

	// Socials is a JSONB socialnetwork.List (same shape Contact/Agent use). An
	// empty list round-trips as SQL NULL (List.Value/Scan).
	Socials socialnetwork.List `json:"socials,omitempty" yaml:"socials,omitempty"`

	Message *string `json:"message,omitempty" yaml:"message,omitempty"`

	// PropertyRef is an OPTIONAL independent-lifecycle reference to a property
	// (nullable; sell/service/ask carry none). Deleting a property must NOT
	// delete inquiries about it.
	PropertyRef *string `json:"property_ref,omitempty" yaml:"property_ref,omitempty"`
	PageRef     *string `json:"page_ref,omitempty" yaml:"page_ref,omitempty"`

	// Location is the chosen location code (district/market dimension, e.g.
	// koh-phangan) or the LocationOther sentinel; REQUIRED. No hard FK (the "other"
	// sentinel isn't a dictionary row). When "other", LocationCustom carries the
	// free-text location.
	Location       *string `json:"location,omitempty" yaml:"location,omitempty"`
	LocationCustom *string `json:"location_custom,omitempty" yaml:"location_custom,omitempty"`
	// PropertyType is an OPTIONAL property_types(code) the inquiry is about.
	// When it is the PropertyTypeOther sentinel, PropertyTypeCustom carries the
	// free-text type the visitor described.
	PropertyType       *string `json:"property_type,omitempty" yaml:"property_type,omitempty"`
	PropertyTypeCustom *string `json:"property_type_custom,omitempty" yaml:"property_type_custom,omitempty"`

	Status InquiryStatus `json:"inquiry_status" yaml:"inquiry_status"`

	ReadAt    *time.Time `json:"read_at,omitempty" yaml:"read_at,omitempty"`
	HandledAt *time.Time `json:"handled_at,omitempty" yaml:"handled_at,omitempty"`
	HandledBy *string    `json:"handled_by,omitempty" yaml:"handled_by,omitempty"`

	Version int `json:"version" yaml:"version"`

	CreatedBy *string `json:"created_by,omitempty" yaml:"created_by,omitempty"`
	UpdatedBy *string `json:"updated_by,omitempty" yaml:"updated_by,omitempty"`

	CreatedAt time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" yaml:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty"`
}

// Phone reconstructs the embedded PhoneNumber value object from the flat fields
// (like Contact.Phone()). A nil pointer yields the empty part.
func (i Inquiry) Phone() phonenumber.PhoneNumber {
	return phonenumber.PhoneNumber{
		CountryCode: deref(i.PhoneCountryCode),
		Number:      deref(i.PhoneNumber),
	}
}

// hasContactChannel reports whether at least one follow-up channel is present:
// a non-blank email, a non-blank phone number, or at least one social entry.
func (i Inquiry) hasContactChannel() bool {
	if i.ContactEmail != nil && strings.TrimSpace(*i.ContactEmail) != "" {
		return true
	}
	if i.PhoneNumber != nil && strings.TrimSpace(*i.PhoneNumber) != "" {
		return true
	}
	for _, sn := range i.Socials {
		if !sn.IsZero() {
			return true
		}
	}
	return false
}

// Validate enforces the entity.yml constraints:
//   - source is a known InquirySource;
//   - status is a known InquiryStatus;
//   - at least one contact channel (email OR phone OR a social) is present;
//   - contact_email, when present, looks like an email;
//   - message length ≤ 5000.
//
// Social entries, when present, must each be valid (no half-filled rows). The
// caller maps the result to a 400/422 via errors.Is(err, ErrValidation).
func (i Inquiry) Validate() error {
	if err := i.Source.Validate(); err != nil {
		return err
	}
	if !i.Status.Valid() {
		return errors.New("inquiry_status must be one of new|read|handled")
	}
	if !i.hasContactChannel() {
		return errors.New("at least one contact channel (email, phone, or social) is required")
	}
	// Location is REQUIRED for form-originated leads — UNLESS a property is attached (a
	// property-attached enquiry derives its location from the property itself, so the public form
	// does not ask for it, 2607-055 S6), OR the lead came from the assistant handoff (source
	// "ask"). A chat handoff has NO location picker — it carries a contact channel + the
	// conversation as the message instead — so requiring a structured location rejected every
	// chat lead with a 400 (2607-119: "number provided → error → loop"). The public "ask" form
	// still submits a location, so relaxing it here only stops rejecting the location-less chat
	// lead. When a location IS provided, the "other" sentinel still requires LocationCustom below.
	hasProperty := i.PropertyRef != nil && strings.TrimSpace(*i.PropertyRef) != ""
	locationOptional := hasProperty || i.Source == SourceAsk
	if !locationOptional && (i.Location == nil || strings.TrimSpace(*i.Location) == "") {
		return errors.New("location is required")
	}
	if i.Location != nil && strings.EqualFold(strings.TrimSpace(*i.Location), LocationOther) {
		if i.LocationCustom == nil || strings.TrimSpace(*i.LocationCustom) == "" {
			return errors.New(`location_custom is required when location is "other"`)
		}
	}
	// Property type is optional; but if the "other" sentinel is chosen, the
	// free-text PropertyTypeCustom must be supplied.
	if i.PropertyType != nil && strings.EqualFold(strings.TrimSpace(*i.PropertyType), PropertyTypeOther) {
		if i.PropertyTypeCustom == nil || strings.TrimSpace(*i.PropertyTypeCustom) == "" {
			return errors.New(`property_type_custom is required when property_type is "other"`)
		}
	}
	if i.ContactEmail != nil && strings.TrimSpace(*i.ContactEmail) != "" {
		if !emailPattern.MatchString(strings.TrimSpace(*i.ContactEmail)) {
			return errors.New("contact_email is not a valid email address")
		}
	}
	if i.Message != nil && len(*i.Message) > maxMessageLen {
		return errors.New("message exceeds maximum length")
	}
	for idx, sn := range i.Socials {
		if sn.IsZero() {
			continue
		}
		if err := sn.Validate(); err != nil {
			return errors.New("social network entry at index " + strconv.Itoa(idx) + ": " + err.Error())
		}
	}
	return nil
}

// MarkRead flips a NEW inquiry to read exactly once: it sets Status=read and
// stamps ReadAt the first time it is opened. It is idempotent — a second call,
// or a call on an already-read/handled inquiry, is a no-op so ReadAt is set
// ONCE (the genuine first-open time) and a handled inquiry never regresses.
func (i *Inquiry) MarkRead(now time.Time) {
	if i.Status != StatusNew {
		return
	}
	i.Status = StatusRead
	t := now
	i.ReadAt = &t
}

// MarkHandled records the explicit close: Status=handled, HandledBy=actorID,
// HandledAt=now. Terminal — there is no transition out of handled.
func (i *Inquiry) MarkHandled(actorID string, now time.Time) {
	i.Status = StatusHandled
	id := actorID
	i.HandledBy = &id
	t := now
	i.HandledAt = &t
}

// deref returns the pointed-to string or "" for a nil pointer.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// =============================================================================
// POSTGRES MAPPER — projection of entities/inquiry/platforms/api-chi/md.yml
// =============================================================================

// Fields exposes typed, compile-time-safe field descriptors for List filtering
// + sorting (e.g. inquiry.Fields.Status.Eq("new")).
var Fields = struct {
	ID        repository.Field[string]
	Status    repository.Field[string]
	CreatedAt repository.Field[string]
	DeletedAt repository.Field[string]
}{
	ID:        repository.Field[string]{Name: "id"},
	Status:    repository.Field[string]{Name: "inquiry_status"},
	CreatedAt: repository.Field[string]{Name: "created_at"},
	DeletedAt: repository.Field[string]{Name: "deleted_at"},
}

// Compile-time check: PostgresMapper implements postgres.Mapper[Inquiry].
var _ postgres.Mapper[Inquiry] = PostgresMapper{}

// PostgresMapper maps Inquiry to the {schema}.inquiries table for any schema.
type PostgresMapper struct {
	schema string
}

// NewPostgresMapper creates a mapper scoped to the given PostgreSQL schema.
func NewPostgresMapper(schema string) PostgresMapper {
	return PostgresMapper{schema: schema}
}

// TableName returns the fully qualified primary table for INSERT/UPDATE/DELETE.
func (m PostgresMapper) TableName() string {
	return m.schema + ".inquiries"
}

// SelectFrom aliases the primary table — Columns() prefixes every field with the
// "i" alias, so the FROM clause must publish it.
func (m PostgresMapper) SelectFrom() string {
	return m.TableName() + " i"
}

// Columns returns the SELECT column list. Order MUST exactly match FromRow scan.
func (m PostgresMapper) Columns() []string {
	return []string{
		"i.id",
		"i.source",
		"i.contact_name",
		"i.contact_email",
		"i.contact_phone_country_code",
		"i.contact_phone_number",
		"i.socials",
		"i.message",
		"i.property_ref",
		"i.page_ref",
		"i.location",
		"i.location_custom",
		"i.property_type",
		"i.property_type_custom",
		"i.inquiry_status",
		"i.read_at",
		"i.handled_at",
		"i.handled_by",
		"i.version",
		"i.created_by",
		"i.updated_by",
		"i.created_at",
		"i.updated_at",
		"i.deleted_at",
	}
}

// FieldColumn maps logical field names to aliased columns for WHERE/ORDER BY.
func (m PostgresMapper) FieldColumn(field string) string {
	switch field {
	case "id":
		return "i.id"
	case "inquiry_status":
		return "i.inquiry_status"
	case "created_at":
		return "i.created_at"
	case "deleted_at":
		return "i.deleted_at"
	default:
		return field
	}
}

// ToRow converts an Inquiry to a column→value map for INSERT. deleted_at is
// omitted (DB default NULL); read_at/handled_at/handled_by are omitted on the
// create path (they are set later by MarkRead/MarkHandled). source + status are
// stored as their string forms.
func (m PostgresMapper) ToRow(i Inquiry) (map[string]any, error) {
	return map[string]any{
		"id":                         i.ID,
		"source":                     i.Source.String(),
		"contact_name":               i.ContactName,
		"contact_email":              i.ContactEmail,
		"contact_phone_country_code": i.PhoneCountryCode,
		"contact_phone_number":       i.PhoneNumber,
		"socials":                    i.Socials,
		"message":                    i.Message,
		"property_ref":               i.PropertyRef,
		"page_ref":                   i.PageRef,
		"location":                   i.Location,
		"location_custom":            i.LocationCustom,
		"property_type":              i.PropertyType,
		"property_type_custom":       i.PropertyTypeCustom,
		"inquiry_status":             i.Status.String(),
		"version":                    i.Version,
		"created_by":                 i.CreatedBy,
		"updated_by":                 i.UpdatedBy,
		"created_at":                 i.CreatedAt,
		"updated_at":                 i.UpdatedAt,
	}, nil
}

// FromRow scans a row into an Inquiry. Column order MUST exactly match Columns().
func (m PostgresMapper) FromRow(scan func(dest ...any) error) (Inquiry, error) {
	var i Inquiry
	err := scan(
		&i.ID,
		&i.Source,
		&i.ContactName,
		&i.ContactEmail,
		&i.PhoneCountryCode,
		&i.PhoneNumber,
		&i.Socials,
		&i.Message,
		&i.PropertyRef,
		&i.PageRef,
		&i.Location,
		&i.LocationCustom,
		&i.PropertyType,
		&i.PropertyTypeCustom,
		&i.Status,
		&i.ReadAt,
		&i.HandledAt,
		&i.HandledBy,
		&i.Version,
		&i.CreatedBy,
		&i.UpdatedBy,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
	)
	if err != nil {
		return i, err
	}
	return i, nil
}
