// section_faqlist_test.go exercises the `faqlist` section adapter (FDD4.PD.TEST_RED,
// SDLC 2607-001, test-plan case 5). FAQListSection is the closed-kind section that
// BORROWS library FAQ content onto a page (replaces the inline `faq` kind on /sell):
//
//	mode=category  → render all published items of one FAQ category (by slug)
//	mode=selection → render a curated, ordered subset (by faq_item IDs)
//
// It holds string REFERENCES only (slug / item IDs) — pkg/cms stays decoupled from
// pkg/faq (no import); the FAQ service resolves the refs server-side at render.
package cms

import (
	"errors"
	"testing"
)

// TestSectionKindFAQList_Registered confirms the adapter registered into the
// closed section-kind set (a SectionKind.Validate() pass requires registration).
func TestSectionKindFAQList_Registered(t *testing.T) {
	if err := SectionKindFAQList.Validate(); err != nil {
		t.Fatalf("SectionKindFAQList.Validate() err = %v, want nil (kind must be registered)", err)
	}
	if !SectionKindFAQList.Registered() {
		t.Fatal("SectionKindFAQList.Registered() = false, want true")
	}
}

// TestFAQListSection_Validate is test-plan case 5: a faqlist payload requires a
// valid mode AND its mode-specific reference (a category slug for mode=category,
// at least one item ID for mode=selection). Every failure wraps ErrValidation.
func TestFAQListSection_Validate(t *testing.T) {
	tests := []struct {
		name    string
		section FAQListSection
		wantErr bool
	}{
		{
			name:    "category mode valid",
			section: FAQListSection{Mode: FAQListModeCategory, Category: "buyers-guide"},
			wantErr: false,
		},
		{
			name:    "selection mode valid",
			section: FAQListSection{Mode: FAQListModeSelection, Items: []string{"item-1", "item-2"}},
			wantErr: false,
		},
		{name: "empty mode", section: FAQListSection{}, wantErr: true},
		{name: "unknown mode", section: FAQListSection{Mode: FAQListMode("all")}, wantErr: true},
		{
			name:    "category mode missing slug",
			section: FAQListSection{Mode: FAQListModeCategory},
			wantErr: true,
		},
		{
			name:    "category mode blank slug",
			section: FAQListSection{Mode: FAQListModeCategory, Category: "   "},
			wantErr: true,
		},
		{
			name:    "selection mode empty items",
			section: FAQListSection{Mode: FAQListModeSelection},
			wantErr: true,
		},
		{
			name:    "selection mode nil items",
			section: FAQListSection{Mode: FAQListModeSelection, Items: nil},
			wantErr: true,
		},
		{
			name:    "selection mode only blank ids",
			section: FAQListSection{Mode: FAQListModeSelection, Items: []string{"", "  "}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.section.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrValidation) {
				t.Fatalf("Validate() err = %v, want ErrValidation wrap", err)
			}
		})
	}
}
