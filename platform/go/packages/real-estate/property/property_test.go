package property_test

import (
	"testing"

	"github.com/shredbx/sbx-core/pkg/property"
)

func TestPropertyType_TypoFix(t *testing.T) {
	// Supabase had 'appartment' typo — our constant must be 'apartment'
	if string(property.PropertyApartment) != "apartment" {
		t.Errorf("PropertyApartment: want 'apartment', got %q", string(property.PropertyApartment))
	}
}

func TestTitleDeed_PorBorTor5(t *testing.T) {
	// New title type from Supabase import data
	if string(property.TitlePorBorTor5) != "por-bor-tor-5" {
		t.Errorf("TitlePorBorTor5: want 'por-bor-tor-5', got %q", string(property.TitlePorBorTor5))
	}
}

func TestProperty_ZeroValue_IsUnpublished(t *testing.T) {
	var p property.Property
	if p.IsPublished {
		t.Error("zero-value Property must be unpublished")
	}
}

func TestProperty_NilContent_IsDraft(t *testing.T) {
	var p property.Property
	if p.Title != nil || p.PropertyType != nil {
		t.Error("zero-value Property content fields must be nil (draft)")
	}
	if p.ForSale || p.ForLease {
		t.Error("zero-value Property listing flags must be false (draft)")
	}
	if p.SalePrice != nil || p.LeasePrice != nil {
		t.Error("zero-value Property prices must be nil (POA / not set)")
	}
}
