import { describe, expect, it } from 'vitest';
import { formatDate } from './date.js';

describe('formatDate', () => {
	it('formats ISO date string with medium style by default', () => {
		expect(formatDate('2026-01-15T00:00:00Z')).toBe('Jan 15, 2026');
	});

	it('short style produces compact form (2-digit year)', () => {
		expect(formatDate('2026-01-15T00:00:00Z', { style: 'short' })).toBe('1/15/26');
	});

	it('long style includes full month name', () => {
		expect(formatDate('2026-01-15T00:00:00Z', { style: 'long' })).toBe('January 15, 2026');
	});

	it('full style includes day of week', () => {
		expect(formatDate('2026-01-15T00:00:00Z', { style: 'full' })).toBe(
			'Thursday, January 15, 2026'
		);
	});

	it('returns empty string for empty input', () => {
		expect(formatDate('')).toBe('');
	});

	it('returns empty string for invalid date', () => {
		expect(formatDate('not-a-date')).toBe('');
	});

	it('interprets date in UTC to avoid day-boundary shift', () => {
		// 2026-06-14 at midnight UTC should render as Jun 14, not Jun 13
		expect(formatDate('2026-06-14T00:00:00Z')).toBe('Jun 14, 2026');
	});
});
