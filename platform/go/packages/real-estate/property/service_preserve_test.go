package property_test

import (
	"context"
	"testing"

	"github.com/shredbx/sbx-core/pkg/property"
	"github.com/shredbx/sbx-core/pkg/repository"
)

// fakeRepo is a minimal in-memory repository.Repository[Property] used to
// exercise PropertyService's preserve-on-omit logic, which only runs when a
// repo is wired (the nil-repo path in the other service tests short-circuits
// before the existing-row fetch). Update is a FULL REPLACE on purpose — it
// stores exactly what it is given, so any field preservation observed in the
// stored row MUST have come from the service layer, not the repo.
type fakeRepo struct {
	items map[string]property.Property
}

func newFakeRepo() *fakeRepo { return &fakeRepo{items: map[string]property.Property{}} }

func (r *fakeRepo) Get(_ context.Context, id string) (property.Property, error) {
	p, ok := r.items[id]
	if !ok {
		return property.Property{}, repository.ErrNotFound
	}
	return p, nil
}

func (r *fakeRepo) List(_ context.Context, _ repository.ListOptions) ([]property.Property, int, error) {
	out := make([]property.Property, 0, len(r.items))
	for _, p := range r.items {
		out = append(out, p)
	}
	return out, len(out), nil
}

func (r *fakeRepo) Create(_ context.Context, p property.Property) (property.Property, error) {
	r.items[p.ID] = p
	return p, nil
}

func (r *fakeRepo) Update(_ context.Context, id string, p property.Property) (property.Property, error) {
	if _, ok := r.items[id]; !ok {
		return property.Property{}, repository.ErrNotFound
	}
	p.ID = id
	r.items[id] = p // full replace — no field preservation here
	return p, nil
}

func (r *fakeRepo) Delete(_ context.Context, id string) error {
	delete(r.items, id)
	return nil
}

// Regression guard (2605-170): the Information-facet "Save" PUT sends neither
// is_published nor lifecycle_status. A naive full-replace silently unpublished
// a live listing and reset its lifecycle. Update must carry both forward from
// the existing row when the payload omits them. pool is nil so the
// attribute-group writes are no-ops; the assertion is purely about the
// primary-row preservation in PropertyService.Update.
func TestPropertyService_Update_PreservesPublishAndLifecycle(t *testing.T) {
	repo := newFakeRepo()
	svc := property.NewPropertyService(repo, nil, "bestierealestate")
	ctx := context.Background()

	// Create then publish + advance lifecycle so the existing row is a live,
	// non-default state (the values a metadata PUT must not clobber).
	created, err := svc.Create(ctx, property.Property{
		Title:   ptr("Live Listing"),
		ForSale: true,
	})
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if created.IsPublished {
		t.Fatalf("Create: expected is_published=false at create time, got true")
	}
	if err := svc.SetPublished(ctx, created.ID, true); err != nil {
		t.Fatalf("SetPublished(true): unexpected error: %v", err)
	}
	if err := svc.SetLifecycleStatus(ctx, created.ID, property.LifecycleSold); err != nil {
		t.Fatalf("SetLifecycleStatus(sold): unexpected error: %v", err)
	}

	// Sanity: the stored row is now published + sold before the metadata PUT.
	pre, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get(pre): unexpected error: %v", err)
	}
	if !pre.IsPublished || pre.LifecycleStatus != property.LifecycleSold {
		t.Fatalf("precondition: want published+sold, got published=%v lifecycle=%q",
			pre.IsPublished, pre.LifecycleStatus)
	}

	// Simulate the edit-form metadata PUT: IsPublished is the zero value (false,
	// indistinguishable from "omitted") and LifecycleStatus is "" (omitted), with
	// a real field change to the title.
	updated, err := svc.Update(ctx, created.ID, property.Property{
		Title:           ptr("Live Listing — Updated Copy"),
		ForSale:         true,
		IsPublished:     false, // zero value — must NOT unpublish
		LifecycleStatus: "",    // omitted — must NOT reset lifecycle
	})
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}

	// The returned row must still be published + sold (preserved from existing).
	if !updated.IsPublished {
		t.Error("Update: is_published was silently unpublished — regression in preserve-on-omit")
	}
	if updated.LifecycleStatus != property.LifecycleSold {
		t.Errorf("Update: lifecycle_status reset to %q — expected preserved %q",
			updated.LifecycleStatus, property.LifecycleSold)
	}

	// The real field change must persist (we did not freeze the row).
	if updated.Title == nil || *updated.Title != "Live Listing — Updated Copy" {
		t.Errorf("Update: title change not persisted: got %v", updated.Title)
	}

	// And the stored row reflects the same (re-read to be sure the write landed).
	post, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get(post): unexpected error: %v", err)
	}
	if !post.IsPublished {
		t.Error("Update: stored row was unpublished")
	}
	if post.LifecycleStatus != property.LifecycleSold {
		t.Errorf("Update: stored lifecycle_status=%q, expected %q", post.LifecycleStatus, property.LifecycleSold)
	}
	if post.Title == nil || *post.Title != "Live Listing — Updated Copy" {
		t.Errorf("Update: stored title not persisted: got %v", post.Title)
	}
}

// Companion: an explicit lifecycle change in the SAME Update path is honored
// (preserve-on-omit must not become preserve-always for lifecycle). is_published
// is always preserved by design — PATCH /publish is its sole writer — so it is
// intentionally not made explicitly settable through Update here.
func TestPropertyService_Update_ExplicitLifecycleHonored(t *testing.T) {
	repo := newFakeRepo()
	svc := property.NewPropertyService(repo, nil, "bestierealestate")
	ctx := context.Background()

	created, err := svc.Create(ctx, property.Property{Title: ptr("Listing"), ForSale: true})
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if err := svc.SetPublished(ctx, created.ID, true); err != nil {
		t.Fatalf("SetPublished: unexpected error: %v", err)
	}

	updated, err := svc.Update(ctx, created.ID, property.Property{
		Title:           ptr("Listing"),
		ForSale:         true,
		LifecycleStatus: property.LifecycleWithdrawn, // explicit — must be applied
	})
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}
	if updated.LifecycleStatus != property.LifecycleWithdrawn {
		t.Errorf("Update: explicit lifecycle_status not applied: got %q want %q",
			updated.LifecycleStatus, property.LifecycleWithdrawn)
	}
	if !updated.IsPublished {
		t.Error("Update: is_published must stay preserved even on an explicit lifecycle change")
	}
}

// primaryTableRepo models a real repository more faithfully than the bare fakeRepo
// for the DRIFT-1 guard: a real repo.Update writes ONLY the primary `properties`
// row, so building specs + policies (their own tables) do NOT survive a repo
// round-trip — they reach the response solely via the service's Save*/carry-forward
// path. The bare fakeRepo's full-struct replace would persist them "for free" and
// mask the bug; stripping the two attribute groups on Update exposes it.
type primaryTableRepo struct{ *fakeRepo }

func (r *primaryTableRepo) Update(ctx context.Context, id string, p property.Property) (property.Property, error) {
	p.BuildingSpecs = nil
	p.Policies = nil
	return r.fakeRepo.Update(ctx, id, p)
}

// DRIFT-1 regression guard (2606-047): Create() persists BuildingSpecs + Policies,
// but Update() omitted both Save* calls AND the response carry-forward — so editing
// a property with building specs or lease terms silently dropped them. pool is nil
// (the attribute-group DB writes are no-ops, untestable at unit level), so this
// asserts the response carry-forward over the primary-table-only repo above; the
// buggy Update leaves BuildingSpecs/Policies nil on the returned property.
func TestPropertyService_Update_PersistsBuildingSpecsAndPolicies(t *testing.T) {
	repo := &primaryTableRepo{newFakeRepo()}
	svc := property.NewPropertyService(repo, nil, "bestierealestate")
	ctx := context.Background()

	created, err := svc.Create(ctx, property.Property{Title: ptr("Spec Listing"), ForSale: true})
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	specs := &property.BuildingSpecs{Floors: ptr(3), ParkingSpaces: ptr(2), YearBuilt: ptr(2021)}
	policies := &property.Policies{MinimumLeaseMonths: ptr(12)}

	updated, err := svc.Update(ctx, created.ID, property.Property{
		Title:         ptr("Spec Listing"),
		ForSale:       true,
		BuildingSpecs: specs,
		Policies:      policies,
	})
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}

	if updated.BuildingSpecs == nil {
		t.Fatal("Update: building specs were silently dropped on save (DRIFT-1 regression)")
	}
	if updated.BuildingSpecs.YearBuilt == nil || *updated.BuildingSpecs.YearBuilt != 2021 {
		t.Errorf("Update: building specs not carried through: got %+v", updated.BuildingSpecs)
	}
	if updated.Policies == nil {
		t.Fatal("Update: lease policies were silently dropped on save (DRIFT-1 regression)")
	}
	if updated.Policies.MinimumLeaseMonths == nil || *updated.Policies.MinimumLeaseMonths != 12 {
		t.Errorf("Update: policies not carried through: got %+v", updated.Policies)
	}
}
