package address

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// MapView is the persisted UI viewport state for an address-bound map
// (location picker, polygon picker). Fields are passthrough — bad values
// (zoom=999, unknown map_type) degrade gracefully at the map SDK and are
// NOT validated here. Use Address.LocationMapView / Address.RegionMapView.
//
// Coordinate ordering on Center is [lng, lat] for parity with GeoJSON. UI
// layers that use {lat, lng} object literals must swap at the boundary.
type MapView struct {
	Zoom    float64     `json:"zoom,omitempty"`
	Center  *[2]float64 `json:"center,omitempty"` // [longitude, latitude]
	MapType string      `json:"map_type,omitempty"`
	Heading float64     `json:"heading,omitempty"`
	Tilt    float64     `json:"tilt,omitempty"`
}

// IsZero reports whether all fields are zero values — i.e. no view state
// has been persisted yet. The mapper returns nil to the JSONB column in
// that case so the row reads as NULL.
func (v MapView) IsZero() bool {
	return v.Zoom == 0 &&
		v.Center == nil &&
		v.MapType == "" &&
		v.Heading == 0 &&
		v.Tilt == 0
}

// Value implements driver.Valuer for JSONB columns. Mirrors the pgx
// pattern in GeoJSONPolygon: marshal to []byte, or nil for the zero value
// so the row's column reads as SQL NULL.
func (v MapView) Value() (driver.Value, error) {
	if v.IsZero() {
		return nil, nil
	}
	return json.Marshal(v)
}

// Scan implements sql.Scanner. Accepts []byte (the pgx default) or string
// for safety. nil source means the row had a NULL column — return the
// zero value, not an error.
func (v *MapView) Scan(src interface{}) error {
	if src == nil {
		*v = MapView{}
		return nil
	}
	var data []byte
	switch s := src.(type) {
	case []byte:
		data = s
	case string:
		data = []byte(s)
	default:
		return fmt.Errorf("MapView.Scan: unsupported source type %T", src)
	}
	if len(data) == 0 {
		*v = MapView{}
		return nil
	}
	return json.Unmarshal(data, v)
}

// MarshalJSON returns JSON null for the zero value so API consumers can
// detect "no view persisted" without inspecting fields. Mirrors the
// GeoJSONPolygon convention.
func (v MapView) MarshalJSON() ([]byte, error) {
	if v.IsZero() {
		return []byte("null"), nil
	}
	type alias MapView
	return json.Marshal(alias(v))
}

// UnmarshalJSON treats JSON null as the zero value (inverse of MarshalJSON).
func (v *MapView) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*v = MapView{}
		return nil
	}
	type alias MapView
	var tmp alias
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*v = MapView(tmp)
	return nil
}
