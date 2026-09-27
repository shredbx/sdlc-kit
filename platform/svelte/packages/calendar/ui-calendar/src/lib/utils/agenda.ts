// Pure grouping + day-heading helpers for AgendaList (the Upcoming rail + the Agenda
// body share them). No DOM, no Date.now() — `today` is passed in — so they unit-test in
// isolation and stay deterministic. Items are display-zone ISO (…+07:00 wall-clock), so a
// calendar day is the bare YYYY-MM-DD prefix (the same rule the host day-list uses).
import { parseDate } from '@internationalized/date';
import type { CalendarItem } from '../types.js';

export interface DayGroup {
	/** Bare YYYY-MM-DD (display-zone wall-clock day). */
	day: string;
	items: CalendarItem[];
}

// Group items by their start day — days ascending, items within a day by start time.
// Empty days are absent (the agenda shows only days that have events — matches design-5).
export function groupByDay(items: CalendarItem[]): DayGroup[] {
	const byDay = new Map<string, CalendarItem[]>();
	for (const item of items) {
		const day = item.start.slice(0, 10);
		const bucket = byDay.get(day);
		if (bucket) bucket.push(item);
		else byDay.set(day, [item]);
	}
	return [...byDay.entries()]
		.sort((a, b) => a[0].localeCompare(b[0]))
		.map(([day, bucket]) => ({
			day,
			items: [...bucket].sort((a, b) => a.start.localeCompare(b.start))
		}));
}

// A human day heading: "Today · Sat, June 6" for the current day, else "Sat, June 6".
// `today` (bare YYYY-MM-DD) is injected so this stays pure + testable; when omitted the
// "Today" prefix is simply never applied. parseDate→toDate keeps the day correct on any
// browser zone (never a bare new Date(dayISO), which would shift west of the display zone).
export function dayHeading(
	dayISO: string,
	today: string | undefined,
	locale: string,
	timeZone: string
): string {
	const full = new Intl.DateTimeFormat(locale, {
		weekday: 'short',
		month: 'long',
		day: 'numeric',
		timeZone
	}).format(parseDate(dayISO).toDate(timeZone));
	return today && dayISO === today ? `Today · ${full}` : full;
}
