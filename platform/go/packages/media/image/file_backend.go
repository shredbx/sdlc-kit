package image

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// =============================================================================
// FILE BACKEND — metadata storage on filesystem (YAML files)
// =============================================================================

// FileBackend stores image metadata as YAML files.
// Mandatory per document type rules — enables development without a database.
type FileBackend struct {
	rootDir string
	mu      sync.RWMutex
}

// NewFileBackend creates a filesystem-based metadata backend.
func NewFileBackend(rootDir string) *FileBackend {
	return &FileBackend{rootDir: rootDir}
}

func (b *FileBackend) metadataPath(id string) string {
	return filepath.Join(b.rootDir, "metadata", id+".yml")
}

func (b *FileBackend) metadataDir() string {
	return filepath.Join(b.rootDir, "metadata")
}

// Create saves image metadata to a YAML file.
func (b *FileBackend) Create(ctx context.Context, img *Image) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	dir := b.metadataDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("%w: mkdir: %v", ErrBackendError, err)
	}

	return b.writeMetadata(img)
}

// Get loads image metadata from a YAML file.
func (b *FileBackend) Get(ctx context.Context, id string) (*Image, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	path := b.metadataPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrImageNotFound
		}
		return nil, fmt.Errorf("%w: read: %v", ErrBackendError, err)
	}

	var img Image
	if err := yaml.Unmarshal(data, &img); err != nil {
		return nil, fmt.Errorf("%w: unmarshal: %v", ErrBackendError, err)
	}

	return &img, nil
}

// Update overwrites image metadata YAML file.
func (b *FileBackend) Update(ctx context.Context, img *Image) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	path := b.metadataPath(img.ID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ErrImageNotFound
	}

	return b.writeMetadata(img)
}

// Delete removes the image metadata YAML file (hard delete for filesystem).
func (b *FileBackend) Delete(ctx context.Context, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	path := b.metadataPath(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return ErrImageNotFound
		}
		return fmt.Errorf("%w: delete: %v", ErrBackendError, err)
	}
	return nil
}

// DeleteByKey removes the metadata file whose image key matches, if any.
// Idempotent + best-effort (filesystem counterpart of PostgresBackend.DeleteByKey;
// FileBackend.Delete hard-removes the file, so this mirrors that for key reclaim).
func (b *FileBackend) DeleteByKey(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	dir := b.metadataDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("%w: readdir: %v", ErrBackendError, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var img Image
		if err := yaml.Unmarshal(data, &img); err != nil {
			continue
		}
		if img.Key == key {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
	return nil
}

// ListBySlot returns the non-deleted images whose key is under the owner-scoped slot
// prefix {owner}/{ownerId}/{facet}/ (or system/{facet}/). Filesystem counterpart of
// PostgresBackend.ListBySlot — scan metadata YAMLs, match key prefix, skip deleted.
func (b *FileBackend) ListBySlot(ctx context.Context, owner ImageOwner, ownerID string, facet ImageFacet) ([]Image, error) {
	prefix := slotPrefix(owner, ownerID, facet)
	b.mu.RLock()
	defer b.mu.RUnlock()

	dir := b.metadataDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Image{}, nil
		}
		return nil, fmt.Errorf("%w: readdir: %v", ErrBackendError, err)
	}

	var images []Image
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var img Image
		if err := yaml.Unmarshal(data, &img); err != nil {
			continue
		}
		if img.DeletedAt != nil {
			continue
		}
		if strings.HasPrefix(img.Key, prefix) {
			images = append(images, img)
		}
	}
	return images, nil
}

// List returns all image metadata from the filesystem.
func (b *FileBackend) List(ctx context.Context, opts ListOpts) ([]Image, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	dir := b.metadataDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Image{}, nil
		}
		return nil, fmt.Errorf("%w: readdir: %v", ErrBackendError, err)
	}

	var images []Image
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var img Image
		if err := yaml.Unmarshal(data, &img); err != nil {
			continue
		}

		// Filter by facet if specified
		if opts.Facet != "" && img.Purpose != opts.Facet {
			continue
		}

		// Skip soft-deleted
		if img.DeletedAt != nil {
			continue
		}

		images = append(images, img)
	}

	// Apply pagination
	if opts.Offset > 0 && opts.Offset < len(images) {
		images = images[opts.Offset:]
	} else if opts.Offset >= len(images) {
		return []Image{}, nil
	}

	if opts.Limit > 0 && opts.Limit < len(images) {
		images = images[:opts.Limit]
	}

	return images, nil
}

func (b *FileBackend) writeMetadata(img *Image) error {
	data, err := yaml.Marshal(img)
	if err != nil {
		return fmt.Errorf("%w: marshal: %v", ErrBackendError, err)
	}

	path := b.metadataPath(img.ID)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("%w: write: %v", ErrBackendError, err)
	}
	return nil
}

// =============================================================================
// FILE OBJECT STORE — binary storage on filesystem
// =============================================================================

// FileObjectStore stores binary image data as files on disk.
// Used for development and testing — no R2/S3 needed.
type FileObjectStore struct {
	rootDir   string
	baseURL   string // URL prefix for generated URLs (e.g., "http://localhost:5000/images")
	mu        sync.RWMutex
}

// NewFileObjectStore creates a filesystem-based object store.
// baseURL is the URL prefix for file URLs (e.g., "http://localhost:5000/images").
func NewFileObjectStore(rootDir, baseURL string) *FileObjectStore {
	return &FileObjectStore{rootDir: rootDir, baseURL: baseURL}
}

// Upload writes binary data to the filesystem.
func (s *FileObjectStore) Upload(ctx context.Context, key string, data io.Reader, contentType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.rootDir, key)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("%w: mkdir: %v", ErrBackendError, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%w: create: %v", ErrBackendError, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, data); err != nil {
		return fmt.Errorf("%w: copy: %v", ErrBackendError, err)
	}

	return nil
}

// Download reads binary data from the filesystem.
func (s *FileObjectStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.rootDir, key)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrImageNotFound
		}
		return nil, fmt.Errorf("%w: open: %v", ErrBackendError, err)
	}
	return f, nil
}

// Delete removes binary data from the filesystem.
func (s *FileObjectStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.rootDir, key)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil // idempotent
		}
		return fmt.Errorf("%w: delete: %v", ErrBackendError, err)
	}
	return nil
}

// URL returns a local URL for the given key.
func (s *FileObjectStore) URL(key string) string {
	return s.baseURL + "/" + key
}
