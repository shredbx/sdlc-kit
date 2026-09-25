import { describe, expect, it } from 'vitest';
import { formatNumber } from './number.js';

describe('formatNumber', () => {
	it('formats integer with grouping by default', () => {
		expect(formatNumber(12345)).toBe('12,345');
	});

	it('formats decimal with auto precision', () => {
		expect(formatNumber(12345.6)).toBe('12,345.6');
	});

	it('notation short produces compact form', () => {
		expect(formatNumber(12345, { notation: 'short' })).toBe('12.3K');
	});

	it('decimals off truncates to integer', () => {
		expect(formatNumber(12345.6, { decimals: 'off' })).toBe('12,346');
	});

	it('grouping off removes thousands separator', () => {
		expect(formatNumber(12345, { grouping: 'off' })).toBe('12345');
	});

	it('returns empty string for non-finite values', () => {
		expect(formatNumber(NaN)).toBe('');
		expect(formatNumber(Infinity)).toBe('');
		expect(formatNumber(-Infinity)).toBe('');
	});

	it('handles zero', () => {
		expect(formatNumber(0)).toBe('0');
	});

	it('handles negative numbers', () => {
		expect(formatNumber(-1234)).toBe('-1,234');
	});
});
