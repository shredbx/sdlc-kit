package image

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// MEMORY BACKEND — in-memory metadata storage for testing
// =============================================================================

// MemoryBackend stores image metadata in memory.
// Thread-safe, no external dependencies. Ideal for unit tests.
type MemoryBackend struct {
	mu     sync.RWMutex
	images map[string]*Image
}

// NewMemoryBackend creates an in-memory metadata backend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{images: make(map[string]*Image)}
}

// Create stores image metadata in memory.
func (b *MemoryBackend) Create(ctx context.Context, img *Image) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	cp := *img
	b.images[img.ID] = &cp
	return nil
}

// Get retrieves image metadata by ID.
func (b *MemoryBackend) Get(ctx context.Context, id string) (*Image, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	img, ok := b.images[id]
	if !ok {
		return nil, ErrImageNotFound
	}
	if img.DeletedAt != nil {
		return nil, ErrImageNotFound
	}
	cp := *img
	return &cp, nil
}

// Update replaces image metadata in memory.
func (b *MemoryBackend) Update(ctx context.Context, img *Image) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	existing, ok := b.images[img.ID]
	if !ok || existing.DeletedAt != nil {
		return ErrImageNotFound
	}

	cp := *img
	b.images[img.ID] = &cp
	return nil
}

// Delete soft-deletes an image (sets DeletedAt).
func (b *MemoryBackend) Delete(ctx context.Context, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	img, ok := b.images[id]
	if !ok || img.DeletedAt != nil {
		return ErrImageNotFound
	}

	now := time.Now().UTC()
	img.DeletedAt = &now
	return nil
}

// DeleteByKey soft-deletes the image matching the given object key, if any.
// Idempotent + best-effort (memory counterpart of PostgresBackend.DeleteByKey).
func (b *MemoryBackend) DeleteByKey(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now().UTC()
	for _, img := range b.images {
		if img.Key == key && img.DeletedAt == nil {
			img.DeletedAt = &now
		}
	}
	return nil
}

// ListBySlot returns the non-deleted images whose key is under the owner-scoped slot
// prefix {owner}/{ownerId}/{facet}/ (or system/{facet}/). Memory counterpart of
// PostgresBackend.ListBySlot — iterate, match key prefix, skip deleted.
func (b *MemoryBackend) ListBySlot(ctx context.Context, owner ImageOwner, ownerID string, facet ImageFacet) ([]Image, error) {
	prefix := slotPrefix(owner, ownerID, facet)
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []Image
	for _, img := range b.images {
		if img.DeletedAt != nil {
			continue
		}
		if strings.HasPrefix(img.Key, prefix) {
			result = append(result, *img)
		}
	}
	return result, nil
}

// List returns all non-deleted images matching the options.
func (b *MemoryBackend) List(ctx context.Context, opts ListOpts) ([]Image, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []Image
	for _, img := range b.images {
		if img.DeletedAt != nil {
			continue
		}
		if opts.Facet != "" && img.Purpose != opts.Facet {
			continue
		}
		result = append(result, *img)
	}

	// Apply pagination
	if opts.Offset > 0 && opts.Offset < len(result) {
		result = result[opts.Offset:]
	} else if opts.Offset >= len(result) {
		return []Image{}, nil
	}

	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result, nil
}

// =============================================================================
// MEMORY OBJECT STORE — in-memory binary storage for testing
// =============================================================================

// MemoryObjectStore stores binary data in memory.
// Thread-safe, no external dependencies. Ideal for unit tests.
type MemoryObjectStore struct {
	mu      sync.RWMutex
	objects map[string][]byte
	baseURL string
}

// NewMemoryObjectStore creates an in-memory object store.
func NewMemoryObjectStore(baseURL string) *MemoryObjectStore {
	return &MemoryObjectStore{
		objects: make(map[string][]byte),
		baseURL: baseURL,
	}
}

// Upload stores binary data in memory.
func (s *MemoryObjectStore) Upload(ctx context.Context, key string, data io.Reader, contentType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	buf, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	s.objects[key] = buf
	return nil
}

// Download retrieves binary data from memory.
func (s *MemoryObjectStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.objects[key]
	if !ok {
		return nil, ErrImageNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// Delete removes binary data from memory.
func (s *MemoryObjectStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.objects, key)
	return nil
}

// URL returns a mock URL for the given key.
func (s *MemoryObjectStore) URL(key string) string {
	return s.baseURL + "/" + key
}

// Len returns the number of stored objects (for test assertions).
func (s *MemoryObjectStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.objects)
}

// Has checks if an object exists (for test assertions).
func (s *MemoryObjectStore) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.objects[key]
	return ok
}
