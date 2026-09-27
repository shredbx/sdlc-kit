package scheduler

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// memStore is an in-memory Store stub (Rule #9 — the engine never hard-couples
// to postgres; tests inject a stub, mirroring rss.Store + WithStore). It exercises
// the engine gates without a DB. ClaimRun is guarded by a mutex so the
// single-flight test can assert exactly-one-claim under concurrency.
type memStore struct {
	mu      sync.Mutex
	jobs    map[ScheduleJobName]Job
	open    map[ScheduleJobName]bool // a job currently has an OPEN (running) run
	runs    map[string]Run
	nextID  int
	now     time.Time
	outputs []string // FinishRun outputs captured for assertion
}

func newMemStore(now time.Time) *memStore {
	return &memStore{
		jobs: map[ScheduleJobName]Job{},
		open: map[ScheduleJobName]bool{},
		runs: map[string]Run{},
		now:  now,
	}
}

func (m *memStore) put(j Job) {
	m.jobs[j.Name] = j
}

func (m *memStore) DueJobs(ctx context.Context, now time.Time, registered []ScheduleJobName) ([]Job, error) {
	reg := map[ScheduleJobName]bool{}
	for _, n := range registered {
		reg[n] = true
	}
	var out []Job
	for _, j := range m.jobs {
		if !reg[j.Name] {
			continue
		}
		if m.open[j.Name] {
			continue
		}
		due := j.RunRequestedAt != nil
		if !due {
			// reuse the same DueNow gate the engine uses
			s := New(m)
			due = s.IsDue(j, now)
		}
		if due {
			out = append(out, j)
		}
	}
	return out, nil
}

func (m *memStore) JobByName(ctx context.Context, name ScheduleJobName) (Job, bool, error) {
	j, ok := m.jobs[name]
	return j, ok, nil
}

func (m *memStore) ClaimRun(ctx context.Context, name ScheduleJobName) (string, RunTrigger, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.jobs[name]; !ok {
		return "", TriggerSchedule, false, errors.New("no such job")
	}
	if m.open[name] {
		return "", TriggerSchedule, false, nil // already claimed — single-flight
	}
	m.nextID++
	id := "run-" + itoa(m.nextID)
	m.open[name] = true
	m.runs[id] = Run{ID: id, JobName: name, StartedAt: m.now, Status: RunStatusRunning}
	// Mirror the postgres store: a pending manual request labels the claim manual
	// and is consumed by it.
	trigger := TriggerSchedule
	if j, ok := m.jobs[name]; ok && j.RunRequestedAt != nil {
		trigger = TriggerManual
		j.RunRequestedAt = nil
		m.jobs[name] = j
	}
	return id, trigger, true, nil
}

func (m *memStore) FinishRun(ctx context.Context, runID string, finishedAt time.Time, status RunStatus, output string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[runID]
	if !ok {
		return errors.New("no such run")
	}
	r.FinishedAt = &finishedAt
	r.Status = status
	r.Output = output
	m.runs[runID] = r
	m.open[r.JobName] = false
	m.outputs = append(m.outputs, output)
	return nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func ptrTime(t time.Time) *time.Time { return &t }

// SC1 — a job whose interval has elapsed since its last run is due.
func TestIsDue_DueWhenElapsed(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	s := New(newMemStore(now))
	job := Job{
		Name:            "news-refresh",
		Enabled:         true,
		IntervalMinutes: 10,
		LastRunAt:       ptrTime(now.Add(-15 * time.Minute)),
	}
	if !s.IsDue(job, now) {
		t.Fatalf("expected job due: 15m elapsed >= 10m interval")
	}
}

// SC2 — a disabled job, or one whose interval has NOT elapsed, is not due.
func TestIsDue_SkipWhenDisabledOrNotDue(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	s := New(newMemStore(now))

	disabled := Job{Name: "x", Enabled: false, IntervalMinutes: 10, LastRunAt: ptrTime(now.Add(-1 * time.Hour))}
	if s.IsDue(disabled, now) {
		t.Fatalf("disabled job must never be due")
	}

	notYet := Job{Name: "y", Enabled: true, IntervalMinutes: 30, LastRunAt: ptrTime(now.Add(-5 * time.Minute))}
	if s.IsDue(notYet, now) {
		t.Fatalf("job not due: 5m elapsed < 30m interval")
	}

	// run_requested_at forces due regardless of interval (the manual "Run now" path).
	requested := notYet
	requested.RunRequestedAt = ptrTime(now)
	if !s.IsDue(requested, now) {
		t.Fatalf("a run-requested job must be due regardless of interval")
	}
}

// SC3 — Due filters to registered jobs that are enabled-and-due OR run-requested,
// and excludes jobs with an open run.
func TestDue_FiltersRegisteredAndDueOrRequested(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	store := newMemStore(now)

	// due by interval, registered → included
	store.put(Job{Name: "due-interval", Enabled: true, IntervalMinutes: 10, LastRunAt: ptrTime(now.Add(-20 * time.Minute))})
	// not due, but run-requested, registered → included
	store.put(Job{Name: "requested", Enabled: true, IntervalMinutes: 60, LastRunAt: ptrTime(now.Add(-1 * time.Minute)), RunRequestedAt: ptrTime(now)})
	// due by interval but NOT registered → excluded
	store.put(Job{Name: "unregistered", Enabled: true, IntervalMinutes: 10, LastRunAt: ptrTime(now.Add(-20 * time.Minute))})
	// due by interval, registered, but already has an open run → excluded
	store.put(Job{Name: "has-open-run", Enabled: true, IntervalMinutes: 10, LastRunAt: ptrTime(now.Add(-20 * time.Minute))})
	store.open["has-open-run"] = true
	// disabled, registered → excluded
	store.put(Job{Name: "disabled", Enabled: false, IntervalMinutes: 10, LastRunAt: ptrTime(now.Add(-20 * time.Minute))})

	s := New(store)
	registered := []ScheduleJobName{"due-interval", "requested", "has-open-run", "disabled"}
	got, err := s.Due(context.Background(), now, registered)
	if err != nil {
		t.Fatalf("Due error: %v", err)
	}

	names := map[ScheduleJobName]bool{}
	for _, j := range got {
		names[j.Name] = true
	}
	if !names["due-interval"] {
		t.Errorf("expected due-interval included")
	}
	if !names["requested"] {
		t.Errorf("expected run-requested job included")
	}
	if names["unregistered"] {
		t.Errorf("unregistered job must be excluded")
	}
	if names["has-open-run"] {
		t.Errorf("job with open run must be excluded")
	}
	if names["disabled"] {
		t.Errorf("disabled job must be excluded")
	}
}

// SC4 — two concurrent Start calls on one job → exactly one claims the run.
func TestStart_SingleFlightClaim(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	store := newMemStore(now)
	store.put(Job{Name: "backup", Enabled: true, IntervalMinutes: 5})
	s := New(store)

	var claims int32
	var wg sync.WaitGroup
	const goroutines = 8
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _, claimed, err := s.Start(context.Background(), "backup")
			if err != nil {
				t.Errorf("Start error: %v", err)
				return
			}
			if claimed {
				atomic.AddInt32(&claims, 1)
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&claims); got != 1 {
		t.Fatalf("expected exactly 1 claim, got %d", got)
	}
}

// SC5 — ClampInterval raises any value below the workspace floor to the minimum.
func TestClampInterval_BelowMinimum(t *testing.T) {
	cases := []struct{ in, want int }{
		{0, MinIntervalMinutes},
		{-5, MinIntervalMinutes},
		{MinIntervalMinutes, MinIntervalMinutes},
		{10, 10},
	}
	for _, c := range cases {
		if got := ClampInterval(c.in); got != c.want {
			t.Errorf("ClampInterval(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

// A claim consuming a pending manual "Run now" reports TriggerManual; an
// ordinary interval claim reports TriggerSchedule (2607-004 #6c — the executing
// recipe labels its artifact by this).
func TestStart_TriggerReflectsManualRequest(t *testing.T) {
	now := time.Date(2026, 7, 3, 3, 0, 0, 0, time.UTC)
	store := newMemStore(now)
	req := now.Add(-time.Minute)
	store.jobs["db-backup"] = Job{Name: "db-backup", Enabled: true, IntervalMinutes: 1440, RunRequestedAt: &req}
	s := New(store, WithNow(func() time.Time { return now }))

	_, trigger, claimed, err := s.Start(context.Background(), "db-backup")
	if err != nil || !claimed {
		t.Fatalf("claim failed: claimed=%v err=%v", claimed, err)
	}
	if trigger != TriggerManual {
		t.Fatalf("trigger = %q, want manual (run_requested_at was set)", trigger)
	}
	if store.jobs["db-backup"].RunRequestedAt != nil {
		t.Fatalf("run_requested_at must be consumed by the claim")
	}

	// A second job with no pending request claims as an ordinary schedule tick.
	store.jobs["news-refresh"] = Job{Name: "news-refresh", Enabled: true, IntervalMinutes: 60}
	_, trigger2, claimed2, err := s.Start(context.Background(), "news-refresh")
	if err != nil || !claimed2 {
		t.Fatalf("second claim failed: claimed=%v err=%v", claimed2, err)
	}
	if trigger2 != TriggerSchedule {
		t.Fatalf("trigger = %q, want schedule", trigger2)
	}
}
