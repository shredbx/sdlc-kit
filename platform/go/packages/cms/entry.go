// CmsEntry is the BR-local domain for the shared content collection that powers
// BOTH the public Guides knowledge base AND the Services pages, discriminated by
// a `kind` field (guide | service) with typed per-kind extension fields. One
// `content_entries` table serves both kinds (Option C — shared core + typed
// per-kind extension; entity.yml).
//
// CmsEntry is BR-project-scoped (not a shared sbx-core primitive), so it lives
// in this internal/cms package alongside cms.Page and REUSES the package's Slug
// type + ErrValidation. The struct, its validation, and the postgres mapper are
// a projection of entities/cms-entry/platforms/api-chi/md.yml — keep the column
// list here in exact sync with that md.yml and the content_entries migration.
//
// SEARCH reuses the generic postgres searcher (postgres.NewPostgresSearcher
// [cms.CmsEntry]) — the EntryPostgresMapper implements postgres.TextSearchable so
// the 3-tier FTS pipeline matches p.search_vector (trigger-maintained, Decision
// #0272 weighting) and fuzzy/ilike on p.title.
package cms

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shredbx/sbx-core/pkg/repository"
	"github.com/shredbx/sbx-core/pkg/repository/postgres"
	"github.com/shredbx/sbx-core/pkg/seo"
)

// =============================================================================
// ENTRY KIND — the collection discriminator (a named type, never a raw string)
// =============================================================================

// EntryKind discriminates a content_entries row. It is a STRUCTURAL enum (not a
// user-managed dictionary — adding a kind is a code + schema decision), mirroring
// cms.Slug: the guide|service invariant travels with the value.
type EntryKind string

const (
	// EntryKindGuide is a knowledge-base article (public /guides surface).
	EntryKindGuide EntryKind = "guide"
	// EntryKindService is a services page entry (public /services surface).
	EntryKindService EntryKind = "service"
	// EntryKindBlog is a blog post (public /lab/blog surface, shredbx). Like a
	// guide it MAY carry a category; BR never emits this kind, so BR is unaffected.
	EntryKindBlog EntryKind = "blog"
)

// String renders the kind for SQL params and routing.
func (k EntryKind) String() string { return string(k) }

// Validate enforces the entity.yml `kind` set: exactly guide | service | blog.
func (k EntryKind) Validate() error {
	switch k {
	case EntryKindGuide, EntryKindService, EntryKindBlog:
		return nil
	default:
		return fmt.Errorf("kind must be one of guide|service|blog, got %q", string(k))
	}
}

// =============================================================================
// ENTRY DETAILS — typed structured JSONB payload (per-kind extension)
// =============================================================================

// IncludedItem is one "What's included" check-card on a service detail page.
type IncludedItem struct {
	Title       string `json:"title,omitempty" yaml:"title,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// CtaConfig is the primary contact call-to-action for a service.
type CtaConfig struct {
	Label string `json:"label,omitempty" yaml:"label,omitempty"`
	Href  string `json:"href,omitempty" yaml:"href,omitempty"`
	Phone string `json:"phone,omitempty" yaml:"phone,omitempty"`
}

// ServiceDetails is the service-only structured content (services do not search
// on these, so they live un-indexed in the typed JSONB `details` column).
type ServiceDetails struct {
	IncludedItems []IncludedItem `json:"included_items,omitempty" yaml:"included_items,omitempty"`
	Cta           *CtaConfig     `json:"cta,omitempty" yaml:"cta,omitempty"`
}

// IsZero reports whether the service block carries no data.
func (s *ServiceDetails) IsZero() bool {
	return s == nil || (len(s.IncludedItems) == 0 && s.Cta == nil)
}

// MediaDetails is optional media attachments (guides today; extensible to any
// kind). NOT a discriminated "media" type — per BR rule each facet is its own
// typed list (images are images, video links are links).
type MediaDetails struct {
	Gallery []string `json:"gallery,omitempty" yaml:"gallery,omitempty"`
	Reels   []string `json:"reels,omitempty" yaml:"reels,omitempty"`
	Videos  []string `json:"videos,omitempty" yaml:"videos,omitempty"`
}

// IsZero reports whether the media block carries no data.
func (m *MediaDetails) IsZero() bool {
	return m == nil || (len(m.Gallery) == 0 && len(m.Reels) == 0 && len(m.Videos) == 0)
}

// EntryDetails is the OPTIONAL typed structured payload on a CmsEntry, persisted
// as a single JSONB column. It is a TYPED value (never map[string]any): adding a
// future per-kind section is an additive struct field, not a migration. JSONB
// round-trip mirrors cms.PageDetails exactly — a zero EntryDetails Values to SQL
// NULL, and a NULL/empty/"null" source Scans back to zero.
type EntryDetails struct {
	Service *ServiceDetails `json:"service,omitempty" yaml:"service,omitempty"`
	Media   *MediaDetails   `json:"media,omitempty" yaml:"media,omitempty"`
}

// IsZero reports whether the details carry no structured data (→ stored as NULL).
func (d EntryDetails) IsZero() bool {
	return d.Service.IsZero() && d.Media.IsZero()
}

// Value implements driver.Valuer for the JSONB column. A zero EntryDetails
// returns nil → SQL NULL (no empty blob persisted).
func (d EntryDetails) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}
	return json.Marshal(d)
}

// Scan implements sql.Scanner. Accepts []byte (pgx default) or string; a NULL or
// empty/`null` source yields a zero EntryDetails, not an error.
func (d *EntryDetails) Scan(src any) error {
	if src == nil {
		*d = EntryDetails{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("EntryDetails.Scan: unsupported source type %T", src)
	}
	if len(data) == 0 || string(data) == "null" {
		*d = EntryDetails{}
		return nil
	}
	return json.Unmarshal(data, d)
}

// =============================================================================
// CMSENTRY — the entity
// =============================================================================

// CmsEntry is one row in the shared content_entries collection. Document-type:
// UUID identity, timestamps, soft delete, optimistic version. Field order in the
// struct is cosmetic; the storage column order is fixed by
// EntryPostgresMapper.Columns()/FromRow().
type CmsEntry struct {
	ID string `json:"id" yaml:"id"`

	// Shared core (every entry).
	Slug          Slug      `json:"slug" yaml:"slug"`
	Kind          EntryKind `json:"kind" yaml:"kind"`
	Category      *string   `json:"category,omitempty" yaml:"category,omitempty"`
	Title         string    `json:"title" yaml:"title"`
	CoverImageURL *string   `json:"cover_image_url,omitempty" yaml:"cover_image_url,omitempty"`
	Excerpt       *string   `json:"excerpt,omitempty" yaml:"excerpt,omitempty"`
	BodyMarkdown  *string   `json:"body_markdown,omitempty" yaml:"body_markdown,omitempty"`
	Published     bool      `json:"published" yaml:"published"`
	SortOrder     int       `json:"sort_order" yaml:"sort_order"`

	// Tags — content-level (ANY kind, Decision #0279 M-A): trimmed labels, slug-
	// identity matched, feeding the search_vector at weight A + the shared `tags` dict.
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	// Guide-only typed extension (real columns; false/NULL for services).
	Featured bool    `json:"featured" yaml:"featured"`
	Author   *string `json:"author,omitempty" yaml:"author,omitempty"`

	// Details is the optional typed structured payload (JSONB). Zero value
	// round-trips as SQL NULL.
	Details EntryDetails `json:"details,omitempty" yaml:"details,omitempty"`

	// SeoMeta is the optional per-entry SEO + Open Graph override payload
	// (SVC-ADMIN, 2607-007 — parity with CmsPage.SeoMeta / property.seo_meta).
	// The SHARED seo.SeoMeta VALUE type: this mapper is hand-written, so the value
	// field scans directly (&e.SeoMeta is a valid sql.Scanner — same rationale as
	// CmsPage). Zero value persists as SQL NULL, never "{}".
	SeoMeta seo.SeoMeta `json:"seo_meta,omitempty" yaml:"seo_meta,omitempty"`

	Version int `json:"version" yaml:"version"`

	CreatedBy *string `json:"created_by,omitempty" yaml:"created_by,omitempty"`
	UpdatedBy *string `json:"updated_by,omitempty" yaml:"updated_by,omitempty"`

	CreatedAt time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" yaml:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty"`
}

// maxEntryBodyLen is the entity.yml `body_markdown` maxLength constraint.
const maxEntryBodyLen = 100000

// Validate enforces the entity.yml constraints: slug required + kebab-case; kind
// in {guide,service,blog}; title required (non-empty trimmed); body_markdown (when
// present) within the 100k cap; category is meaningful for guides and blog posts
// (a service carrying a category is rejected — services have no taxonomy). Every
// failure wraps ErrValidation so callers test with errors.Is(err, ErrValidation).
func (e CmsEntry) Validate() error {
	if err := e.Slug.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := e.Kind.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if strings.TrimSpace(e.Title) == "" {
		return fmt.Errorf("%w: title is required", ErrValidation)
	}
	if e.BodyMarkdown != nil && len(*e.BodyMarkdown) > maxEntryBodyLen {
		return fmt.Errorf("%w: body_markdown exceeds maximum length", ErrValidation)
	}
	// Category is a taxonomy for guides and blog posts only — a service must not
	// carry one (the DB FK + this rule both enforce it; the rule gives a clear 422
	// before the write).
	if e.Kind != EntryKindGuide && e.Kind != EntryKindBlog && e.Category != nil && strings.TrimSpace(*e.Category) != "" {
		return fmt.Errorf("%w: category applies to guides and blog posts only", ErrValidation)
	}
	// SERP length invariants propagate (same as CmsPage): meta_title <=60,
	// meta_description <=160 — enforced at the write boundary, not just the editor.
	if err := e.SeoMeta.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	return nil
}

// =============================================================================
// POSTGRES MAPPER — projection of entities/cms-entry/platforms/api-chi/md.yml
// =============================================================================

// Fields exposes typed, compile-time-safe field descriptors for list filtering +
// ordering (e.g. cms.EntryFields.Kind.Eq("guide")).
var EntryFields = struct {
	ID        repository.Field[string]
	Kind      repository.Field[string]
	Slug      repository.Field[string]
	Category  repository.Field[string]
	Published repository.Field[bool]
	Featured  repository.Field[bool]
	SortOrder repository.Field[int]
	DeletedAt repository.Field[string]
	CreatedAt repository.Field[string]
}{
	ID:        repository.Field[string]{Name: "id"},
	Kind:      repository.Field[string]{Name: "kind"},
	Slug:      repository.Field[string]{Name: "slug"},
	Category:  repository.Field[string]{Name: "category"},
	Published: repository.Field[bool]{Name: "published"},
	Featured:  repository.Field[bool]{Name: "featured"},
	SortOrder: repository.Field[int]{Name: "sort_order"},
	DeletedAt: repository.Field[string]{Name: "deleted_at"},
	CreatedAt: repository.Field[string]{Name: "created_at"},
}

// Compile-time checks: EntryPostgresMapper implements postgres.Mapper[CmsEntry]
// and the optional postgres.TextSearchable seam.
var (
	_ postgres.Mapper[CmsEntry] = EntryPostgresMapper{}
	_ postgres.TextSearchable   = EntryPostgresMapper{}
)

// EntryPostgresMapper maps CmsEntry to the {schema}.content_entries table.
type EntryPostgresMapper struct {
	schema string
}

// NewEntryPostgresMapper creates a mapper scoped to the given PostgreSQL schema.
func NewEntryPostgresMapper(schema string) EntryPostgresMapper {
	return EntryPostgresMapper{schema: schema}
}

// TableName returns the fully qualified primary table for INSERT/UPDATE/DELETE.
func (m EntryPostgresMapper) TableName() string {
	return m.schema + ".content_entries"
}

// SelectFrom aliases the primary table `p` — Columns() prefixes every field with
// the "p" alias (so p.search_vector resolves in the generic searcher's tiers).
func (m EntryPostgresMapper) SelectFrom() string {
	return m.TableName() + " p"
}

// Columns returns the SELECT column list. Order MUST exactly match FromRow scan.
// search_vector is intentionally OMITTED — it is trigger-maintained and never
// read into the model.
func (m EntryPostgresMapper) Columns() []string {
	return []string{
		"p.id",
		"p.slug",
		"p.kind",
		"p.category",
		"p.title",
		"p.cover_image_url",
		"p.excerpt",
		"p.body_markdown",
		"p.published",
		"p.sort_order",
		"p.tags",
		"p.featured",
		"p.author",
		"p.details",
		"p.seo_meta",
		"p.version",
		"p.created_by",
		"p.updated_by",
		"p.created_at",
		"p.updated_at",
		"p.deleted_at",
	}
}

// FieldColumn maps logical field names to aliased columns for WHERE/ORDER.
func (m EntryPostgresMapper) FieldColumn(field string) string {
	switch field {
	case "id":
		return "p.id"
	case "slug":
		return "p.slug"
	case "kind":
		return "p.kind"
	case "category":
		return "p.category"
	case "published":
		return "p.published"
	case "featured":
		return "p.featured"
	case "sort_order":
		return "p.sort_order"
	case "created_at":
		return "p.created_at"
	case "deleted_at":
		return "p.deleted_at"
	default:
		return field
	}
}

// ToRow converts a CmsEntry to a column→value map for INSERT/UPDATE. deleted_at
// and search_vector are OMITTED: deleted_at defaults NULL; search_vector is
// maintained by the content_entries_search_vector_update trigger (never written
// here). tags is stored as a Postgres text[] (pgx maps []string natively);
// details is the EntryDetails Valuer. slug + kind store their string forms.
func (m EntryPostgresMapper) ToRow(e CmsEntry) (map[string]any, error) {
	return map[string]any{
		"id":              e.ID,
		"slug":            e.Slug.String(),
		"kind":            e.Kind.String(),
		"category":        e.Category,
		"title":           e.Title,
		"cover_image_url": e.CoverImageURL,
		"excerpt":         e.Excerpt,
		"body_markdown":   e.BodyMarkdown,
		"published":       e.Published,
		"sort_order":      e.SortOrder,
		"tags":            e.Tags,
		"featured":        e.Featured,
		"author":          e.Author,
		"details":         e.Details,
		"seo_meta":        e.SeoMeta,
		"version":         e.Version,
		"created_by":      e.CreatedBy,
		"updated_by":      e.UpdatedBy,
		"created_at":      e.CreatedAt,
		"updated_at":      e.UpdatedAt,
	}, nil
}

// FromRow scans a row into a CmsEntry. Column order MUST exactly match Columns().
func (m EntryPostgresMapper) FromRow(scan func(dest ...any) error) (CmsEntry, error) {
	var e CmsEntry
	err := scan(
		&e.ID,
		&e.Slug,
		&e.Kind,
		&e.Category,
		&e.Title,
		&e.CoverImageURL,
		&e.Excerpt,
		&e.BodyMarkdown,
		&e.Published,
		&e.SortOrder,
		&e.Tags,
		&e.Featured,
		&e.Author,
		&e.Details,
		&e.SeoMeta,
		&e.Version,
		&e.CreatedBy,
		&e.UpdatedBy,
		&e.CreatedAt,
		&e.UpdatedAt,
		&e.DeletedAt,
	)
	if err != nil {
		return e, err
	}
	return e, nil
}

// SearchVectorColumn implements postgres.TextSearchable — the Tier-1 prefix
// tsquery matches the trigger-maintained content_entries.search_vector.
func (m EntryPostgresMapper) SearchVectorColumn() string { return "p.search_vector" }

// FuzzyTextFields implements postgres.TextSearchable — Tier-2 fuzzy + Tier-3
// ILIKE scan the title (the only free-text column worth a typo/substring match;
// category + tags already ride weight A in the tsvector).
func (m EntryPostgresMapper) FuzzyTextFields() []string { return []string{"p.title"} }
