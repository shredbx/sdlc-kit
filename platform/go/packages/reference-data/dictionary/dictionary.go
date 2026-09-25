// Package dictionary provides generic enumeration CRUD for dictionary-type entities.
//
// Dictionary manages flat code/label enumerations used as type references in entity attributes
// (e.g., property-type, transaction-type, land-size-unit).
//
// Two backends are provided:
//   - FileBackend: reads values from entity.yml files (read-only, source of truth)
//   - PostgresBackend: lookup tables in PostgreSQL (read-write, for runtime queries)
//
// Dictionary entities follow the dictionary type protocol (.framework/core/types/dictionary.yml):
//   - Flat structure (no hierarchy)
//   - code (lowercase kebab-case) + label (human-readable) + optional description
//   - Append-only in production — deprecated values are marked, never removed
package dictionary

import (
	"context"
	"errors"
)

// Common errors
var (
	ErrNotFound       = errors.New("dictionary entry not found")
	ErrBackendError   = errors.New("dictionary backend error")
	ErrDuplicate      = errors.New("dictionary entry already exists")
	ErrInvalidPayload = errors.New("dictionary invalid payload")
)

// Entry represents a single value in a dictionary enumeration.
type Entry struct {
	Code        string `json:"code" yaml:"code"`
	Label       string `json:"label" yaml:"label"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Icon        string `json:"icon,omitempty" yaml:"icon,omitempty"`
	Color       string `json:"color,omitempty" yaml:"color,omitempty"`
	// Group is an optional intra-dictionary grouping key, letting one dictionary
	// organise its values into sections (e.g. Amenities → Building / Interior /
	// Exterior). Empty = ungrouped. Loaded from the dict table's "group" column.
	Group        string            `json:"group,omitempty" yaml:"group,omitempty"`
	SortOrder    int               `json:"sort_order" yaml:"sort_order"`
	Translations map[string]string `json:"translations,omitempty" yaml:"translations,omitempty"`
	ImageURL     string            `json:"image_url,omitempty" yaml:"image_url,omitempty"`
	VideoURL     string            `json:"video_url,omitempty" yaml:"video_url,omitempty"`
	Deprecated   bool              `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
}

// CatalogueMetadata filters which entries apply to a given region or property type.
// Nil = applies universally. Used by amenity/highlight dictionaries.
type CatalogueMetadata struct {
	Category      string   `json:"category,omitempty" yaml:"category,omitempty"`
	Regions       []string `json:"regions,omitempty" yaml:"regions,omitempty"`
	PropertyTypes []string `json:"property_types,omitempty" yaml:"property_types,omitempty"`
}

// DictEnum represents a complete dictionary enumeration definition.
type DictEnum struct {
	Name     string             `json:"name" yaml:"name"`
	Purpose  string             `json:"purpose" yaml:"purpose"`
	Entries  []Entry            `json:"entries" yaml:"entries"`
	Metadata *CatalogueMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Backend defines the storage interface for dictionary enumerations.
type Backend interface {
	// List returns all entries for a named dictionary.
	List(ctx context.Context, name string) ([]Entry, error)

	// Get returns a single entry by dictionary name and code.
	Get(ctx context.Context, name, code string) (*Entry, error)

	// Create adds a new entry to a dictionary.
	Create(ctx context.Context, name string, entry Entry) error

	// Seed inserts multiple entries, skipping duplicates.
	Seed(ctx context.Context, name string, entries []Entry) error
}

// Store wraps a Backend and provides higher-level operations.
type Store struct {
	backend Backend
}

// NewStore creates a new dictionary Store.
func NewStore(backend Backend) *Store {
	return &Store{backend: backend}
}

// List returns all entries for a dictionary.
func (s *Store) List(ctx context.Context, name string) ([]Entry, error) {
	return s.backend.List(ctx, name)
}

// Get returns a specific entry by code.
func (s *Store) Get(ctx context.Context, name, code string) (*Entry, error) {
	return s.backend.Get(ctx, name, code)
}

// Create adds a new entry.
func (s *Store) Create(ctx context.Context, name string, entry Entry) error {
	return s.backend.Create(ctx, name, entry)
}

// Seed inserts entries from entity.yml values, skipping existing ones.
func (s *Store) Seed(ctx context.Context, name string, entries []Entry) error {
	return s.backend.Seed(ctx, name, entries)
}
