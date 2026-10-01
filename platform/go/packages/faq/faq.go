// Package faq is the Problem-Domain layer of the reusable FAQ fabric package
// (SDLC 2607-001; design doc docs/plans/2026-07-07-br-faq-fabric-package-design.md;
// fabric module .sbx/workspace/fabric/modules/faq.yml).
//
// FAQ is a first-class, normalized, 3-tier content entity:
//
//	Category → Step (the ordered section) → Item (Q + A + optional Bestie Tip)
//
// authored once and reused across /faq, /sell (the `faqlist` section kind), nav,
// and sitemap. This PD layer is PURE — zero storage imports (only pkg/seo for the
// optional per-category SEO override). The postgres mappers (MD), the Store
// interface + adapter (SI seam), and the read/write Service facade live in their
// own files; HTTP handlers stay in the app (thin consumer, mirroring how BR
// consumes pkg/cms today).
//
// The types + Validate() here are a projection of the entity.yml sources:
//   - .sbx/workspace/clients/bestie/projects/bestierealestate/entities/faq-category/
//   - .../faq-step/   and   .../faq-item/
//
// and mirror pkg/cms conventions (cms.Slug, cms.ErrValidation, CmsEntry.Validate):
// every domain package owns its own named Slug + ErrValidation so the package stays
// decoupled and reusable; every validation failure wraps faq.ErrValidation so
// callers test errors.Is(err, faq.ErrValidation).
package faq

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/shredbx/sbx-core/pkg/seo"
)

// ErrValidation wraps every validation failure. Callers test with
// errors.Is(err, faq.ErrValidation). Mirrors cms.ErrValidation / property.ErrValidation.
var ErrValidation = errors.New("validation")

// =============================================================================
// SLUG — the category URL key (/faq/<slug>); mirrors cms.Slug's kebab invariant.
// =============================================================================

// slugPattern is the kebab-case constraint from entity.yml attribute `slug`:
// lowercase alphanumerics in dash-joined groups (e.g. "buyers-guide", "sellers-guide").
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Slug is the unique, immutable URL key identifying a FAQCategory. It is a named
// type — never a raw string in signatures — so the kebab-case invariant travels
// with the value (standing rule: normalize discriminators).
type Slug string

// String renders the slug for SQL params and URLs.
func (s Slug) String() string { return string(s) }

// Validate enforces the entity.yml `slug` constraints: required (non-empty after
// trim) and kebab-case (^[a-z0-9]+(?:-[a-z0-9]+)*$).
func (s Slug) Validate() error {
	v := strings.TrimSpace(string(s))
	if v == "" {
		return errors.New("slug is required")
	}
	if !slugPattern.MatchString(v) {
		return errors.New("slug must be kebab-case (lowercase letters, digits, single dashes)")
	}
	return nil
}

// =============================================================================
// STATUS — the Publishable lifecycle (draft → published). A named type, never raw.
// =============================================================================

// Status is the per-row Publishable lifecycle of a Category/Step/Item. It is a
// guarded state machine: the closed set is draft|published, and lifecycle moves go
// through TransitionTo so an invalid (corrupt) source/target is rejected at the
// domain layer before any write. (Mirrors pkg/property listing lifecycle; this is
// for normalized ROWS — NOT the cms JSONB two-slot, which is for inline page copy.)
type Status string

const (
	// StatusDraft is the authoring state — invisible publicly, editable freely.
	StatusDraft Status = "draft"
	// StatusPublished is the live state — resolvable on /faq, in nav, in sitemap.
	StatusPublished Status = "published"
)

// Validate enforces the closed set: exactly draft|published.
func (s Status) Validate() error {
	switch s {
	case StatusDraft, StatusPublished:
		return nil
	default:
		return fmt.Errorf("%w: status must be one of draft|published, got %q", ErrValidation, string(s))
	}
}

// String renders the status for SQL params and routing.
func (s Status) String() string { return string(s) }

// TransitionTo is the per-row state-machine guard. With a 2-value lifecycle every
// draft↔published move is legal; the guard exists to reject any move involving an
// invalid (non draft/published) status — the corrupt-row defense. The Service
// facade (Publish/Unpublish) delegates here so the guard is enforced in one place.
// Returns the resulting status (target on success, the original on rejection).
func (s Status) TransitionTo(target Status) (Status, error) {
	if err := s.Validate(); err != nil {
		return s, fmt.Errorf("%w: cannot transition from invalid status %q", ErrValidation, string(s))
	}
	if err := target.Validate(); err != nil {
		return s, fmt.Errorf("%w: cannot transition to invalid status %q", ErrValidation, string(target))
	}
	return target, nil
}

// =============================================================================
// FAQCATEGORY — the top of the 3-tier hierarchy (owns Steps; cascade-delete).
// =============================================================================

// FAQCategory is one guidance collection (e.g. "Properties Buyer Guide"). It owns
// its Steps (containment — cascade-delete follows ownership, Decision #0019) and
// speaks the foundation Publishable protocol. Field order is cosmetic; the storage
// column order is fixed by the mapper (mapper.go).
type Category struct {
	ID string `json:"id" yaml:"id"`

	// Slug is the immutable URL key → /faq/<slug> + the nav destination.
	Slug Slug `json:"slug" yaml:"slug"`
	// Title is the display heading (required).
	Title string `json:"title" yaml:"title"`
	// Blurb is the short intro under the title (optional).
	Blurb string `json:"blurb,omitempty" yaml:"blurb,omitempty"`
	// Position orders categories in the all-view / rail.
	Position int `json:"position" yaml:"position"`
	// Featured marks the single headline category (DB-enforced exactly-one via a
	// partial unique index WHERE is_featured; drives the hero step cards).
	Featured bool `json:"featured" yaml:"featured"`
	// SEO is the optional per-category SEO/OG + canonical/noindex override. A VALUE
	// type (mirrors CmsEntry.SeoMeta): a zero value persists as SQL NULL via the
	// seo.SeoMeta Scanner, never "{}" — so the mapper scans &c.SEO directly.
	SEO seo.SeoMeta `json:"seo,omitempty" yaml:"seo,omitempty"`
	// Status is the Publishable lifecycle (draft|published).
	Status Status `json:"status" yaml:"status"`

	// Document base — identity, optimistic version, audit, soft delete.
	Version   int        `json:"version" yaml:"version"`
	CreatedBy *string    `json:"created_by,omitempty" yaml:"created_by,omitempty"`
	UpdatedBy *string    `json:"updated_by,omitempty" yaml:"updated_by,omitempty"`
	CreatedAt time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" yaml:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty"`
}

// Validate enforces the entity.yml constraints: slug required + kebab-case; title
// required (non-empty trimmed); status valid; SEO (when present) valid. Every
// failure wraps ErrValidation so callers test with errors.Is.
func (c Category) Validate() error {
	if err := c.Slug.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if strings.TrimSpace(c.Title) == "" {
		return fmt.Errorf("%w: title is required", ErrValidation)
	}
	if err := c.Status.Validate(); err != nil {
		return err // already wrapped in ErrValidation
	}
	if err := c.SEO.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	return nil
}

// =============================================================================
// FAQSTEP — the ordered section within a category (owns Items; cascade-delete).
// =============================================================================

// Step is one ordered section within a Category (e.g. "Step 0 — Planning Your
// Purchase"). Position is 0-based AND is the "Step N" label. Owned by its Category
// (cascade-deleted with it) and owns its Items (cascade-deletes them).
type Step struct {
	ID string `json:"id" yaml:"id"`

	// CategoryID is the owning FAQCategory (FK, cascade-delete).
	CategoryID string `json:"category_id" yaml:"category_id"`
	// Position is 0-based — both the "Step N" label and the order (Step 0 = the
	// plan-before-buy step in the Buyers Guide).
	Position int `json:"position" yaml:"position"`
	// Title is the step heading (required).
	Title string `json:"title" yaml:"title"`
	// Description is the optional 1–2 line step intro.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Status is the Publishable lifecycle (draft|published).
	Status Status `json:"status" yaml:"status"`

	Version   int        `json:"version" yaml:"version"`
	CreatedBy *string    `json:"created_by,omitempty" yaml:"created_by,omitempty"`
	UpdatedBy *string    `json:"updated_by,omitempty" yaml:"updated_by,omitempty"`
	CreatedAt time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" yaml:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty"`

	// NOTE: Step has no Validate() yet — it is inert model data. A Step.Validate
	// (title required + status valid, mirroring Category/Item) is added test-first
	// when the first Step scenario demands it (TDD: no behavior without a failing
	// test). Recorded in the app notes; do not add ad-hoc.
}

// =============================================================================
// FAQITEM — the leaf: one Q + A + optional Bestie Tip (owned by a Step).
// =============================================================================

// Item is one question/answer pair (with an optional "Bestie Tip") under a Step.
// It is the unit of AI search (Store.Search) and the unit picked into pages (the
// faqlist section). The "<step>.<item>" label is DERIVED via DisplayLabel, never
// stored twice. Owned by its Step (cascade-deleted with it).
type Item struct {
	ID string `json:"id" yaml:"id"`

	// StepID is the owning FAQStep (FK, cascade-delete).
	StepID string `json:"step_id" yaml:"step_id"`
	// Position is the ".N" in the derived "<step>.<item>" label and the order.
	Position int `json:"position" yaml:"position"`
	// Question is the Q (required).
	Question string `json:"question" yaml:"question"`
	// Answer is the markdown A (required; sanitized at the render boundary —
	// mirrors the Prose section kind, #0273 / NFR-007).
	Answer string `json:"answer" yaml:"answer"`
	// Tip is the optional "Bestie Tip" markdown (sanitized at render).
	Tip string `json:"tip,omitempty" yaml:"tip,omitempty"`
	// Status is the Publishable lifecycle (draft|published).
	Status Status `json:"status" yaml:"status"`

	Version   int        `json:"version" yaml:"version"`
	CreatedBy *string    `json:"created_by,omitempty" yaml:"created_by,omitempty"`
	UpdatedBy *string    `json:"updated_by,omitempty" yaml:"updated_by,omitempty"`
	CreatedAt time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" yaml:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty"`
}

// Validate enforces: question required; answer required; status valid. (Tip is
// optional.)
func (i Item) Validate() error {
	if strings.TrimSpace(i.Question) == "" {
		return fmt.Errorf("%w: question is required", ErrValidation)
	}
	if strings.TrimSpace(i.Answer) == "" {
		return fmt.Errorf("%w: answer is required", ErrValidation)
	}
	if err := i.Status.Validate(); err != nil {
		return err
	}
	return nil
}

// =============================================================================
// DISPLAY LABEL — the derived "<step>.<item>" marker (never stored twice).
// =============================================================================

// DisplayLabel derives the "<stepPosition>.<itemPosition>" marker for an Item
// (e.g. step 1, item 2 → "1.2"; step 0, item 1 → "0.1"). It is a pure projection
// of the two positions — the label is computed at render, not persisted, so it can
// never drift from the real ordering.
func DisplayLabel(stepPosition, itemPosition int) string {
	return fmt.Sprintf("%d.%d", stepPosition, itemPosition)
}
