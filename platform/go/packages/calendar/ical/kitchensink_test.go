package ical_test

import (
	"strings"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/calendar"
	"github.com/shredbx/sbx-core/pkg/calendar/ical"
)

// (losslessness gate — kitchen-sink round-trip tests for the iCal mapper)

// kitchenSink builds an Event with EVERY field populated: every scalar non-zero,
// every slice with >=2 elements, and special characters (comma, semicolon,
// newline, backslash) in every free-text field. It is the input to the
// losslessness gate below — if the mapper silently drops any field, the
// field-by-field comparison in TestKitchenSink_RoundTrip fails.
func kitchenSink() calendar.Event {
	start := time.Date(2026, 6, 15, 10, 30, 0, 0, time.UTC)
	return calendar.Event{
		ID:         "11111111-1111-1111-1111-111111111111",
		Start:      start,
		End:        start.Add(90 * time.Minute),
		AllDay:     false,
		Title:      `Viewing, Unit A2; "north" wing\path` + "\nbring keys",
		Notes:      "buyers: A, B, C; cash\nready\\now",
		EventType:  calendar.EventTypeViewing,
		Status:     calendar.StatusNoShow, // a state with NO standard STATUS inverse — X- carrier is the only lossless path
		Visibility: calendar.VisibilityPrivate,
		OwnerID:    "22222222-2222-2222-2222-222222222222",
		Reminders: []calendar.Reminder{
			{ID: "rem-1", EventID: "11111111-1111-1111-1111-111111111111", LeadMinutes: 30, Channel: calendar.ChannelInApp},
			{ID: "rem-2", EventID: "11111111-1111-1111-1111-111111111111", LeadMinutes: 1440, Channel: calendar.ChannelEmail},
		},
		References: []calendar.Reference{
			{
				ID:       "ref-1",
				RefType:  "contact",
				RefID:    "c-aaaa",
				Relation: "attendee",
				AsType:   "buyer",
				Label:    `Smith, Jones & Co; "VIP"\client`,
				Subtitle: "primary; cash buyer, ready",
			},
			{
				ID:       "ref-2",
				RefType:  "property",
				RefID:    "p-bbbb",
				Relation: "subject",
				AsType:   "listing",
				Label:    "Unit A2\nKo Pha-ngan",
				Subtitle: `villa\seaview, 2BR`,
			},
		},
	}
}

// assertEventDeepEqual compares every field of two events. It is the losslessness
// contract: it must fail if ANY field of want is not reproduced in got.
//
// The ONLY allowed exceptions, each asserted/skipped explicitly with a comment:
//   - DTSTAMP: the marshalling instant (time.Now at ToVEVENT) — it is NOT a field
//     of Event, so there is nothing to compare; it is documented here for the
//     reader and is the iCal counterpart of the vCard child-Note identity carve-out.
//   - CreatedAt / UpdatedAt / DeletedAt: the Document base timestamps have NO
//     iCalendar wire slot (DTSTAMP is the object-creation instant, not CreatedAt;
//     LAST-MODIFIED is not the soft-delete column). They are intentionally NOT
//     carried — the sync layer reconciles Document-base columns out of band, not
//     through the iCal body. They are excluded here on purpose, never silently.
func assertEventDeepEqual(t *testing.T, want, got calendar.Event) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("ID: want %q, got %q", want.ID, got.ID)
	}
	if !got.Start.Equal(want.Start) {
		t.Errorf("Start: want %s, got %s", want.Start, got.Start)
	}
	if !got.End.Equal(want.End) {
		t.Errorf("End: want %s, got %s", want.End, got.End)
	}
	if got.AllDay != want.AllDay {
		t.Errorf("AllDay: want %v, got %v", want.AllDay, got.AllDay)
	}
	if got.Title != want.Title {
		t.Errorf("Title: want %q, got %q", want.Title, got.Title)
	}
	if got.Notes != want.Notes {
		t.Errorf("Notes: want %q, got %q", want.Notes, got.Notes)
	}
	if got.EventType != want.EventType {
		t.Errorf("EventType: want %q, got %q", want.EventType, got.EventType)
	}
	if got.Status != want.Status {
		t.Errorf("Status: want %q, got %q", want.Status, got.Status)
	}
	if got.Visibility != want.Visibility {
		t.Errorf("Visibility: want %q, got %q", want.Visibility, got.Visibility)
	}
	if got.OwnerID != want.OwnerID {
		t.Errorf("OwnerID: want %q, got %q", want.OwnerID, got.OwnerID)
	}

	// Reminders — full struct, in order.
	if len(got.Reminders) != len(want.Reminders) {
		t.Fatalf("Reminders len: want %d, got %d", len(want.Reminders), len(got.Reminders))
	}
	for i := range want.Reminders {
		if got.Reminders[i] != want.Reminders[i] {
			t.Errorf("Reminders[%d]: want %+v, got %+v", i, want.Reminders[i], got.Reminders[i])
		}
	}

	// References — full struct (all 7 sub-fields), in order.
	if len(got.References) != len(want.References) {
		t.Fatalf("References len: want %d, got %d", len(want.References), len(got.References))
	}
	for i := range want.References {
		if got.References[i] != want.References[i] {
			t.Errorf("References[%d]:\n want %+v\n got  %+v", i, want.References[i], got.References[i])
		}
	}
}

// TestKitchenSink_RoundTrip is the losslessness gate for the iCal mapper: every
// field of a fully-populated Event must survive FromVEVENT(ToVEVENT(e)). A
// field-SUBSET test would pass while silently dropping fields — this test would
// not. It is the regression guard for the Visibility/OwnerID/References/Reminders
// gaps closed in this pass.
func TestKitchenSink_RoundTrip(t *testing.T) {
	e := kitchenSink()
	out := ical.ToVEVENT(e)
	got, err := ical.FromVEVENT(out)
	if err != nil {
		t.Fatalf("FromVEVENT error: %v\n--- block ---\n%s", err, out)
	}
	assertEventDeepEqual(t, e, got)
}

// TestKitchenSink_AllDay re-runs the gate for an all-day event so the date-only
// path is also proven lossless across the full field set.
func TestKitchenSink_AllDay(t *testing.T) {
	e := kitchenSink()
	e.AllDay = true
	e.Start = time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	e.End = e.Start.AddDate(0, 0, 3) // three-day all-day, exclusive end
	out := ical.ToVEVENT(e)
	got, err := ical.FromVEVENT(out)
	if err != nil {
		t.Fatalf("FromVEVENT error: %v\n--- block ---\n%s", err, out)
	}
	assertEventDeepEqual(t, e, got)
}

// TestKitchenSink_StandardProperties asserts the standard (non-X-) carriers the
// review requires: CLASS for Visibility and VALARM for each Reminder. These exist
// so foreign calendar apps render correctly even without reading the X- lines.
func TestKitchenSink_StandardProperties(t *testing.T) {
	e := kitchenSink() // VisibilityPrivate, 2 reminders (30m in_app, 1440m email)
	block := ical.ToVEVENT(e)

	if !strings.Contains(block, "CLASS:PRIVATE") {
		t.Errorf("private visibility must emit CLASS:PRIVATE, got:\n%s", block)
	}

	// The event type ALSO emits a standard CATEGORIES line (interop) — kitchenSink is a
	// viewing. X-SBX-EVENT-TYPE stays the authoritative carrier; CATEGORIES is for
	// foreign-app grouping and is ignored on parse.
	if !strings.Contains(block, "CATEGORIES:viewing") {
		t.Errorf("event type must emit CATEGORIES:viewing, got:\n%s", block)
	}

	// Each reminder maps to a standard VALARM with a relative TRIGGER.
	if n := strings.Count(block, "BEGIN:VALARM"); n != 2 {
		t.Errorf("want 2 VALARM blocks, got %d:\n%s", n, block)
	}
	for _, want := range []string{
		"BEGIN:VALARM", "ACTION:DISPLAY", "TRIGGER:-PT30M", "X-SBX-CHANNEL:in_app",
		"TRIGGER:-PT1440M", "X-SBX-CHANNEL:email", "END:VALARM",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("VALARM missing %q in:\n%s", want, block)
		}
	}
}

// TestKitchenSink_ViaCalendar runs the full-field gate through the VCALENDAR
// wrapper (ToCalendar/FromCalendar) — the path the sync layer uses — to prove the
// nested VALARM sub-blocks survive multi-event extraction, not just bare FromVEVENT.
func TestKitchenSink_ViaCalendar(t *testing.T) {
	e := kitchenSink()
	cal := ical.ToCalendar(e)
	events, err := ical.FromCalendar(cal)
	if err != nil {
		t.Fatalf("FromCalendar error: %v\n--- cal ---\n%s", err, cal)
	}
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	assertEventDeepEqual(t, e, events[0])
}

// TestReference_NonAttendeeOmitsATTENDEE asserts the interop ATTENDEE line is
// emitted ONLY for attendee-relation references — a subject/attachment ref carries
// no ATTENDEE — and that every reference still round-trips losslessly via X-SBX-REF.
func TestReference_NonAttendeeOmitsATTENDEE(t *testing.T) {
	e := kitchenSink()
	e.References = []calendar.Reference{
		{ID: "ref-x", RefType: "property", RefID: "p-1", Relation: "subject", Label: "Unit B"},
	}
	block := ical.ToVEVENT(e)
	if strings.Contains(block, "ATTENDEE") {
		t.Errorf("non-attendee reference must not emit ATTENDEE, got:\n%s", block)
	}
	got, err := ical.FromVEVENT(block)
	if err != nil {
		t.Fatalf("FromVEVENT error: %v", err)
	}
	if len(got.References) != 1 || got.References[0] != e.References[0] {
		t.Errorf("reference round-trip wrong: %+v", got.References)
	}
}

// TestReference_OmittedOptionalSubfields proves a reference with only its required
// identity (refType, refId, label) round-trips — the omitempty JSON fields decode
// back to empty strings, not garbage.
func TestReference_OmittedOptionalSubfields(t *testing.T) {
	e := kitchenSink()
	e.References = []calendar.Reference{
		{ID: "", RefType: "contact", RefID: "c-9", Relation: "", AsType: "", Label: "Lek", Subtitle: ""},
	}
	got := func() calendar.Event {
		out := ical.ToVEVENT(e)
		g, err := ical.FromVEVENT(out)
		if err != nil {
			t.Fatalf("FromVEVENT error: %v", err)
		}
		return g
	}()
	if len(got.References) != 1 || got.References[0] != e.References[0] {
		t.Errorf("minimal reference round-trip wrong:\n want %+v\n got  %+v", e.References[0], got.References)
	}
}

// TestKitchenSink_VisibilityClass proves the 2-value CLASS mapping is lossless in
// both directions (team<->PUBLIC, private<->PRIVATE).
func TestKitchenSink_VisibilityClass(t *testing.T) {
	cases := map[calendar.Visibility]string{
		calendar.VisibilityTeam:    "CLASS:PUBLIC",
		calendar.VisibilityPrivate: "CLASS:PRIVATE",
	}
	for vis, wantLine := range cases {
		e := kitchenSink()
		e.Visibility = vis
		block := ical.ToVEVENT(e)
		if !strings.Contains(block, wantLine) {
			t.Errorf("visibility %q: want %q in:\n%s", vis, wantLine, block)
		}
		got, err := ical.FromVEVENT(block)
		if err != nil {
			t.Fatalf("FromVEVENT error: %v", err)
		}
		if got.Visibility != vis {
			t.Errorf("visibility %q did not round-trip, got %q", vis, got.Visibility)
		}
	}
}
