package ical_test

import (
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/calendar"
	"github.com/shredbx/sbx-core/pkg/calendar/ical"
)

// (source projection gate — the pluggable SourceMetaResolver projects each event's
// primary attached source onto the standard VEVENT LOCATION/GEO/URL properties at
// export, so an ICS file opened in an EXTERNAL calendar is located at, and click-
// through to, the source's PUBLIC page.)

// propertyResolver maps a property reference to its public page (URL only) and resolves
// everything else (contacts, etc.) to the zero SourceMeta — the canonical "no public
// detail" answer. It is the test stand-in for the real, switchable provider the HTTP
// adapter binds. propertyMetaResolver below adds LOCATION/GEO.
func propertyResolver(refType, refID string) ical.SourceMeta {
	if refType == "property" {
		return ical.SourceMeta{URL: "https://bestie.example/properties/" + refID}
	}
	return ical.SourceMeta{}
}

// propertyMetaResolver is propertyResolver enriched with a place (LOCATION) and
// coordinates (GEO), exercising the full SourceMeta projection. The location carries a
// comma to prove TEXT-escaping on the LOCATION line.
func propertyMetaResolver(refType, refID string) ical.SourceMeta {
	if refType == "property" {
		return ical.SourceMeta{
			URL:      "https://bestie.example/properties/" + refID,
			Location: "88/12 Bophut, Ko Pha-ngan",
			Lat:      9.7384,
			Lng:      100.0512,
		}
	}
	return ical.SourceMeta{}
}

// TestToVEVENTWith_EmitsPrimarySourceURL asserts the kitchen-sink event (which has
// a property ref AND a contact ref) emits exactly the property's public URL — the
// resolver maps the property to a page and the contact to "", and the property is
// the about/subject ref, so it is the primary. The contact's (empty) URL is never
// emitted.
func TestToVEVENTWith_EmitsPrimarySourceURL(t *testing.T) {
	e := kitchenSink() // ref-1 contact (attendee), ref-2 property p-bbbb (subject)
	block := ical.ToVEVENTWith(e, ical.MarshalOptions{ResolveSource: propertyResolver})

	want := "URL:https://bestie.example/properties/p-bbbb"
	if !strings.Contains(block, want) {
		t.Errorf("want %q in block:\n%s", want, block)
	}
	// Exactly one URL line — never the contact's empty URL, never duplicated.
	if n := strings.Count(block, "URL:"); n != 1 {
		t.Errorf("want exactly 1 URL: line, got %d in:\n%s", n, block)
	}
}

// TestToVEVENTWith_PrefersAboutSubjectRef proves the primary is the about/subject
// ref even when an earlier, non-subject ref ALSO resolves to a non-empty URL.
// Here both refs are properties (both resolve), but only the SECOND is the subject —
// the URL must point at the subject, not merely the first that resolves.
func TestToVEVENTWith_PrefersAboutSubjectRef(t *testing.T) {
	e := kitchenSink()
	e.References = []calendar.Reference{
		{ID: "r1", RefType: "property", RefID: "p-first", Relation: "attachment", Label: "Garage"},
		{ID: "r2", RefType: "property", RefID: "p-subject", Relation: "subject", Label: "Unit A2"},
	}
	block := ical.ToVEVENTWith(e, ical.MarshalOptions{ResolveSource: propertyResolver})

	want := "URL:https://bestie.example/properties/p-subject"
	if !strings.Contains(block, want) {
		t.Errorf("subject ref must win: want %q in:\n%s", want, block)
	}
	// The non-subject ref must not be the chosen URL — assert on the URL line
	// specifically (p-first still legitimately appears in its X-SBX-REF line, which
	// must round-trip; the URL projection never removes a reference).
	if strings.Contains(urlLine(block), "p-first") {
		t.Errorf("non-subject ref must not be the URL target, got URL line %q in:\n%s", urlLine(block), block)
	}
}

// urlLine returns the single URL: content line from a VEVENT block (unfolded),
// or "" if none. It lets a test assert on the URL value alone rather than the whole
// block, where a refId also appears inside its X-SBX-REF JSON.
func urlLine(block string) string {
	for _, line := range strings.Split(block, "\r\n") {
		if strings.HasPrefix(line, "URL:") {
			return line
		}
	}
	return ""
}

// TestToVEVENTWith_AboutRelationAlsoPreferred proves "about" (the alternate spelling
// of the subject relation) is preferred too, not just "subject".
func TestToVEVENTWith_AboutRelationAlsoPreferred(t *testing.T) {
	e := kitchenSink()
	e.References = []calendar.Reference{
		{ID: "r1", RefType: "property", RefID: "p-other", Relation: "attachment", Label: "Other"},
		{ID: "r2", RefType: "property", RefID: "p-about", Relation: "about", Label: "The thing"},
	}
	block := ical.ToVEVENTWith(e, ical.MarshalOptions{ResolveSource: propertyResolver})
	if !strings.Contains(block, "URL:https://bestie.example/properties/p-about") {
		t.Errorf("about-relation ref must be preferred, got:\n%s", block)
	}
}

// TestToVEVENTWith_FallsBackToFirstResolving proves that when NO ref is about/subject,
// the primary is the first ref that resolves to a non-empty URL (order preserved).
func TestToVEVENTWith_FallsBackToFirstResolving(t *testing.T) {
	e := kitchenSink()
	e.References = []calendar.Reference{
		{ID: "r1", RefType: "contact", RefID: "c-1", Relation: "attendee", Label: "Lek"},       // resolves ""
		{ID: "r2", RefType: "property", RefID: "p-1", Relation: "attachment", Label: "Unit"},   // first non-empty
		{ID: "r3", RefType: "property", RefID: "p-2", Relation: "attachment", Label: "Garage"}, // also non-empty
	}
	block := ical.ToVEVENTWith(e, ical.MarshalOptions{ResolveSource: propertyResolver})
	if !strings.Contains(block, "URL:https://bestie.example/properties/p-1") {
		t.Errorf("no subject ref: want first-resolving ref p-1, got:\n%s", block)
	}
	if n := strings.Count(block, "URL:"); n != 1 {
		t.Errorf("want exactly 1 URL: line, got %d in:\n%s", n, block)
	}
}

// TestToVEVENTWith_ContactsOnlyEmitsNoURL asserts an event whose only refs are
// contacts (all resolve to "") emits NO URL line at all.
func TestToVEVENTWith_ContactsOnlyEmitsNoURL(t *testing.T) {
	e := kitchenSink()
	e.References = []calendar.Reference{
		{ID: "r1", RefType: "contact", RefID: "c-1", Relation: "attendee", Label: "Lek"},
		{ID: "r2", RefType: "contact", RefID: "c-2", Relation: "organizer", Label: "Som"},
	}
	block := ical.ToVEVENTWith(e, ical.MarshalOptions{ResolveSource: propertyResolver})
	if strings.Contains(block, "URL:") {
		t.Errorf("contacts-only event must emit no URL line, got:\n%s", block)
	}
}

// TestToVEVENTWith_NoRefsEmitsNoURL asserts an event with NO references emits no URL.
func TestToVEVENTWith_NoRefsEmitsNoURL(t *testing.T) {
	e := kitchenSink()
	e.References = nil
	block := ical.ToVEVENTWith(e, ical.MarshalOptions{ResolveSource: propertyResolver})
	if strings.Contains(block, "URL:") {
		t.Errorf("ref-less event must emit no URL line, got:\n%s", block)
	}
}

// TestToVEVENTWith_NilResolverMatchesPlain is the non-breaking contract: a zero
// MarshalOptions (nil resolver) must produce output BYTE-FOR-BYTE identical to the
// plain ToVEVENT — no URL line, current behaviour exactly. DTSTAMP is the only
// volatile line, so it is stripped from both sides before comparison.
func TestToVEVENTWith_NilResolverMatchesPlain(t *testing.T) {
	e := kitchenSink()
	plain := stripDTSTAMP(ical.ToVEVENT(e))
	withZero := stripDTSTAMP(ical.ToVEVENTWith(e, ical.MarshalOptions{}))
	if plain != withZero {
		t.Errorf("zero MarshalOptions must equal plain ToVEVENT:\n--- plain ---\n%s\n--- with ---\n%s", plain, withZero)
	}
	if strings.Contains(withZero, "URL:") {
		t.Errorf("nil resolver must not emit URL line, got:\n%s", withZero)
	}
}

// TestToCalendarWith_NilResolverMatchesPlain mirrors the byte-identity contract for
// the VCALENDAR wrapper entry point.
func TestToCalendarWith_NilResolverMatchesPlain(t *testing.T) {
	e := kitchenSink()
	plain := stripDTSTAMP(ical.ToCalendar(e))
	withZero := stripDTSTAMP(ical.ToCalendarWith(ical.MarshalOptions{}, e))
	if plain != withZero {
		t.Errorf("zero MarshalOptions must equal plain ToCalendar:\n--- plain ---\n%s\n--- with ---\n%s", plain, withZero)
	}
}

// TestToCalendarWith_EmitsURLPerEvent proves the resolver is applied to every event
// in the VCALENDAR wrapper, not just the first.
func TestToCalendarWith_EmitsURLPerEvent(t *testing.T) {
	e1 := kitchenSink()
	e2 := kitchenSink()
	e2.ID = "33333333-3333-3333-3333-333333333333"
	e2.References = []calendar.Reference{
		{ID: "r9", RefType: "property", RefID: "p-cccc", Relation: "subject", Label: "Unit C"},
	}
	cal := ical.ToCalendarWith(ical.MarshalOptions{ResolveSource: propertyResolver}, e1, e2)
	for _, want := range []string{
		"URL:https://bestie.example/properties/p-bbbb",
		"URL:https://bestie.example/properties/p-cccc",
	} {
		if !strings.Contains(cal, want) {
			t.Errorf("calendar missing %q:\n%s", want, cal)
		}
	}
}

// TestFromVEVENT_IgnoresURL proves URL: is a derived projection: it is IGNORED on
// parse (no Event field receives it) and the round-trip still reproduces every
// carried field. A URL line authored elsewhere must not break import.
func TestFromVEVENT_IgnoresURL(t *testing.T) {
	e := kitchenSink()
	block := ical.ToVEVENTWith(e, ical.MarshalOptions{ResolveSource: propertyResolver})
	if !strings.Contains(block, "URL:") {
		t.Fatalf("precondition: block should contain a URL line:\n%s", block)
	}
	got, err := ical.FromVEVENT(block)
	if err != nil {
		t.Fatalf("FromVEVENT error: %v\n--- block ---\n%s", err, block)
	}
	// The whole field set still round-trips — the URL is regenerated next export,
	// never stored on a Reference, so reconstruction is unchanged.
	assertEventDeepEqual(t, e, got)
}

// TestRoundTrip_LosslessGateUnchanged is the explicit regression guard the user
// required: the EXISTING no-resolver round-trip gate still passes byte-stable. It
// re-runs the kitchen-sink gate through the plain (resolver-free) entry points and
// confirms no URL line leaks in.
func TestRoundTrip_LosslessGateUnchanged(t *testing.T) {
	e := kitchenSink()
	out := ical.ToVEVENT(e)
	if strings.Contains(out, "URL:") {
		t.Errorf("plain ToVEVENT must not emit a URL line, got:\n%s", out)
	}
	got, err := ical.FromVEVENT(out)
	if err != nil {
		t.Fatalf("FromVEVENT error: %v", err)
	}
	assertEventDeepEqual(t, e, got)
}

// TestToVEVENTWith_EmitsLocationAndGeo asserts the full source projection: with a
// resolver that returns place + coordinates, the primary (subject) property emits
// LOCATION (TEXT-escaped) and GEO (lat;lng) alongside URL.
func TestToVEVENTWith_EmitsLocationAndGeo(t *testing.T) {
	e := kitchenSink() // ref-2 property p-bbbb is the subject (primary)
	block := ical.ToVEVENTWith(e, ical.MarshalOptions{ResolveSource: propertyMetaResolver})

	for _, want := range []string{
		`LOCATION:88/12 Bophut\, Ko Pha-ngan`, // comma escaped per RFC 5545 TEXT
		"GEO:9.7384;100.0512",
		"URL:https://bestie.example/properties/p-bbbb",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("source projection missing %q in:\n%s", want, block)
		}
	}
}

// TestToVEVENTWith_GeoOmittedWhenZero proves the both-zero coordinate gate: a resolver
// that supplies a URL + LOCATION but leaves Lat/Lng at 0 emits NO GEO line (0,0 is the
// property-domain "unset" sentinel).
func TestToVEVENTWith_GeoOmittedWhenZero(t *testing.T) {
	resolver := func(refType, refID string) ical.SourceMeta {
		if refType == "property" {
			return ical.SourceMeta{URL: "https://bestie.example/properties/" + refID, Location: "Somewhere"}
		}
		return ical.SourceMeta{}
	}
	block := ical.ToVEVENTWith(kitchenSink(), ical.MarshalOptions{ResolveSource: resolver})
	if strings.Contains(block, "GEO:") {
		t.Errorf("zero coordinates must emit no GEO line, got:\n%s", block)
	}
	if !strings.Contains(block, "LOCATION:Somewhere") {
		t.Errorf("LOCATION must still emit when set, got:\n%s", block)
	}
}

// TestToVEVENT_EmitsCategoriesFromEventType asserts CATEGORIES is intrinsic — emitted
// by the PLAIN marshaller (no resolver) straight from the event type, for foreign-app
// grouping. kitchenSink is a viewing.
func TestToVEVENT_EmitsCategoriesFromEventType(t *testing.T) {
	block := ical.ToVEVENT(kitchenSink())
	if !strings.Contains(block, "CATEGORIES:viewing") {
		t.Errorf("plain ToVEVENT must emit CATEGORIES:viewing, got:\n%s", block)
	}
}

// TestFromVEVENT_IgnoresLocationGeoCategories proves the new derived/interop lines are
// IGNORED on parse: a block carrying LOCATION/GEO (resolver) + CATEGORIES (intrinsic)
// still round-trips every CARRIED field exactly — none leaks into an Event field.
func TestFromVEVENT_IgnoresLocationGeoCategories(t *testing.T) {
	e := kitchenSink()
	block := ical.ToVEVENTWith(e, ical.MarshalOptions{ResolveSource: propertyMetaResolver})
	for _, must := range []string{"LOCATION:", "GEO:", "CATEGORIES:"} {
		if !strings.Contains(block, must) {
			t.Fatalf("precondition: block should contain %q:\n%s", must, block)
		}
	}
	got, err := ical.FromVEVENT(block)
	if err != nil {
		t.Fatalf("FromVEVENT error: %v\n--- block ---\n%s", err, block)
	}
	assertEventDeepEqual(t, e, got)
}

// TestToVEVENTWith_GeoFormatsNegativeCoordinates guards formatGeo against a sign
// regression: southern-hemisphere / western coordinates must emit verbatim, lat;lng.
func TestToVEVENTWith_GeoFormatsNegativeCoordinates(t *testing.T) {
	resolver := func(refType, refID string) ical.SourceMeta {
		if refType == "property" {
			return ical.SourceMeta{URL: "https://bestie.example/properties/" + refID, Lat: -33.8688, Lng: 151.2093}
		}
		return ical.SourceMeta{}
	}
	block := ical.ToVEVENTWith(kitchenSink(), ical.MarshalOptions{ResolveSource: resolver})
	if !strings.Contains(block, "GEO:-33.8688;151.2093") {
		t.Errorf("negative latitude must format verbatim (lat;lng), got:\n%s", block)
	}
}

// TestToVEVENTWith_LongLocationFoldsAndParses proves a real-length address (>75 octets,
// with commas → escaped) folds to ≤75-octet lines and parses back cleanly: the parser
// must unfold the continuation and IGNORE the (derived) LOCATION without choking, and
// every carried field must still round-trip.
func TestToVEVENTWith_LongLocationFoldsAndParses(t *testing.T) {
	longAddr := "88/12 Moo 5, Bophut Soi 3, Tambon Bophut, Amphoe Ko Samui, Surat Thani 84320, Thailand"
	resolver := func(refType, refID string) ical.SourceMeta {
		if refType == "property" {
			return ical.SourceMeta{URL: "https://bestie.example/properties/" + refID, Location: longAddr}
		}
		return ical.SourceMeta{}
	}
	e := kitchenSink()
	block := ical.ToVEVENTWith(e, ical.MarshalOptions{ResolveSource: resolver})

	for _, line := range strings.Split(block, "\r\n") {
		if len(line) > 75 {
			t.Errorf("folded line exceeds 75 octets (%d): %q", len(line), line)
		}
	}
	got, err := ical.FromVEVENT(block)
	if err != nil {
		t.Fatalf("FromVEVENT error on folded LOCATION: %v\n--- block ---\n%s", err, block)
	}
	assertEventDeepEqual(t, e, got)
}

// stripDTSTAMP removes the single volatile DTSTAMP line (the marshalling instant)
// so two marshals of the same event are byte-comparable.
func stripDTSTAMP(block string) string {
	var b strings.Builder
	for _, line := range strings.Split(block, "\r\n") {
		if strings.HasPrefix(line, "DTSTAMP:") {
			continue
		}
		b.WriteString(line)
		b.WriteString("\r\n")
	}
	return b.String()
}
