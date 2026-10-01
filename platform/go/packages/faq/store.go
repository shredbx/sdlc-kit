// Store — the persistence SEAM. pkg/faq ships the INTERFACE only (the contract the
// app implements, tests fake, and the future FAQ Tool wraps). The postgres adapter
// + its testcontainers test live in the BR app (internal/repository) — same split
// as pkg/cms (mapper here) + internal/repository/contententry.go (repo + test
// there), because the testcontainers integration test dep belongs to the app, not
// the sbx-core framework module.
//
// This keeps pkg/faq storage-import-free except for mapper.go (postgres.Mapper),
// so the package stays reusable + lean. The interface is tool-shaped (Search matches
// the future FAQ Tool input_schema, #0322 P4).
package faq

import "context"

// FAQQuery is the read contract for Store.Search — deliberately tool-shaped for the
// future FAQ Tool input_schema (#0322, P4): a category filter + free-text + a limit.
type FAQQuery struct {
	Category string // category slug filter ("" = any category)
	Text     string // free-text match against question/answer
	Limit    int
}

// ListOptions narrows a list call. Zero value lists every live row. PublishedOnly
// filters status='published' (the public path — SC17/SC18 exclude drafts).
type ListOptions struct {
	PublishedOnly bool
}

// CategoryView is a category with its published steps + their items — the render
// tree the /faq page consumes (built by the Service facade).
type CategoryView struct {
	Category Category
	Steps    []StepView
}

// StepView is one step with its items.
type StepView struct {
	Step  Step
	Items []Item
}

// Store is the persistence port (Strategy + Repository). The BR app's
// internal/repository/FAQRepository implements it; tests use a fake; the future FAQ
// Tool wraps Store.Search.
type Store interface {
	// Reads
	GetCategory(ctx context.Context, id string) (Category, error)
	GetCategoryBySlug(ctx context.Context, slug Slug) (Category, error)
	ListCategories(ctx context.Context, opts ListOptions) ([]Category, error)
	ListSteps(ctx context.Context, categoryID string, opts ListOptions) ([]Step, error)
	ListItems(ctx context.Context, stepID string, opts ListOptions) ([]Item, error)
	Search(ctx context.Context, q FAQQuery) ([]Item, error)
	ResolveItems(ctx context.Context, ids []string) ([]Item, error)
	// Writes
	UpsertCategory(ctx context.Context, c Category) (Category, error)
	UpsertStep(ctx context.Context, s Step) (Step, error)
	UpsertItem(ctx context.Context, i Item) (Item, error)
	SetFeatured(ctx context.Context, categoryID string) error
	ReorderSteps(ctx context.Context, categoryID string, orderedIDs []string) error
	ReorderItems(ctx context.Context, stepID string, orderedIDs []string) error
	DeleteCategory(ctx context.Context, id string) error
}
