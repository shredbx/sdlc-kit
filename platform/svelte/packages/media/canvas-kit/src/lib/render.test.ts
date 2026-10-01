// G4 (task 2606-013) — tolerant empty-slot render. renderImageLayer draws the preloaded
// source when one exists; otherwise it falls back to the layer's placeholder `fill` so a
// linked-but-empty slot (a bound token whose index is out of range for the current record)
// reads as an intentional box, not a blank gap. The binding/link itself is unaffected —
// this is purely the render path.

import { describe, it, expect, vi } from 'vitest';
import { renderImageLayer, renderTextLayer, renderShapeLayer, renderLayer, renderPage, renderCropGhost, CROP_GHOST_ALPHA, IMAGE_FAILED, truncateToWidth, fitFontSize } from './render.js';
import { tilePositions } from './watermark-tiling.js';
import type { Layer } from './types/layer.js';
import type { MapConfig } from './types/layer.js';
import type { Page } from './types/document.js';

/** Minimal CanvasRenderingContext2D double that records the draw calls we assert on.
 *  Includes the vector-path surface the I-2 broken-image state draws with. */
function mockCtx() {
	return {
		fillStyle: '' as string | CanvasGradient | CanvasPattern,
		strokeStyle: '' as string | CanvasGradient | CanvasPattern,
		lineWidth: 0,
		save: vi.fn(),
		restore: vi.fn(),
		drawImage: vi.fn(),
		fillRect: vi.fn(),
		strokeRect: vi.fn(),
		setLineDash: vi.fn(),
		beginPath: vi.fn(),
		rect: vi.fn(),
		clip: vi.fn(),
		moveTo: vi.fn(),
		lineTo: vi.fn(),
		arc: vi.fn(),
		arcTo: vi.fn(),
		closePath: vi.fn(),
		translate: vi.fn(),
		rotate: vi.fn(),
		fill: vi.fn(),
		stroke: vi.fn(),
		fillText: vi.fn(),
		measureText: vi.fn(() => ({ width: 50 })),
		globalAlpha: 1,
		font: '',
		textAlign: 'left' as CanvasTextAlign,
		textBaseline: 'alphabetic' as CanvasTextBaseline,
		shadowColor: 'transparent',
		shadowBlur: 0,
		shadowOffsetX: 0,
		shadowOffsetY: 0
	};
}

const fakeImage = {} as unknown as CanvasImageSource;
/** ImageSource that "loads" only the one known URL. */
const loaded = (src: string) => (src === 'http://cdn/img.webp' ? fakeImage : undefined);

function imageLayer(extra: Partial<Layer>): Layer {
	return {
		id: 'l1',
		type: 'image',
		name: 'Image',
		visible: true,
		x: 10,
		y: 20,
		width: 100,
		height: 80,
		...extra
	} as Layer;
}

describe('renderImageLayer — G4 tolerant empty-slot', () => {
	it('draws the loaded source and does NOT also draw the placeholder fill', () => {
		const ctx = mockCtx();
		renderImageLayer(
			ctx as unknown as CanvasRenderingContext2D,
			imageLayer({ src: 'http://cdn/img.webp', fill: { color: '#D0D0C8', transparency: 0, pattern: 'solid' } }),
			loaded
		);
		expect(ctx.drawImage).toHaveBeenCalledWith(fakeImage, 10, 20, 100, 80);
		// A loaded image short-circuits BEFORE the fill block (proves the early return, not the absence of a fill).
		expect(ctx.fillRect).not.toHaveBeenCalled();
	});

	it('falls back to the placeholder fill when src is empty (out-of-range / unbound)', () => {
		const ctx = mockCtx();
		renderImageLayer(
			ctx as unknown as CanvasRenderingContext2D,
			imageLayer({ src: '', fill: { color: '#D0D0C8', transparency: 0, pattern: 'solid' } }),
			loaded
		);
		expect(ctx.drawImage).not.toHaveBeenCalled();
		expect(ctx.fillRect).toHaveBeenCalledWith(10, 20, 100, 80);
	});

	it('renders BLANK while src is set but STILL LOADING (ImageSource → undefined)', () => {
		const ctx = mockCtx();
		renderImageLayer(
			ctx as unknown as CanvasRenderingContext2D,
			imageLayer({ src: 'http://cdn/missing.webp', fill: { color: '#D0D0C8', transparency: 0, pattern: 'solid' } }),
			loaded
		);
		// An in-flight image must not be masked by the placeholder for its loading frame;
		// a FAILED load is a different, explicit signal (IMAGE_FAILED — I-2 describe below).
		expect(ctx.drawImage).not.toHaveBeenCalled();
		expect(ctx.fillRect).not.toHaveBeenCalled();
	});

	it('draws nothing when there is no image AND no fill', () => {
		const ctx = mockCtx();
		renderImageLayer(ctx as unknown as CanvasRenderingContext2D, imageLayer({ src: '' }), loaded);
		expect(ctx.drawImage).not.toHaveBeenCalled();
		expect(ctx.fillRect).not.toHaveBeenCalled();
	});

	it('draws nothing for a zero-area layer (even with a fill)', () => {
		const ctx = mockCtx();
		renderImageLayer(
			ctx as unknown as CanvasRenderingContext2D,
			imageLayer({ src: '', width: 0, fill: { color: '#D0D0C8', transparency: 0, pattern: 'solid' } }),
			loaded
		);
		expect(ctx.drawImage).not.toHaveBeenCalled();
		expect(ctx.fillRect).not.toHaveBeenCalled();
	});
});

describe('renderImageLayer — tint + glow (#0300 any-background legibility)', () => {
	it('applies a centred glow halo (offset-free) around the loaded image', () => {
		const ctx = mockCtx();
		let blurAtDraw = -1;
		let colorAtDraw = '';
		let offXAtDraw = -1;
		ctx.drawImage = vi.fn(() => {
			blurAtDraw = ctx.shadowBlur;
			colorAtDraw = ctx.shadowColor as string;
			offXAtDraw = ctx.shadowOffsetX;
		});
		renderImageLayer(
			ctx as unknown as CanvasRenderingContext2D,
			imageLayer({ src: 'http://cdn/img.webp', glow: { intensity: 6, color: '#000000' } }),
			loaded
		);
		expect(blurAtDraw).toBe(18); // intensity × 3
		expect(colorAtDraw).toBe('#000000');
		expect(offXAtDraw).toBe(0); // a glow, never an offset box shadow (2026-06-07)
	});

	it('leaves the glow unset for a plain image (no shadow leak at draw)', () => {
		const ctx = mockCtx();
		let blurAtDraw = -1;
		ctx.drawImage = vi.fn(() => {
			blurAtDraw = ctx.shadowBlur;
		});
		renderImageLayer(
			ctx as unknown as CanvasRenderingContext2D,
			imageLayer({ src: 'http://cdn/img.webp' }),
			loaded
		);
		expect(blurAtDraw).toBe(0);
	});

	it('draws the untinted source when no offscreen buffer is usable (headless) — never throws', () => {
		const ctx = mockCtx();
		// Headless: the bare test-double source has no natural size, so tintedSource short-
		// circuits to the original — the draw still happens (the recolor is verified live in
		// the browser, where OffscreenCanvas composites the silhouette).
		expect(() =>
			renderImageLayer(
				ctx as unknown as CanvasRenderingContext2D,
				imageLayer({ src: 'http://cdn/img.webp', tint: { color: '#e5392b', strength: 1 } }),
				loaded
			)
		).not.toThrow();
		expect(ctx.drawImage).toHaveBeenCalledWith(fakeImage, 10, 20, 100, 80);
	});
});

describe('renderImageLayer — I-2 broken-image state', () => {
	/** ImageSource reporting a FAILED load for the known URL. */
	const failed = (src: string) => (src === 'http://cdn/img.webp' ? IMAGE_FAILED : undefined);

	it('renders the broken state (field + dashed border + glyph), never drawImage', () => {
		const ctx = mockCtx();
		renderImageLayer(ctx as unknown as CanvasRenderingContext2D, imageLayer({ src: 'http://cdn/img.webp' }), failed);
		expect(ctx.drawImage).not.toHaveBeenCalled();
		expect(ctx.clip).toHaveBeenCalled(); // glyph/slash can never scribble outside the layer box
		expect(ctx.rect).toHaveBeenCalledWith(10, 20, 100, 80); // …and the clip IS the layer box
		expect(ctx.fillRect).toHaveBeenCalledWith(10, 20, 100, 80); // muted field over the layer box
		expect(ctx.strokeRect).toHaveBeenCalled(); // dashed border + glyph frame
		expect(ctx.stroke).toHaveBeenCalled(); // mountain + slash paths
	});

	it('failed state wins over a placeholder fill (no silent placeholder masking)', () => {
		const ctx = mockCtx();
		renderImageLayer(
			ctx as unknown as CanvasRenderingContext2D,
			imageLayer({ src: 'http://cdn/img.webp', fill: { color: '#D0D0C8', transparency: 0, pattern: 'solid' } }),
			failed
		);
		// One fillRect = the broken-state field; the placeholder-fill path must NOT also run.
		expect(ctx.fillRect).toHaveBeenCalledTimes(1);
		expect(ctx.drawImage).not.toHaveBeenCalled();
	});

	it('still-loading (undefined) stays blank — failure is an explicit signal, not a default', () => {
		const ctx = mockCtx();
		renderImageLayer(ctx as unknown as CanvasRenderingContext2D, imageLayer({ src: 'http://cdn/other.webp' }), failed);
		expect(ctx.drawImage).not.toHaveBeenCalled();
		expect(ctx.fillRect).not.toHaveBeenCalled();
		expect(ctx.strokeRect).not.toHaveBeenCalled();
	});
});

// Text box (background + border + corner radius) and central rotation —
// user directive 2026-06-07 (design pass A follow-up).

function textLayer(extra: Partial<Layer>): Layer {
	return {
		id: 't1',
		type: 'text',
		name: 'Text',
		visible: true,
		x: 10,
		y: 20,
		width: 100,
		height: 40,
		content: 'Hello',
		...extra
	} as Layer;
}

describe('renderTextLayer — text box (bg + border + radius)', () => {
	it('TC-RD-10 fills a rounded background when fill + corner_radius are set', () => {
		const ctx = mockCtx();
		renderTextLayer(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ fill: { color: '#C8A851', transparency: 0 }, corner_radius: 8 })
		);
		expect(ctx.beginPath).toHaveBeenCalled();
		expect(ctx.arcTo).toHaveBeenCalledTimes(4); // rounded path, not a plain rect
		expect(ctx.fill).toHaveBeenCalled();
	});

	it('TC-RD-11 strokes a border from StrokeStyle (color + thickness + dash)', () => {
		const ctx = mockCtx();
		renderTextLayer(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ stroke: { color: '#0D4F4F', thickness: 2, dash_type: 'dashed', glow_intensity: 0 } })
		);
		expect(ctx.strokeStyle).toBe('#0D4F4F');
		expect(ctx.lineWidth).toBe(2);
		expect(ctx.setLineDash).toHaveBeenCalledWith([8, 4]);
		expect(ctx.stroke).toHaveBeenCalled();
	});

	it('TC-RD-12 zero-thickness border draws nothing; radius 0 traces a plain rect', () => {
		const ctx = mockCtx();
		renderTextLayer(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ fill: { color: '#fff', transparency: 0 }, stroke: { color: '#000', thickness: 0, dash_type: 'solid', glow_intensity: 0 } })
		);
		expect(ctx.stroke).not.toHaveBeenCalled();
		expect(ctx.rect).toHaveBeenCalledWith(10, 20, 100, 40);
		expect(ctx.arcTo).not.toHaveBeenCalled();
	});

	it('TC-RD-13 radius clamps to half the short side (no self-crossing path)', () => {
		const ctx = mockCtx();
		renderTextLayer(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ fill: { color: '#fff', transparency: 0 }, corner_radius: 999 })
		);
		// First arcTo call's radius argument = clamped 20 (h/2 of 40), never 999.
		expect(ctx.arcTo.mock.calls[0][4]).toBe(20);
	});
});

describe('renderShapeLayer — corner radius', () => {
	it('TC-RD-14 rectangle honours corner_radius via the shared rounded path', () => {
		const ctx = mockCtx();
		renderShapeLayer(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ type: 'shape', shape_type: 'rectangle', content: undefined, fill: { color: '#333', transparency: 0 }, corner_radius: 6 })
		);
		expect(ctx.arcTo).toHaveBeenCalledTimes(4);
		expect(ctx.fill).toHaveBeenCalled();
	});
});

describe('renderLayer — central rotation', () => {
	it('TC-RD-15 rotates around the layer centre (translate → rotate → translate back)', () => {
		const ctx = mockCtx();
		renderLayer(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ rotation: 90 })
		);
		// centre of x10 y20 w100 h40 = (60, 40)
		expect(ctx.translate).toHaveBeenNthCalledWith(1, 60, 40);
		expect(ctx.rotate).toHaveBeenCalledWith(Math.PI / 2);
		expect(ctx.translate).toHaveBeenNthCalledWith(2, -60, -40);
	});

	it('TC-RD-16 rotation 0 (or unset) applies no transform', () => {
		const ctx = mockCtx();
		renderLayer(ctx as unknown as CanvasRenderingContext2D, textLayer({}));
		expect(ctx.rotate).not.toHaveBeenCalled();
		expect(ctx.translate).not.toHaveBeenCalled();
	});

	it('TC-RD-17 rotation applies to image layers too', () => {
		const ctx = mockCtx();
		renderLayer(
			ctx as unknown as CanvasRenderingContext2D,
			imageLayer({ rotation: 45, src: 'http://cdn/img.webp' }),
			loaded
		);
		expect(ctx.rotate).toHaveBeenCalledWith(Math.PI / 4);
		expect(ctx.drawImage).toHaveBeenCalled();
	});
});

// Shadow + glow effects (user directive 2026-06-07, ported from land-canvas) —
// box shadow on fills, border glow passes, glyph glow/shadow, image border.

describe('effects — box shadow, border glow, text glow/shadow, image border', () => {
	it('TC-RD-18 box shadow rides the FILL and is reset before the stroke', () => {
		const ctx = mockCtx();
		let blurAtFill = -1;
		let blurAtStroke = -1;
		ctx.fill = vi.fn(() => { blurAtFill = ctx.shadowBlur; });
		ctx.stroke = vi.fn(() => { blurAtStroke = ctx.shadowBlur; });
		renderTextLayer(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({
				fill: { color: '#ffffff', transparency: 0 },
				stroke: { color: '#000000', thickness: 1, dash_type: 'solid', glow_intensity: 0 },
				shadow: { color: '#000000', blur: 8, offset_x: 0, offset_y: 4, opacity: 0.4 }
			})
		);
		expect(blurAtFill).toBe(8);
		expect(blurAtStroke).toBe(0); // resetShadow ran before the border
		expect(ctx.shadowOffsetY).toBe(0); // and offsets are reset after
	});

	it('TC-RD-19 border glow strokes 3 widening passes + the final solid stroke', () => {
		const ctx = mockCtx();
		renderTextLayer(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ stroke: { color: '#0D4F4F', thickness: 2, dash_type: 'solid', glow_intensity: 5 } })
		);
		expect(ctx.stroke).toHaveBeenCalledTimes(4);
		expect(ctx.strokeStyle).toBe('#0D4F4F'); // final pass = the solid stroke color
		expect(ctx.lineWidth).toBe(2);
	});

	it('TC-RD-20 text glow sets a centred halo (color, blur = intensity×3) for the glyphs', () => {
		const ctx = mockCtx();
		let colorAtText = '';
		let blurAtText = -1;
		ctx.fillText = vi.fn(() => { colorAtText = String(ctx.shadowColor); blurAtText = ctx.shadowBlur; });
		renderTextLayer(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ text_glow: { intensity: 5, color: '#ffffff' } })
		);
		expect(colorAtText).toBe('#ffffff');
		expect(blurAtText).toBe(15);
		expect(ctx.shadowOffsetX).toBe(0);
	});

	it('TC-RD-21 text shadow (no glow) sets the offset blur at the given opacity', () => {
		const ctx = mockCtx();
		let colorAtText = '';
		ctx.fillText = vi.fn(() => { colorAtText = String(ctx.shadowColor); });
		renderTextLayer(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ text_shadow: { color: '#000000', blur: 6, offset_x: 2, offset_y: 3, opacity: 0.5 } })
		);
		expect(colorAtText).toBe('rgba(0,0,0,0.5)');
		expect(ctx.shadowOffsetX).toBe(2);
		expect(ctx.shadowOffsetY).toBe(3);
	});

	it('TC-RD-22 image layers stroke a border (glow-capable) around the drawn image', () => {
		const ctx = mockCtx();
		renderImageLayer(
			ctx as unknown as CanvasRenderingContext2D,
			imageLayer({ src: 'http://cdn/img.webp', stroke: { color: '#C8A851', thickness: 4, dash_type: 'solid', glow_intensity: 0 } }),
			loaded
		);
		expect(ctx.drawImage).toHaveBeenCalled();
		expect(ctx.stroke).toHaveBeenCalledTimes(1);
		expect(ctx.strokeStyle).toBe('#C8A851');
	});

	it('TC-RD-23 image layers NEVER take a box shadow (user call) — drawImage runs shadow-free', () => {
		const ctx = mockCtx();
		let blurAtDraw = -1;
		ctx.drawImage = vi.fn(() => { blurAtDraw = ctx.shadowBlur; });
		renderImageLayer(
			ctx as unknown as CanvasRenderingContext2D,
			imageLayer({ src: 'http://cdn/img.webp', shadow: { color: '#000', blur: 9, offset_x: 0, offset_y: 4, opacity: 0.4 } }),
			loaded
		);
		expect(blurAtDraw).toBe(0);
	});
});

describe('renderLayer — per-edge inset crop (2026-06-07)', () => {
	it('TC-RD-24 clips to box-minus-insets before drawing', () => {
		const ctx = mockCtx();
		renderLayer(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ crop_top: 5, crop_right: 10, crop_bottom: 5, crop_left: 20 })
		);
		// box x10 y20 w100 h40 → clip rect x30 y25 w70 h30
		expect(ctx.rect).toHaveBeenCalledWith(30, 25, 70, 30);
		expect(ctx.clip).toHaveBeenCalledTimes(1);
		expect(ctx.fillText).toHaveBeenCalled(); // layer still draws (clipped)
	});

	it('TC-RD-25 no crop fields → no clip', () => {
		const ctx = mockCtx();
		renderLayer(ctx as unknown as CanvasRenderingContext2D, textLayer({}));
		expect(ctx.clip).not.toHaveBeenCalled();
	});

	it('TC-RD-26 crop composes with rotation — transform first, clip second', () => {
		const ctx = mockCtx();
		const order: string[] = [];
		ctx.rotate = vi.fn(() => order.push('rotate'));
		ctx.clip = vi.fn(() => order.push('clip'));
		renderLayer(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ rotation: 30, crop_left: 10 })
		);
		expect(order).toEqual(['rotate', 'clip']); // crop window rotates WITH the layer
	});

	it('TC-RD-27 over-crop clamps to a zero-area window (never negative rects)', () => {
		const ctx = mockCtx();
		renderLayer(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ crop_left: 80, crop_right: 80 })
		);
		expect(ctx.rect).toHaveBeenCalledWith(90, 20, 0, 40);
	});
});

describe('renderCropGhost — dimmed cropped-away strips (scissor gesture, 2026-06-07)', () => {
	it('TC-RD-28 no crop → no-op (nothing saved or drawn)', () => {
		const ctx = mockCtx();
		renderCropGhost(ctx as unknown as CanvasRenderingContext2D, textLayer({}));
		expect(ctx.save).not.toHaveBeenCalled();
		expect(ctx.fillText).not.toHaveBeenCalled();
	});

	it('TC-RD-29 inverse clip: full box + kept window with evenodd, content dimmed by ghost alpha', () => {
		const ctx = mockCtx();
		renderCropGhost(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ crop_top: 5, crop_right: 10, crop_bottom: 5, crop_left: 20, opacity: 0.5 })
		);
		expect(ctx.rect).toHaveBeenCalledWith(10, 20, 100, 40); // outer = the layer box
		expect(ctx.rect).toHaveBeenCalledWith(30, 25, 70, 30); // inner = the kept window
		expect(ctx.clip).toHaveBeenCalledWith('evenodd'); // strips only — kept region never double-painted
		expect(ctx.fillText).toHaveBeenCalled(); // ghost content draws (clipped to the strips)
		expect(ctx.globalAlpha).toBeCloseTo(0.5 * CROP_GHOST_ALPHA, 5);
	});

	it('TC-RD-30 ghost rotates with the layer — transform first, clip second', () => {
		const ctx = mockCtx();
		const order: string[] = [];
		ctx.rotate = vi.fn(() => order.push('rotate'));
		ctx.clip = vi.fn(() => order.push('clip'));
		renderCropGhost(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ rotation: 30, crop_left: 10 })
		);
		expect(order).toEqual(['rotate', 'clip']);
	});

	it('TC-RD-31 hidden layer → no ghost', () => {
		const ctx = mockCtx();
		renderCropGhost(
			ctx as unknown as CanvasRenderingContext2D,
			textLayer({ visible: false, crop_top: 10 })
		);
		expect(ctx.save).not.toHaveBeenCalled();
	});
});

// --- S-FIT text fitting (task 2606-003) -------------------------------------
describe('truncateToWidth', () => {
	const len = (s: string) => s.length; // 1 unit per glyph

	it('returns the whole text when it already fits', () => {
		expect(truncateToWidth('Hello', 10, len)).toBe('Hello');
	});
	it('ellipsises the longest fitting prefix', () => {
		// '…' counts as 1; the widest prefix+… within 6 units is "Hello…"
		expect(truncateToWidth('HelloWorld', 6, len)).toBe('Hello…');
	});
	it('returns just … when not even one glyph fits, and "" for a zero box', () => {
		expect(truncateToWidth('AB', 1, len)).toBe('…');
		expect(truncateToWidth('AB', 0, len)).toBe('');
	});
});

describe('fitFontSize', () => {
	it('grows toward the box (cap) when everything fits', () => {
		expect(fitFontSize(20, () => true)).toBeGreaterThan(20);
	});
	it('shrinks to the largest size that fits', () => {
		const size = fitFontSize(20, (s) => s <= 12); // base 20, floor 10
		expect(size).toBeGreaterThan(11);
		expect(size).toBeLessThanOrEqual(12.01);
	});
	it('returns the 50% floor when even it overflows', () => {
		expect(fitFontSize(20, () => false)).toBe(10);
	});
});

describe('renderTextLayer — S-FIT modes', () => {
	// A ctx whose measureText scales with the current font px (parsed from ctx.font),
	// so the fit/truncate logic is actually exercised.
	function sizeCtx() {
		const ctx = mockCtx() as ReturnType<typeof mockCtx> & { _font: string };
		let cur = 16;
		Object.defineProperty(ctx, 'font', {
			get: () => ctx._font,
			set: (v: string) => {
				ctx._font = v;
				const m = /(\d+(?:\.\d+)?)px/.exec(v);
				if (m) cur = parseFloat(m[1]);
			}
		});
		ctx.measureText = vi.fn((s: string) => ({ width: s.length * cur * 0.6 }) as TextMetrics);
		return ctx;
	}
	const lastDrawn = (ctx: ReturnType<typeof sizeCtx>): string =>
		(ctx.fillText.mock.calls.at(-1)?.[0] as string) ?? '';
	const long = 'A'.repeat(60);

	it('none (default) draws the full content — no ellipsis, no clip', () => {
		const ctx = sizeCtx();
		renderTextLayer(ctx as unknown as CanvasRenderingContext2D, textLayer({ content: long, width: 100, height: 40 }));
		expect(lastDrawn(ctx)).toBe(long);
		expect(ctx.clip).not.toHaveBeenCalled();
	});

	it('truncate → single line ending in … that fits the box', () => {
		const ctx = sizeCtx();
		renderTextLayer(ctx as unknown as CanvasRenderingContext2D, textLayer({ content: long, width: 100, height: 40, text_fit: 'truncate' }));
		const drawn = lastDrawn(ctx);
		expect(drawn.endsWith('…')).toBe(true);
		expect(drawn.length).toBeLessThan(long.length);
	});

	it('crop → clips to the box and draws the natural (un-ellipsised) text', () => {
		const ctx = sizeCtx();
		renderTextLayer(ctx as unknown as CanvasRenderingContext2D, textLayer({ content: long, width: 100, height: 40, text_fit: 'crop' }));
		expect(ctx.clip).toHaveBeenCalled();
		expect(lastDrawn(ctx)).toBe(long); // never ellipsised
	});

	it('fit → shrinks the font (and ellipsises at the 50% floor if still overflowing)', () => {
		const ctx = sizeCtx();
		renderTextLayer(ctx as unknown as CanvasRenderingContext2D, textLayer({ content: long, width: 100, height: 40, font_size: 32, text_fit: 'fit' }));
		// The font finally set is smaller than the 32px base.
		const px = parseFloat(/(\d+(?:\.\d+)?)px/.exec(ctx._font)?.[1] ?? '0');
		expect(px).toBeLessThan(32);
		// 60 'A's can't fit even at the 16px floor → ellipsised.
		expect(lastDrawn(ctx).endsWith('…')).toBe(true);
	});

	it('fit → SHRINK-ONLY: a short string in a roomy box keeps the set size (never grows)', () => {
		const ctx = sizeCtx();
		renderTextLayer(ctx as unknown as CanvasRenderingContext2D, textLayer({ content: 'Hi', width: 400, height: 200, font_size: 16, text_fit: 'fit' }));
		const px = parseFloat(/(\d+(?:\.\d+)?)px/.exec(ctx._font)?.[1] ?? '0');
		expect(px).toBeLessThanOrEqual(16); // capped at the set size — 'fit' never enlarges
	});

	it('fill → grows the font ABOVE the set size to fill a roomy box', () => {
		const ctx = sizeCtx();
		renderTextLayer(ctx as unknown as CanvasRenderingContext2D, textLayer({ content: 'Hi', width: 400, height: 200, font_size: 16, text_fit: 'fill' }));
		const px = parseFloat(/(\d+(?:\.\d+)?)px/.exec(ctx._font)?.[1] ?? '0');
		expect(px).toBeGreaterThan(16); // 'fill' enlarges to fill the empty box
	});
});

// ── Map layer render dispatch (Decision #0297) ─────────────────────────────

const MAP_CONFIG: MapConfig = {
	center: { lat: 13.7563, lng: 100.5018 },
	zoom: 14,
	mapType: 'roadmap'
};

function mapLayer(extra: Partial<Layer> = {}): Layer {
	return {
		id: 'map-1',
		type: 'map',
		name: 'Map',
		visible: true,
		locked: false,
		x: 0, y: 0,
		width: 640,
		height: 400,
		src: 'http://cdn/static-map.png',
		map_config: MAP_CONFIG,
		opacity: 1,
		...extra
	} as Layer;
}

function markerLayer(extra: Partial<Layer> = {}): Layer {
	return {
		id: 'marker-1',
		type: 'marker',
		name: 'Marker',
		visible: true,
		locked: false,
		x: 10, y: 10,
		width: 32, height: 40,
		map_surface_id: 'map-1',
		geo_ref: { lat: 13.75, lng: 100.5, anchor_lats: [], anchor_lngs: [] },
		opacity: 1,
		...extra
	} as Layer;
}

function outlineLayer(extra: Partial<Layer> = {}): Layer {
	return {
		id: 'outline-1',
		type: 'outline',
		name: 'Region',
		visible: true,
		locked: false,
		x: 0, y: 0,
		width: 200, height: 200,
		map_surface_id: 'map-1',
		coord_system: 'map' as const,
		closed: true,
		anchors: [],
		opacity: 1,
		...extra
	} as Layer;
}

function calloutMapLayer(extra: Partial<Layer> = {}): Layer {
	return {
		id: 'callout-1',
		type: 'callout',
		name: 'Callout',
		visible: true,
		locked: false,
		x: 0, y: 0,
		width: 160, height: 40,
		content: 'Hello',
		map_surface_id: 'map-1',
		geo_ref: { lat: 13.7563, lng: 100.5018, anchor_lats: [], anchor_lngs: [] },
		map_config: MAP_CONFIG,
		opacity: 1,
		...extra
	} as Layer;
}

describe('renderLayer — map layer dispatch (TC-MAP)', () => {
	const mapSrc = 'http://cdn/static-map.png';
	const fakeMapImg = {} as unknown as CanvasImageSource;
	const mapImages = (src: string) => (src === mapSrc ? fakeMapImg : undefined);

	it('TC-MAP-01 map layer → renderImageLayer (drawImage called with the loaded source)', () => {
		const ctx = mockCtx();
		renderLayer(ctx as unknown as CanvasRenderingContext2D, mapLayer(), mapImages);
		expect(ctx.drawImage).toHaveBeenCalledWith(fakeMapImg, 0, 0, 640, 400);
	});

	it('TC-MAP-02 anchored marker (map_surface_id set) → no-op (never renderShapeLayer)', () => {
		const ctx = mockCtx();
		renderLayer(ctx as unknown as CanvasRenderingContext2D, markerLayer());
		// A no-op layer: drawImage, fillRect, fill, stroke are all absent.
		expect(ctx.drawImage).not.toHaveBeenCalled();
		expect(ctx.fill).not.toHaveBeenCalled();
		expect(ctx.stroke).not.toHaveBeenCalled();
		expect(ctx.fillRect).not.toHaveBeenCalled();
	});

	it('TC-MAP-03 anchored outline (map_surface_id set) → no-op', () => {
		const ctx = mockCtx();
		renderLayer(ctx as unknown as CanvasRenderingContext2D, outlineLayer());
		expect(ctx.fill).not.toHaveBeenCalled();
		expect(ctx.stroke).not.toHaveBeenCalled();
		expect(ctx.drawImage).not.toHaveBeenCalled();
	});

	it('TC-MAP-04 free-floating marker (no map_surface_id) → renderShapeLayer (fill or stroke)', () => {
		const ctx = mockCtx();
		// A marker with fill and no map_surface_id goes to renderShapeLayer.
		renderLayer(
			ctx as unknown as CanvasRenderingContext2D,
			markerLayer({
				map_surface_id: undefined,
				fill: { color: '#e5392b', transparency: 0, pattern: 'solid' }
			})
		);
		// renderShapeLayer calls fill() for the fill rect.
		expect(ctx.fill).toHaveBeenCalled();
	});

	it('TC-MAP-05 anchored callout → renders as text at its (already-resolved) x/y', () => {
		// render.ts does NOT project — resolveFrame projects geo_ref → x/y using the
		// PARENT map size (see export.test.ts TC-EX-CALLOUT-*). render just draws text.
		const ctx = mockCtx();
		renderLayer(ctx as unknown as CanvasRenderingContext2D, calloutMapLayer({ x: 250, y: 120 }));
		// Text drawn; NO drawImage/fill/stroke (it's a plain text layer, not projected/baked here).
		expect(ctx.fillText).toHaveBeenCalled();
		expect(ctx.drawImage).not.toHaveBeenCalled();
	});

	it('TC-MAP-06 non-anchored callout (no map_surface_id) → normal text render at x/y', () => {
		const ctx = mockCtx();
		renderLayer(
			ctx as unknown as CanvasRenderingContext2D,
			calloutMapLayer({ map_surface_id: undefined })
		);
		expect(ctx.fillText).toHaveBeenCalled();
	});
});

// [WM] LIVE TILING RENDER (task 2606-001 · W-C). Proves the artboard renderer actually
// PAINTS one source body per tile origin — the WYSIWYG repeat, not a preview-only flatten
// (the Strike #1 trap). renderLayer with a tile + region must drawImage once per
// tilePositions() origin; renderPage must compute that region itself (tileFillRegion) for
// any layer carrying `tile`. A non-tiled layer still draws exactly once (no regression).
describe('renderLayer / renderPage — live tiling (W-C)', () => {
	const TILED = { repeatH: true, repeatV: false, gapX: 50, gapY: 0 } as const;
	const REGION = { minX: 0, minY: 0, maxX: 400, maxY: 400 };
	const tileSrc = (extra: Partial<Layer> = {}) =>
		imageLayer({ src: 'http://cdn/img.webp', x: 100, y: 100, width: 50, height: 50, ...extra });

	it('a non-tiled image draws its body exactly ONCE', () => {
		const ctx = mockCtx();
		renderLayer(ctx as unknown as CanvasRenderingContext2D, tileSrc(), loaded);
		expect(ctx.drawImage).toHaveBeenCalledTimes(1);
	});

	it('a tiled image draws its body once per tilePositions() origin (live repeats)', () => {
		const layer = tileSrc({ tile: { ...TILED } });
		const expected = tilePositions({ x: 100, y: 100, w: 50, h: 50 }, layer.tile!, REGION).length;
		expect(expected).toBeGreaterThan(1); // sanity: the region yields multiple copies
		const ctx = mockCtx();
		renderLayer(ctx as unknown as CanvasRenderingContext2D, layer, loaded, REGION);
		expect(ctx.drawImage).toHaveBeenCalledTimes(expected);
	});

	it('a tile set but NO region falls back to a single placement (renderLayer guard)', () => {
		const ctx = mockCtx();
		renderLayer(ctx as unknown as CanvasRenderingContext2D, tileSrc({ tile: { ...TILED } }), loaded);
		expect(ctx.drawImage).toHaveBeenCalledTimes(1);
	});

	it('renderPage computes the fill region itself and tiles a layer carrying `tile`', () => {
		const page = {
			id: 'p1',
			name: 'Page',
			width: 400,
			height: 400,
			layers: [tileSrc({ tile: { repeatH: true, repeatV: true, gapX: 50, gapY: 50 } })]
		} as unknown as Page;
		const ctx = mockCtx();
		renderPage(ctx as unknown as CanvasRenderingContext2D, page, undefined, loaded);
		expect(ctx.drawImage.mock.calls.length).toBeGreaterThan(1);
	});
});
