package scheduler

import (
	"context"
	"sort"
	"testing"
	"time"
)

// memAdminStore is an in-memory AdminStore stub (Rule #9 — a consumer's admin API
// depends on the interface, not postgres; tests inject this). It mirrors the
// postgres semantics that matter to the control plane: ListJobs returns ALL jobs
// ordered by name, UpdateTiming clamps the interval via ClampInterval, and an
// unknown name is found=false (never an error).
type memAdminStore struct {
	jobs map[ScheduleJobName]AdminJob
	runs map[ScheduleJobName][]Run // newest-first, as the store returns them
}

// compile-time assertion that the stub satisfies the admin contract.
var _ AdminStore = (*memAdminStore)(nil)

func newMemAdminStore() *memAdminStore {
	return &memAdminStore{
		jobs: map[ScheduleJobName]AdminJob{},
		runs: map[ScheduleJobName][]Run{},
	}
}

func (m *memAdminStore) put(j AdminJob) { m.jobs[j.Name] = j }

// putRun prepends a run (so the slice stays newest-first, mirroring the
// ORDER BY started_at DESC the postgres store returns).
func (m *memAdminStore) putRun(r Run) {
	m.runs[r.JobName] = append([]Run{r}, m.runs[r.JobName]...)
}

func (m *memAdminStore) ListJobs(ctx context.Context) ([]AdminJob, error) {
	out := make([]AdminJob, 0, len(m.jobs))
	for _, j := range m.jobs {
		out = append(out, j)
	}
	sort.Slice(out, func(i, k int) bool { return out[i].Name < out[k].Name })
	return out, nil
}

func (m *memAdminStore) UpdateTiming(ctx context.Context, name ScheduleJobName, enabled bool, intervalMinutes int) (AdminJob, bool, error) {
	j, ok := m.jobs[name]
	if !ok {
		return AdminJob{}, false, nil
	}
	j.Enabled = enabled
	j.IntervalMinutes = ClampInterval(intervalMinutes) // store-side floor backstop
	m.jobs[name] = j
	return j, true, nil
}

func (m *memAdminStore) RequestRun(ctx context.Context, name ScheduleJobName) (AdminJob, bool, error) {
	j, ok := m.jobs[name]
	if !ok {
		return AdminJob{}, false, nil
	}
	now := time.Now()
	j.RunRequestedAt = &now
	m.jobs[name] = j
	return j, true, nil
}

func (m *memAdminStore) RecentRuns(ctx context.Context, name ScheduleJobName, limit int) ([]Run, error) {
	if limit <= 0 {
		return nil, nil
	}
	all := m.runs[name]
	if len(all) > limit {
		all = all[:limit]
	}
	out := make([]Run, len(all))
	copy(out, all)
	return out, nil
}

// AC1 — ListJobs returns every job, ordered by name, with the UI-only kind.
func TestAdmin_ListJobs_ReturnsAllOrdered(t *testing.T) {
	store := newMemAdminStore()
	store.put(AdminJob{Job: Job{Name: "news-refresh", Enabled: true, IntervalMinutes: 30}, Kind: "go-handler"})
	store.put(AdminJob{Job: Job{Name: "db-backup", Enabled: false, IntervalMinutes: 1440}, Kind: "make-target"})

	got, err := store.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("ListJobs error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(got))
	}
	if got[0].Name != "db-backup" || got[1].Name != "news-refresh" {
		t.Fatalf("expected ordered [db-backup news-refresh], got [%s %s]", got[0].Name, got[1].Name)
	}
	if got[0].Kind != "make-target" || got[1].Kind != "go-handler" {
		t.Fatalf("kind discriminator not carried: got [%s %s]", got[0].Kind, got[1].Kind)
	}
}

// AC2 — UpdateTiming clamps an interval below the floor to MinIntervalMinutes.
func TestAdmin_UpdateTiming_ClampsBelowFloor(t *testing.T) {
	store := newMemAdminStore()
	store.put(AdminJob{Job: Job{Name: "news-refresh", Enabled: true, IntervalMinutes: 30}, Kind: "go-handler"})

	j, found, err := store.UpdateTiming(context.Background(), "news-refresh", true, 0)
	if err != nil {
		t.Fatalf("UpdateTiming error: %v", err)
	}
	if !found {
		t.Fatalf("expected found=true for an existing job")
	}
	if j.IntervalMinutes != MinIntervalMinutes {
		t.Fatalf("expected interval clamped to %d, got %d", MinIntervalMinutes, j.IntervalMinutes)
	}
}

// AC3 — UpdateTiming on an unknown name is found=false, never an error.
func TestAdmin_UpdateTiming_UnknownName(t *testing.T) {
	store := newMemAdminStore()
	_, found, err := store.UpdateTiming(context.Background(), "nope", true, 10)
	if err != nil {
		t.Fatalf("unknown name must not error, got: %v", err)
	}
	if found {
		t.Fatalf("expected found=false for an unknown job")
	}
}

// AC4 — RequestRun on an unknown name is found=false, never an error.
func TestAdmin_RequestRun_UnknownName(t *testing.T) {
	store := newMemAdminStore()
	_, found, err := store.RequestRun(context.Background(), "nope")
	if err != nil {
		t.Fatalf("unknown name must not error, got: %v", err)
	}
	if found {
		t.Fatalf("expected found=false for an unknown job")
	}
}

// AC5 — RequestRun sets run_requested_at on an existing job.
func TestAdmin_RequestRun_SetsRequestedFlag(t *testing.T) {
	store := newMemAdminStore()
	store.put(AdminJob{Job: Job{Name: "db-backup", Enabled: true, IntervalMinutes: 1440}, Kind: "make-target"})

	j, found, err := store.RequestRun(context.Background(), "db-backup")
	if err != nil {
		t.Fatalf("RequestRun error: %v", err)
	}
	if !found {
		t.Fatalf("expected found=true for an existing job")
	}
	if j.RunRequestedAt == nil {
		t.Fatalf("expected run_requested_at to be set after RequestRun")
	}
}

// AC6 — RecentRuns returns a job's runs newest-first, capped at limit.
func TestAdmin_RecentRuns_NewestFirstRespectsLimit(t *testing.T) {
	store := newMemAdminStore()
	base := time.Date(2026, 6, 20, 3, 0, 0, 0, time.UTC)
	// Insert oldest→newest; putRun prepends so the store stays newest-first.
	for i := 0; i < 3; i++ {
		fin := base.Add(time.Duration(i)*time.Hour + 30*time.Second)
		store.putRun(Run{
			ID:         "run-" + string(rune('a'+i)),
			JobName:    "db-backup",
			StartedAt:  base.Add(time.Duration(i) * time.Hour),
			FinishedAt: &fin,
			Status:     RunStatusOK,
			Output:     "OK",
		})
	}

	got, err := store.RecentRuns(context.Background(), "db-backup", 2)
	if err != nil {
		t.Fatalf("RecentRuns error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected limit to cap at 2, got %d", len(got))
	}
	// Newest first: the last inserted (i=2) leads.
	if !got[0].StartedAt.After(got[1].StartedAt) {
		t.Fatalf("expected newest-first order, got %v before %v", got[0].StartedAt, got[1].StartedAt)
	}
}

// AC7 — RecentRuns for an unknown job (or a non-positive limit) is empty, never an error.
func TestAdmin_RecentRuns_UnknownAndZeroLimit(t *testing.T) {
	store := newMemAdminStore()
	store.putRun(Run{ID: "r1", JobName: "db-backup", StartedAt: time.Now(), Status: RunStatusOK})

	unknown, err := store.RecentRuns(context.Background(), "nope", 10)
	if err != nil {
		t.Fatalf("unknown job must not error, got: %v", err)
	}
	if len(unknown) != 0 {
		t.Fatalf("expected no runs for an unknown job, got %d", len(unknown))
	}

	zero, err := store.RecentRuns(context.Background(), "db-backup", 0)
	if err != nil {
		t.Fatalf("zero limit must not error, got: %v", err)
	}
	if len(zero) != 0 {
		t.Fatalf("expected no runs for a non-positive limit, got %d", len(zero))
	}
}
