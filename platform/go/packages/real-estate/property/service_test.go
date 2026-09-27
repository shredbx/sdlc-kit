package property_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shredbx/sbx-core/pkg/address"
	"github.com/shredbx/sbx-core/pkg/money"
	"github.com/shredbx/sbx-core/pkg/property"
)

func ptr[T any](v T) *T { return &v }

// SC-01: Create property with valid input
func TestPropertyService_Create_Valid(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")

	input := property.Property{
		Title:         ptr("Luxury Pool Villa in Phuket"),
		Text:          ptr("Beautiful 3-bedroom pool villa in Rawai with stunning sea views"),
		ForSale:       true,
		ForLease:      false,
		PropertyType:  ptr(property.PropertyPoolVilla),
		TitleDeed:     ptr(property.TitleChanote),
		SalePrice:     ptr(int64(1500000000)),
		PriceCurrency: ptr(money.CurrencyCode("THB")),
		CoverImageID:  ptr("11111111-1111-1111-1111-111111111111"),
		IsPublished:   false,
		Address: property.Address{
			Street:      "88/12 Moo 6",
			SubDistrict: "Rawai",
			City:        "Muang",
			Province:    "Phuket",
			PostalCode:  "83130",
			Country:     "TH",
			Latitude:    7.7710,
			Longitude:   98.3397,
		},
		Sizes: &property.Sizes{
			LandSize:   ptr(800.0),
			HouseSize:  ptr(350.0),
			LivingSize: ptr(280.0),
		},
		Rooms: &property.Rooms{
			Bedrooms:    ptr(3),
			Bathrooms:   ptr(4),
			Kitchens:    ptr(1),
			LivingRooms: ptr(2),
		},
		Furnished: ptr(property.FurnishedFully),
		Amenities: []property.Amenity{
			{Code: "private-pool", Group: property.AmenityGroupExterior},
			{Code: "garden", Group: property.AmenityGroupExterior},
			{Code: "covered-parking", Group: property.AmenityGroupBuilding},
			{Code: "24h-security", Group: property.AmenityGroupBuilding},
		},
	}

	result, err := svc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create valid property: unexpected error: %v", err)
	}
	if result.ID == "" {
		t.Error("Create valid property: expected non-empty ID")
	}
	if result.IsPublished {
		t.Error("Create valid property: expected is_published=false")
	}
	if result.Title == nil || *result.Title != "Luxury Pool Villa in Phuket" {
		t.Errorf("Create valid property: title mismatch: got %v", result.Title)
	}
}

// SC-02: Update property with valid changes
func TestPropertyService_Update_Valid(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")

	update := property.Property{
		Title:     ptr("Updated Pool Villa — Price Reduced"),
		ForSale:   true,
		SalePrice: ptr(int64(1200000000)),
		Furnished: ptr(property.FurnishedFully),
		Amenities: []property.Amenity{
			{Code: "private-pool", Group: property.AmenityGroupExterior},
			{Code: "garden", Group: property.AmenityGroupExterior},
		},
	}

	result, err := svc.Update(context.Background(), "test-id", update)
	if err != nil {
		t.Fatalf("Update valid property: unexpected error: %v", err)
	}
	if result.Title == nil || *result.Title != "Updated Pool Villa — Price Reduced" {
		t.Errorf("Update: title mismatch: got %v", result.Title)
	}
}

// SC-03: Publish property (set is_published=true)
func TestPropertyService_SetPublished_True(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")

	err := svc.SetPublished(context.Background(), "test-id", true)
	if err != nil {
		t.Fatalf("SetPublished(true): unexpected error: %v", err)
	}
}

// SC-15: Create with empty title → validation error
func TestPropertyService_Create_EmptyTitle(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")

	input := property.Property{
		Title:         ptr(""),
		ForSale:       true,
		ForLease:      false,
		PropertyType:  ptr(property.PropertyVilla),
		SalePrice:     ptr(int64(100000000)),
		PriceCurrency: ptr(money.CurrencyCode("THB")),
	}

	_, err := svc.Create(context.Background(), input)
	if err == nil {
		t.Fatal("Create with empty title: expected error, got nil")
	}
	if !errors.Is(err, property.ErrValidation) {
		t.Errorf("Create with empty title: expected ErrValidation, got %v", err)
	}
}

// SC-FIX5: Create with empty furnished string → normalized to nil
func TestPropertyService_Create_EmptyFurnished_Normalized(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")

	input := property.Property{
		Title:     ptr("Test Property"),
		ForSale:   true,
		Furnished: ptr(property.Furnished("")),
	}

	result, err := svc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Furnished != nil {
		t.Errorf("expected furnished=nil after normalization, got %q", *result.Furnished)
	}
}

// SC-16: Create with negative price → validation error
func TestPropertyService_Create_NegativePrice(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")

	input := property.Property{
		Title:         ptr("Test Property"),
		ForSale:       true,
		ForLease:      false,
		PropertyType:  ptr(property.PropertyVilla),
		SalePrice:     ptr(int64(-100)),
		PriceCurrency: ptr(money.CurrencyCode("THB")),
	}

	_, err := svc.Create(context.Background(), input)
	if err == nil {
		t.Fatal("Create with negative price: expected error, got nil")
	}
	if !errors.Is(err, property.ErrValidation) {
		t.Errorf("Create with negative price: expected ErrValidation, got %v", err)
	}
}

// 2605-145: half-set lat (lat=9.74, lng=0) → ErrValidation. Closes audit V1+S1.
func TestPropertyService_Create_HalfSetCoords_Rejected(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")

	input := property.Property{
		Title:   ptr("Test Property"),
		ForSale: true,
		Address: property.Address{Latitude: 9.7489, Longitude: 0},
	}

	_, err := svc.Create(context.Background(), input)
	if err == nil {
		t.Fatal("Create with half-set lat: expected error, got nil")
	}
	if !errors.Is(err, property.ErrValidation) {
		t.Errorf("Create with half-set lat: expected ErrValidation, got %v", err)
	}
}

// 2605-145: malformed polygon → ErrValidation. Closes audit S2 (Go-side).
func TestPropertyService_Create_InvalidPolygon_Rejected(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")

	input := property.Property{
		Title:   ptr("Test Property"),
		ForSale: true,
		Address: property.Address{
			Polygon: address.GeoJSONPolygon{
				Type:        "Point", // wrong type — should be "Polygon"
				Coordinates: [][][2]float64{{{100, 9}, {101, 9}, {101, 10}, {100, 9}}},
			},
		},
	}

	_, err := svc.Create(context.Background(), input)
	if err == nil {
		t.Fatal("Create with non-Polygon type: expected error, got nil")
	}
	if !errors.Is(err, property.ErrValidation) {
		t.Errorf("Create with non-Polygon: expected ErrValidation, got %v", err)
	}
}

// 2605-145: Update path also rejects half-set coords (parallel to Create).
func TestPropertyService_Update_HalfSetCoords_Rejected(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")

	input := property.Property{
		ID:      "11111111-1111-1111-1111-111111111111",
		Title:   ptr("Test Property"),
		ForSale: true,
		Address: property.Address{Latitude: 0, Longitude: 100.031},
	}

	_, err := svc.Update(context.Background(), input.ID, input)
	if err == nil {
		t.Fatal("Update with half-set lng: expected error, got nil")
	}
	if !errors.Is(err, property.ErrValidation) {
		t.Errorf("Update with half-set lng: expected ErrValidation, got %v", err)
	}
}
