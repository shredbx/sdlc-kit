// Package dictionary file backend — reads dictionary values from entity.yml files.
//
// The FileBackend is read-only. It reads the `values` list from dictionary-type
// entity.yml files. This is the source of truth for dictionary values.
//
// Directory structure:
//
//	{rootDir}/{name}/entity.yml  →  values: [{code, label, sort_order, ...}]
package dictionary

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// FileBackend reads dictionary values from entity.yml files.
type FileBackend struct {
	rootDir string // Path to entities/ directory
}

// NewFileBackend creates a file backend rooted at the entities directory.
func NewFileBackend(rootDir string) *FileBackend {
	return &FileBackend{rootDir: rootDir}
}

// entityFile holds the parts of entity.yml we care about for dictionary values.
type entityFile struct {
	Name         string             `yaml:"name"`
	Purpose      string             `yaml:"purpose"`
	TypeCategory string             `yaml:"type_category"`
	Values       []Entry            `yaml:"values"`
	Metadata     *CatalogueMetadata `yaml:"metadata"`
}

// readEntityValues reads and parses dictionary values from an entity.yml file.
func (b *FileBackend) readEntityValues(name string) (*entityFile, error) {
	path := filepath.Join(b.rootDir, name, "entity.yml")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %v", ErrBackendError, path, err)
	}

	var ef entityFile
	if err := yaml.Unmarshal(data, &ef); err != nil {
		return nil, fmt.Errorf("%w: parse %s: %v", ErrBackendError, path, err)
	}

	if ef.TypeCategory != "dictionary" {
		return nil, fmt.Errorf("%w: %s is not a dictionary entity (type_category=%s)", ErrBackendError, name, ef.TypeCategory)
	}

	return &ef, nil
}

// List returns all entries for a named dictionary, sorted by sort_order.
func (b *FileBackend) List(ctx context.Context, name string) ([]Entry, error) {
	ef, err := b.readEntityValues(name)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, len(ef.Values))
	copy(entries, ef.Values)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SortOrder < entries[j].SortOrder
	})

	return entries, nil
}

// Get returns a single entry by code.
func (b *FileBackend) Get(ctx context.Context, name, code string) (*Entry, error) {
	ef, err := b.readEntityValues(name)
	if err != nil {
		return nil, err
	}

	for _, e := range ef.Values {
		if e.Code == code {
			return &e, nil
		}
	}

	return nil, fmt.Errorf("%w: code '%s' not found in dictionary '%s'", ErrNotFound, code, name)
}

// Create is not supported for file backend (read-only source of truth).
func (b *FileBackend) Create(ctx context.Context, name string, entry Entry) error {
	return fmt.Errorf("%w: file backend is read-only", ErrBackendError)
}

// Seed is not supported for file backend (read-only source of truth).
func (b *FileBackend) Seed(ctx context.Context, name string, entries []Entry) error {
	return fmt.Errorf("%w: file backend is read-only", ErrBackendError)
}

// ListAll reads all dictionary entities in the root directory.
func (b *FileBackend) ListAll(ctx context.Context) ([]DictEnum, error) {
	dirEntries, err := os.ReadDir(b.rootDir)
	if err != nil {
		return nil, fmt.Errorf("%w: read dir: %v", ErrBackendError, err)
	}

	var dicts []DictEnum
	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}

		ef, err := b.readEntityValues(de.Name())
		if err != nil {
			continue // Skip non-dictionary entities
		}

		dicts = append(dicts, DictEnum{
			Name:     ef.Name,
			Purpose:  ef.Purpose,
			Entries:  ef.Values,
			Metadata: ef.Metadata,
		})
	}

	return dicts, nil
}
