// Package geocoordinate provides a reusable embedded type for GPS coordinates
// following the WGS84 standard.
//
// GeoCoordinate is a value object with embedded reference semantics — consuming
// entities gain typed columns ({attr}_latitude, {attr}_longitude) via 2-column
// expansion, following the same pattern as the Money type.
//
// Example:
//
//	kp := geocoordinate.NewGeoCoordinate(9.7489, 100.0310) // Koh Phangan
//	if err := kp.Validate(); err != nil { ... }
//	dist := kp.DistanceTo(bangkok) // ~455 km
package geocoordinate

import (
	"fmt"
	"math"
)

// GeoCoordinate represents a WGS84 GPS position as a latitude/longitude pair
// in decimal degrees. It is an immutable value object — create new instances
// rather than mutating existing ones.
type GeoCoordinate struct {
	Latitude  float64 `json:"latitude" yaml:"latitude"`
	Longitude float64 `json:"longitude" yaml:"longitude"`
}

// NewGeoCoordinate creates a GeoCoordinate with the given latitude and longitude.
func NewGeoCoordinate(lat, lng float64) GeoCoordinate {
	return GeoCoordinate{Latitude: lat, Longitude: lng}
}

// Validate checks that the coordinate is within WGS84 bounds.
// Latitude must be in [-90, 90], longitude in [-180, 180].
func (g GeoCoordinate) Validate() error {
	if g.Latitude < -90 || g.Latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90, got %f", g.Latitude)
	}
	if g.Longitude < -180 || g.Longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180, got %f", g.Longitude)
	}
	return nil
}

// IsZero returns true if both latitude and longitude are zero.
func (g GeoCoordinate) IsZero() bool {
	return g.Latitude == 0 && g.Longitude == 0
}

// earthRadiusKm is the mean radius of Earth in kilometers.
const earthRadiusKm = 6371.0

// DistanceTo returns the Haversine distance in kilometers between two coordinates.
func (g GeoCoordinate) DistanceTo(other GeoCoordinate) float64 {
	lat1 := g.Latitude * math.Pi / 180
	lat2 := other.Latitude * math.Pi / 180
	dLat := (other.Latitude - g.Latitude) * math.Pi / 180
	dLng := (other.Longitude - g.Longitude) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}
