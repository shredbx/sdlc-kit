// Package property provides domain types and PostgreSQL persistence for real
// estate property listings across multiple projects (bestierealestate, bestays).
//
// Named types replace raw strings for all discriminator fields:
//
//	PropertyType, TitleDeed, LandSizeUnit
//
// Listing intent is modelled as two independent boolean flags — ForSale and
// ForLease — with one optional price per mode (SalePrice, LeasePrice as
// *int64). At least one flag must be true (invariant — enforced by
// ValidateListingIntent and a Postgres CHECK constraint). Either price may
// be nil → "Price on Application".
//
// All other content fields use pointer types — nil means draft (not yet set).
// is_published is a bool (not pointer) — it is behavioral, always present.
//
// Example (BR):
//
//	mapper := property.NewPostgresMapper("bestierealestate")
//	store := postgres.NewPostgresStore[property.Property](db, mapper)
//	props, total, _ := store.List(ctx, repository.ListOptions{
//	    Filter: property.Fields.IsPublished.Eq(true),
//	})
package property

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/shredbx/sbx-core/pkg/address"
	"github.com/shredbx/sbx-core/pkg/money"
	"github.com/shredbx/sbx-core/pkg/seo"
)

// =============================================================================
// LISTING INTENT — sentinel errors for property validation
// =============================================================================

// ErrNoListingIntent indicates a property has neither ForSale nor ForLease
// set to true. At least one must be true (BR business rule — pure rental
// lives in Bestays).
var ErrNoListingIntent = errors.New("at least one of for_sale or for_lease must be true")

// ErrNegativePrice indicates a SalePrice or LeasePrice was set to a negative
// value. Prices are stored in satang (THB × 100); negative values are
// nonsensical. Use nil for "Price on Application", never a negative number.
var ErrNegativePrice = errors.New("price must be >= 0 when set; use nil for Price on Application")

// ErrInvalidBuildingSpec indicates a BuildingSpecs field is out of its sane range:
// floors < 1, a negative floor level / parking count, or a year_built/last_renovated
// outside [1800, currentYear+5] (the upper bound leaves room for off-plan listings).
var ErrInvalidBuildingSpec = errors.New("building spec value out of range")

// ErrNegativeMonths indicates a Policies month field (minimum lease, security
// deposit, or advance payment) was set to a negative value.
var ErrNegativeMonths = errors.New("policy month values must be >= 0 when set")

// =============================================================================
// NAMED TYPES — discriminator fields (VARCHAR FK to dictionary tables)
// =============================================================================

// PropertyType classifies the physical category of a property.
type PropertyType string

const (
	PropertyVilla      PropertyType = "villa"
	PropertyPoolVilla  PropertyType = "pool-villa"
	PropertyHouse      PropertyType = "house"
	PropertyTownhouse  PropertyType = "townhouse"
	PropertyCondo      PropertyType = "condo"
	PropertyApartment  PropertyType = "apartment"
	PropertyStudio     PropertyType = "studio"
	PropertyPenthouse  PropertyType = "penthouse"
	PropertyDuplex     PropertyType = "duplex"
	PropertyLoft       PropertyType = "loft"
	PropertyBungalow   PropertyType = "bungalow"
	PropertyLand       PropertyType = "land"
	PropertyOffice     PropertyType = "office"
	PropertyShop       PropertyType = "shop"
	PropertyCommercial PropertyType = "commercial"
	PropertyBuilding   PropertyType = "building"
	PropertyCabin      PropertyType = "cabin"
	PropertyFarmhouse  PropertyType = "farmhouse"
	PropertyHouseboat  PropertyType = "houseboat"
	PropertyTreehouse  PropertyType = "treehouse"
	PropertyBusiness   PropertyType = "business"
	PropertyOther      PropertyType = "other"
)

// TitleDeed classifies the Thai land ownership certificate type.
type TitleDeed string

const (
	TitleChanote    TitleDeed = "chanote"
	TitleNorSor3Gor TitleDeed = "nor-sor-3-gor"
	TitleNorSor3    TitleDeed = "nor-sor-3"
	TitleSorKor1    TitleDeed = "sor-kor-1"
	TitleLeasehold  TitleDeed = "leasehold"
	TitlePorBorTor5 TitleDeed = "por-bor-tor-5"
)

// LandSizeUnit is the measurement unit for land area.
type LandSizeUnit string

const (
	UnitSqm  LandSizeUnit = "sqm"
	UnitRai  LandSizeUnit = "rai"
	UnitNgan LandSizeUnit = "ngan"
	UnitWah  LandSizeUnit = "wah"
)

// Furnished classifies the furnishing level of a property.
type Furnished string

const (
	FurnishedFully       Furnished = "fully"
	FurnishedPartially   Furnished = "partially"
	FurnishedUnfurnished Furnished = "unfurnished"
)

// Ownership classifies HOW a buyer holds the property — distinct from TitleDeed
// (which legal certificate registers the land). A Chanote (title deed) parcel can
// be held freehold, leasehold, or via a Thai company. FK to the project-scoped
// ownership_types dictionary.
type Ownership string

const (
	OwnershipFreehold  Ownership = "freehold"
	OwnershipLeasehold Ownership = "leasehold"
	OwnershipCompany   Ownership = "company"
)

// Valid reports whether o is a seeded ownership code — a parse-time 422 guard;
// the FK enforces the same rule at the DB layer.
func (o Ownership) Valid() bool {
	switch o {
	case OwnershipFreehold, OwnershipLeasehold, OwnershipCompany:
		return true
	}
	return false
}

// Condition classifies the state of a property — a physical-quality grade (new /
// excellent / good / fair / needs-renovation) plus the readiness state move-in-ready,
// which BR seeds alongside the grades (migration 20260711000000, to respect seller
// data). FK to the project-scoped conditions dictionary; a consumer that does not
// seed a code simply never has it pass the FK.
type Condition string

const (
	ConditionNew             Condition = "new"
	ConditionExcellent       Condition = "excellent"
	ConditionGood            Condition = "good"
	ConditionFair            Condition = "fair"
	ConditionNeedsRenovation Condition = "needs-renovation"
	ConditionMoveInReady     Condition = "move-in-ready"
)

// Valid reports whether c is a seeded condition code.
func (c Condition) Valid() bool {
	switch c {
	case ConditionNew, ConditionExcellent, ConditionGood, ConditionFair,
		ConditionNeedsRenovation, ConditionMoveInReady:
		return true
	}
	return false
}

// Direction is the compass facing of a property. FK to the directions dictionary.
type Direction string

const (
	DirectionNorth     Direction = "north"
	DirectionSouth     Direction = "south"
	DirectionEast      Direction = "east"
	DirectionWest      Direction = "west"
	DirectionNortheast Direction = "northeast"
	DirectionNorthwest Direction = "northwest"
	DirectionSoutheast Direction = "southeast"
	DirectionSouthwest Direction = "southwest"
)

// Valid reports whether d is one of the eight compass directions.
func (d Direction) Valid() bool {
	switch d {
	case DirectionNorth, DirectionSouth, DirectionEast, DirectionWest,
		DirectionNortheast, DirectionNorthwest, DirectionSoutheast, DirectionSouthwest:
		return true
	}
	return false
}

// RoadAccess describes how the property is reached. FK to the road_access_types dictionary.
type RoadAccess string

const (
	RoadAccessDirect  RoadAccess = "direct"
	RoadAccessWalking RoadAccess = "walking"
	RoadAccessVehicle RoadAccess = "vehicle"
)

// Valid reports whether r is a seeded road-access code.
func (r RoadAccess) Valid() bool {
	switch r {
	case RoadAccessDirect, RoadAccessWalking, RoadAccessVehicle:
		return true
	}
	return false
}

// Topography classifies the land's terrain — a single-select scalar (mutually
// exclusive, so a dictionary field like Condition/Direction, not an amenity). FK
// to the topographies dictionary.
type Topography string

const (
	TopographyFlat     Topography = "flat"
	TopographySloped   Topography = "sloped"
	TopographyHillside Topography = "hillside"
)

// Valid reports whether t is a seeded topography code.
func (t Topography) Valid() bool {
	switch t {
	case TopographyFlat, TopographySloped, TopographyHillside:
		return true
	}
	return false
}

// LifecycleStatus describes where a listing is in its market lifecycle.
// One state at a time (Decision D-1, 2026-05-24). FK to a project-scoped
// dictionary table (BR: lifecycle_statuses) — package-level constants are
// the names BR seeds.
type LifecycleStatus string

const (
	LifecycleActive    LifecycleStatus = "active"
	LifecycleSold      LifecycleStatus = "sold"
	LifecycleLeased    LifecycleStatus = "leased"
	LifecycleWithdrawn LifecycleStatus = "withdrawn"
	LifecycleArchived  LifecycleStatus = "archived"
)

// ValidLifecycleStatus reports whether code matches one of the BR-seeded
// values. Used as a parse-time guard on the API; the FK enforces the same
// rule at the DB layer.
func ValidLifecycleStatus(code LifecycleStatus) bool {
	switch code {
	case LifecycleActive, LifecycleSold, LifecycleLeased,
		LifecycleWithdrawn, LifecycleArchived:
		return true
	}
	return false
}

// Rank is the merchandising priority of a listing — a fixed ordinal 0..5 (5 =
// highest). It orders listings rank-first when no explicit sort is chosen, and is
// the secondary tiebreaker under any explicit sort (Decision #0318). A fixed
// code-side scale (NOT a project-extensible DB dictionary) — labels are i18n-able
// at the UI boundary. Persisted as a SMALLINT NOT NULL DEFAULT 0 column; the
// zero value (RankDefault) is the unranked baseline.
type Rank int

const (
	RankDefault  Rank = 0
	RankLow      Rank = 1
	RankMedium   Rank = 2
	RankHigh     Rank = 3
	RankVeryHigh Rank = 4
	RankHighest  Rank = 5
)

// RankLabels maps each rank ordinal to its fixed English label (i18n-able at the
// UI boundary). Keys cover the full 0..5 scale.
var RankLabels = map[Rank]string{
	RankDefault:  "Default",
	RankLow:      "Low",
	RankMedium:   "Medium",
	RankHigh:     "High",
	RankVeryHigh: "Very High",
	RankHighest:  "Highest",
}

// Label returns the fixed English label for the rank, or "" for an out-of-range
// value (the Valid guard / DB CHECK keep stored ranks within 0..5).
func (r Rank) Label() string {
	return RankLabels[r]
}

// Valid reports whether r is within the 0..5 ordinal scale — a parse-time 422
// guard; the DB CHECK (rank BETWEEN 0 AND 5) enforces the same rule at the DB layer.
func (r Rank) Valid() bool {
	return r >= RankDefault && r <= RankHighest
}

// AmenityGroup categorizes amenities for UI grouping and filtering.
type AmenityGroup string

const (
	AmenityGroupBuilding     AmenityGroup = "building"
	AmenityGroupInterior     AmenityGroup = "interior"
	AmenityGroupExterior     AmenityGroup = "exterior"
	AmenityGroupLocation     AmenityGroup = "location"
	AmenityGroupNeighborhood AmenityGroup = "neighborhood"
	AmenityGroupServices     AmenityGroup = "services"
)

// Amenity is a single amenity tag with its dictionary code and group.
// Stored in {schema}.property_amenities join table.
type Amenity struct {
	Code  string       `json:"code" yaml:"code"`
	Group AmenityGroup `json:"group" yaml:"group"`
}

// Address is a type alias for address.Address. The codegen template emits
// unqualified `Address` references in the generated mapper; this alias keeps
// the in-package code addressable while the canonical type still lives in
// pkg/address. See entity.yml attribute_groups[name=Address] struct_package.
type Address = address.Address

// GeoJSONPolygon is a type alias for address.GeoJSONPolygon — same
// pattern as Address. The mapper template references the bare type name
// when scanning attribute-group columns.
type GeoJSONPolygon = address.GeoJSONPolygon

// MapView is a type alias for address.MapView — co-located with Polygon
// in property_locations.map_view JSONB. Same pattern as GeoJSONPolygon:
// the codegen template emits the unqualified type name when scanning
// attribute-group columns.
type MapView = address.MapView

// =============================================================================
// TRANSLATIONS — per-row dynamic-content localization (Decision #0278 P2)
// =============================================================================

// Translations holds per-field, per-locale localized values:
// field -> ISO-639-1 locale -> value, e.g. {"text": {"th": "..."}}.
//
// The base column (e.g. Text, Title) stays the canonical default-locale (en)
// value AND the fallback (rule LOC-001). This is a named type — never
// map[string]any — so the no-any rule holds and the shape is self-documenting.
// Shared in pkg/property so other property consumers (bestays) reuse the same
// storage + resolver. Persisted as a JSONB column via Scan/Value below.
type Translations map[string]map[string]string

// Resolve returns the localized value for (field, locale), falling back to base
// when the locale value is absent or empty (rule LOC-002). Nil-safe at every
// level: a nil Translations, a missing field key, a missing locale key, or an
// empty stored value all fall through to base. A nil base yields "".
func (t Translations) Resolve(field, locale string, base *string) string {
	fallback := ""
	if base != nil {
		fallback = *base
	}
	if t == nil {
		return fallback
	}
	locales, ok := t[field]
	if !ok || locales == nil {
		return fallback
	}
	v, ok := locales[locale]
	if !ok || v == "" {
		return fallback
	}
	return v
}

// IsZero reports whether there are no translations to persist. The mapper
// writes SQL NULL for the zero value (rule LOC-001 — nullable/omitempty).
func (t Translations) IsZero() bool {
	return len(t) == 0
}

// Value implements driver.Valuer for the translations JSONB column. Mirrors the
// pkg/address MapView/GeoJSONPolygon pattern: marshal to JSON, or nil for the
// empty map so the column reads as SQL NULL (never a literal "null"/"{}").
func (t Translations) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return json.Marshal(t)
}

// Scan implements sql.Scanner. Accepts []byte (pgx default) or string. A NULL
// column (nil src) or an empty payload scans to a nil map — not an error — so a
// row without translations round-trips to the zero value.
func (t *Translations) Scan(src any) error {
	if src == nil {
		*t = nil
		return nil
	}
	var data []byte
	switch s := src.(type) {
	case []byte:
		data = s
	case string:
		data = []byte(s)
	default:
		return fmt.Errorf("Translations.Scan: unsupported source type %T", src)
	}
	if len(data) == 0 || string(data) == "null" {
		*t = nil
		return nil
	}
	return json.Unmarshal(data, t)
}

// =============================================================================
// PROPERTY — document entity
// =============================================================================

// Property is the core real estate listing entity.
// Pointer fields are nullable — nil = draft (not yet set by the agent).
// is_published is a bool (not pointer) — always present, gates public visibility.
type Property struct {
	ID           string     `json:"id" yaml:"id"`
	Title        *string    `json:"title,omitempty" yaml:"title,omitempty" validate:"omitempty,min=1,max=500"`

	// Slug is the public URL key (/properties/{slug}) — NOT NULL, unique among LIVE
	// properties. Auto-derived from Title via the graceful candidate ladder
	// (SlugCandidates → ResolveSlug, transliterates, never empty); the slug FOLLOWS the
	// title (regenerates on title change). Every PUBLISHED slug a property has carried is
	// recorded in property_slug_history (the FriendlyId "History" pattern) so old/UUID
	// links 301 forever and a slug is never reissued to another property. A raw string
	// column (like Title) — the Slug named type guards the derivation + repository
	// signatures. Input validation is deliberately NOT `required`: the slug is derived
	// server-side from Title on Create/Update, so the API decode must accept a slugless
	// body and let the service fill it before persist (the DB NOT-NULL is satisfied by
	// the derived value, never by client input). Marking it `required` here 422s every
	// create/edit because the admin form never sends a slug — regression-guarded by the
	// PropertyManageHandler Create/Update tests.
	Slug string `json:"slug" yaml:"slug" validate:"max=200"`

	TitleDeed    *TitleDeed `json:"title_deed,omitempty" yaml:"title_deed,omitempty"`
	TitleDeedRaw *string    `json:"title_deed_raw,omitempty" yaml:"title_deed_raw,omitempty" validate:"omitempty,max=200"`
	Text         *string    `json:"text,omitempty" yaml:"text,omitempty" validate:"omitempty,max=20000"`

	// Translations — per-row localized field values (field -> locale -> value),
	// e.g. {"text": {"th": "..."}}. Title/Text above stay the canonical
	// default-locale value + fallback. Resolved at read time via
	// Translations.Resolve (rule LOC-002). Persisted as a JSONB column.
	Translations Translations `json:"translations,omitempty" yaml:"translations,omitempty"`

	PropertyType *PropertyType `json:"property_type,omitempty" yaml:"property_type,omitempty"`
	IsPublished  bool          `json:"is_published" yaml:"is_published"`

	// Listing intent — at least one must be true (Postgres CHECK + ValidateListingIntent).
	ForSale  bool `json:"for_sale" yaml:"for_sale"`
	ForLease bool `json:"for_lease" yaml:"for_lease"`

	// Per-mode prices in satang (THB × 100). nil = "Price on Application".
	// Range cap = 999,999,999,999 satang ≈ 10 billion THB — far above any realistic
	// listing while preventing int64 overflow scenarios and obvious garbage values.
	SalePrice  *int64 `json:"sale_price,omitempty" yaml:"sale_price,omitempty" validate:"omitempty,gte=0,lte=999999999999"`
	LeasePrice *int64 `json:"lease_price,omitempty" yaml:"lease_price,omitempty" validate:"omitempty,gte=0,lte=999999999999"`

	// Currency code applies to BOTH sale and lease prices (one currency per property).
	PriceCurrency *money.CurrencyCode `json:"price_currency,omitempty" yaml:"price_currency,omitempty" validate:"omitempty,currencycode"`

	// FK to images(id). Cover URL is resolved by the response builder via JOIN;
	// the property struct itself never carries the URL. Set to nil to clear the
	// cover; set to a valid image id to wire one.
	CoverImageID *string `json:"cover_image_id,omitempty" yaml:"cover_image_id,omitempty" validate:"omitempty,uuid"`

	// FK to contacts(id) — the landlord for this property. Optional. Projects
	// without a contacts table can leave it nil; the column ALTER lives in the
	// project's migrations alongside the contacts table.
	LandlordContactID *string `json:"landlord_contact_id,omitempty" yaml:"landlord_contact_id,omitempty" validate:"omitempty,uuid"`

	// FK to agents(id) — the listing agent for this property. Optional. nil means
	// "use the default agent" (resolved by the consumer's public detail handler).
	// Projects without an agents table can leave it nil; the column ALTER lives in
	// the project's migrations alongside the agents table.
	ListingAgentID *string `json:"listing_agent_id,omitempty" yaml:"listing_agent_id,omitempty" validate:"omitempty,uuid"`

	Furnished *Furnished `json:"furnished,omitempty" yaml:"furnished,omitempty"`

	// LifecycleStatus — where the listing is in its market lifecycle.
	// Single-column discriminator (Decision D-1). FK to the project-scoped
	// lifecycle_statuses dictionary. Defaults to "active" at the DB layer;
	// always present in API responses.
	LifecycleStatus LifecycleStatus `json:"lifecycle_status" yaml:"lifecycle_status"`

	// L1 superset scalars (M1). Ownership = how the buyer holds the property
	// (distinct from TitleDeed — the legal certificate). Condition / Direction /
	// RoadAccess are FK to project-scoped dictionaries; nil = not set.
	Ownership  *Ownership  `json:"ownership_type,omitempty" yaml:"ownership_type,omitempty"`
	Condition  *Condition  `json:"condition,omitempty" yaml:"condition,omitempty"`
	Direction  *Direction  `json:"direction,omitempty" yaml:"direction,omitempty"`
	RoadAccess *RoadAccess `json:"road_access,omitempty" yaml:"road_access,omitempty"`
	Topography *Topography `json:"topography,omitempty" yaml:"topography,omitempty"`

	// Rank is the merchandising priority 0..5 (Decision #0318) — a value, never a
	// pointer (NOT NULL DEFAULT 0). It leads ORDER BY when no explicit sort is set,
	// and is the secondary tiebreaker under any explicit sort. Replaces the retired
	// is_featured flag + the inert listing_priority field.
	Rank Rank `json:"rank" yaml:"rank"`

	// Per-section PUBLIC visibility (task 2606-120 P1 Batch 3). Full-privacy default:
	// each ships FALSE (private/hidden) and a manager opts IN per listing to publish the
	// section. Enforced ONLY on the public projection (the auth-gated /manage path always
	// returns everything). LocationPublic → coordinates + location map (the exact pin);
	// PlotPublic → plot polygon + plot map; PoliciesPublic → the Lease & Policies group.
	// Always present in JSON (no omitempty) so an absent value reads as the FALSE default.
	LocationPublic bool `json:"location_public" yaml:"location_public"`
	PlotPublic     bool `json:"plot_public" yaml:"plot_public"`
	PoliciesPublic bool `json:"policies_public" yaml:"policies_public"`

	// Address — value-bound attribute group composed from pkg/address.Address.
	// Zero-value via address.Address.IsZero() means "not yet provided".
	Address address.Address `json:"address,omitempty" yaml:"address,omitempty"`

	// Other attribute-groups from dependent tables (nil if row absent)
	Sizes *Sizes `json:"sizes,omitempty" yaml:"sizes,omitempty"`
	Rooms *Rooms `json:"rooms,omitempty" yaml:"rooms,omitempty"`

	// L1 superset attribute-groups (M1). Companion-loaded (LoadBuildingSpecs /
	// LoadPolicies), NOT the mapper LEFT JOIN — Policies carries TEXT[] arrays the
	// codegen cannot emit, and both are kept off the generated scan surface. nil = no row.
	BuildingSpecs *BuildingSpecs `json:"building_specs,omitempty" yaml:"building_specs,omitempty"`
	Policies      *Policies      `json:"policies,omitempty" yaml:"policies,omitempty"`

	// Containment children (M2) — sellable/leasable sub-listings within this
	// property. 1:many cascade-delete with their own UUID identity (Decision
	// #0019), companion-loaded (LoadUnits/SaveUnit/DeleteUnit), NOT the generated
	// mapper. nil/empty = no units.
	Units []PropertyUnit `json:"units,omitempty" yaml:"units,omitempty"`

	// Dictionary-backed join table (loaded separately, not via LEFT JOIN).
	// Each amenity row is validated by its own struct tags; the slice cap of 200
	// prevents pathological payloads.
	Amenities []Amenity `json:"amenities,omitempty" yaml:"amenities,omitempty" validate:"omitempty,max=200,dive"`

	// Tags are free-form custom labels (Decision #0279 PT). Stored as a TEXT[] column
	// on the properties row (NOT the codegen mapper — see LoadTags/SaveTags), holding
	// the trimmed LABEL strings, deduped by slug identity. Distinct from the structured
	// Amenities taxonomy and from derived autotags (PA, never stored). Capped at 50
	// labels × 100 chars; loaded/written via the LoadTags/SaveTags companion methods.
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty" validate:"omitempty,max=50,dive,max=100"`

	// SeoMeta — per-listing SEO + Open Graph overrides (FF3a). Pointer + omitempty:
	// nil = no overrides set (the consumer derives meta/OG from title/cover at render
	// time), and the API emits the field as absent rather than a noisy empty object on
	// every listing. Persisted as a single nullable JSONB column (properties.seo_meta)
	// round-tripped via seo.SeoMeta's Scan/Value. Like Tags, it lives OUTSIDE the
	// generated mapper and is loaded/written via the LoadSeoMeta/SaveSeoMeta companion
	// methods — the codegen scans `&p.Field`, which for a *seo.SeoMeta pointer would be
	// a **SeoMeta that pgx cannot Scan; the companion path keeps the nullable-pointer
	// contract the frontend SeoHead renderer depends on. seo.SeoMeta satisfies the
	// Bindable protocol structurally (FI-1) — no framework wrapper.
	SeoMeta *seo.SeoMeta `json:"seo_meta,omitempty" yaml:"seo_meta,omitempty"`

	// Document base
	CreatedAt time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" yaml:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty"`
}

// Sizes is a dependent attribute-group (table: {schema}.property_sizes).
// All measurements are in square meters.
type Sizes struct {
	LandSize   *float64 `json:"land_size,omitempty" yaml:"land_size,omitempty"`
	HouseSize  *float64 `json:"house_size,omitempty" yaml:"house_size,omitempty"`
	LivingSize *float64 `json:"living_size,omitempty" yaml:"living_size,omitempty"`

	// L1 superset extensions (M1) — all in sqm, nil = not provided.
	TotalArea   *float64 `json:"total_area,omitempty" yaml:"total_area,omitempty"`
	UsableArea  *float64 `json:"usable_area,omitempty" yaml:"usable_area,omitempty"`
	BalconyArea *float64 `json:"balcony_area,omitempty" yaml:"balcony_area,omitempty"`
}

// IsZero reports whether every measurement is unset (all fields nil). Used by the
// unit companion loader to collapse a row of all-NULL size columns back to a nil
// *Sizes (mirrors how Property's mapper LEFT JOIN yields nil for an absent sub-row),
// so a unit without size data round-trips to nil rather than an empty struct. A nil
// receiver is zero. Mirrors the Translations.IsZero pattern.
func (s *Sizes) IsZero() bool {
	if s == nil {
		return true
	}
	return s.LandSize == nil && s.HouseSize == nil && s.LivingSize == nil &&
		s.TotalArea == nil && s.UsableArea == nil && s.BalconyArea == nil
}

// Rooms is a dependent attribute-group (table: {schema}.property_rooms).
// Nil = unknown count; 0 = confirmed zero rooms.
type Rooms struct {
	Bedrooms    *int `json:"bedrooms,omitempty" yaml:"bedrooms,omitempty"`
	Bathrooms   *int `json:"bathrooms,omitempty" yaml:"bathrooms,omitempty"`
	Kitchens    *int `json:"kitchens,omitempty" yaml:"kitchens,omitempty"`
	LivingRooms *int `json:"living_rooms,omitempty" yaml:"living_rooms,omitempty"`

	// L1 superset extensions (M1).
	DiningRooms  *int `json:"dining_rooms,omitempty" yaml:"dining_rooms,omitempty"`
	Offices      *int `json:"offices,omitempty" yaml:"offices,omitempty"`
	StorageRooms *int `json:"storage_rooms,omitempty" yaml:"storage_rooms,omitempty"`
	MaidRooms    *int `json:"maid_rooms,omitempty" yaml:"maid_rooms,omitempty"`
	GuestRooms   *int `json:"guest_rooms,omitempty" yaml:"guest_rooms,omitempty"`
}

// IsZero reports whether every room count is unset (all fields nil). Used by the
// unit companion loader to collapse a row of all-NULL room columns back to a nil
// *Rooms — a unit without room data round-trips to nil rather than an empty struct.
// A nil receiver is zero. Note: a confirmed zero count is a non-nil *int(0), which
// is NOT zero here, so an explicit "0 bedrooms" is preserved.
func (r *Rooms) IsZero() bool {
	if r == nil {
		return true
	}
	return r.Bedrooms == nil && r.Bathrooms == nil && r.Kitchens == nil && r.LivingRooms == nil &&
		r.DiningRooms == nil && r.Offices == nil && r.StorageRooms == nil &&
		r.MaidRooms == nil && r.GuestRooms == nil
}

// BuildingSpecs is a dependent attribute-group (table: {schema}.property_building_specs).
// Loaded/written via the LoadBuildingSpecs/SaveBuildingSpecs companion methods, NOT the
// generated mapper — a 1:1 sub-table kept off the mapper LEFT JOIN to bound the generated
// scan surface (mirrors how Amenities are companion-loaded). nil = no row.
type BuildingSpecs struct {
	Floors        *int `json:"floors,omitempty" yaml:"floors,omitempty"`
	FloorLevel    *int `json:"floor_level,omitempty" yaml:"floor_level,omitempty"`
	ParkingSpaces *int `json:"parking_spaces,omitempty" yaml:"parking_spaces,omitempty"`
	YearBuilt     *int `json:"year_built,omitempty" yaml:"year_built,omitempty"`
	LastRenovated *int `json:"last_renovated,omitempty" yaml:"last_renovated,omitempty"`
}

// minBuildingYear is the lower bound for year_built / last_renovated. The upper
// bound is computed at validation time (current year + 5) to allow off-plan listings.
const minBuildingYear = 1800

func validBuildingYear(y int) bool {
	return y >= minBuildingYear && y <= time.Now().Year()+5
}

// Validate enforces sane ranges on the set fields; nil fields are skipped, and a nil
// receiver is valid (no specs row). Returns ErrInvalidBuildingSpec on the first breach.
func (b *BuildingSpecs) Validate() error {
	if b == nil {
		return nil
	}
	if b.Floors != nil && *b.Floors < 1 {
		return ErrInvalidBuildingSpec
	}
	if b.FloorLevel != nil && *b.FloorLevel < 0 {
		return ErrInvalidBuildingSpec
	}
	if b.ParkingSpaces != nil && *b.ParkingSpaces < 0 {
		return ErrInvalidBuildingSpec
	}
	if b.YearBuilt != nil && !validBuildingYear(*b.YearBuilt) {
		return ErrInvalidBuildingSpec
	}
	if b.LastRenovated != nil && !validBuildingYear(*b.LastRenovated) {
		return ErrInvalidBuildingSpec
	}
	return nil
}

// Policies is a dependent attribute-group (table: {schema}.property_policies) holding
// lease terms, rules, and fees for rental listings. The three slice fields are stored as
// TEXT[] columns and round-tripped via the LoadPolicies/SavePolicies companion methods —
// the codegen mapper cannot emit arrays (mirrors the Tags exception). nil = no row.
type Policies struct {
	Inclusions            []string `json:"inclusions,omitempty" yaml:"inclusions,omitempty"`
	Restrictions          []string `json:"restrictions,omitempty" yaml:"restrictions,omitempty"`
	HouseRules            []string `json:"house_rules,omitempty" yaml:"house_rules,omitempty"`
	MinimumLeaseMonths    *int     `json:"minimum_lease_months,omitempty" yaml:"minimum_lease_months,omitempty"`
	SecurityDepositMonths *int     `json:"security_deposit_months,omitempty" yaml:"security_deposit_months,omitempty"`
	AdvancePaymentMonths  *int     `json:"advance_payment_months,omitempty" yaml:"advance_payment_months,omitempty"`
}

// Validate enforces non-negative month values; nil fields are skipped and a nil
// receiver is valid (no policies row).
func (p *Policies) Validate() error {
	if p == nil {
		return nil
	}
	for _, v := range []*int{p.MinimumLeaseMonths, p.SecurityDepositMonths, p.AdvancePaymentMonths} {
		if v != nil && *v < 0 {
			return ErrNegativeMonths
		}
	}
	return nil
}

// ValidateListingIntent returns ErrNoListingIntent when both ForSale and
// ForLease are false. Postgres enforces the same rule via a CHECK constraint;
// this Go-side check produces a 400 with a clear field-targeted error instead
// of bubbling a constraint-violation 500.
func (p *Property) ValidateListingIntent() error {
	if !p.ForSale && !p.ForLease {
		return ErrNoListingIntent
	}
	return nil
}

// ValidatePrices returns ErrNegativePrice when SalePrice or LeasePrice is set
// to a negative value. Nil prices (POA) are always valid.
func (p *Property) ValidatePrices() error {
	if p.SalePrice != nil && *p.SalePrice < 0 {
		return ErrNegativePrice
	}
	if p.LeasePrice != nil && *p.LeasePrice < 0 {
		return ErrNegativePrice
	}
	return nil
}

// Validate aggregates all property-level invariants. Callers (REST handlers,
// CLI tools, bulk importers) get a single entry point. Returns the first
// failing invariant; chain calls if you need full validation reports.
func (p *Property) Validate() error {
	if err := p.ValidateListingIntent(); err != nil {
		return err
	}
	if err := p.ValidatePrices(); err != nil {
		return err
	}
	if err := p.BuildingSpecs.Validate(); err != nil {
		return err
	}
	if err := p.Policies.Validate(); err != nil {
		return err
	}
	for i := range p.Units {
		if err := p.Units[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}
