package geocoordinate

import (
	"math"
	"testing"
)

func TestNewGeoCoordinate(t *testing.T) {
	t.Run("creates valid coordinate", func(t *testing.T) {
		g := NewGeoCoordinate(9.7489, 100.0310)
		if g.Latitude != 9.7489 || g.Longitude != 100.0310 {
			t.Errorf("expected {9.7489, 100.0310}, got {%f, %f}", g.Latitude, g.Longitude)
		}
	})

	t.Run("creates zero coordinate", func(t *testing.T) {
		g := NewGeoCoordinate(0, 0)
		if g.Latitude != 0 || g.Longitude != 0 {
			t.Errorf("expected {0, 0}, got {%f, %f}", g.Latitude, g.Longitude)
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		lat     float64
		lng     float64
		wantErr bool
	}{
		{"valid Koh Phangan", 9.7489, 100.0310, false},
		{"valid max bounds", 90.0, 180.0, false},
		{"valid min bounds", -90.0, -180.0, false},
		{"invalid latitude high", 91.0, 0.0, true},
		{"invalid latitude low", -91.0, 0.0, true},
		{"invalid longitude high", 0.0, 181.0, true},
		{"invalid longitude low", 0.0, -181.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGeoCoordinate(tt.lat, tt.lng)
			err := g.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsZero(t *testing.T) {
	t.Run("zero coordinate", func(t *testing.T) {
		g := NewGeoCoordinate(0, 0)
		if !g.IsZero() {
			t.Error("expected IsZero() = true for {0, 0}")
		}
	})

	t.Run("non-zero coordinate", func(t *testing.T) {
		g := NewGeoCoordinate(9.7489, 100.0310)
		if g.IsZero() {
			t.Error("expected IsZero() = false for Koh Phangan")
		}
	})
}

func TestDistanceTo(t *testing.T) {
	t.Run("Koh Phangan to Bangkok ~460km", func(t *testing.T) {
		kp := NewGeoCoordinate(9.7489, 100.0310)
		bk := NewGeoCoordinate(13.7563, 100.5018)
		d := kp.DistanceTo(bk)
		// Haversine distance should be approximately 445-465 km
		if d < 440 || d > 470 {
			t.Errorf("expected ~455 km, got %f km", d)
		}
	})

	t.Run("same point returns 0", func(t *testing.T) {
		g := NewGeoCoordinate(9.7489, 100.0310)
		d := g.DistanceTo(g)
		if math.Abs(d) > 0.001 {
			t.Errorf("expected 0, got %f", d)
		}
	})
}
