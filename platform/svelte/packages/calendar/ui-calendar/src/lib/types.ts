// @sbx/ui-calendar — the contract (Level 1 § ① CalendarAdapter).
// Mirrors @sbx/ui-map MapAdapter: a clean seam between presentation and data.
// The calendar components render CalendarItem[] and emit callbacks; they import
// NO data source. createMockAdapter() satisfies this contract in-memory now; a
// createBookingHttpAdapter(baseUrl) will satisfy the SAME contract later — the
// L0 scenarios pass unchanged when the backend lands.
//
// No `any` anywhere (use `unknown`). These are the only public types views/pages
// depend on.

// The five soft booking states (D9 / Finding F1) — scheduled → confirmed →
// done | cancelled | no_show. A `cancelled` event is greyed, never deleted.
export type EventStatus = 'scheduled' | 'confirmed' | 'done' | 'cancelled' | 'no_show';

// The view modes the Calendar wrapper toggles between. month/week/day are grid
// views; 'list' is a scannable chronological agenda (EC List plugin) — the natural
// narrow/mobile layout. The view system fills its consumer's frame; the consumer
// controls the frame size.
export type CalendarView = 'month' | 'week' | 'day' | 'list';

// Viewport-space rect (CSS px, from getBoundingClientRect) of a clicked event — the
// optional 2nd arg of onItemClick. A host anchors a preview popover (EventPreviewPopover)
// to it; a host that doesn't preview simply ignores it (back-compatible).
export interface PreviewAnchor {
	x: number;
	y: number;
	width: number;
	height: number;
}

// A cached, render-ready link from an event to an independent-lifecycle entity
// (people + things). The cached `label` survives deletes of the target entity.
export interface RefChip {
	referenceId: string;
	refType: string;
	refId: string;
	relation?: string; // canonical dict code (e.g. "attendee", "about")
	asType?: string; // canonical dict code (e.g. "Landlord", "Tenant")
	label: string;
	subtitle?: string;
	icon?: string;
}

// Domain-agnostic — the only thing the calendar views render.
export interface CalendarItem {
	id: string;
	start: string; // ISO-8601
	end: string; // ISO-8601
	allDay?: boolean;
	title: string;
	eventType: string;
	status: EventStatus;
	notes?: string; // free-text detail (Booking.Notes) — editable via update()
	refs: RefChip[];
	color?: string;
	meta?: Record<string, string>;
}

// Per-event-type visual style (design-5 colours by event TYPE, not status). A muted
// `fill` + a readable `text` foreground + an optional solid `accent` (left bar / dot).
// The HOST owns the event_type dictionary, so the host supplies the map; the package
// ships NO palette of its own (it stays domain-agnostic). Values may be any CSS colour —
// incl. a `var(--token)` or `color-mix(...)` — so a consumer themes from its own tokens.
export interface EventTypeStyle {
	fill: string;
	text: string;
	accent?: string;
	/** Optional human label for the type (e.g. "Handover" for code "key-handover").
	    AgendaList / preview / filter fall back to the capitalised code when absent. */
	label?: string;
}

// Soft guidance (overlap, possible-duplicate, inactive ref) — surfaced by the UI
// WITHOUT blocking. Hard failures (e.g. end ≤ start) are thrown as Errors instead.
export interface Warning {
	code: string;
	message: string;
	field?: string;
	data?: unknown;
}

export interface MutationResult {
	item: CalendarItem;
	warnings: Warning[];
}

export interface DateRange {
	start: string; // ISO-8601
	end: string; // ISO-8601
}

export interface CreateEventInput {
	eventType: string;
	start: string;
	end: string;
	title?: string;
	notes?: string;
}

export interface AddRefInput {
	refType: string;
	refId: string;
	relation?: string;
	asType?: string;
	// Cached display label snapshotted at link time (D11) — survives a later delete of
	// the target entity. Optional: when omitted the adapter synthesizes `refType:refId`.
	label?: string;
}

// A partial edit of an existing event's own fields — the text/type/all-day side
// that `move` (times) and `setStatus` (status) don't cover. Every field is
// optional; only the keys present are applied (a patch). This is what makes the
// host's create form double as an EDITOR (Level 1 verb CreateEvent + the implied
// edit). Re-scheduling is still `move`; status is still `setStatus`.
export interface UpdateEventInput {
	eventType?: string;
	title?: string;
	notes?: string;
	allDay?: boolean;
}

// The frontend-first centerpiece. Tiny surface (F2): renders only events. The mock
// and the future HTTP adapter both implement THIS interface. `update` is the only
// addition beyond the original 6 — editing an event's type/title/notes has no other
// verb (move = times, setStatus = status), and an "Event editor" needs it.
export interface CalendarAdapter {
	list(range: DateRange, scope: 'mine' | 'all'): Promise<CalendarItem[]>;
	create(input: CreateEventInput): Promise<MutationResult>;
	update(id: string, patch: UpdateEventInput): Promise<MutationResult>;
	move(id: string, start: string, end: string): Promise<MutationResult>;
	setStatus(id: string, status: EventStatus): Promise<MutationResult>;
	addReference(id: string, ref: AddRefInput): Promise<MutationResult>;
	removeReference(id: string, referenceId: string): Promise<MutationResult>;
}

// ── DictionaryPort (Level 1 § ②) — canonical + create-on-the-fly ──────────────
// The seam behind EventTypePicker (and, later, person_type / ref_type / relation
// pickers). The mock satisfies it in-memory now; an HTTP-backed port satisfies the
// SAME interface later. No `any`: dict names are a named union, options are typed.
export type DictName = 'event_type' | 'person_type' | 'ref_type' | 'relation';

export interface DictOption {
	code: string;
	label: string;
	icon?: string;
	aliasOf?: string; // canonical code when this row is an alias (D6, depth-1) — deferred feature
}

export interface DictionaryPort {
	// Labels for a picker (canonical + aliases). Stable order.
	options(dict: DictName): Promise<DictOption[]>;
	// Create-on-the-fly (D7). Normalizes + dedupes: an existing label/code returns
	// the existing option rather than a near-duplicate row (A8).
	create(dict: DictName, label: string): Promise<DictOption>;
}
