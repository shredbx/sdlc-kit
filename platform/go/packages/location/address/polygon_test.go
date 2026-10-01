package address

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewPolygon(t *testing.T) {
	t.Run("auto-closes open ring", func(t *testing.T) {
		p := NewPolygon([][2]float64{{100.0, 9.75}, {100.001, 9.75}, {100.001, 9.751}})
		if len(p.Coordinates) != 1 {
			t.Fatalf("expected 1 ring, got %d", len(p.Coordinates))
		}
		ring := p.Coordinates[0]
		if len(ring) != 4 {
			t.Errorf("expected closed ring with 4 positions, got %d", len(ring))
		}
		if ring[0] != ring[len(ring)-1] {
			t.Error("ring not closed")
		}
	})
	t.Run("preserves already-closed ring", func(t *testing.T) {
		p := NewPolygon([][2]float64{{100.0, 9.75}, {100.001, 9.75}, {100.001, 9.751}, {100.0, 9.75}})
		if len(p.Coordinates[0]) != 4 {
			t.Errorf("expected 4 positions, got %d", len(p.Coordinates[0]))
		}
	})
	t.Run("empty input returns empty polygon", func(t *testing.T) {
		p := NewPolygon(nil)
		if !p.IsZero() {
			t.Error("expected IsZero for empty input")
		}
	})
}

func TestGeoJSONPolygon_Validate(t *testing.T) {
	closed := [][2]float64{{100.0, 9.75}, {100.001, 9.75}, {100.001, 9.751}, {100.0, 9.75}}
	tests := []struct {
		name    string
		poly    *GeoJSONPolygon
		wantErr string
	}{
		{"nil polygon is valid", nil, ""},
		{
			"valid closed ring",
			&GeoJSONPolygon{Type: "Polygon", Coordinates: [][][2]float64{closed}},
			"",
		},
		{
			"wrong type",
			&GeoJSONPolygon{Type: "MultiPolygon", Coordinates: [][][2]float64{closed}},
			`type must be "Polygon"`,
		},
		{
			"no rings",
			&GeoJSONPolygon{Type: "Polygon", Coordinates: nil},
			"at least one ring",
		},
		{
			"ring too short (3 positions)",
			&GeoJSONPolygon{Type: "Polygon", Coordinates: [][][2]float64{{{100.0, 9.75}, {100.001, 9.75}, {100.0, 9.75}}}},
			"at least 4 positions",
		},
		{
			"ring not closed",
			&GeoJSONPolygon{Type: "Polygon", Coordinates: [][][2]float64{{{100.0, 9.75}, {100.001, 9.75}, {100.001, 9.751}, {100.001, 9.752}}}},
			"must be closed",
		},
		{
			"longitude out of bounds",
			&GeoJSONPolygon{Type: "Polygon", Coordinates: [][][2]float64{{{181.0, 9.75}, {100.0, 9.75}, {100.0, 9.751}, {181.0, 9.75}}}},
			"longitude",
		},
		{
			"latitude out of bounds",
			&GeoJSONPolygon{Type: "Polygon", Coordinates: [][][2]float64{{{100.0, 91.0}, {100.001, 9.75}, {100.001, 9.751}, {100.0, 91.0}}}},
			"latitude",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.poly.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestGeoJSONPolygon_IsZero(t *testing.T) {
	if (GeoJSONPolygon{}).IsZero() != true {
		t.Error("empty struct should be zero")
	}
	if (GeoJSONPolygon{Type: "Polygon"}).IsZero() != true {
		t.Error("polygon with no rings should be zero")
	}
	p := NewPolygon([][2]float64{{100.0, 9.75}, {100.001, 9.75}, {100.001, 9.751}})
	if p.IsZero() {
		t.Error("polygon with valid ring should not be zero")
	}
}

func TestGeoJSONPolygon_RoundTripJSON(t *testing.T) {
	original := NewPolygon([][2]float64{
		{100.0, 9.75},
		{100.001, 9.75},
		{100.001, 9.751},
	})
	data, err := json.Marshal(&original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded GeoJSONPolygon
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Type != "Polygon" {
		t.Errorf("expected type Polygon, got %q", decoded.Type)
	}
	if len(decoded.Coordinates) != 1 || len(decoded.Coordinates[0]) != 4 {
		t.Errorf("expected 1 ring with 4 positions, got %d / %d", len(decoded.Coordinates), len(decoded.Coordinates[0]))
	}
}

func TestGeoJSONPolygon_Value(t *testing.T) {
	t.Run("empty produces NULL", func(t *testing.T) {
		v, err := (GeoJSONPolygon{}).Value()
		if err != nil {
			t.Fatalf("Value: %v", err)
		}
		if v != nil {
			t.Errorf("expected nil for empty polygon, got %v", v)
		}
	})
	t.Run("non-empty produces JSON bytes", func(t *testing.T) {
		p := NewPolygon([][2]float64{{100.0, 9.75}, {100.001, 9.75}, {100.001, 9.751}})
		v, err := p.Value()
		if err != nil {
			t.Fatalf("Value: %v", err)
		}
		data, ok := v.([]byte)
		if !ok {
			t.Fatalf("expected []byte, got %T", v)
		}
		if !strings.Contains(string(data), `"type":"Polygon"`) {
			t.Errorf("expected JSON with type=Polygon, got %s", string(data))
		}
	})
}

func TestGeoJSONPolygon_MarshalJSON(t *testing.T) {
	t.Run("empty marshals to null", func(t *testing.T) {
		data, err := json.Marshal(GeoJSONPolygon{})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(data) != "null" {
			t.Errorf("expected `null`, got %s", string(data))
		}
	})
	t.Run("populated marshals to GeoJSON object", func(t *testing.T) {
		p := NewPolygon([][2]float64{{100.0, 9.75}, {100.001, 9.75}, {100.001, 9.751}})
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !strings.Contains(string(data), `"type":"Polygon"`) {
			t.Errorf("expected GeoJSON object, got %s", string(data))
		}
	})
}

func TestGeoJSONPolygon_UnmarshalJSON(t *testing.T) {
	t.Run("null becomes empty polygon", func(t *testing.T) {
		var p GeoJSONPolygon
		if err := json.Unmarshal([]byte("null"), &p); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !p.IsZero() {
			t.Errorf("expected empty polygon, got %+v", p)
		}
	})
	t.Run("standard GeoJSON decodes", func(t *testing.T) {
		data := []byte(`{"type":"Polygon","coordinates":[[[100,9.75],[100.001,9.75],[100.001,9.751],[100,9.75]]]}`)
		var p GeoJSONPolygon
		if err := json.Unmarshal(data, &p); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if p.Type != "Polygon" || len(p.Coordinates) != 1 {
			t.Errorf("unexpected: %+v", p)
		}
	})
}

func TestGeoJSONPolygon_Scan(t *testing.T) {
	jsonBytes := []byte(`{"type":"Polygon","coordinates":[[[100.0,9.75],[100.001,9.75],[100.001,9.751],[100.0,9.75]]]}`)
	t.Run("scans []byte", func(t *testing.T) {
		var p GeoJSONPolygon
		if err := p.Scan(jsonBytes); err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if p.Type != "Polygon" || len(p.Coordinates) != 1 {
			t.Errorf("unexpected scan result: %+v", p)
		}
	})
	t.Run("scans string", func(t *testing.T) {
		var p GeoJSONPolygon
		if err := p.Scan(string(jsonBytes)); err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if p.Type != "Polygon" {
			t.Errorf("unexpected scan result: %+v", p)
		}
	})
	t.Run("scans nil → zero polygon", func(t *testing.T) {
		var p GeoJSONPolygon
		if err := p.Scan(nil); err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if !p.IsZero() {
			t.Errorf("expected zero polygon, got %+v", p)
		}
	})
	t.Run("rejects unsupported type", func(t *testing.T) {
		var p GeoJSONPolygon
		err := p.Scan(123)
		if err == nil {
			t.Error("expected error for int source")
		}
	})
}

func TestAddress_PolygonInValidate(t *testing.T) {
	addr := NewAddress("123 Main St", "Bangkok", "TH")
	good := NewPolygon([][2]float64{{100.0, 9.75}, {100.001, 9.75}, {100.001, 9.751}})
	addr.Polygon = good
	if err := addr.Validate(); err != nil {
		t.Errorf("expected valid address with polygon, got %v", err)
	}

	bad := GeoJSONPolygon{
		Type:        "WrongType",
		Coordinates: [][][2]float64{{{100.0, 9.75}, {100.001, 9.75}, {100.001, 9.751}, {100.0, 9.75}}},
	}
	addr.Polygon = bad
	if err := addr.Validate(); err == nil {
		t.Error("expected polygon validation error")
	}
}

func TestAddress_IsZeroWithPolygon(t *testing.T) {
	addr := Address{}
	if !addr.IsZero() {
		t.Error("empty address should be zero")
	}
	addr.Polygon = NewPolygon([][2]float64{{100.0, 9.75}, {100.001, 9.75}, {100.001, 9.751}})
	if addr.IsZero() {
		t.Error("address with non-empty polygon should NOT be zero")
	}
}
