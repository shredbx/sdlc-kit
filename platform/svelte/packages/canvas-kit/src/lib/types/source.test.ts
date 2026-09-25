// TDD RED — @sbx/canvas-kit role-based image variants (FDD4.PD.TEST_RED, task 2606-001).
// pickImageVariant picks the right URL for a render role from a ResolvedImage,
// with a GRACEFUL fallback chain so a provider that offers no role variants (maps,
// branding) or a legacy document still resolves. Every test MUST fail until
// pickImageVariant is implemented (GREEN).
//
// CASE ENUMERATION (scenarios SC-grid-thumbnail / SC-artboard-medium /
//                   SC-export-original / SC-fallback-backcompat)
//   TC-PV-01 success  thumbnail role + full image → thumbnail variant
//   TC-PV-02 success  medium role + full image    → medium variant
//   TC-PV-03 success  original role + full image   → original variant
//   TC-PV-04 edge     thumbnail role, no thumbnail → legacy thumb (back-compat alias)
//   TC-PV-05 edge     thumbnail role, no thumbnail/thumb → medium (coarser-but-present)
//   TC-PV-06 edge     medium role, no medium       → url
//   TC-PV-07 edge     original role, no original   → url (back-compat)
//   TC-PV-08 edge     url-only image (maps/branding/legacy) → every role returns url

import { describe, expect, it } from 'vitest';
import { pickImageVariant } from './source.js';
import type { ResolvedImage } from './source.js';

describe('pickImageVariant — role-based image variant selection', () => {
	const full: ResolvedImage = {
		url: 'u/url',
		thumb: 'u/legacy-thumb',
		thumbnail: 'u/thumbnail',
		medium: 'u/medium',
		original: 'u/original'
	};

	it('TC-PV-01 thumbnail role → thumbnail variant', () => {
		expect(pickImageVariant(full, 'thumbnail')).toBe('u/thumbnail');
	});

	it('TC-PV-02 medium role → medium variant', () => {
		expect(pickImageVariant(full, 'medium')).toBe('u/medium');
	});

	it('TC-PV-03 original role → original variant', () => {
		expect(pickImageVariant(full, 'original')).toBe('u/original');
	});

	it('TC-PV-04 thumbnail falls back to legacy thumb when thumbnail absent', () => {
		const img: ResolvedImage = { url: 'u/url', thumb: 'u/legacy-thumb' };
		expect(pickImageVariant(img, 'thumbnail')).toBe('u/legacy-thumb');
	});

	it('TC-PV-05 thumbnail falls back to medium when thumbnail+thumb absent', () => {
		const img: ResolvedImage = { url: 'u/url', medium: 'u/medium' };
		expect(pickImageVariant(img, 'thumbnail')).toBe('u/medium');
	});

	it('TC-PV-06 medium falls back to url when medium absent', () => {
		const img: ResolvedImage = { url: 'u/url' };
		expect(pickImageVariant(img, 'medium')).toBe('u/url');
	});

	it('TC-PV-07 original falls back to url when original absent (back-compat)', () => {
		const img: ResolvedImage = { url: 'u/url' };
		expect(pickImageVariant(img, 'original')).toBe('u/url');
	});

	it('TC-PV-08 url-only image (maps/branding/legacy) → every role returns url', () => {
		const img: ResolvedImage = { url: 'u/only' };
		expect(pickImageVariant(img, 'thumbnail')).toBe('u/only');
		expect(pickImageVariant(img, 'medium')).toBe('u/only');
		expect(pickImageVariant(img, 'original')).toBe('u/only');
	});
});
