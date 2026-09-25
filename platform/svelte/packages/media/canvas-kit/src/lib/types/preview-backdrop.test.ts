// TDD — document-level preview backdrop (Slice A2.3 · #0298).
//
// CASE ENUMERATION
//   TC-PB-01 success  DEFAULT_PREVIEW_BACKDROP returns kind:'none' (transparent preview)
//   TC-PB-02 regress  returns a FRESH object each call (no shared singleton)
//   TC-PB-03 regress  a returned object can be mutated without affecting later calls

import { describe, expect, it } from 'vitest';
import { DEFAULT_PREVIEW_BACKDROP, type PreviewBackdrop } from './preview-backdrop.js';

describe('DEFAULT_PREVIEW_BACKDROP', () => {
	it('TC-PB-01 returns the transparent (none) default', () => {
		const expected: PreviewBackdrop = { kind: 'none' };
		expect(DEFAULT_PREVIEW_BACKDROP()).toEqual(expected);
	});

	it('TC-PB-02 returns a FRESH object each call (not a shared singleton)', () => {
		expect(DEFAULT_PREVIEW_BACKDROP()).not.toBe(DEFAULT_PREVIEW_BACKDROP());
	});

	it('TC-PB-03 mutating a returned object never leaks into a later call', () => {
		const first = DEFAULT_PREVIEW_BACKDROP();
		first.kind = 'color';
		first.color = '#123456';
		expect(DEFAULT_PREVIEW_BACKDROP().kind).toBe('none');
		expect(DEFAULT_PREVIEW_BACKDROP().color).toBeUndefined();
	});
});
