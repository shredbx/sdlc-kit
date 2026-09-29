// Persistence for CmsPage — ported from bestierealestate's own
// internal/repository/cmspage.go (CmsPageRepository), merged into this package
// rather than kept as a separate one: the original split it out only because the
// entity type and its repository lived in different Go packages (pkg/cms vs the
// app's own internal/repository); here both already live in the same package, so
// the split has no purpose. Reads delegate to the generic
// postgres.PostgresStore[CmsPage] (persistence/repository/postgres — the same
// engine any other entity's mapper plugs into). Repository adds the two
// slug-keyed operations the generic store cannot express:
//
//   - GetBySlug — fetch the single live (deleted_at IS NULL) row for a slug;
//     repository.ErrNotFound when absent.
//   - UpsertBySlug — create-on-first-edit: insert if the slug has no live row,
//     else update its title + body_markdown + details + section content +
//     published flag, bump version, and stamp updated_by. One statement
//     (INSERT … ON CONFLICT … DO UPDATE) against the partial unique index
//     uq_cms_pages_slug keeps the operation atomic and race-free. Decision #0281
//     (AE4) collapsed the former separate Publish step into this single direct-save.
package cms

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shredbx/sbx-core/pkg/database"
	"github.com/shredbx/sbx-core/pkg/repository"
	"github.com/shredbx/sbx-core/pkg/repository/postgres"
	"github.com/shredbx/sbx-core/pkg/seo"
)

// Repository persists CmsPage rows for one schema.
type Repository struct {
	store  *postgres.PostgresStore[CmsPage]
	pool   *pgxpool.Pool
	schema string
}

// NewRepository wires the generic store + the schema-scoped pool used for the
// slug-keyed get + upsert.
func NewRepository(db *database.DB, schema string) *Repository {
	mapper := NewPostgresMapper(schema)
	return &Repository{
		store:  postgres.NewPostgresStore[CmsPage](db, mapper),
		pool:   db.Pool(),
		schema: schema,
	}
}

// Get fetches one page by id. Returns repository.ErrNotFound if absent.
func (r *Repository) Get(ctx context.Context, id string) (CmsPage, error) {
	return r.store.Get(ctx, id)
}

// GetBySlug fetches the single live page for a slug (deleted_at IS NULL).
// Returns repository.ErrNotFound when no live row exists for the slug.
func (r *Repository) GetBySlug(ctx context.Context, slug Slug) (CmsPage, error) {
	if err := slug.Validate(); err != nil {
		return CmsPage{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	mapper := NewPostgresMapper(r.schema)
	sql := `SELECT ` + joinColumns(mapper.Columns()) + ` FROM ` + mapper.SelectFrom() +
		` WHERE c.slug = $1 AND c.deleted_at IS NULL LIMIT 1`
	row := r.pool.QueryRow(ctx, sql, slug.String())
	p, err := mapper.FromRow(row.Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CmsPage{}, repository.NewNotFoundError(r.schema+".cms_pages", slug.String())
		}
		return CmsPage{}, fmt.Errorf("cms get-by-slug %q: %w", slug.String(), err)
	}
	return p, nil
}

// List returns every live (deleted_at IS NULL) page, ordered by slug. Full rows,
// same shape as Get — bos-demo's page count doesn't warrant a separate summary
// projection or pagination yet (add both when a real consumer's page count needs
// it, not speculatively). Backs the admin Pages list, which had no source of truth
// after the flat pages model (with its own hardcoded slug list) was replaced by
// this kit — the real bestierealestate app avoids needing this by fixing its pages
// through a SITE_SURFACES registry; bos-demo's dynamic page model has no equivalent.
func (r *Repository) List(ctx context.Context) ([]CmsPage, error) {
	mapper := NewPostgresMapper(r.schema)
	sql := `SELECT ` + joinColumns(mapper.Columns()) + ` FROM ` + mapper.SelectFrom() +
		` WHERE c.deleted_at IS NULL ORDER BY c.slug`
	rows, err := r.pool.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("cms list: %w", err)
	}
	defer rows.Close()

	pages := make([]CmsPage, 0)
	for rows.Next() {
		p, err := mapper.FromRow(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("cms list: %w", err)
		}
		pages = append(pages, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cms list: %w", err)
	}
	return pages, nil
}

// UpsertBySlug creates the page on first edit, or updates an existing live row's
// title + body_markdown + details + section content + published flag — bumping
// version and stamping updated_by. The actorID is the authenticated user; pass ""
// when unknown (stored NULL). Returns the saved page.
//
// INSERT … ON CONFLICT (slug) WHERE deleted_at IS NULL DO UPDATE targets the
// partial unique index uq_cms_pages_slug, so concurrent first-edits collapse to a
// single row instead of racing two INSERTs. version starts at 1 on insert and is
// incremented (cms_pages.version + 1) on update. created_by is set only on insert;
// updated_by is set on both paths. The DB trigger maintains updated_at.
//
// Field presence governs EVERY field — a partial save NEVER wipes an untouched one:
//   - title / body_markdown:  a nil pointer (the field was not part of this save)
//     COALESCE-keeps the stored value; a present value (incl. "" to clear) replaces
//     it. So a markdown-only edit keeps the title, a hero edit keeps the body, and
//     an SEO edit keeps both. (SQL NULL can't be SET via this path — "" is the
//     empty form; no caller needs a literal NULL.)
//   - replaceDetails / details:  true → EXCLUDED.details replaces wholesale (an
//     empty PageDetails → NULL, so a fully-cleared hero persists); false (details
//     nil — a markdown-only edit) → COALESCE keeps the stored block.
//   - replaceContent / content:  the structured-section list, written straight to
//     published_content (the public-served slot — there is no separate publish
//     step). true → EXCLUDED.published_content replaces (a present-but-empty
//     SectionList persists as "no sections" / "[]"); false (content nil — a
//     markdown-only / simple-page save) → COALESCE keeps the stored content so a
//     non-section edit never wipes a manager's sections.
//   - published (*bool):         the live gate. Non-nil SETS it; nil (the toggle
//     was not part of this save — e.g. a simple About/Terms edit) KEEPS the stored
//     value, so such a save never silently unpublishes a live page. On INSERT a nil
//     published defaults to false (a new page is a draft).
//   - replaceSeoMeta / seoMeta:  the per-page SEO/OG override block (seo.SeoMeta),
//     SAME field-presence semantics as details — true → EXCLUDED.seo_meta replaces
//     wholesale (an empty SeoMeta → NULL, so cleared overrides persist); false
//     (seoMeta nil — a save that didn't touch SEO) → COALESCE keeps the stored block.
//   - layout (*string):          a scalar like title/body_markdown — nil COALESCE-
//     keeps the stored value; a present value (incl. "") replaces it. On INSERT a
//     nil layout falls back to the column default ("default") via COALESCE.
//
// draft_content is not written here (retained on the entity for the expand phase
// only, dropped in the original's own #0281 cleanup — see cmspage.go's doc comment).
func (r *Repository) UpsertBySlug(ctx context.Context, slug Slug, title, bodyMarkdown *string, details *PageDetails, replaceDetails bool, content *SectionList, replaceContent bool, published *bool, seoMeta *seo.SeoMeta, replaceSeoMeta bool, layout *string, actorID string) (CmsPage, error) {
	if err := slug.Validate(); err != nil {
		return CmsPage{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	// Enforce the entity invariants before touching the DB (body length cap +
	// contact-address geometry + section validity) so the upsert path can't bypass
	// entity validation. The section content validates as PublishedContent (its
	// storage slot under direct-save).
	check := CmsPage{Slug: slug, Title: title, BodyMarkdown: bodyMarkdown}
	if details != nil {
		check.Details = *details
	}
	if content != nil {
		check.PublishedContent = *content
	}
	if seoMeta != nil {
		check.SeoMeta = *seoMeta
	}
	if err := check.Validate(); err != nil {
		return CmsPage{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	var actor *string
	if actorID != "" {
		actor = &actorID
	}

	// detailsArg / contentArg are the JSONB values. A nil pointer (caller did not
	// touch the block) Values to SQL NULL; the DO UPDATE then COALESCE-preserves the
	// stored value (see the field-presence doc above). contentArg targets
	// published_content (direct-save).
	var detailsArg any
	if details != nil {
		detailsArg = *details
	}
	var contentArg any
	if content != nil {
		contentArg = *content
	}
	var seoMetaArg any
	if seoMeta != nil {
		seoMetaArg = *seoMeta
	}

	// RETURNING reads the affected row directly off the INSERT … ON CONFLICT — a
	// single atomic statement, no second SELECT. cms_pages has no joined tables, so
	// the RETURNING column list is Columns() with the "c." alias stripped — same
	// order, same FromRow scan. `published` ($9) is passed as a *bool: pgx maps nil
	// → SQL NULL (keep stored on update / default false on insert) and a non-nil
	// pointer → the boolean.
	mapper := NewPostgresMapper(r.schema)
	sql := `INSERT INTO ` + r.schema + `.cms_pages (slug, title, body_markdown, details, published_content, published, seo_meta, version, created_by, updated_by, layout)
	        VALUES ($1, $2, $3, $4, $7, COALESCE($9, false), $10, 1, $5, $5, COALESCE($12, 'default'))
	        ON CONFLICT (slug) WHERE deleted_at IS NULL
	        DO UPDATE SET
	            title             = COALESCE(EXCLUDED.title, ` + r.schema + `.cms_pages.title),
	            body_markdown     = COALESCE(EXCLUDED.body_markdown, ` + r.schema + `.cms_pages.body_markdown),
	            details           = CASE WHEN $6 THEN EXCLUDED.details
	                                     ELSE COALESCE(EXCLUDED.details, ` + r.schema + `.cms_pages.details) END,
	            published_content = CASE WHEN $8 THEN EXCLUDED.published_content
	                                     ELSE COALESCE(EXCLUDED.published_content, ` + r.schema + `.cms_pages.published_content) END,
	            published         = CASE WHEN $9 IS NULL THEN ` + r.schema + `.cms_pages.published
	                                     ELSE $9 END,
	            seo_meta          = CASE WHEN $11 THEN EXCLUDED.seo_meta
	                                     ELSE COALESCE(EXCLUDED.seo_meta, ` + r.schema + `.cms_pages.seo_meta) END,
	            layout            = COALESCE($12, ` + r.schema + `.cms_pages.layout),
	            version           = ` + r.schema + `.cms_pages.version + 1,
	            updated_by        = EXCLUDED.updated_by
	        RETURNING ` + joinColumns(unaliasColumns(mapper.Columns()))

	row := r.pool.QueryRow(ctx, sql, slug.String(), title, bodyMarkdown, detailsArg, actor, replaceDetails, contentArg, replaceContent, published, seoMetaArg, replaceSeoMeta, layout)
	p, err := mapper.FromRow(row.Scan)
	if err != nil {
		return CmsPage{}, fmt.Errorf("cms upsert-by-slug %q: %w", slug.String(), err)
	}
	return p, nil
}

// joinColumns joins aliased column names for a SELECT list.
func joinColumns(cols []string) string {
	return strings.Join(cols, ", ")
}

// unaliasColumns strips a leading "<alias>." from each column for use in a
// single-table RETURNING (where the table alias is not in scope). Safe only for
// single-table mappers like cms_pages — joined SELECTs must keep their aliases.
func unaliasColumns(cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		if dot := strings.IndexByte(c, '.'); dot >= 0 {
			out[i] = c[dot+1:]
		} else {
			out[i] = c
		}
	}
	return out
}
