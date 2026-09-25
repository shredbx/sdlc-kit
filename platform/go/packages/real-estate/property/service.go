package property

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shredbx/sbx-core/pkg/repository"
	"github.com/shredbx/sbx-core/pkg/seo"
)

var ErrValidation = errors.New("validation")

type PropertyService struct {
	repo     repository.Repository[Property]
	searcher repository.Searcher[Property]
	pool     *pgxpool.Pool
	schema   string

	// deriveTagSlugs is an OPTIONAL, app-injected hook that materializes a
	// property's unified tag-slug identity (custom-tag slugs ∪ app-derived autotag
	// slugs) from the fully-assembled aggregate. It is set by the consumer
	// (SetDeriveTagSlugs) so the shared package stays free of any autotag / BR
	// knowledge. When nil the behaviour is unchanged: Create/Update never write
	// the tag_slugs column. Invoked AFTER SaveTags in both write paths, so the
	// custom tags it reads off the property are the same ones just persisted.
	deriveTagSlugs func(Property) []string
}

func NewPropertyService(repo repository.Repository[Property], pool *pgxpool.Pool, schema string) *PropertyService {
	return &PropertyService{repo: repo, pool: pool, schema: schema}
}

// SetSearcher injects the Searcher implementation (optional — search disabled if nil).
func (s *PropertyService) SetSearcher(searcher repository.Searcher[Property]) {
	s.searcher = searcher
}

// SetDeriveTagSlugs injects the app's unified tag-slug deriver (optional). The hook
// receives the fully-assembled Property aggregate (custom tags + amenities +
// location + type + transaction) and returns the deduped, ordered slug
// set persisted to properties.tag_slugs by Create/Update. A nil deriver (the default)
// leaves tag_slugs untouched, so the shared package's behaviour is unchanged for any
// consumer that does not opt in. The shared package never knows HOW the slugs are
// derived — that BR-specific knowledge is injected here (Rule #9, modular arch).
func (s *PropertyService) SetDeriveTagSlugs(derive func(Property) []string) {
	s.deriveTagSlugs = derive
}

func (s *PropertyService) Search(ctx context.Context, opts repository.SearchOptions) (repository.SearchResult[Property], error) {
	if s.searcher == nil {
		return repository.SearchResult[Property]{}, nil
	}
	return s.searcher.Search(ctx, opts)
}

func (s *PropertyService) Create(ctx context.Context, p Property) (Property, error) {
	if err := validateCreate(p); err != nil {
		return Property{}, err
	}
	normalizeFurnished(&p)
	p.ID = uuid.NewString()
	p.IsPublished = false
	// F.7: every new property starts active. Lifecycle transitions happen
	// via the dedicated PATCH /lifecycle endpoint, never at create time.
	if p.LifecycleStatus == "" {
		p.LifecycleStatus = LifecycleActive
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	return s.create(ctx, p)
}

// CreateImported inserts a property with a caller-supplied ID and legacy
// timestamps preserved — the bulk-import path (task 2607-001). Unlike Create,
// which mints a fresh UUID + now(), it keeps p.ID (required, non-empty) and
// preserves p.CreatedAt/UpdatedAt (a zero value falls back to now), so a
// re-imported listing keeps its identity and its true age. It also PRESERVES the
// caller's IsPublished — a re-import reproduces the source's original live/draft
// state exactly (100% identity), so an originally-published listing lands
// published (create() records its published slug); the importer, not this method,
// decides the state. Defaults an unset lifecycle to active. Shares Create's
// persistence body via create().
func (s *PropertyService) CreateImported(ctx context.Context, p Property) (Property, error) {
	if err := validateCreate(p); err != nil {
		return Property{}, err
	}
	if strings.TrimSpace(p.ID) == "" {
		return Property{}, fmt.Errorf("%w: imported property requires a non-empty id", ErrValidation)
	}
	normalizeFurnished(&p)
	if p.LifecycleStatus == "" {
		p.LifecycleStatus = LifecycleActive
	}
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}
	return s.create(ctx, p)
}

// create is the shared persistence body for Create and CreateImported. It
// assumes p already has its ID, IsPublished, LifecycleStatus, and timestamps
// set, then derives a unique slug, INSERTs (with slug-race retry), and writes
// every attribute-group / amenity / tag / seo sub-table.
func (s *PropertyService) create(ctx context.Context, p Property) (Property, error) {
	if s.repo == nil {
		// Fixture mode: still derive a slug so the in-memory aggregate is realistic
		// (the candidate ladder is pure; SlugTaken is a nil-pool no-op → first
		// candidate). A derivation error here cannot occur with a nil pool.
		if slug, err := s.generateSlug(ctx, p, p.ID); err == nil {
			p.Slug = slug.String()
		}
		return p, nil
	}

	// Slug (2606-120): derive a unique current slug from title (+area) via the
	// graceful candidate ladder, then INSERT — retrying on a live-unique-index race
	// (B5) so a concurrent create of the same title never 500s. The taken predicate
	// re-queries on each attempt, so a retry sees the row that just won the race.
	var result Property
	var err error
	for attempt := 0; attempt < slugCreateMaxAttempts; attempt++ {
		slug, genErr := s.generateSlug(ctx, p, p.ID)
		if genErr != nil {
			return Property{}, genErr
		}
		p.Slug = slug.String()
		result, err = s.repo.Create(ctx, p)
		if err == nil {
			break
		}
		// A slug-index collision surfaces as repository.ErrConflict (PostgresStore maps
		// 23505). Retry with a freshly generated candidate; any other error is fatal.
		if !errors.Is(err, repository.ErrConflict) {
			return result, err
		}
	}
	if err != nil {
		return result, fmt.Errorf("create: slug allocation exhausted after %d attempts: %w", slugCreateMaxAttempts, err)
	}

	// Attribute-group writes (2605-066). See Update() for rationale.
	if err := s.SaveAddress(ctx, result.ID, p.Address); err != nil {
		return result, fmt.Errorf("create: %w", err)
	}
	if err := s.SaveSizes(ctx, result.ID, p.Sizes); err != nil {
		return result, fmt.Errorf("create: %w", err)
	}
	if err := s.SaveRooms(ctx, result.ID, p.Rooms); err != nil {
		return result, fmt.Errorf("create: %w", err)
	}
	if p.BuildingSpecs != nil {
		if err := s.SaveBuildingSpecs(ctx, result.ID, p.BuildingSpecs); err != nil {
			return result, fmt.Errorf("create: save building specs: %w", err)
		}
	}
	if p.Policies != nil {
		if err := s.SavePolicies(ctx, result.ID, p.Policies); err != nil {
			return result, fmt.Errorf("create: save policies: %w", err)
		}
	}

	if len(p.Amenities) > 0 {
		if err := s.SaveAmenities(ctx, result.ID, p.Amenities); err != nil {
			return result, fmt.Errorf("create: save amenities: %w", err)
		}
	}
	if len(p.Tags) > 0 {
		if err := s.SaveTags(ctx, result.ID, p.Tags); err != nil {
			return result, fmt.Errorf("create: save tags: %w", err)
		}
	}
	// Unified tag-slug identity (2606-001): after the custom tags are persisted,
	// materialize the full custom ∪ autotag slug set so the ?tags= filter and the
	// ?q= search read ONE source. Runs unconditionally (autotags apply even with no
	// custom tags); a nil hook leaves tag_slugs untouched. The derive reads the
	// in-memory aggregate p (amenities/location/type already attached),
	// not a refetch, so it sees exactly what was just written.
	if s.deriveTagSlugs != nil {
		if err := s.saveTagSlugs(ctx, result.ID, s.deriveTagSlugs(p)); err != nil {
			return result, fmt.Errorf("create: save tag_slugs: %w", err)
		}
	}
	if p.SeoMeta != nil {
		if err := s.SaveSeoMeta(ctx, result.ID, p.SeoMeta); err != nil {
			return result, fmt.Errorf("create: save seo_meta: %w", err)
		}
	}
	savedTags := NormalizeTags(p.Tags)

	// Published-slug history (2606-120, publish-gated): record the slug only when the
	// property is published. Create currently forces IsPublished=false (C1: a draft
	// has no public refs, so no history row), so this is dormant at create — the first
	// history row is written by SetPublished(true) (C2). Kept here so the recording is
	// correct should create ever produce a published row, and so the lifecycle rule
	// (record-iff-published) lives in ONE shape across both write paths.
	if result.IsPublished {
		if err := s.RecordPublishedSlug(ctx, Slug(result.Slug), result.ID); err != nil {
			return result, fmt.Errorf("create: record published slug: %w", err)
		}
	}

	// Units (M2) are intentionally NOT persisted here — they are a 1:many child with
	// stable identity (images/canvas bind to a unit id), managed only via the per-row
	// SaveUnit/DeleteUnit sub-resource path. A Create/Update never writes units; the
	// response carries units: null (no consumer reads units off this response).
	// Refetch so the returned property has the joined attribute-group values.
	if refreshed, err := s.repo.Get(ctx, result.ID); err == nil {
		refreshed.Amenities = p.Amenities
		refreshed.Tags = savedTags
		refreshed.SeoMeta = p.SeoMeta
		return refreshed, nil
	}

	result.Amenities = p.Amenities
	result.Tags = savedTags
	result.SeoMeta = p.SeoMeta
	return result, nil
}

func (s *PropertyService) Update(ctx context.Context, id string, p Property) (Property, error) {
	if err := validateUpdate(p); err != nil {
		return Property{}, err
	}
	normalizeFurnished(&p)
	p.ID = id
	p.UpdatedAt = time.Now()

	// Server-owned fields a metadata PUT must not clobber. The edit form (and any
	// PATCH-unaware client) sends neither, so a naive full replace would corrupt
	// them. Both have dedicated PATCH endpoints that are the formal way to change
	// them; PUT keeps PATCH-like (preserve-on-omit) semantics so a metadata edit
	// never has a destructive side effect. Fetch the existing row once and carry
	// the protected values forward:
	//   - lifecycle_status: FK-constrained NOT NULL. Preserve only when omitted
	//     ("" sentinel); PATCH /lifecycle owns explicit changes.
	//   - is_published: ALWAYS preserve. It is a non-pointer bool with no "omitted"
	//     sentinel (false-from-client is indistinguishable from absent), and the
	//     edit form never sends it — so without this, every Information-facet Save
	//     silently UNPUBLISHED a live listing. PATCH /publish (TogglePublish) is
	//     the sole writer of publish state.
	if s.repo != nil {
		if existing, err := s.repo.Get(ctx, id); err == nil {
			if p.LifecycleStatus == "" {
				p.LifecycleStatus = existing.LifecycleStatus
			}
			p.IsPublished = existing.IsPublished
		}
	}
	if p.LifecycleStatus == "" {
		p.LifecycleStatus = LifecycleActive
	}

	if s.repo == nil {
		// Fixture mode: derive a slug from the request (pure ladder; nil-pool SlugTaken
		// is a no-op) so the in-memory result is realistic.
		if slug, err := s.generateSlug(ctx, p, id); err == nil {
			p.Slug = slug.String()
		}
		return p, nil
	}

	// Slug (2606-120): the slug FOLLOWS the title — re-derive from the (new) title via
	// the same graceful ladder, with the taken predicate EXCLUDING self so a rename can
	// RECLAIM this property's own prior slug un-suffixed (B3); a slug owned by another
	// property in history/live is never hijacked (B2). Retry on a live-unique-index race
	// (B5) — repo.Update maps 23505 to repository.ErrConflict.
	var result Property
	var err error
	for attempt := 0; attempt < slugCreateMaxAttempts; attempt++ {
		slug, genErr := s.generateSlug(ctx, p, id)
		if genErr != nil {
			return Property{}, genErr
		}
		p.Slug = slug.String()
		result, err = s.repo.Update(ctx, id, p)
		if err == nil {
			break
		}
		if !errors.Is(err, repository.ErrConflict) {
			return result, err
		}
	}
	if err != nil {
		return result, fmt.Errorf("update: slug allocation exhausted after %d attempts: %w", slugCreateMaxAttempts, err)
	}

	// Attribute-group writes — the primary repo.Update only writes properties.*
	// (the mapper's ToRow scope). Attribute groups live in dependent tables and
	// require their own upsert path (2605-066). Always called: an unspecified
	// (zero-value) Address still produces a property_locations row so the LEFT
	// JOIN in Get returns predictable zero values rather than NULLs.
	if err := s.SaveAddress(ctx, id, p.Address); err != nil {
		return result, fmt.Errorf("update: %w", err)
	}
	if err := s.SaveSizes(ctx, id, p.Sizes); err != nil {
		return result, fmt.Errorf("update: %w", err)
	}
	if err := s.SaveRooms(ctx, id, p.Rooms); err != nil {
		return result, fmt.Errorf("update: %w", err)
	}

	// Building specs + lease policies — nil = field absent (preserve existing),
	// non-nil = replace. Mirrors Create; their omission here silently dropped specs
	// and policies on every edit-form Save (DRIFT-1, 2606-047).
	if p.BuildingSpecs != nil {
		if err := s.SaveBuildingSpecs(ctx, id, p.BuildingSpecs); err != nil {
			return result, fmt.Errorf("update: save building specs: %w", err)
		}
		result.BuildingSpecs = p.BuildingSpecs
	}
	if p.Policies != nil {
		if err := s.SavePolicies(ctx, id, p.Policies); err != nil {
			return result, fmt.Errorf("update: save policies: %w", err)
		}
		result.Policies = p.Policies
	}

	// nil = field absent in request (don't touch). non-nil = replace set.
	if p.Amenities != nil {
		if err := s.SaveAmenities(ctx, id, p.Amenities); err != nil {
			return result, fmt.Errorf("update: save amenities: %w", err)
		}
		result.Amenities = p.Amenities
	}
	if p.Tags != nil {
		if err := s.SaveTags(ctx, id, p.Tags); err != nil {
			return result, fmt.Errorf("update: save tags: %w", err)
		}
		result.Tags = NormalizeTags(p.Tags)
	}
	// Unified tag-slug identity (2606-001): rebuild the full custom ∪ autotag slug
	// set after every full-aggregate save. Editing ANY tag input (amenities,
	// location, type, transaction, or custom tags) flows through this
	// path, so a single unconditional, nil-safe derive keeps tag_slugs in lockstep
	// with the property's other facets. The derive reads `p` (the request aggregate,
	// carrying the just-saved amenities/location/type/tags); when the
	// request omitted tags (p.Tags == nil) the existing custom tags are preserved in
	// the column but NOT re-folded here — that is acceptable because a tag-only edit
	// always sends p.Tags, and an attribute edit that omits tags still re-derives the
	// autotag portion from the fields it did send. A nil hook is a no-op.
	if s.deriveTagSlugs != nil {
		if err := s.saveTagSlugs(ctx, id, s.deriveTagSlugs(p)); err != nil {
			return result, fmt.Errorf("update: save tag_slugs: %w", err)
		}
	}
	// nil = field absent in request (preserve existing). non-nil = replace; a
	// pointer to an empty SeoMeta clears the overrides (SaveSeoMeta writes NULL).
	if p.SeoMeta != nil {
		if err := s.SaveSeoMeta(ctx, id, p.SeoMeta); err != nil {
			return result, fmt.Errorf("update: save seo_meta: %w", err)
		}
		result.SeoMeta = p.SeoMeta
	}

	// Published-slug history (2606-120, publish-gated): when this Update leaves the
	// property PUBLISHED (is_published is preserved from the existing row above), record
	// the now-current slug (C3: a retitle-while-published records the NEW slug; the prior
	// slug is already in history from its own publish/edit, so its URL keeps 301-ing).
	// ON CONFLICT DO NOTHING makes a no-op-rename (same slug) idempotent. A draft Update
	// records nothing (C1). Records the slug just persisted (result.Slug).
	if result.IsPublished {
		if err := s.RecordPublishedSlug(ctx, Slug(result.Slug), id); err != nil {
			return result, fmt.Errorf("update: record published slug: %w", err)
		}
	}

	// Units (M2) are not written here — see Create: they have a dedicated per-row CRUD
	// path to preserve stable unit ids. Refetch via Get so the response reflects the
	// just-written attribute groups (the in-memory `result` only has the primary-table view).
	if refreshed, err := s.repo.Get(ctx, id); err == nil {
		refreshed.Amenities = result.Amenities
		refreshed.Tags = result.Tags
		refreshed.SeoMeta = result.SeoMeta
		refreshed.BuildingSpecs = result.BuildingSpecs
		refreshed.Policies = result.Policies
		return refreshed, nil
	}

	return result, nil
}

func (s *PropertyService) Delete(ctx context.Context, id string) error {
	if s.repo == nil {
		return nil
	}
	return s.repo.Delete(ctx, id)
}

func (s *PropertyService) SetPublished(ctx context.Context, id string, published bool) error {
	if s.repo == nil {
		return nil
	}
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	existing.IsPublished = published
	if _, err = s.repo.Update(ctx, id, existing); err != nil {
		return err
	}
	// Published-slug history (2606-120): the PUBLISH transition is the primary recorder
	// (C2 first publish → current slug → history; C5/C6 republish → ON CONFLICT DO
	// NOTHING idempotent no-op). Unpublishing records nothing — the slug stays reserved
	// in history (D6/E2). The slug is unchanged by a publish toggle (existing.Slug), so
	// this records the property's current slug exactly.
	if published {
		if err := s.RecordPublishedSlug(ctx, Slug(existing.Slug), id); err != nil {
			return fmt.Errorf("set published: record slug: %w", err)
		}
	}
	return nil
}

// SetLifecycleStatus updates only the lifecycle_status column. Does not touch
// is_published (Decision D-7 — sold/leased stay public by default). Caller
// must pass a valid dictionary code; service rejects unknown values up-front
// so the FK violation isn't surfaced as a generic 500.
func (s *PropertyService) SetLifecycleStatus(ctx context.Context, id string, status LifecycleStatus) error {
	if !ValidLifecycleStatus(status) {
		return fmt.Errorf("%w: unknown lifecycle status %q", ErrValidation, status)
	}
	if s.repo == nil {
		return nil
	}
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	existing.LifecycleStatus = status
	_, err = s.repo.Update(ctx, id, existing)
	return err
}

// SetVisibility surgically patches the 3 per-section public visibility flags
// (location_public / plot_public / policies_public) and NOTHING else. It is a
// partial pointer-patch: a nil pointer leaves that flag unchanged, so a caller
// can flip ONE flag without resending — or clobbering — any other property field.
// Only the 3 visibility columns (+ updated_at) are ever written; this never
// affects address, sizes, pricing, lifecycle, publish state, or any other field.
//
// Persistence splits on the pool: in production (pool != nil) it issues ONE
// pool-direct UPDATE touching only the 3 columns + updated_at — true surgical
// option (a), so it cannot clobber a concurrent mid-edit of any unrelated column;
// in fixture mode (pool == nil) it persists the resolved aggregate via s.repo.Update
// (the YAMLStore), keeping the endpoint fully unit-testable. Returns the property
// with the resulting flags so the response echoes the post-patch state without a
// re-Get. An unknown id yields repository.ErrNotFound (→ the handler maps to 404).
func (s *PropertyService) SetVisibility(ctx context.Context, id string, locationPublic, plotPublic, policiesPublic *bool) (Property, error) {
	if s.repo == nil {
		// No-repo guard mirroring the sibling Set* methods — only headless/fixture
		// wiring leaves repo nil; production always wires it, so this is never a
		// real-id path (it returns a zero Property, not a 404).
		return Property{}, nil
	}
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return Property{}, err
	}

	// Apply non-nil pointers; nil = leave that flag unchanged.
	if locationPublic != nil {
		existing.LocationPublic = *locationPublic
	}
	if plotPublic != nil {
		existing.PlotPublic = *plotPublic
	}
	if policiesPublic != nil {
		existing.PoliciesPublic = *policiesPublic
	}

	if s.pool == nil {
		// Fixture mode: persist the resolved aggregate via the in-memory store so the
		// handler tests assert persist + echo + 404 end-to-end (YAMLStore.Update returns
		// ErrNotFound for an unknown id).
		if _, err := s.repo.Update(ctx, id, existing); err != nil {
			return Property{}, err
		}
		return existing, nil
	}

	// Production: ONE surgical UPDATE — only the 3 visibility columns + updated_at,
	// never reads-then-rewrites any unrelated column. Do NOT "simplify" this to
	// s.repo.Update(existing): that re-runs slug derivation + attribute-group upserts
	// and rewrites the whole row — the fixture path above is column-equivalent ONLY
	// because the in-memory store has none of that machinery.
	tag, err := s.pool.Exec(ctx,
		`UPDATE `+s.schema+`.properties
		    SET location_public = $1, plot_public = $2, policies_public = $3, updated_at = NOW()
		  WHERE id = $4 AND deleted_at IS NULL`,
		existing.LocationPublic, existing.PlotPublic, existing.PoliciesPublic, id)
	if err != nil {
		return Property{}, fmt.Errorf("set visibility: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Property{}, repository.ErrNotFound
	}
	return existing, nil
}

func (s *PropertyService) Get(ctx context.Context, id string) (Property, error) {
	if s.repo == nil {
		return Property{}, nil
	}
	return s.repo.Get(ctx, id)
}

func (s *PropertyService) List(ctx context.Context, opts repository.ListOptions) ([]Property, int, error) {
	if s.repo == nil {
		return nil, 0, nil
	}
	return s.repo.List(ctx, opts)
}

// Exists reports whether a LIVE (non-soft-deleted) property row exists. It is the
// single domain entry-point for the existence guard the child-collection handlers
// run before a write (documents / image-collections / notes) — replacing the
// per-handler duplicated `SELECT EXISTS(… deleted_at IS NULL)` checks. A nil pool
// (fixture mode) yields false, nil. Read-only.
func (s *PropertyService) Exists(ctx context.Context, id string) (bool, error) {
	if s.pool == nil {
		return false, nil
	}
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM `+s.schema+`.properties WHERE id = $1 AND deleted_at IS NULL)`,
		id).Scan(&exists); err != nil {
		return false, fmt.Errorf("property exists: %w", err)
	}
	return exists, nil
}

// LoadAmenities fetches amenities for a property from the join table.
func (s *PropertyService) LoadAmenities(ctx context.Context, propertyID string) ([]Amenity, error) {
	if s.pool == nil {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT amenity_code, "group" FROM `+s.schema+`.property_amenities WHERE property_id = $1 ORDER BY "group", amenity_code`,
		propertyID)
	if err != nil {
		return nil, fmt.Errorf("load amenities: %w", err)
	}
	defer rows.Close()

	var amenities []Amenity
	for rows.Next() {
		var a Amenity
		if err := rows.Scan(&a.Code, &a.Group); err != nil {
			return nil, fmt.Errorf("scan amenity: %w", err)
		}
		amenities = append(amenities, a)
	}
	return amenities, rows.Err()
}

// SaveAmenities replaces all amenities for a property (transactional DELETE+INSERT).
func (s *PropertyService) SaveAmenities(ctx context.Context, propertyID string, amenities []Amenity) error {
	if s.pool == nil {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("save amenities: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM `+s.schema+`.property_amenities WHERE property_id = $1`,
		propertyID); err != nil {
		return fmt.Errorf("save amenities: delete: %w", err)
	}

	for _, a := range amenities {
		if _, err := tx.Exec(ctx,
			`INSERT INTO `+s.schema+`.property_amenities (property_id, amenity_code, "group") VALUES ($1, $2, $3)`,
			propertyID, a.Code, string(a.Group)); err != nil {
			return fmt.Errorf("save amenities: insert %s: %w", a.Code, err)
		}
	}

	return tx.Commit(ctx)
}

// SaveAddress upserts the Address attribute-group row in property_locations.
// Uses ON CONFLICT (property_id) DO UPDATE so the same call works for create
// (no row yet) and update (existing row). Empty Address (all zero values) is
// still persisted as an explicit row — the LEFT JOIN in Get expects either a
// row with empty strings or NULL columns, both of which scan to zero values.
//
// Called by Create + Update after the primary properties row is written.
// Pre-2605-066 this gap silently dropped all address edits made via the form —
// the only addresses present in DB were those backfilled by SQL migrations.
func (s *PropertyService) SaveAddress(ctx context.Context, propertyID string, addr Address) error {
	if s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO `+s.schema+`.property_locations
			(property_id, street, unit, sub_district, city, province, postal_code, country, latitude, longitude, polygon, location_map_view, region_map_view)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 ON CONFLICT (property_id) DO UPDATE SET
			street            = EXCLUDED.street,
			unit              = EXCLUDED.unit,
			sub_district      = EXCLUDED.sub_district,
			city              = EXCLUDED.city,
			province          = EXCLUDED.province,
			postal_code       = EXCLUDED.postal_code,
			country           = EXCLUDED.country,
			latitude          = EXCLUDED.latitude,
			longitude         = EXCLUDED.longitude,
			polygon           = EXCLUDED.polygon,
			location_map_view = EXCLUDED.location_map_view,
			region_map_view   = EXCLUDED.region_map_view`,
		propertyID,
		addr.Street, addr.Unit, addr.SubDistrict, addr.City, addr.Province,
		addr.PostalCode, addr.Country, addr.Latitude, addr.Longitude, addr.Polygon,
		addr.LocationMapView, addr.RegionMapView,
	)
	if err != nil {
		return fmt.Errorf("save address: %w", err)
	}
	return nil
}

// SaveSizes upserts the Sizes attribute-group row in property_sizes.
// nil sizes → DELETE the row (intent: "this property has no size data").
func (s *PropertyService) SaveSizes(ctx context.Context, propertyID string, sizes *Sizes) error {
	if s.pool == nil {
		return nil
	}
	if sizes == nil {
		if _, err := s.pool.Exec(ctx,
			`DELETE FROM `+s.schema+`.property_sizes WHERE property_id = $1`,
			propertyID); err != nil {
			return fmt.Errorf("save sizes: delete: %w", err)
		}
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO `+s.schema+`.property_sizes
			(property_id, land_size, house_size, living_size, total_area, usable_area, balcony_area)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (property_id) DO UPDATE SET
			land_size    = EXCLUDED.land_size,
			house_size   = EXCLUDED.house_size,
			living_size  = EXCLUDED.living_size,
			total_area   = EXCLUDED.total_area,
			usable_area  = EXCLUDED.usable_area,
			balcony_area = EXCLUDED.balcony_area`,
		propertyID, sizes.LandSize, sizes.HouseSize, sizes.LivingSize,
		sizes.TotalArea, sizes.UsableArea, sizes.BalconyArea,
	)
	if err != nil {
		return fmt.Errorf("save sizes: %w", err)
	}
	return nil
}

// SaveRooms upserts the Rooms attribute-group row in property_rooms.
// nil rooms → DELETE the row.
func (s *PropertyService) SaveRooms(ctx context.Context, propertyID string, rooms *Rooms) error {
	if s.pool == nil {
		return nil
	}
	if rooms == nil {
		if _, err := s.pool.Exec(ctx,
			`DELETE FROM `+s.schema+`.property_rooms WHERE property_id = $1`,
			propertyID); err != nil {
			return fmt.Errorf("save rooms: delete: %w", err)
		}
		return nil
	}
	// property_rooms columns are NOT NULL DEFAULT 0 in the BR schema (bootstrap
	// migration). The Go model uses *int for room counts to distinguish "unknown"
	// from "zero", but the DB doesn't support that — so we coalesce nil → 0 at
	// the write boundary. Reading back via the LEFT JOIN still produces *int (0)
	// rather than nil, which is consistent with the legacy data shape.
	_, err := s.pool.Exec(ctx,
		`INSERT INTO `+s.schema+`.property_rooms
			(property_id, bedrooms, bathrooms, kitchens, living_rooms, dining_rooms, offices, storage_rooms, maid_rooms, guest_rooms)
		 VALUES ($1, COALESCE($2::int, 0), COALESCE($3::int, 0), COALESCE($4::int, 0), COALESCE($5::int, 0),
			COALESCE($6::int, 0), COALESCE($7::int, 0), COALESCE($8::int, 0), COALESCE($9::int, 0), COALESCE($10::int, 0))
		 ON CONFLICT (property_id) DO UPDATE SET
			bedrooms      = COALESCE(EXCLUDED.bedrooms, 0),
			bathrooms     = COALESCE(EXCLUDED.bathrooms, 0),
			kitchens      = COALESCE(EXCLUDED.kitchens, 0),
			living_rooms  = COALESCE(EXCLUDED.living_rooms, 0),
			dining_rooms  = COALESCE(EXCLUDED.dining_rooms, 0),
			offices       = COALESCE(EXCLUDED.offices, 0),
			storage_rooms = COALESCE(EXCLUDED.storage_rooms, 0),
			maid_rooms    = COALESCE(EXCLUDED.maid_rooms, 0),
			guest_rooms   = COALESCE(EXCLUDED.guest_rooms, 0)`,
		propertyID, rooms.Bedrooms, rooms.Bathrooms, rooms.Kitchens, rooms.LivingRooms,
		rooms.DiningRooms, rooms.Offices, rooms.StorageRooms, rooms.MaidRooms, rooms.GuestRooms,
	)
	if err != nil {
		return fmt.Errorf("save rooms: %w", err)
	}
	return nil
}

// LoadBuildingSpecs fetches the property_building_specs row. A missing row (or nil
// pool) yields nil, nil — the backward-compat contract the public renderer depends on
// (no row -> no Building Specifications section). Companion-loaded (NOT the generated
// mapper) to keep the mapper LEFT-JOIN surface bounded.
func (s *PropertyService) LoadBuildingSpecs(ctx context.Context, propertyID string) (*BuildingSpecs, error) {
	if s.pool == nil {
		return nil, nil
	}
	var b BuildingSpecs
	err := s.pool.QueryRow(ctx,
		`SELECT floors, floor_level, parking_spaces, year_built, last_renovated
		   FROM `+s.schema+`.property_building_specs WHERE property_id = $1`,
		propertyID).Scan(&b.Floors, &b.FloorLevel, &b.ParkingSpaces, &b.YearBuilt, &b.LastRenovated)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load building specs: %w", err)
	}
	return &b, nil
}

// SaveBuildingSpecs upserts the property_building_specs row; nil specs → DELETE
// (intent: "this property has no building spec data").
func (s *PropertyService) SaveBuildingSpecs(ctx context.Context, propertyID string, specs *BuildingSpecs) error {
	if s.pool == nil {
		return nil
	}
	if specs == nil {
		if _, err := s.pool.Exec(ctx,
			`DELETE FROM `+s.schema+`.property_building_specs WHERE property_id = $1`,
			propertyID); err != nil {
			return fmt.Errorf("save building specs: delete: %w", err)
		}
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO `+s.schema+`.property_building_specs
			(property_id, floors, floor_level, parking_spaces, year_built, last_renovated)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (property_id) DO UPDATE SET
			floors         = EXCLUDED.floors,
			floor_level    = EXCLUDED.floor_level,
			parking_spaces = EXCLUDED.parking_spaces,
			year_built     = EXCLUDED.year_built,
			last_renovated = EXCLUDED.last_renovated`,
		propertyID, specs.Floors, specs.FloorLevel, specs.ParkingSpaces, specs.YearBuilt, specs.LastRenovated,
	)
	if err != nil {
		return fmt.Errorf("save building specs: %w", err)
	}
	return nil
}

// LoadPolicies fetches the property_policies row. A missing row (or nil pool) yields
// nil, nil (backward-compat: no Lease Terms & Policies section). The three TEXT[]
// columns map to []string natively via pgx — companion-loaded since the codegen
// mapper cannot emit arrays (mirrors the Tags exception).
func (s *PropertyService) LoadPolicies(ctx context.Context, propertyID string) (*Policies, error) {
	if s.pool == nil {
		return nil, nil
	}
	var p Policies
	err := s.pool.QueryRow(ctx,
		`SELECT inclusions, restrictions, house_rules, minimum_lease_months, security_deposit_months, advance_payment_months
		   FROM `+s.schema+`.property_policies WHERE property_id = $1`,
		propertyID).Scan(&p.Inclusions, &p.Restrictions, &p.HouseRules,
		&p.MinimumLeaseMonths, &p.SecurityDepositMonths, &p.AdvancePaymentMonths)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load policies: %w", err)
	}
	return &p, nil
}

// SavePolicies upserts the property_policies row; nil policies → DELETE. The slice
// fields are written as TEXT[] (nil slice -> '{}' via COALESCE, matching the column
// default), so a round-trip never returns a nil-vs-empty distinction for the arrays.
func (s *PropertyService) SavePolicies(ctx context.Context, propertyID string, pol *Policies) error {
	if s.pool == nil {
		return nil
	}
	if pol == nil {
		if _, err := s.pool.Exec(ctx,
			`DELETE FROM `+s.schema+`.property_policies WHERE property_id = $1`,
			propertyID); err != nil {
			return fmt.Errorf("save policies: delete: %w", err)
		}
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO `+s.schema+`.property_policies
			(property_id, inclusions, restrictions, house_rules, minimum_lease_months, security_deposit_months, advance_payment_months)
		 VALUES ($1, COALESCE($2::text[], '{}'), COALESCE($3::text[], '{}'), COALESCE($4::text[], '{}'), $5, $6, $7)
		 ON CONFLICT (property_id) DO UPDATE SET
			inclusions              = COALESCE(EXCLUDED.inclusions, '{}'),
			restrictions            = COALESCE(EXCLUDED.restrictions, '{}'),
			house_rules             = COALESCE(EXCLUDED.house_rules, '{}'),
			minimum_lease_months    = EXCLUDED.minimum_lease_months,
			security_deposit_months = EXCLUDED.security_deposit_months,
			advance_payment_months  = EXCLUDED.advance_payment_months`,
		propertyID, pol.Inclusions, pol.Restrictions, pol.HouseRules,
		pol.MinimumLeaseMonths, pol.SecurityDepositMonths, pol.AdvancePaymentMonths,
	)
	if err != nil {
		return fmt.Errorf("save policies: %w", err)
	}
	return nil
}

// =============================================================================
// UNITS — 1:many containment child (companion-persisted; NOT the generated mapper).
// Per-row CRUD (LoadUnits / SaveUnit / DeleteUnit) in the transaction_party 1:many
// style — NOT the property's 1:1 attribute-group upsert. Units cascade-delete with
// the property at the DB layer (FK ON DELETE CASCADE).
// =============================================================================

// LoadUnits returns all units for a property, ordered by sort_order then title.
// A nil pool (fixture mode) yields nil, nil. Read-only — never mutates.
func (s *PropertyService) LoadUnits(ctx context.Context, propertyID string) ([]PropertyUnit, error) {
	if s.pool == nil {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, property_id, title, description, cover_image_id,
		        sale_price, lease_price, price_currency, is_active, disabled, sort_order,
		        land_size, house_size, living_size, total_area, usable_area, balcony_area,
		        bedrooms, bathrooms, kitchens, living_rooms, dining_rooms, offices,
		        storage_rooms, maid_rooms, guest_rooms,
		        created_at, updated_at
		   FROM `+s.schema+`.property_units
		  WHERE property_id = $1
		  ORDER BY sort_order, title`,
		propertyID)
	if err != nil {
		return nil, fmt.Errorf("load units: %w", err)
	}
	defer rows.Close()

	var units []PropertyUnit
	for rows.Next() {
		var u PropertyUnit
		// Scan the flat size/room columns into value-typed groups, then collapse an
		// all-NULL group back to a nil pointer (IsZero) so a unit without size/room data
		// round-trips to nil — mirroring how Property's mapper LEFT JOIN yields nil for
		// an absent sub-row.
		var sizes Sizes
		var rooms Rooms
		if err := rows.Scan(&u.ID, &u.PropertyID, &u.Title, &u.Description, &u.CoverImageID,
			&u.SalePrice, &u.LeasePrice, &u.PriceCurrency, &u.IsActive, &u.Disabled, &u.SortOrder,
			&sizes.LandSize, &sizes.HouseSize, &sizes.LivingSize,
			&sizes.TotalArea, &sizes.UsableArea, &sizes.BalconyArea,
			&rooms.Bedrooms, &rooms.Bathrooms, &rooms.Kitchens, &rooms.LivingRooms,
			&rooms.DiningRooms, &rooms.Offices, &rooms.StorageRooms, &rooms.MaidRooms, &rooms.GuestRooms,
			&u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("load units: scan: %w", err)
		}
		if !sizes.IsZero() {
			u.Sizes = &sizes
		}
		if !rooms.IsZero() {
			u.Rooms = &rooms
		}
		units = append(units, u)
	}
	return units, rows.Err()
}

// SaveUnit validates and upserts a single unit (INSERT … ON CONFLICT (id) DO UPDATE),
// per-row 1:many style. Validation runs FIRST (so the gate is exercised in fixture
// mode); an id is assigned for new units. A nil pool returns the unit with its
// assigned id/timestamps without persisting (fixture echo). The caller sets
// u.PropertyID (the parent) and u.IsActive (new units default active at the API layer).
func (s *PropertyService) SaveUnit(ctx context.Context, u PropertyUnit) (PropertyUnit, error) {
	if err := u.Validate(); err != nil {
		return PropertyUnit{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	now := time.Now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	u.UpdatedAt = now
	if s.pool == nil {
		return u, nil
	}
	// Flatten the size/room groups into nil-safe per-column locals: a nil group writes
	// NULL to every one of its columns; a set field writes its (possibly nil) value.
	var (
		landSize, houseSize, livingSize    *float64
		totalArea, usableArea, balconyArea *float64
	)
	if u.Sizes != nil {
		landSize, houseSize, livingSize = u.Sizes.LandSize, u.Sizes.HouseSize, u.Sizes.LivingSize
		totalArea, usableArea, balconyArea = u.Sizes.TotalArea, u.Sizes.UsableArea, u.Sizes.BalconyArea
	}
	var (
		bedrooms, bathrooms, kitchens, livingRooms *int
		diningRooms, offices, storageRooms         *int
		maidRooms, guestRooms                      *int
	)
	if u.Rooms != nil {
		bedrooms, bathrooms, kitchens, livingRooms = u.Rooms.Bedrooms, u.Rooms.Bathrooms, u.Rooms.Kitchens, u.Rooms.LivingRooms
		diningRooms, offices, storageRooms = u.Rooms.DiningRooms, u.Rooms.Offices, u.Rooms.StorageRooms
		maidRooms, guestRooms = u.Rooms.MaidRooms, u.Rooms.GuestRooms
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO `+s.schema+`.property_units
			(id, property_id, title, description, cover_image_id, sale_price, lease_price,
			 price_currency, is_active, sort_order, created_at, updated_at, disabled,
			 land_size, house_size, living_size, total_area, usable_area, balcony_area,
			 bedrooms, bathrooms, kitchens, living_rooms, dining_rooms, offices,
			 storage_rooms, maid_rooms, guest_rooms)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
			 $14, $15, $16, $17, $18, $19,
			 $20, $21, $22, $23, $24, $25, $26, $27, $28)
		 ON CONFLICT (id) DO UPDATE SET
			title          = EXCLUDED.title,
			description    = EXCLUDED.description,
			cover_image_id = EXCLUDED.cover_image_id,
			sale_price     = EXCLUDED.sale_price,
			lease_price    = EXCLUDED.lease_price,
			price_currency = EXCLUDED.price_currency,
			is_active      = EXCLUDED.is_active,
			disabled       = EXCLUDED.disabled,
			sort_order     = EXCLUDED.sort_order,
			updated_at     = EXCLUDED.updated_at,
			land_size      = EXCLUDED.land_size,
			house_size     = EXCLUDED.house_size,
			living_size    = EXCLUDED.living_size,
			total_area     = EXCLUDED.total_area,
			usable_area    = EXCLUDED.usable_area,
			balcony_area   = EXCLUDED.balcony_area,
			bedrooms       = EXCLUDED.bedrooms,
			bathrooms      = EXCLUDED.bathrooms,
			kitchens       = EXCLUDED.kitchens,
			living_rooms   = EXCLUDED.living_rooms,
			dining_rooms   = EXCLUDED.dining_rooms,
			offices        = EXCLUDED.offices,
			storage_rooms  = EXCLUDED.storage_rooms,
			maid_rooms     = EXCLUDED.maid_rooms,
			guest_rooms    = EXCLUDED.guest_rooms`,
		u.ID, u.PropertyID, u.Title, u.Description, u.CoverImageID, u.SalePrice, u.LeasePrice,
		u.PriceCurrency, u.IsActive, u.SortOrder, u.CreatedAt, u.UpdatedAt, u.Disabled,
		landSize, houseSize, livingSize, totalArea, usableArea, balconyArea,
		bedrooms, bathrooms, kitchens, livingRooms, diningRooms, offices,
		storageRooms, maidRooms, guestRooms)
	if err != nil {
		return PropertyUnit{}, fmt.Errorf("save unit: %w", err)
	}
	return u, nil
}

// DeleteUnit removes a single unit by id (hard delete — units have no soft-delete).
// A nil pool (fixture mode) is a no-op; a missing row is not an error (idempotent).
func (s *PropertyService) DeleteUnit(ctx context.Context, unitID string) error {
	if s.pool == nil {
		return nil
	}
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM `+s.schema+`.property_units WHERE id = $1`, unitID); err != nil {
		return fmt.Errorf("delete unit: %w", err)
	}
	return nil
}

// =============================================================================
// IMAGE COLLECTIONS — 1:many containment child (companion-persisted; NOT the
// generated mapper). A named, ordered subset of the property's All-Images pool
// (property_images); members are pkg/image references via the
// image_collection_items join. Per-row CRUD (LoadImageCollections /
// SaveImageCollection / DeleteImageCollection / SetCollectionItems) in the
// PropertyUnit 1:many style. Collections cascade-delete with the property at the
// DB layer (FK ON DELETE CASCADE); item.image_id is FK RESTRICT so deleting a
// collection detaches members only — the pool image survives.
// =============================================================================

// EnsureDefaultCollection guarantees the property has exactly one default
// "Gallery" collection, self-healing the creation gap: PropertyService.Create
// never seeds one, and the migration backfill only seeded properties that had
// pool images AT migration time — so a property created fresh, or one with no
// images then, can have NO default, and the admin media panel (which renders the
// Gallery card from the collections list) then shows no curatable Gallery.
//
// It is an idempotent INSERT … ON CONFLICT DO NOTHING against the partial unique
// index uniq_image_collection_default (property_id) WHERE is_default, mirroring
// the migration's seed (name "Gallery", is_default = true, sort_order 0). A
// concurrent double-create cannot 500: the loser's insert is absorbed by ON
// CONFLICT DO NOTHING (the conflict target is the partial index's predicate), so
// the property still ends with exactly one default. Returns whether a row was
// inserted (true) vs already present (false).
//
// This is a WRITE and MUST be called only on the auth-gated manage path (the
// admin collections List), never on the public/anonymous read path — those read
// via LoadImageCollections + DefaultCollection and degrade gracefully (nil
// default → no primary gallery) without writing. A nil pool (fixture mode) is a
// no-op (false, nil).
func (s *PropertyService) EnsureDefaultCollection(ctx context.Context, propertyID string) (bool, error) {
	if s.pool == nil {
		return false, nil
	}
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO `+s.schema+`.image_collections
			(id, property_id, name, is_default, sort_order)
		 VALUES ($1, $2, $3, true, 0)
		 ON CONFLICT (property_id) WHERE is_default DO NOTHING`,
		uuid.NewString(), propertyID, DefaultCollectionName)
	if err != nil {
		return false, fmt.Errorf("ensure default collection: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// UnitBelongsToProperty reports whether unit unitID exists AND belongs to propertyID —
// the cross-entity invariant the collection Create path checks before scoping a gallery
// to a unit, so a unit gallery can never target a unit of a DIFFERENT property (→ 422).
// A missing or foreign unit returns false, nil. A nil pool (fixture mode) returns
// false, nil: there is no DB to verify against, so ownership is denied (the safe
// default). Mirrors deal.Orchestrator.UnitBelongsToProperty — the shared service is the
// proper domain home for a pure property/unit ownership check (see api-chi/notes.md C1).
func (s *PropertyService) UnitBelongsToProperty(ctx context.Context, unitID, propertyID string) (bool, error) {
	if s.pool == nil {
		return false, nil
	}
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (
		    SELECT 1 FROM `+s.schema+`.property_units
		     WHERE id = $1 AND property_id = $2)`,
		unitID, propertyID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("unit belongs to property: %w", err)
	}
	return exists, nil
}

// LoadImageCollections returns all collections for a property, ordered by
// is_default DESC (the default "Gallery" first) then sort_order then name, each
// with its ordered membership (items by item sort_order). The joined image
// projection (url) is included for one-round-trip render; a
// soft-deleted image is filtered out (defense in depth — item.image_id is
// RESTRICT). A nil pool (fixture mode) yields nil, nil. Read-only.
func (s *PropertyService) LoadImageCollections(ctx context.Context, propertyID string) ([]ImageCollection, error) {
	if s.pool == nil {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, property_id, name, is_default, sort_order, created_at, updated_at
		   FROM `+s.schema+`.image_collections
		  WHERE property_id = $1
		  ORDER BY is_default DESC, sort_order, name`,
		propertyID)
	if err != nil {
		return nil, fmt.Errorf("load image collections: %w", err)
	}
	defer rows.Close()

	var collections []ImageCollection
	for rows.Next() {
		var c ImageCollection
		if err := rows.Scan(&c.ID, &c.PropertyID, &c.Name, &c.IsDefault, &c.SortOrder,
			&c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("load image collections: scan: %w", err)
		}
		c.Items = []ImageCollectionItem{}
		collections = append(collections, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load image collections: %w", err)
	}

	// Per-collection ordered membership (one query, grouped by collection_id).
	for i := range collections {
		items, err := s.loadCollectionItems(ctx, collections[i].ID)
		if err != nil {
			return nil, err
		}
		collections[i].Items = items
	}
	return collections, nil
}

// loadCollectionItems returns one collection's ordered membership with the joined
// image projection. Internal helper for LoadImageCollections + the post-write
// reload in SetCollectionItems.
func (s *PropertyService) loadCollectionItems(ctx context.Context, collectionID string) ([]ImageCollectionItem, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT ci.image_id, ci.sort_order, i.url, COALESCE(i.alt_text, '')
		   FROM `+s.schema+`.image_collection_items ci
		   JOIN `+s.schema+`.images i ON i.id = ci.image_id
		  WHERE ci.collection_id = $1 AND i.deleted_at IS NULL
		  ORDER BY ci.sort_order`,
		collectionID)
	if err != nil {
		return nil, fmt.Errorf("load collection items: %w", err)
	}
	defer rows.Close()

	items := []ImageCollectionItem{}
	for rows.Next() {
		var it ImageCollectionItem
		if err := rows.Scan(&it.ImageID, &it.SortOrder, &it.URL, &it.AltText); err != nil {
			return nil, fmt.Errorf("load collection items: scan: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// SaveImageCollection validates and upserts a single collection's METADATA
// (name / is_default / sort_order) — INSERT … ON CONFLICT (id) DO UPDATE, per-row
// 1:many style. Membership is managed separately via SetCollectionItems (the join
// is its own write path). Validation runs FIRST (so the gate is exercised in
// fixture mode); an id is assigned for new collections. A nil pool returns the
// collection with its assigned id/timestamps without persisting (fixture echo).
// The caller sets c.PropertyID (the parent). The returned collection's Items are
// whatever the caller passed in (membership is not written here); callers that
// need the persisted membership reload via LoadImageCollections.
func (s *PropertyService) SaveImageCollection(ctx context.Context, c ImageCollection) (ImageCollection, error) {
	if err := c.Validate(); err != nil {
		return ImageCollection{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	if s.pool == nil {
		return c, nil
	}
	// Albums are PROPERTY-owned (Decision #0317) — no unit_id column; a unit surfaces an
	// album via unit_collection_links (LinkUnitCollection), never by ownership here.
	_, err := s.pool.Exec(ctx,
		`INSERT INTO `+s.schema+`.image_collections
			(id, property_id, name, is_default, sort_order, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (id) DO UPDATE SET
			name       = EXCLUDED.name,
			sort_order = EXCLUDED.sort_order,
			updated_at = EXCLUDED.updated_at`,
		c.ID, c.PropertyID, c.Name, c.IsDefault, c.SortOrder, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return ImageCollection{}, fmt.Errorf("save image collection: %w", err)
	}
	return c, nil
}

// LoadUnitCollectionLinks returns all (unit, album) links for a property — the unit<->album
// many-to-many of Decision #0317. Scoped through the property's units (a link row has no
// property_id of its own), ordered by unit then sort_order for stable rendering. A nil pool
// (fixture mode) yields nil, nil. Read-only.
func (s *PropertyService) LoadUnitCollectionLinks(ctx context.Context, propertyID string) ([]UnitCollectionLink, error) {
	if s.pool == nil {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT l.unit_id, l.collection_id, l.sort_order
		   FROM `+s.schema+`.unit_collection_links l
		   JOIN `+s.schema+`.property_units u ON u.id = l.unit_id
		  WHERE u.property_id = $1
		  ORDER BY l.unit_id, l.sort_order`,
		propertyID)
	if err != nil {
		return nil, fmt.Errorf("load unit collection links: %w", err)
	}
	defer rows.Close()

	var links []UnitCollectionLink
	for rows.Next() {
		var l UnitCollectionLink
		if err := rows.Scan(&l.UnitID, &l.CollectionID, &l.SortOrder); err != nil {
			return nil, fmt.Errorf("load unit collection links: scan: %w", err)
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

// LinkUnitCollection links a property album to a unit (Decision #0317) — an idempotent INSERT …
// ON CONFLICT (unit_id, collection_id) DO NOTHING that appends the link after the unit's existing
// ones (sort_order = MAX+1). The caller pre-verifies the unit and the collection belong to the
// SAME property (UnitBelongsToProperty + the collection's property_id) so a forged cross-property
// id is a clean 422, never an FK 500. A nil pool (fixture mode) is a no-op.
func (s *PropertyService) LinkUnitCollection(ctx context.Context, unitID, collectionID string) error {
	if s.pool == nil {
		return nil
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO `+s.schema+`.unit_collection_links (id, unit_id, collection_id, sort_order)
		 SELECT $1, $2, $3, COALESCE(MAX(sort_order) + 1, 0)
		   FROM `+s.schema+`.unit_collection_links WHERE unit_id = $2
		 ON CONFLICT (unit_id, collection_id) DO NOTHING`,
		uuid.NewString(), unitID, collectionID); err != nil {
		return fmt.Errorf("link unit collection: %w", err)
	}
	return nil
}

// UnlinkUnitCollection removes ONE (unit, album) link (Decision #0317) — the album survives (it
// is property-owned); only this unit stops surfacing it. Idempotent (a missing link is not an
// error). A nil pool (fixture mode) is a no-op.
func (s *PropertyService) UnlinkUnitCollection(ctx context.Context, unitID, collectionID string) error {
	if s.pool == nil {
		return nil
	}
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM `+s.schema+`.unit_collection_links WHERE unit_id = $1 AND collection_id = $2`,
		unitID, collectionID); err != nil {
		return fmt.Errorf("unlink unit collection: %w", err)
	}
	return nil
}

// DeleteImageCollection removes a single collection by id. The membership rows in
// image_collection_items cascade-delete with the collection (FK ON DELETE
// CASCADE), so members are DETACHED — the underlying pool images are never touched
// (item.image_id is FK RESTRICT, but the cascade is on the collection_id side).
// Deleting the DEFAULT collection is REJECTED here (the one-default-per-property
// invariant) — the handler maps that to 409. A nil pool (fixture mode) is a no-op;
// a missing row is not an error (idempotent).
func (s *PropertyService) DeleteImageCollection(ctx context.Context, collectionID string) error {
	if s.pool == nil {
		return nil
	}
	var isDefault bool
	err := s.pool.QueryRow(ctx,
		`SELECT is_default FROM `+s.schema+`.image_collections WHERE id = $1`,
		collectionID).Scan(&isDefault)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // idempotent — already gone
	}
	if err != nil {
		return fmt.Errorf("delete image collection: lookup: %w", err)
	}
	if isDefault {
		return fmt.Errorf("%w: the default collection cannot be deleted", ErrValidation)
	}
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM `+s.schema+`.image_collections WHERE id = $1`, collectionID); err != nil {
		return fmt.Errorf("delete image collection: %w", err)
	}
	return nil
}

// SetCollectionItems replaces a collection's membership with the ordered set of
// imageIDs (array position = sort_order, normalized 0..n-1). It is the
// drag-from-pool + reorder write path (PUT …/items): a transactional
// DELETE-all-then-INSERT, mirroring the gallery SetImages contract. The caller
// pre-verifies each image_id exists in the property's pool (clean 422 instead of
// an FK violation). Returns the reloaded collection (metadata + joined membership)
// so the response reflects the persisted server order — never the optimistic
// client order. A nil pool (fixture mode) returns nil, nil (no membership to load).
func (s *PropertyService) SetCollectionItems(ctx context.Context, collectionID string, imageIDs []string) (*ImageCollection, error) {
	if s.pool == nil {
		return nil, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("set collection items: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM `+s.schema+`.image_collection_items WHERE collection_id = $1`,
		collectionID); err != nil {
		return nil, fmt.Errorf("set collection items: clear: %w", err)
	}
	for i, imageID := range imageIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO `+s.schema+`.image_collection_items (id, collection_id, image_id, sort_order)
			 VALUES ($1, $2, $3, $4)`,
			uuid.NewString(), collectionID, imageID, i); err != nil {
			return nil, fmt.Errorf("set collection items: insert %s: %w", imageID, err)
		}
	}
	if _, err := tx.Exec(ctx,
		`UPDATE `+s.schema+`.image_collections SET updated_at = now() WHERE id = $1`,
		collectionID); err != nil {
		return nil, fmt.Errorf("set collection items: touch: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("set collection items: commit: %w", err)
	}

	// Reload the persisted collection (metadata + server-ordered membership).
	return s.getImageCollection(ctx, collectionID)
}

// ReorderImageCollections normalizes the display order of a property's collections
// to match the given id sequence (array position = sort_order, 0..n-1). Ids not
// belonging to the property are ignored (the UPDATE is scoped by property_id).
// Returns the reloaded, reordered collection set. A nil pool (fixture mode) yields
// nil, nil.
func (s *PropertyService) ReorderImageCollections(ctx context.Context, propertyID string, orderedIDs []string) ([]ImageCollection, error) {
	if s.pool == nil {
		return nil, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("reorder image collections: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	for i, id := range orderedIDs {
		if _, err := tx.Exec(ctx,
			`UPDATE `+s.schema+`.image_collections
			    SET sort_order = $1, updated_at = now()
			  WHERE id = $2 AND property_id = $3`,
			i, id, propertyID); err != nil {
			return nil, fmt.Errorf("reorder image collections: update %s: %w", id, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("reorder image collections: commit: %w", err)
	}
	return s.LoadImageCollections(ctx, propertyID)
}

// getImageCollection loads a single collection (metadata + ordered membership) by
// id. Returns repository.ErrNotFound when the row is absent so the handler maps it
// to 404. Internal helper for the write paths that reload a single collection.
func (s *PropertyService) getImageCollection(ctx context.Context, collectionID string) (*ImageCollection, error) {
	var c ImageCollection
	err := s.pool.QueryRow(ctx,
		`SELECT id, property_id, name, is_default, sort_order, created_at, updated_at
		   FROM `+s.schema+`.image_collections WHERE id = $1`,
		collectionID).Scan(&c.ID, &c.PropertyID, &c.Name, &c.IsDefault, &c.SortOrder,
		&c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get image collection: %w", err)
	}
	items, err := s.loadCollectionItems(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	c.Items = items
	return &c, nil
}

// GetImageCollection is the exported single-collection read (metadata + ordered
// membership) used by handlers after a metadata write (POST/PATCH) to return the
// full collection shape. A nil pool (fixture mode) yields nil, nil.
func (s *PropertyService) GetImageCollection(ctx context.Context, collectionID string) (*ImageCollection, error) {
	if s.pool == nil {
		return nil, nil
	}
	return s.getImageCollection(ctx, collectionID)
}

// =============================================================================
// PROPERTY DOCUMENTS — companion-loaded child (delegates to DocumentStore)
// =============================================================================
//
// Mirrors the image-collection companion methods: the service is the single
// domain entry-point handlers call; the row CRUD lives in DocumentStore (no
// storage dependency). The R2 object lifecycle (upload, delete-on-row-delete) is
// the handler's — it holds the image.ObjectStore — so these methods touch ONLY
// the row, exactly like LoadImageCollections / SaveImageCollection.

// LoadPropertyDocuments returns a property's documents ordered by
// (visibility, sort_order). When publicOnly is true the read is the public-listing
// projection (visibility = public only); PRIVATE documents never enter that
// result. A nil pool (fixture mode) yields a non-nil empty slice. Read-only.
func (s *PropertyService) LoadPropertyDocuments(ctx context.Context, propertyID string, publicOnly bool) ([]PropertyDocument, error) {
	// Guard on the CONCRETE pool: passing a typed-nil *pgxpool.Pool into the store's
	// pgxQuerier interface field would make the store's `pool == nil` check false
	// (a non-nil interface wrapping a nil pointer), so the nil check must be here.
	if s.pool == nil {
		return NewDocumentStore(nil, s.schema).Load(ctx, propertyID, publicOnly)
	}
	return NewDocumentStore(s.pool, s.schema).Load(ctx, propertyID, publicOnly)
}

// GetPropertyDocument returns a single document scoped to its property, or nil when
// absent (missing / wrong-property / soft-deleted). Used to 404 cleanly before a
// PATCH/DELETE and to resolve the R2 key on delete. A nil pool yields nil, nil.
func (s *PropertyService) GetPropertyDocument(ctx context.Context, propertyID, docID string) (*PropertyDocument, error) {
	if s.pool == nil {
		return nil, nil
	}
	return NewDocumentStore(s.pool, s.schema).Get(ctx, propertyID, docID)
}

// SavePropertyDocument validates then upserts a single document row. Validation
// runs FIRST (so the gate is exercised in fixture mode); the caller has already
// uploaded the R2 object and set URL. A nil pool returns the document with its
// assigned id/timestamps without persisting (fixture echo).
func (s *PropertyService) SavePropertyDocument(ctx context.Context, d PropertyDocument) (PropertyDocument, error) {
	if s.pool == nil {
		return NewDocumentStore(nil, s.schema).Save(ctx, nil, d)
	}
	return NewDocumentStore(s.pool, s.schema).Save(ctx, s.pool, d)
}

// DeletePropertyDocument soft-deletes one document ROW scoped to its property
// (idempotent — a missing row is a no-op). It deletes ONLY the row; the handler
// deletes the R2 object best-effort (it owns the ObjectStore). A nil pool is a
// no-op.
func (s *PropertyService) DeletePropertyDocument(ctx context.Context, propertyID, docID string) error {
	if s.pool == nil {
		return nil
	}
	return NewDocumentStore(s.pool, s.schema).Delete(ctx, s.pool, propertyID, docID)
}

// EnableDocumentShare turns on anyone-with-link sharing for one document (scope-07):
// mint a fresh bearer token, store it, and return the updated row (nil when the
// document is absent / wrong-property). Re-enabling resets the link (old token dies).
// The concrete-pool guard mirrors LoadPropertyDocuments (a typed-nil pool wrapped in
// the store's interface would defeat the store's own nil check).
func (s *PropertyService) EnableDocumentShare(ctx context.Context, propertyID, docID string) (*PropertyDocument, error) {
	if s.pool == nil {
		return NewDocumentStore(nil, s.schema).EnableShare(ctx, propertyID, docID)
	}
	return NewDocumentStore(s.pool, s.schema).EnableShare(ctx, propertyID, docID)
}

// DisableDocumentShare revokes link sharing (NULLs the token) for one document —
// idempotent (a missing / already-off row is a no-op). A nil pool is a no-op.
func (s *PropertyService) DisableDocumentShare(ctx context.Context, propertyID, docID string) error {
	if s.pool == nil {
		return nil
	}
	return NewDocumentStore(s.pool, s.schema).DisableShare(ctx, propertyID, docID)
}

// GetDocumentByShareToken resolves a public share token to its document across any
// property (nil when unknown / revoked / soft-deleted). A nil pool yields nil, nil.
func (s *PropertyService) GetDocumentByShareToken(ctx context.Context, token string) (*PropertyDocument, error) {
	if s.pool == nil {
		return nil, nil
	}
	return NewDocumentStore(s.pool, s.schema).GetByShareToken(ctx, token)
}

// =============================================================================
// PROPERTY NOTES — companion-loaded child (delegates to NoteStore)
// =============================================================================
//
// Mirrors the document companion methods: the service is the single domain
// entry-point handlers call; the row CRUD lives in NoteStore. Notes are append +
// delete only (no edit / bulk-save) with NO min-one invariant (a property may have
// zero notes) — see note_store.go. The nil-pool guard is on the CONCRETE pool (a
// typed-nil *pgxpool.Pool wrapped in NoteStore's pgxQuerier interface would defeat
// the store's own `pool == nil` check), exactly as LoadPropertyDocuments.

// LoadNotes returns each property's notes keyed by property id, ordered oldest-first
// (newest last); a property with no notes is absent from the map. A nil pool yields
// an empty map. Read-only.
func (s *PropertyService) LoadNotes(ctx context.Context, propertyIDs ...string) (map[string][]Note, error) {
	if s.pool == nil {
		return NewNoteStore(nil, s.schema).Load(ctx, propertyIDs...)
	}
	return NewNoteStore(s.pool, s.schema).Load(ctx, propertyIDs...)
}

// AddNote appends one note to a property and returns the created Note (with its
// server-assigned id + created_at). A blank body is rejected with ErrEmptyNote (the
// handler maps it to 400). In fixture mode (nil pool) the blank-body gate still runs
// and a non-blank body is echoed without persisting (no id/created_at).
func (s *PropertyService) AddNote(ctx context.Context, propertyID, body string) (Note, error) {
	if s.pool == nil {
		if strings.TrimSpace(body) == "" {
			return Note{}, ErrEmptyNote
		}
		return Note{PropertyID: propertyID, Body: strings.TrimSpace(body)}, nil
	}
	return NewNoteStore(s.pool, s.schema).Add(ctx, s.pool, propertyID, body)
}

// DeleteNote removes one note from a property (idempotent — a missing note is a
// no-op, never an error). A nil pool is a no-op.
func (s *PropertyService) DeleteNote(ctx context.Context, propertyID, noteID string) error {
	if s.pool == nil {
		return nil
	}
	return NewNoteStore(s.pool, s.schema).Delete(ctx, s.pool, propertyID, noteID)
}

// LoadTags fetches the custom tag labels for a property from the properties.tags
// TEXT[] column (Decision #0279 PT). pgx maps TEXT[] -> []string natively, so no
// pq.Array wrapper is needed. The column is intentionally NOT part of the generated
// mapper (the codegen has no scalar-array support — see pkg/property/tags.go), so
// this companion read uses a direct column query. A missing row yields empty
// (callers validate existence via repo.Get before loading attributes).
func (s *PropertyService) LoadTags(ctx context.Context, propertyID string) ([]string, error) {
	if s.pool == nil {
		return nil, nil
	}
	var tags []string
	err := s.pool.QueryRow(ctx,
		`SELECT tags FROM `+s.schema+`.properties WHERE id = $1`,
		propertyID).Scan(&tags)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load tags: %w", err)
	}
	return tags, nil
}

// SaveTags writes the normalized custom tag labels to the properties.tags TEXT[]
// column. Normalization (trim, drop-empty, dedupe-by-slug, cap) happens here so every
// write path is uniform. nil/empty normalizes to '{}' — the column is NOT NULL.
func (s *PropertyService) SaveTags(ctx context.Context, propertyID string, tags []string) error {
	if s.pool == nil {
		return nil
	}
	normalized := NormalizeTags(tags)
	if normalized == nil {
		normalized = []string{}
	}
	if _, err := s.pool.Exec(ctx,
		`UPDATE `+s.schema+`.properties SET tags = $1 WHERE id = $2`,
		normalized, propertyID); err != nil {
		return fmt.Errorf("save tags: %w", err)
	}
	return nil
}

// LoadTagSlugs reads the materialized unified tag-slug identity (custom-tag slugs ∪
// app-derived autotag slugs) from the properties.tag_slugs TEXT[] column (2606-001).
// pgx maps TEXT[] -> []string natively, so no wrapper is needed; the column is NOT
// part of the generated mapper (codegen has no scalar-array support), so this
// companion read mirrors LoadTags. The column is app-maintained — written ONLY by
// saveTagSlugs via the Create/Update deriver hook, never by SQL triggers — so a read
// reflects the last full-aggregate save. A missing row or nil pool yields nil.
func (s *PropertyService) LoadTagSlugs(ctx context.Context, propertyID string) ([]string, error) {
	if s.pool == nil {
		return nil, nil
	}
	var slugs []string
	err := s.pool.QueryRow(ctx,
		`SELECT tag_slugs FROM `+s.schema+`.properties WHERE id = $1`,
		propertyID).Scan(&slugs)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load tag_slugs: %w", err)
	}
	return slugs, nil
}

// saveTagSlugs writes the app-derived unified tag-slug identity to the
// properties.tag_slugs TEXT[] column (2606-001). The slugs are written VERBATIM — the
// injected deriver already produces TagSlug-normalized, deduped, ordered values, so
// (unlike SaveTags) no re-normalization happens here. nil/empty writes '{}' (the
// column is NOT NULL DEFAULT '{}'), never SQL NULL. Unexported: the only writers are
// the Create/Update hook calls — there is no standalone tag_slugs endpoint (the column
// is a projection of the full aggregate, mutated only via a full property save).
func (s *PropertyService) saveTagSlugs(ctx context.Context, propertyID string, slugs []string) error {
	if s.pool == nil {
		return nil
	}
	if slugs == nil {
		slugs = []string{}
	}
	if _, err := s.pool.Exec(ctx,
		`UPDATE `+s.schema+`.properties SET tag_slugs = $1 WHERE id = $2`,
		slugs, propertyID); err != nil {
		return fmt.Errorf("save tag_slugs: %w", err)
	}
	return nil
}

// LoadSeoMeta fetches the per-listing SEO/OG overrides from the properties.seo_meta
// JSONB column (FF3a). The column is intentionally NOT part of the generated mapper
// (the codegen scans `&p.Field`; for a *seo.SeoMeta that is a **SeoMeta pgx cannot
// Scan into the type's sql.Scanner) — so this companion read mirrors LoadTags. A
// NULL column or a missing row yields nil (no overrides); callers validate existence
// via repo.Get before loading attributes. seo.SeoMeta.Scan turns SQL NULL into the
// zero value, so a stored payload that round-trips to empty is normalized to nil.
func (s *PropertyService) LoadSeoMeta(ctx context.Context, propertyID string) (*seo.SeoMeta, error) {
	if s.pool == nil {
		return nil, nil
	}
	var meta seo.SeoMeta
	err := s.pool.QueryRow(ctx,
		`SELECT seo_meta FROM `+s.schema+`.properties WHERE id = $1`,
		propertyID).Scan(&meta)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load seo_meta: %w", err)
	}
	if meta.IsZero() {
		return nil, nil
	}
	return &meta, nil
}

// SaveSeoMeta writes the per-listing SEO/OG overrides to the properties.seo_meta
// JSONB column. A nil pointer (or a pointer to an entirely-empty SeoMeta) writes SQL
// NULL via seo.SeoMeta.Value — so clearing the overrides removes the JSONB object
// rather than storing "{}". Mirrors SaveTags: one UPDATE against the properties row,
// uniform across every write path. Caller is responsible for SeoMeta.Validate (the
// API boundary rejects over-length meta_title/meta_description with 422 before here).
//
// Unlike SaveTags, this is the surgical, standalone seo-only write path used by the
// admin SEO facet (PATCH .../seo) — so a missing property id (0 rows affected) is a
// repository.ErrNotFound the handler maps to 404, rather than a silent no-op. The
// full-object Update already validates existence via the mapper, so this guard only
// matters for the dedicated endpoint.
func (s *PropertyService) SaveSeoMeta(ctx context.Context, propertyID string, meta *seo.SeoMeta) error {
	if s.pool == nil {
		return nil
	}
	var value seo.SeoMeta
	if meta != nil {
		value = *meta
	}
	// seo.SeoMeta.Value() returns nil (SQL NULL) for the zero value; passing the value
	// (not a pointer) lets pgx invoke driver.Valuer to get NULL-or-JSONB uniformly.
	tag, err := s.pool.Exec(ctx,
		`UPDATE `+s.schema+`.properties SET seo_meta = $1 WHERE id = $2`,
		value, propertyID)
	if err != nil {
		return fmt.Errorf("save seo_meta: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func normalizeFurnished(p *Property) {
	if p.Furnished != nil && *p.Furnished == "" {
		p.Furnished = nil
	}
}

func validateCreate(p Property) error {
	if p.Title == nil || strings.TrimSpace(*p.Title) == "" {
		return fmt.Errorf("%w: title is required", ErrValidation)
	}
	if err := p.ValidateListingIntent(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := p.ValidatePrices(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := validateAddressGeometry(p.Address); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := validateListingAttrs(p); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	return nil
}

// validateListingAttrs enforces the L1 superset wire-boundary rules (M1): the five
// dictionary discriminators must be seeded codes when set (a parse-time 422 guard —
// the DB FK enforces the same but surfaces as a 500), and the BuildingSpecs / Policies
// ranges must hold. Shared by create + update.
func validateListingAttrs(p Property) error {
	if p.Ownership != nil && !p.Ownership.Valid() {
		return fmt.Errorf("invalid ownership_type %q", *p.Ownership)
	}
	if p.Condition != nil && !p.Condition.Valid() {
		return fmt.Errorf("invalid condition %q", *p.Condition)
	}
	if p.Direction != nil && !p.Direction.Valid() {
		return fmt.Errorf("invalid direction %q", *p.Direction)
	}
	if p.RoadAccess != nil && !p.RoadAccess.Valid() {
		return fmt.Errorf("invalid road_access %q", *p.RoadAccess)
	}
	if p.Topography != nil && !p.Topography.Valid() {
		return fmt.Errorf("invalid topography %q", *p.Topography)
	}
	if err := p.BuildingSpecs.Validate(); err != nil {
		return err
	}
	return p.Policies.Validate()
}

func validateUpdate(p Property) error {
	if p.Title != nil && strings.TrimSpace(*p.Title) == "" {
		return fmt.Errorf("%w: title must not be empty", ErrValidation)
	}
	if err := p.ValidateListingIntent(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := p.ValidatePrices(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := validateAddressGeometry(p.Address); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := validateListingAttrs(p); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	return nil
}

// validateAddressGeometry enforces the geometry sub-invariants on Address —
// coord pair + polygon shape — WITHOUT requiring postal completeness
// (street/city/country). BR allows draft properties with minimal addresses,
// so the full Address.Validate() is too strict here; only the
// never-correct-to-skip geometry invariants are enforced. MapView viewport
// state is cosmetic and is intentionally NOT validated — bad zoom/tilt/etc
// degrade gracefully at the map SDK layer.
func validateAddressGeometry(addr Address) error {
	return addr.ValidateGeometry()
}
