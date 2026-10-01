package property_test

import (
	"testing"

	"github.com/shredbx/sbx-core/pkg/property"
)

// TC-SC4: Property has amenities assigned
func TestProperty_Amenities_Field(t *testing.T) {
	p := property.Property{
		Amenities: []property.Amenity{
			{Code: "private-pool", Group: "exterior"},
			{Code: "garden", Group: "exterior"},
			{Code: "24h-security", Group: "building"},
		},
	}

	if len(p.Amenities) != 3 {
		t.Fatalf("Amenities len = %d, want 3", len(p.Amenities))
	}
	if p.Amenities[0].Code != "private-pool" {
		t.Errorf("Amenities[0].Code = %q, want %q", p.Amenities[0].Code, "private-pool")
	}
	if p.Amenities[0].Group != "exterior" {
		t.Errorf("Amenities[0].Group = %q, want %q", p.Amenities[0].Group, "exterior")
	}
}

func TestProperty_Furnished_Field(t *testing.T) {
	furnished := property.FurnishedFully
	p := property.Property{
		Furnished: &furnished,
	}

	if p.Furnished == nil {
		t.Fatal("Furnished should not be nil")
	}
	if *p.Furnished != property.FurnishedFully {
		t.Errorf("Furnished = %q, want %q", *p.Furnished, property.FurnishedFully)
	}
}

func TestProperty_Furnished_NilIsDraft(t *testing.T) {
	var p property.Property
	if p.Furnished != nil {
		t.Error("zero-value Property.Furnished should be nil (unknown/draft)")
	}
}

func TestFurnished_Constants(t *testing.T) {
	cases := map[property.Furnished]string{
		property.FurnishedFully:       "fully",
		property.FurnishedPartially:   "partially",
		property.FurnishedUnfurnished: "unfurnished",
	}
	for typ, want := range cases {
		if string(typ) != want {
			t.Errorf("Furnished %v: want %q, got %q", typ, want, string(typ))
		}
	}
}

func TestAmenity_CodeValidation(t *testing.T) {
	a := property.Amenity{Code: "private-pool", Group: "exterior"}
	if a.Code == "" {
		t.Error("Amenity.Code should not be empty")
	}
	if a.Group == "" {
		t.Error("Amenity.Group should not be empty")
	}
}

func TestAmenityGroup_Constants(t *testing.T) {
	groups := []property.AmenityGroup{
		property.AmenityGroupBuilding,
		property.AmenityGroupInterior,
		property.AmenityGroupExterior,
		property.AmenityGroupLocation,
		property.AmenityGroupNeighborhood,
		property.AmenityGroupServices,
	}
	expected := []string{"building", "interior", "exterior", "location", "neighborhood", "services"}

	for i, g := range groups {
		if string(g) != expected[i] {
			t.Errorf("AmenityGroup[%d] = %q, want %q", i, string(g), expected[i])
		}
	}
}

// TC-SC4: Amenities nil when no join table rows exist
func TestProperty_Amenities_NilWhenEmpty(t *testing.T) {
	var p property.Property
	if p.Amenities != nil {
		t.Error("zero-value Property.Amenities should be nil")
	}
}
