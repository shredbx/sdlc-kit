<script lang="ts">
	// Calendar — the PUBLIC wrapper and primary export. It renders CalendarItem[]
	// via the mature vkurko @event-calendar v5 engine (MIT, Svelte-5-native) and
	// forwards the SAME callback API the views used before: onCreate / onMove /
	// onItemClick / onViewChange / onDateChange. The library is an implementation
	// detail BEHIND this seam — consumers depend only on these props, never on EC.
	//
	// PRESENTATION ONLY: it imports NO data source. The host owns the adapter and
	// the items; this maps items↔EC events and EC callbacks↔our contract. Fit-to-
	// frame via `height: '100%'` so the calendar ADOPTS the consumer's frame. Brand
	// red #e5392b is the only accent (today highlight + active toolbar button).

	import {
		Calendar as EventCalendar,
		TimeGrid,
		DayGrid,
		List,
		Interaction,
		type EventCalendarInstance
	} from '@event-calendar/core';
	import '@event-calendar/core/index.css';
	import { onMount, untrack } from 'svelte';
	import DatePicker from './DatePicker.svelte';
	import type {
		CalendarItem,
		CalendarView,
		EventStatus,
		EventTypeStyle,
		PreviewAnchor,
		RefChip
	} from './types.js';
	import { resolveEventStyle } from './utils/event-color.js';
	import { refSummary } from './utils/ref-summary.js';
	import {
		DEFAULT_TIME_ZONE,
		addDaysBareISO,
		ecLocalDateToISO,
		ecLocalDayISO,
		focusDayISO,
		isRepeatSlotClick,
		shouldReemitFocus,
		zoneOffsetISO
	} from './utils/timegrid.js';

	interface Props {
		items: CalendarItem[];
		/** Focus day (ISO). Any instant or bare date — the library centers on it. */
		date: string;
		/** Active view. Defaults to 'month'. */
		view?: CalendarView;
		/** The active/selected day (bare YYYY-MM-DD). Renders an active-day highlight. */
		selectedDate?: string;
		/** Create an event. Fired by DOUBLE-click on a slot, or by drag-selecting a range. */
		onCreate?: (startISO: string, endISO: string) => void;
		onMove?: (id: string, startISO: string, endISO: string) => void;
		/** Fired on an event click. The 2nd arg is the event's viewport rect — a host can
		 *  anchor an EventPreviewPopover to it (optional; back-compatible — ignore to skip). */
		onItemClick?: (item: CalendarItem, anchor?: PreviewAnchor) => void;
		/** Emitted when the toggler changes the view. */
		onViewChange?: (view: CalendarView) => void;
		/** Emitted when prev/next/today change the focus date (bare YYYY-MM-DD ISO). */
		onDateChange?: (dateISO: string) => void;
		/**
		 * Emitted on a SINGLE click on a day/slot — selects that day (bare YYYY-MM-DD).
		 * The host can react by filtering its event list, opening a side panel, etc.
		 * The calendar also re-focuses on the day so a view-switch centers on it.
		 */
		onDateSelect?: (dayISO: string) => void;
		timeZone?: string;
		locale?: string;
		/** Clock format for event + axis times. '12h' → "2:00 PM", '24h' → "14:00". Default '12h'. */
		timeFormat?: '12h' | '24h';
		/** When true, a non-blocking loading overlay covers the grid (HTTP-adapter latency). */
		loading?: boolean;
		/**
		 * Hint shown centered when there are NO items to show (and not loading). The host
		 * passes the empty copy (e.g. "No bookings yet — double-click a day to add one").
		 * Omit to show nothing on an empty calendar.
		 */
		emptyHint?: string;
		/**
		 * event_type → muted visual style (design-5 colours by TYPE, not status). The host
		 * owns the event_type dictionary, so the host supplies the map; the package ships no
		 * palette. A per-event CalendarItem.color (legacy solid fill) still takes precedence;
		 * a type absent from the map keeps the status-class default fill (prior look).
		 */
		typeColors?: Record<string, EventTypeStyle>;
		/**
		 * Render EC's built-in header toolbar (prev/next/today · title · view toggle ·
		 * New event). Default true. A host that supplies its OWN toolbar (custom serif
		 * title, stats, brand styling — design-5) sets this false and drives the calendar
		 * via the `view`/`date` props + onViewChange/onDateChange, using the exported
		 * `stepFocus`/`todayISO` helpers for prev/next/today.
		 */
		toolbar?: boolean;
	}

	let {
		items,
		date,
		view = 'month',
		selectedDate,
		onCreate,
		onMove,
		onItemClick,
		onViewChange,
		onDateChange,
		onDateSelect,
		timeZone = DEFAULT_TIME_ZONE,
		locale = 'en-US',
		timeFormat = '12h',
		loading = false,
		emptyHint,
		typeColors,
		toolbar = true
	}: Props = $props();

	// ── Contract ↔ library mapping ───────────────────────────────────────────────

	// Our view modes ↔ EC's view ids. 'list' → a weekly agenda (listWeek).
	const VIEW_TO_EC: Record<CalendarView, string> = {
		month: 'dayGridMonth',
		week: 'timeGridWeek',
		day: 'timeGridDay',
		list: 'listWeek'
	};
	function ecViewToOurs(ecView: string): CalendarView {
		if (ecView === 'dayGridMonth') return 'month';
		if (ecView === 'timeGridDay') return 'day';
		if (ecView === 'listWeek') return 'list';
		return 'week'; // timeGridWeek
	}

	// An EC event shape we control — no `any`. extendedProps carries the bits the
	// library doesn't model so eventClick can reconstruct the CalendarItem. `kind`
	// distinguishes real items from the synthetic active-day highlight (a
	// `display:'background'` band) so the status/content renderers can skip it.
	interface EcEvent {
		id: string;
		title: string;
		start: string;
		end?: string;
		allDay: boolean;
		display?: 'background';
		/** Per-event fill override (CalendarItem.color). EC honors backgroundColor first. */
		backgroundColor?: string;
		extendedProps: {
			kind: 'item' | 'selection';
			status: EventStatus;
			eventType: string;
			refs: RefChip[];
			// Carried so eventClick reconstructs the FULL CalendarItem losslessly — a host
			// that round-trips the clicked item back into its store keeps color + meta.
			color?: string;
			meta?: Record<string, string>;
			// Resolved muted-type colours (design-5). textColor drives the event-content text
			// via a CSS var (we own that DOM); accentColor is the solid left bar / dot.
			textColor?: string;
			accentColor?: string;
		};
	}

	function toEcEvent(item: CalendarItem): EcEvent {
		// design-5 colours by event TYPE (muted fill + readable text); a per-event
		// item.color (legacy solid) still wins; neither → the status-class default fill.
		const style = resolveEventStyle(item, typeColors);
		return {
			id: item.id,
			title: item.title,
			start: item.start,
			end: item.end,
			allDay: item.allDay ?? false,
			// undefined → EC falls back to the status-class fill (--ec-event-bg-color).
			backgroundColor: style?.fill,
			extendedProps: {
				kind: 'item',
				status: item.status,
				eventType: item.eventType,
				refs: item.refs,
				color: item.color,
				meta: item.meta,
				textColor: style?.text,
				accentColor: style?.accent ?? style?.text
			}
		};
	}

	// The active-day highlight: a non-interactive `display:'background'` band on the
	// selected day. Shaped per view — an all-day band tints the month cell; a timed
	// 00:00–23:59:59 band tints the full week/day column. Brand red, faint.
	function selectionEvent(dayISO: string): EcEvent {
		const isMonth = view === 'month';
		return {
			id: '__selected-day__',
			title: '',
			start: isMonth ? dayISO : `${dayISO}T00:00:00`,
			// Month: an all-day band whose end is the NEXT day (exclusive, per EC/FC
			// semantics) — exactly one cell, via zone-pure day arithmetic. Week/Day: a
			// timed band filling the column.
			end: isMonth ? addDaysBareISO(dayISO, 1) : `${dayISO}T23:59:59`,
			allDay: isMonth,
			display: 'background',
			// Colour comes from the .ec-selected-day CSS class (token-driven), not inline.
			extendedProps: { kind: 'selection', status: 'scheduled', eventType: '', refs: [] }
		};
	}

	// EC callbacks hand back FAKE-LOCAL Dates (their LOCAL components equal the
	// display-zone wall-clock; the instant is browser-dependent). Convert via the
	// component-based helper so the ISO is correct on ANY browser zone — never read
	// the instant. start/end are always Dates from EC, so the type is narrowed.
	function toISO(d: Date): string {
		return ecLocalDateToISO(d, timeZone);
	}

	// A clone of a fake-local Date with its LOCAL time set to `hour`:00 — used to seed
	// a working-hours default when creating on an all-day cell (no meaningful time).
	function withLocalHour(d: Date, hour: number): Date {
		const clone = new Date(d);
		clone.setHours(hour, 0, 0, 0);
		return clone;
	}

	// A clone of a fake-local Date advanced by `mins` WALL-CLOCK minutes via the LOCAL
	// field (setMinutes normalizes the local Y/M/D/H/M), NEVER instant math
	// (getTime()+ms). Across a browser-zone DST transition, +30 of instant ≠ +30 of
	// wall-clock; reading back local components after setMinutes keeps it +30 wall-clock,
	// matching ecLocalDateToISO's component-only rule. Seeds the default create duration.
	function addLocalMinutes(d: Date, mins: number): Date {
		const clone = new Date(d);
		clone.setMinutes(clone.getMinutes() + mins);
		return clone;
	}

	// The five soft states drive a status class (NOT inline backgroundColor) so the
	// accent stays token-based and themeable via --ec-status-* in the style block.
	function statusClass(status: EventStatus): string {
		return `ec-status-${status}`;
	}

	// ── eventContent: time + title + (refs) chips — DOM nodes, never raw HTML ─────
	// We build real DOM nodes (not an html string) so the title is inserted as text
	// — no injection, no escaping dance, and NEVER truncated (CSS allows full wrap).

	interface EventContentInfo {
		event: {
			title: string;
			extendedProps: {
				kind?: 'item' | 'selection';
				refs: RefChip[];
				textColor?: string;
				accentColor?: string;
			};
		};
		timeText: string;
	}

	function renderEventContent(info: EventContentInfo): { domNodes: Node[] } {
		// The active-day highlight is a pure background band — no content.
		if (info.event.extendedProps.kind === 'selection') return { domNodes: [] };

		const root = document.createElement('div');
		root.className = 'cal-event';

		// design-5: the resolved muted-type colours ride the content DOM (we own it) as CSS
		// vars — text colour for the time/title/chips, accent for an optional left bar. Absent
		// → the CSS falls back to EC's event-text colour (white on the solid status fill).
		const ext = info.event.extendedProps;
		if (ext.textColor) root.style.setProperty('--cal-event-color', ext.textColor);
		if (ext.accentColor) root.style.setProperty('--cal-event-accent', ext.accentColor);

		if (info.timeText) {
			const time = document.createElement('span');
			time.className = 'cal-event-time';
			time.textContent = info.timeText;
			root.appendChild(time);
		}

		const title = document.createElement('span');
		title.className = 'cal-event-title';
		title.textContent = info.event.title;
		root.appendChild(title);

		// COMPACT ref summary — ONE line of counts by relation (e.g. "👥 2 · 📍 1"),
		// never a per-ref tower (that turned every event into a wall of name chips, the
		// #1 complaint). The full ref list lives in the preview modal + the detail page,
		// so the chip intentionally collapses to counts here. Each non-zero bucket is a
		// small segment: an icon glyph + its count; the segments join with a thin dot.
		const refs = info.event.extendedProps.refs;
		if (refs && refs.length > 0) {
			const { attendees, properties, materials } = refSummary(refs);
			const segments: { icon: string; count: number; label: string }[] = [
				{ icon: '👥', count: attendees, label: 'attendees' },
				{ icon: '📍', count: properties, label: 'properties' },
				{ icon: '🏷️', count: materials, label: 'other references' }
			].filter((s) => s.count > 0);
			if (segments.length > 0) {
				const summary = document.createElement('span');
				summary.className = 'cal-event-refs';
				segments.forEach((seg, i) => {
					if (i > 0) {
						const sep = document.createElement('span');
						sep.className = 'cal-event-refs-sep';
						sep.setAttribute('aria-hidden', 'true');
						sep.textContent = '·';
						summary.appendChild(sep);
					}
					const part = document.createElement('span');
					part.className = 'cal-event-refs-part';
					const glyph = document.createElement('span');
					glyph.setAttribute('aria-hidden', 'true');
					glyph.textContent = seg.icon;
					const num = document.createElement('span');
					num.className = 'cal-event-refs-count';
					// Accessible: the screen reader hears "2 attendees", not the bare glyph.
					num.textContent = String(seg.count);
					num.setAttribute('aria-label', `${seg.count} ${seg.label}`);
					part.append(glyph, num);
					summary.appendChild(part);
				});
				root.appendChild(summary);
			}
		}

		return { domNodes: [root] };
	}

	// ── EC callback info shapes (typed — no `any`) ───────────────────────────────
	interface EcMoveInfo {
		event: { id: string; start: Date; end: Date };
	}
	interface EcClickInfo {
		event: {
			id: string;
			title: string;
			start: Date;
			end: Date;
			allDay: boolean;
			extendedProps: {
				status: EventStatus;
				eventType: string;
				refs: RefChip[];
				color?: string;
				meta?: Record<string, string>;
			};
		};
		/** The native click — EC v5 passes it; we read the event element's rect for anchoring. */
		jsEvent?: MouseEvent;
	}
	interface EcSelectInfo {
		start: Date;
		end: Date;
	}
	interface EcDateClickInfo {
		date: Date;
		allDay: boolean;
	}
	interface EcDatesSetInfo {
		view: { type: string; currentStart?: Date };
		start: Date;
		end: Date;
	}

	const DOUBLE_CLICK_MS = 350;
	// Tracks the last slot click so a second quick click on the SAME slot reads as a
	// double-click. We detect it on dateClick (EC has no native dblclick callback)
	// without debouncing the single click, so selection stays instant.
	let lastClickKey = '';
	let lastClickAt = 0;

	function handleEventDrop(info: EcMoveInfo): void {
		onMove?.(info.event.id, toISO(info.event.start), toISO(info.event.end));
	}
	function handleEventResize(info: EcMoveInfo): void {
		onMove?.(info.event.id, toISO(info.event.start), toISO(info.event.end));
	}
	// Drag across a range → create an event spanning the dragged duration.
	function handleSelect(info: EcSelectInfo): void {
		onCreate?.(toISO(info.start), toISO(info.end));
		ec?.unselect();
	}
	// Keyboard-reachable create (WCAG 2.1.1): EC's dateClick/select are pointer-only and
	// its day cells aren't focusable, so a focusable toolbar "New event" button is the
	// keyboard path to the calendar's primary action. Seeds a 09:00–09:30 working-hours
	// block on the CURRENT focus day (`date`), zone-tagged like every other create path.
	function handleNewEvent(): void {
		if (!onCreate) return;
		const day = focusDayISO(date, timeZone);
		const offset = zoneOffsetISO(timeZone, date);
		onCreate(`${day}T09:00:00${offset}`, `${day}T09:30:00${offset}`);
	}
	// Single click → SELECT the day. Double click on the same slot → CREATE.
	function handleDateClick(info: EcDateClickInfo): void {
		const clicked = info.date;
		const dayISO = ecLocalDayISO(clicked);
		const slotKey = `${dayISO}@${clicked.getTime()}`;
		const now = Date.now();

		if (isRepeatSlotClick(slotKey, now, lastClickKey, lastClickAt, DOUBLE_CLICK_MS)) {
			// Second quick click on the same slot → create. An all-day cell (month /
			// all-day row) has no meaningful time, so seed a 09:00–09:30 working-hours
			// block instead of a 00:00–00:30 midnight sliver. The end is advanced by 30
			// WALL-CLOCK minutes (addLocalMinutes), never instant math, so it survives a
			// browser-zone DST transition.
			const base = info.allDay ? withLocalHour(clicked, 9) : clicked;
			const end = addLocalMinutes(base, 30);
			onCreate?.(toISO(base), toISO(end));
			lastClickKey = '';
			lastClickAt = 0;
			return;
		}

		// First click → select the day (instant) and re-focus so a view-switch
		// centers on it. The host marks it active via the `selectedDate` prop.
		lastClickKey = slotKey;
		lastClickAt = now;
		onDateSelect?.(dayISO);
		onDateChange?.(dayISO);
	}
	// The clicked event's viewport rect (for an anchored preview popover). EC passes the
	// native MouseEvent; read the enclosing `.ec-event` card under it, falling back to the
	// click point, then to undefined (host centers the popover) when no event is available.
	function anchorFromEvent(e?: MouseEvent): PreviewAnchor | undefined {
		if (!e) return undefined;
		const el = (e.target as HTMLElement | null)?.closest?.('.ec-event') as HTMLElement | null;
		if (el) {
			const r = el.getBoundingClientRect();
			return { x: r.left, y: r.top, width: r.width, height: r.height };
		}
		return { x: e.clientX, y: e.clientY, width: 0, height: 0 };
	}
	function handleEventClick(info: EcClickInfo): void {
		onItemClick?.(
			{
				id: info.event.id,
				start: toISO(info.event.start),
				end: toISO(info.event.end),
				allDay: info.event.allDay,
				title: info.event.title,
				eventType: info.event.extendedProps.eventType,
				status: info.event.extendedProps.status,
				refs: info.event.extendedProps.refs,
				color: info.event.extendedProps.color,
				meta: info.event.extendedProps.meta
			},
			anchorFromEvent(info.jsEvent)
		);
	}
	// datesSet fires on view-switch, navigation (prev/next/today) AND initial render.
	// View sync is safe to emit on change. Date sync is the trap: the visible range
	// start (e.g. a month grid's leading day) is NOT the focus day, so emitting it
	// would corrupt the host's focus and break view-switching. We therefore only
	// re-sync the date when the user NAVIGATED AWAY — i.e. the current focus day is
	// no longer inside the visible range [start, end). A pure view-switch keeps the
	// same focus day (it's still visible), so we leave it untouched.
	function handleDatesSet(info: EcDatesSetInfo): void {
		const nextView = ecViewToOurs(info.view.type);
		if (nextView !== view) onViewChange?.(nextView);
		// Day-granular lexical compare (YYYY-MM-DD sorts chronologically). info.start/
		// end are fake-local EC Dates → read their LOCAL day; the focus prop projects
		// into the display zone. shouldReemitFocus is the unit-tested loop-breaker.
		const focusDay = focusDayISO(date, timeZone);
		const startDay = ecLocalDayISO(info.start);
		const endDay = ecLocalDayISO(info.end);
		if (shouldReemitFocus(focusDay, startDay, endDay)) {
			onDateChange?.(ecLocalDayISO(info.view.currentStart ?? info.start));
		}
	}

	// 12h/24h time formatting → Intl options EC applies to event chips + axis labels.
	// '12h' keeps the prior default ("2:00 PM"); '24h' is zero-padded ("14:00").
	function timeFormatOpts(
		fmt: '12h' | '24h'
	): { hour: 'numeric' | '2-digit'; minute: '2-digit'; hour12: boolean } {
		return fmt === '24h'
			? { hour: '2-digit', minute: '2-digit', hour12: false }
			: { hour: 'numeric', minute: '2-digit', hour12: true };
	}

	// ── DatePicker popover (jump-to-date) ────────────────────────────────────────
	// A mini-month popover toggled from a toolbar button. EC owns the toolbar DOM, so
	// on open we MEASURE the rendered `.ec-datePicker` button and anchor the popover
	// under it within the positioned `.calx`. Picking a day navigates the calendar
	// (controlled: drive EC's date for instant feedback + emit onDateChange) and the
	// popover returns focus to its trigger on close (a11y).
	let calxEl = $state<HTMLElement | null>(null);
	let pickerOpen = $state(false);
	let pickerTop = $state(0);
	let pickerLeft = $state(0);
	let pickerTriggerEl = $state<HTMLElement | null>(null);

	function toggleDatePicker(): void {
		if (pickerOpen) {
			closeDatePicker();
			return;
		}
		const trigger = calxEl?.querySelector<HTMLElement>('.ec-button.ec-datePicker') ?? null;
		pickerTriggerEl = trigger;
		if (trigger && calxEl) {
			const btn = trigger.getBoundingClientRect();
			const host = calxEl.getBoundingClientRect();
			const POPOVER_PX = 256; // 16rem — keep in sync with .dp width
			pickerTop = btn.bottom - host.top + 4;
			// Clamp so the popover never overflows the host's right edge.
			pickerLeft = Math.max(0, Math.min(btn.left - host.left, host.width - POPOVER_PX));
		}
		pickerOpen = true;
	}
	function closeDatePicker(): void {
		pickerOpen = false;
		pickerTriggerEl?.focus(); // return focus to the trigger — never lose it on close
	}
	function handleDatePick(dayISO: string): void {
		options.date = dayISO; // drive EC immediately; the guarded $effect ignores the echo
		onDateChange?.(dayISO); // keep the host's `date` prop as the source of truth
	}
	// The anchor is measured once at open, so a scroll/resize would detach the popover
	// from its trigger. Closing on either (rather than re-measuring) matches native
	// popover/<select> UX and is the simplest correct behaviour.
	$effect(() => {
		if (!pickerOpen) return;
		const close = () => closeDatePicker();
		window.addEventListener('scroll', close, true); // capture → catch any scroll container
		window.addEventListener('resize', close);
		return () => {
			window.removeEventListener('scroll', close, true);
			window.removeEventListener('resize', close);
		};
	});

	// ── Reactive EC options ──────────────────────────────────────────────────────
	// The library re-renders when `options` mutates. We keep a single $state object
	// and drive events/view/date from props via $effect so prop changes propagate.
	let ec: EventCalendarInstance | undefined = $state(undefined);

	interface EcOptions {
		view: string;
		date: string;
		/** EC's display zone as a fixed UTC offset (±HH:MM) — see zoneOffsetISO. */
		timeZone: string;
		height: string;
		editable: boolean;
		/** Allow resizing from the START edge too (top in timegrid), not just the end. */
		eventResizableFromStart: boolean;
		selectable: boolean;
		nowIndicator: boolean;
		firstDay: number;
		locale: string;
		headerToolbar: { start: string; center: string; end: string } | false;
		buttonText: Record<string, string>;
		/** Custom toolbar buttons (the "New event" CTA + the "Jump to date" picker). */
		customButtons: Record<string, { text: string; click: () => void }>;
		/** Intl options for event-chip times + time-axis labels (12h/24h). */
		eventTimeFormat: { hour: 'numeric' | '2-digit'; minute: '2-digit'; hour12: boolean };
		slotLabelFormat: { hour: 'numeric' | '2-digit'; minute: '2-digit'; hour12: boolean };
		dayMaxEvents: boolean;
		events: EcEvent[];
		eventContent: (info: EventContentInfo) => { domNodes: Node[] };
		eventClassNames: (info: {
			event: { extendedProps: { kind?: 'item' | 'selection'; status: EventStatus } };
		}) => string[];
		eventDrop: (info: EcMoveInfo) => void;
		eventResize: (info: EcMoveInfo) => void;
		select: (info: EcSelectInfo) => void;
		dateClick: (info: EcDateClickInfo) => void;
		eventClick: (info: EcClickInfo) => void;
		datesSet: (info: EcDatesSetInfo) => void;
	}

	// The initializer captures the CURRENT prop values once (untrack makes that
	// one-time read explicit, not a reactive miss); the $effect block below keeps
	// view/date/locale/events in sync as the props change afterwards.
	let options: EcOptions = $state({
		view: untrack(() => VIEW_TO_EC[view]),
		date: untrack(() => focusDayISO(date, timeZone)),
		timeZone: untrack(() => zoneOffsetISO(timeZone, date)),
		height: '100%',
		editable: true,
		eventResizableFromStart: true, // top-edge (start) resize too — more discoverable
		selectable: true,
		nowIndicator: true,
		firstDay: 0, // Sunday — matches the en-US documented week start
		locale: untrack(() => locale),
		// A host can suppress EC's header entirely (toolbar={false}) to render its own
		// (design-5 custom toolbar); otherwise EC builds the default three-section header.
		headerToolbar: untrack(() =>
			toolbar
				? {
						start: 'prev,next today',
						// The "Jump to date" picker sits right after the focus-date title.
						center: 'title datePicker',
						// The "New event" CTA is appended only when the host accepts creates.
						end: onCreate
							? 'dayGridMonth,timeGridWeek,timeGridDay,listWeek newEvent'
							: 'dayGridMonth,timeGridWeek,timeGridDay,listWeek'
					}
				: false
		),
		buttonText: {
			today: 'Today',
			dayGridMonth: 'Month',
			timeGridWeek: 'Week',
			timeGridDay: 'Day',
			listWeek: 'List'
		},
		customButtons: {
			newEvent: { text: 'New event', click: handleNewEvent },
			datePicker: { text: 'Jump to date', click: toggleDatePicker }
		},
		eventTimeFormat: untrack(() => timeFormatOpts(timeFormat)),
		slotLabelFormat: untrack(() => timeFormatOpts(timeFormat)),
		dayMaxEvents: false, // never hide events behind a "+N more" — show them all
		events: untrack(() => items.map(toEcEvent)),
		eventContent: renderEventContent,
		eventClassNames: (info) =>
			info.event.extendedProps.kind === 'selection'
				? ['ec-selected-day']
				: [statusClass(info.event.extendedProps.status)],
		eventDrop: handleEventDrop,
		eventResize: handleEventResize,
		select: handleSelect,
		dateClick: handleDateClick,
		eventClick: handleEventClick,
		datesSet: handleDatesSet
	});

	// Item → EC-event mapping, memoised so it rebuilds only when `items` change (not
	// on every view/selectedDate toggle).
	const baseEcEvents = $derived(items.map(toEcEvent));
	// EC's display zone as a fixed offset, derived from the IANA prop (+ focus date
	// for DST zones; constant for Asia/Bangkok).
	const ecTimeZone = $derived(zoneOffsetISO(timeZone, date));

	// Prop → option sync. Each guarded so we only assign on real change (EC reacts to
	// any assignment; redundant writes can fight datesSet's own navigation).
	$effect(() => {
		const next = [...baseEcEvents];
		// Append the active-day highlight band (re-shaped whenever the view or selected
		// day changes — selectionEvent reads `view`, tracked by this effect).
		if (selectedDate) next.push(selectionEvent(selectedDate));
		options.events = next;
	});
	$effect(() => {
		const ecView = VIEW_TO_EC[view];
		if (options.view !== ecView) options.view = ecView;
	});
	$effect(() => {
		const day = focusDayISO(date, timeZone);
		if (options.date !== day) options.date = day;
	});
	$effect(() => {
		if (options.timeZone !== ecTimeZone) options.timeZone = ecTimeZone;
	});
	$effect(() => {
		if (options.locale !== locale) options.locale = locale;
	});
	$effect(() => {
		// hour12 is the only field that flips between the two formats.
		if (options.eventTimeFormat.hour12 !== (timeFormat === '12h')) {
			const fmt = timeFormatOpts(timeFormat);
			options.eventTimeFormat = fmt;
			options.slotLabelFormat = fmt;
		}
	});

	// Client-only mount. EC is a fully interactive widget with no SSR value, and our
	// `eventContent` returns live DOM nodes that the engine cannot hydrate (it threw
	// `HierarchyRequestError: appendChild` on first paint). Gating the render on a
	// client-set flag skips SSR entirely, eliminating the hydration mismatch.
	let mounted = $state(false);
	onMount(() => {
		mounted = true;
	});
</script>

<div class="calx" bind:this={calxEl}>
	{#if mounted}
		<EventCalendar bind:this={ec} plugins={[TimeGrid, DayGrid, List, Interaction]} {options} />

		{#if pickerOpen}
			<div class="cal-datepicker-pop" style="top: {pickerTop}px; left: {pickerLeft}px;">
				<DatePicker
					value={date}
					{timeZone}
					{locale}
					anchorEl={pickerTriggerEl}
					onSelect={handleDatePick}
					onClose={closeDatePicker}
				/>
			</div>
		{/if}

		{#if loading}
			<!-- Non-blocking latency overlay (the host toggles `loading` around adapter calls). -->
			<div class="cal-overlay" role="status" aria-live="polite">
				<span class="cal-spinner" aria-hidden="true"></span>
				<span class="cal-overlay-text">Loading…</span>
			</div>
		{:else if emptyHint && items.length === 0}
			<!-- Empty state — shown only when the host supplies copy AND there is nothing to render. -->
			<div class="cal-empty">
				<p class="cal-empty-text">{emptyHint}</p>
			</div>
		{/if}
	{/if}
</div>

<style>
	/* The wrapper fills the consumer's frame; EC's height:'100%' adopts it. NEVER a
	   fixed pixel height — the host's container sets the size. */
	.calx {
		position: relative; /* anchor for the date-picker popover + loading/empty overlays */
		width: 100%;
		height: 100%;
		min-height: 0;
		display: flex;
	}
	.calx > :global(.ec) {
		flex: 1 1 auto;
		min-height: 0;

		/* The ONE accent: brand red. Centralised so every red lives in one place.
		   Deliberately a literal (NOT var(--color-accent)) — the host's accent may be
		   a different colour (e.g. BR gold); the calendar accent is always brand red. */
		/* The calendar accent. Default = brand red, but a host may retheme it by setting
		   --cal-accent-color on any ancestor (BR sets it to --br-color-accent → gold) so the
		   today highlight + active controls match the host brand without forking the package. */
		--cal-accent: var(--cal-accent-color, #e5392b);
		--cal-on-accent: #ffffff;
		/* Calendar glyph for the icon-only "Jump to date" toolbar button (mask → currentColor). */
		--cal-datepicker-icon: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Crect x='3' y='4' width='18' height='18' rx='2'/%3E%3Cline x1='16' y1='2' x2='16' y2='6'/%3E%3Cline x1='8' y1='2' x2='8' y2='6'/%3E%3Cline x1='3' y1='10' x2='21' y2='10'/%3E%3C/svg%3E");

		/* ── Brand theming: map EC's --ec-* to our --color-* tokens ──────────────
		   Backgrounds/text/border inherit the host palette; the today highlight and
		   the active toolbar button use the brand-red --cal-accent ONLY. */
		--ec-bg-color: var(--color-bg, #ffffff);
		--ec-text-color: var(--color-text, #1d1d1b);
		--ec-border-color: var(--color-border, #e4e4e2);
		--ec-highlight-color: color-mix(in srgb, var(--cal-accent) 12%, transparent);
		--ec-today-bg-color: color-mix(in srgb, var(--cal-accent) 7%, transparent);

		/* Toolbar buttons: neutral resting, brand-red when active/pressed. */
		--ec-button-bg-color: var(--color-bg-secondary, #f1f1ef);
		--ec-button-text-color: var(--color-text, #1d1d1b);
		--ec-button-border-color: var(--color-border, #e4e4e2);
		--ec-button-active-bg-color: var(--cal-accent);
		--ec-button-active-text-color: var(--cal-on-accent);

		/* Per-status event fills — LOCAL tokens so a consumer can recolour any status
		   without depending on the host's --color-* palette (which may be absent or, like
		   BR's --color-bg-secondary, an unexpected value). Defaults preserve the prior
		   host mapping; override --cal-status-* to retheme. */
		--cal-status-scheduled: var(--color-info, #2563eb);
		--cal-status-confirmed: var(--color-success, #16a34a);
		--cal-status-done: var(--color-text-muted, #6b7280);
		--cal-status-cancelled: color-mix(in srgb, var(--color-text-muted, #9ca3af) 55%, transparent);

		/* Default event fill = the scheduled status (overridden per-status below). */
		--ec-event-bg-color: var(--cal-status-scheduled);
		--ec-event-text-color: #ffffff;

		font: inherit;
		color: var(--ec-text-color);
		background: var(--ec-bg-color);
	}

	/* Compact, on-brand toolbar (EC builds its own header from headerToolbar). */
	.calx > :global(.ec) :global(.ec-toolbar) {
		gap: 0.4rem 0.75rem;
		margin-bottom: 0.5rem;
		flex-wrap: wrap;
	}
	.calx > :global(.ec) :global(.ec-button) {
		font-size: 0.8rem;
		font-weight: 600;
		padding: 0.3rem 0.7rem;
		border-radius: 0.45rem;
		text-transform: none;
	}
	/* The keyboard-reachable create CTA — brand red, filled (distinct from the neutral
	   view toggles), in the toolbar's right-aligned end section. */
	.calx > :global(.ec) :global(.ec-button.ec-newEvent) {
		background: var(--cal-accent);
		color: var(--cal-on-accent);
		border-color: var(--cal-accent);
	}
	.calx > :global(.ec) :global(.ec-button.ec-newEvent:hover) {
		background: color-mix(in srgb, var(--cal-accent) 88%, #000000);
		border-color: color-mix(in srgb, var(--cal-accent) 88%, #000000);
	}
	/* "Jump to date" — an icon-only ghost button beside the title. The text label
	   ("Jump to date") is the accessible name (font-size:0 hides it visually but keeps
	   it in the a11y tree); the calendar glyph is a currentColor mask. */
	.calx > :global(.ec) :global(.ec-button.ec-datePicker) {
		font-size: 0;
		line-height: 0;
		padding: 0.3rem 0.45rem;
		background: transparent;
		border-color: transparent;
		color: var(--color-text-muted, #8a8a85);
	}
	.calx > :global(.ec) :global(.ec-button.ec-datePicker)::before {
		content: '';
		display: inline-block;
		width: 1rem;
		height: 1rem;
		background-color: currentColor;
		-webkit-mask: var(--cal-datepicker-icon) center / 1rem no-repeat;
		mask: var(--cal-datepicker-icon) center / 1rem no-repeat;
	}
	.calx > :global(.ec) :global(.ec-button.ec-datePicker:hover) {
		color: var(--color-text, #1d1d1b);
		background: color-mix(in srgb, currentColor 8%, transparent);
	}
	.calx > :global(.ec) :global(.ec-title) {
		font-size: 1.05rem;
		font-weight: 600;
		letter-spacing: -0.01em;
	}
	/* Focus ring uses brand red, never browser default. Events are focusable
	   (role=button, tabindex=0) and Enter/Space-activate eventClick — so they MUST
	   show a ring too (keyboard a11y; drag/resize remain pointer-only by design). */
	.calx > :global(.ec) :global(.ec-button:focus-visible),
	.calx > :global(.ec) :global(.ec-event:focus-visible) {
		outline: 2px solid var(--cal-accent);
		outline-offset: 1px;
	}

	/* ── Event hover lift ────────────────────────────────────────────────────────
	   The host hover idiom: a small scale + a playful half-degree tilt + a soft shadow
	   lift, so an event feels grabbable without jumping. Subtle (1.02 / -0.5deg) so it
	   never overlaps neighbours or fights the resizer/selection. position+z-index raise
	   the hovered card above its siblings so the shadow + tilt aren't clipped by the next
	   event. Honours prefers-reduced-motion (shadow only, no transform). */
	.calx > :global(.ec) :global(.ec-event:not(.ec-bg-event)) {
		transition:
			transform 150ms ease,
			box-shadow 150ms ease;
	}
	.calx > :global(.ec) :global(.ec-event:not(.ec-bg-event):hover) {
		transform: scale(1.02) rotate(-0.5deg);
		box-shadow: 0 6px 16px rgba(0, 0, 0, 0.18);
		z-index: 5;
		position: relative;
	}
	@media (prefers-reduced-motion: reduce) {
		.calx > :global(.ec) :global(.ec-event:not(.ec-bg-event)) {
			transition: box-shadow 150ms ease;
		}
		.calx > :global(.ec) :global(.ec-event:not(.ec-bg-event):hover) {
			transform: none;
		}
	}

	/* ── Resize handle affordance ────────────────────────────────────────────────
	   EC's default resizer is a near-invisible ~8px edge strip with only a cursor —
	   the #1 reason resize "feels broken". Make it discoverable: a translucent-white
	   grip on hover/focus (high contrast over the coloured event card) AND a larger hit
	   target (EC binds pointerdown to this div, so growing it genuinely enlarges the grab
	   zone). timegrid = bottom/top edge (block axis); daygrid = left/right edge (inline). */
	.calx > :global(.ec) :global(.ec-event) :global(.ec-resizer) {
		min-block-size: 10px;
		min-inline-size: 10px;
		/* Transparent at REST (no shading on the card edges); the grip appears only
		   on hover/focus, the standard calendar pattern. The enlarged hit target +
		   resize cursor still make it discoverable. */
		background: transparent;
		border-radius: 2px;
		transition: background 0.12s ease;
	}
	.calx > :global(.ec) :global(.ec-event:hover) :global(.ec-resizer),
	.calx > :global(.ec) :global(.ec-event:focus-visible) :global(.ec-resizer) {
		background: color-mix(in srgb, var(--cal-on-accent) 55%, transparent);
	}

	/* ── Active-day highlight ────────────────────────────────────────────────────
	   The synthetic `display:'background'` selection band. Brand red, faint — tints
	   the selected month cell / week-day column without obscuring its events. */
	.calx > :global(.ec) :global(.ec-bg-event.ec-selected-day) {
		background-color: color-mix(in srgb, var(--cal-accent) 9%, transparent);
	}

	/* ── Month event-card breathing space ────────────────────────────────────────
	   EC's daygrid stacks events in a CSS subgrid; even with EC's own vertical gap the
	   stacked cards read as one block against the cell. Inset each card inline and ring
	   it in the CELL colour so the card edge is unambiguous and the seam disappears. The
	   ring tracks --ec-day-bg-color (the today/highlight tint), NOT the flat page
	   --color-bg — otherwise a white halo paints over a red-tinted today/selected cell.
	   Scoped to month; timegrid events are time-positioned and need no inset. */
	.calx > :global(.ec.ec-month-view) :global(.ec-event) {
		margin-inline: 3px;
		border-radius: 5px;
		box-shadow: 0 0 0 2px var(--ec-day-bg-color, var(--color-bg, #ffffff));
	}

	/* ── Week / Day production polish (timegrid) ─────────────────────────────────
	   EC's generic timegrid leaves the BR-themed header/axis/gridlines blended into the
	   page bg (the col-head border resolves transparent, the header + all-day row carry no
	   separator), so the grid reads "loose + misaligned". These overrides give the week/day
	   views a tidy, aligned chrome WITHOUT touching the engine: single-line day headers, a
	   crisp header separator, visible day-column + hour gridlines, a clean all-day band, and
	   a readable time axis. All token-driven (--color-*) so the host theme drives the colour. */

	/* Day-column headers: a single-line "Sun, 6/7" (NEVER wrapped to two lines — the #1
	   week-view complaint), centered, with a visible bottom separator under the whole row. */
	.calx > :global(.ec.ec-time-grid) :global(.ec-header) {
		border-bottom: 1px solid var(--color-border, #e4e4e2);
	}
	.calx > :global(.ec.ec-time-grid) :global(.ec-col-head) {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0.45rem 0.25rem;
		border-right: 1px solid var(--color-border, #e4e4e2);
		background: var(--color-bg, #ffffff);
	}
	.calx > :global(.ec.ec-time-grid) :global(.ec-col-head) :global(time) {
		/* Single line, always — no wrap, no clip (it fits; nowrap just forbids the break). */
		white-space: nowrap;
		font-size: 0.78rem;
		font-weight: 600;
		letter-spacing: -0.01em;
		color: var(--color-text, #1d1d1b);
	}
	/* Today's column header reads in the brand accent so "where am I" is obvious. */
	.calx > :global(.ec.ec-time-grid) :global(.ec-col-head.ec-today) :global(time) {
		color: var(--cal-accent);
	}

	/* All-day band: a subtle surface + a bottom rule so it reads as its own row, not a gap.
	   The surface is a faint mix of the BORDER token over the bg (NOT --color-bg-secondary —
	   a host like BR maps that to a dark brand fill, which would paint the band dark teal).
	   The border-token mix is guaranteed light-on-light / subtle on any theme. */
	.calx > :global(.ec.ec-time-grid) :global(.ec-all-day) {
		background: color-mix(in srgb, var(--color-border, #e4e4e2) 28%, var(--color-bg, #ffffff));
		border-bottom: 1px solid var(--color-border, #e4e4e2);
	}

	/* Hour gridlines + day-column separators: track the border token (not transparent), so
	   the grid is legibly ruled. EC paints these on .ec-line / day columns — make them visible. */
	.calx > :global(.ec.ec-time-grid) :global(.ec-line) {
		border-bottom: 1px solid color-mix(in srgb, var(--color-border, #e4e4e2) 70%, transparent);
	}
	.calx > :global(.ec.ec-time-grid) :global(.ec-body) :global(.ec-days) :global(.ec-day) {
		border-right: 1px solid color-mix(in srgb, var(--color-border, #e4e4e2) 70%, transparent);
	}

	/* Time axis: right-aligned, muted, lifted off the hour line so labels never overlap it. */
	.calx > :global(.ec.ec-time-grid) :global(.ec-time) {
		font-size: 0.7rem;
		font-weight: 500;
		color: var(--color-text-muted, #8a8a85);
	}
	.calx > :global(.ec.ec-time-grid) :global(.ec-sidebar) {
		border-right: 1px solid var(--color-border, #e4e4e2);
	}

	/* ── Status accents ────────────────────────────────────────────────────────
	   Status colour comes from a class (set via eventClassNames), NOT inline
	   backgroundColor — so the palette stays token-driven and overridable. Each
	   fill reads a --cal-status-* token (defined on .ec above), so a consumer
	   retheme touches the tokens, not these rules.
	   scheduled → info · confirmed → success · done → grey-strong ·
	   cancelled/no_show → muted grey (cancelled never deleted, just greyed). */
	.calx > :global(.ec) :global(.ec-event.ec-status-scheduled) {
		--ec-event-bg-color: var(--cal-status-scheduled);
	}
	.calx > :global(.ec) :global(.ec-event.ec-status-confirmed) {
		--ec-event-bg-color: var(--cal-status-confirmed);
	}
	.calx > :global(.ec) :global(.ec-event.ec-status-done) {
		--ec-event-bg-color: var(--cal-status-done);
	}
	.calx > :global(.ec) :global(.ec-event.ec-status-cancelled),
	.calx > :global(.ec) :global(.ec-event.ec-status-no_show) {
		--ec-event-bg-color: var(--cal-status-cancelled);
		opacity: 0.85;
	}
	.calx > :global(.ec) :global(.ec-event.ec-status-cancelled) :global(.cal-event-title),
	.calx > :global(.ec) :global(.ec-event.ec-status-no_show) :global(.cal-event-title) {
		text-decoration: line-through;
	}
	/* The compact ref summary inherits .cal-event's resolved colour, so it stays legible
	   on every fill (incl. the faint grey cancelled/no_show) with no per-status override. */

	/* ── Event content: time + title + chips. NEVER truncated. ─────────────────
	   overflow-wrap:anywhere + normal white-space lets long titles wrap fully;
	   no ellipsis, no line-clamp, no overflow:hidden on the text. */
	.calx > :global(.ec) :global(.cal-event) {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		width: 100%;
		padding: 0.05rem 0;
		line-height: 1.25;
		/* design-5: a muted-type event carries its readable text colour via --cal-event-color
		   (set on this node in renderEventContent). Absent (uncoloured host / status-fill
		   default) → EC's event-text colour (white on the solid status fill). currentColor for
		   the chips + time then tracks whichever applies. */
		color: var(--cal-event-color, var(--ec-event-text-color, #ffffff));
	}
	.calx > :global(.ec) :global(.cal-event-time) {
		font-size: 0.7rem;
		font-weight: 600;
		opacity: 0.92;
	}
	.calx > :global(.ec) :global(.cal-event-title) {
		font-size: 0.78rem;
		font-weight: 600;
		overflow-wrap: anywhere;
		white-space: normal;
		/* COMPACT chip ONLY: clamp the title to ~2 lines so a long title can't grow the
		   event into a tower (the modal + detail page show it in full — the "never clip"
		   rule is satisfied THERE, not in the dense month/agenda chip). */
		display: -webkit-box;
		-webkit-box-orient: vertical;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		overflow: hidden;
	}
	/* COMPACT ref summary — ONE line of counts (icon + number per relation bucket),
	   replacing the old per-ref chip tower. currentColor tracks the event's resolved
	   text colour, so it reads on both the solid status fill (white) and a muted-type
	   fill (dark). Single line: it never grows the card. */
	.calx > :global(.ec) :global(.cal-event-refs) {
		display: inline-flex;
		align-items: center;
		flex-wrap: nowrap;
		gap: 0.25rem;
		margin-top: 0.05rem;
		font-size: 0.66rem;
		font-weight: 600;
		opacity: 0.92;
		white-space: nowrap;
	}
	.calx > :global(.ec) :global(.cal-event-refs-part) {
		display: inline-flex;
		align-items: center;
		gap: 0.12rem;
	}
	.calx > :global(.ec) :global(.cal-event-refs-count) {
		font-variant-numeric: tabular-nums;
	}
	.calx > :global(.ec) :global(.cal-event-refs-sep) {
		opacity: 0.55;
	}

	/* ── Date-picker popover ─────────────────────────────────────────────────────
	   Absolutely positioned within .calx, anchored under the toolbar trigger (the
	   host measures the button rect → pickerTop/pickerLeft). */
	.cal-datepicker-pop {
		position: absolute;
		z-index: 20;
	}

	/* ── Loading + empty overlays ────────────────────────────────────────────────
	   Loading is NON-blocking (pointer-events:none) so the stale grid stays visible
	   and interactive underneath while data refreshes — latency reads as "refreshing",
	   not "empty". The empty hint is likewise pass-through so double-click-to-create
	   still works on an empty calendar. */
	.cal-overlay {
		position: absolute;
		inset: 0;
		z-index: 15;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		pointer-events: none;
		background: color-mix(in srgb, var(--ec-bg-color, #ffffff) 55%, transparent);
	}
	.cal-overlay-text {
		font-size: 0.85rem;
		font-weight: 600;
		color: var(--ec-text-color, #1d1d1b);
	}
	.cal-spinner {
		width: 1.1rem;
		height: 1.1rem;
		border-radius: 50%;
		border: 2px solid color-mix(in srgb, currentColor 25%, transparent);
		border-top-color: var(--cal-accent);
		animation: cal-spin 0.7s linear infinite;
	}
	@keyframes cal-spin {
		to {
			transform: rotate(360deg);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.cal-spinner {
			animation-duration: 2.4s;
		}
	}
	.cal-empty {
		position: absolute;
		inset: 0;
		z-index: 5;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1rem;
		pointer-events: none;
	}
	.cal-empty-text {
		margin: 0;
		max-width: 22rem;
		text-align: center;
		font-size: 0.9rem;
		line-height: 1.5;
		color: var(--color-text-muted, #8a8a85);
	}
</style>
