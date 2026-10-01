package feed

import (
	"testing"
	"time"
)

// DueNow is the pure --if-due gate decision (#7). Cover the four cases:
// disabled, never-run, due, not-due.
func TestDueNow(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	hourAgo := now.Add(-60 * time.Minute)
	tenMinAgo := now.Add(-10 * time.Minute)

	tests := []struct {
		name     string
		enabled  bool
		last     *time.Time
		interval int
		want     bool
	}{
		{"disabled never runs", false, nil, 60, false},
		{"disabled even when overdue", false, &hourAgo, 30, false},
		{"never run is always due", true, nil, 60, true},
		{"due — interval elapsed", true, &hourAgo, 30, true},
		{"due — exactly at interval", true, &hourAgo, 60, true},
		{"not due — within interval", true, &tenMinAgo, 60, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := DueNow(tc.enabled, tc.last, tc.interval, now); got != tc.want {
				t.Fatalf("DueNow(%v, %v, %d) = %v, want %v", tc.enabled, tc.last, tc.interval, got, tc.want)
			}
		})
	}
}
