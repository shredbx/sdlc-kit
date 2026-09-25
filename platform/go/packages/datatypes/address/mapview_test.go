package address

import (
	"encoding/json"
	"testing"
)

func TestMapView_IsZero(t *testing.T) {
	if !(MapView{}).IsZero() {
		t.Error("zero MapView should be IsZero")
	}
	if (MapView{Zoom: 10}).IsZero() {
		t.Error("MapView with Zoom should NOT be IsZero")
	}
	if (MapView{MapType: "satellite"}).IsZero() {
		t.Error("MapView with MapType should NOT be IsZero")
	}
	c := [2]float64{100.031, 9.7489}
	if (MapView{Center: &c}).IsZero() {
		t.Error("MapView with Center should NOT be IsZero")
	}
}

func TestMapView_Value_ZeroReturnsNil(t *testing.T) {
	v, err := (MapView{}).Value()
	if err != nil {
		t.Fatalf("Value error: %v", err)
	}
	if v != nil {
		t.Errorf("zero MapView.Value should return nil, got %v", v)
	}
}

func TestMapView_Value_NonZeroReturnsJSON(t *testing.T) {
	c := [2]float64{100.031, 9.7489}
	mv := MapView{Zoom: 17.5, Center: &c, MapType: "satellite", Heading: 90, Tilt: 30}
	v, err := mv.Value()
	if err != nil {
		t.Fatalf("Value error: %v", err)
	}
	bytes, ok := v.([]byte)
	if !ok {
		t.Fatalf("Value should return []byte, got %T", v)
	}
	var out MapView
	if err := json.Unmarshal(bytes, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Zoom != 17.5 {
		t.Errorf("Zoom: want 17.5, got %v", out.Zoom)
	}
	if out.MapType != "satellite" {
		t.Errorf("MapType: want satellite, got %q", out.MapType)
	}
	if out.Heading != 90 {
		t.Errorf("Heading: want 90, got %v", out.Heading)
	}
	if out.Center == nil || (*out.Center)[0] != 100.031 || (*out.Center)[1] != 9.7489 {
		t.Errorf("Center: want [100.031, 9.7489], got %v", out.Center)
	}
}

func TestMapView_Scan_RoundTrip(t *testing.T) {
	c := [2]float64{100.031, 9.7489}
	original := MapView{Zoom: 17.5, Center: &c, MapType: "satellite", Heading: 90, Tilt: 30}
	v, err := original.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	var rescanned MapView
	if err := rescanned.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if rescanned.Zoom != original.Zoom ||
		rescanned.MapType != original.MapType ||
		rescanned.Heading != original.Heading ||
		rescanned.Tilt != original.Tilt {
		t.Errorf("round-trip mismatch: got %+v, want %+v", rescanned, original)
	}
	if rescanned.Center == nil || (*rescanned.Center) != c {
		t.Errorf("Center round-trip: got %v, want %v", rescanned.Center, c)
	}
}

func TestMapView_Scan_Nil(t *testing.T) {
	var mv MapView
	if err := mv.Scan(nil); err != nil {
		t.Fatalf("Scan nil: %v", err)
	}
	if !mv.IsZero() {
		t.Errorf("Scan(nil) should produce zero MapView, got %+v", mv)
	}
}

func TestMapView_Scan_AcceptsString(t *testing.T) {
	var mv MapView
	if err := mv.Scan(`{"zoom":12,"map_type":"roadmap"}`); err != nil {
		t.Fatalf("Scan string: %v", err)
	}
	if mv.Zoom != 12 || mv.MapType != "roadmap" {
		t.Errorf("Scan from string mismatch: %+v", mv)
	}
}

func TestMapView_MarshalJSON_ZeroReturnsNull(t *testing.T) {
	b, err := json.Marshal(MapView{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(b) != "null" {
		t.Errorf("zero MapView should marshal as null, got %s", b)
	}
}

func TestMapView_UnmarshalJSON_NullProducesZero(t *testing.T) {
	var mv MapView
	if err := json.Unmarshal([]byte("null"), &mv); err != nil {
		t.Fatalf("Unmarshal null: %v", err)
	}
	if !mv.IsZero() {
		t.Errorf("Unmarshal(null) should be zero, got %+v", mv)
	}
}
