// TDD — document-level watermark params (Slice A2.2 · #0298).
//
// CASE ENUMERATION
//   TC-WM-01 success  DEFAULT_WATERMARK_PARAMS returns the expected default values
//   TC-WM-02 regress  returns a FRESH object each call (no shared singleton)
//   TC-WM-03 regress  a returned object can be mutated without affecting later calls
//   TC-WM-04 success  SHUTTERSTOCK_CLASSIC_PARAMS returns the shredbx classic look (full-frame)
//   TC-WM-05 regress  SHUTTERSTOCK_CLASSIC_PARAMS is pure — a fresh object each call

import { describe, expect, it } from 'vitest';
import { DEFAULT_WATERMARK_PARAMS, SHUTTERSTOCK_CLASSIC_PARAMS, type WatermarkParams } from './watermark.js';

describe('DEFAULT_WATERMARK_PARAMS', () => {
	it('TC-WM-01 returns the expected default values', () => {
		const expected: WatermarkParams = {
			tiling: 'none',
			rotation: 0,
			density: 100,
			opacity: 50,
			position: 'bottom-right',
			theme: 'dark',
			blend: 'normal'
		};
		expect(DEFAULT_WATERMARK_PARAMS()).toEqual(expected);
	});

	it('TC-WM-02 returns a FRESH object each call (not a shared singleton)', () => {
		expect(DEFAULT_WATERMARK_PARAMS()).not.toBe(DEFAULT_WATERMARK_PARAMS());
	});

	it('TC-WM-03 mutating a returned object never leaks into a later call', () => {
		const first = DEFAULT_WATERMARK_PARAMS();
		first.opacity = 12;
		first.tiling = 'grid';
		expect(DEFAULT_WATERMARK_PARAMS().opacity).toBe(50);
		expect(DEFAULT_WATERMARK_PARAMS().tiling).toBe('none');
	});
});

describe('SHUTTERSTOCK_CLASSIC_PARAMS', () => {
	it('TC-WM-04 returns the shredbx classic look — diagonal, -20°, auto-contrast, full-tile', () => {
		const expected: WatermarkParams = {
			tiling: 'diagonal',
			rotation: -20,
			density: 140,
			opacity: 18,
			position: 'full-tile',
			theme: 'auto',
			blend: 'normal'
		};
		expect(SHUTTERSTOCK_CLASSIC_PARAMS()).toEqual(expected);
	});

	it('TC-WM-05 returns a FRESH object each call (the seed + the Inspector preset share it)', () => {
		const a = SHUTTERSTOCK_CLASSIC_PARAMS();
		a.opacity = 80;
		expect(SHUTTERSTOCK_CLASSIC_PARAMS()).not.toBe(a);
		expect(SHUTTERSTOCK_CLASSIC_PARAMS().opacity).toBe(18);
	});
});
