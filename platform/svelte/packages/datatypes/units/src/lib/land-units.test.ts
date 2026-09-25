import { describe, expect, it } from 'vitest';
import {
	LAND_SIZE_UNITS,
	convert,
	decimalPlacesFor,
	formatArea,
	formatAreaCompound,
	formatLandSize,
	formatLandSizePlain,
	fromSqm,
	isLandSizeUnit,
	toSqm
} from './land-units.js';

describe('LAND_SIZE_UNITS', () => {
	it('has exact Thai-unit sqm factors', () => {
		// These are domain constants — any drift breaks compatibility
		// with cadastral records. Locking them in tests is intentional.
		expect(LAND_SIZE_UNITS.sqm.sqmFactor).toBe(1);
		expect(LAND_SIZE_UNITS.wah.sqmFactor).toBe(4);
		expect(LAND_SIZE_UNITS.ngan.sqmFactor).toBe(400);
		expect(LAND_SIZE_UNITS.rai.sqmFactor).toBe(1600);
	});

	it('uses exact international sqft factor', () => {
		// 1 international foot = 0.3048 m → 1 sqft = 0.3048² = 0.09290304
		expect(LAND_SIZE_UNITS.sqft.sqmFactor).toBe(0.09290304);
	});
});

describe('toSqm / fromSqm', () => {
	it('round-trips identity for sqm', () => {
		expect(toSqm(200, 'sqm')).toBe(200);
		expect(fromSqm(200, 'sqm')).toBe(200);
	});

	it('converts wah → sqm exactly', () => {
		expect(toSqm(100, 'wah')).toBe(400);
		expect(toSqm(50, 'wah')).toBe(200);
	});

	it('converts rai → sqm exactly', () => {
		expect(toSqm(1, 'rai')).toBe(1600);
		expect(toSqm(2.5, 'rai')).toBe(4000);
	});

	it('converts sqm → rai with division', () => {
		expect(fromSqm(1600, 'rai')).toBe(1);
		expect(fromSqm(200, 'rai')).toBeCloseTo(0.125, 10);
	});

	it('returns NaN for non-finite input', () => {
		expect(toSqm(Number.NaN, 'rai')).toBeNaN();
		expect(toSqm(Number.POSITIVE_INFINITY, 'rai')).toBeNaN();
		expect(fromSqm(Number.NaN, 'rai')).toBeNaN();
	});
});

describe('convert', () => {
	it('returns input unchanged when from === to', () => {
		expect(convert(200, 'sqm', 'sqm')).toBe(200);
		expect(convert(1.5, 'rai', 'rai')).toBe(1.5);
	});

	it('converts sqm → rai correctly', () => {
		expect(convert(1600, 'sqm', 'rai')).toBe(1);
		expect(convert(200, 'sqm', 'rai')).toBeCloseTo(0.125, 10);
	});

	it('converts rai → wah correctly', () => {
		// 1 rai = 1600 sqm = 400 wah
		expect(convert(1, 'rai', 'wah')).toBe(400);
	});

	it('converts ngan → wah correctly', () => {
		// 1 ngan = 400 sqm = 100 wah
		expect(convert(1, 'ngan', 'wah')).toBe(100);
	});

	it('round-trips A → B → A', () => {
		const v = 200;
		expect(convert(convert(v, 'sqm', 'rai'), 'rai', 'sqm')).toBeCloseTo(v, 10);
		expect(convert(convert(v, 'sqm', 'sqft'), 'sqft', 'sqm')).toBeCloseTo(v, 8);
	});
});

describe('isLandSizeUnit', () => {
	it('accepts known unit ids', () => {
		expect(isLandSizeUnit('sqm')).toBe(true);
		expect(isLandSizeUnit('rai')).toBe(true);
		expect(isLandSizeUnit('wah')).toBe(true);
	});

	it('rejects unknown / invalid input', () => {
		expect(isLandSizeUnit('hectares')).toBe(false);
		expect(isLandSizeUnit('')).toBe(false);
		expect(isLandSizeUnit(null)).toBe(false);
		expect(isLandSizeUnit(123)).toBe(false);
	});
});

describe('decimalPlacesFor', () => {
	it('gives 0 decimals for sqm and sqft', () => {
		expect(decimalPlacesFor('sqm')).toBe(0);
		expect(decimalPlacesFor('sqft')).toBe(0);
	});
	it('gives more decimals for larger Thai units', () => {
		expect(decimalPlacesFor('wah')).toBe(2);
		expect(decimalPlacesFor('ngan')).toBe(3);
		expect(decimalPlacesFor('rai')).toBe(3);
	});
});

describe('formatLandSize', () => {
	it('formats sqm without decimals', () => {
		expect(formatLandSize(1600, 'sqm')).toBe('1,600');
	});
	it('formats small rai with 3 decimals', () => {
		expect(formatLandSize(0.125, 'rai')).toBe('0.125');
	});
	it('formats integer rai cleanly without trailing zeros', () => {
		expect(formatLandSize(2, 'rai')).toBe('2');
	});
	it('returns empty string for NaN', () => {
		expect(formatLandSize(Number.NaN, 'sqm')).toBe('');
	});
});

describe('formatLandSizePlain (editable input string — natural precision, no grouping)', () => {
	it('keeps whole-number sqm/sqft intact — never gut trailing zeros (regression: 3300→33)', () => {
		expect(formatLandSizePlain(3300, 'sqm')).toBe('3300');
		expect(formatLandSizePlain(1000, 'sqft')).toBe('1000');
		expect(formatLandSizePlain(200, 'sqm')).toBe('200');
		expect(formatLandSizePlain(10, 'sqft')).toBe('10');
		expect(formatLandSizePlain(0, 'sqm')).toBe('0');
	});
	it('rounds away float tails at the unit natural precision (regression: 3342.99997)', () => {
		expect(formatLandSizePlain(3342.99997, 'sqft')).toBe('3343');
		expect(formatLandSizePlain(310.57486, 'sqm')).toBe('311');
	});
	it('does NOT group digits (raw editable value, unlike formatLandSize)', () => {
		expect(formatLandSizePlain(1600, 'sqm')).toBe('1600');
	});
	it('trims trailing fractional zeros but keeps significant decimals', () => {
		expect(formatLandSizePlain(0.194, 'rai')).toBe('0.194');
		expect(formatLandSizePlain(0.12, 'wah')).toBe('0.12');
		expect(formatLandSizePlain(2.5, 'rai')).toBe('2.5');
		expect(formatLandSizePlain(2, 'rai')).toBe('2');
	});
	it('returns empty string for non-finite', () => {
		expect(formatLandSizePlain(Number.NaN, 'sqm')).toBe('');
		expect(formatLandSizePlain(Number.POSITIVE_INFINITY, 'sqm')).toBe('');
	});
});

describe('formatArea (options-aware, label-appended)', () => {
	// Stored value is always sqm; 8000 sqm = 5 rai = 20 ngan = 2000 wah.
	it('converts to rai and appends the unit label', () => {
		expect(formatArea(8000, { unit: 'rai' })).toBe('5 rai');
	});
	it("labels sqm as 'm²' with thousands separators", () => {
		expect(formatArea(8000, { unit: 'sqm' })).toBe('8,000 m²');
	});
	it('converts to ngan and wah', () => {
		expect(formatArea(8000, { unit: 'ngan' })).toBe('20 ngan');
		expect(formatArea(8000, { unit: 'wah' })).toBe('2,000 wah');
	});
	it("notation 'short' renders compact", () => {
		expect(formatArea(8000, { unit: 'sqm', notation: 'short' })).toBe('8K m²');
	});
	it("decimals 'off' drops the unit-aware precision", () => {
		// 1234 sqm = 0.77125 rai → auto keeps 3dp, off rounds to whole.
		expect(formatArea(1234, { unit: 'rai' })).toBe('0.771 rai');
		expect(formatArea(1234, { unit: 'rai', decimals: 'off' })).toBe('1 rai');
	});
	it('defaults (full/auto) match the prior formatLandSize + label output', () => {
		expect(formatArea(2000, { unit: 'wah' })).toBe(`${formatLandSize(fromSqm(2000, 'wah'), 'wah')} wah`);
	});
	it('returns empty string for non-finite input', () => {
		expect(formatArea(Number.NaN, { unit: 'rai' })).toBe('');
	});
});

describe('formatAreaCompound (Thai rai/ngan/wah decomposition)', () => {
	// 1 rai = 400 wah² = 1600 sqm · 1 ngan = 100 wah² = 400 sqm · 1 wah² = 4 sqm.
	// Work in integer wah² internally so component carry never rolls over.
	it('renders whole rai with no remainder', () => {
		expect(formatAreaCompound(8000)).toBe('5 rai'); // 5 * 1600
	});
	it('renders rai + ngan when the remainder is a whole ngan', () => {
		expect(formatAreaCompound(8400)).toBe('5 rai 1 ngan'); // 8000 + 400
	});
	it('renders the full rai + ngan + wah triple', () => {
		expect(formatAreaCompound(8520)).toBe('5 rai 1 ngan 30 wah'); // 8000 + 400 + 120
	});
	it('drops higher zero components (wah only)', () => {
		expect(formatAreaCompound(120)).toBe('30 wah'); // 120 / 4
	});
	it('renders a lone ngan', () => {
		expect(formatAreaCompound(400)).toBe('1 ngan');
	});
	it('shows 0 wah for an empty area', () => {
		expect(formatAreaCompound(0)).toBe('0 wah');
	});
	it('returns empty string for non-finite / negative input', () => {
		expect(formatAreaCompound(Number.NaN)).toBe('');
		expect(formatAreaCompound(-5)).toBe('');
	});
});
