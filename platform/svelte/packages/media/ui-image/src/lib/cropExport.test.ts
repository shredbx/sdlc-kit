// TC-TS-2 + TC-TS-3 (RED) — the pure crop-geometry + output-format helpers behind
// SquareLogoCropper.svelte. The Svelte component is a thin canvas shell; the testable
// LOGIC lives here so vitest (node, no real canvas) can prove it. Traces task 2606-118:
//
//   computeSquareCrop(srcW, srcH, frame?) => { size }   — a SQUARE master, longest
//       edge clamped to LOGO_MAX_DIMENSION (512). SC5: 1600x900 non-square → square ≤512.
//       SC6: an already-square 512x512 → stays square ≤512, framing untouched.
//
//   pickLogoOutputFormat(sourceMime) => 'image/png' | 'image/webp' | SVG_PASSTHROUGH
//       — SC4 format law: PNG-in → PNG-out (alpha kept); SVG → sanitized vector
//       passthrough (NO crop, NO raster sizes); every other raster → WebP master.
//
// RED expectation: ./cropExport does not exist yet → this whole module fails to import.
import { describe, it, expect } from 'vitest';
import {
	computeSquareCrop,
	pickLogoOutputFormat,
	SVG_PASSTHROUGH,
	LOGO_MAX_DIMENSION
} from './cropExport';

describe('computeSquareCrop — square master, longest edge clamped to 512 (SC5/SC6)', () => {
	// SC5 — non-square source must yield a SQUARE region capped at 512.
	it('reduces a 1600x900 non-square source to a square no larger than 512', () => {
		const { size } = computeSquareCrop(1600, 900);
		expect(size).toBeLessThanOrEqual(LOGO_MAX_DIMENSION);
		expect(size).toBeGreaterThan(0);
	});

	// SC6 — an already-square source stays square, still ≤512.
	it('keeps an already-square 512x512 source square and within the cap', () => {
		const { size } = computeSquareCrop(512, 512);
		expect(size).toBeLessThanOrEqual(LOGO_MAX_DIMENSION);
		expect(size).toBeGreaterThan(0);
	});

	// EDGE — an oversized square is clamped down to exactly the cap.
	it('clamps a 2000x2000 source down to the 512 ceiling', () => {
		const { size } = computeSquareCrop(2000, 2000);
		expect(size).toBe(LOGO_MAX_DIMENSION);
	});

	// EDGE — a small source is not upscaled past its own bound (we never invent pixels).
	it('does not upscale a source smaller than the cap', () => {
		const { size } = computeSquareCrop(300, 300);
		expect(size).toBeLessThanOrEqual(300);
	});

	// The cap itself is the agreed logo policy value.
	it('exposes the logo master cap as 512', () => {
		expect(LOGO_MAX_DIMENSION).toBe(512);
	});
});

describe('pickLogoOutputFormat — SC4 format law (PNG/SVG passthrough else WebP)', () => {
	// PNG keeps PNG so transparency survives.
	it('keeps PNG for a PNG source (alpha preserved)', () => {
		expect(pickLogoOutputFormat('image/png')).toBe('image/png');
	});

	// SVG → sanitized vector passthrough sentinel (no crop, no raster sizes).
	it('returns the SVG passthrough sentinel for an SVG source', () => {
		expect(pickLogoOutputFormat('image/svg+xml')).toBe(SVG_PASSTHROUGH);
	});

	// Every other raster → WebP master.
	it('converts JPEG to WebP', () => {
		expect(pickLogoOutputFormat('image/jpeg')).toBe('image/webp');
	});

	it('normalises a WebP source to a WebP master', () => {
		expect(pickLogoOutputFormat('image/webp')).toBe('image/webp');
	});

	it('converts other raster formats (avif, heic) to WebP', () => {
		expect(pickLogoOutputFormat('image/avif')).toBe('image/webp');
		expect(pickLogoOutputFormat('image/heic')).toBe('image/webp');
	});

	// The SVG sentinel is distinct from any raster MIME — callers branch on it to
	// SKIP the crop/resize path entirely.
	it('uses a passthrough sentinel that is not a raster image MIME', () => {
		expect(SVG_PASSTHROUGH).not.toBe('image/png');
		expect(SVG_PASSTHROUGH).not.toBe('image/webp');
	});
});
