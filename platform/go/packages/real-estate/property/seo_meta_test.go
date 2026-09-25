package property_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/property"
	"github.com/shredbx/sbx-core/pkg/seo"
)

// TestProperty_SeoMeta_JSONWireShape pins the API contract the frontend SeoHead
// renderer depends on: a Property with a populated SeoMeta serializes the field
// under the snake_case key "seo_meta" carrying the seven optional sub-fields, and
// a Property whose SeoMeta is nil OMITS the key entirely (the pointer + omitempty
// gives the consumer a clean "absent" rather than a literal null object on every
// listing).
func TestProperty_SeoMeta_JSONWireShape(t *testing.T) {
	p := property.Property{
		ID:      "p1",
		ForSale: true,
		SeoMeta: &seo.SeoMeta{MetaTitle: "Villa", MetaKeywords: []string{"villa", "sea"}},
	}

	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, `"seo_meta":{`) {
		t.Errorf(`expected "seo_meta" object in JSON, got: %s`, s)
	}
	if !strings.Contains(s, `"meta_title":"Villa"`) {
		t.Errorf(`expected meta_title in seo_meta, got: %s`, s)
	}
	if !strings.Contains(s, `"meta_keywords":["villa","sea"]`) {
		t.Errorf(`expected meta_keywords array in seo_meta, got: %s`, s)
	}

	// Round-trip back into a Property (the manage create/update decode path).
	var back property.Property
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.SeoMeta == nil || back.SeoMeta.MetaTitle != "Villa" {
		t.Errorf("seo_meta did not round-trip: %+v", back.SeoMeta)
	}
}

// TestProperty_SeoMeta_NilOmitted proves a property without SEO overrides emits no
// seo_meta key (omitempty on the *seo.SeoMeta pointer) — the contract's `| null`
// is represented as field-absent, never a noisy empty object per listing.
func TestProperty_SeoMeta_NilOmitted(t *testing.T) {
	p := property.Property{ID: "p1", ForLease: true}
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), "seo_meta") {
		t.Errorf("nil SeoMeta should omit the seo_meta key, got: %s", out)
	}
}
