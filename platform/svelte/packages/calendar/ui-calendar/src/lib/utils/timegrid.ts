// Pure, testable date helpers for @sbx/ui-calendar.
//
// SCOPE (post-@event-calendar refactor): the calendar's grid/drag/resize/overlap
// math now lives inside the vkurko @event-calendar engine. The only pure helpers
// that remain here back the DatePicker mini-month popover (monthMatrix/stepFocus/
// todayISO) and the package's display-zone default — all DOM-free and unit-tested.
//
// Calendar-correct + locale-aware via @internationalized/date (no hand-rolled
// month arithmetic). Wall-clock is always projected into an explicit `timeZone`
// so a focus day resolves the same regardless of the runner's TZ.

import {
	CalendarDate,
	CalendarDateTime,
	getLocalTimeZone,
	parseAbsolute,
	parseDate,
	startOfMonth,
	startOfWeek,
	toCalendarDate,
	toZoned,
	today,
	type ZonedDateTime
} from '@internationalized/date';
import type { CalendarView } from '../types.js';

const pad2 = (n: number): string => String(n).padStart(2, '0');

// Default display zone for BR (Asia/Bangkok has no DST, so wall-clock is stable).
// Every function takes an explicit `timeZone`; this is only the caller-side default.
export const DEFAULT_TIME_ZONE = 'Asia/Bangkok';

/** A single grid cell descriptor for the month matrix (DatePicker). */
export interface MonthCell {
	date: CalendarDate;
	/** false when the day spills in from the previous/next month (greyed). */
	inMonth: boolean;
	/** true when the day is "today" in the display time zone. */
	isToday: boolean;
}

// ── ISO helpers ──────────────────────────────────────────────────────────────

/**
 * Parse an ISO instant into a ZonedDateTime in the given display zone.
 * Falls back to "now" in that zone for unparseable input (defensive, never throws).
 */
export function toZdt(iso: string, timeZone: string): ZonedDateTime {
	try {
		return parseAbsolute(iso, timeZone);
	} catch {
		// parseAbsolute needs an offset/Z; a bare `YYYY-MM-DD` has none. Treat it as
		// wall-clock in the target zone by appending `Z`; if that also fails, anchor
		// to the epoch (deterministic enough — DatePicker only needs the date).
		try {
			return parseAbsolute(`${iso}T00:00:00Z`, timeZone);
		} catch {
			return parseAbsolute(new Date(0).toISOString(), timeZone);
		}
	}
}

// ── @event-calendar boundary conversions ─────────────────────────────────────
// EC v5 runs in a fixed-offset display zone (its `timeZone` option). Its callbacks
// hand back "fake-local" Dates: built via `new Date(y, mo, d, h, mi, s)` from EC's
// internal calendar-zone wall-clock, so the Date's LOCAL components equal the
// intended wall-clock while its absolute instant is browser-dependent. Therefore
// EVERY conversion below reads/writes wall-clock COMPONENTS — never the instant
// (getTime/toISOString) — which makes the calendar correct on ANY browser zone, not
// just one that happens to match the display zone. (This is the class of bug that
// made drag/resize/create "jump" off the +07 machine.)

/**
 * A fake-local EC Date → a zone-tagged ISO in the display `timeZone`
 * (e.g. `2026-05-13T10:00:00+07:00`). Reads the Date's LOCAL components (the
 * calendar wall-clock) and re-tags them in `timeZone` via `toZoned` (DST-correct,
 * browser-independent). The bracketed IANA suffix EC/back-ends can't parse is
 * stripped. Use for every drag/resize/select/click/create start & end.
 */
export function ecLocalDateToISO(date: Date, timeZone: string): string {
	const cdt = new CalendarDateTime(
		date.getFullYear(),
		date.getMonth() + 1,
		date.getDate(),
		date.getHours(),
		date.getMinutes(),
		date.getSeconds()
	);
	return toZoned(cdt, timeZone).toString().replace(/\[[^\]]+\]$/, '');
}

/** Bare `YYYY-MM-DD` from a fake-local EC Date's LOCAL components (calendar-zone day). */
export function ecLocalDayISO(date: Date): string {
	return `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())}`;
}

/**
 * Bare `YYYY-MM-DD` of a focus value — which may be a bare date OR a zoned instant —
 * projected into the display `timeZone`. A bare date is already a calendar day
 * (returned verbatim, no zone shift); an instant is projected via `toZdt`. Replaces
 * the old runtime-local `new Date('…T00:00:00')` parse that drifted east of the zone.
 */
export function focusDayISO(iso: string, timeZone: string): string {
	if (iso.length <= 10) return iso.slice(0, 10);
	return toCalendarDate(toZdt(iso, timeZone)).toString();
}

/** Zone-pure day arithmetic on a bare `YYYY-MM-DD` (no timezone interpretation). */
export function addDaysBareISO(dayISO: string, days: number): string {
	return parseDate(dayISO.slice(0, 10)).add({ days }).toString();
}

/**
 * The display zone's fixed UTC offset (`±HH:MM`) at `atISO`, for EC's `timeZone`
 * option (which accepts an offset string, NOT an IANA name). Fixed-offset means no
 * DST shift WITHIN a view — exact for zero-DST zones like Asia/Bangkok; for DST
 * zones it pins the offset at the focus date (a documented EC limitation).
 */
export function zoneOffsetISO(timeZone: string, atISO: string): string {
	const totalMin = Math.round(toZdt(atISO, timeZone).offset / 60000);
	const sign = totalMin < 0 ? '-' : '+';
	const abs = Math.abs(totalMin);
	return `${sign}${pad2(Math.floor(abs / 60))}:${pad2(abs % 60)}`;
}

/**
 * Should `datesSet` re-sync the host focus date? Only when the focus day has left
 * the visible `[startDay, endDay)` range — i.e. a real navigation, not a pure
 * view-switch (which keeps the same focus day visible). Pure + unit-tested so this
 * loop-breaker can't silently regress (it lived untested inside the component).
 */
export function shouldReemitFocus(focusDay: string, startDay: string, endDay: string): boolean {
	return focusDay < startDay || focusDay >= endDay;
}

/**
 * Is this slot click a double-click on the SAME slot — the second click landing on
 * `lastKey` within `windowMs` of `lastAt`? EC has no native dblclick callback, so the
 * wrapper synthesizes one from two dateClicks. The stateful last-click bookkeeping
 * stays in the component; this pure predicate is the testable decision (the timing
 * logic that would otherwise silently regress, untested inside the component).
 */
export function isRepeatSlotClick(
	slotKey: string,
	now: number,
	lastKey: string,
	lastAt: number,
	windowMs: number
): boolean {
	return slotKey === lastKey && now - lastAt < windowMs;
}

// ── Month structure (DatePicker mini-month) ──────────────────────────────────

function isSameCalendarDay(a: CalendarDate, b: CalendarDate): boolean {
	return a.year === b.year && a.month === b.month && a.day === b.day;
}

/**
 * 6×7 = 42-cell month matrix anchored on `date`'s month. Always 6 rows for a
 * stable grid height. Each cell flags out-of-month + today.
 */
export function monthMatrix(dateISO: string, timeZone: string, locale = 'en-US'): MonthCell[] {
	const anchor = toCalendarDate(toZdt(dateISO, timeZone));
	const gridStart = startOfWeek(startOfMonth(anchor), locale);
	const localTz = getLocalTimeZone();
	const todayDisplay = today(timeZone);
	const todayLocal = today(localTz);
	return Array.from({ length: 42 }, (_, i) => {
		const date = gridStart.add({ days: i });
		return {
			date,
			inMonth: date.month === anchor.month && date.year === anchor.year,
			isToday: isSameCalendarDay(date, todayDisplay) || isSameCalendarDay(date, todayLocal)
		};
	});
}

// ── Focus-date navigation (DatePicker month stepper) ─────────────────────────

/**
 * Step the focus date by one unit of `view`: ±1 day, ±1 week, or ±1 month.
 * Returns a bare `YYYY-MM-DD` ISO (a focus day, not an instant). The DatePicker
 * only uses 'month' steps; day/week are kept for parity with the view contract.
 */
export function stepFocus(dateISO: string, timeZone: string, view: CalendarView, direction: 1 | -1): string {
	const day = toCalendarDate(toZdt(dateISO, timeZone));
	const next =
		view === 'day'
			? day.add({ days: direction })
			: view === 'week'
				? day.add({ weeks: direction })
				: day.add({ months: direction });
	return next.toString();
}

/** Today as a bare `YYYY-MM-DD` ISO in the display zone. */
export function todayISO(timeZone: string): string {
	return today(timeZone).toString();
}
