package feed

import "time"

// DueNow is the pure decision for the `refresh-feeds --if-due` cron self-throttle.
// It returns true only when auto-fetch is enabled AND either it has never run
// (last == nil) or at least `interval` minutes have elapsed since the last run.
//
// Extracted as a pure function so the gate logic is unit-testable without a DB or
// a clock: the caller passes now (and the persisted last/interval), and DueNow
// makes the decision. A non-positive interval is treated as "always due when
// enabled" (defensive — the DB CHECK keeps interval >= 5 in practice).
func DueNow(enabled bool, last *time.Time, intervalMinutes int, now time.Time) bool {
	if !enabled {
		return false
	}
	if last == nil {
		return true
	}
	if intervalMinutes <= 0 {
		return true
	}
	return now.Sub(*last) >= time.Duration(intervalMinutes)*time.Minute
}
