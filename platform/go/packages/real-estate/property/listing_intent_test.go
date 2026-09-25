package property_test

import (
	"errors"
	"testing"

	"github.com/shredbx/sbx-core/pkg/property"
)

func i64(v int64) *int64 { return &v }

func TestProperty_ForSale_Only(t *testing.T) {
	p := property.Property{ForSale: true, ForLease: false, SalePrice: i64(15000000)}
	if err := p.ValidateListingIntent(); err != nil {
		t.Fatalf("for-sale-only is valid, got error: %v", err)
	}
	if p.LeasePrice != nil {
		t.Errorf("expected LeasePrice nil for for-sale-only listing, got %v", *p.LeasePrice)
	}
}

func TestProperty_ForLease_Only(t *testing.T) {
	p := property.Property{ForSale: false, ForLease: true, LeasePrice: i64(45000)}
	if err := p.ValidateListingIntent(); err != nil {
		t.Fatalf("for-lease-only is valid, got error: %v", err)
	}
	if p.SalePrice != nil {
		t.Errorf("expected SalePrice nil for for-lease-only listing, got %v", *p.SalePrice)
	}
}

func TestProperty_DualMode(t *testing.T) {
	p := property.Property{
		ForSale:    true,
		ForLease:   true,
		SalePrice:  i64(12000000),
		LeasePrice: i64(30000),
	}
	if err := p.ValidateListingIntent(); err != nil {
		t.Fatalf("dual-mode is valid, got error: %v", err)
	}
}

func TestProperty_POA_NilPrice(t *testing.T) {
	p := property.Property{ForSale: true, ForLease: false}
	if err := p.ValidateListingIntent(); err != nil {
		t.Fatalf("POA (for-sale, nil price) is valid, got error: %v", err)
	}
	if p.SalePrice != nil {
		t.Errorf("expected SalePrice nil for POA listing, got %v", *p.SalePrice)
	}
}

func TestProperty_RejectNoListingIntent(t *testing.T) {
	p := property.Property{ForSale: false, ForLease: false}
	err := p.ValidateListingIntent()
	if err == nil {
		t.Fatal("expected error when both flags are false, got nil")
	}
	if !errors.Is(err, property.ErrNoListingIntent) {
		t.Errorf("want ErrNoListingIntent, got %v", err)
	}
}

func TestProperty_RejectNegativePrices(t *testing.T) {
	p := property.Property{ForSale: true, SalePrice: i64(-1)}
	err := p.ValidatePrices()
	if err == nil {
		t.Fatal("expected error for negative sale_price, got nil")
	}
	if !errors.Is(err, property.ErrNegativePrice) {
		t.Errorf("want ErrNegativePrice, got %v", err)
	}
}
