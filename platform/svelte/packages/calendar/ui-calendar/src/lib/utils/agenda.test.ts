import { describe, expect, it } from 'vitest';
import { dayHeading, groupByDay } from './agenda.js';
import type { CalendarItem } from '../types.js';

function ev(id: string, start: string): CalendarItem {
	return { id, start, end: start, title: id, eventType: 'viewing', status: 'scheduled', refs: [] };
}

describe('groupByDay', () => {
	it('groups by start day — days ascending, items within a day by start', () => {
		const groups = groupByDay([
			ev('b', '2026-06-08T10:00:00+07:00'),
			ev('a2', '2026-06-06T15:00:00+07:00'),
			ev('a1', '2026-06-06T09:00:00+07:00')
		]);
		expect(groups.map((g) => g.day)).toEqual(['2026-06-06', '2026-06-08']);
		expect(groups[0].items.map((i) => i.id)).toEqual(['a1', 'a2']);
		expect(groups[1].items.map((i) => i.id)).toEqual(['b']);
	});

	it('returns [] for no items', () => {
		expect(groupByDay([])).toEqual([]);
	});
});

describe('dayHeading', () => {
	it('prefixes "Today ·" for the current day', () => {
		expect(dayHeading('2026-06-06', '2026-06-06', 'en-US', 'Asia/Bangkok')).toBe(
			'Today · Sat, June 6'
		);
	});

	it('omits the prefix for other days', () => {
		expect(dayHeading('2026-06-08', '2026-06-06', 'en-US', 'Asia/Bangkok')).toBe('Mon, June 8');
	});

	it('omits the prefix when today is undefined', () => {
		expect(dayHeading('2026-06-06', undefined, 'en-US', 'Asia/Bangkok')).toBe('Sat, June 6');
	});
});
