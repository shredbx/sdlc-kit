package property_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shredbx/sbx-core/pkg/property"
	"github.com/shredbx/sbx-core/pkg/repository"
)

func loadTestStore(t *testing.T) *property.YAMLStore {
	t.Helper()
	store, err := property.NewYAMLStore("testdata/seed.yml")
	if err != nil {
		t.Fatalf("NewYAMLStore: %v", err)
	}
	return store
}

func TestYAMLStore_List(t *testing.T) {
	store := loadTestStore(t)
	items, total, err := store.List(context.Background(), repository.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Errorf("List total: want 3, got %d", total)
	}
	if len(items) != 3 {
		t.Errorf("List items: want 3, got %d", len(items))
	}
}

func TestYAMLStore_List_FilterPublished(t *testing.T) {
	store := loadTestStore(t)
	opts := repository.ListOptions{
		Filter: repository.And(property.Fields.IsPublished.Eq(true)),
	}
	items, total, err := store.List(context.Background(), opts)
	if err != nil {
		t.Fatalf("List published: %v", err)
	}
	if total != 2 {
		t.Errorf("List published total: want 2, got %d", total)
	}
	for _, item := range items {
		if !item.IsPublished {
			t.Errorf("List published: got unpublished item %s", item.ID)
		}
	}
}

func TestYAMLStore_List_FilterByForSale(t *testing.T) {
	store := loadTestStore(t)
	opts := repository.ListOptions{
		Filter: repository.And(property.Fields.ForSale.Eq(true)),
	}
	items, _, err := store.List(context.Background(), opts)
	if err != nil {
		t.Fatalf("List by for_sale: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("List for_sale items: want 2, got %d", len(items))
	}
}

func TestYAMLStore_Get(t *testing.T) {
	store := loadTestStore(t)
	p, err := store.Get(context.Background(), "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.ID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("Get ID: want 11111111-1111-1111-1111-111111111111, got %s", p.ID)
	}
	if p.Title == nil || *p.Title != "Luxury Pool Villa in Phuket" {
		t.Errorf("Get title: want 'Luxury Pool Villa in Phuket', got %v", p.Title)
	}
	if p.Address.Province != "Phuket" {
		t.Errorf("Get address.province: want 'Phuket', got %q", p.Address.Province)
	}
}

func TestYAMLStore_Get_NotFound(t *testing.T) {
	store := loadTestStore(t)
	_, err := store.Get(context.Background(), "nonexistent-id")
	if err == nil {
		t.Fatal("Get nonexistent: expected error")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("Get nonexistent: want ErrNotFound, got %v", err)
	}
}

func TestYAMLStore_Create(t *testing.T) {
	store := loadTestStore(t)
	p := property.Property{
		Title:       ptr("New Test Property"),
		IsPublished: false,
	}
	created, err := store.Create(context.Background(), p)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Error("Create: expected non-empty ID")
	}

	fetched, err := store.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if fetched.Title == nil || *fetched.Title != "New Test Property" {
		t.Errorf("Get after Create: title mismatch")
	}
}

func TestYAMLStore_Delete(t *testing.T) {
	store := loadTestStore(t)

	err := store.Delete(context.Background(), "22222222-2222-2222-2222-222222222222")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	items, total, _ := store.List(context.Background(), repository.ListOptions{})
	if total != 2 {
		t.Errorf("List after Delete: want 2, got %d", total)
	}
	for _, item := range items {
		if item.ID == "22222222-2222-2222-2222-222222222222" {
			t.Error("List after Delete: deleted item still visible")
		}
	}
}

func TestYAMLStore_Delete_NotFound(t *testing.T) {
	store := loadTestStore(t)
	err := store.Delete(context.Background(), "nonexistent")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("Delete nonexistent: want ErrNotFound, got %v", err)
	}
}

func TestYAMLStore_ImplementsRepository(t *testing.T) {
	var _ repository.Repository[property.Property] = (*property.YAMLStore)(nil)
}
