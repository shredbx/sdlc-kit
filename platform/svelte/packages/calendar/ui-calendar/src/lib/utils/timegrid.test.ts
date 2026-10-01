// Pure-math contract for the surviving date helpers (DatePicker mini-month +
// focus navigation). No DOM, no rendering. Asia/Bangkok wall-clock (no DST) in
// the fixtures, matching the adapter contract suite. The grid/drag/resize math
// moved into the @event-calendar engine, so those tests are gone with it.

import { describe, expect, it } from 'vitest';
import {
	addDaysBareISO,
	ecLocalDateToISO,
	ecLocalDayISO,
	focusDayISO,
	isRepeatSlotClick,
	monthMatrix,
	shouldReemitFocus,
	stepFocus,
	todayISO,
	zoneOffsetISO
} from './timegrid.js';

const TZ = 'Asia/Bangkok';

describe('ecLocalDateToISO — fake-local EC Date → display-zone wall-clock ISO', () => {
	// EC hands back a Date whose LOCAL components are the calendar wall-clock. These
	// fixtures construct exactly that. The function must read the COMPONENTS and
	// re-tag them in the passed zone — never the (browser-dependent) instant — so the
	// SAME wall-clock yields different offsets per zone WITHOUT shifting the clock.
	const fakeLocal = new Date(2026, 5, 11, 10, 0, 0); // 2026-06-11 10:00 (local components)
	it('tags the wall-clock in Bangkok (+07:00)', () => {
		expect(ecLocalDateToISO(fakeLocal, 'Asia/Bangkok')).toBe('2026-06-11T10:00:00+07:00');
	});
	it('re-tags the SAME wall-clock in a non-Bangkok zone — browser-independent', () => {
		// 2026-06-11 is EDT in New York → -04:00. The clock stays 10:00; only the tag
		// changes. This is the exact case the +07-only test machine could never expose.
		expect(ecLocalDateToISO(fakeLocal, 'America/New_York')).toBe('2026-06-11T10:00:00-04:00');
	});
	it('strips the bracketed IANA suffix (EC/back-ends cannot parse it)', () => {
		expect(ecLocalDateToISO(fakeLocal, TZ)).not.toMatch(/\[/);
	});
});

describe('ecLocalDayISO — bare day from a fake-local Date', () => {
	it('reads Y/M/D from local components', () => {
		expect(ecLocalDayISO(new Date(2026, 5, 11, 23, 30, 0))).toBe('2026-06-11');
	});
});

describe('focusDayISO — focus prop (bare OR instant) → display-zone day', () => {
	it('passes a bare date through unchanged (no zone shift)', () => {
		expect(focusDayISO('2026-06-11', TZ)).toBe('2026-06-11');
	});
	it('projects an instant into the display zone', () => {
		// 22:00-04:00 (NY) on Jun 11 is 09:00+07:00 on Jun 12 in Bangkok → the 12th.
		expect(focusDayISO('2026-06-11T22:00:00-04:00', TZ)).toBe('2026-06-12');
	});
});

describe('addDaysBareISO — zone-pure day arithmetic', () => {
	it('adds a day with no off-by-one and crosses month boundaries', () => {
		expect(addDaysBareISO('2026-06-11', 1)).toBe('2026-06-12');
		expect(addDaysBareISO('2026-06-30', 1)).toBe('2026-07-01');
	});
});

describe('zoneOffsetISO — EC timeZone option (±HH:MM)', () => {
	it('derives the fixed offset for the display zone', () => {
		expect(zoneOffsetISO('Asia/Bangkok', '2026-06-11')).toBe('+07:00');
		expect(zoneOffsetISO('UTC', '2026-06-11')).toBe('+00:00');
	});
	// The sign / abs%60 minute-remainder / zero-pad branches are exactly the cases +07
	// and +00 never exercise — half-hour, negative, and high-positive zones.
	it('handles half-hour, negative, and +14 zones (the branches +07 never hits)', () => {
		expect(zoneOffsetISO('Asia/Kolkata', '2026-06-11')).toBe('+05:30'); // half-hour minute remainder
		expect(zoneOffsetISO('Pacific/Kiritimati', '2026-06-11')).toBe('+14:00'); // > +12 hours
		expect(zoneOffsetISO('America/New_York', '2026-06-11')).toBe('-04:00'); // negative sign (EDT)
	});
	it('pins the offset at the FOCUS DATE for DST zones (documented limitation)', () => {
		// Same IANA zone, two focus dates → two offsets. EC gets a fixed offset per view,
		// re-derived as the focus date moves across a DST boundary.
		expect(zoneOffsetISO('America/New_York', '2026-06-11')).toBe('-04:00'); // summer (EDT)
		expect(zoneOffsetISO('America/New_York', '2026-01-11')).toBe('-05:00'); // winter (EST)
	});
});

describe('isRepeatSlotClick — synthesized double-click decision', () => {
	it('is a repeat only on the SAME slot within the window', () => {
		expect(isRepeatSlotClick('d@100', 1300, 'd@100', 1000, 350)).toBe(true); // same key, 300ms < 350
		expect(isRepeatSlotClick('d@100', 1400, 'd@100', 1000, 350)).toBe(false); // 400ms ≥ window
		expect(isRepeatSlotClick('d@200', 1100, 'd@100', 1000, 350)).toBe(false); // different slot
		expect(isRepeatSlotClick('d@100', 1000, '', 0, 350)).toBe(false); // no prior click
	});
});

describe('shouldReemitFocus — datesSet loop-breaker', () => {
	it('re-emits only when the focus day leaves the visible [start,end) range', () => {
		// focus inside a month grid → pure view-switch, do NOT re-emit.
		expect(shouldReemitFocus('2026-06-11', '2026-05-31', '2026-07-05')).toBe(false);
		// focus before the range → navigated away, re-emit.
		expect(shouldReemitFocus('2026-04-01', '2026-05-31', '2026-07-05')).toBe(true);
		// end is EXCLUSIVE — a focus day equal to endDay is outside.
		expect(shouldReemitFocus('2026-07-05', '2026-05-31', '2026-07-05')).toBe(true);
	});
});

describe('monthMatrix — 6×7 with out-of-month flags', () => {
	const cells = monthMatrix('2026-05-12T14:00:00+07:00', TZ, 'en-US');
	it('is exactly 42 cells (6 rows × 7 cols)', () => expect(cells).toHaveLength(42));
	it('flags leading days from April as out-of-month', () => {
		// May 2026 starts on a Friday; en-US grid begins Sun Apr 26 → those are outside.
		expect(cells[0].inMonth).toBe(false);
		expect(cells[0].date.month).toBe(4);
	});
	it('includes in-month days flagged true', () => {
		const may1 = cells.find((c) => c.date.month === 5 && c.date.day === 1);
		expect(may1?.inMonth).toBe(true);
	});
	it('flags trailing June days as out-of-month', () => {
		expect(cells[41].inMonth).toBe(false);
	});
	it('accepts a bare YYYY-MM-DD focus day (no offset)', () => {
		const bare = monthMatrix('2026-05-12', TZ, 'en-US');
		expect(bare).toHaveLength(42);
		expect(bare.some((c) => c.date.month === 5 && c.date.day === 1 && c.inMonth)).toBe(true);
	});
	it('anchors on the DISPLAY zone, not the runner zone (cross-zone fixture)', () => {
		// Same structural facts must hold under a non-Bangkok display zone — proving the
		// matrix is display-zone-anchored, the case the +07-only suite could never expose.
		const ny = monthMatrix('2026-05-12', 'America/New_York', 'en-US');
		expect(ny).toHaveLength(42);
		expect(ny[0].inMonth).toBe(false); // leading April day
		expect(ny[0].date.month).toBe(4);
		expect(ny.some((c) => c.date.month === 5 && c.date.day === 1 && c.inMonth)).toBe(true);
	});
});

describe('stepFocus — DatePicker month stepper', () => {
	it('steps ±1 month in month view', () => {
		expect(stepFocus('2026-05-12', TZ, 'month', 1)).toBe('2026-06-12');
		expect(stepFocus('2026-05-12', TZ, 'month', -1)).toBe('2026-04-12');
	});
	it('steps ±1 day / ±1 week for view parity', () => {
		expect(stepFocus('2026-05-12', TZ, 'day', 1)).toBe('2026-05-13');
		expect(stepFocus('2026-05-12', TZ, 'week', 1)).toBe('2026-05-19');
	});
});

describe('todayISO — bare YYYY-MM-DD in the display zone', () => {
	it('matches the YYYY-MM-DD shape', () => {
		expect(todayISO(TZ)).toMatch(/^\d{4}-\d{2}-\d{2}$/);
	});
});
