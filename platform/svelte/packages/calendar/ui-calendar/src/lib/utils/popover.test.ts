import { describe, expect, it } from 'vitest';
import { placePopover } from './popover.js';

const VIEWPORT = { width: 1000, height: 800 };
const CARD = { width: 300, height: 200 };

describe('placePopover', () => {
	it('places the card just below the anchor when there is room', () => {
		const p = placePopover({ x: 100, y: 100, width: 120, height: 40 }, CARD, VIEWPORT);
		expect(p).toEqual({ top: 148, left: 100 }); // 100 + 40 + 8 margin; left = anchor.x
	});

	it('flips above the anchor when there is no room below', () => {
		// anchor near the bottom: below would overflow (760+200 > 792), above has room (560>=8).
		const p = placePopover({ x: 100, y: 760, width: 120, height: 40 }, CARD, VIEWPORT);
		expect(p.top).toBe(552); // 760 - 200 - 8
	});

	it('clamps left so the card never overflows the right edge', () => {
		const p = placePopover({ x: 950, y: 100, width: 120, height: 40 }, CARD, VIEWPORT);
		expect(p.left).toBe(692); // 1000 - 300 - 8
	});

	it('clamps left to the margin near the left edge', () => {
		const p = placePopover({ x: 2, y: 100, width: 120, height: 40 }, CARD, VIEWPORT);
		expect(p.left).toBe(8);
	});

	it('centers the card when no anchor is given', () => {
		const p = placePopover(undefined, CARD, VIEWPORT);
		expect(p).toEqual({ top: 300, left: 350 }); // (800-200)/2, (1000-300)/2
	});
});
