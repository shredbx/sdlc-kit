package address

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// GeoJSONPolygon is the wire format for a parcel boundary attached to an
// Address. Matches the GeoJSON RFC 7946 Polygon geometry exactly so it
// round-trips through JSONB columns and JSON APIs without translation.
//
// Coordinate ordering is [longitude, latitude] per GeoJSON spec — NOT
// [lat, lng]. UI-layer types that use {lat, lng} object literals must
// swap at the boundary.
//
// Holes (inner rings) are part of the GeoJSON spec but we accept and
// preserve them transparently. The first ring is the outer boundary;
// any additional rings are holes. Most real-estate parcels are
// simply-connected and will only carry one ring.
type GeoJSONPolygon struct {
	Type        string         `json:"type"`
	Coordinates [][][2]float64 `json:"coordinates"`
}

// NewPolygon constructs a polygon from a single outer ring. The ring is
// auto-closed if not already (appends the first point to the end). Holes
// are not supported by this constructor — pass a fully-formed
// GeoJSONPolygon if you need them.
func NewPolygon(outerRing [][2]float64) GeoJSONPolygon {
	if len(outerRing) == 0 {
		return GeoJSONPolygon{Type: "Polygon", Coordinates: nil}
	}
	first := outerRing[0]
	last := outerRing[len(outerRing)-1]
	ring := make([][2]float64, len(outerRing))
	copy(ring, outerRing)
	if first[0] != last[0] || first[1] != last[1] {
		ring = append(ring, [2]float64{first[0], first[1]})
	}
	return GeoJSONPolygon{Type: "Polygon", Coordinates: [][][2]float64{ring}}
}

// Validate enforces RFC 7946 structural rules + WGS84 coordinate bounds.
//   - type must be "Polygon"
//   - at least one ring required
//   - each ring must have at least 4 positions (3 unique vertices + closing copy)
//   - first and last positions of each ring must be equal (closed)
//   - each coordinate must have valid WGS84 lng/lat bounds
func (p *GeoJSONPolygon) Validate() error {
	if p == nil {
		return nil // nil polygon is allowed — represents "not set"
	}
	if p.Type != "Polygon" {
		return fmt.Errorf("polygon type must be %q, got %q", "Polygon", p.Type)
	}
	if len(p.Coordinates) == 0 {
		return errors.New("polygon must have at least one ring")
	}
	for i, ring := range p.Coordinates {
		if len(ring) < 4 {
			return fmt.Errorf("polygon ring %d must have at least 4 positions, got %d", i, len(ring))
		}
		first := ring[0]
		last := ring[len(ring)-1]
		if first[0] != last[0] || first[1] != last[1] {
			return fmt.Errorf("polygon ring %d must be closed (first == last)", i)
		}
		for j, pos := range ring {
			lng, lat := pos[0], pos[1]
			if lng < -180 || lng > 180 {
				return fmt.Errorf("polygon ring %d position %d: longitude %g out of [-180, 180]", i, j, lng)
			}
			if lat < -90 || lat > 90 {
				return fmt.Errorf("polygon ring %d position %d: latitude %g out of [-90, 90]", i, j, lat)
			}
		}
	}
	return nil
}

// IsZero reports whether the polygon carries no usable geometry. Used by
// mappers and JSON encoders to elide the column / field when empty.
// Value receiver so callers can write `addr.Polygon.IsZero()` on the
// embedded value type without a pointer.
func (p GeoJSONPolygon) IsZero() bool {
	return len(p.Coordinates) == 0
}

// Value implements driver.Valuer so a GeoJSONPolygon round-trips
// through JSONB columns transparently. Returns nil for empty polygons
// (DB NULL).
func (p GeoJSONPolygon) Value() (driver.Value, error) {
	if p.IsZero() {
		return nil, nil
	}
	// driver.Value of type []byte is interpreted as a binary literal by
	// pgx for non-JSON columns; JSONB columns parse the bytes as JSON
	// directly. Returning a string would also work but []byte is the
	// canonical pgx path.
	return json.Marshal(p)
}

// MarshalJSON makes empty polygons serialize as JSON null so consumer
// APIs can detect "no boundary set" without inspecting struct internals.
// Non-empty polygons marshal as the standard GeoJSON object.
func (p GeoJSONPolygon) MarshalJSON() ([]byte, error) {
	if p.IsZero() {
		return []byte("null"), nil
	}
	// Use an alias type to avoid recursion through our custom marshaller.
	type alias GeoJSONPolygon
	return json.Marshal(alias(p))
}

// Scan implements sql.Scanner for *GeoJSONPolygon. Accepts either []byte
// (the usual pgx path) or string for safety. Pointer receiver because
// Scan must mutate the destination.
func (p *GeoJSONPolygon) Scan(src interface{}) error {
	if src == nil {
		*p = GeoJSONPolygon{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("GeoJSONPolygon.Scan: unsupported source type %T", src)
	}
	if len(data) == 0 {
		*p = GeoJSONPolygon{}
		return nil
	}
	return json.Unmarshal(data, p)
}

// UnmarshalJSON allows the JSON literal `null` to round-trip back to an
// empty polygon (the inverse of MarshalJSON's null shortcut).
func (p *GeoJSONPolygon) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*p = GeoJSONPolygon{}
		return nil
	}
	type alias GeoJSONPolygon
	var tmp alias
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*p = GeoJSONPolygon(tmp)
	return nil
}
