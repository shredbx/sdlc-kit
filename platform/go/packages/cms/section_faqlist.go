package cms

// faqlist section adapter (Decision #0273 escape hatch; SDLC 2607-001) — a
// closed-kind section that BORROWS library FAQ content onto a page by reference,
// replacing the inline `faq` kind on borrowed pages (e.g. /sell):
//
//	mode=category  → render all published items of one FAQ category (by slug)
//	mode=selection → render a curated, ordered subset (by faq_item IDs)
//
// The payload holds STRING REFERENCES only (a category slug, or ordered faq_item
// IDs) — never pkg/faq types — so pkg/cms stays decoupled from pkg/faq (cms cannot
// import faq; faq is the consumer). The FAQ service resolves the refs server-side
// at render (no client N+1, NFR-013; a dangling/deleted ref is skipped, SC08).
//
// This is a new adapter file + one registerSectionKind call — ZERO change to the
// section core (project governance rule #9: a new variant is a new adapter). The
// inline `faq` kind (section_faq.go) stays registered through the transition.

import (
	"fmt"
	"strings"
)

func init() {
	registerSectionKind(SectionKindFAQList, func() SectionPayload { return &FAQListSection{} })
}

// SectionKindFAQList is the discriminator for a borrowed-from-library FAQ section.
const SectionKindFAQList SectionKind = "faqlist"

// FAQListMode is the faqlist resolution strategy: render a whole category vs a
// curated ordered selection. A named type — never a raw string — so the invariant
// travels with the value (mirrors SectionKind / EntryKind).
type FAQListMode string

const (
	// FAQListModeCategory renders every published item of one FAQ category, ordered
	// by step then item position.
	FAQListModeCategory FAQListMode = "category"
	// FAQListModeSelection renders a curated, ordered subset of items (the
	// /sell "pick from the popup + arrange up/down" case).
	FAQListModeSelection FAQListMode = "selection"
)

// Validate enforces the closed set: category | selection.
func (m FAQListMode) Validate() error {
	switch m {
	case FAQListModeCategory, FAQListModeSelection:
		return nil
	default:
		return fmt.Errorf("%w: faqlist mode must be one of category|selection, got %q", ErrValidation, string(m))
	}
}

// FAQListSection is the faqlist payload: a mode plus its mode-specific reference.
// The Section custom marshaler round-trips it as {"kind":"faqlist","faqlist":{...}}
// in the page's JSONB content (the payload key equals the kind).
type FAQListSection struct {
	Mode     FAQListMode `json:"mode" yaml:"mode"`
	Category string      `json:"category,omitempty" yaml:"category,omitempty"` // category slug (mode=category)
	Items    []string    `json:"items,omitempty" yaml:"items,omitempty"`      // ordered faq_item IDs (mode=selection)
}

// Compile-time check: *FAQListSection satisfies SectionPayload (Validate() error).
var _ SectionPayload = (*FAQListSection)(nil)

// Validate requires a valid mode AND its mode-specific reference. It checks
// PRESENCE only — the slug's kebab shape and the item IDs' existence are enforced
// by pkg/faq when the FAQ service resolves the refs (cms cannot import faq, and
// presence is all the section layer can guarantee). Every failure wraps
// ErrValidation so callers test errors.Is.
func (s *FAQListSection) Validate() error {
	if err := s.Mode.Validate(); err != nil {
		return err
	}
	switch s.Mode {
	case FAQListModeCategory:
		if strings.TrimSpace(s.Category) == "" {
			return fmt.Errorf("%w: faqlist category mode requires a category slug", ErrValidation)
		}
	case FAQListModeSelection:
		if !faqListHasNonEmpty(s.Items) {
			return fmt.Errorf("%w: faqlist selection mode requires at least one item id", ErrValidation)
		}
	}
	return nil
}

// faqListHasNonEmpty reports whether items carries at least one non-blank ID
// (a selection of only "" / whitespace is treated as empty).
func faqListHasNonEmpty(items []string) bool {
	for _, id := range items {
		if strings.TrimSpace(id) != "" {
			return true
		}
	}
	return false
}
