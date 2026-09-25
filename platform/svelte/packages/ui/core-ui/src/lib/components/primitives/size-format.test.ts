import { describe, it, expect } from 'vitest';
import { formatSizeValue } from './size-format';

// TC-010 / us6-comma-display — comma display, plain value unchanged.
describe('formatSizeValue', () => {
	it('adds thousands separators in comma mode (us6-comma-display)', () => {
		expect(formatSizeValue('1200', 'comma')).toBe('1,200');
		expect(formatSizeValue('1000000', 'comma')).toBe('1,000,000');
	});
	it('preserves a decimal part (sqm with fraction)', () => {
		expect(formatSizeValue('1234.5', 'comma')).toBe('1,234.5');
	});
	it('returns empty for empty value — never injects 0', () => {
		expect(formatSizeValue('', 'comma')).toBe('');
	});
	it('returns the plain value when format is not comma', () => {
		expect(formatSizeValue('1200', 'plain')).toBe('1200');
		expect(formatSizeValue('1200', undefined)).toBe('1200');
	});
});
