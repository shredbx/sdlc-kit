import { describe, it, expect } from 'vitest';
import { stripToDigits, withCommas, formatMoneyDisplay, moneyCaption, formatPrice } from './money';

// TC-001 / us1-format-blur — blur display: commas + currency + suffix.
describe('formatMoneyDisplay', () => {
	it('adds thousands separators with currency (us1-format-blur)', () => {
		expect(formatMoneyDisplay('15000000', 'THB', '')).toBe('15,000,000 THB');
	});
	it('appends a suffix for lease prices', () => {
		expect(formatMoneyDisplay('45000', 'THB', ' / mo')).toBe('45,000 THB / mo');
	});
	it('returns empty string for empty value', () => {
		expect(formatMoneyDisplay('', 'THB', '')).toBe('');
	});
});

// TC-002 / us1-strip-nondigits — only digits survive.
describe('stripToDigits', () => {
	it('removes letters and commas (us1-strip-nondigits)', () => {
		expect(stripToDigits('15,abc000')).toBe('15000');
	});
	it('removes currency symbols and spaces', () => {
		expect(stripToDigits('฿ 1,500,000 THB')).toBe('1500000');
	});
	it('removes a decimal point (whole units only)', () => {
		expect(stripToDigits('1234.56')).toBe('123456');
	});
	it('returns empty string when no digits present', () => {
		expect(stripToDigits('abc')).toBe('');
	});
});

describe('withCommas', () => {
	it('groups integer digits', () => {
		expect(withCommas('1200')).toBe('1,200');
		expect(withCommas('1000000')).toBe('1,000,000');
	});
	it('leaves short numbers unchanged', () => {
		expect(withCommas('999')).toBe('999');
	});
});

// TC-003 / us1-no-caption-small — shorthand caption boundaries.
describe('moneyCaption', () => {
	it('shows nothing below 10K (us1-no-caption-small)', () => {
		expect(moneyCaption('5000', 'THB', '')).toBe('');
		expect(moneyCaption('9999', 'THB', '')).toBe('');
	});
	it('shows K form from exactly 10,000', () => {
		expect(moneyCaption('10000', 'THB', '')).toBe('≈ 10K THB');
		expect(moneyCaption('150000', 'THB', '')).toBe('≈ 150K THB');
	});
	it('shows M form from exactly 1,000,000', () => {
		expect(moneyCaption('1000000', 'THB', '')).toBe('≈ 1M THB');
		expect(moneyCaption('1500000', 'THB', '')).toBe('≈ 1.5M THB');
		expect(moneyCaption('15000000', 'THB', '')).toBe('≈ 15M THB');
	});
	it('drops the decimal for 100M and above', () => {
		expect(moneyCaption('100000000', 'THB', '')).toBe('≈ 100M THB');
	});
	it('carries the suffix into the caption', () => {
		expect(moneyCaption('45000', 'THB', ' / mo')).toBe('≈ 45K THB / mo');
	});
	it('returns empty for empty or non-numeric value', () => {
		expect(moneyCaption('', 'THB', '')).toBe('');
		expect(moneyCaption('not-a-number', 'THB', '')).toBe('');
	});
});

// formatPrice — canonical minor-unit display. 250000000 satang = ฿2,500,000.
describe('formatPrice', () => {
	const SATANG = 250_000_000;
	it("defaults to symbol + full + whole units ({sign}{value})", () => {
		expect(formatPrice(SATANG, 'THB')).toBe('฿2,500,000');
	});
	it("display 'code' renders {value} {code}", () => {
		expect(formatPrice(SATANG, 'THB', { display: 'code' })).toBe('2,500,000 THB');
	});
	it("display 'none' renders the bare number", () => {
		expect(formatPrice(SATANG, 'THB', { display: 'none' })).toBe('2,500,000');
	});
	it("notation 'short' renders compact with the symbol", () => {
		expect(formatPrice(SATANG, 'THB', { notation: 'short' })).toBe('฿2.5M');
	});
	it("combines code + compact", () => {
		expect(formatPrice(SATANG, 'THB', { display: 'code', notation: 'short' })).toBe('2.5M THB');
	});
	it("decimals 'on' shows the currency fraction", () => {
		expect(formatPrice(SATANG, 'THB', { decimals: 'on' })).toBe('฿2,500,000.00');
	});
	it('derives the minor-unit scale from the currency (JPY has 0 decimals)', () => {
		// ¥ has no minor unit → the stored integer is already major units.
		expect(formatPrice(2_500_000, 'JPY', { display: 'none' })).toBe('2,500,000');
	});
	it('falls back to THB for a malformed currency code', () => {
		expect(formatPrice(SATANG, '', { display: 'code' })).toBe('2,500,000 THB');
	});
	it('returns empty string for non-finite input', () => {
		expect(formatPrice(Number.NaN, 'THB')).toBe('');
	});
});
