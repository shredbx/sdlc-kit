// Canvas 2D renderer — Media Canvas (slice 1). TRIMMED + PORTED from
// clients/andrei/projects/land-canvas/apps/web/svelte/src/lib/utils/render.ts.
// This is the SEAM the export Rasterizer reuses later (export.ts → Rasterizer).
//
// Scope for slice 1: text (font/size/weight/color/align/wrap), shape rectangle
// (fill + stroke), and image (drawImage from a preloaded source). Land-specific
// rendering (outlines, edge labels, markers, glow, patterns, geo) is intentionally
// NOT ported — the Media Canvas UI never surfaces those layer kinds (design §5).
//
// HEADLESS: only the Canvas 2D API is used (allowed in the kit per index.ts header).
// MUST NOT import any Svelte runtime. Pure draw functions; the caller owns the ctx
// lifecycle, clearing, and DPR scaling.

import type { Layer, TileEffect } from './types/layer.js';
import type { ImageTint } from './types/style.js';
import type { Page } from './types/document.js';
import { tilePositions } from './watermark-tiling.js';

/** Convert `#rrggbb` to an `rgba()` string at the given alpha (0-1). */
function hexToRgba(hex: string, alpha: number): string {
	if (!hex || hex.length < 7) return `rgba(0,0,0,${alpha})`;
	const r = parseInt(hex.slice(1, 3), 16);
	const g = parseInt(hex.slice(3, 5), 16);
	const b = parseInt(hex.slice(5, 7), 16);
	return `rgba(${r},${g},${b},${alpha})`;
}

/** Lighten `#rrggbb` toward white by `amount` (0-1) — glow halo color (ported
 *  from land-canvas render.ts). */
function lightenHex(hex: string, amount: number): string {
	if (!hex || hex.length < 7) return hex;
	const ch = (off: number) => {
		const v = parseInt(hex.slice(off, off + 2), 16);
		return Math.min(255, Math.round(v + (255 - v) * amount));
	};
	return `#${ch(1).toString(16).padStart(2, '0')}${ch(3).toString(16).padStart(2, '0')}${ch(5).toString(16).padStart(2, '0')}`;
}

/** Set the ctx drop-shadow from a layer's box ShadowStyle (fill-time only — the
 *  land-canvas semantics: the shadow falls from the filled face, never doubled
 *  by the stroke). Pair with resetShadow() before any stroking. */
function applyBoxShadow(ctx: CanvasRenderingContext2D, layer: Layer): void {
	if (!layer.shadow) return;
	ctx.shadowColor = hexToRgba(layer.shadow.color, layer.shadow.opacity);
	ctx.shadowBlur = layer.shadow.blur;
	ctx.shadowOffsetX = layer.shadow.offset_x;
	ctx.shadowOffsetY = layer.shadow.offset_y;
}

function resetShadow(ctx: CanvasRenderingContext2D): void {
	ctx.shadowColor = 'transparent';
	ctx.shadowBlur = 0;
	ctx.shadowOffsetX = 0;
	ctx.shadowOffsetY = 0;
}

/** Set the ctx shadow as a GLYPH effect before text painting — glow wins over
 *  shadow (ported from land-canvas applyTextEffects): glow = centred halo in the
 *  glow color (blur = intensity×3); shadow = offset blur at the given opacity. */
function applyTextEffects(ctx: CanvasRenderingContext2D, layer: Layer): void {
	if (layer.text_glow?.intensity) {
		ctx.shadowColor = layer.text_glow.color;
		ctx.shadowBlur = layer.text_glow.intensity * 3;
		ctx.shadowOffsetX = 0;
		ctx.shadowOffsetY = 0;
	} else if (layer.text_shadow) {
		const ts = layer.text_shadow;
		ctx.shadowColor = hexToRgba(ts.color, ts.opacity);
		ctx.shadowBlur = ts.blur;
		ctx.shadowOffsetX = ts.offset_x;
		ctx.shadowOffsetY = ts.offset_y;
	}
}

/**
 * Word-wrap `text` so each line fits within `maxWidth` for the ctx's current font.
 * Returns the lines (never empty — a single empty string when there is no text).
 */
export function wrapText(ctx: CanvasRenderingContext2D, text: string, maxWidth: number): string[] {
	if (maxWidth <= 0) return [text];
	const words = text.split(/\s+/);
	const lines: string[] = [];
	let current = '';
	for (const word of words) {
		const test = current ? `${current} ${word}` : word;
		if (ctx.measureText(test).width > maxWidth && current) {
			lines.push(current);
			current = word;
		} else {
			current = test;
		}
	}
	if (current) lines.push(current);
	return lines.length > 0 ? lines : [''];
}

/** Offscreen tint buffer — feature-detected so the kit stays headless-pure: an
 *  OffscreenCanvas (workers + modern browsers) or a detached <canvas>, else null (the
 *  node test path → caller draws the untinted source, never throws). */
function makeTintBuffer(w: number, h: number): OffscreenCanvas | HTMLCanvasElement | null {
	if (typeof OffscreenCanvas !== 'undefined') return new OffscreenCanvas(w, h);
	if (typeof document !== 'undefined') {
		const c = document.createElement('canvas');
		c.width = w;
		c.height = h;
		return c;
	}
	return null;
}

/** Natural pixel size of a drawable source (image / bitmap / canvas / video). 0 when
 *  unknown (a bare test double) → tint is skipped, the original source is drawn. */
function sourceSize(source: CanvasImageSource): { w: number; h: number } {
	const s = source as {
		naturalWidth?: number;
		naturalHeight?: number;
		videoWidth?: number;
		videoHeight?: number;
		width?: number | unknown;
		height?: number | unknown;
	};
	const w = s.naturalWidth || s.videoWidth || (typeof s.width === 'number' ? s.width : 0);
	const h = s.naturalHeight || s.videoHeight || (typeof s.height === 'number' ? s.height : 0);
	return { w, h };
}

/** Tint an image toward a colour while PRESERVING its detail — a `multiply` recolor, not a
 *  flat fill. The old `source-atop` solid fill replaced every opaque pixel, so an opaque
 *  image (a photo) collapsed to a solid rectangle (user report 2026-06-26). `multiply` keeps
 *  the source's luminance structure: a photo stays a photo, and a white silhouette mark still
 *  becomes the tint colour (white·c = c) — the legibility use case (#0300) is unchanged.
 *  `strength` scales the multiply via globalAlpha, so it lerps from the untinted image (0) to
 *  the fully-tinted result (1). Resolution-preserving (the buffer is the source's natural
 *  pixels). Returns the original source untouched when no offscreen buffer is available, so
 *  the draw never fails. */
function tintedSource(source: CanvasImageSource, tint: ImageTint): CanvasImageSource {
	const { w, h } = sourceSize(source);
	if (w <= 0 || h <= 0) return source;
	const buf = makeTintBuffer(w, h);
	if (!buf) return source;
	const bctx = (buf as HTMLCanvasElement).getContext('2d') as
		| CanvasRenderingContext2D
		| OffscreenCanvasRenderingContext2D
		| null;
	if (!bctx) return source;
	const strength = Math.min(1, Math.max(0, tint.strength));
	// Base = the source's real pixels (detail + alpha intact).
	bctx.drawImage(source, 0, 0, w, h);
	// Multiply the tint over it — keeps detail instead of overwriting it. globalAlpha=strength
	// makes this a partial multiply (0 = original image, 1 = full tint).
	bctx.globalCompositeOperation = 'multiply';
	bctx.globalAlpha = strength;
	bctx.fillStyle = tint.color;
	bctx.fillRect(0, 0, w, h);
	// The multiply rect also painted the fully-transparent margin (a PNG mark's cut-out) at
	// alpha=strength — clip the result back to the source's own alpha so transparency is kept.
	bctx.globalAlpha = 1;
	bctx.globalCompositeOperation = 'destination-in';
	bctx.drawImage(source, 0, 0, w, h);
	return buf;
}

/** Sentinel an ImageSource returns for a src whose load FAILED — distinct from
 *  `undefined` (= still loading). A failed layer renders an explicit broken-image
 *  state instead of a silent blank (I-2, Decision #0286 sprint). */
export const IMAGE_FAILED = 'failed' as const;

/** What an ImageSource lookup yields: a drawable source, the failure sentinel,
 *  or undefined while the load is still in flight. */
export type ImageSourceResult = CanvasImageSource | typeof IMAGE_FAILED | undefined;

/** A preloaded image source keyed by `layer.src`, supplied by the UI layer's cache. */
export type ImageSource = (src: string) => ImageSourceResult;

/** Trace a (rounded-)rectangle path — manual arcTo so it works on every ctx
 *  (no roundRect dependency). r = 0 traces a plain rectangle. */
function traceBox(
	ctx: CanvasRenderingContext2D,
	x: number,
	y: number,
	w: number,
	h: number,
	r: number
): void {
	const radius = Math.max(0, Math.min(r, w / 2, h / 2));
	ctx.beginPath();
	if (radius <= 0) {
		ctx.rect(x, y, w, h);
		return;
	}
	ctx.moveTo(x + radius, y);
	ctx.arcTo(x + w, y, x + w, y + h, radius);
	ctx.arcTo(x + w, y + h, x, y + h, radius);
	ctx.arcTo(x, y + h, x, y, radius);
	ctx.arcTo(x, y, x + w, y, radius);
	ctx.closePath();
}

/** Apply the layer's dash setting. */
function applyDash(ctx: CanvasRenderingContext2D, dashType: string): void {
	if (dashType === 'dashed') ctx.setLineDash([8, 4]);
	else if (dashType === 'dotted') ctx.setLineDash([2, 4]);
	else ctx.setLineDash([]);
}

/** Apply the layer's stroke style and stroke the current path (shared by the
 *  shape box, the text-box border and the image border — same StrokeStyle
 *  contract). glow_intensity > 0 adds the land-canvas glow: three widening
 *  faint passes in the lightened stroke color, then a centred shadow halo
 *  under the final solid stroke. `retrace` re-traces the path (the glow
 *  passes re-stroke it at different widths). */
function strokeBox(ctx: CanvasRenderingContext2D, layer: Layer, retrace: () => void): void {
	if (!layer.stroke || layer.stroke.thickness <= 0) return;
	const { color, thickness, dash_type } = layer.stroke;
	const glow = layer.stroke.glow_intensity ?? 0;
	if (glow > 0) {
		const baseAlpha = ctx.globalAlpha; // honour the layer opacity renderLayer set
		ctx.save();
		ctx.setLineDash([]);
		for (let i = 3; i >= 1; i--) {
			ctx.globalAlpha = baseAlpha * glow * 0.06 * (4 - i);
			ctx.lineWidth = thickness + i * glow * 1.5;
			ctx.strokeStyle = lightenHex(color, 0.4);
			retrace();
			ctx.stroke();
		}
		ctx.globalAlpha = baseAlpha;
		ctx.shadowBlur = glow * 3;
		ctx.shadowColor = lightenHex(color, 0.5);
		ctx.shadowOffsetX = 0;
		ctx.shadowOffsetY = 0;
		applyDash(ctx, dash_type);
		ctx.lineWidth = thickness;
		ctx.strokeStyle = color;
		retrace();
		ctx.stroke();
		ctx.restore();
		return;
	}
	applyDash(ctx, dash_type);
	ctx.strokeStyle = color;
	ctx.lineWidth = thickness;
	ctx.stroke();
}

/** Render a shape layer. Slice 1 supports rectangles (fill + stroke, optional
 *  corner_radius); other shape types fall back to their bounding rectangle so
 *  the artboard always shows the box. */
export function renderShapeLayer(ctx: CanvasRenderingContext2D, layer: Layer): void {
	const x = layer.x;
	const y = layer.y;
	const w = layer.width ?? 0;
	const h = layer.height ?? 0;
	if (w <= 0 || h <= 0) return;

	ctx.save();
	const retrace = (): void => traceBox(ctx, x, y, w, h, layer.corner_radius ?? 0);
	retrace();
	if (layer.fill) {
		applyBoxShadow(ctx, layer);
		const alpha = 1 - (layer.fill.transparency ?? 0);
		ctx.fillStyle = hexToRgba(layer.fill.color, alpha);
		ctx.fill();
		resetShadow(ctx);
	}
	strokeBox(ctx, layer, retrace);
	ctx.restore();
}

// --- Text fitting (S-FIT) ---------------------------------------------------
const FIT_FLOOR = 0.5; // 'fit' shrinks no smaller than 50% of the set size
const FIT_CEIL = 4; // 'fit' grow-to-fill cap (the box constraint caps it in practice)
const ELLIPSIS = '…';

/** Longest prefix of `text` plus a trailing … that fits `maxWidth` under `measure`.
 *  Returns the whole text when it already fits, or '…' when not even one glyph fits.
 *  Pure (takes a measure fn) so it's testable without a canvas context. */
export function truncateToWidth(text: string, maxWidth: number, measure: (s: string) => number): string {
	if (maxWidth <= 0) return '';
	if (measure(text) <= maxWidth) return text;
	let lo = 0;
	let hi = text.length;
	while (lo < hi) {
		const mid = Math.ceil((lo + hi) / 2);
		if (measure(text.slice(0, mid) + ELLIPSIS) <= maxWidth) lo = mid;
		else hi = mid - 1;
	}
	return lo > 0 ? text.slice(0, lo).replace(/\s+$/, '') + ELLIPSIS : ELLIPSIS;
}

/** Largest font size in [base·0.5, base·ceil] for which `fits(size)` holds, by binary
 *  search. `ceil` caps growth: 1 = SHRINK-ONLY ('fit' — never exceeds the set size),
 *  >1 = grow-to-fill ('fill' — up to base·ceil). Returns the floor when even it
 *  overflows (the caller then ellipsises). Pure (takes a predicate). */
export function fitFontSize(base: number, fits: (size: number) => boolean, ceil: number = FIT_CEIL): number {
	const floor = base * FIT_FLOOR;
	if (!fits(floor)) return floor;
	const ceiling = base * ceil;
	// The max already fits → use it EXACTLY (fit: the set size; fill: the grow cap) — no ε drift,
	// so 'fit' text in a roomy box renders at the set size, not a hair under it.
	if (fits(ceiling)) return ceiling;
	let lo = floor;
	let hi = ceiling;
	for (let i = 0; i < 14; i++) {
		const mid = (lo + hi) / 2;
		if (fits(mid)) lo = mid;
		else hi = mid;
	}
	return lo;
}

/** Render a text layer with optional box (background fill + border, both honouring
 *  corner_radius), word-wrap, alignment, color, and text-fit (S-FIT: fit/fill/crop/
 *  truncate/none — never raw-clipped in fit/fill/truncate). */
export function renderTextLayer(ctx: CanvasRenderingContext2D, layer: Layer): void {
	if (!layer.content) return;
	const content = layer.content;
	const x = layer.x;
	const y = layer.y;
	const w = layer.width ?? 200;
	const h = layer.height ?? 40;
	const baseSize = layer.font_size ?? 16;
	const pad = 8;
	const innerW = Math.max(0, w - pad * 2);
	const innerH = Math.max(0, h - pad * 2);
	const mode = layer.text_fit ?? 'none';
	const wrap = layer.text_wrap === 'word' && innerW > 0;

	ctx.save();

	// Optional box behind the text — background fill (with its drop shadow) +
	// border (with its glow) share one rounded path.
	if (layer.fill || (layer.stroke && layer.stroke.thickness > 0)) {
		const retrace = (): void => traceBox(ctx, x, y, w, h, layer.corner_radius ?? 0);
		retrace();
		if (layer.fill) {
			applyBoxShadow(ctx, layer);
			const alpha = 1 - (layer.fill.transparency ?? 0);
			ctx.fillStyle = hexToRgba(layer.fill.color, alpha);
			ctx.fill();
			resetShadow(ctx);
		}
		strokeBox(ctx, layer, retrace);
	}

	// Glyph effects — glow wins over shadow (land-canvas semantics); the ctx
	// shadow set here rides under every fillText below, restored by ctx.restore().
	applyTextEffects(ctx, layer);

	const fontStr = (size: number): string =>
		`${layer.font_style ?? 'normal'} ${layer.font_weight ?? 'normal'} ${size}px ${layer.font ?? 'sans-serif'}`;
	// Wrap (or single-line) the content at `size` and whether it fits the inner box.
	const linesAt = (size: number): string[] => {
		ctx.font = fontStr(size);
		return wrap ? wrapText(ctx, content, innerW) : [content];
	};
	const fitsAt = (size: number): boolean => {
		const ls = linesAt(size);
		if (ls.length * size * 1.3 > innerH) return false;
		for (const ln of ls) if (ctx.measureText(ln).width > innerW) return false;
		return true;
	};

	// 'fit' SHRINKS the font to fit (capped at the set size — ceil 1); 'fill' scales it
	// UP and down to fill the box (ceil 4); other modes keep the set size.
	const size =
		mode === 'fit'
			? fitFontSize(baseSize, fitsAt, 1)
			: mode === 'fill'
				? fitFontSize(baseSize, fitsAt)
				: baseSize;
	const lineHeight = size * 1.3;

	ctx.font = fontStr(size);
	ctx.fillStyle = layer.text_color ?? '#111111';
	const align = layer.text_align ?? 'left';
	ctx.textAlign = align;
	let textX = x + pad;
	if (align === 'center') textX = x + w / 2;
	else if (align === 'right') textX = x + w - pad;

	// 'crop' clips to the box and draws at the natural size (overflow hidden, no
	// ellipsis); the clip is restored by ctx.restore().
	if (mode === 'crop') {
		ctx.beginPath();
		ctx.rect(x, y, w, h);
		ctx.clip();
	}

	const measure = (s: string): number => ctx.measureText(s).width;

	// Resolve the lines to draw — ellipsise where the mode requires (never raw-clipped):
	//   truncate → single line + …; fit at the floor that still overflows → single
	//   line + …; otherwise wrap/single at the resolved size.
	let lines: string[];
	if (mode === 'truncate' || ((mode === 'fit' || mode === 'fill') && !fitsAt(size))) {
		ctx.font = fontStr(size);
		lines = [truncateToWidth(content, innerW, measure)];
	} else {
		lines = wrap ? wrapText(ctx, content, innerW) : [content];
	}

	if (lines.length > 1 || wrap) {
		const totalH = lines.length * lineHeight;
		const startY = y + (h - totalH) / 2 + size * 0.8;
		ctx.textBaseline = 'alphabetic';
		for (let i = 0; i < lines.length; i++) {
			ctx.fillText(lines[i], textX, startY + i * lineHeight);
		}
	} else {
		ctx.textBaseline = 'middle';
		ctx.fillText(lines[0], textX, y + h / 2);
	}

	ctx.restore();
}

/** Render an image layer.
 *  - `src` set + source loaded      → draw the image.
 *  - `src` set + load FAILED (CORS / 404 — the ImageSource reports IMAGE_FAILED) →
 *    render the explicit broken-image state (I-2): on screen AND in exports, a failed
 *    asset must be visibly broken, never a silent blank or a masking placeholder.
 *  - `src` set + STILL LOADING (undefined) → blank for the in-flight frame only, never
 *    throw — the loader schedules a redraw on settle.
 *  - NO `src` at all → a genuinely empty slot (unbound, or a bound token that resolved to
 *    '' because its index is out of range for the current record) → draw the placeholder
 *    `fill` (the factory seeds image layers with a neutral one). So a linked-but-empty slot
 *    reads as an intentional box, not a blank gap, while the link persists in the binding. */
export function renderImageLayer(
	ctx: CanvasRenderingContext2D,
	layer: Layer,
	images?: ImageSource
): void {
	const x = layer.x;
	const y = layer.y;
	const w = layer.width ?? 0;
	const h = layer.height ?? 0;
	if (w <= 0 || h <= 0) return;

	// Border around the image bounds (user call 2026-06-07: images take a border
	// + glow but never a box shadow). Drawn over every non-broken face — incl.
	// the in-flight blank, where the frame usefully shows the bounds. Radius 0:
	// a rounded border over an unclipped square image would read broken (rounded
	// image masking = the crop slice's concern).
	const drawBorder = (): void => {
		if (!layer.stroke || layer.stroke.thickness <= 0) return;
		ctx.save();
		const retrace = (): void => traceBox(ctx, x, y, w, h, 0);
		retrace();
		strokeBox(ctx, layer, retrace);
		ctx.restore();
	};

	if (layer.src) {
		const source = images?.(layer.src);
		if (source === IMAGE_FAILED) {
			renderBrokenImageState(ctx, x, y, w, h); // has its own dashed treatment
			return;
		}
		if (source) {
			ctx.save();
			// Centred glow halo (offset-free — images take a glow but never a box shadow,
			// user call 2026-06-07). The contrasting halo keeps the mark legible on any
			// background (#0300). Scoped to this draw by save/restore so the border (own
			// glow) is untouched.
			if (layer.glow && layer.glow.intensity > 0) {
				ctx.shadowColor = layer.glow.color;
				ctx.shadowBlur = layer.glow.intensity * 3;
				ctx.shadowOffsetX = 0;
				ctx.shadowOffsetY = 0;
			}
			const drawSrc =
				layer.tint && layer.tint.strength > 0 ? tintedSource(source, layer.tint) : source;
			ctx.drawImage(drawSrc, x, y, w, h);
			ctx.restore();
		}
		drawBorder(); // loaded AND in-flight: the frame marks the bounds
		return;
	}
	// Empty slot → placeholder fill (mirrors the shape/text fill path).
	if (layer.fill) {
		const alpha = 1 - (layer.fill.transparency ?? 0);
		ctx.save();
		ctx.fillStyle = hexToRgba(layer.fill.color, alpha);
		ctx.fillRect(x, y, w, h);
		ctx.restore();
	}
	drawBorder();
}

/** Explicit broken-image state (I-2): muted field + dashed inset border + a small
 *  vector broken-image glyph. Deterministic canvas2d vectors (no emoji/font glyph —
 *  cross-platform identical), neutral grays (the kit stays brand-agnostic). Exports
 *  show it too, deliberately: a failed asset must never export as a silent blank. */
function renderBrokenImageState(
	ctx: CanvasRenderingContext2D,
	x: number,
	y: number,
	w: number,
	h: number
): void {
	ctx.save();
	// Clip to the layer box — at small sizes the clamped glyph (and its slash
	// overhang) would otherwise scribble over neighbouring layers, on screen and
	// baked into exports.
	ctx.beginPath();
	ctx.rect(x, y, w, h);
	ctx.clip();
	// Muted field.
	ctx.fillStyle = 'rgba(0, 0, 0, 0.05)';
	ctx.fillRect(x, y, w, h);
	// Dashed inset border.
	ctx.strokeStyle = 'rgba(0, 0, 0, 0.25)';
	ctx.lineWidth = 1.5;
	ctx.setLineDash([6, 4]);
	ctx.strokeRect(x + 1, y + 1, w - 2, h - 2);
	// Glyph: a tiny image-frame (rect + sun dot + mountain) with a slash across.
	const s = Math.max(12, Math.min(w, h) * 0.18); // glyph box edge, clamped sane
	const gx = x + (w - s) / 2;
	const gy = y + (h - s) / 2;
	ctx.setLineDash([]);
	ctx.strokeStyle = 'rgba(0, 0, 0, 0.4)';
	ctx.lineWidth = Math.max(1, s * 0.08);
	ctx.strokeRect(gx, gy, s, s);
	// Sun dot (top-left third).
	ctx.fillStyle = 'rgba(0, 0, 0, 0.4)';
	ctx.beginPath();
	ctx.arc(gx + s * 0.3, gy + s * 0.3, s * 0.08, 0, Math.PI * 2);
	ctx.fill();
	// Mountain line.
	ctx.beginPath();
	ctx.moveTo(gx + s * 0.12, gy + s * 0.82);
	ctx.lineTo(gx + s * 0.45, gy + s * 0.45);
	ctx.lineTo(gx + s * 0.65, gy + s * 0.65);
	ctx.lineTo(gx + s * 0.88, gy + s * 0.42);
	ctx.stroke();
	// Slash across the frame (the "broken" cue).
	ctx.beginPath();
	ctx.moveTo(gx - s * 0.18, gy + s * 1.18);
	ctx.lineTo(gx + s * 1.18, gy - s * 0.18);
	ctx.stroke();
	ctx.restore();
}

/** Rotation transform around the layer's box centre — shared by the normal pass
 *  and the crop ghost so both rotate identically. No-op at rotation 0. */
function applyRotation(ctx: CanvasRenderingContext2D, layer: Layer): void {
	const rotation = layer.rotation ?? 0;
	if (rotation === 0) return;
	const cx = layer.x + (layer.width ?? 0) / 2;
	const cy = layer.y + (layer.height ?? 0) / 2;
	ctx.translate(cx, cy);
	ctx.rotate((rotation * Math.PI) / 180);
	ctx.translate(-cx, -cy);
}

/** Per-type draw dispatch — the content pass, with transforms/clips already set. */
function renderLayerBody(ctx: CanvasRenderingContext2D, layer: Layer, images?: ImageSource): void {
	switch (layer.type) {
		case 'text':
		// A callout is plain text. When anchored to a map surface (map_surface_id +
		// geo_ref), resolveFrame has ALREADY projected geo_ref → the callout's absolute
		// x/y using the PARENT map layer's center/zoom/SIZE — so render just draws the
		// text at x/y (no projection here; the kit renderer is per-layer and has no page
		// ref, while resolveFrame sees the whole page and the real map dimensions).
		case 'callout':
			renderTextLayer(ctx, layer);
			break;

		case 'image':
			renderImageLayer(ctx, layer, images);
			break;

		// A map layer's static image is loaded into `src` by the resolve pass and
		// drawn via the standard image renderer (crop/rotation already applied by
		// renderLayer's wrapper). The map_config is NOT consulted here — rendering is
		// pure drawImage from the resolved src.
		case 'map':
			renderImageLayer(ctx, layer, images);
			break;

		// Anchored marker/outline layers are BAKED into the map image; they must NOT
		// be drawn separately (that would double-render them, wrong position on export).
		// Free-floating markers/outlines (no map_surface_id) fall through to
		// renderShapeLayer so they still have a visible bounding box.
		case 'marker':
		case 'outline':
			if (layer.map_surface_id) {
				// No-op — baked into the map layer's static image.
				return;
			}
			renderShapeLayer(ctx, layer);
			break;

		case 'shape':
		default:
			renderShapeLayer(ctx, layer);
			break;
	}
}

/** A page-space fill region (px) a tiling layer repeats to cover — supplied by
 *  renderPage (and the hit-test) so the same copies render and select. */
export interface TileRegion {
	minX: number;
	minY: number;
	maxX: number;
	maxY: number;
}

/**
 * [WM] Draw a TILING layer: the source body repeated at every `tilePositions` origin,
 * with the WHOLE lattice rotated ONCE around the source centre (the layer's own
 * `rotation`). Each copy reuses the standard `renderLayer` path with `rotation` and
 * `tile` cleared — so every copy gets identical image/text/shape + opacity + crop
 * handling, never a second rotation, and never recurses back into tiling. Because the
 * shared renderer draws the copies, the live artboard, the crop ghost and the export
 * rasterizer all show the SAME pattern (WYSIWYG). `geometry.pointInLayer` walks the
 * identical positions, so a click on any copy resolves to this one source layer.
 */
function renderTiledLayer(
	ctx: CanvasRenderingContext2D,
	layer: Layer,
	tile: TileEffect,
	region: TileRegion,
	images?: ImageSource
): void {
	const box = { x: layer.x, y: layer.y, w: layer.width ?? 0, h: layer.height ?? 0 };
	const positions = tilePositions(box, tile, region);
	ctx.save();
	applyRotation(ctx, layer); // rotate the lattice once, around the source centre
	for (const p of positions) {
		// rotation:0 → not re-rotated (the lattice ctx is already rotated); tile:undefined
		// → no recursion; region omitted → renderLayer draws this single copy normally.
		renderLayer(ctx, { ...layer, x: p.x, y: p.y, rotation: 0, tile: undefined }, images);
	}
	ctx.restore();
}

/** Render a single resolved layer onto the context, honouring opacity and rotation.
 *  The caller passes layers that are ALREADY resolved (bindings + animation applied).
 *  Rotation is applied HERE, centrally, around the layer's box centre — every layer
 *  type (text/image/shape) rotates without per-renderer math. When the layer carries a
 *  tile effect AND a fill `region` is supplied (renderPage), it repeats across the page. */
export function renderLayer(
	ctx: CanvasRenderingContext2D,
	layer: Layer,
	images?: ImageSource,
	region?: TileRegion
): void {
	if (layer.visible === false) return;
	const opacity = layer.opacity ?? 1;
	if (opacity <= 0) return;

	// [WM] Tiling is a generic layer/group effect — rendered LIVE here (not a separate
	// preview surface) so authoring is WYSIWYG. Gated on a real repeat + a known region.
	const tile = layer.tile;
	if (tile && (tile.repeatH || tile.repeatV) && region) {
		renderTiledLayer(ctx, layer, tile, region, images);
		return;
	}

	ctx.save();
	ctx.globalAlpha = opacity;
	applyRotation(ctx, layer);
	// Per-edge inset crop (2026-06-07) — clip AFTER the rotation transform so the
	// crop window rotates with the layer. The layer then draws normally underneath;
	// each edge is an animatable scalar (wipe/reveal effects).
	const cropT = layer.crop_top ?? 0;
	const cropR = layer.crop_right ?? 0;
	const cropB = layer.crop_bottom ?? 0;
	const cropL = layer.crop_left ?? 0;
	if (cropT > 0 || cropR > 0 || cropB > 0 || cropL > 0) {
		const w = Math.max(0, (layer.width ?? 0) - cropL - cropR);
		const h = Math.max(0, (layer.height ?? 0) - cropT - cropB);
		traceBox(ctx, layer.x + cropL, layer.y + cropT, w, h, 0);
		ctx.clip();
	}
	renderLayerBody(ctx, layer, images);
	ctx.restore();
}

/** How much a crop ghost dims the cropped-away content (× the layer's opacity). */
export const CROP_GHOST_ALPHA = 0.3;

/**
 * Dimmed ghost of the CROPPED-AWAY strips — drawn by the editor during a live
 * crop gesture (after the normal pass) so the user keeps visual reference of
 * what the crop removes (Canva-style). Same rotation transform as renderLayer;
 * the clip is the INVERSE of the crop window (evenodd: layer box minus the kept
 * window), so the kept region — already drawn by the normal pass — is never
 * double-painted. Drawn AFTER the full page, the ghost intentionally rides over
 * higher layers for the gesture's duration — the reference must stay readable.
 * No-op when nothing is cropped or the layer is hidden/fully transparent.
 */
export function renderCropGhost(ctx: CanvasRenderingContext2D, layer: Layer, images?: ImageSource): void {
	if (layer.visible === false) return;
	if ((layer.opacity ?? 1) <= 0) return;
	const cropT = layer.crop_top ?? 0;
	const cropR = layer.crop_right ?? 0;
	const cropB = layer.crop_bottom ?? 0;
	const cropL = layer.crop_left ?? 0;
	if (cropT <= 0 && cropR <= 0 && cropB <= 0 && cropL <= 0) return;

	ctx.save();
	ctx.globalAlpha = (layer.opacity ?? 1) * CROP_GHOST_ALPHA;
	applyRotation(ctx, layer);
	const w = layer.width ?? 0;
	const h = layer.height ?? 0;
	const kw = Math.max(0, w - cropL - cropR);
	const kh = Math.max(0, h - cropT - cropB);
	ctx.beginPath();
	ctx.rect(layer.x, layer.y, w, h);
	ctx.rect(layer.x + cropL, layer.y + cropT, kw, kh);
	ctx.clip('evenodd');
	renderLayerBody(ctx, layer, images);
	ctx.restore();
}

/**
 * Render a page's layers in z-order (array order = back→front). When `layers` is
 * supplied it overrides `page.layers` — pass the resolved-at-time layers from
 * resolveFrame() so animated/bound frames draw correctly; omit it to draw the
 * static base. Hidden layers are skipped by renderLayer().
 */
export function renderPage(
	ctx: CanvasRenderingContext2D,
	page: Page,
	layers?: Layer[],
	images?: ImageSource
): void {
	const list = layers ?? page.layers;
	for (const layer of list) {
		// A tiling layer needs the page fill region; a plain layer doesn't (region
		// stays undefined → renderLayer draws a single placement).
		renderLayer(ctx, layer, images, layer.tile ? tileFillRegion(page, layer) : undefined);
	}
}

/**
 * [WM] The page-space region a tiling layer must repeat to cover. Exported so the
 * editor's hit-test (geometry.pointInLayer) walks the EXACT copies this renderer drew —
 * render and selection share one region, never drift.
 *  - No rotation → exactly the page box.
 *  - Rotated → a square around the SOURCE CENTRE with half-extent = the distance to the
 *    farthest page corner. Rotation preserves distance from the centre, so the
 *    back-rotated page is always inside this square → no uncovered corner once the
 *    lattice rotates (the √2 coverage technique, owned here — not baked into the pure
 *    `tilePositions`). Off-page copies are clipped by the page-sized canvas.
 */
export function tileFillRegion(page: Page, layer: Layer): TileRegion {
	const W = page.width;
	const H = page.height;
	const rot = layer.rotation ?? 0;
	if (!rot) return { minX: 0, minY: 0, maxX: W, maxY: H };
	const cx = layer.x + (layer.width ?? 0) / 2;
	const cy = layer.y + (layer.height ?? 0) / 2;
	let d = 0;
	for (const [px, py] of [
		[0, 0],
		[W, 0],
		[0, H],
		[W, H]
	] as const) {
		d = Math.max(d, Math.hypot(px - cx, py - cy));
	}
	return { minX: cx - d, minY: cy - d, maxX: cx + d, maxY: cy + d };
}
