package transaction_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shredbx/sbx-core/pkg/repository"
	"github.com/shredbx/sbx-core/pkg/transaction"
)

func loadTestStore(t *testing.T) *transaction.YAMLStore {
	t.Helper()
	store, err := transaction.NewYAMLStore("testdata/seed.yml")
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

func TestYAMLStore_List_FilterByStatus(t *testing.T) {
	store := loadTestStore(t)
	opts := repository.ListOptions{
		Filter: repository.And(transaction.Fields.Status.Eq(string(transaction.StatusOpen))),
	}
	items, total, err := store.List(context.Background(), opts)
	if err != nil {
		t.Fatalf("List open: %v", err)
	}
	if total != 2 {
		t.Errorf("List open total: want 2, got %d", total)
	}
	for _, item := range items {
		if item.Status != transaction.StatusOpen {
			t.Errorf("List open: got non-open deal %s (%s)", item.ID, item.Status)
		}
	}
}

func TestYAMLStore_List_FilterByProperty(t *testing.T) {
	store := loadTestStore(t)
	opts := repository.ListOptions{
		Filter: repository.And(transaction.Fields.PropertyID.Eq(propActive)),
	}
	items, _, err := store.List(context.Background(), opts)
	if err != nil {
		t.Fatalf("List by property: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("List by property: want 1, got %d", len(items))
	}
	if items[0].PropertyID != propActive {
		t.Errorf("List by property: got %s, want %s", items[0].PropertyID, propActive)
	}
}

func TestYAMLStore_Get(t *testing.T) {
	store := loadTestStore(t)
	txn, err := store.Get(context.Background(), dealLeaseOpen)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if txn.Type != transaction.TypeLease {
		t.Errorf("Get type: want lease, got %q", txn.Type)
	}
	if txn.LeaseRent.Amount != 2500000 || txn.LeaseRent.Currency != "THB" {
		t.Errorf("Get lease rent: want 2500000 THB, got %d %s", txn.LeaseRent.Amount, txn.LeaseRent.Currency)
	}
	if txn.LeaseTerm != transaction.Months(12) {
		t.Errorf("Get lease duration: want 12, got %d", txn.LeaseTerm)
	}
}

func TestYAMLStore_Get_NotFound(t *testing.T) {
	store := loadTestStore(t)
	_, err := store.Get(context.Background(), "nonexistent-id")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("Get nonexistent: want ErrNotFound, got %v", err)
	}
}

func TestYAMLStore_Create(t *testing.T) {
	store := loadTestStore(t)
	created, err := store.Create(context.Background(), transaction.Transaction{
		Type:       transaction.TypeSale,
		PropertyID: propActive,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Error("Create: expected non-empty ID")
	}
	if created.Status != transaction.StatusOpen {
		t.Errorf("Create: status defaulted to %q, want open", created.Status)
	}
	if _, err := store.Get(context.Background(), created.ID); err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
}

func TestYAMLStore_Update(t *testing.T) {
	store := loadTestStore(t)
	existing, err := store.Get(context.Background(), dealSaleOpen)
	if err != nil {
		t.Fatalf("Get for update: %v", err)
	}
	note := "price reduced"
	existing.Notes = &note
	updated, err := store.Update(context.Background(), dealSaleOpen, existing)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Notes == nil || *updated.Notes != note {
		t.Errorf("Update: notes not persisted, got %v", updated.Notes)
	}
}

func TestYAMLStore_Update_NotFound(t *testing.T) {
	store := loadTestStore(t)
	_, err := store.Update(context.Background(), "nonexistent", transaction.Transaction{})
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("Update nonexistent: want ErrNotFound, got %v", err)
	}
}

func TestYAMLStore_Delete(t *testing.T) {
	store := loadTestStore(t)
	if err := store.Delete(context.Background(), dealClosed); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, total, _ := store.List(context.Background(), repository.ListOptions{})
	if total != 2 {
		t.Errorf("List after Delete: want 2, got %d", total)
	}
	if _, err := store.Get(context.Background(), dealClosed); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("Get after Delete: want ErrNotFound, got %v", err)
	}
}

func TestYAMLStore_Delete_NotFound(t *testing.T) {
	store := loadTestStore(t)
	if err := store.Delete(context.Background(), "nonexistent"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("Delete nonexistent: want ErrNotFound, got %v", err)
	}
}

func TestYAMLStore_ImplementsRepository(t *testing.T) {
	var _ repository.Repository[transaction.Transaction] = (*transaction.YAMLStore)(nil)
}
