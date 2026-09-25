package address

import (
	"testing"
)

func TestNewAddress(t *testing.T) {
	t.Run("creates address with required fields", func(t *testing.T) {
		a := NewAddress("123 Main St", "Bangkok", "TH")
		if a.Street != "123 Main St" {
			t.Errorf("expected Street %q, got %q", "123 Main St", a.Street)
		}
		if a.City != "Bangkok" {
			t.Errorf("expected City %q, got %q", "Bangkok", a.City)
		}
		if a.Country != "TH" {
			t.Errorf("expected Country %q, got %q", "TH", a.Country)
		}
		if a.Unit != "" || a.Province != "" || a.PostalCode != "" {
			t.Error("expected optional fields to be empty")
		}
		if a.Latitude != 0 || a.Longitude != 0 {
			t.Error("expected coordinates to be zero")
		}
	})
}

func TestAddress_Validate(t *testing.T) {
	tests := []struct {
		name    string
		addr    Address
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid address without geo",
			addr:    NewAddress("123 Main St", "Bangkok", "TH"),
			wantErr: false,
		},
		{
			name: "valid address with geo",
			addr: Address{
				Street:    "123 Moo 4, Bantai",
				City:      "Koh Phangan",
				Country:   "TH",
				Latitude:  9.7489,
				Longitude: 100.0310,
			},
			wantErr: false,
		},
		{
			name: "valid address with all fields",
			addr: Address{
				Street:     "456 Sukhumvit Rd",
				Unit:       "Suite 1201",
				City:       "Bangkok",
				Province:   "Bangkok",
				PostalCode: "10110",
				Country:    "TH",
				Latitude:   13.7563,
				Longitude:  100.5018,
			},
			wantErr: false,
		},
		{
			name:    "empty street",
			addr:    NewAddress("", "Bangkok", "TH"),
			wantErr: true,
			errMsg:  "street must not be empty",
		},
		{
			name:    "whitespace-only street",
			addr:    NewAddress("   ", "Bangkok", "TH"),
			wantErr: true,
			errMsg:  "street must not be empty",
		},
		{
			name:    "empty city",
			addr:    NewAddress("123 Main St", "", "TH"),
			wantErr: true,
			errMsg:  "city must not be empty",
		},
		{
			name:    "empty country",
			addr:    NewAddress("123 Main St", "Bangkok", ""),
			wantErr: true,
			errMsg:  "country must not be empty",
		},
		{
			name:    "country not 2 chars - too long",
			addr:    NewAddress("123 Main St", "Bangkok", "THAI"),
			wantErr: true,
			errMsg:  "country must be a 2-letter ISO 3166-1 alpha-2 code",
		},
		{
			name:    "country not 2 chars - too short",
			addr:    NewAddress("123 Main St", "Bangkok", "T"),
			wantErr: true,
			errMsg:  "country must be a 2-letter ISO 3166-1 alpha-2 code",
		},
		{
			name: "invalid latitude in geo",
			addr: Address{
				Street:    "123 Main St",
				City:      "Bangkok",
				Country:   "TH",
				Latitude:  91.0,
				Longitude: 0,
			},
			wantErr: true,
			errMsg:  "address geo: latitude must be between -90 and 90",
		},
		{
			name: "invalid longitude in geo",
			addr: Address{
				Street:    "123 Main St",
				City:      "Bangkok",
				Country:   "TH",
				Latitude:  0.1,
				Longitude: 181.0,
			},
			wantErr: true,
			errMsg:  "address geo: longitude must be between -180 and 180",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.addr.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if got := err.Error(); len(tt.errMsg) > 0 && !contains(got, tt.errMsg) {
					t.Errorf("Validate() error = %q, want to contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestAddress_IsZero(t *testing.T) {
	t.Run("zero address", func(t *testing.T) {
		a := Address{}
		if !a.IsZero() {
			t.Error("expected IsZero() = true for empty Address{}")
		}
	})

	t.Run("non-zero address", func(t *testing.T) {
		a := NewAddress("123 Main St", "Bangkok", "TH")
		if a.IsZero() {
			t.Error("expected IsZero() = false for address with fields")
		}
	})

	t.Run("only coordinates set", func(t *testing.T) {
		a := Address{Latitude: 9.7489}
		if a.IsZero() {
			t.Error("expected IsZero() = false when latitude is set")
		}
	})

	t.Run("only sub_district set", func(t *testing.T) {
		a := Address{SubDistrict: "Bantai"}
		if a.IsZero() {
			t.Error("expected IsZero() = false when SubDistrict is set")
		}
	})
}

// TestAddress_SubDistrict verifies the 9th field (added 2026-05-20 for BR
// Phase B refactor — Thai ตำบล / sub-district / barangay level).
func TestAddress_SubDistrict(t *testing.T) {
	t.Run("optional field — empty by default", func(t *testing.T) {
		a := NewAddress("123 Main St", "Bangkok", "TH")
		if a.SubDistrict != "" {
			t.Errorf("expected SubDistrict empty by default, got %q", a.SubDistrict)
		}
		if err := a.Validate(); err != nil {
			t.Errorf("address without SubDistrict should validate, got %v", err)
		}
	})

	t.Run("accepts populated value", func(t *testing.T) {
		a := Address{
			Street:      "Moo 4 Bantai",
			SubDistrict: "Bantai",
			City:        "Koh Phangan",
			Country:     "TH",
		}
		if err := a.Validate(); err != nil {
			t.Errorf("address with SubDistrict should validate, got %v", err)
		}
		if a.SubDistrict != "Bantai" {
			t.Errorf("SubDistrict round-trip failed, got %q", a.SubDistrict)
		}
	})
}

func TestAddress_HasGeoCoordinate(t *testing.T) {
	t.Run("with both coordinates", func(t *testing.T) {
		a := Address{Latitude: 9.7489, Longitude: 100.0310}
		if !a.HasGeoCoordinate() {
			t.Error("expected HasGeoCoordinate() = true")
		}
	})

	t.Run("with only latitude", func(t *testing.T) {
		a := Address{Latitude: 9.7489}
		if !a.HasGeoCoordinate() {
			t.Error("expected HasGeoCoordinate() = true with only latitude")
		}
	})

	t.Run("with only longitude", func(t *testing.T) {
		a := Address{Longitude: 100.0310}
		if !a.HasGeoCoordinate() {
			t.Error("expected HasGeoCoordinate() = true with only longitude")
		}
	})

	t.Run("without coordinates", func(t *testing.T) {
		a := NewAddress("123 Main St", "Bangkok", "TH")
		if a.HasGeoCoordinate() {
			t.Error("expected HasGeoCoordinate() = false")
		}
	})
}

func TestAddress_GeoCoordinate(t *testing.T) {
	t.Run("returns correct GeoCoordinate", func(t *testing.T) {
		a := Address{
			Street:    "123 Moo 4",
			City:      "Koh Phangan",
			Country:   "TH",
			Latitude:  9.7489,
			Longitude: 100.0310,
		}
		geo := a.GeoCoordinate()
		if geo.Latitude != 9.7489 {
			t.Errorf("expected Latitude 9.7489, got %f", geo.Latitude)
		}
		if geo.Longitude != 100.0310 {
			t.Errorf("expected Longitude 100.0310, got %f", geo.Longitude)
		}
	})

	t.Run("zero address returns zero GeoCoordinate", func(t *testing.T) {
		a := NewAddress("123 Main St", "Bangkok", "TH")
		geo := a.GeoCoordinate()
		if !geo.IsZero() {
			t.Error("expected zero GeoCoordinate for address without coordinates")
		}
	})
}

func TestAddress_ValidateCoordinatePair(t *testing.T) {
	tests := []struct {
		name    string
		addr    Address
		wantErr bool
		errMsg  string
	}{
		{
			name:    "both zero — coords not set, valid pair",
			addr:    Address{Latitude: 0, Longitude: 0},
			wantErr: false,
		},
		{
			name:    "both set within WGS84 — valid pair",
			addr:    Address{Latitude: 9.7489, Longitude: 100.031},
			wantErr: false,
		},
		{
			name:    "lat set, lng zero — half-set pair",
			addr:    Address{Latitude: 9.7489, Longitude: 0},
			wantErr: true,
			errMsg:  "coordinates must be both set or both zero",
		},
		{
			name:    "lng set, lat zero — half-set pair",
			addr:    Address{Latitude: 0, Longitude: 100.031},
			wantErr: true,
			errMsg:  "coordinates must be both set or both zero",
		},
		{
			name:    "lat out of WGS84 bounds",
			addr:    Address{Latitude: 91.0, Longitude: 100.031},
			wantErr: true,
			errMsg:  "latitude",
		},
		{
			name:    "lng out of WGS84 bounds",
			addr:    Address{Latitude: 9.7489, Longitude: 181.0},
			wantErr: true,
			errMsg:  "longitude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.addr.ValidateCoordinatePair()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestAddress_ValidateGeometry covers the reusable geometry-only validator used by
// informal-data consumers (contacts) and by property drafts. It enforces ONLY the
// never-correct-to-skip invariants (coord-pair both-or-neither + polygon shape) and
// must NOT force postal completeness (street/city/country) the way Validate() does.
func TestAddress_ValidateGeometry(t *testing.T) {
	t.Run("partial address — country only, no street/city — passes", func(t *testing.T) {
		// strict Validate() would reject this ("street must not be empty");
		// geometry-only must allow it.
		if err := (Address{Country: "Thailand"}).ValidateGeometry(); err != nil {
			t.Errorf("partial address should pass geometry-only validation, got %v", err)
		}
	})
	t.Run("fully empty address passes", func(t *testing.T) {
		if err := (Address{}).ValidateGeometry(); err != nil {
			t.Errorf("zero address should pass, got %v", err)
		}
	})
	t.Run("valid full coordinate pair passes", func(t *testing.T) {
		if err := (Address{Latitude: 9.7489, Longitude: 100.031}).ValidateGeometry(); err != nil {
			t.Errorf("valid coordinates should pass, got %v", err)
		}
	})
	t.Run("half-set coordinate pair fails", func(t *testing.T) {
		if err := (Address{Latitude: 9.7489}).ValidateGeometry(); err == nil {
			t.Error("half-set coordinate pair should fail geometry validation")
		}
	})
	t.Run("malformed polygon fails", func(t *testing.T) {
		bad := Address{Polygon: GeoJSONPolygon{Type: "Polygon", Coordinates: [][][2]float64{{{0, 0}, {1, 1}}}}}
		if err := bad.ValidateGeometry(); err == nil {
			t.Error("polygon ring with <4 positions should fail geometry validation")
		}
	})
}

// contains checks if s contains substr (helper to avoid importing strings in test).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
