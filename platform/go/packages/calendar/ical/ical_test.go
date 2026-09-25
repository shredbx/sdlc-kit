package ical_test

import (
	"strings"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/calendar"
	"github.com/shredbx/sbx-core/pkg/calendar/ical"
)

// roundTrip marshals an event to a VEVENT block and parses it back, asserting the
// parsed event equals the source on every field the mapper carries. It is the core
// gate: FromVEVENT(ToVEVENT(e)) == e.
func roundTrip(t *testing.T, e calendar.Event) calendar.Event {
	t.Helper()
	out := ical.ToVEVENT(e)
	got, err := ical.FromVEVENT(out)
	if err != nil {
		t.Fatalf("FromVEVENT error: %v\n--- block ---\n%s", err, out)
	}
	return got
}

func assertEventEqual(t *testing.T, want, got calendar.Event) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("UID: want %q, got %q", want.ID, got.ID)
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
	if got.Status != want.Status {
		t.Errorf("Status: want %q, got %q", want.Status, got.Status)
	}
}

func timed() calendar.Event {
	start := time.Date(2026, 6, 15, 10, 30, 0, 0, time.UTC)
	return calendar.Event{
		ID:         "11111111-1111-1111-1111-111111111111",
		Start:      start,
		End:        start.Add(90 * time.Minute),
		AllDay:     false,
		Title:      "Viewing — Unit A2",
		Notes:      "Bring the keys",
		EventType:  calendar.EventTypeViewing,
		Status:     calendar.StatusScheduled,
		OwnerID:    "22222222-2222-2222-2222-222222222222",
		Visibility: calendar.VisibilityTeam,
	}
}

func TestRoundTrip_Timed(t *testing.T) {
	e := timed()
	got := roundTrip(t, e)
	assertEventEqual(t, e, got)
}

func TestRoundTrip_AllDay(t *testing.T) {
	e := timed()
	e.AllDay = true
	e.Start = time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	e.End = e.Start.AddDate(0, 0, 2) // two-day all-day event, exclusive end
	got := roundTrip(t, e)
	assertEventEqual(t, e, got)

	// An all-day VEVENT must use VALUE=DATE (date-only), not a Z timestamp.
	block := ical.ToVEVENT(e)
	if !strings.Contains(block, "DTSTART;VALUE=DATE:20260615") {
		t.Errorf("all-day DTSTART should be VALUE=DATE date-only, got:\n%s", block)
	}
	if !strings.Contains(block, "DTEND;VALUE=DATE:20260617") {
		t.Errorf("all-day DTEND should be exclusive day-after as VALUE=DATE, got:\n%s", block)
	}
}

func TestRoundTrip_EveryStatus(t *testing.T) {
	statuses := []calendar.EventStatus{
		calendar.StatusScheduled, calendar.StatusConfirmed, calendar.StatusDone,
		calendar.StatusCancelled, calendar.StatusNoShow,
	}
	for _, s := range statuses {
		t.Run(string(s), func(t *testing.T) {
			e := timed()
			e.Status = s
			got := roundTrip(t, e)
			if got.Status != s {
				t.Errorf("status %q did not round-trip, got %q", s, got.Status)
			}
		})
	}
}

func TestStatusMapping_StandardSTATUS(t *testing.T) {
	// The standard RFC 5545 STATUS line must be present and sensible per state, so
	// foreign calendar apps still get a usable status even without the X- property.
	cases := map[calendar.EventStatus]string{
		calendar.StatusScheduled: "STATUS:TENTATIVE",
		calendar.StatusConfirmed: "STATUS:CONFIRMED",
		calendar.StatusDone:      "STATUS:CONFIRMED",
		calendar.StatusCancelled: "STATUS:CANCELLED",
		calendar.StatusNoShow:    "STATUS:CANCELLED",
	}
	for s, want := range cases {
		e := timed()
		e.Status = s
		block := ical.ToVEVENT(e)
		if !strings.Contains(block, want) {
			t.Errorf("status %q: want line %q in block:\n%s", s, want, block)
		}
	}
}

func TestRoundTrip_EmptyOptionals(t *testing.T) {
	// No title, no notes — the mapper must omit SUMMARY/DESCRIPTION and round-trip
	// back to empty strings (not "" with a stray line).
	e := timed()
	e.Title = ""
	e.Notes = ""
	block := ical.ToVEVENT(e)
	if strings.Contains(block, "SUMMARY:") {
		t.Errorf("empty title should omit SUMMARY, got:\n%s", block)
	}
	if strings.Contains(block, "DESCRIPTION:") {
		t.Errorf("empty notes should omit DESCRIPTION, got:\n%s", block)
	}
	got := roundTrip(t, e)
	assertEventEqual(t, e, got)
}

func TestRoundTrip_SpecialChars(t *testing.T) {
	// Escaping gate: comma, semicolon, newline, backslash in title + notes must
	// survive the round-trip exactly.
	cases := []struct {
		name  string
		title string
		notes string
	}{
		{"comma", "Smith, Jones & Co", "buyers: A, B, C"},
		{"semicolon", "Handover; final", "keys; alarm; manual"},
		{"newline", "Line1\nLine2", "para1\npara2\npara3"},
		{"backslash", `path\to\unit`, `escape \ test`},
		{"all", `a,b;c\d`, "x,y;z\nw\\v"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := timed()
			e.Title = tc.title
			e.Notes = tc.notes
			got := roundTrip(t, e)
			if got.Title != tc.title {
				t.Errorf("title escaping: want %q, got %q", tc.title, got.Title)
			}
			if got.Notes != tc.notes {
				t.Errorf("notes escaping: want %q, got %q", tc.notes, got.Notes)
			}
		})
	}
}

func TestFolding_LongLine(t *testing.T) {
	// A long SUMMARY must be folded to <=75 octets per line with CRLF + a single
	// leading space on each continuation, and must unfold back to the original.
	long := strings.Repeat("A very long event title that exceeds seventy five octets. ", 5)
	e := timed()
	e.Title = long
	block := ical.ToVEVENT(e)

	for _, line := range strings.Split(block, "\r\n") {
		if len(line) > 75 {
			t.Errorf("line exceeds 75 octets (%d): %q", len(line), line)
		}
	}
	got := roundTrip(t, e)
	if got.Title != long {
		t.Errorf("folded long title did not round-trip:\nwant %q\ngot  %q", long, got.Title)
	}
}

func TestToVEVENT_StructuralLines(t *testing.T) {
	e := timed()
	block := ical.ToVEVENT(e)
	mustContain := []string{
		"BEGIN:VEVENT",
		"END:VEVENT",
		"UID:11111111-1111-1111-1111-111111111111",
		"DTSTAMP:",
		"DTSTART:20260615T103000Z",
		"DTEND:20260615T120000Z",
	}
	for _, want := range mustContain {
		if !strings.Contains(block, want) {
			t.Errorf("block missing %q:\n%s", want, block)
		}
	}
	if !strings.HasSuffix(block, "\r\n") {
		t.Error("block must end with CRLF")
	}
}

func TestCalendarWrapper(t *testing.T) {
	// ToCalendar wraps one or more VEVENTs in a VCALENDAR with VERSION + PRODID,
	// and FromCalendar extracts them back.
	e1 := timed()
	e2 := timed()
	e2.ID = "33333333-3333-3333-3333-333333333333"
	e2.Status = calendar.StatusConfirmed

	cal := ical.ToCalendar(e1, e2)
	if !strings.HasPrefix(cal, "BEGIN:VCALENDAR\r\n") {
		t.Errorf("calendar must begin with VCALENDAR:\n%s", cal)
	}
	if !strings.Contains(cal, "VERSION:2.0") {
		t.Errorf("calendar must declare VERSION:2.0:\n%s", cal)
	}
	if !strings.Contains(cal, "PRODID:") {
		t.Errorf("calendar must declare PRODID:\n%s", cal)
	}
	if !strings.HasSuffix(cal, "END:VCALENDAR\r\n") {
		t.Errorf("calendar must end with END:VCALENDAR:\n%s", cal)
	}

	events, err := ical.FromCalendar(cal)
	if err != nil {
		t.Fatalf("FromCalendar error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}
	if events[0].ID != e1.ID || events[1].ID != e2.ID {
		t.Errorf("event order/ids wrong: got %q, %q", events[0].ID, events[1].ID)
	}
	if events[1].Status != calendar.StatusConfirmed {
		t.Errorf("second event status: want confirmed, got %q", events[1].Status)
	}
}

func TestFromVEVENT_MissingUID(t *testing.T) {
	block := "BEGIN:VEVENT\r\nDTSTART:20260615T103000Z\r\nDTEND:20260615T113000Z\r\nEND:VEVENT\r\n"
	if _, err := ical.FromVEVENT(block); err == nil {
		t.Error("expected error for VEVENT without UID")
	}
}
