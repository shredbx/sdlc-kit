// Document-level watermark params (Slice A2.2 · Decision #0298).
//
// These describe HOW the whole overlay design is stamped onto a target image — a
// DOCUMENT property, not a per-layer one. They ride the canvas document's opaque
// JSONB (`Document.watermark`), so they round-trip through the existing persistence
// path with no Go / migration change (the server stores the doc verbatim). The
// watermark Inspector block (in @sbx/canvas-ui) edits them in `watermark` mode; the
// publish/apply slices (B/C) read them to bake the overlay onto property images.
//
// SECURITY (#0298): these are authoring hints only. The authoritative watermark
// save/publish validation is server-owned — never trust the client params for
// persistence decisions.

/** How the overlay artwork repeats across the target image. */
export type WatermarkTiling = 'none' | 'grid' | 'diagonal' | 'brick-offset';

/** Where a single (non-tiled) mark anchors; `full-tile` covers the whole surface. */
export type WatermarkPosition =
	| 'top-left'
	| 'top-right'
	| 'bottom-left'
	| 'bottom-right'
	| 'center'
	| 'full-tile';

/** Contrast variant the overlay is rendered for. `auto` mirrors the shredbx
 *  shutterstock-classic compositor (`ColorMode: "auto"`): the apply pipeline derives the
 *  mark colour from the target photo's luminance (light photo → near-black, else warm
 *  off-white). `dark`/`light` pin it. */
export type WatermarkTheme = 'dark' | 'light' | 'auto';

/** Compositing mode used when the overlay is stamped onto the photo. */
export type WatermarkBlend = 'normal' | 'multiply' | 'screen';

/** Document-level watermark stamping params. */
export interface WatermarkParams {
	/** Repeat pattern of the overlay across the target image. */
	tiling: WatermarkTiling;
	/** Rotation of the stamped mark, in degrees. */
	rotation: number;
	/** Spacing between repeats (tiling density) — larger = sparser. */
	density: number;
	/** Overlay opacity, 0–100 (percent). */
	opacity: number;
	/** Anchor for a single mark (ignored by full-coverage tilings). */
	position: WatermarkPosition;
	/** Light/dark rendering variant. */
	theme: WatermarkTheme;
	/** Compositing mode against the photo. */
	blend: WatermarkBlend;
}

/** A FRESH default watermark-params object (mirrors the resolver's purity — never a
 *  shared singleton, so a caller can seed a doc and mutate it without touching the
 *  defaults). The defaults describe a subtle, legible single-mark stamp. */
export function DEFAULT_WATERMARK_PARAMS(): WatermarkParams {
	return {
		tiling: 'none',
		rotation: 0,
		density: 100,
		opacity: 50,
		position: 'bottom-right',
		theme: 'dark',
		blend: 'normal'
	};
}

/** The shredbx "Shutterstock Classic" params (the manual `sbx media image watermark
 *  apply --preset shutterstock-classic` look, #0298 / #0299): a diagonal tiled mark at
 *  -20°, 140px spacing, 18% opacity, auto-contrast, covering the whole frame. SHARED by
 *  the watermark-document seed (a new watermark opens as this mark) AND the Inspector's
 *  one-click preset — one source of truth, never a copy. A FRESH object each call
 *  (purity, like DEFAULT_WATERMARK_PARAMS). Under #0299 these are FULL-FRAME params: the
 *  mark unit is tiled across the photo-sized overlay, never baked at a small tile size. */
export function SHUTTERSTOCK_CLASSIC_PARAMS(): WatermarkParams {
	return {
		tiling: 'diagonal',
		rotation: -20,
		density: 140,
		opacity: 18,
		position: 'full-tile',
		theme: 'auto',
		blend: 'normal'
	};
}
