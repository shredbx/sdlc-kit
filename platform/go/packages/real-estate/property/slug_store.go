package property

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/shredbx/sbx-core/pkg/repository"
)

// slugCreateMaxAttempts bounds the Create/Update slug-allocation retry loop. A retry
// only happens on a live-unique-index race (two concurrent writes deriving the same
// slug, B5); the taken predicate re-queries each attempt so the loser sees the winner's
// row and picks the next candidate. A small bound is ample (a race past a handful of
// attempts is pathological) and guarantees the write never spins — it surfaces a
// conflict error instead of a 500-spin.
const slugCreateMaxAttempts = 5

// =============================================================================
// SLUG persistence — the property public-URL key + its published-slug HISTORY.
//
// Model = the FriendlyId "History" pattern (task 2606-120): the UUID stays
// identity, properties.slug is the readable current URL key (unique among LIVE
// rows), and property_slug_history records every slug a property has been
// PUBLISHED under (current + prior). History drives (a) permanent 301s for
// old/UUID links and (b) uniqueness-against-history so a slug is never reissued
// to a different property.
//
// These are PropertyService methods (not a separate repository type) because the
// service already owns Create/Update and holds the pool + schema — mirroring the
// in-package Exists() probe. The DB error/NotFound style mirrors
// PropertyCollectionRepository (repository.NewNotFoundError, schema-as-constant,
// only user values bound as parameters). Every method is nil-pool-safe (fixture
// mode) so the unit fixtures that pass a nil pool keep working.
// =============================================================================

// GetBySlug loads the LIVE (deleted_at IS NULL) property whose CURRENT slug equals
// slug, composed through the full mapper read (repo.Get) so the returned aggregate
// carries the same joined columns the UUID path returns. repository.ErrNotFound when
// no live property has that current slug — the public handler then 404s. The
// PUBLISHED gate is intentionally NOT applied here: it stays in the public handler
// (one gate), so a live-but-unpublished slug resolves to a row the handler then 404s
// (indistinguishable-404 for drafts). A nil pool yields ErrNotFound (fixture mode).
func (s *PropertyService) GetBySlug(ctx context.Context, slug Slug) (Property, error) {
	if err := slug.Validate(); err != nil {
		return Property{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if s.pool == nil {
		return Property{}, repository.NewNotFoundError(s.schema+".properties", slug.String())
	}
	var id string
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM `+s.schema+`.properties WHERE slug = $1 AND deleted_at IS NULL LIMIT 1`,
		slug.String()).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Property{}, repository.NewNotFoundError(s.schema+".properties", slug.String())
		}
		return Property{}, fmt.Errorf("property get-by-slug %q: %w", slug.String(), err)
	}
	if s.repo == nil {
		return Property{}, repository.NewNotFoundError(s.schema+".properties", slug.String())
	}
	return s.repo.Get(ctx, id)
}

// ResolveSlugHistory looks up the property a historical (or current) PUBLISHED slug
// belongs to. found=false (no error) when the slug was never published by any
// property. This is the 301 seam: a key that is not a current live slug but IS in
// history resolves to its owning property (the handler then loads it by id and the
// route layer 301s to the canonical current slug). A nil pool yields found=false.
func (s *PropertyService) ResolveSlugHistory(ctx context.Context, slug Slug) (propertyID string, found bool, err error) {
	if s.pool == nil {
		return "", false, nil
	}
	row := s.pool.QueryRow(ctx,
		`SELECT property_id FROM `+s.schema+`.property_slug_history WHERE slug = $1 LIMIT 1`,
		slug.String())
	scanErr := row.Scan(&propertyID)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return "", false, nil
	}
	if scanErr != nil {
		return "", false, fmt.Errorf("property resolve slug-history %q: %w", slug.String(), scanErr)
	}
	return propertyID, true, nil
}

// RecordPublishedSlug records a property's current slug in the published-slug history
// — IDEMPOTENT via ON CONFLICT (slug) DO NOTHING, so re-publishing the same slug
// (C5/C6) never creates a duplicate row, and the slug is reserved against reuse by
// any other property forever (until hard-delete cascades it). Called on the PUBLISH
// transitions only (publish-gated recording, C-series): SetPublished(true) and an
// Update that leaves the property published. A nil pool is a no-op (fixture mode).
func (s *PropertyService) RecordPublishedSlug(ctx context.Context, slug Slug, propertyID string) error {
	if s.pool == nil {
		return nil
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO `+s.schema+`.property_slug_history (slug, property_id)
		 VALUES ($1, $2)
		 ON CONFLICT (slug) DO NOTHING`,
		slug.String(), propertyID); err != nil {
		return fmt.Errorf("property record published slug %q: %w", slug.String(), err)
	}
	return nil
}

// SlugTaken reports whether slug is unavailable to the property identified by
// excludePropertyID — the `taken` predicate ResolveSlug consumes. A slug is taken
// when it is published in property_slug_history by ANOTHER property (B2 — never
// hijacked) OR is the current slug of ANOTHER live property. The excludePropertyID
// guard is what enables the B3 RECLAIM: a slug that lives ONLY in THIS property's own
// history (and is no other live/historical owner's) reports FREE, so the property can
// reclaim its own prior slug un-suffixed. For Create, excludePropertyID is the new
// (not-yet-inserted) UUID, so the guard excludes nothing real and this is a pure
// "taken by anyone?" check. A nil pool yields false (fixture mode).
func (s *PropertyService) SlugTaken(ctx context.Context, slug Slug, excludePropertyID string) (bool, error) {
	if s.pool == nil {
		return false, nil
	}
	var taken bool
	err := s.pool.QueryRow(ctx,
		`SELECT
		   EXISTS(SELECT 1 FROM `+s.schema+`.property_slug_history
		           WHERE slug = $1 AND property_id <> $2)
		   OR
		   EXISTS(SELECT 1 FROM `+s.schema+`.properties
		           WHERE slug = $1 AND deleted_at IS NULL AND id <> $2)`,
		slug.String(), excludePropertyID).Scan(&taken)
	if err != nil {
		return false, fmt.Errorf("property slug-taken %q: %w", slug.String(), err)
	}
	return taken, nil
}

// generateSlug derives a unique current slug for a property from its title and area
// using the graceful candidate ladder (title, then title+area) with the DB-backed
// taken predicate (SlugTaken, self-excluding so a rename can reclaim its own prior
// slug — B3). area is the property's sub-district (preferred) or city. excludeID is
// the property's own id (the not-yet-inserted UUID on Create; the row id on Update).
// A SlugTaken DB error is surfaced; otherwise the predicate is conservative (treats a
// transient error path as "free") only via the explicit error return, never silently.
func (s *PropertyService) generateSlug(ctx context.Context, p Property, excludeID string) (Slug, error) {
	title := ""
	if p.Title != nil {
		title = *p.Title
	}
	area := p.Address.SubDistrict
	if area == "" {
		area = p.Address.City
	}

	var takenErr error
	taken := func(cand Slug) bool {
		if takenErr != nil {
			return false // error already captured — stop probing, surface it below
		}
		t, err := s.SlugTaken(ctx, cand, excludeID)
		if err != nil {
			takenErr = err
			// Per this function's contract an error path reads as FREE so the
			// ladder TERMINATES immediately and generateSlug returns takenErr —
			// the create aborts with the real error. (The old `return true` fed
			// EnsureUniqueSlug's suffix loop "taken" forever: a 23-minute hot
			// spin inside a property create, 2607-014.)
			return false
		}
		return t
	}

	slug := ResolveSlug(SlugCandidates(title, area), taken)
	if takenErr != nil {
		return "", fmt.Errorf("generate slug: %w", takenErr)
	}
	return slug, nil
}

// ResolvePublic resolves a public /properties/{key} segment to a property, where key
// may be the CURRENT slug, a HISTORICAL/UUID alias, or garbage. It is the single
// read-seam the public handler calls in place of a bare Get(id):
//
//	current live slug      -> the property               (D1; route serves 200)
//	historical slug        -> its owning property         (D2; route 301s to canonical)
//	legacy/indexed UUID     -> the property by id          (D3; route 301s to canonical)
//	unknown / garbage key  -> repository.ErrNotFound      (D4; handler 404s)
//
// It does NOT issue a redirect — the API always serves the resolved property's full
// payload (carrying the canonical slug); the SvelteKit route layer compares key vs
// response.slug and 301s. The PUBLISHED / soft-delete gates stay in the handler /
// repo.Get (deleted_at IS NULL), so a draft or soft-deleted property resolved by any
// path still surfaces as not-served (D5/D6). A nil repo yields ErrNotFound.
func (s *PropertyService) ResolvePublic(ctx context.Context, key string) (Property, error) {
	if s.repo == nil {
		return Property{}, repository.NewNotFoundError(s.schema+".properties", key)
	}

	// 1) current live slug (D1) — the common case; does not depend on a history row.
	if p, err := s.GetBySlug(ctx, Slug(key)); err == nil {
		return p, nil
	} else if !errors.Is(err, repository.ErrNotFound) && !errors.Is(err, ErrValidation) {
		// A real DB error (not just "no such slug" / an invalid-slug shape) must not be
		// masked as a 404 — surface it so the handler 500s rather than hiding a fault.
		return Property{}, err
	}

	// 2) historical published slug (D2) — resolve to its owning property by id.
	if pid, found, err := s.ResolveSlugHistory(ctx, Slug(key)); err != nil {
		return Property{}, err
	} else if found {
		return s.repo.Get(ctx, pid)
	}

	// 3) legacy/indexed UUID (D3) — only attempt a by-id load when the key is a UUID,
	// so a non-UUID garbage key never hits the id lookup.
	if _, err := uuid.Parse(key); err == nil {
		return s.repo.Get(ctx, key)
	}

	// 4) unknown / garbage (D4).
	return Property{}, repository.NewNotFoundError(s.schema+".properties", key)
}
