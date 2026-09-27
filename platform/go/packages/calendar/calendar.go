// Package calendar provides the team-calendar Event domain type (PD layer) — a
// scheduled occurrence (viewing, meeting, key handover, callback) owned by a team
// member, that attaches any Bindable Source by reference.
//
// Event is the Go projection of the Fabric `calendar` module
// (.sbx/workspace/fabric/modules/calendar.yml), which exposes `Event as [Bindable]`,
// and of the shared-type entity.yml at
// .sbx/workspace/packages/core/go/types/calendar/entity.yml. It is the backend
// counterpart of the frontend @sbx/ui-calendar CalendarItem contract — the Phase 3b
// HTTP adapter translates this type 1:1 onto that wire shape.
//
// Named types replace raw strings for every discriminator field:
//
//	EventStatus, Visibility, EventType, ReminderChannel, SourceKind
//
// Bindable conformance is STRUCTURAL (FI-1 / FI-2 — protocols are the integration
// currency; no Go interface to implement, no registry, no base class — exactly as
// pkg/seo.SeoMeta and pkg/property.PropertyUnit conform). An Event both IS a Source
// (a stable UUID identity exposed via BindKind/BindID) and HOLDS bindings (typed
// References to independent-lifecycle Sources, held by reference, never copied —
// FI-4 / CAL-R1).
//
// This is the PD layer: ZERO storage imports (no database/sql, no pgx). The postgres
// mapper, the HTTP adapter, and the references ChildStore are a deferred Phase 3b.
//
// Example:
//
//	start := time.Now().Add(24 * time.Hour)
//	e := calendar.Event{
//	    ID:         id,
//	    Start:      start,
//	    End:        start.Add(time.Hour),
//	    EventType:  calendar.EventTypeViewing,
//	    Status:     calendar.StatusScheduled,
//	    OwnerID:    ownerID,
//	    Visibility: calendar.VisibilityTeam,
//	    Title:      "Viewing — Unit A2",
//	}
//	if err := e.Validate(); err != nil { ... }
package calendar

import (
	"errors"
	"strings"
	"time"
)

// =============================================================================
// SENTINEL ERRORS — callers test with errors.Is (mirrors pkg/property / pkg/seo)
// =============================================================================

// ErrEndNotAfterStart indicates an Event whose End is not strictly after its
// Start. A zero-length or negative window is nonsensical; for an all-day event
// the End is the EXCLUSIVE day-after, so a single-day event has End = Start+24h.
var ErrEndNotAfterStart = errors.New("event end must be strictly after start")

// ErrInvalidStatus indicates a Status that is not one of the five soft booking
// states (scheduled, confirmed, done, cancelled, no_show). Status is always
// present — an empty status is invalid.
var ErrInvalidStatus = errors.New("event status must be one of scheduled, confirmed, done, cancelled, no_show")

// ErrInvalidVisibility indicates a Visibility that is neither team nor private.
var ErrInvalidVisibility = errors.New("event visibility must be team or private")

// ErrEventTypeRequired indicates an Event with no event_type code. The type is a
// required dictionary discriminator (viewing, meeting, …).
var ErrEventTypeRequired = errors.New("event_type is required")

// ErrOwnerRequired indicates an Event with no owner_id. Every slot has an owner
// (the team member who owns it).
var ErrOwnerRequired = errors.New("owner_id is required")

// ErrNegativeLead indicates a Reminder whose LeadMinutes is negative. A reminder
// fires a non-negative number of minutes BEFORE the event start (0 = at start).
var ErrNegativeLead = errors.New("reminder lead_minutes must be >= 0")

// ErrInvalidChannel indicates a Reminder whose Channel is not a known delivery
// channel (in_app, email).
var ErrInvalidChannel = errors.New("reminder channel must be in_app or email")

// ErrReferenceIncomplete indicates a Reference missing one of its required fields
// (ref_type, ref_id, label). The cached label is required so the chip survives a
// later delete of the target entity (D11).
var ErrReferenceIncomplete = errors.New("reference requires ref_type, ref_id, and a cached label")

// =============================================================================
// NAMED TYPES — discriminator fields (VARCHAR FK to dictionary tables)
// =============================================================================

// EventStatus is the soft booking lifecycle state of an event. The five states
// mirror the frontend contract (@sbx/ui-calendar EventStatus):
// scheduled → confirmed → done | cancelled | no_show. A cancelled event is
// greyed, never hard-deleted. FK to the project-scoped event_statuses dictionary.
type EventStatus string

const (
	StatusScheduled EventStatus = "scheduled"
	StatusConfirmed EventStatus = "confirmed"
	StatusDone      EventStatus = "done"
	StatusCancelled EventStatus = "cancelled"
	StatusNoShow    EventStatus = "no_show"
)

// Valid reports whether s is one of the five soft booking states — a parse-time
// 422 guard; the FK enforces the same rule at the DB layer.
func (s EventStatus) Valid() bool {
	switch s {
	case StatusScheduled, StatusConfirmed, StatusDone, StatusCancelled, StatusNoShow:
		return true
	}
	return false
}

// Visibility controls who may see an event — team (everyone on the team) or
// private (owner only). Enforcement happens in the HTTP adapter at the boundary
// (FI-6), NOT in this PD type; the type only carries the value as data.
type Visibility string

const (
	VisibilityTeam    Visibility = "team"
	VisibilityPrivate Visibility = "private"
)

// Valid reports whether v is team or private.
func (v Visibility) Valid() bool {
	switch v {
	case VisibilityTeam, VisibilityPrivate:
		return true
	}
	return false
}

// EventType classifies what kind of occurrence an event is. FK to the
// project-scoped event_types dictionary — package-level constants are the names a
// project seeds. The host owns the dictionary, so the closed set is not enforced
// here beyond "non-empty"; the constants document the canonical v1 vocabulary.
type EventType string

const (
	EventTypeViewing  EventType = "viewing"
	EventTypeMeeting  EventType = "meeting"
	EventTypeHandover EventType = "handover"
	EventTypeCallback EventType = "callback"
	EventTypeOther    EventType = "other"
)

// ReminderChannel is the delivery channel of a reminder.
type ReminderChannel string

const (
	ChannelInApp ReminderChannel = "in_app"
	ChannelEmail ReminderChannel = "email"
)

// Valid reports whether c is a known delivery channel.
func (c ReminderChannel) Valid() bool {
	switch c {
	case ChannelInApp, ChannelEmail:
		return true
	}
	return false
}

// SourceKind is the Bindable Source-kind code an entity reports for itself — the
// "kind" half of the (kind, id) pair that identifies a Source to other modules
// (Canvas, Calendar, pickers). An Event's own kind is SourceKindEvent.
type SourceKind string

const (
	// SourceKindEvent is the Source kind an Event reports via BindKind().
	SourceKindEvent SourceKind = "event"
)

// =============================================================================
// REFERENCE — a typed binding to an independent-lifecycle Source (held, not copied)
// =============================================================================

// Reference is a cached, render-ready link from an Event to an
// independent-lifecycle entity (a person or a thing). It is the Go projection of
// the frontend RefChip contract (@sbx/ui-calendar). An Event holds References by
// typed ref (RefType + RefID) plus a cached Label snapshotted at link time — the
// label SURVIVES a later delete of the target entity (D11), and the source data is
// resolved live, never copied into the event (FI-4 / CAL-R1).
//
// References have INDEPENDENT lifecycle (Decision #0019 three-map model): deleting
// the target does not delete the Event, and deleting the Event does not delete the
// target. The Go ChildStore wiring (join table + companion Load/Add/Remove) is a
// deferred Phase 3b; this struct carries the slice now so the mapper/adapter
// populate it later.
type Reference struct {
	ID       string `json:"referenceId" yaml:"id"`
	RefType  string `json:"refType" yaml:"ref_type"`
	RefID    string `json:"refId" yaml:"ref_id"`
	Relation string `json:"relation,omitempty" yaml:"relation,omitempty"`
	AsType   string `json:"asType,omitempty" yaml:"as_type,omitempty"`
	Label    string `json:"label" yaml:"label"`
	Subtitle string `json:"subtitle,omitempty" yaml:"subtitle,omitempty"`
}

// Validate checks the reference carries its required identity bits: a ref_type, a
// ref_id, and a cached label. Returns ErrReferenceIncomplete on the first gap.
func (r Reference) Validate() error {
	if strings.TrimSpace(r.RefType) == "" ||
		strings.TrimSpace(r.RefID) == "" ||
		strings.TrimSpace(r.Label) == "" {
		return ErrReferenceIncomplete
	}
	return nil
}

// =============================================================================
// REMINDER — a timed child nudge (containment, cascade-delete with the event)
// =============================================================================

// Reminder is a timed nudge tied to one Event — it fires LeadMinutes before the
// event start, via Channel. It is a containment child (Decision #0019): own UUID
// identity, 1:many, cascade-deletes with the parent event. Lead time is whole
// minutes-before-start (0 = at start), the smallest unit the UI offers; the actual
// delivery is a deferred adapter concern (FI-5) — this PD type only carries the
// schedule.
type Reminder struct {
	ID          string          `json:"id" yaml:"id"`
	EventID     string          `json:"event_id" yaml:"event_id"`
	LeadMinutes int             `json:"lead_minutes" yaml:"lead_minutes"`
	Channel     ReminderChannel `json:"channel" yaml:"channel"`
}

// LeadDuration returns the lead time as a time.Duration before the event start.
func (r Reminder) LeadDuration() time.Duration {
	return time.Duration(r.LeadMinutes) * time.Minute
}

// Validate enforces a non-negative lead time and a known delivery channel.
func (r Reminder) Validate() error {
	if r.LeadMinutes < 0 {
		return ErrNegativeLead
	}
	if !r.Channel.Valid() {
		return ErrInvalidChannel
	}
	return nil
}

// =============================================================================
// EVENT — document entity
// =============================================================================

// Event is the calendar's core scheduled-occurrence entity. It extends the
// Document foundation type (UUID identity, timestamps, soft delete). Time is held
// as time.Time; AllDay marks a whole-day span (End is the exclusive day-after).
// Status and Visibility are always present (defaulted at the DB layer); Title and
// Notes are optional content.
type Event struct {
	ID string `json:"id" yaml:"id"`

	// Time window. For a timed event Start/End are instants; for an all-day event
	// they are dates and End is the EXCLUSIVE day-after (RFC 5545 DTEND semantics).
	Start  time.Time `json:"start" yaml:"start"`
	End    time.Time `json:"end" yaml:"end"`
	AllDay bool      `json:"allDay" yaml:"all_day"`

	// Human content. Title is optional (a typed slot may have no title yet).
	Title string `json:"title,omitempty" yaml:"title,omitempty"`
	Notes string `json:"notes,omitempty" yaml:"notes,omitempty"`

	// Discriminators (named types — never raw strings).
	EventType  EventType   `json:"eventType" yaml:"event_type"`
	Status     EventStatus `json:"status" yaml:"status"`
	Visibility Visibility  `json:"visibility" yaml:"visibility"`

	// OwnerID is the UUID FK → users — the team member who owns the slot. Required.
	OwnerID string `json:"owner_id" yaml:"owner_id"`

	// Reminders is the event's child nudge collection — cascade-deletes with the
	// event (Decision #0019 containment). Companion-loaded in Phase 3b.
	Reminders []Reminder `json:"reminders,omitempty" yaml:"reminders,omitempty"`

	// References is the event's attendee + attachment links — typed refs to
	// independent-lifecycle Sources, held by reference (never copied). Mirrors the
	// frontend RefChip[] contract. Companion-loaded in Phase 3b.
	References []Reference `json:"refs,omitempty" yaml:"references,omitempty"`

	// Document base.
	CreatedAt time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" yaml:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty"`
}

// BindKind reports the event's Bindable Source kind. Part of the structural
// Bindable surface (FI-1 / FI-2): an Event IS a Source identified by (BindKind,
// BindID). No interface is implemented — conformance is the method set.
func (e Event) BindKind() SourceKind { return SourceKindEvent }

// BindID reports the event's stable Source identity (its UUID). Part of the
// structural Bindable surface.
func (e Event) BindID() string { return e.ID }

// ValidateWindow returns ErrEndNotAfterStart unless End is strictly after Start.
// This holds for both timed and all-day events (all-day End is the exclusive
// day-after, so a single-day event still has End > Start by 24h).
func (e Event) ValidateWindow() error {
	if !e.End.After(e.Start) {
		return ErrEndNotAfterStart
	}
	return nil
}

// Validate aggregates all event-level invariants and recurses into children.
// Callers (REST handlers, CLI tools, importers) get a single entry point; it
// returns the first failing invariant.
func (e Event) Validate() error {
	if err := e.ValidateWindow(); err != nil {
		return err
	}
	if strings.TrimSpace(string(e.EventType)) == "" {
		return ErrEventTypeRequired
	}
	if !e.Status.Valid() {
		return ErrInvalidStatus
	}
	if !e.Visibility.Valid() {
		return ErrInvalidVisibility
	}
	if strings.TrimSpace(e.OwnerID) == "" {
		return ErrOwnerRequired
	}
	for i := range e.Reminders {
		if err := e.Reminders[i].Validate(); err != nil {
			return err
		}
	}
	for i := range e.References {
		if err := e.References[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}
