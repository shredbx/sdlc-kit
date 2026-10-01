package calendar_test

import (
	"errors"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/calendar"
)

// baseEvent returns a minimal valid timed event for mutation in table tests.
func baseEvent() calendar.Event {
	start := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	return calendar.Event{
		ID:         "11111111-1111-1111-1111-111111111111",
		Start:      start,
		End:        start.Add(time.Hour),
		AllDay:     false,
		Title:      "Viewing — Unit A2",
		EventType:  calendar.EventTypeViewing,
		Status:     calendar.StatusScheduled,
		OwnerID:    "22222222-2222-2222-2222-222222222222",
		Visibility: calendar.VisibilityTeam,
	}
}

func TestEventStatus_Valid(t *testing.T) {
	valid := []calendar.EventStatus{
		calendar.StatusScheduled, calendar.StatusConfirmed, calendar.StatusDone,
		calendar.StatusCancelled, calendar.StatusNoShow,
	}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("status %q should be valid", s)
		}
	}
	if calendar.EventStatus("bogus").Valid() {
		t.Error("bogus status should be invalid")
	}
}

func TestVisibility_Valid(t *testing.T) {
	if !calendar.VisibilityTeam.Valid() || !calendar.VisibilityPrivate.Valid() {
		t.Error("team and private must be valid visibilities")
	}
	if calendar.Visibility("public").Valid() {
		t.Error("public is not a valid visibility")
	}
}

func TestReminderChannel_Valid(t *testing.T) {
	if !calendar.ChannelInApp.Valid() || !calendar.ChannelEmail.Valid() {
		t.Error("in_app and email must be valid channels")
	}
	if calendar.ReminderChannel("sms").Valid() {
		t.Error("sms is not a valid channel")
	}
}

func TestEvent_Validate_OK(t *testing.T) {
	e := baseEvent()
	if err := e.Validate(); err != nil {
		t.Fatalf("valid event should pass, got %v", err)
	}
}

func TestEvent_Validate_EndAfterStart(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*calendar.Event)
		wantErr error
	}{
		{"end before start", func(e *calendar.Event) { e.End = e.Start.Add(-time.Hour) }, calendar.ErrEndNotAfterStart},
		{"end equals start", func(e *calendar.Event) { e.End = e.Start }, calendar.ErrEndNotAfterStart},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := baseEvent()
			tt.mutate(&e)
			err := e.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("want %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestEvent_Validate_AllDayEndAfterStart(t *testing.T) {
	// All-day: end is the EXCLUSIVE day-after, so a one-day event has end = start + 24h.
	e := baseEvent()
	e.AllDay = true
	e.Start = time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	e.End = e.Start.AddDate(0, 0, 1)
	if err := e.Validate(); err != nil {
		t.Fatalf("valid all-day event should pass, got %v", err)
	}
}

func TestEvent_Validate_RequiresStatus(t *testing.T) {
	e := baseEvent()
	e.Status = calendar.EventStatus("nope")
	if err := e.Validate(); !errors.Is(err, calendar.ErrInvalidStatus) {
		t.Errorf("want ErrInvalidStatus, got %v", err)
	}
	e2 := baseEvent()
	e2.Status = ""
	if err := e2.Validate(); !errors.Is(err, calendar.ErrInvalidStatus) {
		t.Errorf("empty status should fail, got %v", err)
	}
}

func TestEvent_Validate_RequiresVisibility(t *testing.T) {
	e := baseEvent()
	e.Visibility = calendar.Visibility("everyone")
	if err := e.Validate(); !errors.Is(err, calendar.ErrInvalidVisibility) {
		t.Errorf("want ErrInvalidVisibility, got %v", err)
	}
}

func TestEvent_Validate_RequiresEventType(t *testing.T) {
	e := baseEvent()
	e.EventType = ""
	if err := e.Validate(); !errors.Is(err, calendar.ErrEventTypeRequired) {
		t.Errorf("want ErrEventTypeRequired, got %v", err)
	}
}

func TestEvent_Validate_RequiresOwner(t *testing.T) {
	e := baseEvent()
	e.OwnerID = ""
	if err := e.Validate(); !errors.Is(err, calendar.ErrOwnerRequired) {
		t.Errorf("want ErrOwnerRequired, got %v", err)
	}
}

func TestReminder_Validate(t *testing.T) {
	tests := []struct {
		name    string
		r       calendar.Reminder
		wantErr error
	}{
		{"ok", calendar.Reminder{ID: "r1", EventID: "e1", LeadMinutes: 30, Channel: calendar.ChannelInApp}, nil},
		{"zero lead ok", calendar.Reminder{ID: "r1", EventID: "e1", LeadMinutes: 0, Channel: calendar.ChannelEmail}, nil},
		{"negative lead", calendar.Reminder{ID: "r1", EventID: "e1", LeadMinutes: -5, Channel: calendar.ChannelInApp}, calendar.ErrNegativeLead},
		{"bad channel", calendar.Reminder{ID: "r1", EventID: "e1", LeadMinutes: 10, Channel: calendar.ReminderChannel("push")}, calendar.ErrInvalidChannel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.r.Validate()
			if tt.wantErr == nil && err != nil {
				t.Fatalf("want nil, got %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestEvent_Validate_ChildrenChecked(t *testing.T) {
	e := baseEvent()
	e.Reminders = []calendar.Reminder{{ID: "r1", EventID: e.ID, LeadMinutes: -1, Channel: calendar.ChannelInApp}}
	if err := e.Validate(); !errors.Is(err, calendar.ErrNegativeLead) {
		t.Errorf("invalid child reminder should fail parent Validate, got %v", err)
	}

	e2 := baseEvent()
	e2.References = []calendar.Reference{{ID: "x1", RefType: "contact", RefID: "c1", Label: ""}}
	if err := e2.Validate(); !errors.Is(err, calendar.ErrReferenceIncomplete) {
		t.Errorf("reference with empty label should fail, got %v", err)
	}
}

func TestReference_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ref     calendar.Reference
		wantErr error
	}{
		{"ok", calendar.Reference{ID: "x1", RefType: "contact", RefID: "c1", Label: "Somchai Jaidee"}, nil},
		{"missing ref_type", calendar.Reference{ID: "x1", RefType: "", RefID: "c1", Label: "L"}, calendar.ErrReferenceIncomplete},
		{"missing ref_id", calendar.Reference{ID: "x1", RefType: "contact", RefID: "", Label: "L"}, calendar.ErrReferenceIncomplete},
		{"missing label", calendar.Reference{ID: "x1", RefType: "contact", RefID: "c1", Label: ""}, calendar.ErrReferenceIncomplete},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ref.Validate()
			if tt.wantErr == nil && err != nil {
				t.Fatalf("want nil, got %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
		})
	}
}

// BindRef is the Bindable structural surface: an Event IS a Source addressable by
// a stable typed (kind, id). Verifies the method set conforms.
func TestEvent_BindableSurface(t *testing.T) {
	e := baseEvent()
	if got := e.BindKind(); got != calendar.SourceKindEvent {
		t.Errorf("want kind %q, got %q", calendar.SourceKindEvent, got)
	}
	if got := e.BindID(); got != e.ID {
		t.Errorf("want bind id %q, got %q", e.ID, got)
	}
}
