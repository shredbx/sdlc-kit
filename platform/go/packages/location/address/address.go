// Package address provides a reusable embedded type for postal/physical addresses
// with optional GPS coordinates following the WGS84 standard.
//
// Address is a value object with embedded reference semantics — consuming entities
// gain typed columns ({attr}_street, {attr}_unit, {attr}_sub_district, {attr}_city,
// {attr}_province, {attr}_postal_code, {attr}_country, {attr}_latitude,
// {attr}_longitude) via 9-column expansion.
//
// SubDistrict is the level BELOW City (Thai ตำบล, Indian taluk, Philippine
// barangay). Optional; only required for jurisdictions that use it administratively.
//
// Address composes the geo-coordinate type for latitude/longitude validation,
// delegating WGS84 bounds checking to the GeoCoordinate value object.
//
// Example:
//
//	addr := address.NewAddress("123 Moo 4, Bantai", "Koh Phangan", "TH")
//	addr.Latitude = 9.7489
//	addr.Longitude = 100.0310
//	if err := addr.Validate(); err != nil { ... }
//	if addr.HasGeoCoordinate() { geo := addr.GeoCoordinate() }
package address

import (
	"fmt"
	"strings"

	"github.com/shredbx/sbx-core/pkg/geocoordinate"
)

// Address represents a postal/physical address with optional GPS coordinates.
// It is an immutable value object — create new instances rather than mutating.
type Address struct {
	Street      string  `json:"street" yaml:"street"`
	Unit        string  `json:"unit,omitempty" yaml:"unit,omitempty"`
	SubDistrict string  `json:"sub_district,omitempty" yaml:"sub_district,omitempty"`
	City        string  `json:"city" yaml:"city"`
	Province    string  `json:"province,omitempty" yaml:"province,omitempty"`
	PostalCode  string  `json:"postal_code,omitempty" yaml:"postal_code,omitempty"`
	Country     string  `json:"country" yaml:"country"`
	Latitude    float64 `json:"latitude,omitempty" yaml:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty" yaml:"longitude,omitempty"`
	// Polygon — optional parcel boundary in GeoJSON Polygon format.
	// Zero-value (Type == "", no coordinates) means "no boundary drawn".
	// Stored as JSONB at the consumer's discretion; pkg/address only
	// describes the wire shape. Pointer-omitempty doesn't work for
	// embedded structs in Go, so the JSON output carries `polygon: null`
	// when empty via the custom marshaller on GeoJSONPolygon.
	Polygon GeoJSONPolygon `json:"polygon" yaml:"polygon,omitempty"`
	// LocationMapView — persisted UI viewport for the location/pin picker
	// (LocationPicker). Stored in property_locations.location_map_view JSONB.
	// Independent of RegionMapView so the two pickers don't cross-contaminate
	// each other's camera state. Same null-on-empty convention as Polygon.
	LocationMapView MapView `json:"location_map_view" yaml:"location_map_view,omitempty"`
	// RegionMapView — persisted UI viewport for the region/polygon picker
	// (PolygonPicker). Stored in property_locations.region_map_view JSONB.
	// "Region" because the polygon defines the parcel's bounded region;
	// surface vocabulary used by BR ("region map") rather than "polygon view".
	RegionMapView MapView `json:"region_map_view" yaml:"region_map_view,omitempty"`
}

// NewAddress creates an Address with the required street, city, and country.
func NewAddress(street, city, country string) Address {
	return Address{Street: street, City: city, Country: country}
}

// Validate checks that the address has valid required fields and optional coordinate bounds.
func (a Address) Validate() error {
	if strings.TrimSpace(a.Street) == "" {
		return fmt.Errorf("street must not be empty")
	}
	if strings.TrimSpace(a.City) == "" {
		return fmt.Errorf("city must not be empty")
	}
	if strings.TrimSpace(a.Country) == "" {
		return fmt.Errorf("country must not be empty")
	}
	if len(a.Country) != 2 {
		return fmt.Errorf("country must be a 2-letter ISO 3166-1 alpha-2 code, got %q", a.Country)
	}
	if a.HasGeoCoordinate() {
		geo := a.GeoCoordinate()
		if err := geo.Validate(); err != nil {
			return fmt.Errorf("address geo: %w", err)
		}
	}
	if !a.Polygon.IsZero() {
		if err := a.Polygon.Validate(); err != nil {
			return fmt.Errorf("address polygon: %w", err)
		}
	}
	return nil
}

// IsZero returns true if all address fields are at zero/empty values.
func (a Address) IsZero() bool {
	if !a.Polygon.IsZero() {
		return false
	}
	if !a.LocationMapView.IsZero() {
		return false
	}
	if !a.RegionMapView.IsZero() {
		return false
	}
	return a.Street == "" && a.Unit == "" && a.SubDistrict == "" && a.City == "" &&
		a.Province == "" && a.PostalCode == "" && a.Country == "" &&
		a.Latitude == 0 && a.Longitude == 0
}

// HasGeoCoordinate returns true if the address has non-zero latitude or longitude.
func (a Address) HasGeoCoordinate() bool {
	return a.Latitude != 0 || a.Longitude != 0
}

// GeoCoordinate returns the address coordinates as a GeoCoordinate value object.
func (a Address) GeoCoordinate() geocoordinate.GeoCoordinate {
	return geocoordinate.NewGeoCoordinate(a.Latitude, a.Longitude)
}

// ValidateCoordinatePair enforces that latitude and longitude are either both
// zero (coords not provided) or both non-zero (a real WGS84 point). A half-set
// pair — e.g. (9.7489, 0) — is the silent corruption that creates phantom pins
// on the Gulf-of-Guinea Null Island. When both are set, also validates WGS84
// bounds via the embedded GeoCoordinate.
//
// Called separately from Validate() so the service layer can enforce the pair
// invariant even on otherwise-skeletal payloads (e.g. PATCH updates where the
// rest of the address is intentionally absent).
func (a Address) ValidateCoordinatePair() error {
	latSet := a.Latitude != 0
	lngSet := a.Longitude != 0
	if latSet != lngSet {
		return fmt.Errorf("coordinates must be both set or both zero (got lat=%g, lng=%g)", a.Latitude, a.Longitude)
	}
	if latSet {
		if err := a.GeoCoordinate().Validate(); err != nil {
			return fmt.Errorf("address geo: %w", err)
		}
	}
	return nil
}

// ValidateGeometry enforces ONLY the never-correct-to-skip geometry sub-invariants
// — coordinate pair (both-or-neither + WGS84 bounds) and polygon shape — WITHOUT
// requiring postal completeness (street / city / country). It is the reusable,
// lenient counterpart to the strict Validate(), for consumers that legitimately
// hold partial or informal address data: an address book (pkg/contact) and draft
// properties (pkg/property). MapView viewport state is cosmetic and intentionally
// not validated — bad zoom/tilt degrades gracefully at the map SDK layer.
func (a Address) ValidateGeometry() error {
	if err := a.ValidateCoordinatePair(); err != nil {
		return err
	}
	if !a.Polygon.IsZero() {
		if err := a.Polygon.Validate(); err != nil {
			return fmt.Errorf("polygon: %w", err)
		}
	}
	return nil
}
