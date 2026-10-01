// @vitest-environment jsdom
//
// ensureGoogleFont loader tests (@sbx/core-ui/fonts). jsdom has NO FontFace set,
// so we stub `document.fonts` with a controllable `load`/`check` to assert the
// contract WITHOUT a real network/font engine:
//   - idempotency: a family is requested once; a second call shares the promise
//     and injects no second <link> and fires no second `load`;
//   - SSR/no-FontFace safety: resolves false, injects nothing;
//   - already-available faces (brand/system) resolve true with no stylesheet;
//   - markFontLoaded pre-registers a face so a later ensureGoogleFont no-ops.

import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
	ensureGoogleFont,
	markFontLoaded,
	isFontRequested,
	__resetFontCacheForTests
} from './fonts';

/** Install a stub FontFaceSet on document. `available` = families check() returns true for. */
function stubFontSet(opts: { available?: Set<string>; loadResult?: boolean } = {}) {
	const available = opts.available ?? new Set<string>();
	const load = vi.fn(async (spec: string) => {
		// Simulate the face becoming available after a successful load.
		if (opts.loadResult !== false) {
			const m = /"([^"]+)"/.exec(spec);
			if (m) available.add(m[1]);
		}
		return [];
	});
	const check = vi.fn((spec: string) => {
		const m = /"([^"]+)"/.exec(spec);
		return m ? available.has(m[1]) : false;
	});
	Object.defineProperty(document, 'fonts', {
		configurable: true,
		value: { load, check }
	});
	return { load, check, available };
}

function linkCount(): number {
	return document.querySelectorAll('link[data-sbx-font]').length;
}

beforeEach(() => {
	__resetFontCacheForTests();
	document.head.querySelectorAll('link[data-sbx-font]').forEach((n) => n.remove());
});

describe('ensureGoogleFont — idempotency', () => {
	it('loads a family once; a repeat call shares the promise (no 2nd link, no 2nd load)', async () => {
		const { load } = stubFontSet();

		const p1 = ensureGoogleFont('Roboto');
		const p2 = ensureGoogleFont('Roboto');
		expect(p2).toBe(p1); // same promise instance — deduped

		await Promise.all([p1, p2]);

		expect(load).toHaveBeenCalledTimes(1);
		expect(linkCount()).toBe(1);
		expect(isFontRequested('Roboto')).toBe(true);
	});

	it('a third call AFTER settle still does not re-inject or re-load', async () => {
		const { load } = stubFontSet();
		await ensureGoogleFont('Lato');
		await ensureGoogleFont('Lato');
		expect(load).toHaveBeenCalledTimes(1);
		expect(linkCount()).toBe(1);
	});

	it('different families each load once', async () => {
		const { load } = stubFontSet();
		await Promise.all([
			ensureGoogleFont('Roboto'),
			ensureGoogleFont('Open Sans'),
			ensureGoogleFont('Lato')
		]);
		expect(load).toHaveBeenCalledTimes(3);
		expect(linkCount()).toBe(3);
	});

	it('encodes spaces as "+" in the stylesheet href', async () => {
		stubFontSet();
		await ensureGoogleFont('Open Sans');
		const link = document.querySelector<HTMLLinkElement>('link[data-sbx-font="Open Sans"]');
		expect(link?.href).toContain('family=Open+Sans');
	});

	it('resolves true once the face becomes available', async () => {
		stubFontSet();
		await expect(ensureGoogleFont('Roboto')).resolves.toBe(true);
	});
});

describe('ensureGoogleFont — already available (brand/system)', () => {
	it('no-ops with no stylesheet when the face is already paintable', async () => {
		const { load } = stubFontSet({ available: new Set(['Inter']) });
		await expect(ensureGoogleFont('Inter')).resolves.toBe(true);
		expect(load).not.toHaveBeenCalled();
		expect(linkCount()).toBe(0);
	});
});

describe('ensureGoogleFont — guards', () => {
	it('resolves false for an empty / whitespace family', async () => {
		stubFontSet();
		await expect(ensureGoogleFont('')).resolves.toBe(false);
		await expect(ensureGoogleFont('   ')).resolves.toBe(false);
		expect(linkCount()).toBe(0);
	});

	it('resolves false (never throws) when there is no FontFace set', async () => {
		Object.defineProperty(document, 'fonts', { configurable: true, value: undefined });
		await expect(ensureGoogleFont('Roboto')).resolves.toBe(false);
		expect(linkCount()).toBe(0);
	});

	it('resolves false when load rejects', async () => {
		const load = vi.fn().mockRejectedValue(new Error('network'));
		const check = vi.fn().mockReturnValue(false);
		Object.defineProperty(document, 'fonts', { configurable: true, value: { load, check } });
		await expect(ensureGoogleFont('Broken')).resolves.toBe(false);
	});
});

describe('markFontLoaded', () => {
	it('pre-registers a family so a later ensureGoogleFont no-ops', async () => {
		const { load } = stubFontSet();
		markFontLoaded('Inter');
		expect(isFontRequested('Inter')).toBe(true);
		await expect(ensureGoogleFont('Inter')).resolves.toBe(true);
		expect(load).not.toHaveBeenCalled();
		expect(linkCount()).toBe(0);
	});

	it('ignores an empty family', () => {
		markFontLoaded('');
		expect(isFontRequested('')).toBe(false);
	});
});
