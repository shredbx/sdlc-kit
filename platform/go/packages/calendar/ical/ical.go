// Package ical maps a calendar.Event to and from an iCalendar VEVENT (RFC 5545)
// and a VCALENDAR wrapper. It is HAND-ROLLED from the stdlib (strings/bufio/time/
// fmt) — no external iCalendar dependency — and is PURE: no I/O, no clock except
// DTSTAMP (the marshalling instant), no storage.
//
// Round-trip is the contract: FromVEVENT(ToVEVENT(e)) reproduces every field the
// mapper carries. The full field set is:
//
//   - UID (ID), time window (DTSTART/DTEND), all-day flag, SUMMARY (Title),
//     DESCRIPTION (Notes), status, event type. The event type ALSO emits a standard
//     CATEGORIES line (RFC 5545 §3.8.1.2) for foreign-app grouping/filtering — interop
//     only, IGNORED on parse (X-SBX-EVENT-TYPE is the authoritative carrier).
//   - Visibility → the standard CLASS property (team→PUBLIC, private→PRIVATE), a
//     clean 2-value lossless mapping parsed back on import.
//   - OwnerID → X-SBX-OWNER-ID (a bare UUID; ORGANIZER expects a cal-address /
//     mailto, which the domain does not carry, so a custom property is used).
//   - References []Reference → one X-SBX-REF line per reference, the full chip
//     (id, refType, refId, relation, asType, label, subtitle) carried as a compact
//     JSON object in the (TEXT-escaped) value. JSON is the robust lossless carrier
//     for the two free-text fields (label, subtitle), which may hold commas,
//     semicolons, quotes, and backslashes that param-encoding would have to escape
//     by hand. For interop a standard ATTENDEE line is ALSO emitted for any
//     reference whose relation is "attendee" — but X-SBX-REF is authoritative and
//     ATTENDEE is ignored on parse.
//   - Reminders []Reminder → one standard VALARM per reminder (ACTION:DISPLAY,
//     TRIGGER:-PT{LeadMinutes}M, X-SBX-CHANNEL:<channel>), parsed back from VALARM.
//
// An OPTIONAL source projection — the standard LOCATION (§3.8.1.7), GEO (§3.8.1.6),
// and URL (§3.8.4.6) properties — is DERIVED export-only, emitted only via the *With
// marshallers when a MarshalOptions carries a SourceMetaResolver. It describes the
// event's PRIMARY attached source (see primarySource for the selection rule): its
// place (LOCATION), coordinates (GEO), and PUBLIC page (URL) — so an exported ICS
// opened in an external calendar is located at, and click-through to, that source.
// The projection is DERIVED AT EXPORT from a pluggable, switchable provider — never
// stored on a Reference — so it does NOT participate in the round-trip: LOCATION/GEO/
// URL are IGNORED on parse and regenerated next export. The nil-resolver path (plain
// ToVEVENT/ToCalendar) emits none of these lines and is byte-for-byte identical to
// before this projection existed.
//
// Three Document-base timestamps are intentionally NOT carried, never silently:
// CreatedAt, UpdatedAt, DeletedAt have no faithful iCalendar slot (DTSTAMP is the
// object-creation instant, not CreatedAt) and are reconciled by the sync layer out
// of band. DTSTAMP itself is the marshalling instant and is excluded from the
// round-trip equality contract by design.
//
// Losslessness of the five soft booking states (calendar.EventStatus) is the
// subtle part. RFC 5545 VEVENT STATUS has only three values
// (TENTATIVE/CONFIRMED/CANCELLED), so the five internal states cannot survive a
// pure STATUS round-trip. The mapper therefore writes BOTH:
//
//   - a standard STATUS line (best-effort, so foreign calendar apps still get a
//     sensible status), and
//   - a custom X-SBX-STATUS line carrying the exact internal code.
//
// On parse it reads X-SBX-STATUS first (lossless) and falls back to STATUS only
// when the X- property is absent (an event authored elsewhere). RFC 5545 §3.8.8.2
// explicitly permits experimental X- properties, so the output stays valid.
//
// Text values are escaped per RFC 5545 §3.3.11 (backslash, comma, semicolon,
// newline) and content lines are folded to <=75 octets with CRLF + a single
// leading space on each continuation (§3.1). The parser unfolds and unescapes
// symmetrically.
package ical

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shredbx/sbx-core/pkg/calendar"
)

// prodID identifies the product that generated the calendar, per RFC 5545 §3.7.3.
const prodID = "-//SBX//Calendar 1.0//EN"

// crlf is the mandatory line break between content lines (RFC 5545 §3.1).
const crlf = "\r\n"

// maxOctets is the soft line-length limit for folding (RFC 5545 §3.1): lines
// SHOULD be folded so no line is longer than 75 octets, excluding the CRLF.
const maxOctets = 75

// dateLayout is the DATE value layout (RFC 5545 §3.3.4) — date-only, for all-day.
const dateLayout = "20060102"

// dateTimeUTCLayout is the UTC DATE-TIME value layout (RFC 5545 §3.3.5, form 2):
// a "Z" suffix marks the value as UTC. Timed events are normalized to UTC so the
// instant is unambiguous on round-trip.
const dateTimeUTCLayout = "20060102T150405Z"

// =============================================================================
// MARSHAL — Event -> VEVENT / VCALENDAR
// =============================================================================

// SourceMeta is the export-only projection of an attached Source's public-facing
// detail: its place (Location), coordinates (Lat/Lng), and PUBLIC page (URL). It is
// DERIVED AT EXPORT from a pluggable, switchable provider — never stored on a
// Reference — so the lossless round-trip is untouched (FI-4 / CAL-R1). A zero
// SourceMeta (empty URL/Location, zero Lat/Lng) contributes no lines.
type SourceMeta struct {
	// URL is the source's PUBLIC page (RFC 5545 §3.8.4.6). "" → no URL line.
	URL string
	// Location is the human-readable place of the event (RFC 5545 §3.8.1.7), e.g. an
	// attached property's address. "" → no LOCATION line.
	Location string
	// Lat, Lng are decimal-degree coordinates (RFC 5545 §3.8.1.6). Both zero → no GEO
	// line (matches the property-domain "unset coords are stored as 0,0" convention).
	// GEO is emitted latitude-FIRST ("lat;lng") — resolvers must not swap the axes.
	Lat, Lng float64
}

// hasGeo reports whether the coordinate pair is set. The property domain stores an
// unset pair as (0,0) (a DB CHECK enforces both-zero-or-both-nonzero), so both-zero
// means "no coordinates" and no GEO line is emitted.
func (m SourceMeta) hasGeo() bool { return m.Lat != 0 || m.Lng != 0 }

// isEmpty reports whether the meta contributes no lines — the primary-source
// selection uses it to decide whether a reference "resolves".
func (m SourceMeta) isEmpty() bool { return m.URL == "" && m.Location == "" && !m.hasGeo() }

// SourceMetaResolver derives the export-only SourceMeta of an attached Source from its
// typed reference (RefType + RefID). It returns the zero SourceMeta when the source
// has no public-facing detail (e.g. a contact). It is the seam that keeps the mapper
// source-agnostic: LOCATION/GEO/URL are DERIVED AT EXPORT from a pluggable, switchable
// provider — never stored on a Reference — so the lossless round-trip is untouched
// (FI-4 / CAL-R1). The HTTP adapter binds a concrete resolver (e.g. property →
// {address, lat, lng, /properties/{id}}); the mapper neither knows nor enforces any
// particular source kind.
type SourceMetaResolver func(refType, refID string) SourceMeta

// MarshalOptions carries optional, export-only behaviour for the *With marshallers.
// A zero MarshalOptions (the nil resolver) reproduces the plain ToVEVENT/ToCalendar
// output byte-for-byte — the source projection is strictly additive.
type MarshalOptions struct {
	// ResolveSource, when non-nil, projects each event's PRIMARY attached source onto
	// the standard VEVENT LOCATION/GEO/URL properties (RFC 5545 §3.8.1.7 / §3.8.1.6 /
	// §3.8.4.6) so an exported ICS opened in an external calendar is located at, and
	// click-through to, the source. Nil → none of those lines (current behaviour).
	ResolveSource SourceMetaResolver
}

// ToCalendar wraps one or more events in a VCALENDAR block (VERSION + PRODID).
// The output ends with a trailing CRLF.
func ToCalendar(events ...calendar.Event) string {
	return ToCalendarWith(MarshalOptions{}, events...)
}

// ToCalendarWith is ToCalendar with export options. With opts.ResolveSource set, each
// VEVENT carries LOCATION/GEO/URL lines for its primary attached source; a zero
// MarshalOptions yields output identical to ToCalendar.
func ToCalendarWith(opts MarshalOptions, events ...calendar.Event) string {
	var b strings.Builder
	writeLine(&b, "BEGIN:VCALENDAR")
	writeLine(&b, "VERSION:2.0")
	writeLine(&b, "PRODID:"+prodID)
	for _, e := range events {
		b.WriteString(ToVEVENTWith(e, opts))
	}
	writeLine(&b, "END:VCALENDAR")
	return b.String()
}

// ToVEVENT marshals a single event to a VEVENT block (no VCALENDAR wrapper). The
// output ends with a trailing CRLF. DTSTAMP is the current UTC instant (the time
// the object was created, RFC 5545 §3.8.7.2) and is intentionally NOT part of the
// round-trip equality contract.
func ToVEVENT(e calendar.Event) string {
	return ToVEVENTWith(e, MarshalOptions{})
}

// ToVEVENTWith is ToVEVENT with export options. When opts.ResolveSource is non-nil and
// the event's primary attached source resolves to a non-empty SourceMeta, the standard
// LOCATION/GEO/URL properties are emitted (see primarySource for the selection rule). A
// zero MarshalOptions (nil resolver) produces output byte-for-byte identical to ToVEVENT.
func ToVEVENTWith(e calendar.Event, opts MarshalOptions) string {
	var b strings.Builder
	writeLine(&b, "BEGIN:VEVENT")

	writeProp(&b, "UID", e.ID)
	writeProp(&b, "DTSTAMP", time.Now().UTC().Format(dateTimeUTCLayout))

	if e.AllDay {
		// All-day: date-only VALUE=DATE. DTEND is already the exclusive day-after
		// in the domain model (calendar.Event semantics), so it is emitted verbatim.
		writeRawProp(&b, "DTSTART;VALUE=DATE", e.Start.UTC().Format(dateLayout))
		writeRawProp(&b, "DTEND;VALUE=DATE", e.End.UTC().Format(dateLayout))
	} else {
		writeRawProp(&b, "DTSTART", e.Start.UTC().Format(dateTimeUTCLayout))
		writeRawProp(&b, "DTEND", e.End.UTC().Format(dateTimeUTCLayout))
	}

	if e.Title != "" {
		writeProp(&b, "SUMMARY", e.Title)
	}
	if e.Notes != "" {
		writeProp(&b, "DESCRIPTION", e.Notes)
	}

	// Source projection → the PRIMARY attached source's place/coordinates/page
	// (LOCATION/GEO/URL), DERIVED at export from the pluggable resolver, never stored
	// on a Reference. Omitted entirely when no resolver is supplied or no reference
	// resolves — so the nil-resolver path stays byte-stable. All three are IGNORED on
	// parse.
	if opts.ResolveSource != nil {
		if m := primarySource(e.References, opts.ResolveSource); !m.isEmpty() {
			if m.Location != "" {
				writeProp(&b, "LOCATION", m.Location)
			}
			if m.hasGeo() {
				writeRawProp(&b, "GEO", formatGeo(m.Lat, m.Lng))
			}
			if m.URL != "" {
				writeRawProp(&b, "URL", m.URL)
			}
		}
	}

	// Best-effort standard STATUS for foreign apps, then the lossless X- property.
	if std := standardStatus(e.Status); std != "" {
		writeRawProp(&b, "STATUS", std)
	}
	if e.Status != "" {
		writeRawProp(&b, "X-SBX-STATUS", string(e.Status))
	}
	if e.EventType != "" {
		writeRawProp(&b, "X-SBX-EVENT-TYPE", string(e.EventType))
		// CATEGORIES → standard interop projection of the event type (RFC 5545
		// §3.8.1.2) for foreign-app grouping/filtering. Intrinsic (not resolver-gated):
		// emitted by both ToVEVENT and ToVEVENTWith. IGNORED on parse — X-SBX-EVENT-TYPE
		// above is the authoritative carrier. Written via writeProp (TEXT-escaped) since
		// CATEGORIES is a standard TEXT property: a single event type is one category, so
		// any comma in the code is ESCAPED (kept as one value), never a list separator.
		writeProp(&b, "CATEGORIES", string(e.EventType))
	}

	// Visibility → standard CLASS (lossless 2-value mapping). Omitted when empty so
	// a never-set visibility round-trips back to "" rather than a defaulted value.
	if cls := standardClass(e.Visibility); cls != "" {
		writeRawProp(&b, "CLASS", cls)
	}

	// OwnerID → X-SBX-OWNER-ID (bare UUID; no cal-address to satisfy ORGANIZER).
	if e.OwnerID != "" {
		writeRawProp(&b, "X-SBX-OWNER-ID", e.OwnerID)
	}

	// References → one authoritative X-SBX-REF (compact JSON) per reference, plus an
	// interop ATTENDEE for attendee-relation refs (ATTENDEE is ignored on parse).
	for _, ref := range e.References {
		writeRawProp(&b, "X-SBX-REF", escapeText(marshalReference(ref)))
		if strings.EqualFold(ref.Relation, "attendee") {
			head := "ATTENDEE"
			if cn := paramValue(ref.Label); cn != "" {
				head += ";CN=" + cn
			}
			writeRawProp(&b, head, "urn:sbx:"+ref.RefType+":"+ref.RefID)
		}
	}

	// Reminders → standard VALARM blocks (relative TRIGGER + custom channel).
	for _, rem := range e.Reminders {
		writeReminder(&b, rem)
	}

	writeLine(&b, "END:VEVENT")
	return b.String()
}

// primarySource picks the export SourceMeta of the event's PRIMARY attached source
// via the resolver, source-agnostically. Selection rule, in two passes over
// References in declared order:
//
//  1. PREFER the first reference whose Relation is "about" or "subject" (the thing
//     the event is ABOUT — e.g. the property being viewed) that RESOLVES (a non-empty
//     SourceMeta).
//  2. Otherwise fall back to the first reference (any relation) that resolves.
//
// Returns the zero SourceMeta when no reference resolves (e.g. contacts only), so no
// LOCATION/GEO/URL line is emitted. The resolver — not this function — decides which
// source kinds carry detail; an attendee/contact ref simply resolves to the zero
// SourceMeta and is skipped.
func primarySource(refs []calendar.Reference, resolve SourceMetaResolver) SourceMeta {
	// Pass 1: an about/subject reference that resolves.
	for _, ref := range refs {
		if isAboutRelation(ref.Relation) {
			if m := resolve(ref.RefType, ref.RefID); !m.isEmpty() {
				return m
			}
		}
	}
	// Pass 2: the first reference (any relation) that resolves.
	for _, ref := range refs {
		if m := resolve(ref.RefType, ref.RefID); !m.isEmpty() {
			return m
		}
	}
	return SourceMeta{}
}

// isAboutRelation reports whether a reference relation marks the thing the event is
// ABOUT — the subject/topic of the occurrence (the property being viewed), as
// opposed to a participant (attendee/organizer) or a secondary attachment. The two
// accepted spellings are "about" and "subject" (case-insensitive).
func isAboutRelation(relation string) bool {
	switch strings.ToLower(strings.TrimSpace(relation)) {
	case "about", "subject":
		return true
	}
	return false
}

// standardClass maps Visibility to the RFC 5545 CLASS value (§3.8.1.3): team is a
// team-visible event → PUBLIC; private is owner-only → PRIVATE. An empty/unknown
// visibility yields "" (no CLASS line). The inverse classFromStandard restores the
// exact value, making this a lossless 2-value round-trip.
func standardClass(v calendar.Visibility) string {
	switch v {
	case calendar.VisibilityTeam:
		return "PUBLIC"
	case calendar.VisibilityPrivate:
		return "PRIVATE"
	}
	return ""
}

// classFromStandard maps a CLASS value back to a Visibility. PUBLIC → team,
// PRIVATE → private. CONFIDENTIAL (a third RFC value foreign apps may emit) is
// treated as private (the stricter of the two we model). An unknown value yields "".
func classFromStandard(v string) calendar.Visibility {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "PUBLIC":
		return calendar.VisibilityTeam
	case "PRIVATE", "CONFIDENTIAL":
		return calendar.VisibilityPrivate
	}
	return ""
}

// referenceWire is the JSON shape carried in an X-SBX-REF value. It mirrors
// calendar.Reference field-for-field with compact, stable JSON keys so the whole
// chip — including the free-text label/subtitle — round-trips losslessly. Encoding
// the reference as JSON (rather than param-packing) sidesteps hand-escaping of
// commas/semicolons/quotes/backslashes in the text fields.
type referenceWire struct {
	ID       string `json:"id"`
	RefType  string `json:"refType"`
	RefID    string `json:"refId"`
	Relation string `json:"relation,omitempty"`
	AsType   string `json:"asType,omitempty"`
	Label    string `json:"label"`
	Subtitle string `json:"subtitle,omitempty"`
}

// marshalReference serializes a Reference to a compact JSON string for an
// X-SBX-REF value. json.Marshal never errors for this fixed string-only struct.
func marshalReference(r calendar.Reference) string {
	w := referenceWire{
		ID: r.ID, RefType: r.RefType, RefID: r.RefID,
		Relation: r.Relation, AsType: r.AsType, Label: r.Label, Subtitle: r.Subtitle,
	}
	b, _ := json.Marshal(w)
	return string(b)
}

// unmarshalReference reverses marshalReference. A malformed value yields an error
// so a corrupt X-SBX-REF surfaces rather than silently dropping a reference.
func unmarshalReference(s string) (calendar.Reference, error) {
	var w referenceWire
	if err := json.Unmarshal([]byte(s), &w); err != nil {
		return calendar.Reference{}, fmt.Errorf("X-SBX-REF: %w", err)
	}
	return calendar.Reference{
		ID: w.ID, RefType: w.RefType, RefID: w.RefID,
		Relation: w.Relation, AsType: w.AsType, Label: w.Label, Subtitle: w.Subtitle,
	}, nil
}

// writeReminder emits one standard VALARM for a reminder. The TRIGGER is a relative
// duration before the event start (RFC 5545 §3.8.6.3, negative = before); the lead
// is whole minutes, so the value is -PT{LeadMinutes}M. ACTION is DISPLAY (a visual
// nudge); the SBX delivery channel — which RFC 5545 does not model — rides along in
// a custom X-SBX-CHANNEL property so the channel round-trips.
func writeReminder(b *strings.Builder, r calendar.Reminder) {
	writeLine(b, "BEGIN:VALARM")
	writeRawProp(b, "ACTION", "DISPLAY")
	writeRawProp(b, "TRIGGER", "-PT"+strconv.Itoa(r.LeadMinutes)+"M")
	if r.ID != "" {
		writeRawProp(b, "X-SBX-REMINDER-ID", r.ID)
	}
	if r.Channel != "" {
		writeRawProp(b, "X-SBX-CHANNEL", string(r.Channel))
	}
	writeLine(b, "END:VALARM")
}

// reminderFromLines builds a Reminder from the body lines of one VALARM block
// (between BEGIN:VALARM and END:VALARM). LeadMinutes is recovered from the relative
// TRIGGER; Channel from X-SBX-CHANNEL; ID from X-SBX-REMINDER-ID. EventID is set by
// the caller from the enclosing VEVENT UID.
func reminderFromLines(lines []string) (calendar.Reminder, error) {
	var r calendar.Reminder
	for _, line := range lines {
		name, _, value := splitContentLine(line)
		switch name {
		case "TRIGGER":
			mins, err := parseLeadTrigger(value)
			if err != nil {
				return calendar.Reminder{}, fmt.Errorf("ical: VALARM TRIGGER: %w", err)
			}
			r.LeadMinutes = mins
		case "X-SBX-CHANNEL":
			r.Channel = calendar.ReminderChannel(value)
		case "X-SBX-REMINDER-ID":
			r.ID = value
		}
	}
	return r, nil
}

// parseLeadTrigger parses a relative VALARM TRIGGER value of the exact form this
// mapper emits — "-PT{minutes}M" — back into whole minutes-before-start. A leading
// "-" (before the event) is the normal case and yields a non-negative lead; the
// rare "PT0M" (no sign) also parses to 0. Any other duration form is rejected so a
// foreign/unsupported TRIGGER surfaces rather than silently mapping to 0.
func parseLeadTrigger(v string) (int, error) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "-")
	if !strings.HasPrefix(v, "PT") || !strings.HasSuffix(v, "M") {
		return 0, fmt.Errorf("unsupported trigger %q (want -PT<minutes>M)", v)
	}
	digits := strings.TrimSuffix(strings.TrimPrefix(v, "PT"), "M")
	mins, err := strconv.Atoi(digits)
	if err != nil {
		return 0, fmt.Errorf("trigger minutes %q: %w", digits, err)
	}
	if mins < 0 {
		return 0, fmt.Errorf("trigger minutes must be >= 0, got %d", mins)
	}
	return mins, nil
}

// standardStatus maps an internal status to the closest RFC 5545 VEVENT STATUS
// value. scheduled -> TENTATIVE (not yet confirmed); confirmed/done -> CONFIRMED
// (done is a completed confirmed booking); cancelled/no_show -> CANCELLED (the
// event did not happen). An empty/unknown status yields "" (no STATUS line).
func standardStatus(s calendar.EventStatus) string {
	switch s {
	case calendar.StatusScheduled:
		return "TENTATIVE"
	case calendar.StatusConfirmed, calendar.StatusDone:
		return "CONFIRMED"
	case calendar.StatusCancelled, calendar.StatusNoShow:
		return "CANCELLED"
	}
	return ""
}

// formatGeo formats a coordinate pair as an RFC 5545 §3.8.1.6 GEO value:
// "<latitude>;<longitude>" in decimal degrees, using the shortest round-trippable
// decimal representation (strconv 'f', -1) so 9.557 stays 9.557, not 9.5570000.
func formatGeo(lat, lng float64) string {
	return strconv.FormatFloat(lat, 'f', -1, 64) + ";" + strconv.FormatFloat(lng, 'f', -1, 64)
}

// =============================================================================
// UNMARSHAL — VEVENT / VCALENDAR -> Event
// =============================================================================

// FromCalendar extracts every VEVENT from a VCALENDAR (or bare VEVENT) block,
// preserving order. Lines outside VEVENT boundaries (VERSION, PRODID, …) are
// ignored. Returns an error if any contained VEVENT is malformed.
func FromCalendar(s string) ([]calendar.Event, error) {
	lines, err := unfold(s)
	if err != nil {
		return nil, err
	}
	var events []calendar.Event
	var block []string
	inEvent := false
	for _, line := range lines {
		switch {
		case line == "BEGIN:VEVENT":
			inEvent = true
			block = []string{line}
		case line == "END:VEVENT":
			if !inEvent {
				return nil, fmt.Errorf("ical: END:VEVENT without BEGIN")
			}
			block = append(block, line)
			e, perr := eventFromLines(block)
			if perr != nil {
				return nil, perr
			}
			events = append(events, e)
			inEvent = false
		case inEvent:
			block = append(block, line)
		}
	}
	if inEvent {
		return nil, fmt.Errorf("ical: unterminated VEVENT")
	}
	return events, nil
}

// FromVEVENT parses a single VEVENT block (with or without a VCALENDAR wrapper)
// into an Event. UID is required; DTSTART/DTEND are required.
func FromVEVENT(s string) (calendar.Event, error) {
	lines, err := unfold(s)
	if err != nil {
		return calendar.Event{}, err
	}
	// Trim to the VEVENT body if a wrapper is present.
	body := lines
	for i, line := range lines {
		if line == "BEGIN:VEVENT" {
			body = lines[i:]
			break
		}
	}
	return eventFromLines(body)
}

// eventFromLines builds an Event from the already-unfolded lines of one VEVENT
// block (BEGIN:VEVENT … END:VEVENT). It enforces the required properties. Nested
// VALARM sub-blocks are collected and parsed into Reminders; all other lines are
// VEVENT-level properties.
func eventFromLines(lines []string) (calendar.Event, error) {
	var e calendar.Event
	var haveUID, haveStart, haveEnd bool
	var xStatus string

	var alarm []string // accumulates the current VALARM body
	inAlarm := false

	for _, line := range lines {
		if line == "BEGIN:VEVENT" || line == "END:VEVENT" {
			continue
		}
		// VALARM sub-block handling — a VEVENT may contain zero or more VALARMs.
		switch {
		case line == "BEGIN:VALARM":
			inAlarm = true
			alarm = nil
			continue
		case line == "END:VALARM":
			if !inAlarm {
				return calendar.Event{}, fmt.Errorf("ical: END:VALARM without BEGIN")
			}
			rem, perr := reminderFromLines(alarm)
			if perr != nil {
				return calendar.Event{}, perr
			}
			e.Reminders = append(e.Reminders, rem)
			inAlarm = false
			continue
		case inAlarm:
			alarm = append(alarm, line)
			continue
		}

		name, params, value := splitContentLine(line)
		switch name {
		case "UID":
			e.ID = unescapeText(value)
			haveUID = true
		case "DTSTART":
			ts, allDay, perr := parseDateOrDateTime(value, params)
			if perr != nil {
				return calendar.Event{}, fmt.Errorf("ical: DTSTART: %w", perr)
			}
			e.Start = ts
			if allDay {
				e.AllDay = true
			}
			haveStart = true
		case "DTEND":
			ts, allDay, perr := parseDateOrDateTime(value, params)
			if perr != nil {
				return calendar.Event{}, fmt.Errorf("ical: DTEND: %w", perr)
			}
			e.End = ts
			if allDay {
				e.AllDay = true
			}
			haveEnd = true
		case "SUMMARY":
			e.Title = unescapeText(value)
		case "DESCRIPTION":
			e.Notes = unescapeText(value)
		case "STATUS":
			// Fallback only — overridden by X-SBX-STATUS when present.
			if e.Status == "" {
				e.Status = statusFromStandard(value)
			}
		case "X-SBX-STATUS":
			xStatus = value
		case "X-SBX-EVENT-TYPE":
			e.EventType = calendar.EventType(value)
		case "CLASS":
			// Fallback only — there is no X- override for visibility (CLASS is exact).
			if e.Visibility == "" {
				e.Visibility = classFromStandard(value)
			}
		case "X-SBX-OWNER-ID":
			e.OwnerID = value
		case "X-SBX-REF":
			ref, perr := unmarshalReference(unescapeText(value))
			if perr != nil {
				return calendar.Event{}, fmt.Errorf("ical: %w", perr)
			}
			e.References = append(e.References, ref)
		case "ATTENDEE":
			// Interop duplicate of X-SBX-REF — ignored (X-SBX-REF is authoritative).
		case "URL":
			// Derived export projection of the primary source's public page — IGNORED
			// on parse (it is never stored on a Reference; it is regenerated from the
			// resolver on the next export). Ignoring it keeps Reference reconstruction
			// and the lossless round-trip untouched.
		case "LOCATION", "GEO":
			// Derived export projection of the primary source's place/coordinates —
			// IGNORED on parse (never stored on a Reference; regenerated from the
			// resolver each export), mirroring URL above.
		case "CATEGORIES":
			// Interop projection of the event type — IGNORED on parse. X-SBX-EVENT-TYPE
			// is authoritative; a foreign free-text category would not map to the BR
			// event-type dictionary.
		}
	}

	if inAlarm {
		return calendar.Event{}, fmt.Errorf("ical: unterminated VALARM")
	}
	if xStatus != "" {
		e.Status = calendar.EventStatus(xStatus)
	}
	// Reminders carry their parent event id (EventID == event UID); reconstruct it
	// from the enclosing VEVENT rather than emitting a redundant wire line.
	for i := range e.Reminders {
		e.Reminders[i].EventID = e.ID
	}
	if !haveUID {
		return calendar.Event{}, fmt.Errorf("ical: VEVENT missing required UID")
	}
	if !haveStart {
		return calendar.Event{}, fmt.Errorf("ical: VEVENT missing required DTSTART")
	}
	if !haveEnd {
		return calendar.Event{}, fmt.Errorf("ical: VEVENT missing required DTEND")
	}
	return e, nil
}

// statusFromStandard maps a foreign STATUS value to an internal status when no
// X-SBX-STATUS is present. The mapping is the best inverse of standardStatus:
// TENTATIVE -> scheduled, CONFIRMED -> confirmed, CANCELLED -> cancelled. (done
// and no_show have no standard representation; they are only recoverable via the
// X- property.) An unknown value yields "".
func statusFromStandard(v string) calendar.EventStatus {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "TENTATIVE":
		return calendar.StatusScheduled
	case "CONFIRMED":
		return calendar.StatusConfirmed
	case "CANCELLED", "CANCELED":
		return calendar.StatusCancelled
	}
	return ""
}

// parseDateOrDateTime parses a DTSTART/DTEND value as either a date-only value
// (VALUE=DATE → all-day, parsed at UTC midnight) or a DATE-TIME. DATE-TIME forms
// supported: UTC (trailing Z) and floating local (no Z, no TZID) — both parsed in
// UTC so the round-trip of our own UTC output is exact.
func parseDateOrDateTime(value string, params map[string]string) (time.Time, bool, error) {
	if strings.EqualFold(params["VALUE"], "DATE") || (len(value) == 8 && !strings.Contains(value, "T")) {
		ts, err := time.ParseInLocation(dateLayout, value, time.UTC)
		if err != nil {
			return time.Time{}, false, err
		}
		return ts, true, nil
	}
	if strings.HasSuffix(value, "Z") {
		ts, err := time.ParseInLocation(dateTimeUTCLayout, value, time.UTC)
		if err != nil {
			return time.Time{}, false, err
		}
		return ts, false, nil
	}
	// Floating DATE-TIME (no Z) — parse in UTC.
	ts, err := time.ParseInLocation("20060102T150405", value, time.UTC)
	if err != nil {
		return time.Time{}, false, err
	}
	return ts, false, nil
}

// =============================================================================
// CONTENT-LINE WRITERS
// =============================================================================

// writeLine writes a verbatim line + CRLF (no folding) — used for fixed structural
// lines that are guaranteed short (BEGIN/END/VERSION).
func writeLine(b *strings.Builder, line string) {
	b.WriteString(line)
	b.WriteString(crlf)
}

// writeProp writes "NAME:<escaped value>", escaping the value per RFC 5545
// §3.3.11, then folds the assembled line to <=75 octets.
func writeProp(b *strings.Builder, name, value string) {
	writeFolded(b, name+":"+escapeText(value))
}

// writeRawProp writes "NAME:<value>" WITHOUT text-escaping the value — used for
// values that are not RFC 5545 TEXT (timestamps, STATUS keywords, dictionary
// codes) and never contain special characters. The line is still folded.
func writeRawProp(b *strings.Builder, name, value string) {
	writeFolded(b, name+":"+value)
}

// writeFolded folds a single assembled content line to <=75 octets per RFC 5545
// §3.1: a CRLF is inserted and the next segment begins with a single space. The
// fold is octet-based but never splits a multi-byte UTF-8 sequence.
func writeFolded(b *strings.Builder, line string) {
	if len(line) <= maxOctets {
		b.WriteString(line)
		b.WriteString(crlf)
		return
	}
	// First segment: up to 75 octets. Continuations: a leading space + up to 74
	// octets (the space counts toward the 75-octet line length).
	bytes := []byte(line)
	first := true
	for len(bytes) > 0 {
		limit := maxOctets
		prefix := ""
		if !first {
			prefix = " "
			limit = maxOctets - 1
		}
		cut := limit
		if cut > len(bytes) {
			cut = len(bytes)
		} else {
			// Back off so we never split a UTF-8 continuation byte (0b10xxxxxx).
			for cut > 0 && bytes[cut]&0xC0 == 0x80 {
				cut--
			}
		}
		b.WriteString(prefix)
		b.Write(bytes[:cut])
		b.WriteString(crlf)
		bytes = bytes[cut:]
		first = false
	}
}

// =============================================================================
// CONTENT-LINE PARSING
// =============================================================================

// unfold reads the raw block, normalizes line endings, and unfolds continuation
// lines (a line beginning with a space or tab continues the previous line, RFC
// 5545 §3.1). Returns the logical (unfolded) lines with no trailing CRLF.
func unfold(s string) ([]string, error) {
	// Normalize bare LF / CR to a common scanner-friendly form. We scan by \n and
	// strip a trailing \r so both CRLF and LF inputs parse.
	scanner := bufio.NewScanner(strings.NewReader(s))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	var logical []string
	for scanner.Scan() {
		raw := strings.TrimSuffix(scanner.Text(), "\r")
		if raw == "" {
			continue
		}
		if (raw[0] == ' ' || raw[0] == '\t') && len(logical) > 0 {
			// Continuation: drop exactly one leading folding character, append.
			logical[len(logical)-1] += raw[1:]
			continue
		}
		logical = append(logical, raw)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ical: scan: %w", err)
	}
	return logical, nil
}

// splitContentLine splits a logical line into its property name, parameters, and
// value. Format: NAME[;PARAM=VALUE[;…]]:VALUE (RFC 5545 §3.1). The split on the
// first unquoted colon separates the property part from the value; parameter
// values may be quoted but our writer never quotes, so a simple split suffices for
// round-tripping our own output while still tolerating a leading quoted param.
func splitContentLine(line string) (name string, params map[string]string, value string) {
	params = map[string]string{}
	colon := indexUnquoted(line, ':')
	if colon < 0 {
		return line, params, ""
	}
	head := line[:colon]
	value = line[colon+1:]

	parts := strings.Split(head, ";")
	name = strings.ToUpper(parts[0])
	for _, p := range parts[1:] {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			params[strings.ToUpper(kv[0])] = strings.Trim(kv[1], `"`)
		}
	}
	return name, params, value
}

// indexUnquoted returns the index of the first occurrence of sep that is NOT
// inside a double-quoted run, or -1. iCalendar parameter values may quote a colon
// (e.g. an ALTREP URI), so the name/value colon must be found outside quotes.
func indexUnquoted(s string, sep byte) int {
	inQuote := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			inQuote = !inQuote
		case sep:
			if !inQuote {
				return i
			}
		}
	}
	return -1
}

// =============================================================================
// TEXT ESCAPING — RFC 5545 §3.3.11
// =============================================================================

// paramValue formats a value for a property PARAMETER (RFC 5545 §3.2). It is used
// only for the cosmetic, interop-only ATTENDEE CN — which is never read back
// (X-SBX-REF is authoritative) — so it sanitizes rather than round-trips: control
// characters (CR/LF) and embedded double-quotes are stripped, and the value is
// double-quoted when it contains a separator (':', ';', ',') so the line stays
// well-formed for our own unfold/split. An empty result drops the CN entirely.
func paramValue(s string) string {
	s = strings.NewReplacer("\r", "", "\n", " ", `"`, "").Replace(s)
	if strings.ContainsAny(s, ":;,") {
		return `"` + s + `"`
	}
	return s
}

// escapeText escapes a Go string for an iCalendar TEXT value: backslash, newline,
// comma, and semicolon. Order matters — backslash is escaped first so the escape
// characters introduced for the others are not double-escaped. A literal CR is
// dropped (CRLF newlines collapse to the single \n escape).
func escapeText(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		";", `\;`,
		",", `\,`,
		"\r\n", `\n`,
		"\n", `\n`,
		"\r", `\n`,
	)
	return r.Replace(s)
}

// unescapeText reverses escapeText. It walks the string so an escaped backslash
// (\\) is consumed as a single backslash and never re-interpreted. Both \n and \N
// decode to a newline (RFC 5545 allows either case).
func unescapeText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n', 'N':
			b.WriteByte('\n')
		case '\\', ';', ',':
			b.WriteByte(s[i])
		default:
			// Unknown escape — keep the escaped character verbatim (lenient).
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
