// Package notify provides a small, provider-agnostic notification abstraction:
// a Notifier interface plus pluggable adapters (Telegram, SMTP) and composition
// helpers (Noop, Multi).
//
// It is a shared sbx-core primitive — any app that needs to push a short
// message to an operator channel (a new lead arrived, a job failed) depends on
// the Notifier interface and is handed a concrete adapter at wiring time. The
// adapters never reach back into a domain; they only know Message.
//
// Design rules honored here:
//   - No raw strings as behavior discriminators — Message is a typed value.
//   - No `any` — the interface is concrete; adapters carry their own config.
//   - Best-effort fan-out (MultiNotifier) logs and continues, so one dead
//     channel never silences the others nor blocks the caller's main flow.
package notify

import "context"

// Message is a single notification payload. Title is the headline, Body is the
// free-text detail, and Fields carries optional key/value rows an adapter may
// render (e.g. "Email" → "a@b.com"). All three are optional; an adapter must
// render gracefully when any are empty.
type Message struct {
	Title  string
	Body   string
	Fields map[string]string
}

// Notifier delivers a Message to some operator channel. Implementations must be
// safe for concurrent use and must not panic on a partially-filled Message.
type Notifier interface {
	Notify(ctx context.Context, msg Message) error
}

// NoopNotifier is the null Notifier: it accepts every Message and returns nil.
// It is the wiring default when no real channel is configured, so callers never
// branch on a nil Notifier.
type NoopNotifier struct{}

// Notify discards the message and reports success.
func (NoopNotifier) Notify(_ context.Context, _ Message) error { return nil }

// Compile-time checks.
var (
	_ Notifier = NoopNotifier{}
	_ Notifier = (*MultiNotifier)(nil)
)

// MultiNotifier fans a Message out to every wrapped Notifier, best-effort: it
// always attempts every target, logging (via the injected logf) and continuing
// past a failure. It returns the FIRST error encountered (or nil if all
// succeeded) so a caller can surface "at least one channel failed" without ever
// short-circuiting delivery to the healthy channels.
type MultiNotifier struct {
	targets []Notifier
	// logf logs a per-target failure. Defaults to a no-op so a zero-config
	// MultiNotifier is silent rather than nil-panicking.
	logf func(format string, args ...any)
}

// NewMultiNotifier composes the given notifiers into one best-effort target.
// logf may be nil (failures are then silent). Nil entries in targets are
// skipped so optional adapters can be passed unconditionally.
func NewMultiNotifier(logf func(format string, args ...any), targets ...Notifier) *MultiNotifier {
	clean := make([]Notifier, 0, len(targets))
	for _, t := range targets {
		if t != nil {
			clean = append(clean, t)
		}
	}
	return &MultiNotifier{targets: clean, logf: logf}
}

// Notify delivers to every target, returning the first error (nil if all OK).
func (m *MultiNotifier) Notify(ctx context.Context, msg Message) error {
	var first error
	for _, t := range m.targets {
		if err := t.Notify(ctx, msg); err != nil {
			if m.logf != nil {
				m.logf("notify: target %T failed: %v", t, err)
			}
			if first == nil {
				first = err
			}
		}
	}
	return first
}
