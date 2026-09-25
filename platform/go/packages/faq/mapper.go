// MD layer — postgres mappers for the 3 FAQ tables. Each mapper is a flat projection
// of its PD struct (faq.go) onto {schema}.faq_{categories,steps,items}, mirroring
// pkg/cms EntryPostgresMapper: TableName/SelectFrom/Columns/FieldColumn/ToRow/FromRow
// + repository.Field[T] descriptors for the Store's list filtering/ordering.
//
// Column order MUST stay in exact sync with FromRow's scan destinations (the
// round-trip test in mapper_test.go locks this) and with the faq_content_model
// migration. deleted_at is OMITTED from ToRow (defaults NULL; never written here);
// status + slug store their string forms; seo stores the seo.SeoMeta value (a
// driver.Valuer → JSONB, zero value → NULL).
package faq

import (
	"github.com/shredbx/sbx-core/pkg/repository"
	"github.com/shredbx/sbx-core/pkg/repository/postgres"
)

// Compile-time checks: each mapper satisfies postgres.Mapper[T].
var (
	_ postgres.Mapper[Category] = CategoryPostgresMapper{}
	_ postgres.Mapper[Step]     = StepPostgresMapper{}
	_ postgres.Mapper[Item]     = ItemPostgresMapper{}
)

// =============================================================================
// CATEGORY — {schema}.faq_categories
// =============================================================================

// CategoryFields exposes typed field descriptors for list filtering/ordering
// (e.g. faq.CategoryFields.Status.Eq("published")).
var CategoryFields = struct {
	ID        repository.Field[string]
	Slug      repository.Field[string]
	Status    repository.Field[string]
	Position  repository.Field[int]
	Featured  repository.Field[bool]
	DeletedAt repository.Field[string]
	CreatedAt repository.Field[string]
}{
	ID:        repository.Field[string]{Name: "id"},
	Slug:      repository.Field[string]{Name: "slug"},
	Status:    repository.Field[string]{Name: "status"},
	Position:  repository.Field[int]{Name: "position"},
	Featured:  repository.Field[bool]{Name: "is_featured"},
	DeletedAt: repository.Field[string]{Name: "deleted_at"},
	CreatedAt: repository.Field[string]{Name: "created_at"},
}

// CategoryPostgresMapper maps Category to {schema}.faq_categories.
type CategoryPostgresMapper struct{ schema string }

// NewCategoryPostgresMapper creates a mapper scoped to the given PostgreSQL schema.
func NewCategoryPostgresMapper(schema string) CategoryPostgresMapper {
	return CategoryPostgresMapper{schema: schema}
}

func (m CategoryPostgresMapper) TableName() string  { return m.schema + ".faq_categories" }
func (m CategoryPostgresMapper) SelectFrom() string { return m.TableName() + " p" }

func (m CategoryPostgresMapper) Columns() []string {
	return []string{
		"p.id", "p.slug", "p.title", "p.blurb", "p.position", "p.is_featured",
		"p.seo", "p.status", "p.version", "p.created_by", "p.updated_by",
		"p.created_at", "p.updated_at", "p.deleted_at",
	}
}

func (m CategoryPostgresMapper) FieldColumn(field string) string {
	switch field {
	case "id", "slug", "title", "blurb", "position", "is_featured", "seo",
		"status", "version", "created_by", "updated_by", "created_at", "updated_at", "deleted_at":
		return "p." + field
	default:
		return field
	}
}

func (m CategoryPostgresMapper) ToRow(c Category) (map[string]any, error) {
	return map[string]any{
		"id":          c.ID,
		"slug":        c.Slug.String(),
		"title":       c.Title,
		"blurb":       c.Blurb,
		"position":    c.Position,
		"is_featured": c.Featured,
		"seo":         c.SEO,
		"status":      c.Status.String(),
		"version":     c.Version,
		"created_by":  c.CreatedBy,
		"updated_by":  c.UpdatedBy,
		"created_at":  c.CreatedAt,
		"updated_at":  c.UpdatedAt,
	}, nil
}

func (m CategoryPostgresMapper) FromRow(scan func(dest ...any) error) (Category, error) {
	var c Category
	err := scan(
		&c.ID, &c.Slug, &c.Title, &c.Blurb, &c.Position, &c.Featured, &c.SEO,
		&c.Status, &c.Version, &c.CreatedBy, &c.UpdatedBy, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	if err != nil {
		return c, err
	}
	return c, nil
}

// =============================================================================
// STEP — {schema}.faq_steps
// =============================================================================

var StepFields = struct {
	ID         repository.Field[string]
	CategoryID repository.Field[string]
	Position   repository.Field[int]
	Status     repository.Field[string]
	DeletedAt  repository.Field[string]
}{
	ID:         repository.Field[string]{Name: "id"},
	CategoryID: repository.Field[string]{Name: "category_id"},
	Position:   repository.Field[int]{Name: "position"},
	Status:     repository.Field[string]{Name: "status"},
	DeletedAt:  repository.Field[string]{Name: "deleted_at"},
}

// StepPostgresMapper maps Step to {schema}.faq_steps.
type StepPostgresMapper struct{ schema string }

func NewStepPostgresMapper(schema string) StepPostgresMapper {
	return StepPostgresMapper{schema: schema}
}

func (m StepPostgresMapper) TableName() string  { return m.schema + ".faq_steps" }
func (m StepPostgresMapper) SelectFrom() string { return m.TableName() + " p" }

func (m StepPostgresMapper) Columns() []string {
	return []string{
		"p.id", "p.category_id", "p.position", "p.title", "p.description", "p.status",
		"p.version", "p.created_by", "p.updated_by", "p.created_at", "p.updated_at", "p.deleted_at",
	}
}

func (m StepPostgresMapper) FieldColumn(field string) string {
	switch field {
	case "id", "category_id", "position", "title", "description", "status",
		"version", "created_by", "updated_by", "created_at", "updated_at", "deleted_at":
		return "p." + field
	default:
		return field
	}
}

func (m StepPostgresMapper) ToRow(s Step) (map[string]any, error) {
	return map[string]any{
		"id":          s.ID,
		"category_id": s.CategoryID,
		"position":    s.Position,
		"title":       s.Title,
		"description": s.Description,
		"status":      s.Status.String(),
		"version":     s.Version,
		"created_by":  s.CreatedBy,
		"updated_by":  s.UpdatedBy,
		"created_at":  s.CreatedAt,
		"updated_at":  s.UpdatedAt,
	}, nil
}

func (m StepPostgresMapper) FromRow(scan func(dest ...any) error) (Step, error) {
	var s Step
	err := scan(
		&s.ID, &s.CategoryID, &s.Position, &s.Title, &s.Description, &s.Status,
		&s.Version, &s.CreatedBy, &s.UpdatedBy, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
	)
	if err != nil {
		return s, err
	}
	return s, nil
}

// =============================================================================
// ITEM — {schema}.faq_items
// =============================================================================

var ItemFields = struct {
	ID        repository.Field[string]
	StepID    repository.Field[string]
	Position  repository.Field[int]
	Status    repository.Field[string]
	DeletedAt repository.Field[string]
}{
	ID:        repository.Field[string]{Name: "id"},
	StepID:    repository.Field[string]{Name: "step_id"},
	Position:  repository.Field[int]{Name: "position"},
	Status:    repository.Field[string]{Name: "status"},
	DeletedAt: repository.Field[string]{Name: "deleted_at"},
}

// ItemPostgresMapper maps Item to {schema}.faq_items.
type ItemPostgresMapper struct{ schema string }

func NewItemPostgresMapper(schema string) ItemPostgresMapper {
	return ItemPostgresMapper{schema: schema}
}

func (m ItemPostgresMapper) TableName() string  { return m.schema + ".faq_items" }
func (m ItemPostgresMapper) SelectFrom() string { return m.TableName() + " p" }

func (m ItemPostgresMapper) Columns() []string {
	return []string{
		"p.id", "p.step_id", "p.position", "p.question", "p.answer", "p.tip", "p.status",
		"p.version", "p.created_by", "p.updated_by", "p.created_at", "p.updated_at", "p.deleted_at",
	}
}

func (m ItemPostgresMapper) FieldColumn(field string) string {
	switch field {
	case "id", "step_id", "position", "question", "answer", "tip", "status",
		"version", "created_by", "updated_by", "created_at", "updated_at", "deleted_at":
		return "p." + field
	default:
		return field
	}
}

func (m ItemPostgresMapper) ToRow(i Item) (map[string]any, error) {
	return map[string]any{
		"id":         i.ID,
		"step_id":    i.StepID,
		"position":   i.Position,
		"question":   i.Question,
		"answer":     i.Answer,
		"tip":        i.Tip,
		"status":     i.Status.String(),
		"version":    i.Version,
		"created_by": i.CreatedBy,
		"updated_by": i.UpdatedBy,
		"created_at": i.CreatedAt,
		"updated_at": i.UpdatedAt,
	}, nil
}

func (m ItemPostgresMapper) FromRow(scan func(dest ...any) error) (Item, error) {
	var i Item
	err := scan(
		&i.ID, &i.StepID, &i.Position, &i.Question, &i.Answer, &i.Tip, &i.Status,
		&i.Version, &i.CreatedBy, &i.UpdatedBy, &i.CreatedAt, &i.UpdatedAt, &i.DeletedAt,
	)
	if err != nil {
		return i, err
	}
	return i, nil
}
