import { describe, expect, it } from 'vitest';
import {
	clampCoords,
	clampLat,
	coordsEqual,
	isNullIsland,
	isValidCoords,
	normalizeLng
} from './coords.js';

describe('clampLat', () => {
	it('passes valid latitudes unchanged', () => {
		expect(clampLat(0)).toBe(0);
		expect(clampLat(45.5)).toBe(45.5);
		expect(clampLat(-89.999)).toBe(-89.999);
	});
	it('clamps above max', () => {
		expect(clampLat(95)).toBe(90);
		expect(clampLat(180)).toBe(90);
	});
	it('clamps below min', () => {
		expect(clampLat(-95)).toBe(-90);
	});
	it('replaces NaN and Infinity with 0', () => {
		expect(clampLat(NaN)).toBe(0);
		expect(clampLat(Infinity)).toBe(0);
		expect(clampLat(-Infinity)).toBe(0);
	});
});

describe('normalizeLng', () => {
	it('passes valid longitudes unchanged', () => {
		expect(normalizeLng(0)).toBe(0);
		expect(normalizeLng(100.5)).toBe(100.5);
		expect(normalizeLng(-179.999)).toBe(-179.999);
	});
	it('wraps eastward past 180', () => {
		expect(normalizeLng(185)).toBe(-175);
	});
	it('wraps westward past -180', () => {
		expect(normalizeLng(-185)).toBe(175);
	});
	it('handles NaN', () => {
		expect(normalizeLng(NaN)).toBe(0);
	});
});

describe('clampCoords', () => {
	it('clamps both fields independently', () => {
		expect(clampCoords({ lat: 95, lng: 185 })).toEqual({ lat: 90, lng: -175 });
	});
});

describe('isValidCoords', () => {
	it('accepts valid coords', () => {
		expect(isValidCoords({ lat: 9.7567, lng: 99.9886 })).toBe(true);
		expect(isValidCoords({ lat: 0, lng: 0 })).toBe(true);
	});
	it('rejects null/undefined', () => {
		expect(isValidCoords(null)).toBe(false);
		expect(isValidCoords(undefined)).toBe(false);
	});
	it('rejects out-of-range values', () => {
		expect(isValidCoords({ lat: 95, lng: 0 })).toBe(false);
		expect(isValidCoords({ lat: 0, lng: 185 })).toBe(false);
	});
	it('rejects NaN/Infinity', () => {
		expect(isValidCoords({ lat: NaN, lng: 0 })).toBe(false);
		expect(isValidCoords({ lat: 0, lng: Infinity })).toBe(false);
	});
	it('rejects non-number types', () => {
		expect(isValidCoords({ lat: '9.7', lng: 100 } as unknown as { lat: unknown; lng: unknown })).toBe(false);
	});
});

describe('isNullIsland', () => {
	it('detects exact (0,0)', () => {
		expect(isNullIsland({ lat: 0, lng: 0 })).toBe(true);
	});
	it('rejects near-zero', () => {
		expect(isNullIsland({ lat: 0.0001, lng: 0 })).toBe(false);
		expect(isNullIsland({ lat: 0, lng: 0.0001 })).toBe(false);
	});
	it('rejects real coords', () => {
		expect(isNullIsland({ lat: 9.7567, lng: 99.9886 })).toBe(false);
	});
});

describe('coordsEqual', () => {
	it('treats equal coords as equal', () => {
		expect(coordsEqual({ lat: 1, lng: 2 }, { lat: 1, lng: 2 })).toBe(true);
	});
	it('respects epsilon', () => {
		expect(coordsEqual({ lat: 1.0, lng: 2.0 }, { lat: 1.0000001, lng: 2.0 })).toBe(true);
		expect(coordsEqual({ lat: 1.0, lng: 2.0 }, { lat: 1.01, lng: 2.0 })).toBe(false);
	});
});
