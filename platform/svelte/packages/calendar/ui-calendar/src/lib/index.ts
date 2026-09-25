/// <reference path="./event-calendar.d.ts" />
// @sbx/ui-calendar — public surface.
// The Calendar wrapper renders CalendarItem[] via the vkurko @event-calendar
// engine (an implementation detail behind the adapter seam). The CalendarAdapter
// contract is the only thing a host depends on. Components import NO data source.

// Types — the contract --------------------------------------------------------
export type {
	AddRefInput,
	CalendarAdapter,
	CalendarItem,
	CalendarView,
	CreateEventInput,
	DateRange,
	DictionaryPort,
	DictName,
	DictOption,
	EventStatus,
	EventTypeStyle,
	MutationResult,
	PreviewAnchor,
	RefChip,
	UpdateEventInput,
	Warning
} from './types.js';

// Adapter factories -----------------------------------------------------------
export { createMockAdapter, createMockDictionary } from './adapters/mock.js';
// HTTP adapter — the real backend behind the SAME CalendarAdapter contract (Go manage
// events API). Drop-in for createMockAdapter; see adapters/http.ts.
export { createEventHttpAdapter, type HttpAdapterOptions } from './adapters/http.js';

// Navigation helpers ----------------------------------------------------------
// For a host that supplies its OWN toolbar (Calendar toolbar={false}): step the focus
// date by the active view's unit (prev/next) or jump to today. Pure + display-zone-aware
// (the same helpers DatePicker uses internally), so a custom toolbar needs no date math.
export { stepFocus, todayISO } from './utils/timegrid.js';

// Components ------------------------------------------------------------------
// Calendar is the primary export: the toolbar + view system, filling its
// consumer's frame. DatePicker is a mini-month popover. EventTypePicker is the
// pick-or-create-on-the-fly dict control (D7) the host's event editor composes.
export { default as Calendar } from './Calendar.svelte';
export { default as AgendaList } from './AgendaList.svelte';
// EventPreviewPopover — anchored event-detail popup, reusable from ANY surface (calendar
// grid, dashboard widget, entity activity list). Owns status/remove-ref/edit; the host-
// specific add-ref picker is a snippet slot. See the component header.
export { default as EventPreviewPopover } from './EventPreviewPopover.svelte';
export { default as DatePicker } from './DatePicker.svelte';
// EventTypeFilter — checkbox list of event types (design-5/B Agenda sidebar); also serves
// as a read-only legend. Controlled by the host (hidden set + counts).
export { default as EventTypeFilter } from './EventTypeFilter.svelte';
export { default as EventTypePicker } from './EventTypePicker.svelte';
